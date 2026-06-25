package invites_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common/testcommon"
	"github.com/NicoClack/cryptic-stash/backend/server/endpoints/v1/invites"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/NicoClack/cryptic-stash/backend/testhelpers"
	"github.com/descope/virtualwebauthn"
	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/stretchr/testify/require"
)

// TODO: create tests for the whole flow. Assert that the passkey and session are superuser

func TestCreateUser_NoWebAuthnSession(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)
	inviteOb, code := createInvite(t, app, "test@example.com", app.Clock.Now().Add(time.Hour))

	// The WebAuthn session is normally created by the generate options endpoint, which isn't called here
	createUserPayload := invites.CreateUserPayload{
		CredentialName: "Test Key",
		CredentialCreationResponse: protocol.CredentialCreationResponse{
			AttestationResponse: protocol.AuthenticatorAttestationResponse{
				AuthenticatorResponse: protocol.AuthenticatorResponse{
					ClientDataJSON: protocol.URLEncodedBase64([]byte("{}")),
				},
			},
		},
	}
	respRecorder := testcommon.Post(
		t, app.Server,
		fmt.Sprintf("/api/v1/invites/%s/create-user", inviteOb.ID),
		createUserPayload,
		testcommon.WithBearerToken(code),
	)
	testcommon.AssertJSONResponse(
		t, respRecorder,
		http.StatusBadRequest,
		gin.H{
			"errors": []servercommon.ErrorDetail{
				{
					Message: "invalid WebAuthn credential",
					Code:    "INVALID_CREDENTIAL",
				},
			},
		},
	)
}

func TestCreateUser_UsernameTaken(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)
	dbClient := app.Database.Client()
	email := "taken@example.com"

	dbClient.User.Create().
		SetUsername(email).
		SetCreatedAt(app.Clock.Now()).
		SetUpdatedAt(app.Clock.Now()).
		SaveX(t.Context())
	inviteOb, code := createInvite(t, app, email, app.Clock.Now().Add(time.Hour))

	vAuthenticator := virtualwebauthn.NewAuthenticator()
	credential := virtualwebauthn.NewCredential(virtualwebauthn.KeyTypeEC2)
	vAuthenticator.AddCredential(credential)
	credentialJSON := virtualwebauthn.CreateAttestationResponse(
		testcommon.NewWebAuthnRelyingParty(app.Env),
		vAuthenticator,
		credential,
		virtualwebauthn.AttestationOptions{
			Challenge: []byte("12345"),
		},
	)

	var createUserPayload invites.CreateUserPayload
	stdErr := json.Unmarshal([]byte(credentialJSON), &createUserPayload)
	require.NoError(t, stdErr)
	createUserPayload.CredentialName = "Test Key"

	respRecorder := testcommon.Post(
		t, app.Server,
		fmt.Sprintf("/api/v1/invites/%s/create-user", inviteOb.ID),
		createUserPayload,
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
