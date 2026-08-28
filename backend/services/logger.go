package services

import (
	"log/slog"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/loggers"
)

type Logger struct {
	*slog.Logger
	Handler *loggers.Handler
}

// A subset of loggers.HandlerOptions, some options are based on env vars instead
type LoggerOptions struct {
	OnLog func(record slog.Record)
}

func NewLogger(app *common.App, options *LoggerOptions) *Logger {
	if options == nil {
		options = &LoggerOptions{}
	}

	handlerOptions := loggers.HandlerOptions{
		Level:          slog.LevelInfo,
		SaveToDatabase: app.Env.LOG_STORE_INTERVAL > 0,
	}
	if app.Env.IS_DEV {
		handlerOptions.Level = slog.LevelDebug
	}
	if options.OnLog != nil {
		handlerOptions.OnLog = options.OnLog
	}
	handler := loggers.NewHandler(app, handlerOptions)
	return &Logger{
		Logger:  slog.New(handler),
		Handler: handler,
	}
}

func (service *Logger) Start() {
	go service.Handler.Listen()
}
func (service *Logger) Shutdown() {
	service.Handler.Shutdown()
}
