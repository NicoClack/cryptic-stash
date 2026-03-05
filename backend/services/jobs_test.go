package services_test

import (
	"sync"
	"testing"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/common/testcommon"
	"github.com/NicoClack/cryptic-stash/backend/services"
)

func TestJobsShutdown_HandlesConcurrentCalls(t *testing.T) {
	t.Parallel()

	app := &common.App{
		Env:      testcommon.DefaultEnv(),
		Database: testcommon.CreateDB(t),
		Logger:   testcommon.NewTestLogger(),
	}
	app.Database.Start()
	t.Cleanup(app.Database.Shutdown)
	jobService := services.NewJobs(app)
	jobService.Start()

	var wg sync.WaitGroup
	for range 100 {
		wg.Go(jobService.Shutdown)
	}
	wg.Wait()
}

func TestJobsShutdown_NoOpWhenNotStarted(t *testing.T) {
	t.Parallel()

	app := &common.App{
		Env:      testcommon.DefaultEnv(),
		Database: testcommon.CreateDB(t),
		Logger:   testcommon.NewTestLogger(),
	}
	app.Database.Start()
	t.Cleanup(app.Database.Shutdown)

	jobService := services.NewJobs(app)

	testcommon.AssertNoOp(t, jobService.Shutdown)
}

func TestJobsStart_SubsequentCallsAreNoOp(t *testing.T) {
	t.Parallel()

	app := &common.App{
		Env:      testcommon.DefaultEnv(),
		Database: testcommon.CreateDB(t),
		Logger:   testcommon.NewTestLogger(),
	}
	app.Database.Start()
	t.Cleanup(app.Database.Shutdown)

	jobService := services.NewJobs(app)
	t.Cleanup(jobService.Shutdown)

	jobService.Start()
	testcommon.AssertNoOp(t, jobService.Start)

	var wg sync.WaitGroup
	for range 5 {
		wg.Go(func() {
			testcommon.AssertNoOp(t, jobService.Start)
		})
	}
	wg.Wait()
}
