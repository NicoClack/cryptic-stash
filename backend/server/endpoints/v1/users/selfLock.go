package users

import (
	"context"
	"net/http"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/common/dbcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/ent/user"
	userjobs "github.com/NicoClack/cryptic-stash/backend/jobs/definitions/users"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/NicoClack/cryptic-stash/backend/twofactoractions"
	"github.com/gin-gonic/gin"
)

const MAX_SELF_LOCK_DURATION = 14 * (24 * time.Hour)

type SelfLockPayload struct {
	Username string    `binding:"required,min=1,max=32,alphanum,lowercase" json:"username"`
	Password string    `binding:"required,min=8,max=256"                   json:"password"` // #nosec G117
	Until    time.Time `binding:"required"                                 json:"until"`
}
type SelfLockResponse struct {
	Errors            []servercommon.ErrorDetail `binding:"required" json:"errors"`
	TwoFactorActionID string                     `                   json:"twoFactorActionId"`
}

func SelfLock(app *servercommon.ServerApp) gin.HandlerFunc {
	clock := app.Clock

	return servercommon.NewHandler(func(ginCtx *gin.Context) error {
		body := SelfLockPayload{}
		if ctxErr := servercommon.ParseBody(&body, ginCtx); ctxErr != nil {
			return ctxErr
		}
		if body.Username == common.AdminUsername {
			return servercommon.NewInvalidUsernameError()
		}
		until := clock.Now().Add(
			min(
				body.Until.Sub(clock.Now()), // Convert to duration
				MAX_SELF_LOCK_DURATION,
			),
		)

		userOb, stdErr := dbcommon.WithReadTx(
			ginCtx.Request.Context(), app.Database,
			func(tx *ent.Tx, ctx context.Context) (*ent.User, error) {
				userOb, stdErr := tx.User.Query().
					Where(user.Username(body.Username)).
					WithStash().
					Only(ctx)
				if stdErr != nil {
					return nil, servercommon.SendUnauthorizedIfNotFound(stdErr)
				}

				return userOb, nil
			},
		)
		if stdErr != nil {
			return stdErr
		}
		if app.Core.IsUserLocked(userOb) {
			return servercommon.NewUnauthorizedError()
		}

		stashOb := userOb.Edges.Stash
		if stashOb == nil {
			return servercommon.NewUnauthorizedError()
		}
		stashKek := app.Core.HashPassword(
			body.Password,
			stashOb.PasswordSalt,
			&common.PasswordHashSettings{
				Time:    stashOb.HashTime,
				Memory:  stashOb.HashMemory,
				Threads: stashOb.HashThreads,
			},
		)
		_, wrappedErr := app.Core.Decrypt(stashOb.EncryptionDataKey, stashKek)
		if wrappedErr != nil {
			return servercommon.NewUnauthorizedError()
		}

		return dbcommon.WithWriteTx(
			ginCtx.Request.Context(), app.Database,
			func(tx *ent.Tx, ctx context.Context) error {
				action, code, wrappedErr := app.TwoFactorActions.Create(
					"users/TEMP_SELF_LOCK_1",
					clock.Now().Add(twofactoractions.DEFAULT_CODE_LIFETIME),
					//exhaustruct:enforce
					&userjobs.TempSelfLock1Body{
						Username: body.Username,
						Until:    until,
					},
					ctx,
				)
				if wrappedErr != nil {
					return wrappedErr
				}

				_, _, wrappedErr = app.Messengers.SendUsingAll(
					&common.Message{
						Type: common.Message2FA,
						User: userOb,
						Code: code,
					},
					ctx,
				)
				if wrappedErr != nil {
					return wrappedErr
				}

				// TODO: wait for job to run and return error if it fails?
				ginCtx.JSON(http.StatusOK, SelfLockResponse{
					Errors:            []servercommon.ErrorDetail{},
					TwoFactorActionID: action.ID.String(),
				})
				return nil
			},
		)
	})
}
