package superuser_test

// Tests that span both endpoints

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/auth"
	"github.com/NicoClack/cryptic-stash/backend/common/dbcommon"
	"github.com/NicoClack/cryptic-stash/backend/common/testcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/ent/session"
	"github.com/NicoClack/cryptic-stash/backend/server/endpoints/v1/users/superuser"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/NicoClack/cryptic-stash/backend/testhelpers"
	"github.com/descope/virtualwebauthn"
	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func createUserWithPasskey(
	t *testing.T,
	counter int,
	allowSuperUser bool,
	app *testhelpers.App,
	setupAuthenticator func(*virtualwebauthn.Authenticator),
) (
	*ent.User,
	*ent.Passkey,
	virtualwebauthn.Credential,
	virtualwebauthn.Authenticator,
) {
	userOb := testcommon.NewDummyUser(counter, app.TestDatabase.Client(), t.Context(), app.Clock)

	vAuthenticator := virtualwebauthn.NewAuthenticator()
	if setupAuthenticator != nil {
		setupAuthenticator(&vAuthenticator)
	} else {
		// Simulate a physical security key
		vAuthenticator.Options.Transports = []virtualwebauthn.Transport{
			virtualwebauthn.TransportUSB,
			virtualwebauthn.TransportNFC,
		}
		vAuthenticator.Options.UserHandle = userOb.ID[:]
	}
	credential := virtualwebauthn.NewCredential(virtualwebauthn.KeyTypeEC2)
	vAuthenticator.AddCredential(credential)

	passkeyOb, stdErr := dbcommon.WithReadWriteTx(
		t.Context(), app.Database,
		func(tx *ent.Tx, ctx context.Context) (*ent.Passkey, error) {
			options, sessionData, wrappedErr := app.Auth.StartRegisterPasskey(&auth.RealWebAuthnUser{
				User: userOb,
			}, t.Context())
			if wrappedErr != nil {
				return nil, wrappedErr
			}

			credentialJSON := virtualwebauthn.CreateAttestationResponse(
				testcommon.NewWebAuthnRelyingParty(app.Env),
				vAuthenticator,
				credential,
				virtualwebauthn.AttestationOptions{
					Challenge: options.Challenge,
				},
			)
			parsedCredential, stdErr := protocol.ParseCredentialCreationResponseBytes([]byte(credentialJSON))
			if stdErr != nil {
				return nil, stdErr
			}
			passkeyOb, wrappedErr := app.Auth.FinishRegisterPasskey(
				"Test Passkey",
				allowSuperUser,
				userOb.Username,
				sessionData,
				parsedCredential,
				tx,
				ctx,
				func(userID uuid.UUID, tx *ent.Tx) (*ent.User, error) {
					return userOb, nil
				},
			)
			if wrappedErr != nil {
				return nil, wrappedErr
			}
			return passkeyOb, nil
		},
	)
	require.NoError(t, stdErr)
	return userOb, passkeyOb, credential, vAuthenticator
}

func TestElevationFlow_SinglePasskey(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)
	dbClient := app.Database.Client()
	relyingParty := testcommon.NewWebAuthnRelyingParty(app.Env)
	userOb, passkeyOb, credential, vAuthenticator := createUserWithPasskey(t, 1, true, app, nil)
	sessionToken := createSession(t, false, userOb.ID, passkeyOb.ID, app)

	// Ensure a small gap so we can test that expiresAt is reset.
	// We can't use a mocked clock because go-webauthn and the TempKeyValue service would end up
	// disagreeing on the current time, causing the WebAuthn session TTL in TempKeyValue to immediately expire.
	time.Sleep(10 * time.Millisecond)
	elevationStartedAt := time.Now()

	decodedToken, stdErr := base64.RawURLEncoding.DecodeString(sessionToken)
	require.NoError(t, stdErr)
	hashedToken := sha256.Sum256(decodedToken)

	sessionOb, stdErr := dbClient.Session.Query().
		Where(session.HashedToken(hashedToken[:])).
		Only(t.Context())
	require.NoError(t, stdErr)
	require.False(t, sessionOb.SuperUserMode)

	startRecorder := testcommon.Post(
		t, app.Server,
		"/api/v1/users/superuser/start-elevation/",
		nil,
		testcommon.WithBearerToken(sessionToken),
	)
	require.Equal(t, http.StatusOK, startRecorder.Code)

	var startResp superuser.StartElevationResponse
	stdErr = json.Unmarshal(startRecorder.Body.Bytes(), &startResp)
	require.NoError(t, stdErr)
	require.NotEqual(t, uuid.Nil, startResp.WebAuthnSessionID)
	require.NotNil(t, startResp.PublicKey)
	require.Equal(t, relyingParty.ID, startResp.PublicKey.RelyingPartyID)

	require.Len(t, vAuthenticator.Credentials, 1)
	foundCredential := vAuthenticator.Credentials[0]
	require.Equal(t, credential, foundCredential)

	assertionResponse := virtualwebauthn.CreateAssertionResponse(
		relyingParty,
		vAuthenticator,
		foundCredential,
		virtualwebauthn.AssertionOptions{
			Challenge: startResp.PublicKey.Challenge,
		},
	)

	var parsedAssertion protocol.CredentialAssertionResponse
	stdErr = json.Unmarshal([]byte(assertionResponse), &parsedAssertion)
	require.NoError(t, stdErr)

	finishRecorder := testcommon.Post(
		t, app.Server,
		"/api/v1/users/superuser/finish-elevation/",
		superuser.FinishElevationPayload{
			CredentialAssertionResponse: parsedAssertion,
			WebAuthnSessionID:           startResp.WebAuthnSessionID,
		},
		testcommon.WithBearerToken(sessionToken),
	)
	testcommon.AssertJSONResponse(
		t, finishRecorder,
		http.StatusOK,
		superuser.FinishElevationResponse{
			Errors: []servercommon.ErrorDetail{},
		},
	)

	sessionOb, stdErr = dbClient.Session.Query().
		Where(session.HashedToken(hashedToken[:])).
		Only(t.Context())
	require.NoError(t, stdErr)
	require.True(t, sessionOb.SuperUserMode)
	require.Equal(t, sessionOb.UserID, userOb.ID)
	require.Greater(
		t,
		sessionOb.ExpiresAt,
		elevationStartedAt, // The session expiry should have been extended
	)
}

