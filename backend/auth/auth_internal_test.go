package auth

import (
	"testing"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/common/testcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func createUser(t *testing.T, dbClient *ent.Client, username string) *ent.User {
	t.Helper()

	now := time.Now()
	return dbClient.User.Create().
		SetUsername(username).
		SetCreatedAt(now).
		SetUpdatedAt(now).
		SaveX(t.Context())
}

func createDummyPasskey(
	t *testing.T,
	dbClient *ent.Client,
	userID uuid.UUID,
	name string,
	allowSudo bool,
	isSecondGroup bool,
) *ent.Passkey {
	t.Helper()

	credentialID := common.CryptoRandomBytes(16)
	return dbClient.Passkey.Create().
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		SetUserID(userID).
		SetName(name).
		SetAllowSudo(allowSudo).
		SetIsSecondGroup(isSecondGroup).
		SetCredentialID(credentialID).
		SetCredential(webauthn.Credential{
			ID:        credentialID,
			PublicKey: common.CryptoRandomBytes(32),
			Flags: webauthn.CredentialFlags{
				UserPresent:  true,
				UserVerified: true,
			},
			Authenticator: webauthn.Authenticator{
				AAGUID:    common.CryptoRandomBytes(16),
				SignCount: 1,
			},
		}).
		SaveX(t.Context())
}

func createDummySession(
	t *testing.T,
	dbClient *ent.Client,
	userID uuid.UUID,
	passkeyID uuid.UUID,
	elevationPasskeyID *uuid.UUID, // Optional
	isSudo bool,
) *ent.Session {
	t.Helper()

	sessionCreate := dbClient.Session.Create().
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		SetUserID(userID).
		SetPasskeyID(passkeyID).
		SetIsSudo(isSudo).
		SetHashedToken(common.CryptoRandomBytes(32)).
		SetExpiresAt(time.Now().Add(time.Hour)).
		SetUserAgent("test-agent").
		SetIP("127.0.0.1")
	if elevationPasskeyID != nil {
		sessionCreate.SetElevationPasskeyID(*elevationPasskeyID)
	}
	return sessionCreate.SaveX(t.Context())
}

func TestDemoteInvalidSudoSessions(t *testing.T) {
	t.Parallel()

	db := testcommon.CreateDB(t)
	dbClient := db.Client()
	userOb := createUser(t, dbClient, "user1")
	sudoFirstGroupPasskey := createDummyPasskey(t, dbClient, userOb.ID, "sudo-first-group-passkey", true, false)
	nonSudoFirstGroupPasskey := createDummyPasskey(
		t,
		dbClient,
		userOb.ID,
		"non-sudo-first-group-passkey",
		false,
		false,
	)
	sudoSecondGroupPasskey := createDummyPasskey(
		t,
		dbClient,
		userOb.ID,
		"sudo-second-group-passkey",
		true,
		true,
	)

	// To keep
	crossGroupSudoSession := createDummySession(
		t,
		dbClient,
		userOb.ID,
		sudoFirstGroupPasskey.ID,
		new(sudoSecondGroupPasskey.ID),
		true,
	)
	nonSudoCrossGroupSession := createDummySession( // Invalid state
		t,
		dbClient,
		userOb.ID,
		sudoFirstGroupPasskey.ID,
		new(sudoSecondGroupPasskey.ID),
		false,
	)
	// To demote
	sameGroupSudoSession := createDummySession(
		t,
		dbClient,
		userOb.ID,
		sudoFirstGroupPasskey.ID,
		new(nonSudoFirstGroupPasskey.ID),
		true,
	)
	noElevationSudoSession := createDummySession(t, dbClient, userOb.ID, sudoFirstGroupPasskey.ID, nil, true)

	tx := testcommon.StartWriteTx(t, db)
	count, wrappedErr := demoteInvalidSudoSessions(userOb.ID, tx, t.Context())
	require.NoError(t, wrappedErr)
	require.Equal(t, 2, count) // sameGroupSudoSession + noElevationSudoSession
	require.NoError(t, tx.Commit())

	assertSudoStatus := func(t *testing.T, sessionID uuid.UUID, isSudo bool) {
		t.Helper()
		sessionOb := dbClient.Session.GetX(t.Context(), sessionID)
		require.Equal(t, isSudo, sessionOb.IsSudo)
		if isSudo {
			require.NotNil(t, sessionOb.ElevationPasskeyID)
		} else {
			require.Nil(t, sessionOb.ElevationPasskeyID)
		}
	}
	assertSudoStatus(t, crossGroupSudoSession.ID, true)
	assertSudoStatus(t, sameGroupSudoSession.ID, false)
	assertSudoStatus(t, noElevationSudoSession.ID, false)
	// Non-sudo sessions are untouched (still not sudo, elevation passkey retained)
	nonSudoCrossGroupSession = dbClient.Session.GetX(t.Context(), nonSudoCrossGroupSession.ID)
	require.False(t, nonSudoCrossGroupSession.IsSudo)
	require.NotNil(t, nonSudoCrossGroupSession.ElevationPasskeyID)
}

func TestDemoteInvalidSudoSessions_NoInvalidSessions_ReturnsZero(t *testing.T) {
	t.Parallel()

	db := testcommon.CreateDB(t)
	dbClient := db.Client()
	userOb := createUser(t, dbClient, "user1")
	group1Passkey := createDummyPasskey(t, dbClient, userOb.ID, "group-1-passkey", true, false)
	group2Passkey := createDummyPasskey(t, dbClient, userOb.ID, "group-2-passkey", true, true)
	createDummySession(t, dbClient, userOb.ID, group1Passkey.ID, new(group2Passkey.ID), true)

	tx := testcommon.StartWriteTx(t, db)
	count, wrappedErr := demoteInvalidSudoSessions(userOb.ID, tx, t.Context())
	require.NoError(t, wrappedErr)
	require.Equal(t, 0, count)
	require.NoError(t, tx.Commit())

	sessionOb := dbClient.Session.Query().
		Where().
		OnlyX(t.Context())
	require.True(t, sessionOb.IsSudo)
	require.Equal(t, userOb.ID, sessionOb.UserID)
	require.Equal(t, group1Passkey.ID, sessionOb.PasskeyID)
	require.NotNil(t, sessionOb.ElevationPasskeyID)
	require.Equal(t, group2Passkey.ID, *sessionOb.ElevationPasskeyID)
}
