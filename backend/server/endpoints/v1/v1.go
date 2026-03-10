package v1

import (
	"github.com/NicoClack/cryptic-stash/backend/server/endpoints/v1/admin"
	"github.com/NicoClack/cryptic-stash/backend/server/endpoints/v1/setup"
	"github.com/NicoClack/cryptic-stash/backend/server/endpoints/v1/signuplinks"
	"github.com/NicoClack/cryptic-stash/backend/server/endpoints/v1/twofactoractions"
	"github.com/NicoClack/cryptic-stash/backend/server/endpoints/v1/users"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
)

func ConfigureEndpoints(group *servercommon.Group) {
	setup.ConfigureEndpoints(group.Group("/setup"))
	if !group.App.Env.ENABLE_ENV_SETUP {
		signuplinks.ConfigureEndpoints(group.Group("/signup-links"))
		users.ConfigureEndpoints(group.Group("/users"))
		twofactoractions.ConfigureEndpoints(group.Group("/two-factor-actions"))

		group.POST("/admin/login/", admin.Login(group.App))
		adminGroup := group.Group("/admin")
		adminGroup.Use(group.App.AdminMiddleware)
		admin.ConfigureEndpoints(adminGroup)
	}
}