func TestElevationFlow_CredentialFromDifferentUser_SendsBadRequest(t *testing.T) {
	t.Parallel()

	runTest := func(t *testing.T, claimsOtherUserHandle bool, claimsOtherCredentialID bool) {
		app := testhelpers.NewApp(t, nil)
		relyingParty := testcommon.NewWebAuthnRelyingParty(app.Env)
		user1Ob, passkey1Ob, _, _ := createUserWithPasskey(t, 1, true, app, nil)
		sessionToken1 := createSession(t, false, user1Ob.ID, passkey1Ob.ID, app)

		var setupAuthenticator func(*virtualwebauthn.Authenticator)
		if claimsOtherUserHandle {
			setupAuthenticator = func(vAuthenticator *virtualwebauthn.Authenticator) {
				vAuthenticator.Options.Transports = []virtualwebauthn.Transport{
					virtualwebauthn.TransportUSB,
					virtualwebauthn.TransportNFC,
				}
				// User 2 claims to have a passkey for user 1
				vAuthenticator.Options.UserHandle = user1Ob.ID[:]
			}
		}
		_, _, credential2, vAuthenticator2 := createUserWithPasskey(
			t,
			2,
			true,
			app,
			setupAuthenticator,
		)

		startRecorder := testcommon.Post(
			t, app.Server,
			"/api/v1/users/superuser/start-elevation/",
			nil,
			testcommon.WithBearerToken(sessionToken1),
		)
		require.Equal(t, http.StatusOK, startRecorder.Code)

		var startResp superuser.StartElevationResponse
		stdErr := json.Unmarshal(startRecorder.Body.Bytes(), &startResp)
		require.NoError(t, stdErr)

		// Use user's 2 credential to respond to user 1's WebAuthn challenge
		require.Len(t, vAuthenticator2.Credentials, 1)
		foundCredential := vAuthenticator2.Credentials[0]
		require.Equal(t, credential2, foundCredential)

		assertionResponse := virtualwebauthn.CreateAssertionResponse(
			relyingParty,
			vAuthenticator2,
			foundCredential,
			virtualwebauthn.AssertionOptions{
				Challenge: startResp.PublicKey.Challenge,
			},
		)

		var parsedAssertion protocol.CredentialAssertionResponse
		stdErr = json.Unmarshal([]byte(assertionResponse), &parsedAssertion)
		require.NoError(t, stdErr)

		if claimsOtherCredentialID {
			// User 2 claims to have the credential ID of user 1's passkey
			parsedAssertion.RawID = passkey1Ob.CredentialID
		}
		finishRecorder := testcommon.Post(
			t, app.Server,
			"/api/v1/users/superuser/finish-elevation/",
			superuser.FinishElevationPayload{
				CredentialAssertionResponse: parsedAssertion,
				WebAuthnSessionID:           startResp.WebAuthnSessionID,
			},
			testcommon.WithBearerToken(sessionToken1),
		)

		// Should be rejected due to the user mismatch
		testcommon.AssertJSONResponse(
			t, finishRecorder,
			http.StatusBadRequest,
			gin.H{
				"errors": []servercommon.ErrorDetail{
					{
						Message: "invalid credential",
						Code:    "INVALID_CREDENTIAL",
					},
				},
			},
		)

		// Verify that user 1 is still not elevated
		decodedToken, stdErr := base64.RawURLEncoding.DecodeString(sessionToken1)
		require.NoError(t, stdErr)
		hashedToken := sha256.Sum256(decodedToken)
		sessionOb, stdErr := app.Database.Client().Session.Query().
			Where(session.HashedToken(hashedToken[:])).
			Only(t.Context())
		require.NoError(t, stdErr)
		require.False(t, sessionOb.SuperUserMode)
	}

	t.Run("Authenticator responds with wrong passkey", func(t *testing.T) {
		t.Parallel()
		runTest(t, false, false)
		// Internal error is "User does not own all credentials from the allowed credential list"
	})
	t.Run("Authenticator claims to have a passkey for a different user", func(t *testing.T) {
		t.Parallel()
		runTest(t, true, false)
		// Internal error is "The credential ID provided is not owned by the user"
	})
	t.Run("Authenticator claims to have the same passkey as another user (same credential ID)", func(t *testing.T) {
		t.Parallel()
		runTest(t, true, true)
		// Internal error is "Error validating the assertion signature"
	})
	t.Run("Authenticator claims to have same credential ID but for its own user", func(t *testing.T) {
		t.Parallel()
		runTest(t, false, true)
		// Internal error is "User does not own all credentials from the allowed credential list"
	})
}

