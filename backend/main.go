package main

import (
	"log/slog"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/services"
	"github.com/jonboulle/clockwork"
)

func main() {
	app := &common.App{
		Env:   services.LoadEnvironmentVariables(),
		Clock: clockwork.NewRealClock(),
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
				app.Logger.Shutdown() // Note: logs will still be sent to stdout/stderr, just not stored
			}
		}, false),
		services.NewShutdownTask(func() {
			if app.Database != nil {
				app.Database.Shutdown()
			}
		}, false),
	)
	app.ShutdownService = shutdownService

	{
		logger := services.NewLogger(app)
		app.Logger = logger
		slog.SetDefault(logger.Logger)
	}
	app.TempKeyValue = services.NewTempKeyValue(app)
	app.Auth = services.NewAuth(app)
	app.RateLimiter = services.NewRateLimiter(app)
	app.Core = services.NewCore(app)
	app.Setup = services.NewSetupService(app)
	app.Database = services.NewDatabase(app)
	app.KeyValue = services.NewKeyValue(app)
	app.Database.Start()
	app.KeyValue.Init()
	app.Logger.Start()
	app.TwoFactorActions = services.NewTwoFactorActions(app)
	{
		messengerService := services.NewMessengers(app)
		app.Messengers = messengerService
		app.Jobs = services.NewJobs(app, messengerService.RegisterJobs)
		// TODO: check for stalled jobs and mark them as failed before the scheduler starts
	}
	app.Server = services.NewServer(app)
	app.Scheduler = services.NewScheduler(app)

	app.Scheduler.Start() // Note: initialises some state, e.g the rotating admin code
	app.Server.Start()
	app.Jobs.Start()

	shutdownService.Listen()
}
