package signuplinks

import (
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
)

func ConfigureEndpoints(group *servercommon.Group) {
	group.GET("/", List(group.App))
	group.GET("/:id", Get(group.App))
	group.POST("/create", Create(group.App))
}