func TestElevationFlow_RejectsTamperedSignature(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)
	relyingParty := testcommon.NewWebAuthnRelyingParty(app.Env)
	userOb, passkeyOb, credential, vAuthenticator := createUserWithPasskey(t, 1, true, app, nil)
	sessionToken := createSession(t, false, userOb.ID, passkeyOb.ID, app)

	startRecorder := testcommon.Post(
		t, app.Server,
		"/api/v1/users/superuser/start-elevation/",
		nil,
		testcommon.WithBearerToken(sessionToken),
	)
	require.Equal(t, http.StatusOK, startRecorder.Code)

	var startResp superuser.StartElevationResponse
	stdErr := json.Unmarshal(startRecorder.Body.Bytes(), &startResp)
	require.NoError(t, stdErr)

	require.Len(t, vAuthenticator.Credentials, 1)
	foundCredential := vAuthenticator.Credentials[0]
	require.Equal(t, credential, foundCredential)

	assertionResponse := virtualwebauthn.CreateAssertionResponse(
		relyingParty,
		vAuthenticator,
		foundCredential,
		virtualwebauthn.AssertionOptions{
			Challenge: startResp.PublicKey.Challenge,
		},
	)

	var parsedAssertion protocol.CredentialAssertionResponse
	stdErr = json.Unmarshal([]byte(assertionResponse), &parsedAssertion)
	require.NoError(t, stdErr)

	// Tamper with the signature
	require.NotEmpty(t, parsedAssertion.AssertionResponse.Signature)
	parsedAssertion.AssertionResponse.Signature[0] ^= 0xFF

	finishRecorder := testcommon.Post(
		t, app.Server,
		"/api/v1/users/superuser/finish-elevation/",
		superuser.FinishElevationPayload{
			CredentialAssertionResponse: parsedAssertion,
			WebAuthnSessionID:           startResp.WebAuthnSessionID,
		},
		testcommon.WithBearerToken(sessionToken),
	)

	testcommon.AssertJSONResponse(
		t, finishRecorder,
		http.StatusBadRequest,
		gin.H{
			"errors": []servercommon.ErrorDetail{
				{
					Message: "invalid credential",
					Code:    "INVALID_CREDENTIAL",
				},
			},
		},
	)
}

