package testcommon_test

import (
	"testing"

	"github.com/NicoClack/cryptic-stash/backend/common/testcommon"
)

func TestNewTestLogger_WarningsDontFail(t *testing.T) {
	t.Parallel()

	logger := testcommon.NewTestLogger(t)
	logger.Warn("a warning") // Shouldn't fail the test
}

func TestNewTestLoggerWithOptions_DisableFailOnError(t *testing.T) {
	t.Parallel()

	logger := testcommon.NewTestLoggerWithOptions(t, testcommon.TestLoggerOptions{
		DisableFailOnError: true,
	})
	logger.Error("an error was logged") // Shouldn't fail the test
}
