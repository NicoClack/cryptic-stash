package schema_test

import (
	"testing"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/common/testcommon"
	"github.com/stretchr/testify/require"
)

func TestStash_EncryptedFields(t *testing.T) {
	t.Parallel()

	t.Run("encryptionDataKey can be read back", func(t *testing.T) {
		t.Parallel()
		db := testcommon.CreateDB(t)
		t.Cleanup(db.Shutdown)
		ctx := t.Context()
		dbClient := db.Client()

		userOb := createUser(t, dbClient, ctx)
		dataEncryptionKey := []byte("32-bytes-of-test-data-for-key!!!")
		now := time.Now()
		stashOb := dbClient.Stash.Create().
			SetCreatedAt(now).
			SetUpdatedAt(now).
			SetPublicName("test stash").
			// These would normally be encrypted, but it doesn't matter
			SetContent([]byte("file content")).
			SetFileName([]byte("file.txt")).
			//
			SetEncryptionDataKey(dataEncryptionKey).
			SetPasswordSalt(make([]byte, 16)).
			SetHashTime(1).SetHashMemory(1024).SetHashThreads(1).
			SetUser(userOb).
			SetDownloadSessionsValidFrom(now).
			SaveX(ctx)

		stashOb = dbClient.Stash.GetX(ctx, stashOb.ID)
		require.Equal(t, dataEncryptionKey, stashOb.EncryptionDataKey)
	})

	t.Run("bytes field (encryptionDataKey) is encrypted directly, not JSON-encoded", func(t *testing.T) {
		t.Parallel()
		db := testcommon.CreateDB(t)
		t.Cleanup(db.Shutdown)
		ctx := t.Context()
		dbClient := db.Client()

		userOb := createUser(t, dbClient, ctx)
		keyBytes := common.CryptoRandomBytes(32)
		now := time.Now()
		stashOb := dbClient.Stash.Create().
			SetCreatedAt(now).
			SetUpdatedAt(now).
			SetPublicName("direct bytes stash").
			// These would normally be encrypted, but it doesn't matter
			SetContent([]byte("content")).
			SetFileName([]byte("file.txt")).
			//
			SetEncryptionDataKey(keyBytes).
			SetPasswordSalt(make([]byte, 16)).
			SetHashTime(1).SetHashMemory(1024).SetHashThreads(1).
			SetUser(userOb).
			SetDownloadSessionsValidFrom(now).
			SaveX(ctx)

		idStr := stashOb.ID.String()
		assertEncryptedInDB(t, db,
			"SELECT encryption_data_key FROM stashes WHERE id = ?", []any{idStr},
			keyBytes, "stash_1",
			// ^ Not wrapped in JSON quotes
		)
	})
}
