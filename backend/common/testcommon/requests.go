package testcommon

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/stretchr/testify/require"
)

func Post(t *testing.T, server common.ServerService, url string, body any) *httptest.ResponseRecorder {
	t.Helper()

	var encodedBody []byte
	if body != nil {
		var stdErr error
		encodedBody, stdErr = json.Marshal(body)
		require.NoError(t, stdErr)
	}

	req, stdErr := http.NewRequestWithContext(t.Context(), http.MethodPost, url, bytes.NewBuffer(encodedBody))
	require.NoError(t, stdErr)
	req.Header.Set("Content-Type", "application/json")

	respRecorder := httptest.NewRecorder()
	server.ServeHTTP(respRecorder, req)
	return respRecorder
}
