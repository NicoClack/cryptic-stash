package signuplinks

import "github.com/NicoClack/cryptic-stash/backend/server/servercommon"

func ConfigureEndpoints(group *servercommon.Group) {
	group.GET("/:id", GetSignupLink(group.App))
	group.POST("/:id/create-user", CreateUser(group.App))
}
