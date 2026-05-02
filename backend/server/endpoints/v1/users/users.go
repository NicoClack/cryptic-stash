package users

import (
	"github.com/NicoClack/cryptic-stash/backend/server/endpoints/v1/users/login"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
)

func ConfigureEndpoints(group *servercommon.Group) {
	login.ConfigureEndpoints(group.Group("/login"))
	group.POST("/get-authorization-code/", GetAuthorizationCode(group.App))
	group.GET("/auth-test/", AuthTest(group.App))
	group.POST("/download/", Download(group.App))
	group.POST("/self-lock/", SelfLock(group.App))
}
