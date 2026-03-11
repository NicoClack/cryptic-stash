package users

import (
	"context"
	"net/http"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/common/dbcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/ent/user"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/gin-gonic/gin"
)

type AdminLockPayload struct {
	Username string `binding:"required,min=1,max=32" json:"username"`
}
type AdminLockResponse struct {
	Errors []servercommon.ErrorDetail `binding:"required" json:"errors"`
}

func AdminLock(app *servercommon.ServerApp) gin.HandlerFunc {
	return servercommon.NewHandler(func(ginCtx *gin.Context) error {
		body := AdminLockPayload{}
		if ctxErr := servercommon.ParseBody(&body, ginCtx); ctxErr != nil {
			return ctxErr
		}
		if serverErr := servercommon.ValidateUsername(body.Username); serverErr != nil {
			return serverErr
		}

		return dbcommon.WithWriteTx(
			ginCtx.Request.Context(), app.Database,
			func(tx *ent.Tx, ctx context.Context) error {
				now := app.Clock.Now()
				userOb, stdErr := tx.User.Query().
					Where(user.Username(body.Username)).
					Only(ctx)
				if stdErr != nil {
					return servercommon.Send404IfNotFound(stdErr)
				}
				userOb, stdErr = userOb.Update().
					SetUpdatedAt(now).
					SetLocked(true).
					ClearLockedUntil().
					Save(ctx)
				if stdErr != nil {
					return servercommon.Send404IfNotFound(stdErr)
				}

				wrappedErr := app.Core.InvalidateUserDownloadSessions(userOb.ID, ctx)
				if wrappedErr != nil {
					return wrappedErr
				}
				_, _, wrappedErr = app.Messengers.SendUsingAll(
					&common.Message{
						Type: common.MessageLock,
						User: userOb,
					},
					ctx,
				)
				if wrappedErr != nil {
					return wrappedErr
				}

				ginCtx.JSON(http.StatusOK, AdminLockResponse{
					Errors: []servercommon.ErrorDetail{},
				})
				return nil
			},
		)
	})
}
