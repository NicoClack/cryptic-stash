package testcommon

import (
	"log/slog"
	"os"

	"github.com/lmittmann/tint"
)

type TestLogger struct {
	*slog.Logger
}

func NewTestLogger() *TestLogger {
	logger := slog.New(tint.NewTextHandler(os.Stderr, &tint.Options{
		Level:      slog.LevelDebug,
		TimeFormat: "15:04:05.000",
	}))

	return &TestLogger{
		Logger: logger,
	}
}

func (l *TestLogger) Start() {}

func (l *TestLogger) Shutdown() {}
