package auth

import (
	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

const (
	WebAuthnSessionStoreName = "WEBAUTHN_SESSIONS"
	SessionTokenLength       = 32 // 256 bits
)

func NewWebAuthnApp(env *common.Env) *webauthn.WebAuthn {
	origin := common.GetOrigin(env.FRONTEND_BASE_URL)
	relyingPartyID := env.FRONTEND_BASE_URL.Hostname()

	webAuthnApp, stdErr := webauthn.New(&webauthn.Config{
		RPID:                        relyingPartyID,
		RPDisplayName:               "Cryptic Stash",
		RPOrigins:                   []string{origin},
		RPTopOriginVerificationMode: protocol.TopOriginImplicitVerificationMode,
		AttestationPreference:       protocol.PreferNoAttestation,
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			UserVerification: protocol.VerificationRequired,
		},
		Timeouts: webauthn.TimeoutsConfig{
			Login: webauthn.TimeoutConfig{
				Enforce:    true,
				Timeout:    env.WEBAUTHN_SESSION_TIMEOUT,
				TimeoutUVD: env.WEBAUTHN_SESSION_TIMEOUT,
			},
			Registration: webauthn.TimeoutConfig{
				Enforce:    true,
				Timeout:    env.WEBAUTHN_SESSION_TIMEOUT,
				TimeoutUVD: env.WEBAUTHN_SESSION_TIMEOUT,
			},
		},
	})
	if stdErr != nil {
		panic("failed to create webauthn instance. error:\n" + stdErr.Error())
	}

	return webAuthnApp
}

type RealWebAuthnUser struct {
	*ent.User
}

func (webAuthnUser *RealWebAuthnUser) WebAuthnID() []byte {
	return webAuthnUser.ID[:]
}
func (webAuthnUser *RealWebAuthnUser) WebAuthnName() string {
	return webAuthnUser.Username
}
func (webAuthnUser *RealWebAuthnUser) WebAuthnDisplayName() string {
	return webAuthnUser.Username
}
func (webAuthnUser *RealWebAuthnUser) WebAuthnCredentials() []webauthn.Credential {
	credentials := make([]webauthn.Credential, 0, len(webAuthnUser.Edges.Passkeys))
	for _, passkeyOb := range webAuthnUser.Edges.Passkeys {
		credentials = append(credentials, webauthn.Credential{
			ID:        passkeyOb.CredentialID,
			PublicKey: passkeyOb.PublicKey,
			Authenticator: webauthn.Authenticator{
				AAGUID:    passkeyOb.Aaguid[:],
				SignCount: passkeyOb.SignCount,
			},
		})
	}
	return credentials
}

type TempWebAuthnUser struct {
	ID          []byte
	Name        string
	DisplayName string
}

func (webAuthnUser *TempWebAuthnUser) WebAuthnID() []byte {
	return webAuthnUser.ID
}
func (webAuthnUser *TempWebAuthnUser) WebAuthnName() string {
	return webAuthnUser.Name
}
func (webAuthnUser *TempWebAuthnUser) WebAuthnDisplayName() string {
	return webAuthnUser.DisplayName
}
func (webAuthnUser *TempWebAuthnUser) WebAuthnCredentials() []webauthn.Credential {
	return nil
}
