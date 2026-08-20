package testcommon

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/lmittmann/tint"
)

type TestLogger struct {
	*slog.Logger
}

type TestLoggerOptions struct {
	DisableFailOnError bool
}

func NewTestLogger(t *testing.T) *TestLogger {
	return NewTestLoggerWithOptions(t, TestLoggerOptions{})
}

func NewTestLoggerWithOptions(t *testing.T, options TestLoggerOptions) *TestLogger {
	tintHandler := tint.NewTextHandler(os.Stderr, &tint.Options{
		Level:      slog.LevelDebug,
		TimeFormat: "15:04:05.000",
	})

	return &TestLogger{
		Logger: slog.New(&TestHandler{
			t:           t,
			tintHandler: tintHandler,
			options:     options,
		}),
	}
}

func (l *TestLogger) Start() {}

func (l *TestLogger) Shutdown() {}

type TestHandler struct {
	t           *testing.T
	tintHandler slog.Handler
	options     TestLoggerOptions
}

func (handler *TestHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return handler.tintHandler.Enabled(ctx, level)
}
func (handler *TestHandler) Handle(ctx context.Context, record slog.Record) error {
	if !handler.options.DisableFailOnError && record.Level >= slog.LevelError {
		handler.t.Helper()
		handler.t.Errorf(
			"testcommon.TestLogger (often used by testcommon services by default): an error was logged: %s",
			record.Message,
		)
	}
	return handler.tintHandler.Handle(ctx, record)
}
func (handler *TestHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &TestHandler{
		t:           handler.t,
		tintHandler: handler.tintHandler.WithAttrs(attrs),
		options:     handler.options,
	}
}
func (handler *TestHandler) WithGroup(name string) slog.Handler {
	return &TestHandler{
		t:           handler.t,
		tintHandler: handler.tintHandler.WithGroup(name),
		options:     handler.options,
	}
}
