package services

import (
	"context"
	"runtime"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/common/dbcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/schedulers"
)

type Scheduler struct {
	*schedulers.Engine
}

func NewScheduler(app *common.App) *Scheduler {
	engine := schedulers.NewEngine(app)

	engine.Register(
		func(taskContext *schedulers.TaskContext) {
			app.Logger.Info("sending active download session reminders...")
			stdErr := dbcommon.WithWriteTx(
				taskContext.Context, taskContext.App.Database,
				func(tx *ent.Tx, ctx context.Context) error {
					return app.Core.SendActiveDownloadSessionReminders(ctx)
				},
			)
			if stdErr != nil {
				taskContext.App.Logger.Error(
					"unable to send active download session reminders!",
					"error", stdErr,
				)
			}
		},
		schedulers.PersistentFixedInterval(
			"SEND_ACTIVE_SESSION_REMINDERS",
			app.Env.ACTIVE_DOWNLOAD_SESSION_REMINDER_INTERVAL,
		),
	)
	engine.Register(
		func(taskContext *schedulers.TaskContext) {
			app.Logger.Info("running cleanup task...")
			stdErr := dbcommon.WithWriteTx(
				taskContext.Context, taskContext.App.Database,
				func(tx *ent.Tx, ctx context.Context) error {
					wrappedErr := app.Core.DeleteExpiredDownloadSessions(ctx)
					if wrappedErr != nil {
						return wrappedErr
					}
					wrappedErr = app.TwoFactorActions.DeleteExpiredActions(ctx)
					if wrappedErr != nil {
						return wrappedErr
					}

					return nil
				},
			)
			if stdErr != nil {
				taskContext.App.Logger.Error(
					"cleanup task's transaction failed",
					"error", stdErr,
				)
			}
			app.RateLimiter.DeleteInactiveUsers()
		},
		schedulers.SimpleFixedInterval(app.Env.CLEAN_UP_INTERVAL),
	)
	if app.Env.FULL_GC_INTERVAL > 0 {
		engine.Register(
			func(taskContext *schedulers.TaskContext) {
				app.Logger.Info("running full garbage collection...")
				runtime.GC()
			},
			schedulers.SimpleFixedInterval(app.Env.FULL_GC_INTERVAL),
		)
	}

	return &Scheduler{
		Engine: engine,
	}
}
func (scheduler *Scheduler) Start() {
	go scheduler.Run()
}
