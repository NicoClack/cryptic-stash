package testcommon

// Utilities for testing using a raw SQL connection as opposed to an Ent client, e.g in the schema tests.
// Encrypted fields are normally handled automatically.

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"io"
	"testing"

	"github.com/NicoClack/cryptic-stash/backend/ent/schema"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/hkdf"
)

// Returns a single row, single column value, or nil.
func RawColumnQuery(t *testing.T, sqlDB *sql.DB, query string, args ...any) []byte {
	t.Helper()
	var result []byte
	stdErr := sqlDB.QueryRowContext(t.Context(), query, args...).Scan(&result)
	require.NoError(t, stdErr)
	return result
}

// Gets the key that the schema will have used to encrypt a field with the given key name
func GetSchemaEncryptionKey(keyName string) []byte {
	baseKey := make([]byte, 32) // All tests need to have env.BASE_ENCRYPTION_KEY set to this
	reader := hkdf.New(sha256.New, baseKey, nil, []byte(keyName))
	key := make([]byte, 32)
	if _, stdErr := io.ReadFull(reader, key); stdErr != nil {
		panic(fmt.Sprintf("testcommon.GetSchemaEncryptionKey: failed to derive key for %s: %v", keyName, stdErr))
	}
	return key
}

// Decrypts a field that uses schema.EncryptedField
// Note: structs are stored using JSON, so they will still need decoding after this
func DecryptDBField(t *testing.T, encryptedBytes []byte, keyName string) []byte {
	t.Helper()
	key := GetSchemaEncryptionKey(keyName)
	plaintext, stdErr := schema.Decrypt(encryptedBytes, key)
	require.NoError(t, stdErr)
	return plaintext
}
