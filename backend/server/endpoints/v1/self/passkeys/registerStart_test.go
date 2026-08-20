package passkeys_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/NicoClack/cryptic-stash/backend/common/testcommon"
	"github.com/NicoClack/cryptic-stash/backend/server/endpoints/v1/self/passkeys"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/NicoClack/cryptic-stash/backend/testhelpers"
	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestRegisterStart(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)
	userOb := testcommon.NewDummyUser(1, app.Database.Client(), t.Context(), app.Clock)
	passkeyOb := createPasskey(t, "Test passkey", false, false, userOb.ID, app.Database.Client())
	sessionToken := createSession(t, userOb.ID, passkeyOb.ID, new(passkeyOb.ID), app)

	respRecorder := testcommon.Post(
		t, app.Server,
		"/api/v1/self/passkeys/register/start/",
		nil,
		testcommon.WithBearerToken(sessionToken),
	)

	require.Equal(t, http.StatusOK, respRecorder.Code)

	var resp passkeys.RegisterStartResponse
	stdErr := json.Unmarshal(respRecorder.Body.Bytes(), &resp)
	require.NoError(t, stdErr)
	require.Empty(t, resp.Errors)
	require.Len(t, resp.PublicKey.Challenge, 32)
	require.NotEqual(t, uuid.Nil, resp.WebAuthnSessionID)
	require.Equal(
		t,
		[]protocol.CredentialDescriptor{
			passkeyOb.Credential.Descriptor(),
			// ^ Shouldn't be able to reregister the existing passkey
		},
		resp.PublicKey.CredentialExcludeList,
	)
}
func TestRegisterStart_NoAuthorizationHeader(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)

	respRecorder := testcommon.Post(
		t, app.Server,
		"/api/v1/self/passkeys/register/start/",
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
func TestRegisterStart_SessionNotSudo_SendsForbidden(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)
	dbClient := app.Database.Client()
	userOb := testcommon.NewDummyUser(1, dbClient, t.Context(), app.Clock)
	passkeyOb := createPasskey(t, "Test passkey", false, false, userOb.ID, dbClient)
	sessionToken := createSession(t, userOb.ID, passkeyOb.ID, nil, app)

	respRecorder := testcommon.Post(
		t, app.Server,
		"/api/v1/self/passkeys/register/start/",
		nil,
		testcommon.WithBearerToken(sessionToken),
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
