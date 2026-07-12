package auth

import "github.com/NicoClack/cryptic-stash/backend/common"

const (
	ErrTypeStartRegisterPasskey  = "start register passkey"
	ErrTypeFinishRegisterPasskey = "finish register passkey"
	ErrTypeStartLogin            = "start login"
	ErrTypeFinishLogin           = "finish login"
	ErrTypeStartElevation        = "start elevation"
	ErrTypeFinishElevation       = "finish elevation"

	ErrTypeValidateLogin       = "validate login"
	ErrTypeCreateSession       = "create session"
	ErrTypeValidateSession     = "validate session"
	ErrTypeGetEligiblePasskeys = "get eligible passkeys"
	ErrTypeElevateSession      = "elevate session"

	ErrTypeGetUserCallback = "get user callback"
	// FinishLogin has to read the user in a callback provided to go-webauthn
	ErrTypeInternalGetUser = "internal get user"
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
var ErrNoSuperEligiblePasskeys = common.NewErrorWithCategories(
	"no passkeys are eligible for superuser mode",
	common.ErrTypeAuth, common.ErrTypeClient,
)
var ErrNeitherPasskeySuperEligible = common.NewErrorWithCategories(
	"neither passkey is eligible for superuser mode",
	common.ErrTypeAuth, common.ErrTypeClient,
)
var ErrSessionAlreadyElevated = common.NewErrorWithCategories(
	"session is already in superuser mode",
	common.ErrTypeAuth, common.ErrTypeClient,
)

var ErrWrapperStartRegisterPasskey = common.NewErrorWrapper(common.ErrTypeAuth, ErrTypeStartRegisterPasskey)
var ErrWrapperFinishRegisterPasskey = common.NewErrorWrapper(common.ErrTypeAuth, ErrTypeFinishRegisterPasskey)

var ErrWrapperStartLogin = common.NewErrorWrapper(common.ErrTypeAuth, ErrTypeStartLogin)
var ErrWrapperFinishLogin = common.NewErrorWrapper(common.ErrTypeAuth, ErrTypeFinishLogin)
var ErrWrapperStartElevation = common.NewErrorWrapper(common.ErrTypeAuth, ErrTypeStartElevation)
var ErrWrapperFinishElevation = common.NewErrorWrapper(common.ErrTypeAuth, ErrTypeFinishElevation)

var ErrWrapperValidateLogin = common.NewErrorWrapper(common.ErrTypeAuth, ErrTypeValidateLogin)
var ErrWrapperCreateSession = common.NewErrorWrapper(common.ErrTypeAuth, ErrTypeCreateSession)
var ErrWrapperValidateSession = common.NewErrorWrapper(common.ErrTypeAuth, ErrTypeValidateSession)
var ErrWrapperGetEligiblePasskeys = common.NewErrorWrapper(common.ErrTypeAuth, ErrTypeGetEligiblePasskeys)
var ErrWrapperElevateSession = common.NewErrorWrapper(common.ErrTypeAuth, ErrTypeElevateSession)

var ErrWrapperGetUserCallback = common.NewErrorWrapper(common.ErrTypeAuth, ErrTypeGetUserCallback)
var ErrWrapperInternalGetUser = common.NewErrorWrapper(common.ErrTypeAuth, ErrTypeInternalGetUser)
var ErrWrapperDatabase = common.NewErrorWrapper(common.ErrTypeAuth).SetChild(common.ErrWrapperDatabase)
