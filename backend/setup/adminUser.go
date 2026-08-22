package setup

import (
	"context"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/ent/user"
	"github.com/google/uuid"
)

func CheckAdminHasMessengers(
	ctx context.Context,
	adminID uuid.UUID,
	messengers common.MessengerService,
) (bool, common.WrappedError) {
	tx := ent.TxFromContext(ctx)
	if tx == nil {
		return false, ErrWrapperCheckAdminHasMessengers.Wrap(common.ErrNoTxInContext)
	}

	userOb, stdErr := tx.User.Query().
		Where(user.ID(adminID)).
		WithMessengers().
		Only(ctx)
	if stdErr != nil {
		return false, ErrWrapperCheckAdminHasMessengers.Wrap(
			ErrWrapperDatabase.Wrap(stdErr),
		)
	}

	messengerTypes := messengers.GetConfiguredMessengerTypes(userOb)
	return len(messengerTypes) > 0, nil
}
