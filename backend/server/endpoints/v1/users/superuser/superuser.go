package superuser

import "github.com/NicoClack/cryptic-stash/backend/server/servercommon"

func ConfigureEndpoints(group *servercommon.Group) {
	group.POST("/start-elevation/", StartElevation(group.App))
	group.POST("/finish-elevation/", FinishElevation(group.App))
}
