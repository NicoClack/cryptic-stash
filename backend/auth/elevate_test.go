package auth_test

import (
	"testing"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/auth"
	"github.com/NicoClack/cryptic-stash/backend/common/testcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/ent/session"
	"github.com/NicoClack/cryptic-stash/backend/ent/user"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func loadUserWithPasskeys(t *testing.T, dbClient *ent.Client, userID uuid.UUID) *ent.User {
	t.Helper()

	return dbClient.User.Query().
		Where(user.ID(userID)).
		WithPasskeys().
		OnlyX(t.Context())
}

func loadSessionWithPasskey(t *testing.T, dbClient *ent.Client, sessionID uuid.UUID) *ent.Session {
	t.Helper()

	return dbClient.Session.Query().
		Where(session.ID(sessionID)).
		WithPasskey().
		OnlyX(t.Context())
}

func getPasskeyIDs(t *testing.T, passkeys []*ent.Passkey) []uuid.UUID {
	t.Helper()

	ids := make([]uuid.UUID, 0, len(passkeys))
	for _, passkeyOb := range passkeys {
		ids = append(ids, passkeyOb.ID)
	}
	return ids
}

func TestGetEligiblePasskeysForSudo_SingleGroup_SessionPasskeyIsSudo(t *testing.T) {
	t.Parallel()

	db := testcommon.CreateDB(t)
	dbClient := db.Client()
	userOb := createUser(t, dbClient)
	sudoPasskey := createDummyPasskey(t, dbClient, userOb.ID, "sudo-passkey", true, false)
	nonSudoPasskey := createDummyPasskey(t, dbClient, userOb.ID, "non-sudo-passkey", false, false)

	sessionOb := createDummySession(t, dbClient, userOb.ID, sudoPasskey.ID)
	loadedSession := loadSessionWithPasskey(t, dbClient, sessionOb.ID)
	loadedUser := loadUserWithPasskeys(t, dbClient, userOb.ID)

	eligible, wrappedErr := auth.GetEligiblePasskeysForSudo(loadedSession, loadedUser)
	require.NoError(t, wrappedErr)
	require.ElementsMatch(
		t,
		[]uuid.UUID{
			nonSudoPasskey.ID, // Non-sudo are allowed because the session was created with a sudo passkey
			sudoPasskey.ID,
		},
		getPasskeyIDs(t, eligible),
	)
}

func TestGetEligiblePasskeysForSudo_SingleGroup_SessionPasskeyIsNonSudo(t *testing.T) {
	t.Parallel()

	db := testcommon.CreateDB(t)
	dbClient := db.Client()
	userOb := createUser(t, dbClient)
	sudoPasskey := createDummyPasskey(t, dbClient, userOb.ID, "sudo-passkey", true, false)
	nonSudoPasskey := createDummyPasskey(t, dbClient, userOb.ID, "non-sudo-passkey", false, false)

	sessionOb := createDummySession(t, dbClient, userOb.ID, nonSudoPasskey.ID)
	loadedSession := loadSessionWithPasskey(t, dbClient, sessionOb.ID)
	loadedUser := loadUserWithPasskeys(t, dbClient, userOb.ID)

	eligible, wrappedErr := auth.GetEligiblePasskeysForSudo(loadedSession, loadedUser)
	require.NoError(t, wrappedErr)
	passkeyIDs := getPasskeyIDs(t, eligible)
	require.NotContains(
		t,
		passkeyIDs,
		nonSudoPasskey.ID,
		// ^ Because the session was created by a non-sudo passkey, the elevation passkey must be sudo
	)
	require.ElementsMatch(
		t,
		[]uuid.UUID{
			sudoPasskey.ID,
		},
		passkeyIDs,
	)
}

func TestGetEligiblePasskeysForSudo_DualGroup_Group1SessionPasskeyIsSudo(t *testing.T) {
	t.Parallel()

	db := testcommon.CreateDB(t)
	dbClient := db.Client()
	userOb := createUser(t, dbClient)
	group1SudoPasskey := createDummyPasskey(t, dbClient, userOb.ID, "group-1-sudo-passkey", true, false)
	group2SudoPasskey := createDummyPasskey(t, dbClient, userOb.ID, "group-2-sudo-passkey", true, true)
	group2NonSudoPasskey := createDummyPasskey(t, dbClient, userOb.ID, "group-2-non-sudo", false, true)

	sessionOb := createDummySession(t, dbClient, userOb.ID, group1SudoPasskey.ID)
	loadedSession := loadSessionWithPasskey(t, dbClient, sessionOb.ID)
	loadedUser := loadUserWithPasskeys(t, dbClient, userOb.ID)

	eligible, wrappedErr := auth.GetEligiblePasskeysForSudo(loadedSession, loadedUser)
	require.NoError(t, wrappedErr)
	passkeyIDs := getPasskeyIDs(t, eligible)
	require.NotContains(t, passkeyIDs, group1SudoPasskey.ID)
	require.ElementsMatch(
		t,
		[]uuid.UUID{
			// Must be from the opposite group
			group2SudoPasskey.ID,
			group2NonSudoPasskey.ID, // Non-sudo passkeys allowed because the session was created with a sudo passkey
		},
		passkeyIDs,
	)
}

func TestGetEligiblePasskeysForSudo_DualGroup_Group1SessionPasskeyIsNonSudo(t *testing.T) {
	t.Parallel()

	db := testcommon.CreateDB(t)
	dbClient := db.Client()
	userOb := createUser(t, dbClient)
	group1SudoPasskey := createDummyPasskey(t, dbClient, userOb.ID, "group-1-sudo", true, false)
	group1NonSudoPasskey := createDummyPasskey(t, dbClient, userOb.ID, "group-1-non-sudo", false, false)
	group2SudoPasskey := createDummyPasskey(t, dbClient, userOb.ID, "group-2-sudo", true, true)
	group2NonSudoPasskey := createDummyPasskey(t, dbClient, userOb.ID, "group-2-non-sudo", false, true)

	sessionOb := createDummySession(t, dbClient, userOb.ID, group1NonSudoPasskey.ID)
	loadedSession := loadSessionWithPasskey(t, dbClient, sessionOb.ID)
	loadedUser := loadUserWithPasskeys(t, dbClient, userOb.ID)

	eligible, wrappedErr := auth.GetEligiblePasskeysForSudo(loadedSession, loadedUser)
	require.NoError(t, wrappedErr)
	passkeyIDs := getPasskeyIDs(t, eligible)
	require.NotContains(t, passkeyIDs, group1SudoPasskey.ID) // Must be from the opposite group
	require.NotContains(
		t,
		passkeyIDs,
		group2NonSudoPasskey.ID,
		// ^ Because the session was created by a non-sudo passkey, the elevation passkey must be sudo
	)
	require.NotContains(t, passkeyIDs, group1NonSudoPasskey.ID) // Not eligible for both reasons above
	require.ElementsMatch(
		t,
		[]uuid.UUID{
			group2SudoPasskey.ID,
		},
		passkeyIDs,
	)
}

func TestGetEligiblePasskeysForSudo_DualGroup_Group2SessionPasskeyIsNonSudo(t *testing.T) {
	t.Parallel()

	db := testcommon.CreateDB(t)
	dbClient := db.Client()
	userOb := createUser(t, dbClient)
	group1SudoPasskey := createDummyPasskey(t, dbClient, userOb.ID, "group-1-sudo", true, false)
	group1NonSudoPasskey := createDummyPasskey(t, dbClient, userOb.ID, "group-1-non-sudo", false, false)
	group2SudoPasskey := createDummyPasskey(t, dbClient, userOb.ID, "group-2-sudo", true, true)
	group2NonSudoPasskey := createDummyPasskey(t, dbClient, userOb.ID, "group-2-non-sudo", false, true)

	sessionOb := createDummySession(t, dbClient, userOb.ID, group2NonSudoPasskey.ID)
	loadedSession := loadSessionWithPasskey(t, dbClient, sessionOb.ID)
	loadedUser := loadUserWithPasskeys(t, dbClient, userOb.ID)

	eligible, wrappedErr := auth.GetEligiblePasskeysForSudo(loadedSession, loadedUser)
	require.NoError(t, wrappedErr)
	passkeyIDs := getPasskeyIDs(t, eligible)
	require.NotContains(t, passkeyIDs, group2SudoPasskey.ID) // Must be from the opposite group
	require.NotContains(
		t,
		passkeyIDs,
		group1NonSudoPasskey.ID,
		// ^ Because the session was created by a non-sudo passkey, the elevation passkey must be sudo
	)
	require.NotContains(t, passkeyIDs, group2NonSudoPasskey.ID) // Not eligible for both reasons above
	require.ElementsMatch(
		t,
		[]uuid.UUID{
			group1SudoPasskey.ID,
		},
		passkeyIDs,
	)
}

func TestElevateSession(t *testing.T) {
	t.Parallel()

	db := testcommon.CreateDB(t)
	dbClient := db.Client()
	userOb := createUser(t, dbClient)
	loginPasskey := createDummyPasskey(t, dbClient, userOb.ID, "login", false, false)
	elevationPasskey := createDummyPasskey(t, dbClient, userOb.ID, "elevate", true, false)
	sessionOb := createDummySession(t, dbClient, userOb.ID, loginPasskey.ID)

	expiresAt := time.Now().Add(time.Hour).UTC()
	tx := testcommon.StartWriteTx(t, db)
	wrappedErr := auth.ElevateSession(sessionOb.ID, elevationPasskey.ID, expiresAt, tx, t.Context())
	require.NoError(t, wrappedErr)
	require.NoError(t, tx.Commit())

	updatedSession := dbClient.Session.GetX(t.Context(), sessionOb.ID)
	require.True(t, updatedSession.IsSudo)
	require.NotNil(t, updatedSession.ElevationPasskeyID)
	require.Equal(t, elevationPasskey.ID, *updatedSession.ElevationPasskeyID)
	require.Equal(t, expiresAt, updatedSession.ExpiresAt)
}
