package dbcommon_test

import (
	"context"
	"errors"
	"testing"

	"github.com/NicoClack/cryptic-stash/backend/common/dbcommon"
	"github.com/NicoClack/cryptic-stash/backend/common/testcommon"
	"github.com/NicoClack/cryptic-stash/backend/core"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/google/uuid"
	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/require"
)

func TestIsUniqueConstraintError_DuplicateUser(t *testing.T) {
	t.Parallel()

	db := testcommon.CreateDB(t)
	t.Cleanup(db.Shutdown)
	ctx := t.Context()
	clock := clockwork.NewRealClock()
	testcommon.NewDummyUser(1, db.Client(), ctx, clock)

	now := clock.Now()
	_, stdErr := db.Client().User.Create().
		SetUsername("user1").
		SetCreatedAt(now).
		SetUpdatedAt(now).
		Save(ctx)
	require.Error(t, stdErr)

	require.True(t, dbcommon.IsUniqueConstraintError(stdErr))
	require.True(t, dbcommon.IsUniqueConstraintError(stdErr, "users"))
	require.False(t, dbcommon.IsUniqueConstraintError(stdErr, "stashes")) // Unrelated table
	require.False(t, dbcommon.IsUniqueConstraintError(stdErr, "nonexistent_table"))
}
func TestIsUniqueConstraintError_NonConstraintErrors(t *testing.T) {
	t.Parallel()

	require.False(t, dbcommon.IsUniqueConstraintError(
		errors.New("some random error mentioning users table"),
	), "users")

	require.False(t, dbcommon.IsUniqueConstraintError(
		errors.New("UNIQUE constraint failed: users.username"),
		// ^ Right message but not *ent.ConstraintError
	), "users")
	require.False(t, dbcommon.IsUniqueConstraintError(nil))
	require.False(t, dbcommon.IsUniqueConstraintError(nil, "users"))
}

func TestIsUniqueConstraintError_UnwrapsTxCallbackError(t *testing.T) {
	t.Parallel()

	db := testcommon.CreateDB(t)
	t.Cleanup(db.Shutdown)
	ctx := t.Context()
	clock := clockwork.NewRealClock()
	testcommon.NewDummyUser(1, db.Client(), ctx, clock)

	stdErr := dbcommon.WithWriteTx(
		ctx, db,
		func(tx *ent.Tx, ctx context.Context) error {
			now := clock.Now()
			_, stdErr := tx.User.Create().
				SetUsername("user1").
				SetCreatedAt(now).
				SetUpdatedAt(now).
				Save(ctx)
			return stdErr
		},
	)
	require.Error(t, stdErr)

	// Ensure the common.WrappedError is unwrapped to *ent.ConstraintError
	require.True(t, dbcommon.IsUniqueConstraintError(stdErr))
	require.True(t, dbcommon.IsUniqueConstraintError(stdErr, "users"))
	require.False(t, dbcommon.IsUniqueConstraintError(stdErr, "stashes"))
}

func TestIsUniqueConstraintError_ForeignKeyViolation_ReturnsFalse(t *testing.T) {
	t.Parallel()

	db := testcommon.CreateDB(t)
	t.Cleanup(db.Shutdown)
	ctx := t.Context()
	clock := clockwork.NewRealClock()

	_, stdErr := db.Client().Stash.Create().
		SetCreatedAt(clock.Now()).
		SetUpdatedAt(clock.Now()).
		SetUserID(uuid.New()). // Will fail
		SetPublicName("test-stash").
		SetContent([]byte("some content")).
		SetFileName([]byte("file.txt")).
		SetEncryptionDataKey(core.GenerateEncryptionKey()).
		SetPasswordSalt(core.GenerateSalt()).
		SetHashTime(1).SetHashMemory(1024).SetHashThreads(1).
		SetDownloadSessionsValidFrom(clock.Now()).
		Save(ctx)
	require.Error(t, stdErr)

	// These are a type of constraint error (FK) but not a unique constraint error
	require.False(t, dbcommon.IsUniqueConstraintError(stdErr))
	require.False(t, dbcommon.IsUniqueConstraintError(stdErr, "stashes"))
	require.False(t, dbcommon.IsUniqueConstraintError(stdErr, "users"))
}
