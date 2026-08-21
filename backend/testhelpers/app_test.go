package testhelpers_test

import (
	"fmt"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/testhelpers"
	"github.com/stretchr/testify/require"
)

func TestNewDefaultOnLogCallback_OnlyFailsForErrors(t *testing.T) {
	t.Parallel()

	errorfCalls := []string{}
	callback := testhelpers.NewDefaultOnLogCallback(func(format string, args ...any) {
		errorfCalls = append(errorfCalls, fmt.Sprintf(format, args...))
	})

	callback(slog.NewRecord(time.Now(), slog.LevelDebug, "debug message", 0))
	callback(slog.NewRecord(time.Now(), slog.LevelInfo, "info message", 0))
	callback(slog.NewRecord(time.Now(), slog.LevelWarn, "warning message", 0))
	require.Empty(t, errorfCalls)

	callback(slog.NewRecord(time.Now(), slog.LevelError, "error message", 0))
	require.Len(t, errorfCalls, 1)
	require.Equal(
		t,
		"testhelpers.NewApp: test failed because an error was logged",
		// ^ The message is generic to reduce clutter in test output
		// If you need to test the message, provide a custom OnLog callback to NewApp
		errorfCalls[0],
	)
}

func TestNewApp_OnLogOverride(t *testing.T) {
	t.Parallel()

	levels := []slog.Level{}
	var mu sync.Mutex // NewApp starts goroutines
	app := testhelpers.NewApp(t, &testhelpers.AppOptions{
		OnLog: func(record slog.Record) {
			mu.Lock()
			defer mu.Unlock()
			levels = append(levels, record.Level)
		},
	})
	mu.Lock()
	levels = []slog.Level{} // Clear the startup logs
	mu.Unlock()

	app.Logger.Info("info")
	app.Logger.Error("error")

	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, []slog.Level{
		slog.LevelInfo,
		slog.LevelError,
	}, levels)
}
