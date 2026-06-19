package schema_test

import (
	"strings"
	"testing"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/common/testcommon"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/stretchr/testify/require"
)

func TestSession_EncryptedFields(t *testing.T) {
	t.Parallel()

	t.Run("userAgent + ip can be read back", func(t *testing.T) {
		t.Parallel()
		db := testcommon.CreateDB(t)
		t.Cleanup(db.Shutdown)
		ctx := t.Context()
		dbClient := db.Client()

		userOb := createUser(t, dbClient, ctx)
		passkeyOb := createPasskey(t, dbClient, ctx, userOb, webauthn.Credential{ID: common.CryptoRandomBytes(16)})
		userAgent := "Mozilla/5.0 (X11; Linux x86_64; rv:130.0) Gecko/20100101 Firefox/130.0"
		ip := "203.0.113.42"
		now := time.Now()
		sessionOb := dbClient.Session.Create().
			SetCreatedAt(now).
			SetUpdatedAt(now).
			SetHashedToken(common.CryptoRandomBytes(32)).
			SetExpiresAt(now.Add(time.Hour)).
			SetUserAgent(userAgent).
			SetIP(ip).
			SetUser(userOb).
			SetPasskey(passkeyOb).
			SaveX(ctx)

		sessionOb = dbClient.Session.GetX(ctx, sessionOb.ID)
		require.Equal(t, userAgent, sessionOb.UserAgent)
		require.Equal(t, ip, sessionOb.IP)
	})

	t.Run("string fields (userAgent + ip) are encrypted directly, not JSON-encoded", func(t *testing.T) {
		t.Parallel()
		db := testcommon.CreateDB(t)
		t.Cleanup(db.Shutdown)
		ctx := t.Context()
		dbClient := db.Client()

		userOb := createUser(t, dbClient, ctx)
		passkeyOb := createPasskey(t, dbClient, ctx, userOb, webauthn.Credential{ID: common.CryptoRandomBytes(16)})
		userAgent := "Mozilla/5.0 (encrypted-test)"
		ip := "10.0.0.1"
		now := time.Now()
		sessionOb := dbClient.Session.Create().
			SetCreatedAt(now).
			SetUpdatedAt(now).
			SetHashedToken(common.CryptoRandomBytes(32)).
			SetExpiresAt(now.Add(time.Hour)).
			SetUserAgent(userAgent).
			SetIP(ip).
			SetUser(userOb).
			SetPasskey(passkeyOb).
			SaveX(ctx)

		idStr := sessionOb.ID.String()
		assertEncryptedInDB(t, db,
			"SELECT user_agent FROM sessions WHERE id = ?", []any{idStr},
			[]byte(userAgent), "security_pii_logging_1",
			// ^ Not wrapped in JSON quotes
		)
		assertEncryptedInDB(t, db,
			"SELECT ip FROM sessions WHERE id = ?", []any{idStr},
			[]byte(ip), "security_pii_logging_1",
			// ^ Not wrapped in JSON quotes
		)
	})

	t.Run("min length constraint on userAgent", func(t *testing.T) {
		t.Parallel()
		db := testcommon.CreateDB(t)
		t.Cleanup(db.Shutdown)
		ctx := t.Context()
		dbClient := db.Client()

		userOb := createUser(t, dbClient, ctx)
		passkeyOb := createPasskey(t, dbClient, ctx, userOb, webauthn.Credential{ID: common.CryptoRandomBytes(16)})
		now := time.Now()
		_, stdErr := dbClient.Session.Create().
			SetCreatedAt(now).
			SetUpdatedAt(now).
			SetHashedToken(common.CryptoRandomBytes(32)).
			SetExpiresAt(now.Add(time.Hour)).
			SetUserAgent("").
			SetIP("1.2.3.4").
			SetUser(userOb).
			SetPasskey(passkeyOb).
			Save(ctx)

		require.ErrorContains(t, stdErr, "EncryptedValidator: value is less than the required length of 1")
	})

	t.Run("max length constraint on ip", func(t *testing.T) {
		t.Parallel()
		db := testcommon.CreateDB(t)
		t.Cleanup(db.Shutdown)
		ctx := t.Context()
		dbClient := db.Client()

		userOb := createUser(t, dbClient, ctx)
		passkeyOb := createPasskey(t, dbClient, ctx, userOb, webauthn.Credential{ID: common.CryptoRandomBytes(16)})
		now := time.Now()
		_, stdErr := dbClient.Session.Create().
			SetCreatedAt(now).
			SetUpdatedAt(now).
			SetHashedToken(common.CryptoRandomBytes(32)).
			SetExpiresAt(now.Add(time.Hour)).
			SetUserAgent("Mozilla/5.0").
			SetIP(strings.Repeat("x", 46)). // exceeds max of 45
			SetUser(userOb).
			SetPasskey(passkeyOb).
			Save(ctx)

		require.ErrorContains(t, stdErr, "EncryptedValidator: value is greater than the allowed length of 45")
	})

	t.Run("max length constraint on userAgent", func(t *testing.T) {
		t.Parallel()
		db := testcommon.CreateDB(t)
		t.Cleanup(db.Shutdown)
		ctx := t.Context()
		dbClient := db.Client()

		userOb := createUser(t, dbClient, ctx)
		passkeyOb := createPasskey(t, dbClient, ctx, userOb, webauthn.Credential{ID: common.CryptoRandomBytes(16)})
		now := time.Now()
		_, stdErr := dbClient.Session.Create().
			SetCreatedAt(now).
			SetUpdatedAt(now).
			SetHashedToken(common.CryptoRandomBytes(32)).
			SetExpiresAt(now.Add(time.Hour)).
			SetUserAgent(strings.Repeat("a", 513)). // exceeds max of 512
			SetIP("1.2.3.4").
			SetUser(userOb).
			SetPasskey(passkeyOb).
			Save(ctx)

		require.ErrorContains(t, stdErr, "EncryptedValidator: value is greater than the allowed length of 512")
	})
}
