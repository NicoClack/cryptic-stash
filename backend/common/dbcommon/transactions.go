package dbcommon

import (
	"context"
	"database/sql"
	"errors"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/ent"
)

func WithReadTx[T any](
	ctx context.Context, db common.DatabaseService,
	fn func(tx *ent.Tx, ctx context.Context) (T, error),
) (T, error) {
	return withRetryingTx(ctx, db, db.ReadTx, fn)
}
func WithWriteTx(
	ctx context.Context, db common.DatabaseService,
	fn func(tx *ent.Tx, ctx context.Context) error,
) error {
	_, stdErr := withRetryingTx(ctx, db, db.WriteTx, func(tx *ent.Tx, ctx context.Context) (struct{}, error) {
		return struct{}{}, fn(tx, ctx)
	})
	return stdErr
}
func WithReadWriteTx[T any](
	ctx context.Context, db common.DatabaseService,
	fn func(tx *ent.Tx, ctx context.Context) (T, error),
) (T, error) {
	return withRetryingTx(ctx, db, db.WriteTx, fn)
}

func withRetryingTx[T any](
	ctx context.Context, db common.DatabaseService,
	txCallback func(ctx context.Context) (*ent.Tx, error),
	fn func(tx *ent.Tx, ctx context.Context) (T, error),
) (T, error) {
	var returnValue T
	wrappedErr := common.WithRetries(ctx, common.GetLogger(ctx, db), func() error {
		return withTx(ctx, db, txCallback, func(tx *ent.Tx, ctx context.Context) error {
			var stdErr error
			returnValue, stdErr = fn(tx, ctx)
			return stdErr
		})
	})
	return returnValue, wrappedErr
}
func withTx(
	ctx context.Context, db common.DatabaseService,
	txCallback func(ctx context.Context) (*ent.Tx, error),
	fn func(tx *ent.Tx, ctx context.Context) error,
) error {
	stdErr := ctx.Err()
	if stdErr != nil {
		return ErrWrapperWithTx.Wrap(stdErr)
	}
	if ent.TxFromContext(ctx) != nil {
		return ErrWrapperWithTx.Wrap(ErrUnexpectedTransaction)
	}
	tx, stdErr := txCallback(ctx)
	if stdErr != nil {
		return ErrWrapperWithTx.Wrap(
			ErrWrapperStartTx.Wrap(
				common.ErrWrapperDatabase.Wrap(stdErr),
			),
		)
	}

	var callbackErr error
	defer func() {
		panicValue := recover()
		if panicValue != nil {
			rollbackErr := tx.Rollback()
			if rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
				common.GetLogger(ctx, db).Error(
					"withTx: error rolling back transaction after panic",
					"error", rollbackErr,
					"panicValue", panicValue,
				)
			}
			panic(panicValue)
		}
		if callbackErr != nil {
			rollbackErr := tx.Rollback()
			// Context timeouts and cancellations usually roll back the transaction, but it's best to do it explicitly
			// so we don't accidentally leave the transaction open
			if rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
				common.GetLogger(ctx, db).Error(
					"withTx: error rolling back transaction",
					"error", rollbackErr,
					"originalError", callbackErr,
				)
			}
		}
	}()
	callbackErr = fn(tx, ent.NewTxContext(ctx, tx))
	if callbackErr != nil {
		return ErrWrapperWithTx.Wrap(ErrWrapperCallback.Wrap(
			common.AutoWrapError(callbackErr),
		))
	}
	stdErr = tx.Commit()
	if stdErr != nil {
		return ErrWrapperWithTx.Wrap(
			ErrWrapperCommitTx.Wrap(
				common.ErrWrapperDatabase.Wrap(stdErr),
			),
		)
	}
	return nil
}
