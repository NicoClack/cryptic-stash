package schema_test

import (
	"strings"
	"testing"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/common/testcommon"
	"github.com/stretchr/testify/require"
)

func TestDownloadSession_EncryptedFields(t *testing.T) {
	t.Parallel()

	t.Run("userAgent + ip can be read back", func(t *testing.T) {
		t.Parallel()
		db := testcommon.CreateDB(t)
		t.Cleanup(db.Shutdown)
		ctx := t.Context()
		dbClient := db.Client()

		stashOb := createUserAndStash(t, dbClient, ctx)
		userAgent := "TestAgent/1.0"
		ip := "192.0.2.1"
		now := time.Now()
		downloadSessionOb := dbClient.DownloadSession.Create().
			SetHashedAuthCode(common.CryptoRandomBytes(32)).
			SetValidFrom(now).
			SetValidUntil(now.Add(24 * time.Hour)).
			SetUserAgent(userAgent).
			SetIP(ip).
			SetCreatedAt(now).
			SetUpdatedAt(now).
			SetStash(stashOb).
			SaveX(ctx)

		downloadSessionOb = dbClient.DownloadSession.GetX(ctx, downloadSessionOb.ID)
		require.Equal(t, userAgent, downloadSessionOb.UserAgent)
		require.Equal(t, ip, downloadSessionOb.IP)
	})

	t.Run("string fields (userAgent + ip) are encrypted directly, not JSON-encoded", func(t *testing.T) {
		t.Parallel()
		db := testcommon.CreateDB(t)
		t.Cleanup(db.Shutdown)
		ctx := t.Context()
		dbClient := db.Client()

		stashOb := createUserAndStash(t, dbClient, ctx)
		now := time.Now()
		userAgent := "direct-bytes-check"
		ip := "5.6.7.8"
		downloadSessionOb := dbClient.DownloadSession.Create().
			SetHashedAuthCode(common.CryptoRandomBytes(32)).
			SetValidFrom(now).
			SetValidUntil(now.Add(24 * time.Hour)).
			SetUserAgent(userAgent).
			SetIP(ip).
			SetCreatedAt(now).
			SetUpdatedAt(now).
			SetStash(stashOb).
			SaveX(ctx)

		idStr := downloadSessionOb.ID.String()
		assertEncryptedInDB(t, db,
			"SELECT user_agent FROM download_sessions WHERE id = ?", []any{idStr},
			[]byte(userAgent), "security_pii_logging_1",
			// ^ Not wrapped in JSON quotes
		)
		assertEncryptedInDB(t, db,
			"SELECT ip FROM download_sessions WHERE id = ?", []any{idStr},
			[]byte(ip), "security_pii_logging_1",
			// ^ Not wrapped in JSON quotes
		)
	})

	t.Run("min length constraint", func(t *testing.T) {
		t.Parallel()
		db := testcommon.CreateDB(t)
		t.Cleanup(db.Shutdown)
		ctx := t.Context()
		dbClient := db.Client()

		stashOb := createUserAndStash(t, dbClient, ctx)
		now := time.Now()
		_, stdErr := dbClient.DownloadSession.Create().
			SetHashedAuthCode(common.CryptoRandomBytes(32)).
			SetValidFrom(now).
			SetValidUntil(now.Add(24 * time.Hour)).
			SetUserAgent("").
			SetIP("1.2.3.4").
			SetCreatedAt(now).
			SetUpdatedAt(now).
			SetStash(stashOb).
			Save(ctx)

		require.ErrorContains(t, stdErr, "EncryptedValidator: value is less than the required length of 1")
	})

	t.Run("max length constraint on ip", func(t *testing.T) {
		t.Parallel()
		db := testcommon.CreateDB(t)
		t.Cleanup(db.Shutdown)
		ctx := t.Context()
		dbClient := db.Client()

		stashOb := createUserAndStash(t, dbClient, ctx)
		now := time.Now()
		_, stdErr := dbClient.DownloadSession.Create().
			SetHashedAuthCode(common.CryptoRandomBytes(32)).
			SetValidFrom(now).
			SetValidUntil(now.Add(24 * time.Hour)).
			SetUserAgent("Agent/1.0").
			SetIP(strings.Repeat("1", 46)). // exceeds max of 45
			SetCreatedAt(now).
			SetUpdatedAt(now).
			SetStash(stashOb).
			Save(ctx)

		require.ErrorContains(t, stdErr, "EncryptedValidator: value is greater than the allowed length of 45")
	})
}
