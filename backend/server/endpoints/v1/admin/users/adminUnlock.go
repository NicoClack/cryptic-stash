package users

import (
	"context"
	"net/http"

	"github.com/NicoClack/cryptic-stash/backend/common/dbcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/gin-gonic/gin"
)

type AdminUnlockPayload struct {
	Username string `binding:"required,min=1,max=32" json:"username"`
}
type AdminUnlockResponse struct {
	Errors []servercommon.ErrorDetail `binding:"required" json:"errors"`
}

func AdminUnlock(app *servercommon.ServerApp) gin.HandlerFunc {
	return servercommon.NewHandler(func(ginCtx *gin.Context) error {
		body := AdminUnlockPayload{}
		if serverErr := servercommon.ParseBody(&body, ginCtx); serverErr != nil {
			return serverErr
		}
		if serverErr := servercommon.ValidateUserEmail(body.Username); serverErr != nil {
			return serverErr
		}

		return dbcommon.WithWriteTx(
			ginCtx.Request.Context(), app.Database,
			func(tx *ent.Tx, ctx context.Context) error {
				// TODO: take stash ID rather than username
				panic("not implemented")

				ginCtx.JSON(http.StatusOK, AdminUnlockResponse{
					Errors: []servercommon.ErrorDetail{},
				})
				return nil
			},
		)
	})
}
