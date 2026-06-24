package servercommon

import (
	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/gin-gonic/gin"
)

type ServerApp struct {
	*common.App
	Router                  *gin.Engine
	AdminMiddleware         gin.HandlerFunc
	DefaultAuthMiddleware   gin.HandlerFunc
	SuperUserModeMiddleware gin.HandlerFunc
}

type Group struct {
	*gin.RouterGroup
	App *ServerApp
}

func (group *Group) Group(relativePath string) *Group {
	return &Group{
		RouterGroup: group.RouterGroup.Group(relativePath),
		App:         group.App,
	}
}
