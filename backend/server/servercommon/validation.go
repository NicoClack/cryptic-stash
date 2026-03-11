package servercommon

import (
	"regexp"

	"github.com/NicoClack/cryptic-stash/backend/common"
)

var usernamePattern = regexp.MustCompile(`^[a-z0-9_-]+$`)

func ValidateUsername(username string) *Error {
	if !usernamePattern.MatchString(username) || username == common.AdminUsername {
		return NewInvalidUsernameError()
	}
	return nil
}
