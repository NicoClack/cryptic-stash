package testhelpers

import (
	"log/slog"
	"testing"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/common/testcommon"
	"github.com/NicoClack/cryptic-stash/backend/messengers"
	"github.com/NicoClack/cryptic-stash/backend/services"
	"github.com/jonboulle/clockwork"
)

type App struct {
	*common.App
	// The mock messenger that is created by default when AppOptions.MockMessengers is nil
	MockMessenger *MockMessenger
	TestDatabase  *testcommon.TestDatabase
}

type AppOptions struct {
	Env            *common.Env
	Clock          clockwork.Clock
	MockMessengers []*MockMessenger
}

// Mainly intended for integration tests, e.g endpoints and jobs
// Other tests should only initialise the specific services they need
func NewApp(t *testing.T, options *AppOptions) *App {
	t.Helper()

	if options == nil {
		options = &AppOptions{}
	}

	var clock clockwork.Clock
	if options.Clock == nil {
		clock = clockwork.NewRealClock()
	} else {
		clock = options.Clock
	}
	var env *common.Env
	if options.Env == nil {
		env = testcommon.DefaultEnv()
	} else {
		env = options.Env
	}
	mockMessengers := options.MockMessengers
	var mockMessenger *MockMessenger
	if mockMessengers == nil {
		mockMessenger = NewMockMessenger("MOCK_MESSENGER")
		mockMessengers = []*MockMessenger{mockMessenger}
	}

	app := &common.App{
		Env:   env,
		Clock: clock,
	}
	shutdownService := services.NewShutdown(
		app,
		services.NewShutdownTask(func() {
			if app.Scheduler != nil {
				app.Scheduler.Shutdown()
			}
		}, true),
		services.NewShutdownTask(func() {
			if app.Server != nil {
				app.Server.Shutdown()
			}
		}, true),

		services.NewShutdownTask(func() {
			if app.Jobs != nil {
				app.Jobs.Shutdown()
			}
		}, false),
		services.NewShutdownTask(func() {
			if app.Logger != nil {
				app.Logger.Shutdown() // Note: logs will still be written to the console after this, just not stored
			}
		}, false),
		services.NewShutdownTask(func() {
			if app.Database != nil {
				app.Database.Shutdown()
			}
		}, false),
	)
	app.ShutdownService = shutdownService
	t.Cleanup(func() {
		app.Shutdown("")
	})

	{
		logger := services.NewLogger(app)
		app.Logger = logger
		slog.SetDefault(logger.Logger)
	}
	app.RateLimiter = services.NewRateLimiter(app)
	app.Core = services.NewCore(app)
	db := testcommon.CreateDB(t)
	app.Database = db
	app.KeyValue = services.NewKeyValue(app)
	app.Database.Start()
	app.KeyValue.Init()
	app.Logger.Start()
	// TODO: TwoFactorActions
	{
		registerFuncs := make([]func(registry *messengers.Registry), 0, len(mockMessengers))
		for _, mockMessenger := range mockMessengers {
			registerFuncs = append(registerFuncs, mockMessenger.Register)
		}
		messengerService := services.NewMessengers(app, registerFuncs...)
		app.Messengers = messengerService
		app.Jobs = services.NewJobs(app, messengerService.RegisterJobs)
	}
	app.Server = services.NewServer(app)
	// TODO: scheduler?

	// We don't need to start the server since it doesn't bind to a port
	app.Jobs.Start()

	return &App{
		App:           app,
		MockMessenger: mockMessenger,
		TestDatabase:  db,
	}
}
