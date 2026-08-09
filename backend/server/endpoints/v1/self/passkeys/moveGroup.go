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

type MoveGroupPayload struct {
	IsSecondGroup bool `json:"isSecondGroup"`
}
type MoveGroupResponse struct {
	Errors []servercommon.ErrorDetail `json:"errors"`
}

func MoveGroup(app *servercommon.ServerApp) gin.HandlerFunc {
	return servercommon.NewObjectIDHandler(func(id uuid.UUID, ginCtx *gin.Context) error {
		actor := ginCtx.MustGet("actor").(*common.Actor)
		sessionOb := ginCtx.MustGet("session").(*ent.Session)

		body := MoveGroupPayload{}
		if serverErr := servercommon.ParseBody(&body, ginCtx); serverErr != nil {
			return serverErr
		}

		stdErr := dbcommon.WithWriteTx(
			ginCtx.Request.Context(),
			app.Database,
			func(tx *ent.Tx, ctx context.Context) error {
				wrappedErr := app.Auth.MovePasskeyGroup(
					id,
					actor.UserID, sessionOb.PasskeyID, sessionOb.ElevationPasskeyID,
					body.IsSecondGroup,
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
				stdErr, auth.ErrPasskeyGroupMoveConstraint,
				http.StatusConflict,
				&servercommon.ErrorDetail{
					Message: "can't move this passkey, as doing so may lock you out of sudo mode. to enable two group auth, " +
						"use two different passkeys in your session and move one of them to the second group. " +
						"to move back, first log in with different passkeys or disable two group auth entirely",
					Code: "GROUP_MOVE_CONSTRAINT",
				},
			).Expect(
				auth.ErrPasskeyNotFound,
				http.StatusNotFound,
				nil,
			)
		}

		ginCtx.JSON(http.StatusOK, &MoveGroupResponse{
			Errors: []servercommon.ErrorDetail{},
		})
		return nil
	})
}
