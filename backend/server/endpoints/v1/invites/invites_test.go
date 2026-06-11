package invites_test

import (
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
	"github.com/stretchr/testify/require"
)

// TODO: make variant to assert that the passkey details like backup eligibility and transports are correctly stored and returned
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
	// TODO: assert session is created and returned

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

	require.NotNil(t, userOb.Edges.Stashes)
	require.Empty(t, userOb.Edges.Stashes)

	require.Equal(t, userID, userOb.Edges.Invite.UserID)
	require.Empty(t, userOb.Edges.Invite.ExpiredReason)
	require.Nil(t, userOb.Edges.Invite.WebAuthnSession) // TODO: why is this passing?

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

// TODO: make variant where it expires
