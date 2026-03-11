package schedulers

import (
	"context"
	"math"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/common/dbcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/ent/periodictask"
	"github.com/google/uuid"
)

type DelayFuncContext struct {
	App     *common.App
	LastRan time.Time
	Context context.Context
}

// You should return the initial value for lastRan if lastRan is zero
type DelayFunc = func(delayCtx *DelayFuncContext) (nextRun time.Time, commit CommitDelayFunc)
type CommitDelayFunc = func(runTime time.Time, ctx context.Context)

func SimpleFixedInterval(interval time.Duration) DelayFunc {
	return func(delayCtx *DelayFuncContext) (time.Time, func(runTime time.Time, ctx context.Context)) {
		nextRun := delayCtx.LastRan
		now := delayCtx.App.Clock.Now()
		if nextRun.IsZero() {
			nextRun = now
		} else {
			nextRun = nextRun.Add(interval)
			skippedCalls := int64(math.Floor(float64(now.UnixNano()-nextRun.UnixNano()) / float64(interval)))
			if skippedCalls > 0 {
				newNextRun := nextRun.Add(interval * time.Duration(skippedCalls))
				delayCtx.App.Logger.Warn(
					"some calls were skipped for SimpleFixedInterval",
					"skippedCallCount", skippedCalls,
					"nextRun", nextRun,
					"newNextRun", newNextRun,
					"now", now,
				)
				nextRun = newNextRun
			}
		}

		return nextRun, func(runTime time.Time, ctx context.Context) {}
	}
}
func PersistentFixedInterval(periodicTaskName string, interval time.Duration) DelayFunc {
	periodicTaskID := uuid.Nil
	return func(delayCtx *DelayFuncContext) (time.Time, CommitDelayFunc) {
		lastRan := delayCtx.LastRan
		commit := func(runTime time.Time, ctx context.Context) {
			if periodicTaskID == uuid.Nil {
				periodicTask, stdErr := dbcommon.WithReadWriteTx(
					ctx,
					delayCtx.App.Database,
					func(tx *ent.Tx, ctx context.Context) (*ent.PeriodicTask, error) {
						now := delayCtx.App.Clock.Now()
						return tx.PeriodicTask.Create().
							SetCreatedAt(now).
							SetUpdatedAt(now).
							SetName(periodicTaskName).
							SetLastRanAt(runTime).
							Save(ctx)
					},
				)
				if stdErr == nil {
					periodicTaskID = periodicTask.ID
				} else {
					delayCtx.App.Logger.Error(
						"unable to create initial PeriodicTask object",
						"error", stdErr,
						"periodicTaskName", periodicTaskName,
					)
				}
			} else {
				stdErr := dbcommon.WithWriteTx(
					ctx,
					delayCtx.App.Database,
					func(tx *ent.Tx, ctx context.Context) error {
						return tx.PeriodicTask.UpdateOneID(periodicTaskID).
							SetUpdatedAt(delayCtx.App.Clock.Now()).
							SetLastRanAt(runTime).
							Exec(ctx)
					},
				)
				if stdErr != nil {
					delayCtx.App.Logger.Error(
						"unable to update lastRun of PeriodicTask object",
						"error", stdErr,
						"periodicTaskName", periodicTaskName,
						"periodicTaskID", periodicTaskID,
					)
				}
			}
		}

		if lastRan.IsZero() {
			periodicTask, stdErr := dbcommon.WithReadTx(
				delayCtx.Context,
				delayCtx.App.Database,
				func(tx *ent.Tx, ctx context.Context) (*ent.PeriodicTask, error) {
					return tx.PeriodicTask.Query().
						Where(periodictask.Name(periodicTaskName)).
						Only(ctx)
				},
			)
			if stdErr == nil {
				periodicTaskID = periodicTask.ID
				lastRan = periodicTask.LastRanAt
			} else {
				if !ent.IsNotFound(stdErr) {
					delayCtx.App.Logger.Error(
						"couldn't read PeriodicTask object on startup, assuming this is the first run",
						"error", stdErr,
						"periodicTaskName", periodicTaskName,
					)
				}
				return delayCtx.App.Clock.Now(), commit
			}
		}

		nextRun := lastRan.Add(interval)
		now := delayCtx.App.Clock.Now()
		skippedCalls := int64(math.Floor(float64(now.UnixNano()-nextRun.UnixNano()) / float64(interval)))
		if skippedCalls > 0 {
			// TODO: this still usually results in a double run
			newNextRun := nextRun.Add(interval * time.Duration(skippedCalls))
			delayCtx.App.Logger.Warn(
				"some calls were skipped for PersistentFixedInterval",
				"skippedCallCount", skippedCalls,
				"periodicTaskName", periodicTaskName,
				"nextRun", nextRun,
				"newNextRun", newNextRun,
				"now", now,
			)
			nextRun = newNextRun
		}
		return nextRun, commit
	}
}

// TODO: add function to run at regular times, with an offset argument (e.g run every day at 9am)
