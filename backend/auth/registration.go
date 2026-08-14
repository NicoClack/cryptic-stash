package auth

import (
	"context"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
)

func StartRegisterPasskey(
	user webauthn.User,
	webAuthnApp *webauthn.WebAuthn,
) (protocol.PublicKeyCredentialCreationOptions, *webauthn.SessionData, common.WrappedError) {
	credentials := user.WebAuthnCredentials()
	excludedCredentials := make([]protocol.CredentialDescriptor, 0, len(credentials))
	for _, credential := range credentials {
		excludedCredentials = append(excludedCredentials, credential.Descriptor())
	}

	creation, sessionData, stdErr := webAuthnApp.BeginRegistration(
		user,
		webauthn.WithExclusions(excludedCredentials),
	)
	if stdErr != nil {
		return protocol.PublicKeyCredentialCreationOptions{},
			nil,
			ErrWrapperStartRegisterPasskey.Wrap(stdErr)
	}
	return creation.Response, sessionData, nil
}

func FinishRegisterPasskey(
	credentialName string,
	allowSudo bool,
	isSecondGroup bool,
	username string,
	session *webauthn.SessionData,
	parsedCredential *protocol.ParsedCredentialCreationData,
	webAuthnApp *webauthn.WebAuthn,
	tx *ent.Tx,
	ctx context.Context,
	getUser func(userID uuid.UUID, tx *ent.Tx) (*ent.User, error),
) (*ent.Passkey, common.WrappedError) {
	if !session.Expires.IsZero() && time.Now().After(session.Expires) {
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

	userID, stdErr := uuid.FromBytes(session.UserID)
	if stdErr != nil {
		return nil, ErrWrapperFinishRegisterPasskey.Wrap(stdErr)
	}

	userOb, stdErr := getUser(userID, tx)
	if stdErr != nil {
		return nil, ErrWrapperFinishRegisterPasskey.Wrap(
			ErrWrapperGetUserCallback.Wrap(
				common.AutoWrapError(stdErr),
			),
		)
	}

	now := time.Now()
	passkeyOb, stdErr := tx.Passkey.Create().
		SetCreatedAt(now).
		SetUpdatedAt(now).
		SetUserID(userOb.ID).
		SetName(credentialName).
		SetAllowSudo(allowSudo).
		SetIsSecondGroup(isSecondGroup).
		SetCredentialID(credential.ID).
		SetCredential(*credential).
		Save(ctx)
	if stdErr != nil {
		return nil, ErrWrapperFinishRegisterPasskey.Wrap(ErrWrapperDatabase.Wrap(stdErr))
	}

	return passkeyOb, nil
}
