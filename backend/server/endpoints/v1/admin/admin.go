package admin

import (
	"github.com/NicoClack/cryptic-stash/backend/server/endpoints/v1/admin/invites"
	"github.com/NicoClack/cryptic-stash/backend/server/endpoints/v1/admin/self"
	"github.com/NicoClack/cryptic-stash/backend/server/endpoints/v1/admin/users"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
)

func ConfigureEndpoints(group *servercommon.Group) {
	// /login is registered in v1.go since it's unauthenticated
	users.ConfigureEndpoints(group.Group("/users"))
	invites.ConfigureEndpoints(group.Group("/invites"))
	self.ConfigureEndpoints(group.Group("/self"))
}
