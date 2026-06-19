package invites_test

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common/testcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent/user"
	"github.com/NicoClack/cryptic-stash/backend/server/endpoints/v1/invites"
	"github.com/NicoClack/cryptic-stash/backend/testhelpers"
	"github.com/descope/virtualwebauthn"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/require"
)

func TestInviteFlow(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)
	email := "integration-test@example.com"
	expiresAt := app.Clock.Now().Add(time.Hour).UTC()
	inviteOb, code := createInvite(t, app, email, expiresAt)

	respRecorder := testcommon.Get(
		t, app.Server,
		fmt.Sprintf("/api/v1/invites/%s", inviteOb.ID),
		testcommon.WithBearerToken(code),
	)
	require.Equal(t, http.StatusOK, respRecorder.Code)
	var inviteResp invites.GetInviteResponse
	stdErr := json.Unmarshal(respRecorder.Body.Bytes(), &inviteResp)
	require.NoError(t, stdErr)
	require.Equal(t, email, inviteResp.Email)
	require.Equal(t, expiresAt, inviteResp.ExpiresAt)

	respRecorder = testcommon.Post(
		t, app.Server,
		fmt.Sprintf("/api/v1/invites/%s/generate-options", inviteOb.ID),
		nil,
		testcommon.WithBearerToken(code),
	)
	require.Equal(t, http.StatusOK, respRecorder.Code)
	var optionsResp invites.GenerateOptionsResponse
	stdErr = json.Unmarshal(respRecorder.Body.Bytes(), &optionsResp)
	require.NoError(t, stdErr)
	userID := testcommon.MustDecodeRawURLBase64UUID(t, optionsResp.PublicKey.User.ID.(string))

	vAuthenticator := virtualwebauthn.NewAuthenticator()
	vAuthenticator.Options.UserHandle = userID[:]
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
			Challenge: optionsResp.PublicKey.Challenge,
		},
	)

	var createUserPayload invites.CreateUserPayload
	stdErr = json.Unmarshal([]byte(credentialJSON), &createUserPayload)
	require.NoError(t, stdErr)
	createUserPayload.CredentialName = "Test Passkey"

	respRecorder = testcommon.Post(
		t, app.Server,
		fmt.Sprintf("/api/v1/invites/%s/create-user", inviteOb.ID),
		createUserPayload,
		testcommon.WithBearerToken(code),
	)
	require.Equal(t, http.StatusCreated, respRecorder.Code)

	var createUserResp invites.CreateUserResponse
	stdErr = json.Unmarshal(respRecorder.Body.Bytes(), &createUserResp)
	require.NoError(t, stdErr)
	require.Equal(t, userID, createUserResp.UserID)
	require.Len(t, createUserResp.Token, 43) // 32 bytes base64 encoded
	// TODO: assert that the session is superuser mode

	userOb := app.Database.Client().User.Query().
		Where(user.Username(email)).
		WithPasskeys().
		WithStashes().
		WithInvite().
		OnlyX(t.Context())
	require.Equal(t, userID, userOb.ID) // The ID is determined when the options are generated earlier
	require.Len(t, userOb.Edges.Passkeys, 1)
	passkeyOb := userOb.Edges.Passkeys[0]
	require.Equal(t, "Test Passkey", passkeyOb.Name)
	require.Equal(t, credential.ID, passkeyOb.CredentialID)
	require.Equal(t, []protocol.AuthenticatorTransport{protocol.USB, protocol.NFC}, passkeyOb.Credential.Transport)
	require.False(t, passkeyOb.Credential.Flags.BackupEligible)
	require.False(t, passkeyOb.Credential.Flags.BackupState)

	require.NotNil(t, userOb.Edges.Stashes)
	require.Empty(t, userOb.Edges.Stashes)

	require.Equal(t, userID, userOb.Edges.Invite.UserID)
	require.Empty(t, userOb.Edges.Invite.ExpiredReason)
	require.Nil(t, userOb.Edges.Invite.WebAuthnSession) // This data isn't needed anymore

	// Invite should now be unusable
	respRecorder = testcommon.Get(
		t, app.Server,
		fmt.Sprintf("/api/v1/invites/%s", inviteOb.ID),
		testcommon.WithBearerToken(code),
	)
	require.Equal(t, http.StatusUnauthorized, respRecorder.Code)
	respRecorder = testcommon.Post(
		t, app.Server,
		fmt.Sprintf("/api/v1/invites/%s/generate-options", inviteOb.ID),
		nil,
		testcommon.WithBearerToken(code),
	)
	require.Equal(t, http.StatusUnauthorized, respRecorder.Code)
	respRecorder = testcommon.Post(
		t, app.Server,
		fmt.Sprintf("/api/v1/invites/%s/create-user", inviteOb.ID),
		createUserPayload, // Reuse the old payload
		testcommon.WithBearerToken(code),
	)
	require.Equal(t, http.StatusUnauthorized, respRecorder.Code)
}

