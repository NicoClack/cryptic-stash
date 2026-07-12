package invites

import "github.com/NicoClack/cryptic-stash/backend/common"

const (
	ErrTypeDeleteExpiredInvites = "delete expired invites"
)

var ErrWrapperDeleteExpiredInvites = common.NewErrorWrapper(common.ErrTypeInvites, ErrTypeDeleteExpiredInvites)

var ErrWrapperDatabase = common.NewErrorWrapper(common.ErrTypeInvites).SetChild(common.ErrWrapperDatabase)
