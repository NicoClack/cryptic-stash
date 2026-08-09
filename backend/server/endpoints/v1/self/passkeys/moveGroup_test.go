package passkeys_test

import (
	"net/http"
	"testing"

	"github.com/NicoClack/cryptic-stash/backend/common/testcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent/passkey"
	"github.com/NicoClack/cryptic-stash/backend/server/endpoints/v1/self/passkeys"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/NicoClack/cryptic-stash/backend/testhelpers"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestMoveGroup_CreateSecondGroup(t *testing.T) {
	t.Parallel()

	panic("not implemented")
}

func TestMoveGroup_ExistingSecondGroup(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)
	dbClient := app.Database.Client()
	userOb := testcommon.NewDummyUser(1, dbClient, t.Context(), app.Clock)
	firstGroupPasskey := createPasskey(t, "first-group-key", true, false, userOb.ID, dbClient)
	passkeyToMove := createPasskey(t, "moving-key", true, false, userOb.ID, dbClient)
	// Create the second group initially
	_ = createPasskey(t, "existing-second", true, true, userOb.ID, dbClient)
	sessionToken := createSession(t, true, firstGroupPasskey.UserID, firstGroupPasskey.ID, app)

	respRecorder := testcommon.Post(
		t, app.Server,
		"/api/v1/self/passkeys/"+passkeyToMove.ID.String()+"/move-group/",
		passkeys.MoveGroupPayload{
			IsSecondGroup: true,
		},
		testcommon.WithBearerToken(sessionToken),
	)

	testcommon.AssertJSONResponse(
		t, respRecorder,
		http.StatusOK,
		gin.H{
			"errors": []servercommon.ErrorDetail{},
		},
	)

	movedPasskey := dbClient.Passkey.Query().
		Where(passkey.ID(passkeyToMove.ID)).
		OnlyX(t.Context())
	require.True(t, movedPasskey.IsSecondGroup)
}

func TestMoveGroup_SamePasskeyUsedTwice_SendsConflictError(t *testing.T) {
	t.Parallel()

	// TODO: create session in the same way as TestList
	panic("not implemented")
}

func TestMoveGroup_NoElevationPasskey_SendsConflictError(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)
	dbClient := app.Database.Client()
	userOb := testcommon.NewDummyUser(1, dbClient, t.Context(), app.Clock)
	loginPk := createPasskey(t, "login-key", true, false, userOb.ID, dbClient)
	sessionToken := createSession(t, true, loginPk.UserID, loginPk.ID, app)
	// ^ sessionOb.ElevationPasskeyID is nil

	respRecorder := testcommon.Post(
		t, app.Server,
		"/api/v1/self/passkeys/"+loginPk.ID.String()+"/move-group/",
		passkeys.MoveGroupPayload{
			IsSecondGroup: true,
		},
		testcommon.WithBearerToken(sessionToken),
	)

	testcommon.AssertJSONResponse(
		t, respRecorder,
		http.StatusConflict,
		gin.H{
			"errors": []servercommon.ErrorDetail{
				{
					Message: "can't move this passkey, as doing so may lock you out of sudo mode. to enable two group auth, " +
						"use two different passkeys in your session and move one of them to the second group. " +
						"to move back, disable two group auth entirely",
					Code: "GROUP_MOVE_CONSTRAINT",
				},
			},
		},
	)
}
