package passkeys_test

import (
	"net/http"
	"testing"

	"github.com/NicoClack/cryptic-stash/backend/common/testcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent/passkey"
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
	sessionToken := createSession(t, true, loginPasskey.UserID, loginPasskey.ID, app)

	respRecorder := testcommon.Post(
		t, app.Server,
		"/api/v1/self/passkeys/"+passkeyToDelete.ID.String()+"/delete/",
		nil,
		testcommon.WithBearerToken(sessionToken),
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
}

func TestDelete_SessionPasskey_SendsConflictError(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)
	dbClient := app.Database.Client()
	userOb := testcommon.NewDummyUser(1, dbClient, t.Context(), app.Clock)
	passkeyOb := createPasskey(t, "login-key", true, false, userOb.ID, dbClient)
	sessionToken := createSession(t, true, passkeyOb.UserID, passkeyOb.ID, app)

	respRecorder := testcommon.Post(
		t, app.Server,
		"/api/v1/self/passkeys/"+passkeyOb.ID.String()+"/delete/",
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
