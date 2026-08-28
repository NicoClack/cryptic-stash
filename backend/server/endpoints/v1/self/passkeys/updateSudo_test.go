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

func TestUpdateSudo_Enable(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)
	dbClient := app.Database.Client()
	userOb := testcommon.NewDummyUser(1, dbClient, t.Context(), app.Clock)
	sudoPasskey := createPasskey(t, "sudo-key", true, false, userOb.ID, dbClient)
	nonSudoPasskey := createPasskey(t, "non-sudo-key", false, false, userOb.ID, dbClient)
	sessionToken := createSession(t, userOb.ID, sudoPasskey.ID, &sudoPasskey.ID, app)

	respRecorder := testcommon.Post(
		t, app.Server,
		"/api/v1/self/passkeys/"+nonSudoPasskey.ID.String()+"/update-sudo/",
		passkeys.UpdateSudoPayload{
			AllowSudo: true,
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
	updated := dbClient.Passkey.Query().
		Where(passkey.ID(nonSudoPasskey.ID)).
		OnlyX(t.Context())
	require.True(t, updated.AllowSudo)
}

func TestUpdateSudo_Enable_PasskeyUsedBySession(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)
	dbClient := app.Database.Client()
	userOb := testcommon.NewDummyUser(1, dbClient, t.Context(), app.Clock)
	loginPasskey := createPasskey(t, "login-key", false, false, userOb.ID, dbClient)
	elevationPasskey := createPasskey(t, "elevation-key", true, false, userOb.ID, dbClient)
	sessionToken := createSession(t, userOb.ID, loginPasskey.ID, &elevationPasskey.ID, app)

	respRecorder := testcommon.Post(
		t, app.Server,
		"/api/v1/self/passkeys/"+loginPasskey.ID.String()+"/update-sudo/",
		passkeys.UpdateSudoPayload{
			AllowSudo: true,
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
	updated := dbClient.Passkey.Query().
		Where(passkey.ID(loginPasskey.ID)).
		OnlyX(t.Context())
	require.True(t, updated.AllowSudo)

	sudoSession := dbClient.Session.Query().
		Where(session.ElevationPasskeyID(elevationPasskey.ID)).
		OnlyX(t.Context())
	require.True(t, sudoSession.IsSudo)
	require.Equal(t, elevationPasskey.ID, *sudoSession.ElevationPasskeyID)
}

func TestUpdateSudo_Disable_WithOtherSudoPasskey(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)
	dbClient := app.Database.Client()
	userOb := testcommon.NewDummyUser(1, dbClient, t.Context(), app.Clock)
	passkeyToDemote := createPasskey(t, "demoting-passkey", true, false, userOb.ID, dbClient)
	otherSudoPasskey := createPasskey(t, "sudo-key", true, false, userOb.ID, dbClient)
	sessionToken := createSession(t, userOb.ID, passkeyToDemote.ID, &otherSudoPasskey.ID, app)
	nonSudoPasskey := createPasskey(t, "non-sudo-key", false, false, userOb.ID, dbClient)
	sessionTokenToDemote := createSession(t, userOb.ID, passkeyToDemote.ID, &nonSudoPasskey.ID, app)

	respRecorder := testcommon.Post(
		t, app.Server,
		"/api/v1/self/passkeys/"+passkeyToDemote.ID.String()+"/update-sudo/",
		passkeys.UpdateSudoPayload{
			AllowSudo: false,
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

	updated := dbClient.Passkey.Query().
		Where(passkey.ID(passkeyToDemote.ID)).
		OnlyX(t.Context())
	require.False(t, updated.AllowSudo)

	// The active session remains sudo, because it has otherSudoPasskey
	sudoSession := dbClient.Session.Query().
		Where(session.ElevationPasskeyID(otherSudoPasskey.ID)).
		OnlyX(t.Context())
	require.True(t, sudoSession.IsSudo)
	require.Equal(t, passkeyToDemote.ID, sudoSession.PasskeyID)

	demotedSession, stdErr := dbClient.Session.Query().
		Where(session.HashedToken(testcommon.HashSessionToken(t, sessionTokenToDemote))).
		Only(t.Context())
	require.NoError(t, stdErr)
	require.False(t, demotedSession.IsSudo)
	require.Nil(t, demotedSession.ElevationPasskeyID)
	require.Equal(
		t,
		// Should be unchanged
		passkeyToDemote.ID, demotedSession.PasskeyID,
	)
}

// One sudo, one non-sudo in the session
func TestUpdateSudo_Disable_TargetIsOnlySudoInSession_SendsConflictError(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)
	dbClient := app.Database.Client()
	userOb := testcommon.NewDummyUser(1, dbClient, t.Context(), app.Clock)
	nonSudoPasskey := createPasskey(t, "non-sudo-key", false, false, userOb.ID, dbClient)
	sudoPasskeyToDemote := createPasskey(t, "sudo-key", true, false, userOb.ID, dbClient)
	sessionToken := createSession(t, userOb.ID, nonSudoPasskey.ID, &sudoPasskeyToDemote.ID, app)

	respRecorder := testcommon.Post(
		t, app.Server,
		"/api/v1/self/passkeys/"+sudoPasskeyToDemote.ID.String()+"/update-sudo/",
		passkeys.UpdateSudoPayload{
			AllowSudo: false,
		},
		testcommon.WithBearerToken(sessionToken),
	)

	testcommon.AssertJSONResponse(
		t, respRecorder,
		http.StatusConflict,
		gin.H{
			"errors": []servercommon.ErrorDetail{
				{
					Message: "can't disable sudo on this passkey, as doing so would remove sudo access from your session",
					Code:    "SUDO_CONSTRAINT",
				},
			},
		},
	)
}

func TestUpdateSudo_Disable_SamePasskeyUsedTwice_RejectsSessionPasskeyAsTarget(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)
	dbClient := app.Database.Client()
	userOb := testcommon.NewDummyUser(1, dbClient, t.Context(), app.Clock)
	passkeyOb := createPasskey(t, "login-key", true, false, userOb.ID, dbClient)
	sessionToken := createSession(t, userOb.ID, passkeyOb.ID, &passkeyOb.ID, app)

	respRecorder := testcommon.Post(
		t, app.Server,
		"/api/v1/self/passkeys/"+passkeyOb.ID.String()+"/update-sudo/",
		passkeys.UpdateSudoPayload{
			AllowSudo: false,
		},
		testcommon.WithBearerToken(sessionToken),
	)

	testcommon.AssertJSONResponse(
		t, respRecorder,
		http.StatusConflict,
		gin.H{
			"errors": []servercommon.ErrorDetail{
				{
					Message: "can't disable sudo on this passkey, as doing so would remove sudo access from your session",
					Code:    "SUDO_CONSTRAINT",
				},
			},
		},
	)
}

func TestUpdateSudo_Disable_NoElevationPasskey_RejectsSessionPasskeyAsTarget(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)
	dbClient := app.Database.Client()
	userOb := testcommon.NewDummyUser(1, dbClient, t.Context(), app.Clock)
	passkeyOb := createPasskey(t, "login-key", true, false, userOb.ID, dbClient)
	// Create a sudo session that somehow doesn't have an elevation passkey
	sessionToken := createSession(t, userOb.ID, passkeyOb.ID, nil, app)
	dbClient.Session.Update().
		Where(session.HashedToken(testcommon.HashSessionToken(t, sessionToken))).
		SetIsSudo(true).
		ExecX(t.Context())

	respRecorder := testcommon.Post(
		t, app.Server,
		"/api/v1/self/passkeys/"+passkeyOb.ID.String()+"/update-sudo/",
		passkeys.UpdateSudoPayload{
			AllowSudo: false,
		},
		testcommon.WithBearerToken(sessionToken),
	)

	testcommon.AssertJSONResponse(
		t, respRecorder,
		http.StatusConflict,
		gin.H{
			"errors": []servercommon.ErrorDetail{
				{
					Message: "can't disable sudo on this passkey, as doing so would remove sudo access from your session",
					Code:    "SUDO_CONSTRAINT",
				},
			},
		},
	)
}
