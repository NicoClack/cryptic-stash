package setup

import (
	"context"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/google/uuid"
)

func GetStatus(
	ctx context.Context,
	adminID uuid.UUID,
	messengers common.MessengerService,
	env *common.Env,
) (*common.SetupStatus, common.WrappedError) {
	status := &common.SetupStatus{}
	if env.ENABLE_ENV_SETUP {
		return status, nil
	}
	status.IsEnvComplete = true

	hasMessengers, wrappedErr := CheckAdminHasMessengers(ctx, adminID, messengers)
	if wrappedErr != nil {
		return status, ErrWrapperGetStatus.Wrap(
			wrappedErr,
		)
	}
	if !hasMessengers {
		return status, nil
	}
	status.AreAdminMessengersConfigured = true

	status.IsComplete = true
	return status, nil
}
