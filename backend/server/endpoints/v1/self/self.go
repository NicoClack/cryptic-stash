package self

import (
	"github.com/NicoClack/cryptic-stash/backend/server/endpoints/v1/self/passkeys"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
)

func ConfigureEndpoints(group *servercommon.Group) {
	passkeyGroup := group.Group("/passkeys")
	passkeyGroup.Use(group.App.SudoModeMiddleware)
	passkeys.ConfigureEndpoints(passkeyGroup)
}
