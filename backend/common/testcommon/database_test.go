package testcommon_test

import (
	"log/slog"
	"testing"

	"github.com/NicoClack/cryptic-stash/backend/common/testcommon"
	"github.com/stretchr/testify/require"
)

func TestTestDatabaseShutdown_IsIdempotent(t *testing.T) {
	t.Parallel()

	db := testcommon.CreateDB(t)
	db.Shutdown()
	db.Shutdown() // The test will fail if any errors are logged
	// Also called by the t.Cleanup in CreateDB
}

func TestCreateDBWithOptions_LoggerOverride(t *testing.T) {
	t.Parallel()

	override := slog.New(slog.DiscardHandler)
	db := testcommon.CreateDBWithOptions(t, testcommon.CreateDBOptions{
		Logger: override,
	})

	require.Same(t, override, db.DefaultLogger())
}
