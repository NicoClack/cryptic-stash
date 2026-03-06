// Most service functions shouldn't return errors wrapped with the service package name.
// It only really makes sense if a function introduces its own errors.
// In which case for now those errors should be wrapped with "services [package]"
// and their category (not the underlying package name?).
// The other errors the function returns shouldn't be wrapped with the services package name.

package services

import "github.com/NicoClack/cryptic-stash/backend/common"

const (
	ErrTypeDecrypt = "decrypt"
)

var ErrWrapperDecrypt = common.NewErrorWrapper(common.ErrTypeServices, ErrTypeDecrypt)

var ErrMissingInnerEncryptionNonce = common.NewErrorWithCategories(
	"missing inner encryption nonce",
	common.ErrTypeServices,
)
