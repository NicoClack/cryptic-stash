package login_test

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/common/testcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent/session"
	"github.com/NicoClack/cryptic-stash/backend/server/endpoints/v1/users/login"
	"github.com/NicoClack/cryptic-stash/backend/testhelpers"
	"github.com/descope/virtualwebauthn"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/stretchr/testify/require"
)

func TestLoginFlow(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)
	origin := common.GetOrigin(app.Env.FRONTEND_BASE_URL)
	userOb := testcommon.NewDummyUser(1, app.TestDatabase.Client(), t.Context(), app.Clock)
	dbClient := app.Database.Client()

	vAuthenticator := virtualwebauthn.NewAuthenticator()
	credential := virtualwebauthn.NewCredential(virtualwebauthn.KeyTypeEC2)

	dbClient.Passkey.Create().
		SetCreatedAt(app.Clock.Now()).
		SetUpdatedAt(app.Clock.Now()).
		SetUserID(userOb.ID).
		SetName("Test Passkey").
		SetCredentialID(credential.ID).
		SetPublicKey(credential.Key.Data).
		SaveX(t.Context())

	vAuthenticator.AddCredential(credential)

	optionsRecorder := testcommon.Post(
		t, app.Server,
		"/api/v1/users/login/options/",
		nil,
	)
	require.Equal(t, http.StatusOK, optionsRecorder.Code)

	assertionOptions, stdErr := virtualwebauthn.ParseAssertionOptions(optionsRecorder.Body.String())
	require.NoError(t, stdErr)
	require.NotNil(t, assertionOptions)
	// The ceremony ID isn't part of the WebAuthn spec,
	// but the frontend needs to include it in its login/finish request later
	var optionsResp login.LoginOptionsResponse
	stdErr = json.Unmarshal(optionsRecorder.Body.Bytes(), &optionsResp)
	require.NoError(t, stdErr)

	foundCredential := vAuthenticator.FindAllowedCredential(*assertionOptions)
	require.NotNil(t, foundCredential)
	require.Equal(t, credential, *foundCredential)

	require.Equal(t, origin, assertionOptions.RelyingPartyID)

	assertionResponse := virtualwebauthn.CreateAssertionResponse(
		virtualwebauthn.RelyingParty{
			ID:     app.Env.FRONTEND_BASE_URL.Hostname(),
			Name:   "Cryptic Stash",
			Origin: origin,
		},
		vAuthenticator,
		credential,
		virtualwebauthn.AssertionOptions{
			Challenge: assertionOptions.Challenge,
		},
	)

	var parsedAssertion protocol.CredentialAssertionResponse
	stdErr = json.Unmarshal([]byte(assertionResponse), &parsedAssertion)
	require.NoError(t, stdErr)

	finishRecorder := testcommon.Post(
		t, app.Server,
		"/api/v1/users/login/finish/",
		login.LoginFinishPayload{
			CredentialAssertionResponse: parsedAssertion,
			WebAuthnSessionID:           optionsResp.SessionID,
		},
	)

	require.Equal(t, http.StatusOK, finishRecorder.Code)

	var finishResp login.LoginFinishResponse
	stdErr = json.Unmarshal(finishRecorder.Body.Bytes(), &finishResp)
	require.NoError(t, stdErr)
	require.Equal(t, userOb.ID.String(), finishResp.UserID)
	require.Len(t, finishResp.Token, 43) // 32 bytes

	decodedToken, stdErr := base64.RawURLEncoding.DecodeString(finishResp.Token)
	require.NoError(t, stdErr)
	require.Len(t, decodedToken, 32)
	hashedToken := sha256.Sum256(decodedToken)
	sessionOb, stdErr := dbClient.Session.Query().
		Where(
			session.HashedToken(hashedToken[:]),
		).
		Only(t.Context())
	require.NoError(t, stdErr)
	require.Equal(t, userOb.ID, sessionOb.UserID)
}

