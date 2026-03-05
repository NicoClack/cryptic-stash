package services_test

import (
	"sync"
	"testing"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/common/testcommon"
	"github.com/NicoClack/cryptic-stash/backend/common/testcommon/mocks"
	"github.com/NicoClack/cryptic-stash/backend/services"
	"github.com/jonboulle/clockwork"
)

func TestSchedulerShutdown_HandlesConcurrentCalls(t *testing.T) {
	t.Parallel()

	app := &common.App{
		Clock:            clockwork.NewRealClock(),
		Env:              testcommon.DefaultEnv(),
		Database:         testcommon.CreateDB(t),
		Logger:           testcommon.NewTestLogger(),
		Core:             mocks.NewEmptyCoreService(),
		TwoFactorActions: mocks.NewEmptyTwoFactorActionService(),
		RateLimiter:      mocks.NewEmptyRateLimiterService(),
	}
	app.Database.Start()
	t.Cleanup(app.Database.Shutdown)
	app.Scheduler = services.NewScheduler(app)
	app.Scheduler.Start()

	time.Sleep(10 * time.Millisecond) // Ensure the tasks have had a chance to start

	var wg sync.WaitGroup
	for range 100 {
		wg.Go(app.Scheduler.Shutdown)
	}
	wg.Wait()
}

func TestSchedulerShutdown_NoOpWhenNotStarted(t *testing.T) {
	t.Parallel()

	app := &common.App{
		Clock:            clockwork.NewRealClock(),
		Env:              testcommon.DefaultEnv(),
		Database:         testcommon.CreateDB(t),
		Logger:           testcommon.NewTestLogger(),
		Core:             mocks.NewEmptyCoreService(),
		TwoFactorActions: mocks.NewEmptyTwoFactorActionService(),
		RateLimiter:      mocks.NewEmptyRateLimiterService(),
	}
	app.Database.Start()
	t.Cleanup(app.Database.Shutdown)

	app.Scheduler = services.NewScheduler(app)

	testcommon.AssertNoOp(t, app.Scheduler.Shutdown)
}

func TestSchedulerStart_SubsequentCallsAreNoOp(t *testing.T) {
	t.Parallel()

	app := &common.App{
		Clock:            clockwork.NewRealClock(),
		Env:              testcommon.DefaultEnv(),
		Database:         testcommon.CreateDB(t),
		Logger:           testcommon.NewTestLogger(),
		Core:             mocks.NewEmptyCoreService(),
		TwoFactorActions: mocks.NewEmptyTwoFactorActionService(),
		RateLimiter:      mocks.NewEmptyRateLimiterService(),
	}
	app.Database.Start()
	t.Cleanup(app.Database.Shutdown)

	app.Scheduler = services.NewScheduler(app)
	t.Cleanup(app.Scheduler.Shutdown)

	app.Scheduler.Start()
	testcommon.AssertNoOp(t, app.Scheduler.Start)

	var wg sync.WaitGroup
	for range 5 {
		wg.Go(func() {
			testcommon.AssertNoOp(t, app.Scheduler.Start)
		})
	}
	wg.Wait()
}
