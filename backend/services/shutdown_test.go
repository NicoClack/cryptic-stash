package services_test

import (
	"testing"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/common/testcommon"
	"github.com/NicoClack/cryptic-stash/backend/services"
	"github.com/jonboulle/clockwork"
)

func TestShutdown_HandlesConcurrentCalls(t *testing.T) {
	t.Parallel()

	app := &common.App{
		Clock:  clockwork.NewRealClock(),
		Logger: testcommon.NewTestLogger(t),
	}
	shutdownService := services.NewShutdown(
		app,
		services.NewShutdownTask(func() {
			app.Logger.Info("doing some shutdown task...")
			app.Clock.Sleep(10 * time.Millisecond)
		}, false),
	)
	app.ShutdownService = shutdownService

	for range 100 {
		go app.Shutdown("")
	}
	shutdownService.ListenForShutdownCall()
}

func TestShutdown_NoOpWhenNotStarted(t *testing.T) {
	t.Parallel()

	app := &common.App{
		Clock:  clockwork.NewRealClock(),
		Logger: testcommon.NewTestLogger(t),
	}
	shutdownService := services.NewShutdown(app)
	app.ShutdownService = shutdownService

	select {
	case <-common.NewCallbackChannel(func() {
		shutdownService.Shutdown("")
	}):
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("Shutdown service blocked when not started; expected no-op")
	}
}
