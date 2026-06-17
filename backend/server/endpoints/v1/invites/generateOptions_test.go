package invites_test

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common/testcommon"
	"github.com/NicoClack/cryptic-stash/backend/server/endpoints/v1/invites"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/NicoClack/cryptic-stash/backend/testhelpers"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/require"
)

func TestGenerateOptions(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)
	email := "test@example.com"
	inviteOb, code := createInvite(t, app, email, app.Clock.Now().Add(time.Hour))
	require.Equal(t, uuid.Nil, inviteOb.UserID)

	respRecorder := testcommon.Post(
		t, app.Server,
		fmt.Sprintf("/api/v1/invites/%s/generate-options", inviteOb.ID),
		nil,
		testcommon.WithBearerToken(code),
	)

	require.Equal(t, http.StatusOK, respRecorder.Code)
	var resp invites.GenerateOptionsResponse
	stdErr := json.Unmarshal(respRecorder.Body.Bytes(), &resp)
	require.NoError(t, stdErr)
	require.Empty(t, resp.Errors)
	require.Equal(t, email, resp.PublicKey.User.Name)
	require.Equal(t, email, resp.PublicKey.User.DisplayName)
	require.Len(t, resp.PublicKey.Challenge, 32)

	updatedInvite := app.Database.Client().Invite.GetX(t.Context(), inviteOb.ID)
	require.NotNil(t, updatedInvite.WebAuthnSession)
	require.Equal(t, resp.PublicKey.Challenge.String(), updatedInvite.WebAuthnSession.Challenge)
	require.Equal(t, uuid.Nil, updatedInvite.UserID)
}

func TestGenerateOptions_InvalidToken(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)
	inviteOb, _ := createInvite(t, app, "test@example.com", app.Clock.Now().Add(time.Hour))

	respRecorder := testcommon.Post(
		t, app.Server,
		fmt.Sprintf("/api/v1/invites/%s/generate-options", inviteOb.ID),
		nil,
		testcommon.WithBearerToken(base64.RawURLEncoding.EncodeToString(
			app.Core.RandomAuthCode(), // Wrong auth code
		)),
	)
	testcommon.AssertJSONResponse(
		t, respRecorder,
		http.StatusUnauthorized,
		gin.H{
			"errors": []servercommon.ErrorDetail{},
		},
	)
}

func TestGenerateOptions_ExpiredInvite(t *testing.T) {
	t.Parallel()

	clock := clockwork.NewFakeClock()
	app := testhelpers.NewApp(t, &testhelpers.AppOptions{
		Clock: clock,
	})
	inviteOb, code := createInvite(t, app, "test@example.com", clock.Now())

	respRecorder := testcommon.Post(
		t, app.Server,
		fmt.Sprintf("/api/v1/invites/%s/generate-options", inviteOb.ID),
		nil,
		testcommon.WithBearerToken(code),
	)
	require.Equal(t, http.StatusOK, respRecorder.Code)

	clock.Advance(app.Env.WEBAUTHN_SESSION_TIMEOUT)
	respRecorder = testcommon.Post(
		t, app.Server,
		fmt.Sprintf("/api/v1/invites/%s/generate-options", inviteOb.ID),
		nil,
		testcommon.WithBearerToken(code),
	)
	testcommon.AssertJSONResponse(
		t, respRecorder,
		http.StatusUnauthorized,
		gin.H{
			"errors": []servercommon.ErrorDetail{},
		},
	)
}
