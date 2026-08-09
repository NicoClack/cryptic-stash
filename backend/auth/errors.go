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

	ErrTypeRenamePasskey       = "rename passkey"
	ErrTypeSetPasskeyAllowSudo = "set passkey allow sudo"
	ErrTypeMovePasskeyGroup    = "move passkey group"
	ErrTypeDeletePasskey       = "delete passkey"
	ErrTypeDisableTwoGroupAuth = "disable two group auth"
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
var ErrNoSudoEligiblePasskeys = common.NewErrorWithCategories(
	"no passkeys are eligible for sudo mode",
	common.ErrTypeAuth, common.ErrTypeClient,
)
var ErrNeitherPasskeySudoEligible = common.NewErrorWithCategories(
	"neither passkey is eligible for sudo mode",
	common.ErrTypeAuth, common.ErrTypeClient,
)
var ErrSessionAlreadyElevated = common.NewErrorWithCategories(
	"session is already in sudo mode",
	common.ErrTypeAuth, common.ErrTypeClient,
)
var ErrWebAuthnUserNotFound = common.NewErrorWithCategories(
	"no user found for WebAuthn user handle",
	common.ErrTypeAuth, common.ErrTypeClient,
)
var ErrPasskeySudoConstraint = common.NewErrorWithCategories(
	"can't disable sudo on this passkey, as doing so would remove sudo access from your session",
	common.ErrTypeAuth, common.ErrTypeClient,
)
var ErrPasskeyGroupMoveConstraint = common.NewErrorWithCategories(
	"can't move this passkey, as doing so may lock you out of sudo mode. to enable two group auth, "+
		"use two different passkeys in your session and move one of them to the second group. "+
		"to move back, first log in with different passkeys or disable two group auth entirely",
	common.ErrTypeAuth,
	common.ErrTypeClient,
)
var ErrPasskeyDeleteConstraint = common.NewErrorWithCategories(
	"can't delete a passkey that is currently in use by your session",
	common.ErrTypeAuth, common.ErrTypeClient,
)
var ErrPasskeyNotFound = common.NewErrorWithCategories(
	"passkey not found",
	common.ErrTypeAuth, common.ErrTypeClient,
)
var ErrUnauthorizedToModifyUser = common.NewErrorWithCategories(
	"unauthorized to modify this user",
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
var ErrWrapperGetEligiblePasskeysForSudo = common.NewErrorWrapper(common.ErrTypeAuth, ErrTypeGetEligiblePasskeys)
var ErrWrapperElevateSession = common.NewErrorWrapper(common.ErrTypeAuth, ErrTypeElevateSession)

var ErrWrapperGetUserCallback = common.NewErrorWrapper(common.ErrTypeAuth, ErrTypeGetUserCallback)
var ErrWrapperInternalGetUser = common.NewErrorWrapper(common.ErrTypeAuth, ErrTypeInternalGetUser)
var ErrWrapperDatabase = common.NewErrorWrapper(common.ErrTypeAuth).SetChild(common.ErrWrapperDatabase)

var ErrWrapperRenamePasskey = common.NewErrorWrapper(common.ErrTypeAuth, ErrTypeRenamePasskey)
var ErrWrapperSetPasskeyAllowSudo = common.NewErrorWrapper(common.ErrTypeAuth, ErrTypeSetPasskeyAllowSudo)
var ErrWrapperMovePasskeyGroup = common.NewErrorWrapper(common.ErrTypeAuth, ErrTypeMovePasskeyGroup)
var ErrWrapperDeletePasskey = common.NewErrorWrapper(common.ErrTypeAuth, ErrTypeDeletePasskey)
var ErrWrapperDisableTwoGroupAuth = common.NewErrorWrapper(common.ErrTypeAuth, ErrTypeDisableTwoGroupAuth)
