package users

import (
	"github.com/NicoClack/cryptic-stash/backend/server/endpoints/v1/admin/users/messengers"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
)

func ConfigureEndpoints(group *servercommon.Group) {
	group.GET("/", ListUsers(group.App))
	messengers.ConfigureEndpoints(group.Group("/:id/messengers"))
}
