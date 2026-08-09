package passkeys

import (
	"context"
	"net/http"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/common/dbcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/gin-gonic/gin"
)

type DisableTwoGroupAuthResponse struct {
	Errors []servercommon.ErrorDetail `json:"errors"`
}

func DisableTwoGroupAuth(app *servercommon.ServerApp) gin.HandlerFunc {
	return servercommon.NewHandler(func(ginCtx *gin.Context) error {
		actor := ginCtx.MustGet("actor").(*common.Actor)

		stdErr := dbcommon.WithWriteTx(
			ginCtx.Request.Context(),
			app.Database,
			func(tx *ent.Tx, ctx context.Context) error {
				wrappedErr := app.Auth.DisableTwoGroupAuth(actor.UserID, actor, tx, ctx)
				if wrappedErr != nil {
					return wrappedErr
				}
				return nil
			},
		)
		if stdErr != nil {
			return stdErr
		}

		ginCtx.JSON(http.StatusOK, &DisableTwoGroupAuthResponse{
			Errors: []servercommon.ErrorDetail{},
		})
		return nil
	})
}
