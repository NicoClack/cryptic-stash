package superuser_test

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/common/testcommon"
	"github.com/NicoClack/cryptic-stash/backend/server/endpoints/v1/users/superuser"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/NicoClack/cryptic-stash/backend/testhelpers"
	"github.com/descope/virtualwebauthn"
	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestFinishElevation_NoAuthHeader_SendsBadRequest(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)

	respRecorder := testcommon.Post(
		t, app.Server,
		"/api/v1/users/superuser/finish-elevation/",
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
func TestFinishElevation_UnknownSessionToken_SendsUnauthorized(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)

	respRecorder := testcommon.Post(
		t, app.Server,
		"/api/v1/users/superuser/finish-elevation/",
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

func TestFinishElevation_MissingWebAuthnSessionID_SendsBadRequest(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)
	userOb := testcommon.NewDummyUser(1, app.Database.Client(), t.Context(), app.Clock)
	sessionToken, _ := createSessionAndPasskey(t, false, userOb, app)

	respRecorder := testcommon.Post(
		t, app.Server,
		"/api/v1/users/superuser/finish-elevation/",
		superuser.FinishElevationPayload{},
		testcommon.WithBearerToken(sessionToken),
	)

	testcommon.AssertJSONResponse(
		t, respRecorder,
		http.StatusBadRequest,
		gin.H{
			"errors": []servercommon.ErrorDetail{
				{
					Message: "WebAuthnSessionID: condition failed: required",
					Code:    "MALFORMED_BODY_JSON",
				},
			},
		},
	)
}

func TestFinishElevation_UnknownWebAuthnSessionID_SendsBadRequest(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)
	userOb := testcommon.NewDummyUser(1, app.Database.Client(), t.Context(), app.Clock)
	sessionToken, _ := createSessionAndPasskey(t, false, userOb, app)

	var parsedAssertion protocol.CredentialAssertionResponse
	{
		vAuthenticator := virtualwebauthn.NewAuthenticator()
		credential := virtualwebauthn.NewCredential(virtualwebauthn.KeyTypeEC2)
		assertionJSON := virtualwebauthn.CreateAssertionResponse(
			testcommon.NewWebAuthnRelyingParty(app.Env),
			vAuthenticator,
			credential,
			virtualwebauthn.AssertionOptions{Challenge: common.CryptoRandomBytes(32)},
		)
		stdErr := json.Unmarshal([]byte(assertionJSON), &parsedAssertion)
		require.NoError(t, stdErr)
	}

	respRecorder := testcommon.Post(
		t, app.Server,
		"/api/v1/users/superuser/finish-elevation/",
		superuser.FinishElevationPayload{
			WebAuthnSessionID:           uuid.New(),
			CredentialAssertionResponse: parsedAssertion,
		},
		testcommon.WithBearerToken(sessionToken),
	)

	testcommon.AssertJSONResponse(
		t, respRecorder,
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

func TestFinishElevation_MissingWebAuthnSession_SendsBadRequest(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)
	userOb := testcommon.NewDummyUser(1, app.Database.Client(), t.Context(), app.Clock)
	sessionToken, _ := createSessionAndPasskey(t, false, userOb, app)

	var parsedAssertion protocol.CredentialAssertionResponse
	{
		vAuthenticator := virtualwebauthn.NewAuthenticator()
		credential := virtualwebauthn.NewCredential(virtualwebauthn.KeyTypeEC2)
		assertionJSON := virtualwebauthn.CreateAssertionResponse(
			testcommon.NewWebAuthnRelyingParty(app.Env),
			vAuthenticator,
			credential,
			virtualwebauthn.AssertionOptions{Challenge: common.CryptoRandomBytes(32)},
		)
		stdErr := json.Unmarshal([]byte(assertionJSON), &parsedAssertion)
		require.NoError(t, stdErr)
	}

	respRecorder := testcommon.Post(
		t, app.Server,
		"/api/v1/users/superuser/finish-elevation/",
		superuser.FinishElevationPayload{
			WebAuthnSessionID:           uuid.New(),
			CredentialAssertionResponse: parsedAssertion,
		},
		testcommon.WithBearerToken(sessionToken),
	)

	testcommon.AssertJSONResponse(
		t, respRecorder,
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

func TestFinishElevation_MalformedCredentialAssertion_SendsBadRequest(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)
	userOb := testcommon.NewDummyUser(1, app.Database.Client(), t.Context(), app.Clock)
	sessionToken, _ := createSessionAndPasskey(t, false, userOb, app)

	respRecorder := testcommon.Post(
		t, app.Server,
		"/api/v1/users/superuser/finish-elevation/",
		gin.H{
			"webAuthnSessionId": uuid.New(),
			"id":                "definitely-not-base64",
			"rawId":             "also-not-base64",
			"type":              "public-key",
		},
		testcommon.WithBearerToken(sessionToken),
	)

	testcommon.AssertJSONResponse(
		t, respRecorder,
		http.StatusBadRequest,
		gin.H{
			"errors": []servercommon.ErrorDetail{
				{
					Message: "malformed WebAuthn assertion response",
					Code:    "MALFORMED_CREDENTIAL_ASSERTION_RESPONSE",
				},
			},
		},
	)
}
