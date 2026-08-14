package users

import (
	"net/http"

	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TODO: remove

type AuthTestResponse struct {
	Errors    []servercommon.ErrorDetail `json:"errors"`
	SessionID uuid.UUID                  `json:"sessionId"`
	UserID    uuid.UUID                  `json:"userId"`
	Username  string                     `json:"username"`
	IsSudo    bool                       `json:"isSudo"`
}

func AuthTest(app *servercommon.ServerApp) gin.HandlerFunc {
	return servercommon.NewHandler(func(ginCtx *gin.Context) error {
		sessionOb := ginCtx.MustGet("session").(*ent.Session)
		userOb := ginCtx.MustGet("user").(*ent.User)

		ginCtx.JSON(http.StatusOK, AuthTestResponse{
			Errors:    []servercommon.ErrorDetail{},
			SessionID: sessionOb.ID,
			UserID:    userOb.ID,
			Username:  userOb.Username,
			IsSudo:    sessionOb.IsSudo,
		})
		return nil
	})
}
