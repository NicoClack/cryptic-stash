package services

import (
	"context"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/ent/invite"
	"github.com/NicoClack/cryptic-stash/backend/invites"
)

type Invites struct {
	app *common.App
}

func NewInvites(app *common.App) *Invites {
	return &Invites{
		app: app,
	}
}

func (service *Invites) DeleteExpiredInvites(tx *ent.Tx, ctx context.Context) common.WrappedError {
	_, stdErr := tx.Invite.Delete().
		Where(
			invite.ExpiresAtLTE(service.app.Clock.Now()),
			invite.UserIDIsNil(),
		).
		Exec(ctx)
	if stdErr != nil {
		return invites.ErrWrapperDeleteExpiredInvites.Wrap(
			invites.ErrWrapperDatabase.Wrap(stdErr),
		)
	}
	return nil
}
