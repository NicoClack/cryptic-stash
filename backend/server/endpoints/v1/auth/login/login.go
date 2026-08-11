package login

import "github.com/NicoClack/cryptic-stash/backend/server/servercommon"

func ConfigureEndpoints(group *servercommon.Group) {
	group.POST("/start/", LoginStart(group.App))
	group.POST("/finish/", FinishLogin(group.App))
}
