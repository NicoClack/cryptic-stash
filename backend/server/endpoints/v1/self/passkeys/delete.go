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

type DeleteResponse struct {
	Errors []servercommon.ErrorDetail `json:"errors"`
}

func Delete(app *servercommon.ServerApp) gin.HandlerFunc {
	return servercommon.NewObjectIDHandler(func(id uuid.UUID, ginCtx *gin.Context) error {
		actor := ginCtx.MustGet("actor").(*common.Actor)
		sessionID := ginCtx.MustGet("session").(*ent.Session).ID

		stdErr := dbcommon.WithWriteTx(
			ginCtx.Request.Context(),
			app.Database,
			func(tx *ent.Tx, ctx context.Context) error {
				wrappedErr := app.Auth.DeletePasskey(id, sessionID, actor, tx, ctx)
				if wrappedErr != nil {
					return wrappedErr
				}
				return nil
			},
		)
		if stdErr != nil {
			return servercommon.ExpectError(
				stdErr, auth.ErrPasskeyDeleteConstraint,
				http.StatusConflict,
				&servercommon.ErrorDetail{
					Message: "can't delete a passkey that is currently in use by your session",
					Code:    "DELETE_CONSTRAINT",
				},
			).Expect(
				auth.ErrPasskeyNotFound,
				http.StatusNotFound,
				nil,
			)
		}

		ginCtx.JSON(http.StatusOK, &DeleteResponse{
			Errors: []servercommon.ErrorDetail{},
		})
		return nil
	})
}
