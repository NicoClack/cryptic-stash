package login_test

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
	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/common/dbcommon"
	"github.com/NicoClack/cryptic-stash/backend/common/testcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/ent/session"
	"github.com/NicoClack/cryptic-stash/backend/server/endpoints/v1/users/login"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/NicoClack/cryptic-stash/backend/testhelpers"
	"github.com/descope/virtualwebauthn"
	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func newRelyingParty(env *common.Env) virtualwebauthn.RelyingParty {
	return virtualwebauthn.RelyingParty{
		ID:     env.FRONTEND_BASE_URL.Hostname(),
		Name:   "Cryptic Stash",
		Origin: common.GetOrigin(env.FRONTEND_BASE_URL),
	}
}

func createUserWithCredential(
	t *testing.T,
	serverAssociatesWithUser bool,
	authenticatorAssociatesWithUser bool,
	app *testhelpers.App,
) (*ent.User, virtualwebauthn.Credential, virtualwebauthn.Authenticator) {
	userOb := testcommon.NewDummyUser(1, app.TestDatabase.Client(), t.Context(), app.Clock)

	vAuthenticator := virtualwebauthn.NewAuthenticator()
	if authenticatorAssociatesWithUser {
		vAuthenticator.Options.UserHandle = userOb.ID[:]
	} else {
		unknownUserID := uuid.New()
		vAuthenticator.Options.UserHandle = unknownUserID[:]
	}
	credential := virtualwebauthn.NewCredential(virtualwebauthn.KeyTypeEC2)
	vAuthenticator.AddCredential(credential)

	stdErr := dbcommon.WithWriteTx(
		t.Context(), app.Database,
		func(tx *ent.Tx, ctx context.Context) error {
			options, sessionData, wrappedErr := app.Auth.StartRegisterPasskey(&auth.RealWebAuthnUser{
				User: userOb,
			}, t.Context())
			if wrappedErr != nil {
				return wrappedErr
			}

			registeredCredential := credential
			if !serverAssociatesWithUser {
				registeredCredential = virtualwebauthn.NewCredential(virtualwebauthn.KeyTypeEC2)
			}
			credentialJSON := virtualwebauthn.CreateAttestationResponse(
				newRelyingParty(app.Env),
				vAuthenticator,
				registeredCredential,
				virtualwebauthn.AttestationOptions{
					Challenge: options.Challenge,
				},
			)
			parsedCredential, stdErr := protocol.ParseCredentialCreationResponseBytes([]byte(credentialJSON))
			if stdErr != nil {
				return stdErr
			}
			_, wrappedErr = app.Auth.FinishRegisterPasskey(
				sessionData,
				userOb.Username,
				parsedCredential,
				"Test Passkey",
				tx,
				ctx,
				func(userID uuid.UUID, tx *ent.Tx) (*ent.User, error) {
					return userOb, nil
				},
			)
			if wrappedErr != nil {
				return wrappedErr
			}
			return nil
		},
	)
	require.NoError(t, stdErr)
	return userOb, credential, vAuthenticator
}

func TestLoginFlow(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)
	dbClient := app.Database.Client()
	relyingParty := newRelyingParty(app.Env)
	userOb, credential, vAuthenticator := createUserWithCredential(t, true, true, app)

	optionsRecorder := testcommon.Post(
		t, app.Server,
		"/api/v1/users/login/options/",
		nil,
	)
	require.Equal(t, http.StatusOK, optionsRecorder.Code)

	assertionOptions, stdErr := virtualwebauthn.ParseAssertionOptions(optionsRecorder.Body.String())
	require.NoError(t, stdErr)
	require.NotNil(t, assertionOptions)
	require.Equal(t, relyingParty.ID, assertionOptions.RelyingPartyID)
	// The ceremony ID isn't part of the WebAuthn spec so virtualwebauthn doesn't parse it,
	// but the frontend needs to include it in its login/finish request later
	var webAuthnSessionID string
	{
		var optionsResp login.LoginOptionsResponse
		stdErr = json.Unmarshal(optionsRecorder.Body.Bytes(), &optionsResp)
		require.NoError(t, stdErr)
		webAuthnSessionID = optionsResp.WebAuthnSessionID
	}

	// vAuthenticator.FindAllowedCredential doesn't work for discoverable credentials
	require.Len(t, vAuthenticator.Credentials, 1)
	foundCredential := vAuthenticator.Credentials[0]
	require.Equal(t, credential, foundCredential)

	assertionResponse := virtualwebauthn.CreateAssertionResponse(
		relyingParty,
		vAuthenticator,
		foundCredential,
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
			WebAuthnSessionID:           webAuthnSessionID,
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

	sessionCount, stdErr := dbClient.Session.Query().Count(t.Context())
	require.NoError(t, stdErr)
	require.Equal(t, 1, sessionCount)
	sessionOb, stdErr := dbClient.Session.Query().
		Where(
			session.HashedToken(hashedToken[:]),
		).
		Only(t.Context())
	require.NoError(t, stdErr)
	require.Equal(t, userOb.ID, sessionOb.UserID)
}

