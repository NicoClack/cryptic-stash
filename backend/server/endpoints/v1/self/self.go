package self

import (
	"github.com/NicoClack/cryptic-stash/backend/server/endpoints/v1/self/passkeys"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
)

func ConfigureEndpoints(group *servercommon.Group) {
	passkeyGroup := group.Group("/passkeys")
	passkeyGroup.Use(group.App.SudoModeMiddleware) // TODO: this is running in addition to the default auth middleware
	passkeys.ConfigureEndpoints(passkeyGroup)
}
