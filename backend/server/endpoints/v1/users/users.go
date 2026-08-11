package users

import "github.com/NicoClack/cryptic-stash/backend/server/servercommon"

func ConfigureEndpoints(group *servercommon.Group) {
	group.POST("/get-authorization-code/", GetAuthorizationCode(group.App))
	group.GET(
		"/auth-test/",
		group.App.DefaultAuthMiddleware,
		AuthTest(group.App),
	)
	group.POST("/download/", Download(group.App))
	group.POST("/self-lock/", SelfLock(group.App))
}
