package mocks

import (
	"context"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/google/uuid"
)

type EmptyTwoFactorActionService struct{}

func NewEmptyTwoFactorActionService() *EmptyTwoFactorActionService {
	return &EmptyTwoFactorActionService{}
}

func (m *EmptyTwoFactorActionService) Create(
	versionedType string,
	expiresAt time.Time,
	body any,
	ctx context.Context,
) (*ent.TwoFactorAction, string, common.WrappedError) {
	return nil, "", nil
}
func (m *EmptyTwoFactorActionService) Confirm(
	actionID uuid.UUID,
	code string,
	ctx context.Context,
) (*ent.Job, common.WrappedError) {
	return nil, nil
}
func (m *EmptyTwoFactorActionService) DeleteExpiredActions(ctx context.Context) common.WrappedError {
	return nil
}