func TestElevationFlow_GivenExpiredWebAuthnSession_RejectsValidSignature(t *testing.T) {
	t.Parallel()

	env := testcommon.DefaultEnv()
	// go-webauthn doesn't allow us to mock the time
	env.WEBAUTHN_SESSION_TIMEOUT = 250 * time.Millisecond
	app := testhelpers.NewApp(t, &testhelpers.AppOptions{
		Env: env,
	})
	relyingParty := testcommon.NewWebAuthnRelyingParty(app.Env)
	userOb, passkeyOb, credential, vAuthenticator := createUserWithPasskey(t, 1, true, app, nil)
	sessionToken := createSession(t, false, userOb.ID, passkeyOb.ID, app)

	startRecorder := testcommon.Post(
		t, app.Server,
		"/api/v1/users/superuser/start-elevation/",
		nil,
		testcommon.WithBearerToken(sessionToken),
	)
	require.Equal(t, http.StatusOK, startRecorder.Code)

	var startResp superuser.StartElevationResponse
	stdErr := json.Unmarshal(startRecorder.Body.Bytes(), &startResp)
	require.NoError(t, stdErr)

	require.Len(t, vAuthenticator.Credentials, 1)
	foundCredential := vAuthenticator.Credentials[0]
	require.Equal(t, credential, foundCredential)

	assertionResponse := virtualwebauthn.CreateAssertionResponse(
		relyingParty,
		vAuthenticator,
		foundCredential,
		virtualwebauthn.AssertionOptions{
			Challenge: startResp.PublicKey.Challenge,
		},
	)

	time.Sleep(env.WEBAUTHN_SESSION_TIMEOUT)

	var parsedAssertion protocol.CredentialAssertionResponse
	stdErr = json.Unmarshal([]byte(assertionResponse), &parsedAssertion)
	require.NoError(t, stdErr)

	finishRecorder := testcommon.Post(
		t, app.Server,
		"/api/v1/users/superuser/finish-elevation/",
		superuser.FinishElevationPayload{
			CredentialAssertionResponse: parsedAssertion,
			WebAuthnSessionID:           startResp.WebAuthnSessionID,
		},
		testcommon.WithBearerToken(sessionToken),
	)

	testcommon.AssertJSONResponse(
		t, finishRecorder,
		http.StatusBadRequest,
		gin.H{
			"errors": []servercommon.ErrorDetail{
				{
					Message: "WebAuthn session missing or expired",
					Code:    "INVALID_WEBAUTHN_SESSION",
				},
			},
		},
	)
}

func TestElevationFlow_GivenElevatedSession_RejectsFurtherElevations(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)
	dbClient := app.Database.Client()
	relyingParty := testcommon.NewWebAuthnRelyingParty(app.Env)
	userOb, passkeyOb, credential, vAuthenticator := createUserWithPasskey(t, 1, true, app, nil)
	sessionToken := createSession(t, false, userOb.ID, passkeyOb.ID, app)

	var sessionOb *ent.Session
	for i := range 2 {
		startRecorder := testcommon.Post(
			t, app.Server,
			"/api/v1/users/superuser/start-elevation/",
			nil,
			testcommon.WithBearerToken(sessionToken),
		)

		if i == 0 {
			require.Equal(t, http.StatusOK, startRecorder.Code)
		} else {
			// This check is only performed by the start endpoint so there's a slight opportunity to extend the session more
			// than it should be possible. But the user can extend the session much more by just waiting until their
			// regular session is about to expire and then completing the elevation flow normally,
			// since that resets the session expiry.
			// The main thing is this check stops the session from being infinitely extended without using an
			// endpoint explicitly intended for that.
			testcommon.AssertJSONResponse(
				t, startRecorder,
				http.StatusConflict,
				gin.H{
					"errors": []servercommon.ErrorDetail{
						{
							Code:    "SESSION_ALREADY_ELEVATED",
							Message: "session is already in superuser mode",
						},
					},
				},
			)

			lastExpiresAt := sessionOb.ExpiresAt
			sessionOb, stdErr := dbClient.Session.Query().
				Where(session.ID(sessionOb.ID)).
				Only(t.Context())
			require.NoError(t, stdErr)
			require.Equal(t, lastExpiresAt, sessionOb.ExpiresAt) // It shouldn't have been extended a second time
			break
		}

		var startResp superuser.StartElevationResponse
		stdErr := json.Unmarshal(startRecorder.Body.Bytes(), &startResp)
		require.NoError(t, stdErr)

		require.Len(t, vAuthenticator.Credentials, 1)
		foundCredential := vAuthenticator.Credentials[0]
		require.Equal(t, credential, foundCredential)

		assertionResponse := virtualwebauthn.CreateAssertionResponse(
			relyingParty,
			vAuthenticator,
			foundCredential,
			virtualwebauthn.AssertionOptions{
				Challenge: startResp.PublicKey.Challenge,
			},
		)

		var parsedAssertion protocol.CredentialAssertionResponse
		stdErr = json.Unmarshal([]byte(assertionResponse), &parsedAssertion)
		require.NoError(t, stdErr)

		finishRecorder := testcommon.Post(
			t, app.Server,
			"/api/v1/users/superuser/finish-elevation/",
			superuser.FinishElevationPayload{
				CredentialAssertionResponse: parsedAssertion,
				WebAuthnSessionID:           startResp.WebAuthnSessionID,
			},
			testcommon.WithBearerToken(sessionToken),
		)
		require.Equal(t, http.StatusOK, finishRecorder.Code)

		decodedToken, stdErr := base64.RawURLEncoding.DecodeString(sessionToken)
		require.NoError(t, stdErr)
		hashedToken := sha256.Sum256(decodedToken)

		sessionOb, stdErr = dbClient.Session.Query().
			Where(session.HashedToken(hashedToken[:])).
			Only(t.Context())
		require.NoError(t, stdErr)
	}
}
