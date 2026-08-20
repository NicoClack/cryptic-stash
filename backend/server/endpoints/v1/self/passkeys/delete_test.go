package passkeys_test

import (
	"net/http"
	"testing"

	"github.com/NicoClack/cryptic-stash/backend/common/testcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent/passkey"
	"github.com/NicoClack/cryptic-stash/backend/ent/session"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/NicoClack/cryptic-stash/backend/testhelpers"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestDelete(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)
	dbClient := app.Database.Client()
	userOb := testcommon.NewDummyUser(1, dbClient, t.Context(), app.Clock)
	loginPasskey := createPasskey(t, "login-key", true, false, userOb.ID, dbClient)
	passkeyToDelete := createPasskey(t, "deleting-key", true, false, userOb.ID, dbClient)
	requestSessionToken := createSession(
		t,
		userOb.ID,
		loginPasskey.ID,
		new(loginPasskey.ID),
		app,
	)
	sessionTokenToDelete := createSession(t, userOb.ID, passkeyToDelete.ID, new(passkeyToDelete.ID), app)
	sessionTokenToDemote := createSession(t, userOb.ID, passkeyToDelete.ID, new(loginPasskey.ID), app)

	respRecorder := testcommon.Post(
		t, app.Server,
		"/api/v1/self/passkeys/"+passkeyToDelete.ID.String()+"/delete/",
		nil,
		testcommon.WithBearerToken(requestSessionToken),
	)

	testcommon.AssertJSONResponse(
		t, respRecorder,
		http.StatusOK,
		gin.H{
			"errors": []servercommon.ErrorDetail{},
		},
	)

	exists, stdErr := dbClient.Passkey.Query().
		Where(passkey.ID(passkeyToDelete.ID)).
		Exist(t.Context())
	require.NoError(t, stdErr)
	require.False(t, exists)

	requestSession := dbClient.Session.Query().
		Where(session.HashedToken(testcommon.HashSessionToken(t, requestSessionToken))).
		OnlyX(t.Context())
	require.True(t, requestSession.IsSudo)
	require.Equal(t, loginPasskey.ID, requestSession.PasskeyID)
	require.Equal(t, loginPasskey.ID, *requestSession.ElevationPasskeyID)

	exists, stdErr = dbClient.Session.Query().
		Where(session.HashedToken(testcommon.HashSessionToken(t, sessionTokenToDelete))).
		Exist(t.Context())
	require.NoError(t, stdErr)
	require.False(t, exists)

	// For now, this is just cascade deleted. In the future, this might be demoted like
	// TestUpdateSudo_Disable_WithOtherSudoPasskey does
	exists, stdErr = dbClient.Session.Query().
		Where(session.HashedToken(testcommon.HashSessionToken(t, sessionTokenToDemote))).
		Exist(t.Context())
	require.NoError(t, stdErr)
	require.False(t, exists)
}

func TestDelete_SessionPasskey_SendsConflictError(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)
	dbClient := app.Database.Client()
	userOb := testcommon.NewDummyUser(1, dbClient, t.Context(), app.Clock)
	loginPasskey := createPasskey(t, "login-key", true, false, userOb.ID, dbClient)
	elevationPasskey := createPasskey(t, "elevation-key", true, false, userOb.ID, dbClient)
	sessionToken := createSession(t, userOb.ID, loginPasskey.ID, new(elevationPasskey.ID), app)

	respRecorder := testcommon.Post(
		t, app.Server,
		"/api/v1/self/passkeys/"+loginPasskey.ID.String()+"/delete/",
		nil,
		testcommon.WithBearerToken(sessionToken),
	)

	testcommon.AssertJSONResponse(
		t, respRecorder,
		http.StatusConflict,
		gin.H{
			"errors": []servercommon.ErrorDetail{
				{
					Message: "can't delete a passkey that is currently in use by your session",
					Code:    "DELETE_CONSTRAINT",
				},
			},
		},
	)
}
