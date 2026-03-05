package messengers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/common/dbcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/ent/user"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/gin-gonic/gin"
)

type EnableMessengerPayload struct {
	VersionedType string          `binding:"required" json:"versionedType"`
	Options       json.RawMessage `binding:"required" json:"options"`
}
type EnableMessengerResponse struct {
	Errors []servercommon.ErrorDetail `binding:"required" json:"errors"`
}

func EnableMessenger(app *servercommon.ServerApp) gin.HandlerFunc {
	return servercommon.NewHandler(func(ginCtx *gin.Context) error {
		body := EnableMessengerPayload{}
		if ctxErr := servercommon.ParseBody(&body, ginCtx); ctxErr != nil {
			return ctxErr
		}
		userID, ctxErr := servercommon.ParseObjectID(ginCtx.Param("id"))
		if ctxErr != nil {
			return ctxErr
		}

		stdErr := dbcommon.WithWriteTx(
			ginCtx.Request.Context(), app.Database,
			func(tx *ent.Tx, ctx context.Context) error {
				userOb, stdErr := tx.User.Query().
					Where(user.ID(userID)).
					Only(ctx)
				if stdErr != nil {
					return servercommon.Send404IfNotFound(stdErr)
				}

				wrappedErr := app.Messengers.EnableMessenger(
					userOb,
					body.VersionedType,
					body.Options,
					ctx,
				)
				if wrappedErr != nil {
					return wrappedErr
				}

				userOb, stdErr = tx.User.Query().
					Where(user.ID(userID)).
					WithMessengers().
					Only(ctx)
				if stdErr != nil {
					return stdErr
				}
				_, _, wrappedErr = app.Messengers.SendUsingAll(
					&common.Message{
						Type: common.MessageTest,
						User: userOb,
					},
					ctx,
				)
				if wrappedErr != nil {
					return wrappedErr
				}

				// The user most likely isn't trying to log in if they've coordinated this with their admin
				// And deleting the download sessions simplifies IsUserSufficientlyNotified
				return app.Core.InvalidateUserDownloadSessions(userOb.ID, ctx)
			},
		)
		if stdErr != nil {
			return stdErr
		}

		ginCtx.JSON(http.StatusOK, EnableMessengerResponse{
			Errors: []servercommon.ErrorDetail{},
		})
		return nil
	})
}
