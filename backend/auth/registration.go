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
	parsedCredential *protocol.ParsedCredentialCreationData,
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

	credential, stdErr := webAuthnApp.CreateCredential(webAuthnUser, *session, parsedCredential)
	if stdErr != nil {
		return nil, ErrWrapperFinishRegisterPasskey.Wrap(stdErr)
	}

	var aaguid uuid.UUID
	if len(credential.Authenticator.AAGUID) == 16 {
		aaguid = [16]byte(credential.Authenticator.AAGUID)
	} else if len(credential.Authenticator.AAGUID) != 0 {
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
