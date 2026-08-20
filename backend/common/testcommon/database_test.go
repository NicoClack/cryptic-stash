package testcommon_test

import (
	"io"
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
}

func TestCreateDBWithOptions_LoggerOverride(t *testing.T) {
	t.Parallel()

	override := slog.New(slog.NewTextHandler(io.Discard, nil))
	db := testcommon.CreateDBWithOptions(t, testcommon.CreateDBOptions{
		Logger: override,
	})

	require.Same(t, override, db.DefaultLogger())
}
