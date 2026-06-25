package services

import (
	"context"
	"crypto/subtle"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/ent/twofactoraction"
	"github.com/NicoClack/cryptic-stash/backend/twofactoractions"
	"github.com/google/uuid"
)

type TwoFactorActions struct {
	app *common.App
}

func NewTwoFactorActions(app *common.App) *TwoFactorActions {
	return &TwoFactorActions{
		app: app,
	}
}

func (service *TwoFactorActions) Create(
	versionedType string,
	expiresAt time.Time,
	body any,
	ctx context.Context,
) (*ent.TwoFactorAction, string, common.WrappedError) {
	encoded, wrappedErr := service.app.Jobs.Encode(
		versionedType,
		body,
	)
	if wrappedErr != nil {
		return nil, "", twofactoractions.ErrWrapperCreate.Wrap(wrappedErr)
	}
	jobType, version, wrappedErr := common.ParseVersionedType(versionedType)
	if wrappedErr != nil { // This shouldn't happen because of the Encode call but just in case
		return nil, "", twofactoractions.ErrWrapperCreate.Wrap(wrappedErr)
	}

	tx := ent.TxFromContext(ctx)
	if tx == nil {
		return nil, "", twofactoractions.ErrWrapperCreate.Wrap(
			twofactoractions.ErrNoTxInContext,
		)
	}
	code := common.CryptoRandomAlphaNum(twofactoractions.CODE_LENGTH)
	action, stdErr := tx.TwoFactorAction.Create().
		SetType(jobType).
		SetVersion(version).
		SetBody(encoded).
		SetExpiresAt(expiresAt).
		SetCode(code).Save(ctx)
	if stdErr != nil {
		return nil, code, twofactoractions.ErrWrapperCreate.Wrap(
			twofactoractions.ErrWrapperDatabase.Wrap(stdErr),
		)
	}

	return action, code, nil
}
func (service *TwoFactorActions) Confirm(
	actionID uuid.UUID, code string,
	ctx context.Context,
) (*ent.Job, common.WrappedError) {
	tx := ent.TxFromContext(ctx)
	if tx == nil {
		return nil, twofactoractions.ErrWrapperConfirm.Wrap(
			twofactoractions.ErrNoTxInContext,
		)
	}
	action, stdErr := tx.TwoFactorAction.Get(ctx, actionID)
	if stdErr != nil {
		return nil, twofactoractions.ErrWrapperConfirm.Wrap(
			twofactoractions.ErrNotFound,
		)
	}
	if subtle.ConstantTimeCompare([]byte(code), []byte(action.Code)) == 0 {
		return nil, twofactoractions.ErrWrapperConfirm.Wrap(
			twofactoractions.ErrWrongCode,
		)
	}

	stdErr = tx.TwoFactorAction.DeleteOne(action).Exec(ctx)
	if stdErr != nil {
		return nil, twofactoractions.ErrWrapperConfirm.Wrap(
			twofactoractions.ErrWrapperDatabase.Wrap(stdErr),
		)
	}

	if action.ExpiresAt.Before(service.app.Clock.Now()) {
		return nil, twofactoractions.ErrWrapperConfirm.Wrap(
			twofactoractions.ErrExpired,
		)
	}

	job, wrappedErr := service.app.Jobs.EnqueueEncoded(
		common.GetVersionedType(action.Type, action.Version),
		action.Body,
		ctx,
	)
	if wrappedErr != nil {
		return nil, twofactoractions.ErrWrapperConfirm.Wrap(wrappedErr)
	}
	return job, nil
}
func (service *TwoFactorActions) DeleteExpiredActions(ctx context.Context) common.WrappedError {
	tx := ent.TxFromContext(ctx)
	if tx == nil {
		return twofactoractions.ErrWrapperDeleteExpiredActions.Wrap(
			twofactoractions.ErrNoTxInContext,
		)
	}

	_, stdErr := tx.TwoFactorAction.Delete().
		Where(twofactoraction.ExpiresAtLTE(service.app.Clock.Now())).
		Exec(ctx)
	if stdErr != nil {
		return twofactoractions.ErrWrapperDeleteExpiredActions.Wrap(
			twofactoractions.ErrWrapperDatabase.Wrap(stdErr),
		)
	}
	return nil
}
