package auth_test

import (
	"crypto/sha256"
	"testing"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/auth"
	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/common/testcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestCreateSession(t *testing.T) {
	t.Parallel()

	db := testcommon.CreateDB(t)
	dbClient := db.Client()
	userOb := createUser(t, dbClient)
	passkeyOb := createDummyPasskey(t, dbClient, userOb.ID, "login-passkey", false, false)

	tx := testcommon.StartWriteTx(t, db)
	sessionOb, sessionToken, wrappedErr := auth.CreateSession(
		false,
		userOb.ID,
		passkeyOb.ID,
		"test-agent",
		"127.0.0.1",
		tx,
		time.Hour,
		t.Context(),
	)
	require.NoError(t, wrappedErr)
	require.NoError(t, tx.Commit())
	require.NotNil(t, sessionOb)
	require.Len(t, sessionToken, auth.SessionTokenLength)

	storedSessionOb := dbClient.Session.GetX(t.Context(), sessionOb.ID)
	hashedToken := sha256.Sum256(sessionToken)
	require.Equal(t, hashedToken[:], storedSessionOb.HashedToken)
	require.NotEqual(t, sessionToken, storedSessionOb.HashedToken) // The raw token shouldn't be stored

	require.Equal(t, userOb.ID, storedSessionOb.UserID)
	require.Equal(t, passkeyOb.ID, storedSessionOb.PasskeyID)
	require.False(t, storedSessionOb.IsSudo)
	require.Equal(t, "test-agent", storedSessionOb.UserAgent)
	require.Equal(t, "127.0.0.1", storedSessionOb.IP)
	require.WithinDuration(t, time.Now().Add(time.Hour), storedSessionOb.ExpiresAt, 2*time.Second)
}

func TestCreateSession_Sudo(t *testing.T) {
	t.Parallel()

	db := testcommon.CreateDB(t)
	dbClient := db.Client()
	userOb := createUser(t, dbClient)
	passkeyOb := createDummyPasskey(t, dbClient, userOb.ID, "login-passkey", false, false)

	tx := testcommon.StartWriteTx(t, db)
	sessionOb, _, wrappedErr := auth.CreateSession(
		true,
		userOb.ID,
		passkeyOb.ID,
		"test-agent",
		"127.0.0.1",
		tx,
		time.Hour,
		t.Context(),
	)
	require.NoError(t, wrappedErr)
	require.NoError(t, tx.Commit())

	storedSessionOb := dbClient.Session.GetX(t.Context(), sessionOb.ID)
	require.True(t, storedSessionOb.IsSudo)

	// The behaviour of immediately created sudo sessions is currently a bit undefined.
	// In the future, auth.CreateSession will probably be updated to take an ElevationPasskeyID arg
	require.Nil(t, storedSessionOb.ElevationPasskeyID)
}

func TestCreateSession_UnknownUser_ReturnsDBConstraintError(t *testing.T) {
	t.Parallel()

	db := testcommon.CreateDB(t)
	tx := testcommon.StartWriteTx(t, db)

	_, _, wrappedErr := auth.CreateSession(
		false,
		uuid.New(),
		uuid.New(),
		"test-agent",
		"127.0.0.1",
		tx,
		time.Hour,
		t.Context(),
	)
	require.True(t, ent.IsConstraintError(wrappedErr))
	innerErr := wrappedErr.Unwrap()
	require.Equal(t, "ent: constraint failed: constraint failed: FOREIGN KEY constraint failed (787)", innerErr.Error())
}

func TestValidateSession(t *testing.T) {
	t.Parallel()

	db := testcommon.CreateDB(t)
	dbClient := db.Client()
	userOb := createUser(t, dbClient)
	passkeyOb := createDummyPasskey(t, dbClient, userOb.ID, "login-passkey", false, false)

	tx := testcommon.StartWriteTx(t, db)
	_, sessionToken, wrappedErr := auth.CreateSession(
		false,
		userOb.ID,
		passkeyOb.ID,
		"test-agent",
		"127.0.0.1",
		tx,
		time.Hour,
		t.Context(),
	)
	require.NoError(t, wrappedErr)
	require.NoError(t, tx.Commit())

	tx2 := testcommon.StartWriteTx(t, db)
	sessionOb, wrappedErr := auth.ValidateSession(sessionToken, tx2, t.Context())
	require.NoError(t, wrappedErr)
	require.NoError(t, tx2.Commit())

	require.NotNil(t, sessionOb)
	require.Equal(t, userOb.ID, sessionOb.UserID)
	require.Equal(t, passkeyOb.ID, sessionOb.PasskeyID)
	// The user edge must be loaded for middleware that needs it
	require.NotNil(t, sessionOb.Edges.User)
	require.Equal(t, userOb.ID, sessionOb.Edges.User.ID)
}

func TestValidateSession_RejectsExpiredSession(t *testing.T) {
	t.Parallel()

	db := testcommon.CreateDB(t)
	dbClient := db.Client()
	userOb := createUser(t, dbClient)
	passkeyOb := createDummyPasskey(t, dbClient, userOb.ID, "login-passkey", false, false)

	tx := testcommon.StartWriteTx(t, db)
	sessionOb, sessionToken, wrappedErr := auth.CreateSession(
		false,
		userOb.ID,
		passkeyOb.ID,
		"test-agent",
		"127.0.0.1",
		tx,
		time.Hour,
		t.Context(),
	)
	require.NoError(t, wrappedErr)
	require.NoError(t, tx.Commit())

	dbClient.Session.UpdateOneID(sessionOb.ID).
		SetExpiresAt(time.Now().Add(-time.Minute)).
		ExecX(t.Context())

	tx2 := testcommon.StartWriteTx(t, db)
	_, wrappedErr = auth.ValidateSession(sessionToken, tx2, t.Context())
	require.ErrorIs(t, wrappedErr, auth.ErrInvalidSession)
}

func TestValidateSession_RejectsUnknownToken(t *testing.T) {
	t.Parallel()

	db := testcommon.CreateDB(t)
	tx := testcommon.StartWriteTx(t, db)

	_, wrappedErr := auth.ValidateSession(common.CryptoRandomBytes(32), tx, t.Context())
	require.ErrorIs(t, wrappedErr, auth.ErrInvalidSession)
}
