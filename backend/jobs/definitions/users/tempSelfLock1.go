package users

import (
	"context"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/common/dbcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/ent/user"
	"github.com/NicoClack/cryptic-stash/backend/jobs"
)

type TempSelfLock1Body struct {
	Username string    `binding:"required" json:"username"`
	Until    time.Time `binding:"required" json:"until"`
}

func TempSelfLock1(app *common.App) *jobs.Definition {
	return &jobs.Definition{
		ID:            "TEMP_SELF_LOCK",
		Version:       1,
		Priority:      jobs.HighPriority,
		Weight:        1,
		NoParallelize: true,
		BodyType:      &TempSelfLock1Body{},
		Handler: func(jobCtx *jobs.Context) error {
			body := &TempSelfLock1Body{}
			jobErr := jobCtx.Decode(body)
			if jobErr != nil {
				return jobErr
			}

			return dbcommon.WithWriteTx(
				jobCtx.Context, app.Database,
				func(tx *ent.Tx, ctx context.Context) error {
					userOb, stdErr := tx.User.Query().
						Where(user.Username(body.Username)).
						Only(ctx)
					if stdErr != nil {
						return stdErr
					}
					userOb, stdErr = userOb.Update().
						SetLockedUntil(body.Until).
						Save(ctx)
					if stdErr != nil {
						return stdErr
					}

					wrappedErr := app.Core.InvalidateUserDownloadSessions(userOb.ID, ctx)
					if wrappedErr != nil {
						return wrappedErr
					}
					_, _, wrappedErr = app.Messengers.SendUsingAll(
						&common.Message{
							Type: common.MessageSelfLock,
							User: userOb,
							Time: body.Until,
						},
						ctx,
					)
					if wrappedErr != nil {
						return wrappedErr
					}

					jobOb, wrappedErr := app.Jobs.EnqueueWithModifier(
						"users/TEMP_SELF_UNLOCK_1",
						//exhaustruct:enforce
						&TempSelfUnlock1Body{
							Username: body.Username,
						},
						func(jobCreate *ent.JobCreate) {
							jobCreate.SetDueAt(body.Until)
						},
						ctx,
					)
					if wrappedErr != nil {
						return wrappedErr
					}

					jobCtx.Logger.Info(
						"user has successfully self-locked",
						"userID", userOb.ID,
						"lockedUntil", body.Until,
						"unlockJobID", jobOb.ID,
					)
					return nil
				},
			)
		},
	}
}
