package passkeys

import "github.com/NicoClack/cryptic-stash/backend/server/servercommon"

func ConfigureEndpoints(group *servercommon.Group) {
	group.POST("/register/start/", RegisterStart(group.App))
	group.POST("/register/finish/", RegisterFinish(group.App))
}
