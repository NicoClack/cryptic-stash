package schema_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/common/testcommon"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/stretchr/testify/require"
)

func TestInvite_EncryptedFields(t *testing.T) {
	t.Parallel()

	t.Run("non-nil userAgent + ip + webAuthnSession can be read back", func(t *testing.T) {
		t.Parallel()
		db := testcommon.CreateDB(t)
		ctx := t.Context()
		dbClient := db.Client()

		now := time.Now()
		userAgent := "InviteAgent/1.0"
		ip := "203.0.113.5"
		webAuthnSession := &webauthn.SessionData{
			Challenge: "test-challenge-string",
			UserID:    []byte("test-user-id"),
		}
		inviteOb := dbClient.Invite.Create().
			SetCreatedAt(now).
			SetUpdatedAt(now).
			SetEmail("test@example.com").
			SetHashedCode(common.CryptoRandomBytes(32)).
			SetExpiresAt(now.Add(24 * time.Hour)).
			SetWebAuthnSession(webAuthnSession).
			SetUserAgent(&userAgent).
			SetIP(&ip).
			SaveX(ctx)

		inviteOb = dbClient.Invite.GetX(ctx, inviteOb.ID)
		require.NotNil(t, inviteOb.WebAuthnSession)
		require.Equal(t, webAuthnSession.Challenge, inviteOb.WebAuthnSession.Challenge)
		require.Equal(t, webAuthnSession.UserID, inviteOb.WebAuthnSession.UserID)
		require.NotNil(t, inviteOb.UserAgent)
		require.Equal(t, userAgent, *inviteOb.UserAgent)
		require.NotNil(t, inviteOb.IP)
		require.Equal(t, ip, *inviteOb.IP)
	})

	t.Run("nil optional fields (userAgent + ip + webAuthnSession) are stored as NULL", func(t *testing.T) {
		t.Parallel()
		db := testcommon.CreateDB(t)
		ctx := t.Context()
		dbClient := db.Client()

		now := time.Now()
		inviteOb := dbClient.Invite.Create().
			SetCreatedAt(now).
			SetUpdatedAt(now).
			SetEmail("null-test@example.com").
			SetHashedCode(common.CryptoRandomBytes(32)).
			SetExpiresAt(now.Add(24 * time.Hour)).
			SaveX(ctx)

		idStr := inviteOb.ID.String()
		assertNullIsUnencrypted(t, db, "SELECT web_authn_session FROM invites WHERE id = ?", idStr)
		assertNullIsUnencrypted(t, db, "SELECT user_agent FROM invites WHERE id = ?", idStr)
		assertNullIsUnencrypted(t, db, "SELECT ip FROM invites WHERE id = ?", idStr)
	})

	t.Run("optional string fields (userAgent + ip) are encrypted directly, not JSON-encoded", func(t *testing.T) {
		t.Parallel()
		db := testcommon.CreateDB(t)
		ctx := t.Context()

		now := time.Now()
		userAgent := "InviteEncryptedAgent/1.0"
		ip := "203.0.113.10"
		inviteOb := db.Client().Invite.Create().
			SetCreatedAt(now).
			SetUpdatedAt(now).
			SetEmail("encrypt-test@example.com").
			SetHashedCode(common.CryptoRandomBytes(32)).
			SetExpiresAt(now.Add(24 * time.Hour)).
			SetUserAgent(&userAgent).
			SetIP(&ip).
			SaveX(ctx)

		idStr := inviteOb.ID.String()
		assertEncryptedInDB(t, db,
			"SELECT user_agent FROM invites WHERE id = ?", []any{idStr},
			[]byte(userAgent), "security_pii_logging_1",
			// ^ Not wrapped in JSON quotes
		)
		assertEncryptedInDB(t, db,
			"SELECT ip FROM invites WHERE id = ?", []any{idStr},
			[]byte(ip), "security_pii_logging_1",
			// ^ Not wrapped in JSON quotes
		)
	})

	t.Run("webAuthnSession is JSON-encoded before encryption", func(t *testing.T) {
		t.Parallel()
		db := testcommon.CreateDB(t)
		ctx := t.Context()

		now := time.Now()
		webAuthnSession := &webauthn.SessionData{
			Challenge: "my-challenge",
			UserID:    []byte("uid123"),
		}
		inviteOb := db.Client().Invite.Create().
			SetEmail("json-test@example.com").
			SetHashedCode(common.CryptoRandomBytes(32)).
			SetExpiresAt(now.Add(24 * time.Hour)).
			SetWebAuthnSession(webAuthnSession).
			SetCreatedAt(now).
			SetUpdatedAt(now).
			SaveX(ctx)

		expectedJSON, stdErr := json.Marshal(webAuthnSession)
		require.NoError(t, stdErr)
		assertEncryptedInDB(t, db,
			"SELECT web_authn_session FROM invites WHERE id = ?", []any{inviteOb.ID.String()},
			expectedJSON, "auth_1",
		)
	})

	t.Run("min length constraint on optional string (userAgent + ip)", func(t *testing.T) {
		t.Parallel()
		db := testcommon.CreateDB(t)
		ctx := t.Context()

		now := time.Now()
		_, stdErr := db.Client().Invite.Create().
			SetCreatedAt(now).
			SetUpdatedAt(now).
			SetEmail("min-test@example.com").
			SetHashedCode(common.CryptoRandomBytes(32)).
			SetExpiresAt(now.Add(24 * time.Hour)).
			SetUserAgent(new("")).
			SetIP(new("")).
			Save(ctx)

		require.ErrorContains(t, stdErr, "EncryptedValidator: value is less than the required length of 1")
	})

	t.Run("max length constraint on optional string", func(t *testing.T) {
		t.Parallel()
		db := testcommon.CreateDB(t)
		ctx := t.Context()

		now := time.Now()
		_, stdErr := db.Client().Invite.Create().
			SetCreatedAt(now).
			SetUpdatedAt(now).
			SetEmail("max-test@example.com").
			SetHashedCode(common.CryptoRandomBytes(32)).
			SetExpiresAt(now.Add(24 * time.Hour)).
			SetUserAgent(new(strings.Repeat("x", 513))).
			SetIP(new(strings.Repeat("y", 46))).
			Save(ctx)

		require.ErrorContains(t, stdErr, "EncryptedValidator: value is greater than the allowed length of 512")
	})
}
