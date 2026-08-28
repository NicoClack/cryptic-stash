package core

import "github.com/NicoClack/cryptic-stash/backend/common"

const (
	ErrTypeSendActiveDownloadSessionReminders = "send active download session reminders"
	ErrTypeDeleteExpiredDownloadSessions      = "delete expired download sessions"
	ErrTypeInvalidateDownloadSessionsForStash = "invalidate download sessions for stash"
	ErrTypeEncrypt                            = "encrypt"
	ErrTypeDecrypt                            = "decrypt"
	// Lower level
	ErrTypeInvalidData = "invalid data"
)

var ErrWrapperInvalidData = common.NewErrorWrapper(common.ErrTypeCore, ErrTypeInvalidData)
var ErrWrapperCreateCipher = common.NewErrorWrapper(common.ErrTypeCore, ErrTypeInvalidData)

var ErrIncorrectPassword = common.NewErrorWithCategories("incorrect password", common.ErrTypeCore)

var ErrWrapperSendActiveDownloadSessionReminders = common.NewErrorWrapper(
	common.ErrTypeCore,
	ErrTypeSendActiveDownloadSessionReminders,
)

var ErrWrapperDeleteExpiredDownloadSessions = common.NewErrorWrapper(
	common.ErrTypeCore,
	ErrTypeDeleteExpiredDownloadSessions,
)

var ErrWrapperInvalidateDownloadSessionsForStash = common.NewErrorWrapper(
	common.ErrTypeCore,
	ErrTypeInvalidateDownloadSessionsForStash,
)

// These functions don't categorize their errors
var ErrWrapperEncrypt = common.NewErrorWrapper(common.ErrTypeCore, ErrTypeEncrypt)
var ErrWrapperDecrypt = common.NewErrorWrapper(common.ErrTypeCore, ErrTypeDecrypt)

var ErrWrapperDatabase = common.NewErrorWrapper(common.ErrTypeCore).
	SetChild(common.ErrWrapperDatabase)
