package auth

import (
	"context"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/jonboulle/clockwork"
)

func StartRegisterPasskey(
	user webauthn.User,
	webAuthnApp *webauthn.WebAuthn,
) (protocol.PublicKeyCredentialCreationOptions, *webauthn.SessionData, common.WrappedError) {
	creation, sessionData, stdErr := webAuthnApp.BeginRegistration(user)
	if stdErr != nil {
		return protocol.PublicKeyCredentialCreationOptions{},
			nil,
			ErrWrapperStartRegisterPasskey.Wrap(stdErr)
	}
	return creation.Response, sessionData, nil
}

func FinishRegisterPasskey(
	session *webauthn.SessionData,
	username string,
	credentialJSON []byte,
	credentialName string,
	webAuthnApp *webauthn.WebAuthn,
	tx *ent.Tx,
	clock clockwork.Clock,
	ctx context.Context,
	getUser func(userID uuid.UUID, tx *ent.Tx) (*ent.User, error),
) (*ent.Passkey, common.WrappedError) {
	if !session.Expires.IsZero() && clock.Now().After(session.Expires) {
		return nil, ErrWrapperFinishRegisterPasskey.Wrap(ErrWebAuthnSessionExpired)
	}

	webAuthnUser := &TempWebAuthnUser{
		ID:          session.UserID,
		Name:        username,
		DisplayName: username,
	}

	parsedCredential, stdErr := protocol.ParseCredentialCreationResponseBytes(credentialJSON)
	if stdErr != nil {
		// TODO: turn ErrInvalidCredential into an error wrapper?
		wrappedErr := ErrWrapperFinishRegisterPasskey.Wrap(ErrInvalidCredential)
		wrappedErr.AddDebugValuesMut(common.DebugValue{
			Name:  "original error",
			Value: stdErr,
		})
		return nil, wrappedErr
	}
	credential, stdErr := webAuthnApp.CreateCredential(webAuthnUser, *session, parsedCredential)
	if stdErr != nil {
		// TODO: turn ErrInvalidCredential into an error wrapper?
		wrappedErr := ErrWrapperFinishRegisterPasskey.Wrap(ErrInvalidCredential)
		wrappedErr.AddDebugValuesMut(
			common.DebugValue{
				Name:  "original error",
				Value: stdErr,
			},
		)
		return nil, wrappedErr
	}

	aaguid := credential.Authenticator.AAGUID
	if len(aaguid) == 0 {
		aaguid = make([]byte, 16)
	} else if len(aaguid) != 16 {
		return nil, ErrWrapperFinishRegisterPasskey.Wrap(ErrInvalidAAGUIDLength)
	}

	userID, stdErr := uuid.FromBytes(session.UserID)
	if stdErr != nil {
		return nil, ErrWrapperFinishRegisterPasskey.Wrap(stdErr)
	}

	userOb, stdErr := getUser(userID, tx)
	if stdErr != nil {
		return nil, ErrWrapperFinishRegisterPasskey.Wrap(
			common.AutoWrapError(stdErr),
		)
	}

	now := clock.Now()
	passkeyOb, stdErr := tx.Passkey.Create().
		SetCreatedAt(now).
		SetUpdatedAt(now).
		SetUserID(userOb.ID).
		SetName(credentialName).
		SetCredentialID(credential.ID).
		SetPublicKey(credential.PublicKey).
		SetAaguid(aaguid).
		SetSignCount(credential.Authenticator.SignCount).
		Save(ctx)
	if stdErr != nil {
		return nil, ErrWrapperFinishRegisterPasskey.Wrap(ErrWrapperDatabase.Wrap(stdErr))
	}

	return passkeyOb, nil
}
