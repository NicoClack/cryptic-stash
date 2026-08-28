package invites

import "github.com/NicoClack/cryptic-stash/backend/common"

const (
	ErrTypeCreateInvite         = "create invite"
	ErrTypeGetInvite            = "get invite"
	ErrTypeGenerateOptions      = "generate invite registration options"
	ErrTypeCreateUser           = "create user from invite"
	ErrTypeDeleteExpiredInvites = "delete expired invites"
)

var ErrWrapperCreateInvite = common.NewErrorWrapper(common.ErrTypeInvites, ErrTypeCreateInvite)
var ErrWrapperGetInvite = common.NewErrorWrapper(common.ErrTypeInvites, ErrTypeGetInvite)
var ErrWrapperGenerateOptions = common.NewErrorWrapper(common.ErrTypeInvites, ErrTypeGenerateOptions)
var ErrWrapperCreateUser = common.NewErrorWrapper(common.ErrTypeInvites, ErrTypeCreateUser)
var ErrWrapperDeleteExpiredInvites = common.NewErrorWrapper(common.ErrTypeInvites, ErrTypeDeleteExpiredInvites)

var ErrWrapperDatabase = common.NewErrorWrapper(common.ErrTypeInvites).SetChild(common.ErrWrapperDatabase)

var ErrInviteNotFound = common.NewErrorWithCategories(
	"invite not found",
	common.ErrTypeInvites, common.ErrTypeClient,
)
var ErrInviteUsed = common.NewErrorWithCategories(
	"invite already used",
	common.ErrTypeInvites, common.ErrTypeClient,
)
var ErrInviteExpired = common.NewErrorWithCategories(
	"invite expired",
	common.ErrTypeInvites, common.ErrTypeClient,
)
var ErrUsernameTaken = common.NewErrorWithCategories(
	"username already taken",
	common.ErrTypeInvites, common.ErrTypeClient,
)
var ErrNoWebAuthnSession = common.NewErrorWithCategories(
	"no active WebAuthn session",
	common.ErrTypeInvites, common.ErrTypeClient,
)
