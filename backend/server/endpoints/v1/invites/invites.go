package invites

import "github.com/NicoClack/cryptic-stash/backend/server/servercommon"

func ConfigureEndpoints(group *servercommon.Group) {
	group.GET("/:id", GetInvite(group.App))
	group.POST("/:id/generate-options", GenerateOptions(group.App))
	group.POST("/:id/create-user", CreateUser(group.App))
}
