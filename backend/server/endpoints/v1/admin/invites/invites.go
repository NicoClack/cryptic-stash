package invites

import (
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
)

func ConfigureEndpoints(group *servercommon.Group) {
	group.GET("/", ListInvites(group.App))
	group.GET("/:id", GetInvite(group.App))
	group.POST("/create", Create(group.App))
}
