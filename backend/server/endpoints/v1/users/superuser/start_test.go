package superuser_test

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/auth"
	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/common/testcommon"
	"github.com/NicoClack/cryptic-stash/backend/server/endpoints/v1/users/superuser"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/NicoClack/cryptic-stash/backend/testhelpers"
	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestStartElevation(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)
	userOb := testcommon.NewDummyUser(1, app.Database.Client(), t.Context(), app.Clock)
	sessionToken, passkeyOb := createSessionAndPasskey(t, false, userOb, app)

	respRecorder := testcommon.Post(
		t, app.Server,
		"/api/v1/users/superuser/start-elevation/",
		nil,
		testcommon.WithBearerToken(sessionToken),
	)
	webAuthnSessionCreatedAt := time.Now()
	require.Equal(t, http.StatusOK, respRecorder.Code)

	var response superuser.StartElevationResponse
	stdErr := json.Unmarshal(respRecorder.Body.Bytes(), &response)
	require.NoError(t, stdErr)
	require.NotEqual(t, response.WebAuthnSessionID, uuid.Nil)
	require.NotNil(t, response.PublicKey)
	require.Len(t, response.PublicKey.Challenge, 32)
	require.Equal(t, "frontend.example.com", response.PublicKey.RelyingPartyID)
	require.Equal(t, 2*time.Minute, time.Duration(response.PublicKey.Timeout)*time.Millisecond)
	require.Len(t, response.PublicKey.AllowedCredentials, 1)
	require.Equal(t, passkeyOb.CredentialID, []byte(response.PublicKey.AllowedCredentials[0].CredentialID))
	require.Equal(t, passkeyOb.Credential.Descriptor(), response.PublicKey.AllowedCredentials[0])

	var webAuthnSessionData *webauthn.SessionData
	ok := app.TempKeyValue.Get(auth.WebAuthnSessionStoreName, response.WebAuthnSessionID.String(), &webAuthnSessionData)
	require.True(t, ok)
	require.Equal(t, response.PublicKey.Challenge.String(), webAuthnSessionData.Challenge)
	require.Equal(t, response.PublicKey.RelyingPartyID, webAuthnSessionData.RelyingPartyID)
	require.WithinDuration(
		t,
		webAuthnSessionCreatedAt.Add(2*time.Minute),
		webAuthnSessionData.Expires,
		100*time.Millisecond,
	)
}

func TestStartElevation_NoAuthHeader_SendsBadRequest(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)

	respRecorder := testcommon.Post(
		t, app.Server,
		"/api/v1/users/superuser/start-elevation/",
		nil,
	)
	testcommon.AssertJSONResponse(
		t, respRecorder,
		http.StatusBadRequest,
		gin.H{
			"errors": []servercommon.ErrorDetail{
				{
					Message: "authorization: header is required",
					Code:    "MISSING_AUTHORIZATION_HEADER",
				},
			},
		},
	)
}
func TestStartElevation_UnknownSessionToken_SendsUnauthorized(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)

	respRecorder := testcommon.Post(
		t, app.Server,
		"/api/v1/users/superuser/start-elevation/",
		nil,
		testcommon.WithBearerToken(base64.RawURLEncoding.EncodeToString(
			common.CryptoRandomBytes(32),
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

func TestStartElevation_MultipleRequests_UniqueSessionsAndChallenges(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)
	userOb := testcommon.NewDummyUser(1, app.Database.Client(), t.Context(), app.Clock)
	sessionToken, _ := createSessionAndPasskey(t, false, userOb, app)

	responses := make([]superuser.StartElevationResponse, 0, 3)
	for range 3 {
		respRecorder := testcommon.Post(
			t, app.Server,
			"/api/v1/users/superuser/start-elevation/",
			nil,
			testcommon.WithBearerToken(sessionToken),
		)
		require.Equal(t, http.StatusOK, respRecorder.Code)

		var response superuser.StartElevationResponse
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

func TestStartElevation_AlreadyElevated_SendsConflictError(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)
	userOb := testcommon.NewDummyUser(1, app.Database.Client(), t.Context(), app.Clock)
	sessionToken, _ := createSessionAndPasskey(t, true, userOb, app)

	respRecorder := testcommon.Post(
		t, app.Server,
		"/api/v1/users/superuser/start-elevation/",
		nil,
		testcommon.WithBearerToken(sessionToken),
	)
	testcommon.AssertJSONResponse(
		t, respRecorder,
		http.StatusConflict,
		gin.H{
			"errors": []servercommon.ErrorDetail{
				{
					Message: "session is already in sudo mode",
					Code:    "SESSION_ALREADY_ELEVATED",
				},
			},
		},
	)
}
