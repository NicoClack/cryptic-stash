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

func NewLogger(app *common.App) *Logger {
	handler := loggers.NewHandler(slog.LevelInfo, true, true, app)
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
