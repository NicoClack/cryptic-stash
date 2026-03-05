package jobs_test

import (
	"context"
	"testing"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/common/dbcommon"
	"github.com/NicoClack/cryptic-stash/backend/common/testcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/jobs"
	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/require"
)

func TestEngine_runsJob(t *testing.T) {
	t.Parallel()
	db := testcommon.CreateDB(t)
	defer db.Shutdown()
	app := &common.App{
		Database: db,
		Env:      testcommon.DefaultEnv(),
		Logger:   testcommon.NewTestLogger(),
		Clock:    clockwork.NewRealClock(),
	}
	type body = struct{}
	completeJobChan := make(chan struct{})
	registry := jobs.NewRegistry(app)
	registry.Register(&jobs.Definition{
		ID:      "test_job",
		Version: 1,
		Handler: func(ctx *jobs.Context) error {
			completeJobChan <- struct{}{}
			return nil
		},
		BodyType: &body{},
		Weight:   1,
	})
	engine := jobs.NewEngine(registry)
	go engine.Listen()
	defer engine.Shutdown()
	stdErr := dbcommon.WithWriteTx(
		t.Context(), db,
		func(tx *ent.Tx, ctx context.Context) error {
			_, wrappedErr := engine.Enqueue("test_job_1", &body{}, ctx)
			return wrappedErr
		},
	)
	require.NoError(t, stdErr)
	select {
	case <-completeJobChan:
	case <-time.After(2 * time.Second):
		t.Fatal("job did not complete in time")
	}
}

func TestEngine_retriesJob(t *testing.T) {
	t.Parallel()
	db := testcommon.CreateDB(t)
	defer db.Shutdown()
	app := &common.App{
		Clock:    clockwork.NewRealClock(),
		Database: db,
		Env:      testcommon.DefaultEnv(),
		Logger:   testcommon.NewTestLogger(),
	}
	type body = struct{}
	completeJobChan := make(chan struct{})
	registry := jobs.NewRegistry(app)
	attempt := 0
	registry.Register(&jobs.Definition{
		ID:      "test_job",
		Version: 1,
		Handler: func(ctx *jobs.Context) error {
			attempt++
			if attempt < 3 {
				return common.NewErrorWithCategories("temporary error").
					ConfigureRetries(2, 10*time.Millisecond, 2)
			}
			completeJobChan <- struct{}{}
			return nil
		},
		BodyType: &body{},
		Weight:   1,
	})
	engine := jobs.NewEngine(registry)
	go engine.Listen()
	defer engine.Shutdown()

	// TODO: use t.Log
	stdErr := dbcommon.WithWriteTx(
		t.Context(), db,
		func(tx *ent.Tx, ctx context.Context) error {
			_, wrappedErr := engine.Enqueue("test_job_1", &body{}, ctx)
			return wrappedErr
		},
	)
	require.NoError(t, stdErr)
	select {
	case <-completeJobChan:
	case <-time.After(2 * time.Second):
		t.Fatal("job did not complete in time")
	}
	require.Equal(t, 3, attempt)

	// TODO: not exiting gracefully
}
