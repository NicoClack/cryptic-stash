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
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestRename(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)
	dbClient := app.Database.Client()
	userOb := testcommon.NewDummyUser(1, dbClient, t.Context(), app.Clock)
	passkeyOb := createPasskey(t, "old-name", true, false, userOb.ID, dbClient)
	// Any passkey can be renamed, including ones used by the session like here
	sessionToken := createSession(t, userOb.ID, passkeyOb.ID, new(passkeyOb.ID), app)

	respRecorder := testcommon.Post(
		t, app.Server,
		"/api/v1/self/passkeys/"+passkeyOb.ID.String()+"/rename/",
		passkeys.RenamePayload{
			Name: "new-name",
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
		Where(passkey.ID(passkeyOb.ID)).
		OnlyX(t.Context())
	require.Equal(t, "new-name", updated.Name)
}

func TestRename_DuplicateName(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)
	dbClient := app.Database.Client()
	userOb := testcommon.NewDummyUser(1, dbClient, t.Context(), app.Clock)
	redPasskeyOb := createPasskey(t, "red", true, false, userOb.ID, dbClient)
	bluePasskeyOb := createPasskey(t, "blue", true, false, userOb.ID, dbClient)
	sessionToken := createSession(t, userOb.ID, redPasskeyOb.ID, new(redPasskeyOb.ID), app)

	respRecorder := testcommon.Post(
		t, app.Server,
		"/api/v1/self/passkeys/"+bluePasskeyOb.ID.String()+"/rename/",
		passkeys.RenamePayload{
			Name: "red", // same name as redPasskeyOb
		},
		testcommon.WithBearerToken(sessionToken),
	)

	testcommon.AssertJSONResponse(
		t, respRecorder,
		http.StatusConflict,
		gin.H{
			"errors": []servercommon.ErrorDetail{
				{
					Message: "passkey name already exists",
					Code:    "DUPLICATE_PASSKEY_NAME",
				},
			},
		},
	)
}

func TestRename_NotFound(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)
	dbClient := app.Database.Client()
	userOb := testcommon.NewDummyUser(1, dbClient, t.Context(), app.Clock)
	passkeyOb := createPasskey(t, "old-name", true, false, userOb.ID, dbClient)
	sessionToken := createSession(t, userOb.ID, passkeyOb.ID, new(passkeyOb.ID), app)

	respRecorder := testcommon.Post(
		t, app.Server,
		"/api/v1/self/passkeys/"+uuid.New().String()+"/rename/",
		passkeys.RenamePayload{
			Name: "new-name",
		},
		testcommon.WithBearerToken(sessionToken),
	)

	testcommon.AssertJSONResponse(
		t, respRecorder,
		http.StatusNotFound,
		gin.H{
			"errors": []servercommon.ErrorDetail{},
		},
	)
	require.Equal(t, http.StatusNotFound, respRecorder.Code)
}
