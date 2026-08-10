package testcommon

import (
	"crypto/sha256"
	"encoding/base64"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// Sometimes UUIDs are opaquely sent to the client like this (e.g in WebAuthn):
// uuid.UUID -> []byte(uuid) -> base64.RawURLEncoding.EncodeToString([]byte(uuid))
// In which case the string is base64 rather than a formatted UUID string,
// so we need to decode it as base64 and then treat the bytes as a UUID.
func MustDecodeRawURLBase64UUID(t *testing.T, encoded string) uuid.UUID {
	t.Helper()

	decodedBytes, stdErr := base64.RawURLEncoding.DecodeString(encoded)
	require.NoError(t, stdErr)
	return uuid.UUID(decodedBytes)
}

func HashSessionToken(t *testing.T, token string) []byte {
	t.Helper()

	decodedToken, stdErr := base64.RawURLEncoding.DecodeString(token)
	require.NoError(t, stdErr)
	hashed := sha256.Sum256(decodedToken)
	return hashed[:]
}
