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

type UpdateSudoPayload struct {
	AllowSudo bool `json:"allowSudo"`
}
type UpdateSudoResponse struct {
	Errors []servercommon.ErrorDetail `json:"errors"`
}

func UpdateSudo(app *servercommon.ServerApp) gin.HandlerFunc {
	return servercommon.NewObjectIDHandler(func(id uuid.UUID, ginCtx *gin.Context) error {
		actor := ginCtx.MustGet("actor").(*common.Actor)
		sessionOb := ginCtx.MustGet("session").(*ent.Session)

		body := UpdateSudoPayload{}
		if serverErr := servercommon.ParseBody(&body, ginCtx); serverErr != nil {
			return serverErr
		}

		stdErr := dbcommon.WithWriteTx(
			ginCtx.Request.Context(),
			app.Database,
			func(tx *ent.Tx, ctx context.Context) error {
				wrappedErr := app.Auth.SetPasskeyAllowSudo(
					id,
					sessionOb.PasskeyID, sessionOb.ElevationPasskeyID,
					body.AllowSudo,
					actor, tx, ctx,
				)
				if wrappedErr != nil {
					return wrappedErr
				}
				return nil
			},
		)
		if stdErr != nil {
			return servercommon.ExpectError(
				stdErr, auth.ErrPasskeySudoConstraint,
				http.StatusConflict,
				&servercommon.ErrorDetail{
					Message: "can't disable sudo on this passkey, as doing so would remove sudo access from your session",
					Code:    "SUDO_CONSTRAINT",
				},
			).Expect(
				auth.ErrPasskeyNotFound,
				http.StatusNotFound,
				nil,
			)
		}

		ginCtx.JSON(http.StatusOK, &UpdateSudoResponse{
			Errors: []servercommon.ErrorDetail{},
		})
		return nil
	})
}
