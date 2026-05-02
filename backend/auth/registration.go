package auth

import (
	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
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
	user webauthn.User,
	sessionData webauthn.SessionData,
	credentialJSON []byte,
	webAuthnApp *webauthn.WebAuthn,
) (*webauthn.Credential, common.WrappedError) {
	parsedCredential, stdErr := protocol.ParseCredentialCreationResponseBytes(credentialJSON)
	if stdErr != nil {
		return nil, ErrWrapperFinishRegisterPasskey.Wrap(stdErr)
	}
	credential, stdErr := webAuthnApp.CreateCredential(user, sessionData, parsedCredential)
	if stdErr != nil {
		return nil, ErrWrapperFinishRegisterPasskey.Wrap(stdErr)
	}
	return credential, nil
}
