package users_test

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/NicoClack/cryptic-stash/backend/server/endpoints/v1/users"
	"github.com/NicoClack/cryptic-stash/backend/testhelpers"
	"github.com/stretchr/testify/require"
)

func TestAuthTest_AllowsValidSession(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)
	client := app.TestDatabase.Client()

	userOb, stdErr := client.User.Create().
		SetUsername("alice").
		SetCreatedAt(app.Clock.Now()).
		SetUpdatedAt(app.Clock.Now()).
		Save(context.Background())
	require.NoError(t, stdErr)

	sessionToken := "session-token-for-tests"
	hashedToken := sha256.Sum256([]byte(sessionToken))
	sessionOb, stdErr := client.Session.Create().
		SetCreatedAt(app.Clock.Now()).
		SetUpdatedAt(app.Clock.Now()).
		SetUser(userOb).
		SetHashedToken(hashedToken[:]).
		SetExpiresAt(app.Clock.Now().Add(app.Env.SESSION_DURATION)).
		SetUserAgent("test-agent").
		SetIP("127.0.0.1").
		Save(context.Background())
	require.NoError(t, stdErr)

	request := httptest.NewRequest(http.MethodGet, "/auth-test/", nil)
	request.Header.Set("Authorization", "Session "+sessionToken)
	responseRecorder := httptest.NewRecorder()
	app.Server.ServeHTTP(responseRecorder, request)

	require.Equal(t, http.StatusOK, responseRecorder.Code)

	var responseBody users.AuthTestResponse
	stdErr = json.Unmarshal(responseRecorder.Body.Bytes(), &responseBody)
	require.NoError(t, stdErr)
	require.Equal(t, sessionOb.ID, responseBody.SessionID)
	require.Equal(t, userOb.ID, responseBody.UserID)
	require.Equal(t, userOb.Username, responseBody.Username)
}
