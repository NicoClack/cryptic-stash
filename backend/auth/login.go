package auth

import (
	"context"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/ent/passkey"
	"github.com/NicoClack/cryptic-stash/backend/ent/user"
	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/jonboulle/clockwork"
)

func StartLogin(
	webAuthnApp *webauthn.WebAuthn,
	tempKV common.TempKeyValueService,
	clock clockwork.Clock,
) (uuid.UUID, protocol.PublicKeyCredentialRequestOptions, common.WrappedError) {
	creation, sessionData, stdErr := webAuthnApp.BeginDiscoverableLogin()
	if stdErr != nil {
		return uuid.Nil, protocol.PublicKeyCredentialRequestOptions{}, ErrWrapperStartLogin.Wrap(stdErr)
	}

	webAuthnSessionID := uuid.New()
	// TODO: what happens if parent transaction fails?
	tempKV.Set(WebAuthnSessionStoreName, webAuthnSessionID.String(), sessionData, sessionData.Expires)

	return webAuthnSessionID, creation.Response, nil
}

func FinishLogin(
	webAuthnSessionID uuid.UUID,
	parsedResponse *protocol.ParsedCredentialAssertionData,
	ginCtx *gin.Context,
	webAuthnApp *webauthn.WebAuthn,
	tx *ent.Tx,
	tempKV common.TempKeyValueService,
	clock clockwork.Clock,
	logger common.Logger,
	sessionDuration time.Duration,
) (*ent.Session, []byte, common.WrappedError) {
	var sessionData *webauthn.SessionData
	if !tempKV.Get(WebAuthnSessionStoreName, webAuthnSessionID.String(), &sessionData) {
		return nil, nil, ErrWrapperFinishLogin.Wrap(ErrInvalidWebAuthnSessionID)
	}
	tx.OnCommit(func(committer ent.Committer) ent.Committer {
		return ent.CommitFunc(func(ctx context.Context, tx *ent.Tx) error {
			stdErr := committer.Commit(ctx, tx)
			if stdErr != nil {
				return stdErr
			}
			tempKV.Delete(WebAuthnSessionStoreName, webAuthnSessionID.String())
			return nil
		})
	})

	ctx := ginCtx.Request.Context()
	var userOb *ent.User
	_, credential, stdErr := webAuthnApp.ValidatePasskeyLogin(
		func(rawID, userHandle []byte) (webauthn.User, error) {
			userID, stdErr := uuid.FromBytes(userHandle)
			if stdErr != nil {
				return nil, nil
			}
			userOb, stdErr = tx.User.Query().
				Where(user.ID(userID)).
				WithPasskeys().
				Only(ctx)
			if stdErr != nil {
				if ent.IsNotFound(stdErr) {
					return nil, nil
				}
				return nil, ErrWrapperDatabase.Wrap(stdErr)
			}
			return &RealWebAuthnUser{
				User: userOb,
			}, nil
		},
		*sessionData,
		parsedResponse,
	)
	if stdErr != nil {
		// TODO: send these errors to the client by checking for ErrTypeClient
		return nil, nil, ErrWrapperFinishLogin.Wrap(stdErr)
	}

	// TODO: this is a bit inefficient
	passkeyID, stdErr := tx.Passkey.Query().
		Where(
			passkey.UserID(userOb.ID),
			passkey.CredentialID(credential.ID),
		).
		OnlyID(ctx)
	if stdErr != nil {
		return nil, nil, ErrWrapperFinishLogin.Wrap(ErrWrapperDatabase.Wrap(stdErr))
	}
	sessionOb, sessionToken, wrappedErr := CreateSession(
		userOb.ID,
		passkeyID,
		ginCtx.Request.UserAgent(),
		ginCtx.ClientIP(),
		tx,
		clock,
		sessionDuration,
		ctx,
	)
	if wrappedErr != nil {
		return nil, nil, ErrWrapperFinishLogin.Wrap(
			wrappedErr,
		)
	}

	if credential.Authenticator.CloneWarning {
		logger.Error(
			"Security warning: authenticator may have been cloned",
			"userID",
			userOb.ID,
			"credentialID",
			credential.ID,
			// Backed up keys might be more likely to trigger this warning?
			// Although most seem to leave the counter at 0
			"credentialBackupState",
			credential.Flags.BackupState,
		)
	}
	stdErr = tx.Passkey.UpdateOneID(passkeyID).
		SetUpdatedAt(clock.Now()).
		SetSignCount(credential.Authenticator.SignCount).
		Exec(ctx)
	if stdErr != nil {
		return nil, nil, ErrWrapperFinishLogin.Wrap(
			ErrWrapperDatabase.Wrap(stdErr),
		)
	}

	return sessionOb, sessionToken, nil
}
