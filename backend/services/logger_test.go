package services_test

import (
	"sync"
	"testing"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/common/testcommon"
	"github.com/NicoClack/cryptic-stash/backend/services"
)

func TestLoggerShutdown_HandlesConcurrentCalls(t *testing.T) {
	t.Parallel()

	app := &common.App{
		Env:      testcommon.DefaultEnv(),
		Database: testcommon.CreateDB(t),
	}
	app.Database.Start()
	t.Cleanup(app.Database.Shutdown)
	app.Logger = services.NewLogger(app)
	app.Logger.Start()

	var wg sync.WaitGroup
	for range 100 {
		wg.Go(app.Logger.Shutdown)
	}
	wg.Wait()
}

func TestLoggerShutdown_NoOpWhenNotStarted(t *testing.T) {
	t.Parallel()

	app := &common.App{
		Env: testcommon.DefaultEnv(),
	}
	app.Logger = services.NewLogger(app)

	testcommon.AssertNoOp(t, app.Logger.Shutdown)
}

func TestLoggerStart_SubsequentCallsAreNoOp(t *testing.T) {
	t.Parallel()

	app := &common.App{
		Env: testcommon.DefaultEnv(),
	}
	app.Logger = services.NewLogger(app)
	t.Cleanup(app.Logger.Shutdown)

	app.Logger.Start()
	testcommon.AssertNoOp(t, app.Logger.Start)

	var wg sync.WaitGroup
	for range 5 {
		wg.Go(func() {
			testcommon.AssertNoOp(t, app.Logger.Start)
		})
	}
	wg.Wait()
}
