package dbcommon_test

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common/dbcommon"
	"github.com/NicoClack/cryptic-stash/backend/common/testcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/ent/job"
	"github.com/stretchr/testify/require"
)

type countingReadTxDB struct {
	*testcommon.TestDatabase
	startTxCount *atomic.Int32
}

func (db *countingReadTxDB) ReadTx(ctx context.Context) (*ent.Tx, error) {
	db.startTxCount.Add(1)
	return db.TestDatabase.ReadTx(ctx)
}

// Write transactions use BEGIN IMMEDIATE, which serialises writers. Read
// transactions use a deferred BEGIN, so multiple readers should be able to run
// concurrently and complete quickly.
func TestWithReadTx_ReadsConcurrently(t *testing.T) {
	t.Parallel()

	const READ_COUNT = int32(100)
	const CALLBACK_SLEEP = 10 * time.Millisecond
	db := testcommon.CreateDB(t)
	t.Cleanup(db.Shutdown)

	now := time.Now()
	jobOb, stdErr := db.Client().Job.Create().
		SetCreatedAt(now).
		SetUpdatedAt(now).
		SetType("test_job").
		SetDueAt(now).
		SetOriginallyDueAt(now).
		SetVersion(1).
		SetPriority(1).
		SetWeight(1).
		SetBody(json.RawMessage("{}")).
		Save(t.Context())
	require.NoError(t, stdErr)

	var startTxCount atomic.Int32
	countingDB := &countingReadTxDB{
		TestDatabase: db,
		startTxCount: &startTxCount,
	}
	var callbackCallCount atomic.Int32
	var concurrentCountMu sync.Mutex
	var concurrentCount int
	var maxConcurrentCount int

	start := time.Now()
	var wg sync.WaitGroup
	for range READ_COUNT {
		wg.Go(func() {
			concurrentCountMu.Lock()
			concurrentCount++
			if concurrentCount > maxConcurrentCount {
				maxConcurrentCount = concurrentCount
			}
			concurrentCountMu.Unlock()
			defer func() {
				concurrentCountMu.Lock()
				concurrentCount--
				concurrentCountMu.Unlock()
			}()

			_, stdErr := dbcommon.WithReadTx(
				t.Context(), countingDB,
				func(tx *ent.Tx, ctx context.Context) (*ent.Job, error) {
					callbackCallCount.Add(1)
					// Hold the read transaction open briefly so concurrent reads overlap
					time.Sleep(CALLBACK_SLEEP)
					return tx.Job.Get(ctx, jobOb.ID)
				},
			)
			require.NoError(t, stdErr)
		})
	}
	wg.Wait()
	elapsed := time.Since(start)

	// More than read must have run concurrently at some point
	require.Greater(t, maxConcurrentCount, 1)
	// Nothing should have had to be retried
	require.Equal(t, READ_COUNT, startTxCount.Load())
	require.Equal(t, READ_COUNT, callbackCallCount.Load())
	// Concurrent reads take ~CALLBACK_SLEEP whereas serialised reads would take READ_COUNT*CALLBACK_SLEEP
	require.Less(t, elapsed, time.Duration(READ_COUNT)*CALLBACK_SLEEP)
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
				now := time.Now()
				return tx.Job.Create().
					SetCreatedAt(now).
					SetUpdatedAt(now).
					SetType("test_job").
					SetDueAt(now).
					SetOriginallyDueAt(now).
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

type countingWriteTxDB struct {
	*testcommon.TestDatabase
	startTxAttemptCount *atomic.Int32
}

func (db *countingWriteTxDB) WriteTx(ctx context.Context) (*ent.Tx, error) {
	db.startTxAttemptCount.Add(1)
	return db.TestDatabase.WriteTx(ctx)
}

func TestWithWriteTx_supports25CollidingIncrements(t *testing.T) {
	t.Parallel()

	INCREMENT_COUNT := int32(25)
	db := testcommon.CreateDB(t)
	t.Cleanup(db.Shutdown)

	var startTxAttemptCount atomic.Int32
	var callbackCallCount atomic.Int32
	countingDB := &countingWriteTxDB{
		TestDatabase:        db,
		startTxAttemptCount: &startTxAttemptCount,
	}

	now := time.Now()
	stdErr := db.Client().Job.Create().
		SetType("counter").
		SetCreatedAt(now).
		SetUpdatedAt(now).
		SetDueAt(now).
		SetOriginallyDueAt(now).
		SetVersion(1).
		SetPriority(1).
		SetWeight(1).
		SetBody(json.RawMessage(`{"count":0}`)).
		Exec(t.Context())
	require.NoError(t, stdErr)

	var wg sync.WaitGroup
	for range INCREMENT_COUNT {
		wg.Go(func() {
			stdErr := dbcommon.WithWriteTx(
				t.Context(), countingDB,
				func(tx *ent.Tx, ctx context.Context) error {
					callbackCallCount.Add(1)
					job, stdErr := tx.Job.Query().Where(job.TypeEQ("counter")).Only(ctx)
					if stdErr != nil {
						return stdErr
					}
					// These transactions now immediately have a write lock, but just in case there's an issue in the future,
					// increase the chance of a collision
					time.Sleep(10 * time.Millisecond)

					var body struct {
						Count int `json:"count"`
					}
					stdErr = json.Unmarshal(job.Body, &body)
					if stdErr != nil {
						return stdErr
					}
					body.Count++
					newBody, stdErr := json.Marshal(body)
					if stdErr != nil {
						return stdErr
					}
					return job.Update().
						SetUpdatedAt(now).
						SetBody(json.RawMessage(newBody)).
						Exec(ctx)
				},
			)
			require.NoError(t, stdErr)
		})
	}
	wg.Wait()

	jobOb, stdErr := db.Client().Job.Query().Where(job.TypeEQ("counter")).Only(t.Context())
	require.NoError(t, stdErr)
	var body struct {
		Count int32 `json:"count"`
	}
	stdErr = json.Unmarshal(jobOb.Body, &body)
	require.NoError(t, stdErr)
	require.Equal(t, INCREMENT_COUNT, body.Count)
	// startTxAttemptCount should be much greater than INCREMENT_COUNT because
	// the starting of a number of transactions should have had to be retried
	require.Greater(t, startTxAttemptCount.Load(), INCREMENT_COUNT)
	// Because BEGIN IMMEDIATE is used, there should have only been one callback running
	// at a time, and therefore none of them should have had to be retried
	require.Equal(t, INCREMENT_COUNT, callbackCallCount.Load())
}
