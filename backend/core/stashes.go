package core

import (
	"context"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/ent/downloadsession"
	"github.com/NicoClack/cryptic-stash/backend/ent/stash"
	"github.com/google/uuid"
	"github.com/jonboulle/clockwork"
)

func InvalidateDownloadSessionsForStash(
	stashID uuid.UUID,
	ctx context.Context,
	clock clockwork.Clock,
) common.WrappedError {
	tx := ent.TxFromContext(ctx)
	if tx == nil {
		return ErrWrapperInvalidateDownloadSessionsForStash.Wrap(common.ErrNoTxInContext)
	}

	_, stdErr := tx.DownloadSession.Delete().
		Where(downloadsession.HasStashWith(stash.ID(stashID))).
		Exec(ctx)
	if stdErr != nil {
		return ErrWrapperInvalidateDownloadSessionsForStash.Wrap(
			ErrWrapperDatabase.Wrap(stdErr),
		)
	}
	now := clock.Now()
	stdErr = tx.Stash.UpdateOneID(stashID).
		SetUpdatedAt(now).
		SetDownloadSessionsValidFrom(now).
		Exec(ctx)
	if stdErr != nil {
		return ErrWrapperInvalidateDownloadSessionsForStash.Wrap(
			ErrWrapperDatabase.Wrap(stdErr),
		)
	}
	return nil
}

func IsStashLocked(stashOb *ent.Stash, clock clockwork.Clock) bool {
	if stashOb.IsAdminLocked || stashOb.IsSelfLocked {
		return true
	}
	if stashOb.SelfLockedUntil == nil {
		return false
	}
	return clock.Now().Before(*stashOb.SelfLockedUntil)
}
