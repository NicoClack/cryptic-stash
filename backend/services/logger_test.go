package services_test

import (
	"sync"
	"testing"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/common/testcommon"
	"github.com/NicoClack/cryptic-stash/backend/services"
	"github.com/stretchr/testify/require"
)

func TestLoggerShutdown_HandlesConcurrentCalls(t *testing.T) {
	t.Parallel()

	app := &common.App{
		Env:      testcommon.DefaultEnv(),
		Database: testcommon.CreateDB(t),
	}
	app.Logger = services.NewLogger(app, nil)
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
	app.Logger = services.NewLogger(app, nil)

	testcommon.AssertNoOp(t, app.Logger.Shutdown)
}

func TestLoggerStart_SubsequentCallsAreNoOp(t *testing.T) {
	t.Parallel()

	app := &common.App{
		Env: testcommon.DefaultEnv(),
	}
	app.Logger = services.NewLogger(app, nil)
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

func TestNewLogger_SaveToDatabaseToggledByLogStoreInterval(t *testing.T) {
	t.Parallel()

	env := testcommon.DefaultEnv()
	app := &common.App{Env: env}

	require.Greater(
		t,
		env.LOG_STORE_INTERVAL,
		time.Duration(0),
		// ^ Logs are stored by default in tests, but only on shutdown due to the high value of LOG_STORE_INTERVAL
	)
	require.True(t, services.NewLogger(app, nil).Handler.SaveToDatabase)

	env.LOG_STORE_INTERVAL = 0
	require.False(t, services.NewLogger(app, nil).Handler.SaveToDatabase)

	env.LOG_STORE_INTERVAL = -time.Second
	require.False(t, services.NewLogger(app, nil).Handler.SaveToDatabase)
}
