package login_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/NicoClack/cryptic-stash/backend/server/endpoints/v1/users/login"
	"github.com/NicoClack/cryptic-stash/backend/testhelpers"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/stretchr/testify/require"
)

func TestLoginOptions_CreatesLoginSession(t *testing.T) {
	t.Parallel()

	app := testhelpers.NewApp(t, nil)

	request := httptest.NewRequest(http.MethodGet, "/login/options/", nil)
	responseRecorder := httptest.NewRecorder()
	app.Server.ServeHTTP(responseRecorder, request)

	require.Equal(t, http.StatusOK, responseRecorder.Code)

	var responseBody login.LoginOptionsResponse
	stdErr := json.Unmarshal(responseRecorder.Body.Bytes(), &responseBody)
	require.NoError(t, stdErr)
	require.NotEmpty(t, responseBody.SessionID)

	var storedSessionData webauthn.SessionData
	require.True(t, app.TempKeyValue.Get("loginWebAuthnSession", responseBody.SessionID, &storedSessionData))
}
