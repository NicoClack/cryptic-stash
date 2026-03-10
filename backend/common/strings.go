package common

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	ErrTypeParseVersionedType = "parse versioned type"
)

var ErrMalformedVersionedType = NewErrorWithCategories(
	"malformed versioned type",
)
var ErrWrapperParseVersionedType = NewErrorWrapper(ErrTypeParseVersionedType)

func GetStringBetween(str string, start string, end string) string {
	startIndex := strings.Index(str, start)
	if startIndex == -1 {
		return ""
	}
	startIndex += len(start)
	endIndex := strings.Index(str[startIndex:], end)
	if endIndex == -1 {
		return ""
	}
	return str[startIndex : startIndex+endIndex]
}

func GetVersionedType(id string, version int) string {
	return fmt.Sprintf("%v_%v", id, version)
}
func ParseVersionedType(versionedType string) (string, int, WrappedError) {
	separatorIndex := strings.LastIndex(versionedType, "_")
	if separatorIndex == -1 {
		return "", 0, ErrWrapperParseVersionedType.Wrap(ErrMalformedVersionedType)
	}
	version, err := strconv.Atoi(versionedType[separatorIndex+1:])
	if err != nil {
		return "", 0, ErrWrapperParseVersionedType.Wrap(ErrMalformedVersionedType)
	}

	return versionedType[:separatorIndex], version, nil
}

func JoinPaths(path1, path2 string) string {
	if path1 == "" {
		return path2
	}
	if path2 == "" {
		return path1
	}
	return fmt.Sprintf("%s/%s", path1, path2)
}

func SanitizeFilename(filename string, defaultFilename string) string {
	trimmedFilename := strings.TrimSpace(filename)
	if trimmedFilename == "" {
		return defaultFilename
	}

	var sanitized strings.Builder
	sanitized.Grow(len(trimmedFilename))

	lastWasReplacement := false
	for _, char := range trimmedFilename {
		if char < 32 || char == 127 {
			continue
		}

		if strings.ContainsRune(`<>:"/\\|?*`, char) {
			if !lastWasReplacement {
				sanitized.WriteByte('_')
				lastWasReplacement = true
			}
			continue
		}

		sanitized.WriteRune(char)
		lastWasReplacement = false
	}

	sanitizedFilename := strings.Trim(sanitized.String(), ". ")
	if sanitizedFilename == "" || sanitizedFilename == "." || sanitizedFilename == ".." {
		return defaultFilename
	}

	return sanitizedFilename
}
