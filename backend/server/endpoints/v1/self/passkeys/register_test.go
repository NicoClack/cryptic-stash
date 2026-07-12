package passkeys_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/NicoClack/cryptic-stash/backend/auth"
	"github.com/NicoClack/cryptic-stash/backend/common/testcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent/passkey"
	"github.com/NicoClack/cryptic-stash/backend/server/endpoints/v1/self/passkeys"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/NicoClack/cryptic-stash/backend/testhelpers"
	"github.com/descope/virtualwebauthn"
	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestRegisterFlow(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)
	dbClient := app.Database.Client()
	userOb := testcommon.NewDummyUser(1, dbClient, t.Context(), app.Clock)
	passkeyOb := createDummyPasskey(t, t.Context(), userOb, dbClient)
	sessionToken := createSuperSession(t, app, passkeyOb)

	startResp := testcommon.Post(
		t, app.Server,
		"/api/v1/self/passkeys/register/start/",
		nil,
		testcommon.WithBearerToken(sessionToken),
	)
	require.Equal(t, http.StatusOK, startResp.Code)

	var startRespBody passkeys.RegisterStartResponse
	stdErr := json.Unmarshal(startResp.Body.Bytes(), &startRespBody)
	require.NoError(t, stdErr)
	require.Empty(t, startRespBody.Errors)
	require.Len(t, startRespBody.PublicKey.Challenge, 32)
	require.NotEqual(t, uuid.Nil, startRespBody.WebAuthnSessionID)
	require.Equal(
		t,
		[]protocol.CredentialDescriptor{
			passkeyOb.Credential.Descriptor(),
			// ^ Shouldn't be able to reregister the existing passkey
		},
		startRespBody.PublicKey.CredentialExcludeList,
	)

	vAuthenticator := virtualwebauthn.NewAuthenticator()
	vAuthenticator.Options.UserHandle = userOb.ID[:]
	vAuthenticator.Options.Transports = []virtualwebauthn.Transport{
		virtualwebauthn.TransportUSB,
		virtualwebauthn.TransportNFC,
	}
	credential := virtualwebauthn.NewCredential(virtualwebauthn.KeyTypeEC2)
	vAuthenticator.AddCredential(credential)

	credentialJSON := virtualwebauthn.CreateAttestationResponse(
		testcommon.NewWebAuthnRelyingParty(app.Env),
		vAuthenticator,
		credential,
		virtualwebauthn.AttestationOptions{
			Challenge: startRespBody.PublicKey.Challenge,
		},
	)

	var credentialResponse protocol.CredentialCreationResponse
	stdErr = json.Unmarshal([]byte(credentialJSON), &credentialResponse)
	require.NoError(t, stdErr)

	finishResp := testcommon.Post(
		t, app.Server,
		"/api/v1/self/passkeys/register/finish/",
		passkeys.RegisterFinishPayload{
			CredentialCreationResponse: credentialResponse,
			WebAuthnSessionID:          startRespBody.WebAuthnSessionID,
			Name:                       "Test Passkey 2",
			AllowSuperUser:             true,
			IsSecondGroup:              false,
		},
		testcommon.WithBearerToken(sessionToken),
	)
	require.Equal(t, http.StatusCreated, finishResp.Code)

	passkeyOb = dbClient.Passkey.Query().
		Where(passkey.CredentialID(credential.ID)).
		OnlyX(t.Context())
	require.Equal(t, "Test Passkey 2", passkeyOb.Name)
	require.True(t, passkeyOb.AllowSuperUser)
	require.False(t, passkeyOb.IsSecondGroup)
}

func TestRegisterFlow_DuplicateName(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)
	dbClient := app.Database.Client()
	userOb := testcommon.NewDummyUser(1, dbClient, t.Context(), app.Clock)
	passkeyOb := createDummyPasskey(t, t.Context(), userOb, dbClient)
	sessionToken := createSuperSession(t, app, passkeyOb)

	startResp := testcommon.Post(
		t, app.Server,
		"/api/v1/self/passkeys/register/start/",
		nil,
		testcommon.WithBearerToken(sessionToken),
	)
	require.Equal(t, http.StatusOK, startResp.Code)

	var startRespBody passkeys.RegisterStartResponse
	stdErr := json.Unmarshal(startResp.Body.Bytes(), &startRespBody)
	require.NoError(t, stdErr)

	vAuthenticator := virtualwebauthn.NewAuthenticator()
	vAuthenticator.Options.UserHandle = userOb.ID[:]
	vAuthenticator.Options.Transports = []virtualwebauthn.Transport{
		virtualwebauthn.TransportUSB,
		virtualwebauthn.TransportNFC,
	}
	credential := virtualwebauthn.NewCredential(virtualwebauthn.KeyTypeEC2)
	vAuthenticator.AddCredential(credential)
	credentialJSON := virtualwebauthn.CreateAttestationResponse(
		testcommon.NewWebAuthnRelyingParty(app.Env),
		vAuthenticator,
		credential,
		virtualwebauthn.AttestationOptions{
			Challenge: startRespBody.PublicKey.Challenge,
		},
	)

	var credentialResponse protocol.CredentialCreationResponse
	stdErr = json.Unmarshal([]byte(credentialJSON), &credentialResponse)
	require.NoError(t, stdErr)

	finishResp := testcommon.Post(
		t, app.Server,
		"/api/v1/self/passkeys/register/finish/",
		passkeys.RegisterFinishPayload{
			CredentialCreationResponse: credentialResponse,
			WebAuthnSessionID:          startRespBody.WebAuthnSessionID,
			Name:                       passkeyOb.Name, // The first passkey
			AllowSuperUser:             false,
			IsSecondGroup:              false,
		},
		testcommon.WithBearerToken(sessionToken),
	)

	testcommon.AssertJSONResponse(
		t, finishResp,
		http.StatusConflict,
		gin.H{
			"errors": []servercommon.ErrorDetail{
				{
					Message: "passkey name already exists",
					Code:    "DUPLICATE_PASSKEY_NAME",
				},
			},
		},
	)

	// The WebAuthn session should still be usable
	var sessionData *webauthn.SessionData
	require.True(
		t,
		app.TempKeyValue.Get(auth.WebAuthnSessionStoreName, startRespBody.WebAuthnSessionID.String(), &sessionData),
	)

	passkeyCount := dbClient.Passkey.Query().
		CountX(t.Context())
	require.Equal(t, 1, passkeyCount)
}
