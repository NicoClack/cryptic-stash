package invites

import (
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestGetInviteURLUsesFragment(t *testing.T) {
	t.Parallel()

	frontendBaseURL, stdErr := url.Parse("https://example.com/app/")
	require.NoError(t, stdErr)
	inviteID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	inviteCode := "abc123_def-456"
	inviteURL := getInviteURL(inviteID, inviteCode, frontendBaseURL)

	parsedURL, stdErr := url.Parse(inviteURL)
	require.NoError(t, stdErr)

	require.Equal(t, "https", parsedURL.Scheme)
	require.Equal(t, "example.com", parsedURL.Host)
	require.Equal(t, "/invites/123e4567-e89b-12d3-a456-426614174000/", parsedURL.Path)
	require.Empty(t, parsedURL.RawQuery)
	require.Equal(t, inviteCode, parsedURL.Fragment)
}
