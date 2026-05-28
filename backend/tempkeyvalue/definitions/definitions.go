package definitions

import (
	"github.com/NicoClack/cryptic-stash/backend/tempkeyvalue"
	"github.com/go-webauthn/webauthn/webauthn"
)

func Register(group *tempkeyvalue.RegistryGroup) {
	group.Register(&tempkeyvalue.Definition{
		Name: "WEBAUTHN_SESSIONS", // TODO: update to an import once packages are better split up?
		Type: &webauthn.SessionData{},
	})
}
