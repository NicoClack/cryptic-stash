package schema_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common/testcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/ent/schema"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/stretchr/testify/require"
)

var (
// validHashedCode is a 32-byte value used to satisfy SHA-256 hash field constraints.
// validHashedCode  = make([]byte, 32)
// validHashedToken = make([]byte, 32)
)

// Queries a raw column in the database and asserts that its likely to be encrypted
func assertEncryptedInDB(
	t *testing.T,
	db *testcommon.TestDatabase,
	query string,
	queryArgs []any,
	wantPlaintext []byte,
	keyName string,
) {
	t.Helper()
	rawBytes := testcommon.RawColumnQuery(t, db.DB(), query, queryArgs...)
	require.NotNil(t, rawBytes, "expected non-NULL encrypted value in DB")
	require.False(
		t,
		bytes.Contains(rawBytes, wantPlaintext),
		"database value must not contain the plaintext",
	)
	require.Greater(
		t,
		len(rawBytes),
		len(wantPlaintext)+schema.GCMNonceSize,
		"database value must contain a GCM nonce and an authentication tag, "+
			"implied by the value being longer than the plaintext + nonce",
	)
	decrypted := testcommon.DecryptDBField(t, rawBytes, keyName)
	require.Equal(t, wantPlaintext, decrypted, "decrypted value must match original plaintext")
}

func assertNullIsUnencrypted(t *testing.T, db *testcommon.TestDatabase, query string, queryArgs ...any) {
	t.Helper()
	rawBytes := testcommon.RawColumnQuery(t, db.DB(), query, queryArgs...)
	require.Nil(t, rawBytes, "optional encrypted field with nil value should store NULL in the database")
}

// TODO: testcommon.NewDummyUser could maybe replace some of this
func createUser(t *testing.T, client *ent.Client, ctx context.Context) *ent.User {
	t.Helper()
	now := time.Now()
	return client.User.Create().
		SetUsername("testuser").
		SetCreatedAt(now).
		SetUpdatedAt(now).
		SaveX(ctx)
}

func createPasskey(
	t *testing.T,
	client *ent.Client,
	ctx context.Context,
	userOb *ent.User,
	credential webauthn.Credential,
) *ent.Passkey {
	t.Helper()
	now := time.Now()
	return client.Passkey.Create().
		SetName("test-passkey").
		SetAllowSudo(false).
		SetCredentialID(credential.ID).
		SetCredential(credential).
		SetCreatedAt(now).
		SetUpdatedAt(now).
		SetUser(userOb).
		SaveX(ctx)
}

func createUserAndStash(t *testing.T, client *ent.Client, ctx context.Context) *ent.Stash {
	t.Helper()
	now := time.Now()
	userOb := createUser(t, client, ctx)
	stashOb := client.Stash.Create().
		SetCreatedAt(now).
		SetUpdatedAt(now).
		SetPublicName("test stash").

		// Normally encrypted, but doesn't matter for these tests
		SetContent([]byte("content")).
		SetFileName([]byte("file.txt")).
		//
		SetEncryptionDataKey(make([]byte, 32)).
		SetPasswordSalt(make([]byte, 16)).
		SetHashTime(1).
		SetHashMemory(1024).
		SetHashThreads(1).
		SetUser(userOb).
		SetDownloadSessionsValidFrom(now).
		SaveX(ctx)
	return stashOb
}