func TestInviteFlow_SyncablePasskey(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)
	email := "syncable@example.com"
	expiresAt := app.Clock.Now().Add(time.Hour).UTC()
	inviteOb, code := createInvite(t, app, email, expiresAt)

	respRecorder := testcommon.Post(
		t, app.Server,
		fmt.Sprintf("/api/v1/invites/%s/generate-options", inviteOb.ID),
		nil,
		testcommon.WithBearerToken(code),
	)
	require.Equal(t, http.StatusOK, respRecorder.Code)
	var optionsResp invites.GenerateOptionsResponse
	stdErr := json.Unmarshal(respRecorder.Body.Bytes(), &optionsResp)
	require.NoError(t, stdErr)
	userID := testcommon.MustDecodeRawURLBase64UUID(t, optionsResp.PublicKey.User.ID.(string))

	vAuthenticator := virtualwebauthn.NewAuthenticator()
	vAuthenticator.Options.UserHandle = userID[:]
	vAuthenticator.Options.BackupEligible = true
	vAuthenticator.Options.BackupState = true
	vAuthenticator.Options.Transports = []virtualwebauthn.Transport{virtualwebauthn.TransportInternal}

	credential := virtualwebauthn.NewCredential(virtualwebauthn.KeyTypeEC2)
	vAuthenticator.AddCredential(credential)

	credentialJSON := virtualwebauthn.CreateAttestationResponse(
		testcommon.NewWebAuthnRelyingParty(app.Env),
		vAuthenticator,
		credential,
		virtualwebauthn.AttestationOptions{
			Challenge: optionsResp.PublicKey.Challenge,
		},
	)

	var createUserPayload invites.CreateUserPayload
	stdErr = json.Unmarshal([]byte(credentialJSON), &createUserPayload)
	require.NoError(t, stdErr)
	createUserPayload.CredentialName = "Syncable Key"

	respRecorder = testcommon.Post(
		t, app.Server,
		fmt.Sprintf("/api/v1/invites/%s/create-user", inviteOb.ID),
		createUserPayload,
		testcommon.WithBearerToken(code),
	)
	require.Equal(t, http.StatusCreated, respRecorder.Code)

	userOb := app.Database.Client().User.Query().
		Where(user.Username(email)).
		WithPasskeys().
		OnlyX(t.Context())
	require.Len(t, userOb.Edges.Passkeys, 1)
	passkeyOb := userOb.Edges.Passkeys[0]
	require.True(t, passkeyOb.Credential.Flags.BackupEligible)
	require.True(t, passkeyOb.Credential.Flags.BackupState)
	require.Len(t, passkeyOb.Credential.Transport, 1)
	require.Equal(t, protocol.Internal, passkeyOb.Credential.Transport[0])
}

func TestInviteFlow_ExpiredInvite(t *testing.T) {
	t.Parallel()

	clock := clockwork.NewFakeClock()
	app := testhelpers.NewApp(t, &testhelpers.AppOptions{Clock: clock})
	dbClient := app.Database.Client()
	email := "expired-invite@example.com"
	expiresAt := clock.Now().Add(time.Hour).UTC()
	inviteOb, code := createInvite(t, app, email, expiresAt)

	respRecorder := testcommon.Get(
		t, app.Server,
		fmt.Sprintf("/api/v1/invites/%s", inviteOb.ID),
		testcommon.WithBearerToken(code),
	)
	require.Equal(t, http.StatusOK, respRecorder.Code)

	// Multiple calls are allowed, the WebAuthn session is updated each time
	var lastWebAuthnSession *webauthn.SessionData
	for range 2 {
		respRecorder = testcommon.Post(
			t, app.Server,
			fmt.Sprintf("/api/v1/invites/%s/generate-options", inviteOb.ID),
			nil,
			testcommon.WithBearerToken(code),
		)
		require.Equal(t, http.StatusOK, respRecorder.Code)

		inviteOb = dbClient.Invite.GetX(t.Context(), inviteOb.ID)
		require.NotNil(t, inviteOb.WebAuthnSession)
		require.NotEqual(t, lastWebAuthnSession, inviteOb.WebAuthnSession)
		lastWebAuthnSession = inviteOb.WebAuthnSession
	}

	clock.Advance(2 * time.Hour)

	// Expired
	respRecorder = testcommon.Get(
		t, app.Server,
		fmt.Sprintf("/api/v1/invites/%s", inviteOb.ID),
		testcommon.WithBearerToken(code),
	)
	require.Equal(t, http.StatusUnauthorized, respRecorder.Code)
	respRecorder = testcommon.Post(
		t, app.Server,
		fmt.Sprintf("/api/v1/invites/%s/generate-options", inviteOb.ID),
		nil,
		testcommon.WithBearerToken(code),
	)
	require.Equal(t, http.StatusUnauthorized, respRecorder.Code)

	// This information isn't too sensitive so it'll just get deleted along with the invite once
	// that's got too old. The user email in that is more valuable information than this
	// TODO: implement that ^
	inviteOb = dbClient.Invite.GetX(t.Context(), inviteOb.ID)
	require.Equal(t, lastWebAuthnSession, inviteOb.WebAuthnSession)

	var createUserPayload invites.CreateUserPayload
	{
		lastChallenge, stdErr := base64.RawURLEncoding.DecodeString(lastWebAuthnSession.Challenge)
		require.NoError(t, stdErr)

		vAuthenticator := virtualwebauthn.NewAuthenticator()
		credential := virtualwebauthn.NewCredential(virtualwebauthn.KeyTypeEC2)
		attestationJSON := virtualwebauthn.CreateAttestationResponse(
			testcommon.NewWebAuthnRelyingParty(app.Env),
			vAuthenticator,
			credential,
			virtualwebauthn.AttestationOptions{
				Challenge: lastChallenge,
			},
		)

		stdErr = json.Unmarshal([]byte(attestationJSON), &createUserPayload)
		require.NoError(t, stdErr)
		createUserPayload.CredentialName = "Passkey that signed last challenge"
	}

	// Should fail because the invite has expired, but it would have succeeded if it was sent earlier
	respRecorder = testcommon.Post(
		t, app.Server,
		fmt.Sprintf("/api/v1/invites/%s/create-user", inviteOb.ID),
		createUserPayload,
		testcommon.WithBearerToken(code),
	)
	require.Equal(t, http.StatusUnauthorized, respRecorder.Code)

	// Shouldn't have been cleared
	inviteOb = dbClient.Invite.GetX(t.Context(), inviteOb.ID)
	require.Equal(t, lastWebAuthnSession, inviteOb.WebAuthnSession)
}
