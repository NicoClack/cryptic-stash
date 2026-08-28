package passkeys_test

import (
	"net/http"
	"testing"

	"github.com/NicoClack/cryptic-stash/backend/common/testcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent/passkey"
	"github.com/NicoClack/cryptic-stash/backend/ent/session"
	"github.com/NicoClack/cryptic-stash/backend/server/endpoints/v1/self/passkeys"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/NicoClack/cryptic-stash/backend/testhelpers"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestMoveGroup_CreateSecondGroup(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)
	dbClient := app.Database.Client()
	userOb := testcommon.NewDummyUser(1, dbClient, t.Context(), app.Clock)
	firstGroupPasskey := createPasskey(t, "first-group-key", true, false, userOb.ID, dbClient)
	passkeyToMove := createPasskey(t, "moving-key", true, false, userOb.ID, dbClient)
	requestSessionToken := createSession(t, userOb.ID, firstGroupPasskey.ID, &passkeyToMove.ID, app)
	otherFirstGroupLoginPasskey := createPasskey(t, "other-login", true, false, userOb.ID, dbClient)
	otherFirstGroupElevationPasskey := createPasskey(t, "other-elevation", true, false, userOb.ID, dbClient)
	sessionTokenToDemote := createSession(
		t,
		userOb.ID,
		otherFirstGroupLoginPasskey.ID,
		&otherFirstGroupElevationPasskey.ID,
		app,
	)

	respRecorder := testcommon.Post(
		t, app.Server,
		"/api/v1/self/passkeys/"+passkeyToMove.ID.String()+"/move-group/",
		passkeys.MoveGroupPayload{
			IsSecondGroup: true,
		},
		testcommon.WithBearerToken(requestSessionToken),
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

	requestSession := dbClient.Session.Query().
		Where(session.HashedToken(testcommon.HashSessionToken(t, requestSessionToken))).
		OnlyX(t.Context())
	require.True(t, requestSession.IsSudo)
	require.Equal(t, firstGroupPasskey.ID, requestSession.PasskeyID)
	require.Equal(t, passkeyToMove.ID, *requestSession.ElevationPasskeyID)

	demotedSession, stdErr := dbClient.Session.Query().
		Where(session.HashedToken(testcommon.HashSessionToken(t, sessionTokenToDemote))).
		Only(t.Context())
	require.NoError(t, stdErr)
	require.False(t, demotedSession.IsSudo)
	require.Nil(t, demotedSession.ElevationPasskeyID)
	require.Equal(
		t,
		// Should remain unchanged since the first passkey of the session is still valid
		otherFirstGroupLoginPasskey.ID, demotedSession.PasskeyID,
	)
}

func TestMoveGroup_CreateSecondGroup_MoveLoginPasskey(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)
	dbClient := app.Database.Client()
	userOb := testcommon.NewDummyUser(1, dbClient, t.Context(), app.Clock)
	loginPasskey := createPasskey(t, "login-key", true, false, userOb.ID, dbClient)
	elevationPasskey := createPasskey(t, "elevation-key", true, false, userOb.ID, dbClient)
	sessionToken := createSession(t, userOb.ID, loginPasskey.ID, &elevationPasskey.ID, app)

	respRecorder := testcommon.Post(
		t, app.Server,
		"/api/v1/self/passkeys/"+loginPasskey.ID.String()+"/move-group/",
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
		Where(passkey.ID(loginPasskey.ID)).
		OnlyX(t.Context())
	require.True(t, movedPasskey.IsSecondGroup)

	sessionOb := dbClient.Session.Query().
		Where(session.HashedToken(testcommon.HashSessionToken(t, sessionToken))).
		OnlyX(t.Context())
	require.True(t, sessionOb.IsSudo)
	require.Equal(t, loginPasskey.ID, sessionOb.PasskeyID)
	require.Equal(t, elevationPasskey.ID, *sessionOb.ElevationPasskeyID)
}

func TestMoveGroup_ExistingSecondGroup(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)
	dbClient := app.Database.Client()
	userOb := testcommon.NewDummyUser(1, dbClient, t.Context(), app.Clock)
	firstGroupPasskey := createPasskey(t, "first-group-key", true, false, userOb.ID, dbClient)
	existingSecondGroupPasskey := createPasskey(t, "existing-second-group-key", true, true, userOb.ID, dbClient)
	passkeyToMove := createPasskey(t, "moving-key", true, false, userOb.ID, dbClient)
	requestSessionToken := createSession(t, userOb.ID, firstGroupPasskey.ID, &existingSecondGroupPasskey.ID, app)
	sessionTokenToDemote := createSession(t, userOb.ID, existingSecondGroupPasskey.ID, &passkeyToMove.ID, app)

	respRecorder := testcommon.Post(
		t, app.Server,
		"/api/v1/self/passkeys/"+passkeyToMove.ID.String()+"/move-group/",
		passkeys.MoveGroupPayload{
			IsSecondGroup: true,
		},
		testcommon.WithBearerToken(requestSessionToken),
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

	requestSession := dbClient.Session.Query().
		Where(session.HashedToken(testcommon.HashSessionToken(t, requestSessionToken))).
		OnlyX(t.Context())
	require.True(t, requestSession.IsSudo)
	require.Equal(t, firstGroupPasskey.ID, requestSession.PasskeyID)
	require.Equal(t, existingSecondGroupPasskey.ID, *requestSession.ElevationPasskeyID)

	demotedSession, stdErr := dbClient.Session.Query().
		Where(session.HashedToken(testcommon.HashSessionToken(t, sessionTokenToDemote))).
		Only(t.Context())
	require.NoError(t, stdErr)
	require.False(t, demotedSession.IsSudo)
	require.Nil(t, demotedSession.ElevationPasskeyID)
	require.Equal(
		t,
		// Should remain unchanged since the first passkey of the session is still valid
		existingSecondGroupPasskey.ID, demotedSession.PasskeyID,
	)
}

func TestMoveGroup_NoSecondGroup_CantMoveNonSessionPasskey(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)
	dbClient := app.Database.Client()
	userOb := testcommon.NewDummyUser(1, dbClient, t.Context(), app.Clock)
	loginPasskey := createPasskey(t, "login-key", true, false, userOb.ID, dbClient)
	elevationPasskey := createPasskey(t, "elevation-key", true, false, userOb.ID, dbClient)
	nonSessionPasskey := createPasskey(t, "unused-key", true, false, userOb.ID, dbClient)
	sessionToken := createSession(t, userOb.ID, loginPasskey.ID, &elevationPasskey.ID, app)

	respRecorder := testcommon.Post(
		t, app.Server,
		"/api/v1/self/passkeys/"+nonSessionPasskey.ID.String()+"/move-group/",
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
						"to move back, first log in with different passkeys or disable two group auth entirely",
					Code: "GROUP_MOVE_CONSTRAINT",
				},
			},
		},
	)
}

