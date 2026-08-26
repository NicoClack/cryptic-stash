package passkeys_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common/testcommon"
	"github.com/NicoClack/cryptic-stash/backend/server/endpoints/v1/self/passkeys"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/NicoClack/cryptic-stash/backend/testhelpers"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestList(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)
	dbClient := app.Database.Client()
	userOb := testcommon.NewDummyUser(1, dbClient, t.Context(), app.Clock)

	firstGroupPasskey := createPasskey(t, "first-group-key", false, false, userOb.ID, dbClient)
	secondGroupPasskey := createPasskey(t, "second-group-key", true, true, userOb.ID, dbClient)
	_ = createPasskey(t, "other-first-group-key", false, false, userOb.ID, dbClient)
	lastUsedAt := time.Now()
	_, stdErr := dbClient.Passkey.UpdateOneID(firstGroupPasskey.ID).
		SetLastUsedAt(lastUsedAt).
		Save(t.Context())
	require.NoError(t, stdErr)
	sessionToken := createSession(t, userOb.ID, firstGroupPasskey.ID, new(secondGroupPasskey.ID), app)

	respRecorder := testcommon.Get(
		t, app.Server,
		"/api/v1/self/passkeys/",
		testcommon.WithBearerToken(sessionToken),
	)

	require.Equal(t, http.StatusOK, respRecorder.Code)

	var resp passkeys.ListResponse
	stdErr = json.Unmarshal(respRecorder.Body.Bytes(), &resp)
	require.NoError(t, stdErr)
	require.Empty(t, resp.Errors)
	require.Len(t, resp.FirstGroupPasskeys, 2)
	require.Len(t, resp.SecondGroupPasskeys, 1)

	for _, passkeyInfo := range resp.FirstGroupPasskeys {
		if passkeyInfo.ID == firstGroupPasskey.ID {
			require.Equal(t, "first-group-key", passkeyInfo.Name)
			require.False(t, passkeyInfo.AllowSudo)
			require.True(t, passkeyInfo.IsSessionFirst)
			require.False(t, passkeyInfo.IsSessionSecond)
			require.WithinDuration(t, firstGroupPasskey.CreatedAt, passkeyInfo.CreatedAt, 0)
			require.NotNil(t, passkeyInfo.LastUsedAt)
			require.WithinDuration(t, lastUsedAt, *passkeyInfo.LastUsedAt, time.Second)
		}
	}

	passkeyInfo := resp.SecondGroupPasskeys[0]
	require.Equal(t, secondGroupPasskey.ID, passkeyInfo.ID)
	require.Equal(t, "second-group-key", passkeyInfo.Name)
	require.True(t, passkeyInfo.AllowSudo)
	require.False(t, passkeyInfo.IsSessionFirst)
	require.True(t, passkeyInfo.IsSessionSecond)
	require.WithinDuration(t, secondGroupPasskey.CreatedAt, passkeyInfo.CreatedAt, 0)
}

func TestList_UnusedPasskey(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)
	dbClient := app.Database.Client()
	userOb := testcommon.NewDummyUser(1, dbClient, t.Context(), app.Clock)

	passkeyOb := createPasskey(t, "passkey", false, false, userOb.ID, dbClient)
	sessionToken := createSession(t, userOb.ID, passkeyOb.ID, new(passkeyOb.ID), app)

	respRecorder := testcommon.Get(
		t, app.Server,
		"/api/v1/self/passkeys/",
		testcommon.WithBearerToken(sessionToken),
	)

	require.Equal(t, http.StatusOK, respRecorder.Code)

	var resp passkeys.ListResponse
	stdErr := json.Unmarshal(respRecorder.Body.Bytes(), &resp)
	require.NoError(t, stdErr)
	require.Empty(t, resp.Errors)
	require.Len(t, resp.FirstGroupPasskeys, 1)
	require.Empty(t, resp.SecondGroupPasskeys)

	passkeyInfo := resp.FirstGroupPasskeys[0]
	require.Equal(t, passkeyOb.ID, passkeyInfo.ID)
	require.Equal(t, "passkey", passkeyInfo.Name)
	require.False(t, passkeyInfo.AllowSudo)
	require.True(t, passkeyInfo.IsSessionFirst)
	require.True(t, passkeyInfo.IsSessionSecond)
	require.WithinDuration(t, passkeyOb.CreatedAt, passkeyInfo.CreatedAt, 0)
	require.Nil(t, passkeyInfo.LastUsedAt)
}

func TestList_RequiresSudo(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)
	dbClient := app.Database.Client()
	userOb := testcommon.NewDummyUser(1, dbClient, t.Context(), app.Clock)
	passkeyOb := createPasskey(t, "test-key", false, false, userOb.ID, dbClient)
	nonSudoToken := createSession(t, userOb.ID, passkeyOb.ID, nil, app)

	respRecorder := testcommon.Get(
		t, app.Server,
		"/api/v1/self/passkeys/",
		testcommon.WithBearerToken(nonSudoToken),
	)

	testcommon.AssertJSONResponse(
		t, respRecorder,
		http.StatusForbidden,
		gin.H{
			"errors": []servercommon.ErrorDetail{
				{
					Message: "sudo mode required",
					Code:    "SUDO_MODE_REQUIRED",
				},
			},
		},
	)
}
