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

func Get(t *testing.T, server common.ServerService, url string, options ...RequestOption) *httptest.ResponseRecorder {
	t.Helper()

	req, stdErr := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
	require.NoError(t, stdErr)

	for _, option := range options {
		option(req)
	}

	respRecorder := httptest.NewRecorder()
	server.ServeHTTP(respRecorder, req)
	return respRecorder
}

func Post(
	t *testing.T,
	server common.ServerService,
	url string,
	body any,
	options ...RequestOption,
) *httptest.ResponseRecorder {
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
	req.Header.Set("User-Agent", "test-agent")
	req.Header.Set("test-proxy-original-ip", "127.0.0.1")

	for _, option := range options {
		option(req)
	}

	respRecorder := httptest.NewRecorder()
	server.ServeHTTP(respRecorder, req)
	return respRecorder
}

type RequestOption func(*http.Request)

func WithBearerToken(token string) RequestOption {
	return func(req *http.Request) {
		req.Header.Set("Authorization", "Bearer "+token)
	}
}
