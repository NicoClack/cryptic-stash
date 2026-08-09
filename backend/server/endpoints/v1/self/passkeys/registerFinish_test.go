package passkeys_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/NicoClack/cryptic-stash/backend/common/dbcommon"
	"github.com/NicoClack/cryptic-stash/backend/common/testcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/server/endpoints/v1/self/passkeys"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/NicoClack/cryptic-stash/backend/testhelpers"
	"github.com/descope/virtualwebauthn"
	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestRegisterFinish_SessionNotSudo_SendsForbidden(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)
	dbClient := app.Database.Client()
	userOb := testcommon.NewDummyUser(1, dbClient, t.Context(), app.Clock)
	passkeyOb := createPasskey(t, "Test passkey", false, false, userOb.ID, dbClient)

	sessionToken, stdErr := dbcommon.WithReadWriteTx(
		t.Context(),
		app.Database,
		func(tx *ent.Tx, ctx context.Context) (string, error) {
			sessionOb, token, wrappedErr := app.Auth.CreateSession(
				false, // Not sudo
				userOb.ID,
				passkeyOb.ID,
				"Mozilla/5.0",
				"127.0.0.1",
				tx,
				ctx,
			)
			if wrappedErr != nil {
				return "", wrappedErr
			}
			require.NotNil(t, sessionOb)

			return base64.RawURLEncoding.EncodeToString(token), nil
		},
	)
	require.NoError(t, stdErr)

	finishResp := testcommon.Post(
		t, app.Server,
		"/api/v1/self/passkeys/register/finish/",
		passkeys.RegisterFinishPayload{
			CredentialCreationResponse: protocol.CredentialCreationResponse{},
			WebAuthnSessionID:          uuid.New(),
			Name:                       "Test passkey 2",
			AllowSudo:                  false,
			IsSecondGroup:              false,
		},
		testcommon.WithBearerToken(sessionToken),
	)

	testcommon.AssertJSONResponse(
		t, finishResp,
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

func TestRegisterFinish_InvalidWebAuthnSession_SendsBadRequest(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)
	userOb := testcommon.NewDummyUser(1, app.Database.Client(), t.Context(), app.Clock)
	passkeyOb := createPasskey(t, "Test passkey", false, false, userOb.ID, app.Database.Client())
	sessionToken := createSession(t, true, passkeyOb.UserID, passkeyOb.ID, app)

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

	var credentialResponse protocol.CredentialCreationResponse
	stdErr := json.Unmarshal([]byte(credentialJSON), &credentialResponse)
	require.NoError(t, stdErr)

	finishResp := testcommon.Post(
		t, app.Server,
		"/api/v1/self/passkeys/register/finish/",
		passkeys.RegisterFinishPayload{
			CredentialCreationResponse: credentialResponse,
			WebAuthnSessionID:          uuid.New(), // Non-existent
			Name:                       "Test",
			AllowSudo:                  false,
			IsSecondGroup:              false,
		},
		testcommon.WithBearerToken(sessionToken),
	)

	testcommon.AssertJSONResponse(
		t, finishResp,
		http.StatusBadRequest,
		gin.H{
			"errors": []servercommon.ErrorDetail{
				{
					Message: "WebAuthnSessionID: missing or expired",
					Code:    "INVALID_WEBAUTHN_SESSION",
				},
			},
		},
	)
}
