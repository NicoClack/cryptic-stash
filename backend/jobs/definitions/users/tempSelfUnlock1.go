package users

import (
	"context"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/common/dbcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/ent/user"
	"github.com/NicoClack/cryptic-stash/backend/jobs"
)

type TempSelfUnlock1Body struct {
	Username string `binding:"required" json:"username"`
}

func TempSelfUnlock1(app *common.App) *jobs.Definition {
	return &jobs.Definition{
		ID:            "TEMP_SELF_UNLOCK",
		Version:       1,
		Weight:        1,
		NoParallelize: true,
		BodyType:      &TempSelfUnlock1Body{},
		Handler: func(jobCtx *jobs.Context) error {
			body := &TempSelfLock1Body{}
			jobErr := jobCtx.Decode(body)
			if jobErr != nil {
				return jobErr
			}

			return dbcommon.WithWriteTx(
				jobCtx.Context, app.Database,
				func(tx *ent.Tx, ctx context.Context) error {
					now := app.Clock.Now()
					userOb, stdErr := tx.User.Query().
						Where(user.Username(body.Username)).
						Only(ctx)
					if stdErr != nil {
						return stdErr
					}
					if userOb.LockedUntil == nil {
						jobCtx.Logger.Info(
							"didn't need to unlock the user because they are already unlocked",
							"userID", userOb.ID,
						)
						return nil
					}
					userOb, stdErr = userOb.Update().
						SetUpdatedAt(now).
						ClearLockedUntil().
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
							Type: common.MessageSelfUnlock,
							User: userOb,
							Time: body.Until,
						},
						ctx,
					)
					if wrappedErr != nil {
						return wrappedErr
					}
					jobCtx.Logger.Info(
						"user was unlocked because the self-lock expired",
						"userID", userOb.ID,
						"isLocked", userOb.Locked,
					)
					return nil
				},
			)
		},
	}
}