func TestMoveGroup_SamePasskeyUsedTwice_SendsConflictError(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)
	dbClient := app.Database.Client()
	userOb := testcommon.NewDummyUser(1, dbClient, t.Context(), app.Clock)
	passkeyOb := createPasskey(t, "login-key", true, false, userOb.ID, dbClient)
	sessionToken := createSession(t, userOb.ID, passkeyOb.ID, &passkeyOb.ID, app)

	respRecorder := testcommon.Post(
		t, app.Server,
		"/api/v1/self/passkeys/"+passkeyOb.ID.String()+"/move-group/",
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
						"to move back, first log in with different passkeys or disable two group auth entirely",
					Code: "GROUP_MOVE_CONSTRAINT",
				},
			},
		},
	)
}

func TestMoveGroup_NoElevationPasskey_SendsConflictError(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)
	dbClient := app.Database.Client()
	userOb := testcommon.NewDummyUser(1, dbClient, t.Context(), app.Clock)
	loginPk := createPasskey(t, "login-key", true, false, userOb.ID, dbClient)
	// Create a sudo session that somehow doesn't have an elevation passkey
	sessionToken := createSession(t, userOb.ID, loginPk.ID, nil, app)
	app.Database.Client().Session.Update().
		Where(session.HashedToken(testcommon.HashSessionToken(t, sessionToken))).
		SetIsSudo(true).
		ExecX(t.Context())

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
						"to move back, first log in with different passkeys or disable two group auth entirely",
					Code: "GROUP_MOVE_CONSTRAINT",
				},
			},
		},
	)
}

func TestMoveGroup_ExistingSecondGroup_TargetIsLoginPasskey_SendsConflictError(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)
	dbClient := app.Database.Client()
	userOb := testcommon.NewDummyUser(1, dbClient, t.Context(), app.Clock)
	loginPasskey := createPasskey(t, "login-key", true, false, userOb.ID, dbClient)
	elevationPasskey := createPasskey(t, "elevation-key", true, true, userOb.ID, dbClient)
	sessionToken := createSession(t, userOb.ID, loginPasskey.ID, &elevationPasskey.ID, app)

	respRecorder := testcommon.Post(
		t, app.Server,
		"/api/v1/self/passkeys/"+loginPasskey.ID.String()+"/move-group/",
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
						"to move back, first log in with different passkeys or disable two group auth entirely",
					Code: "GROUP_MOVE_CONSTRAINT",
				},
			},
		},
	)
}

func TestMoveGroup_ExistingSecondGroup_TargetIsElevationPasskey_SendsConflictError(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)
	dbClient := app.Database.Client()
	userOb := testcommon.NewDummyUser(1, dbClient, t.Context(), app.Clock)
	loginPasskey := createPasskey(t, "login-key", true, false, userOb.ID, dbClient)
	elevationPasskey := createPasskey(t, "elevation-key", true, true, userOb.ID, dbClient)
	sessionToken := createSession(t, userOb.ID, loginPasskey.ID, &elevationPasskey.ID, app)

	respRecorder := testcommon.Post(
		t, app.Server,
		"/api/v1/self/passkeys/"+elevationPasskey.ID.String()+"/move-group/",
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
						"to move back, first log in with different passkeys or disable two group auth entirely",
					Code: "GROUP_MOVE_CONSTRAINT",
				},
			},
		},
	)
}
