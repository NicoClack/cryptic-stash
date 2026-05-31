package login_test

// Tests that span both endpoints should go in login_test.go

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/common/testcommon"
	"github.com/NicoClack/cryptic-stash/backend/server/endpoints/v1/users/login"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/NicoClack/cryptic-stash/backend/testhelpers"
	"github.com/descope/virtualwebauthn"
	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestLoginFinish_MissingSessionID_ReturnsBadRequest(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)

	finishRecorder := testcommon.Post(
		t, app.Server,
		"/api/v1/users/login/finish/",
		login.LoginFinishPayload{},
	)

	testcommon.AssertJSONResponse(
		t, finishRecorder,
		http.StatusBadRequest,
		gin.H{
			"errors": []servercommon.ErrorDetail{
				{
					Message: "WebAuthnSessionID: condition failed: required",
					Code:    "INVALID_BODY_JSON",
				},
			},
		},
	)
}

func TestLoginFinish_InvalidSessionID_ReturnsBadRequest(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)

	finishRecorder := testcommon.Post(
		t, app.Server,
		"/api/v1/users/login/finish/",
		gin.H{
			"webAuthnSessionId": strings.Repeat("a", 36), // Right length but not the format of a UUID
		},
	)

	testcommon.AssertJSONResponse(
		t, finishRecorder,
		http.StatusBadRequest,
		gin.H{
			"errors": []servercommon.ErrorDetail{
				{
					Message: "WebAuthnSessionID: condition failed: uuid",
					Code:    "INVALID_BODY_JSON",
				},
			},
		},
	)
}

func TestLoginFinish_MissingSession_ReturnsBadRequest(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)

	var parsedAssertion protocol.CredentialAssertionResponse
	{
		vAuthenticator := virtualwebauthn.NewAuthenticator()
		credential := virtualwebauthn.NewCredential(virtualwebauthn.KeyTypeEC2)
		assertionJSON := virtualwebauthn.CreateAssertionResponse(
			newRelyingParty(app.Env),
			vAuthenticator,
			credential,
			virtualwebauthn.AssertionOptions{Challenge: common.CryptoRandomBytes(32)},
		)
		stdErr := json.Unmarshal([]byte(assertionJSON), &parsedAssertion)
		require.NoError(t, stdErr)
	}

	finishRecorder := testcommon.Post(
		t, app.Server,
		"/api/v1/users/login/finish/",
		login.LoginFinishPayload{
			WebAuthnSessionID:           uuid.New(),
			CredentialAssertionResponse: parsedAssertion,
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
