package definitions

import (
	"github.com/NicoClack/cryptic-stash/backend/tempkeyvalue"
	"github.com/go-webauthn/webauthn/webauthn"
)

func Register(group *tempkeyvalue.RegistryGroup) {
	group.Register(&tempkeyvalue.Definition{
		Name: "loginWebAuthnSession",
		Type: &webauthn.SessionData{},
	})
}
