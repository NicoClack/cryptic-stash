package servercommon

import (
	"github.com/NicoClack/cryptic-stash/backend/common"
)

// Should be used in combination with the "email" validation tag
func ValidateUserEmail(email string) *Error {
	if email == common.AdminUsername {
		return NewInvalidUserEmailError()
	}
	return nil
}
