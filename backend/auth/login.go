package auth

import (
	"context"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/ent/passkey"
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
) (string, protocol.PublicKeyCredentialRequestOptions, common.WrappedError) {
	creation, sessionData, stdErr := webAuthnApp.BeginDiscoverableLogin()
	if stdErr != nil {
		return "", protocol.PublicKeyCredentialRequestOptions{}, ErrWrapperStartLogin.Wrap(stdErr)
	}

	ceremonyID := uuid.NewString()
	// TODO: what happens if parent transaction fails?
	tempKV.Set(CeremonyStoreName, ceremonyID, *sessionData, sessionData.Expires)

	return ceremonyID, creation.Response, nil
}

func FinishLogin(
	ceremonyID string,
	ginCtx *gin.Context,
	webAuthnApp *webauthn.WebAuthn,
	tx *ent.Tx,
	tempKV common.TempKeyValueService,
	clock clockwork.Clock,
	sessionDuration time.Duration,
) (*ent.Session, []byte, common.WrappedError) {
	var sessionData webauthn.SessionData
	if !tempKV.Get(CeremonyStoreName, ceremonyID, &sessionData) {
		return nil, nil, ErrWrapperFinishLogin.Wrap(ErrInvalidCeremonyID)
	}
	tx.OnCommit(func(c ent.Committer) ent.Committer {
		return ent.CommitFunc(func(ctx context.Context, tx *ent.Tx) error {
			tempKV.Delete(CeremonyStoreName, ceremonyID)
			return nil
		})
	})

	ctx := ginCtx.Request.Context()
	var passkeyOb *ent.Passkey
	_, credential, stdErr := webAuthnApp.FinishPasskeyLogin(
		func(rawID, userHandle []byte) (webauthn.User, error) {
			var stdErr error
			passkeyOb, stdErr = tx.Passkey.Query().
				Where(passkey.CredentialID(rawID)).
				WithUser().
				Only(ctx)
			if stdErr != nil {
				if ent.IsNotFound(stdErr) {
					return nil, ErrInvalidCredential.Clone()
				}
				return nil, ErrWrapperDatabase.Wrap(stdErr)
			}
			return &RealWebAuthnUser{
				User: passkeyOb.Edges.User,
			}, nil
		},
		sessionData,
		ginCtx.Request,
	)
	if stdErr != nil {
		// TODO: how should these errors be sent to the client?
		return nil, nil, ErrWrapperFinishLogin.Wrap(stdErr)
	}
	userOb := passkeyOb.Edges.User

	sessionOb, sessionToken, wrappedErr := CreateSession(
		userOb.ID,
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

	_, stdErr = tx.Passkey.UpdateOneID(passkeyOb.ID).
		SetUpdatedAt(clock.Now()).
		SetSignCount(credential.Authenticator.SignCount). // TODO: log warning/error if mismatch
		Save(ctx)
	if stdErr != nil {
		return nil, nil, ErrWrapperFinishLogin.Wrap(
			ErrWrapperDatabase.Wrap(stdErr),
		)
	}

	return sessionOb, sessionToken, nil
}
