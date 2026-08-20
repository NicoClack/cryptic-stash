package schema_test

import (
	"encoding/json"
	"testing"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/common/testcommon"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/stretchr/testify/require"
)

func TestPasskey_EncryptedFields(t *testing.T) {
	t.Parallel()

	t.Run("credential can be read back", func(t *testing.T) {
		t.Parallel()
		db := testcommon.CreateDB(t)
		ctx := t.Context()
		dbClient := db.Client()

		userOb := createUser(t, dbClient, ctx)
		credential := webauthn.Credential{
			ID:        common.CryptoRandomBytes(16),
			PublicKey: []byte("test-public-key-material"),
		}
		passkeyOb := createPasskey(t, dbClient, ctx, userOb, credential)

		passkeyOb = dbClient.Passkey.GetX(ctx, passkeyOb.ID)
		require.Equal(t, credential.ID, passkeyOb.Credential.ID)
		require.Equal(t, credential.PublicKey, passkeyOb.Credential.PublicKey)
	})

	t.Run("credential is JSON encoded, then encrypted in database", func(t *testing.T) {
		t.Parallel()
		db := testcommon.CreateDB(t)
		ctx := t.Context()
		dbClient := db.Client()

		userOb := createUser(t, dbClient, ctx)
		credential := webauthn.Credential{
			ID:        common.CryptoRandomBytes(16),
			PublicKey: []byte("secret-public-key"),
		}
		passkeyOb := createPasskey(t, dbClient, ctx, userOb, credential)

		expectedJSON, stdErr := json.Marshal(credential)
		require.NoError(t, stdErr)
		assertEncryptedInDB(t, db,
			"SELECT credential FROM passkeys WHERE id = ?", []any{passkeyOb.ID.String()},
			expectedJSON, "auth_1",
		)
	})
}
