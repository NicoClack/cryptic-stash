package invites

import (
	"net/url"

	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

// TODO: create auth service for this?

type webAuthnUser struct {
	id          []byte
	name        string
	displayName string
}

func (u *webAuthnUser) WebAuthnID() []byte                         { return u.id }
func (u *webAuthnUser) WebAuthnName() string                       { return u.name }
func (u *webAuthnUser) WebAuthnDisplayName() string                { return u.displayName }
func (u *webAuthnUser) WebAuthnCredentials() []webauthn.Credential { return nil }

func newWebAuthnApp(app *servercommon.ServerApp) (*webauthn.WebAuthn, string) {
	parsedURL, stdErr := url.Parse(app.Env.FRONTEND_BASE_URL)
	if stdErr != nil {
		panic("invalid FRONTEND_BASE_URL. error:\n" + stdErr.Error())
	}

	origin := parsedURL.Scheme + "://" + parsedURL.Host
	relayingPartyID := parsedURL.Hostname()

	webAuthnApp, stdErr := webauthn.New(&webauthn.Config{
		RPID:          relayingPartyID,
		RPDisplayName: "Cryptic Stash",
		RPOrigins:     []string{origin},
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			UserVerification: protocol.VerificationRequired,
		},
	})
	if stdErr != nil {
		panic("failed to create webauthn instance. error:\n" + stdErr.Error())
	}

	return webAuthnApp, relayingPartyID
}
