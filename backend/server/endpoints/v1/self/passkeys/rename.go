package passkeys

import (
	"context"
	"net/http"

	"github.com/NicoClack/cryptic-stash/backend/auth"
	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/common/dbcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type RenamePayload struct {
	Name string `binding:"required" json:"name"`
}
type RenameResponse struct {
	Errors []servercommon.ErrorDetail `json:"errors"`
}

func Rename(app *servercommon.ServerApp) gin.HandlerFunc {
	return servercommon.NewObjectIDHandler(func(id uuid.UUID, ginCtx *gin.Context) error {
		actor := ginCtx.MustGet("actor").(*common.Actor)

		body := RenamePayload{}
		if serverErr := servercommon.ParseBody(&body, ginCtx); serverErr != nil {
			return serverErr
		}

		stdErr := dbcommon.WithWriteTx(
			ginCtx.Request.Context(),
			app.Database,
			func(tx *ent.Tx, ctx context.Context) error {
				wrappedErr := app.Auth.RenamePasskey(id, body.Name, actor, tx, ctx)
				if wrappedErr != nil {
					return wrappedErr
				}
				return nil
			},
		)
		if stdErr != nil {
			// TODO: should the package return its own error for this?
			if dbcommon.IsUniqueConstraintError(stdErr, "passkeys") {
				return servercommon.NewError(stdErr).
					SetStatus(http.StatusConflict).
					AddDetail(servercommon.ErrorDetail{
						Message: "passkey name already exists",
						Code:    "DUPLICATE_PASSKEY_NAME",
					}).
					DisableLogging()
			}

			return servercommon.ExpectError(
				stdErr, auth.ErrPasskeyNotFound,
				http.StatusNotFound,
				&servercommon.ErrorDetail{
					Message: "passkey not found",
					Code:    "PASSKEY_NOT_FOUND",
				},
			)
		}

		ginCtx.JSON(http.StatusOK, &RenameResponse{
			Errors: []servercommon.ErrorDetail{},
		})
		return nil
	})
}
