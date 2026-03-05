package dbcommon_test

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/common/dbcommon"
	"github.com/NicoClack/cryptic-stash/backend/common/testcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/ent/job"
	"github.com/stretchr/testify/require"
)

func TestWithReadTx_AllowsConcurrentReads(t *testing.T) {
	t.Parallel()

	const READ_COUNT = 100
	db := testcommon.CreateDB(t)
	t.Cleanup(db.Shutdown)

	jobOb, stdErr := db.Client().Job.Create().
		SetType("test_job").
		SetCreatedAt(time.Now()).
		SetDueAt(time.Now()).
		SetOriginallyDueAt(time.Now()).
		SetVersion(1).
		SetPriority(1).
		SetWeight(1).
		SetBody(json.RawMessage("{}")).
		Save(t.Context())
	require.NoError(t, stdErr)
	jobID := jobOb.ID

	var wg sync.WaitGroup
	for range READ_COUNT {
		wg.Go(func() {
			_, stdErr := dbcommon.WithReadTx(
				t.Context(), db,
				func(tx *ent.Tx, ctx context.Context) (*ent.Job, error) {
					return tx.Job.Get(ctx, jobID)
				},
			)
			require.NoError(t, stdErr)
		})
	}

	testcommon.CallWithTimeout(t, wg.Wait, 250*time.Millisecond)
}
func TestWithWriteTx_NestedTransactions_ReturnsError(t *testing.T) {
	t.Parallel()

	db := testcommon.CreateDB(t)
	t.Cleanup(db.Shutdown)

	stdErr := dbcommon.WithWriteTx(
		t.Context(), db,
		func(tx *ent.Tx, ctx context.Context) error {
			return dbcommon.WithWriteTx(
				ctx, db,
				func(tx *ent.Tx, ctx context.Context) error {
					return nil
				},
			)
		},
	)

	require.Error(t, stdErr)
	require.ErrorIs(t, stdErr, dbcommon.ErrUnexpectedTransaction)
}

// SQLite isn't suitable if the program has many more concurrent writes than this
func TestWithWriteTx_Supports50ConcurrentWrites(t *testing.T) {
	t.Parallel()

	JOB_COUNT := 50
	db := testcommon.CreateDB(t) // TODO: use a disk database to more accurately measure performance
	t.Cleanup(db.Shutdown)

	var wg sync.WaitGroup
	createJob := func() {
		stdErr := dbcommon.WithWriteTx(
			t.Context(), db,
			func(tx *ent.Tx, ctx context.Context) error {
				return tx.Job.Create().
					SetType("test_job").
					SetCreatedAt(time.Now()).
					SetDueAt(time.Now()).
					SetOriginallyDueAt(time.Now()).
					SetVersion(1).
					SetPriority(1).
					SetWeight(1).
					SetBody(json.RawMessage("{}")).
					Exec(ctx)
			},
		)
		require.NoError(t, stdErr)
	}
	for range JOB_COUNT {
		wg.Go(createJob)
	}
	wg.Wait()
	count, stdErr := db.Client().Job.Query().Count(t.Context())
	require.NoError(t, stdErr)
	require.Equal(t, JOB_COUNT, count)
}
func TestWithWriteTx_supports25CollidingIncrements(t *testing.T) {
	t.Parallel()

	INCREMENT_COUNT := 25
	db := testcommon.CreateDB(t)
	defer db.Shutdown()

	stdErr := db.Client().Job.Create().
		SetType("counter").
		SetCreatedAt(time.Now()).
		SetDueAt(time.Now()).
		SetOriginallyDueAt(time.Now()).
		SetVersion(1).
		SetPriority(1).
		SetWeight(1).
		SetBody(json.RawMessage(`{"count":0}`)).
		Exec(t.Context())
	require.NoError(t, stdErr)

	var errCount atomic.Int32
	var wg sync.WaitGroup
	for range INCREMENT_COUNT {
		wg.Go(func() {
			stdErr := dbcommon.WithWriteTx(
				t.Context(), db,
				func(tx *ent.Tx, ctx context.Context) error {
					job, stdErr := tx.Job.Query().Where(job.TypeEQ("counter")).Only(ctx)
					if stdErr != nil {
						errCount.Add(1)
						return common.ErrWrapperDatabase.Wrap(stdErr)
					}
					// Extend the window when this transaction hasn't got a write lock
					time.Sleep(10 * time.Millisecond)

					var body struct {
						Count int `json:"count"`
					}
					stdErr = json.Unmarshal(job.Body, &body)
					if stdErr != nil {
						errCount.Add(1)
						return stdErr
					}
					body.Count++
					newBody, stdErr := json.Marshal(body)
					if stdErr != nil {
						errCount.Add(1)
						return stdErr
					}
					stdErr = job.Update().SetBody(json.RawMessage(newBody)).Exec(ctx)
					if stdErr != nil {
						errCount.Add(1)
						return common.ErrWrapperDatabase.Wrap(stdErr)
					}
					return nil
				},
			)
			require.NoError(t, stdErr)
		})
	}
	wg.Wait()

	jobOb, err := db.Client().Job.Query().Where(job.TypeEQ("counter")).Only(t.Context())
	require.NoError(t, err)
	var body struct {
		Count int `json:"count"`
	}
	stdErr = json.Unmarshal(jobOb.Body, &body)
	require.NoError(t, stdErr)
	require.Equal(t, INCREMENT_COUNT, body.Count)
	// Expect at least a few errors to have been retried (it should be way more than this on average)
	require.GreaterOrEqual(t, errCount.Load(), int32(5))
}
