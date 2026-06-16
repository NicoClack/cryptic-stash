package users_test

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/NicoClack/cryptic-stash/backend/ent/schema"
	"github.com/NicoClack/cryptic-stash/backend/server/endpoints/v1/users"
	"github.com/NicoClack/cryptic-stash/backend/testhelpers"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/stretchr/testify/require"
)

// TODO: remove this endpoint and replace this with a more realistic test

func TestAuthTest_AllowsValidSession(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)
	client := app.TestDatabase.Client()

	userOb := client.User.Create().
		SetUsername("alice").
		SetCreatedAt(app.Clock.Now()).
		SetUpdatedAt(app.Clock.Now()).
		SaveX(t.Context())
	passkeyOb := client.Passkey.Create().
		SetCreatedAt(app.Clock.Now()).
		SetUpdatedAt(app.Clock.Now()).
		SetName("test-passkey").
		SetCredentialID([]byte("credential-id")).
		SetCredential(schema.EncryptedCredential{
			EncryptedField: schema.EncryptedField[webauthn.Credential]{
				Decrypted: webauthn.Credential{
					ID:        []byte("credential-id"),
					PublicKey: []byte("public-key"),
				},
				KeyName: "auth_1",
			},
		}).
		SetUser(userOb).
		SaveX(t.Context())

	sessionToken := "session-token-for-tests"
	hashedToken := sha256.Sum256([]byte(sessionToken))
	sessionOb := client.Session.Create().
		SetCreatedAt(app.Clock.Now()).
		SetUpdatedAt(app.Clock.Now()).
		SetUser(userOb).
		SetPasskey(passkeyOb).
		SetHashedToken(hashedToken[:]).
		SetExpiresAt(app.Clock.Now().Add(app.Env.SESSION_DURATION)).
		SetUserAgent(schema.EncryptedField[string]{
			Decrypted: "test-agent",
			KeyName:   "security_pii_logging_1",
		}).
		SetIP(schema.EncryptedField[string]{
			Decrypted: "127.0.0.1",
			KeyName:   "security_pii_logging_1",
		}).
		SaveX(t.Context())

	request := httptest.NewRequest(http.MethodGet, "/api/v1/users/auth-test/", nil)
	request.Header.Set("Authorization", "Session "+base64.RawURLEncoding.EncodeToString([]byte(sessionToken)))
	respRecorder := httptest.NewRecorder()
	app.Server.ServeHTTP(respRecorder, request)

	require.Equal(t, http.StatusOK, respRecorder.Code)

	var responseBody users.AuthTestResponse
	stdErr := json.Unmarshal(respRecorder.Body.Bytes(), &responseBody)
	require.NoError(t, stdErr)
	require.Equal(t, sessionOb.ID, responseBody.SessionID)
	require.Equal(t, userOb.ID, responseBody.UserID)
	require.Equal(t, userOb.Username, responseBody.Username)
}
