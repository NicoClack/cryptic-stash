package auth_test

import (
	"testing"

	"github.com/NicoClack/cryptic-stash/backend/auth"
	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/common/testcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent/passkey"
	"github.com/google/uuid"
	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/require"
)

func getActor(userID uuid.UUID) *common.Actor {
	return &common.Actor{
		UserID:    userID,
		IP:        "127.0.0.1",
		UserAgent: "test-agent",
	}
}

func TestRenamePasskey_CannotModifyOtherUserPasskey(t *testing.T) {
	t.Parallel()

	clock := clockwork.NewRealClock()
	db := testcommon.CreateDB(t)
	dbClient := db.Client()
	ownerUserOb := testcommon.NewDummyUser(1, dbClient, t.Context(), clock)
	otherUserOb := testcommon.NewDummyUser(2, dbClient, t.Context(), clock)
	ownerPasskeyOb := createDummyPasskey(t, dbClient, ownerUserOb.ID, "original-name", false, false)
	createDummyPasskey(t, dbClient, otherUserOb.ID, "other-passkey", false, false)

	tx := testcommon.StartWriteTx(t, db)
	wrappedErr := auth.RenamePasskey(
		ownerPasskeyOb.ID,
		"hacked-name",
		getActor(otherUserOb.ID),
		tx,
		t.Context(),
	)
	require.ErrorIs(t, wrappedErr, auth.ErrPasskeyNotFound)

	updatedPasskey := tx.Passkey.GetX(t.Context(), ownerPasskeyOb.ID)
	require.Equal(t, "original-name", updatedPasskey.Name)
}

func TestSetPasskeyAllowSudo_CannotModifyOtherUserPasskey(t *testing.T) {
	t.Parallel()

	db := testcommon.CreateDB(t)
	dbClient := db.Client()
	clock := clockwork.NewRealClock()
	ownerUserOb := testcommon.NewDummyUser(1, dbClient, t.Context(), clock)
	otherUserOb := testcommon.NewDummyUser(2, dbClient, t.Context(), clock)
	ownerPasskeyOb := createDummyPasskey(t, dbClient, ownerUserOb.ID, "owner-passkey", true, false)
	otherPasskeyOb := createDummyPasskey(t, dbClient, otherUserOb.ID, "other-passkey", true, false)

	tx := testcommon.StartWriteTx(t, db)
	wrappedErr := auth.SetPasskeyAllowSudo(
		ownerPasskeyOb.ID,
		otherPasskeyOb.ID,
		nil, // no elevation passkey, never reached because ownership fails first
		false,
		getActor(otherUserOb.ID),
		tx,
		t.Context(),
		testcommon.NewTestLogger(),
	)
	require.ErrorIs(t, wrappedErr, auth.ErrPasskeyNotFound)

	updatedPasskey := tx.Passkey.GetX(t.Context(), ownerPasskeyOb.ID)
	require.True(t, updatedPasskey.AllowSudo) // Unchanged
}

func TestMovePasskeyGroup_TargetUserMustMatchActor(t *testing.T) {
	t.Parallel()

	db := testcommon.CreateDB(t)
	dbClient := db.Client()
	clock := clockwork.NewRealClock()
	ownerUserOb := testcommon.NewDummyUser(1, dbClient, t.Context(), clock)
	otherUserOb := testcommon.NewDummyUser(2, dbClient, t.Context(), clock)
	ownerPasskeyOb := createDummyPasskey(t, dbClient, ownerUserOb.ID, "owner-passkey", true, false)
	otherPasskeyOb := createDummyPasskey(t, dbClient, otherUserOb.ID, "other-passkey", true, false)

	tx := testcommon.StartWriteTx(t, db)
	wrappedErr := auth.MovePasskeyGroup(
		ownerPasskeyOb.ID,
		ownerUserOb.ID, // The target, which is a different user to the actor
		otherPasskeyOb.ID,
		nil,
		true,
		getActor(otherUserOb.ID),
		tx,
		t.Context(),
		testcommon.NewTestLogger(),
	)
	require.ErrorIs(t, wrappedErr, auth.ErrUnauthorizedToModifyUser)

	updatedPasskey := tx.Passkey.GetX(t.Context(), ownerPasskeyOb.ID)
	require.False(t, updatedPasskey.IsSecondGroup) // Unchanged
}

func TestMovePasskeyGroup_CannotModifyOtherUserPasskey(t *testing.T) {
	t.Parallel()

	db := testcommon.CreateDB(t)
	dbClient := db.Client()
	clock := clockwork.NewRealClock()
	ownerUserOb := testcommon.NewDummyUser(1, dbClient, t.Context(), clock)
	otherUserOb := testcommon.NewDummyUser(2, dbClient, t.Context(), clock)
	ownerPasskeyOb := createDummyPasskey(t, dbClient, ownerUserOb.ID, "owner-passkey", true, false)
	otherPasskeyOb := createDummyPasskey(t, dbClient, otherUserOb.ID, "other-passkey", true, false)
	// An existing second group passkey for the actor so the group-move constraint doesn't apply
	createDummyPasskey(t, dbClient, otherUserOb.ID, "second-group-passkey", true, true)

	tx := testcommon.StartWriteTx(t, db)
	wrappedErr := auth.MovePasskeyGroup(
		ownerPasskeyOb.ID, // Not owned by the target user (otherUserOb)
		otherUserOb.ID,
		otherPasskeyOb.ID,
		nil,
		true,
		getActor(otherUserOb.ID),
		tx,
		t.Context(),
		testcommon.NewTestLogger(),
	)
	require.ErrorIs(t, wrappedErr, auth.ErrPasskeyNotFound)

	reloaded := tx.Passkey.GetX(t.Context(), ownerPasskeyOb.ID)
	require.False(t, reloaded.IsSecondGroup) // Unchanged
}

func TestDeletePasskey_CannotDeleteOtherUserPasskey(t *testing.T) {
	t.Parallel()

	db := testcommon.CreateDB(t)
	dbClient := db.Client()
	clock := clockwork.NewRealClock()
	ownerUserOb := testcommon.NewDummyUser(1, dbClient, t.Context(), clock)
	otherUserOb := testcommon.NewDummyUser(2, dbClient, t.Context(), clock)
	ownerPasskeyOb := createDummyPasskey(t, dbClient, ownerUserOb.ID, "owner-passkey", false, false)
	createDummyPasskey(t, dbClient, otherUserOb.ID, "other-passkey", false, false)

	tx := testcommon.StartWriteTx(t, db)
	wrappedErr := auth.DeletePasskey(
		ownerPasskeyOb.ID,
		uuid.New(), // Just needs to be a session ID not associated with the target passkey
		getActor(otherUserOb.ID),
		tx,
		t.Context(),
	)
	require.ErrorIs(t, wrappedErr, auth.ErrPasskeyNotFound)

	exists, stdErr := tx.Passkey.Query().
		Where(passkey.ID(ownerPasskeyOb.ID)).
		Exist(t.Context())
	require.NoError(t, stdErr)
	require.True(t, exists) // Unchanged
}