func TestLoginFlow_InvalidCredential(t *testing.T) {
	t.Parallel()

	runTest := func(t *testing.T, serverAssociatesWithUser bool, authenticatorAssociatesWithUser bool) {
		app := testhelpers.NewApp(t, nil)
		relyingParty := newRelyingParty(app.Env)
		_, credential, vAuthenticator := createUserWithCredential(
			t,
			serverAssociatesWithUser,
			authenticatorAssociatesWithUser,
			app,
		)

		optionsRecorder := testcommon.Post(
			t, app.Server,
			"/api/v1/users/login/options/",
			nil,
		)
		require.Equal(t, http.StatusOK, optionsRecorder.Code)
		// virtualwebauthn.ParseAssertionOptions isn't used because we need the WebAuthSessionID
		// and TestLoginFlow already has coverage to ensure it's parsable by authenticators like that
		var optionsResp login.LoginOptionsResponse
		stdErr := json.Unmarshal(optionsRecorder.Body.Bytes(), &optionsResp)
		require.NoError(t, stdErr)

		// Valid signature but from an unknown credential
		assertionResponse := virtualwebauthn.CreateAssertionResponse(
			relyingParty,
			vAuthenticator,
			credential,
			virtualwebauthn.AssertionOptions{
				Challenge: optionsResp.PublicKey.Challenge,
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
				WebAuthnSessionID:           optionsResp.WebAuthnSessionID,
			},
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

	t.Run("InvalidCredentialForUser", func(t *testing.T) {
		t.Parallel()
		runTest(t, false, true)
	})
	t.Run("CredentialForNonExistentUser", func(t *testing.T) {
		t.Parallel()
		runTest(t, true, false)
	})
}

func TestLoginFlow_GivenExpiredSession_RejectsValidSignature(t *testing.T) {
	t.Parallel()

	env := testcommon.DefaultEnv()
	// go-webauthn doesn't allow us to mock the time
	env.WEBAUTHN_SESSION_TIMEOUT = 250 * time.Millisecond
	app := testhelpers.NewApp(t, &testhelpers.AppOptions{
		Env: env,
	})
	relyingParty := newRelyingParty(app.Env)
	_, credential, vAuthenticator := createUserWithCredential(t, true, true, app)

	optionsRecorder := testcommon.Post(
		t, app.Server,
		"/api/v1/users/login/options/",
		nil,
	)
	require.Equal(t, http.StatusOK, optionsRecorder.Code)
	// virtualwebauthn.ParseAssertionOptions isn't used because we need the WebAuthSessionID
	// and TestLoginFlow already has coverage to ensure it's parsable by authenticators like that
	var optionsResp login.LoginOptionsResponse
	stdErr := json.Unmarshal(optionsRecorder.Body.Bytes(), &optionsResp)
	require.NoError(t, stdErr)

	assertionResponse := virtualwebauthn.CreateAssertionResponse(
		relyingParty,
		vAuthenticator,
		credential,
		virtualwebauthn.AssertionOptions{
			Challenge: optionsResp.PublicKey.Challenge,
		},
	)

	time.Sleep(env.WEBAUTHN_SESSION_TIMEOUT)

	var parsedAssertion protocol.CredentialAssertionResponse
	stdErr = json.Unmarshal([]byte(assertionResponse), &parsedAssertion)
	require.NoError(t, stdErr)

	finishRecorder := testcommon.Post(
		t, app.Server,
		"/api/v1/users/login/finish/",
		login.LoginFinishPayload{
			CredentialAssertionResponse: parsedAssertion,
			WebAuthnSessionID:           optionsResp.WebAuthnSessionID,
		},
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
