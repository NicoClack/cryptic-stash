package sudo

import "github.com/NicoClack/cryptic-stash/backend/server/servercommon"

func ConfigureEndpoints(group *servercommon.Group) {
	group.POST("/start/", StartElevation(group.App))
	group.POST("/finish/", FinishElevation(group.App))
}
