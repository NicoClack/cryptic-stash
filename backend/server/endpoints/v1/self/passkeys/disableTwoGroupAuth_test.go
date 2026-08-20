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

func TestDisableTwoGroupAuth(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)
	dbClient := app.Database.Client()
	userOb := testcommon.NewDummyUser(1, dbClient, t.Context(), app.Clock)
	firstGroupPasskey := createPasskey(t, "first-group-key", true, false, userOb.ID, dbClient)
	secondGroupPasskey := createPasskey(t, "second-group-key", true, true, userOb.ID, dbClient)
	createPasskey(t, "another-second-group-key", true, true, userOb.ID, dbClient)
	sessionToken := createSession(t, userOb.ID, firstGroupPasskey.ID, new(secondGroupPasskey.ID), app)

	respRecorder := testcommon.Post(
		t, app.Server,
		"/api/v1/self/passkeys/disable-two-group-auth/",
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
	count, stdErr := dbClient.Passkey.Query().
		Where(passkey.UserID(userOb.ID), passkey.IsSecondGroup(true)).
		Count(t.Context())
	require.NoError(t, stdErr)
	require.Zero(t, count)
}

func TestDisableTwoGroupAuth_NoSecondGroupPasskeys_NoOp(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)
	dbClient := app.Database.Client()
	userOb := testcommon.NewDummyUser(1, dbClient, t.Context(), app.Clock)
	passkeyOb := createPasskey(t, "login-key", true, false, userOb.ID, dbClient)
	sessionToken := createSession(t, userOb.ID, passkeyOb.ID, new(passkeyOb.ID), app)

	respRecorder := testcommon.Post(
		t, app.Server,
		"/api/v1/self/passkeys/disable-two-group-auth/",
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
}
