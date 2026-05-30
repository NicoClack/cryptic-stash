package login_test

// Tests that span both endpoints should go in login_test.go

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/auth"
	"github.com/NicoClack/cryptic-stash/backend/common/testcommon"
	"github.com/NicoClack/cryptic-stash/backend/server/endpoints/v1/users/login"
	"github.com/NicoClack/cryptic-stash/backend/testhelpers"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestLoginOptions(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)
	// Cryptic Stash uses username-less login, so we don't actually need to set up a user

	respRecorder := testcommon.Post(
		t, app.Server,
		"/api/v1/users/login/options/",
		nil,
	)
	require.Equal(t, http.StatusOK, respRecorder.Code)

	var response login.LoginOptionsResponse
	stdErr := json.Unmarshal(respRecorder.Body.Bytes(), &response)
	require.NoError(t, stdErr)
	require.Len(t, response.WebAuthnSessionID, len(uuid.NewString()))
	require.NotNil(t, response.PublicKey)
	require.Len(t, response.PublicKey.Challenge, 32)
	// We don't know who's logging in, so we can't suggest credentials
	require.Empty(t, response.PublicKey.AllowedCredentials)
	require.Equal(t, "frontend.example.com", response.PublicKey.RelyingPartyID)
	require.Equal(t, 2*time.Minute, time.Duration(response.PublicKey.Timeout)*time.Millisecond)

	sessionCount := app.Database.Client().Session.Query().CountX(t.Context())
	// The session in this response is a WebAuthn session, not a user session, it can't be used for performing actions
	require.Equal(t, 0, sessionCount)

	var sessionData *webauthn.SessionData
	ok := app.TempKeyValue.Get(auth.WebAuthnSessionStoreName, response.WebAuthnSessionID, &sessionData)
	require.True(t, ok)
	require.Equal(t, response.PublicKey.Challenge.String(), sessionData.Challenge)
	require.Equal(t, response.PublicKey.RelyingPartyID, sessionData.RelyingPartyID)
	require.Empty(t, sessionData.UserID)
	require.WithinDuration(t, app.Clock.Now().Add(2*time.Minute), sessionData.Expires, 10*time.Second)
}

func TestLoginOptions_MultipleRequests_UniqueSessionsAndChallenges(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)

	responses := make([]login.LoginOptionsResponse, 0, 3)
	for range 3 {
		respRecorder := testcommon.Post(
			t, app.Server,
			"/api/v1/users/login/options/",
			nil,
		)
		require.Equal(t, http.StatusOK, respRecorder.Code)

		var response login.LoginOptionsResponse
		stdErr := json.Unmarshal(respRecorder.Body.Bytes(), &response)
		require.NoError(t, stdErr)
		responses = append(responses, response)
	}

	require.NotEqual(t, responses[0], responses[1])
	require.NotEqual(t, responses[1], responses[2])
	require.NotEqual(t, responses[0], responses[2])
	require.NotEqual(t, responses[0].WebAuthnSessionID, responses[1].WebAuthnSessionID)
	require.NotEqual(t, responses[1].WebAuthnSessionID, responses[2].WebAuthnSessionID)
	require.NotEqual(t, responses[0].WebAuthnSessionID, responses[2].WebAuthnSessionID)
	require.NotEqual(t, responses[0].PublicKey.Challenge, responses[1].PublicKey.Challenge)
	require.NotEqual(t, responses[1].PublicKey.Challenge, responses[2].PublicKey.Challenge)
	require.NotEqual(t, responses[0].PublicKey.Challenge, responses[2].PublicKey.Challenge)
}
