package auth

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/ent"
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

func ValidateLogin(
	webAuthnSessionID uuid.UUID,
	parsedResponse *protocol.ParsedCredentialAssertionData,
	ctx context.Context,
	webAuthnApp *webauthn.WebAuthn,
	tx *ent.Tx,
	tempKV common.TempKeyValueService,
	clock clockwork.Clock,
	logger common.Logger,
) (*ent.User, *ent.Passkey, bool, common.WrappedError) {
	var sessionData *webauthn.SessionData
	if !tempKV.Get(WebAuthnSessionStoreName, webAuthnSessionID.String(), &sessionData) {
		return nil, nil, false, ErrWrapperValidateLogin.Wrap(ErrInvalidWebAuthnSessionID)
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

	var userOb *ent.User
	_, credential, stdErr := webAuthnApp.ValidatePasskeyLogin(
		func(rawID, userHandle []byte) (webauthn.User, error) {
			userID, stdErr := uuid.FromBytes(userHandle)
			if stdErr != nil {
				return nil, stdErr
			}
			userOb, stdErr = tx.User.Query().
				Where(user.ID(userID)).
				WithPasskeys().
				Only(ctx)
			if stdErr != nil {
				if ent.IsNotFound(stdErr) {
					return nil, ErrWebAuthnUserNotFound.Clone()
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
		// Avoid wrapping *protocol.Error -> common.WrappedError in another common.WrappedError.
		// Instead we strip the outer layer because ErrWrapperInternalGetUser represents it well enough
		innerErr := errors.Unwrap(stdErr)
		if common.IsErrorType[common.WrappedError](innerErr) {
			return nil, nil, false, ErrWrapperValidateLogin.Wrap(
				ErrWrapperInternalGetUser.Wrap(innerErr),
			)
		}
		// Some other error, probably a client one
		return nil, nil, false, ErrWrapperValidateLogin.Wrap(stdErr)
	}

	var passkeyOb *ent.Passkey
	for _, p := range userOb.Edges.Passkeys {
		if slices.Equal(p.CredentialID, credential.ID) {
			passkeyOb = p
			break
		}
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

	passkeyOb.Credential = *credential
	// TODO: race condition?
	stdErr = tx.Passkey.UpdateOne(passkeyOb).
		SetUpdatedAt(clock.Now()).
		SetCredential(passkeyOb.Credential).
		Exec(ctx)
	if stdErr != nil {
		return nil, nil, false, ErrWrapperValidateLogin.Wrap(
			ErrWrapperDatabase.Wrap(stdErr),
		)
	}

	return userOb, passkeyOb, credential.Authenticator.CloneWarning, nil
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
) (*ent.User, *ent.Passkey, *ent.Session, []byte, common.WrappedError) {
	userOb, passkeyOb, _, wrappedErr := ValidateLogin(
		webAuthnSessionID,
		parsedResponse,
		ginCtx.Request.Context(),
		webAuthnApp,
		tx,
		tempKV,
		clock,
		logger,
	)
	if wrappedErr != nil {
		return nil, nil, nil, nil, ErrWrapperFinishLogin.Wrap(
			wrappedErr,
		)
	}

	sessionOb, sessionToken, wrappedErr := CreateSession(
		false,
		userOb.ID,
		passkeyOb.ID,
		ginCtx.Request.UserAgent(),
		ginCtx.ClientIP(),
		tx,
		clock,
		sessionDuration,
		ginCtx.Request.Context(),
	)
	if wrappedErr != nil {
		return nil, nil, nil, nil, ErrWrapperFinishLogin.Wrap(
			wrappedErr,
		)
	}

	return userOb, passkeyOb, sessionOb, sessionToken, nil
}
