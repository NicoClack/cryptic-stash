package auth

import "github.com/NicoClack/cryptic-stash/backend/common"

const (
	ErrTypeStartRegisterPasskey  = "start register passkey"
	ErrTypeFinishRegisterPasskey = "finish register passkey"
	ErrTypeStartLogin            = "start login"
	ErrTypeFinishLogin           = "finish login"
	ErrTypeCreateSession         = "create session"
	ErrTypeValidateSession       = "validate session"
)

var ErrInvalidWebAuthnSessionID = common.NewErrorWithCategories(
	"invalid WebAuthn session ID",
	common.ErrTypeAuth, common.ErrTypeClient,
)

// TODO: make something like ErrInvalidCredential which isn't WebAuthn specific
var ErrWebAuthnSessionExpired = common.NewErrorWithCategories(
	"WebAuthn session expired",
	common.ErrTypeAuth, common.ErrTypeClient,
)
var ErrInvalidAAGUIDLength = common.NewErrorWithCategories(
	"AAGUID must be 16 bytes",
	common.ErrTypeAuth, common.ErrTypeClient,
)
var ErrInvalidSession = common.NewErrorWithCategories(
	"invalid session",
	common.ErrTypeAuth, common.ErrTypeClient,
)

var ErrWrapperStartRegisterPasskey = common.NewErrorWrapper(common.ErrTypeAuth, ErrTypeStartRegisterPasskey)
var ErrWrapperFinishRegisterPasskey = common.NewErrorWrapper(common.ErrTypeAuth, ErrTypeFinishRegisterPasskey)

var ErrWrapperStartLogin = common.NewErrorWrapper(common.ErrTypeAuth, ErrTypeStartLogin)
var ErrWrapperFinishLogin = common.NewErrorWrapper(common.ErrTypeAuth, ErrTypeFinishLogin)

var ErrWrapperCreateSession = common.NewErrorWrapper(common.ErrTypeAuth, ErrTypeCreateSession)
var ErrWrapperValidateSession = common.NewErrorWrapper(common.ErrTypeAuth, ErrTypeValidateSession)

var ErrWrapperDatabase = common.NewErrorWrapper(common.ErrTypeAuth).SetChild(common.ErrWrapperDatabase)
