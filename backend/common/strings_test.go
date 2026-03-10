package common_test

import (
	"testing"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/stretchr/testify/require"
)

func TestGetStringBetween(t *testing.T) {
	t.Parallel()

	require.Equal(t, "tag", common.GetStringBetween("test [tag]", "[", "]"))
	require.Empty(t, common.GetStringBetween("test", "[", "]"))
	require.Empty(t, common.GetStringBetween("test [", "[", "]"))
	require.Empty(t, common.GetStringBetween("test []", "[", "]"))

	require.Equal(t, "42", common.GetStringBetween("the meaning of life is 42.", "the meaning of life is ", "."))
}

func TestParseVersionedType_GivenValidVersionedType_ReturnsParsed(t *testing.T) {
	t.Parallel()

	jobType, version, stdErr := common.ParseVersionedType("test_job_type_2")
	require.Nil(t, stdErr)
	require.Equal(t, "test_job_type", jobType)
	require.Equal(t, 2, version)
}

func TestParseVersionedType_GivenUnderscoreSuffix_ReturnsError(t *testing.T) {
	t.Parallel()

	_, _, stdErr := common.ParseVersionedType("test_job_type_")
	require.ErrorIs(t, stdErr, common.ErrMalformedVersionedType) // Shouldn't panic
}

func TestSanitizeFilename_GivenSafeFilename_ReturnsOriginal(t *testing.T) {
	t.Parallel()

	require.Equal(t, "backup-codes.txt", common.SanitizeFilename("backup-codes.txt", ""))
}

func TestSanitizeFilename_GivenReservedCharacters_ReplacesWithUnderscores(t *testing.T) {
	t.Parallel()

	require.Equal(t, "_secret_recovery_.txt", common.SanitizeFilename("../secret\\recovery?.txt", ""))
}

func TestSanitizeFilename_GivenControlCharacters_RemovesThem(t *testing.T) {
	t.Parallel()

	require.Equal(t, "recoverycodes.txt", common.SanitizeFilename("recovery\x00codes\n.txt", ""))
}

func TestSanitizeFilename_GivenOnlyWhitespaceAndDots_ReturnsFallback(t *testing.T) {
	t.Parallel()

	require.Equal(t, "download", common.SanitizeFilename(" .. ", "download"))
}
