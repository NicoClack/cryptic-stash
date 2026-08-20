package mocks

import (
	"context"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/ent"
)

type EmptyInviteService struct{}

func NewEmptyInviteService() *EmptyInviteService {
	return &EmptyInviteService{}
}

func (m *EmptyInviteService) DeleteExpiredInvites(tx *ent.Tx, ctx context.Context) common.WrappedError {
	return nil
}
