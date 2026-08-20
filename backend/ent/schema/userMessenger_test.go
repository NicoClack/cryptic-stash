package schema_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common/testcommon"
	"github.com/stretchr/testify/require"
)

func TestUserMessenger_EncryptedFields(t *testing.T) {
	t.Parallel()

	t.Run("options can be read back", func(t *testing.T) {
		t.Parallel()
		db := testcommon.CreateDB(t)
		ctx := t.Context()
		dbClient := db.Client()

		userOb := createUser(t, dbClient, ctx)
		now := time.Now()
		options := json.RawMessage(`{"userId":"1234"}`)
		userMessengerOb := dbClient.UserMessenger.Create().
			SetCreatedAt(now).
			SetUpdatedAt(now).
			SetType("discord").
			SetVersion(1).
			SetOptions(options).
			SetUser(userOb).
			SaveX(ctx)

		userMessengerOb = dbClient.UserMessenger.GetX(ctx, userMessengerOb.ID)
		require.Equal(t, string(options), string(userMessengerOb.Options))
	})

	t.Run("nil options are stored as NULL", func(t *testing.T) {
		t.Parallel()
		db := testcommon.CreateDB(t)
		ctx := t.Context()
		dbClient := db.Client()

		userOb := createUser(t, dbClient, ctx)
		now := time.Now()
		userMessengerOb := dbClient.UserMessenger.Create().
			SetCreatedAt(now).
			SetUpdatedAt(now).
			SetType("develop").
			SetVersion(1).
			SetUser(userOb).
			SaveX(ctx)

		assertNullIsUnencrypted(t, db,
			"SELECT options FROM user_messengers WHERE id = ?", userMessengerOb.ID.String())
	})

	t.Run("json.RawMessage (options) are encrypted as raw JSON bytes, not double-encoded", func(t *testing.T) {
		t.Parallel()
		db := testcommon.CreateDB(t)
		ctx := t.Context()
		dbClient := db.Client()

		userOb := createUser(t, dbClient, ctx)
		options := json.RawMessage(`{"secret_token":"supersecret"}`)
		now := time.Now()
		userMessengerOb := dbClient.UserMessenger.Create().
			SetCreatedAt(now).
			SetUpdatedAt(now).
			SetType("smtp2go").
			SetVersion(1).
			SetOptions(options).
			SetUser(userOb).
			SaveX(ctx)

		assertEncryptedInDB(t, db,
			"SELECT options FROM user_messengers WHERE id = ?", []any{userMessengerOb.ID.String()},
			[]byte(options), "user_messenger_1",
			// ^ Not wrapped in JSON quotes
		)
	})
}