// func TestLoginFlow_ExpiredWebAuthnSession(t *testing.T) {
// 	t.Parallel()

// 	app := testhelpers.NewApp(t, nil)

// 	// Step 1: Call LoginOptions to get a session ID
// 	optionsRecorder := testcommon.Post(
// 		t, app.Server,
// 		"/api/v1/users/login/options/",
// 		nil,
// 	)
// 	require.Equal(t, http.StatusOK, optionsRecorder.Code)
// 	var optionsResp login.LoginOptionsResponse
// 	err := json.Unmarshal(optionsRecorder.Body.Bytes(), &optionsResp)
// 	require.NoError(t, err)

// 	// Step 2: Attempt to finish login with invalid session ID (no valid credential)
// 	// This demonstrates the full endpoint flow, even though we can't complete
// 	// a real WebAuthn assertion without proper credential setup
// 	finishRecorder := testcommon.Post(
// 		t, app.Server,
// 		"/api/v1/users/login/finish/",
// 		login.LoginFinishPayload{
// 			WebAuthnSessionID: "invalid-session-id",
// 		},
// 	)

// 	// Expect error: invalid/expired session
// 	require.Equal(t, http.StatusBadRequest, finishRecorder.Code)
// }

// func TestLoginFlow_InvalidCredentialAfterValidOptions(t *testing.T) {
// 	t.Parallel()

// 	app := testhelpers.NewApp(t, nil)

// 	// Step 1: Get valid session from options
// 	optionsRecorder := testcommon.Post(
// 		t, app.Server,
// 		"/api/v1/users/login/options/",
// 		nil,
// 	)
// 	require.Equal(t, http.StatusOK, optionsRecorder.Code)
// 	var optionsResp login.LoginOptionsResponse
// 	err := json.Unmarshal(optionsRecorder.Body.Bytes(), &optionsResp)
// 	require.NoError(t, err)

// 	// Step 2: Try to finish with the valid session ID but no valid credential
// 	// This will fail because we can't construct a real WebAuthn assertion
// 	finishRecorder := testcommon.Post(
// 		t, app.Server,
// 		"/api/v1/users/login/finish/",
// 		login.LoginFinishPayload{
// 			WebAuthnSessionID: optionsResp.SessionID,
// 		},
// 	)

// 	// Should return either 401 (invalid credential) or 400 (bad request due to missing assertion)
// 	require.True(
// 		t,
// 		finishRecorder.Code == http.StatusUnauthorized || finishRecorder.Code == http.StatusBadRequest,
// 		"Expected 401 or 400, got %d: %s",
// 		finishRecorder.Code,
// 		finishRecorder.Body.String(),
// 	)
// }

// func TestLoginFlow_ExpiredSessionBetweenOptionsAndFinish(t *testing.T) {
// 	t.Parallel()

// 	clock := clockwork.NewFakeClock()
// 	app := testhelpers.NewApp(t, &testhelpers.AppOptions{
// 		Clock: clock,
// 	})

// 	// Step 1: Get a valid session from LoginOptions
// 	optionsRecorder := testcommon.Post(
// 		t, app.Server,
// 		"/api/v1/users/login/options/",
// 		nil,
// 	)
// 	require.Equal(t, http.StatusOK, optionsRecorder.Code)

// 	var optionsResp login.LoginOptionsResponse
// 	err := json.Unmarshal(optionsRecorder.Body.Bytes(), &optionsResp)
// 	require.NoError(t, err)
// 	sessionID := optionsResp.SessionID

// 	// Step 2: Advance time past the session expiration (5 minutes by default)
// 	clock.Advance(6 * time.Minute)

// 	// Step 3: Try to finish login with an expired session
// 	finishRecorder := testcommon.Post(
// 		t, app.Server,
// 		"/api/v1/users/login/finish/",
// 		login.LoginFinishPayload{
// 			WebAuthnSessionID: sessionID,
// 		},
// 	)

// 	require.Equal(t, http.StatusBadRequest, finishRecorder.Code)
// }
