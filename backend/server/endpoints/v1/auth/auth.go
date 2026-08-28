package auth

import (
	"github.com/NicoClack/cryptic-stash/backend/server/endpoints/v1/auth/login"
	"github.com/NicoClack/cryptic-stash/backend/server/endpoints/v1/auth/sudo"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
)

func ConfigureEndpoints(group *servercommon.Group) {
	login.ConfigureEndpoints(group.Group("/login"))

	sudoGroup := group.Group("/sudo")
	sudoGroup.Use(group.App.DefaultAuthMiddleware)
	sudo.ConfigureEndpoints(sudoGroup)
}
