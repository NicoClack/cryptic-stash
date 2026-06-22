package users

import (
	"github.com/NicoClack/cryptic-stash/backend/server/endpoints/v1/users/login"
	"github.com/NicoClack/cryptic-stash/backend/server/middleware"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
)

func ConfigureEndpoints(group *servercommon.Group) {
	login.ConfigureEndpoints(group.Group("/login"))
	group.POST("/get-authorization-code/", GetAuthorizationCode(group.App))
	group.GET(
		"/auth-test/",
		middleware.NewSessionAuth(group.App.Auth, group.App.Database, nil),
		AuthTest(group.App),
	)
	group.POST("/download/", Download(group.App))
	group.POST("/self-lock/", SelfLock(group.App))
}
