package services_test

import (
	"sync"
	"testing"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/common/testcommon"
	"github.com/NicoClack/cryptic-stash/backend/services"
	"github.com/stretchr/testify/require"
)

func TestDatabaseShutdown_HandlesConcurrentCalls(t *testing.T) {
	t.Parallel()

	env := testcommon.DefaultEnv()
	env.MOUNT_PATH = t.TempDir()
	app := &common.App{
		Env: env,
	}
	app.Logger = services.NewLogger(app, nil)
	db := services.NewDatabase(app)
	db.Start()

	var wg sync.WaitGroup
	for range 100 {
		wg.Go(db.Shutdown)
	}
	wg.Wait()
}

func TestDatabaseShutdown_NoOpWhenNotStarted(t *testing.T) {
	t.Parallel()

	env := testcommon.DefaultEnv()
	env.MOUNT_PATH = t.TempDir()
	app := &common.App{
		Env: env,
	}
	app.Logger = services.NewLogger(app, nil)
	db := services.NewDatabase(app)

	testcommon.AssertNoOp(t, db.Shutdown)
}

func TestDatabaseStart_SubsequentCallsAreNoOp(t *testing.T) {
	t.Parallel()

	env := testcommon.DefaultEnv()
	env.MOUNT_PATH = t.TempDir()
	app := &common.App{
		Env: env,
	}
	app.Logger = services.NewLogger(app, nil)
	db := services.NewDatabase(app)
	t.Cleanup(db.Shutdown)

	db.Start()
	client1 := db.Client()
	testcommon.AssertNoOp(t, db.Start)
	client2 := db.Client()
	require.Same(t, client1, client2)

	var wg sync.WaitGroup
	for range 5 {
		wg.Go(func() {
			testcommon.AssertNoOp(t, db.Start)
			client := db.Client()
			if client != client1 {
				t.Error("concurrent Start() call returned different client")
			}
		})
	}
	wg.Wait()
}
