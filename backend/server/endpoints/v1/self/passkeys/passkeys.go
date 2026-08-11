package passkeys

import "github.com/NicoClack/cryptic-stash/backend/server/servercommon"

func ConfigureEndpoints(group *servercommon.Group) {
	group.GET("/", List(group.App))
	group.POST("/register/start/", RegisterStart(group.App))
	group.POST("/register/finish/", RegisterFinish(group.App))
	group.POST("/:id/rename/", Rename(group.App))
	group.POST("/:id/update-sudo/", UpdateSudo(group.App))
	group.POST("/:id/move-group/", MoveGroup(group.App))
	group.POST("/:id/delete/", Delete(group.App))
	group.POST("/disable-two-group-auth/", DisableTwoGroupAuth(group.App))
}
