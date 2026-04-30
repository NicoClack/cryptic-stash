package messengers

import (
	"context"
	"net/http"

	"github.com/NicoClack/cryptic-stash/backend/common/dbcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/ent/user"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/gin-gonic/gin"
)

type DisableMessengerPayload struct {
	VersionedType string `binding:"required" json:"versionedType"`
}

type DisableMessengerResponse struct {
	Errors []servercommon.ErrorDetail `binding:"required" json:"errors"`
}

func DisableMessenger(app *servercommon.ServerApp) gin.HandlerFunc {
	return servercommon.NewHandler(func(ginCtx *gin.Context) error {
		body := DisableMessengerPayload{}
		if serverErr := servercommon.ParseBody(&body, ginCtx); serverErr != nil {
			return serverErr
		}
		userID, serverErr := servercommon.ParseObjectID(ginCtx.Param("id"))
		if serverErr != nil {
			return serverErr
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

				wrappedErr := app.Messengers.DisableMessenger(
					userOb,
					body.VersionedType,
					ctx,
				)
				if wrappedErr != nil {
					if ent.IsNotFound(wrappedErr) {
						return servercommon.NewError(wrappedErr).
							SetStatus(http.StatusBadRequest).
							AddDetail(servercommon.ErrorDetail{
								Message: "messenger not found for user",
								Code:    "MESSENGER_NOT_FOUND",
							}).
							DisableLogging()
					}
					return wrappedErr
				}
				return nil
			},
		)
		if stdErr != nil {
			return stdErr
		}

		ginCtx.JSON(http.StatusOK, DisableMessengerResponse{
			Errors: []servercommon.ErrorDetail{},
		})
		return nil
	})
}
