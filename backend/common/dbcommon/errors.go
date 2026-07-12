package dbcommon

import (
	"errors"
	"strings"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/ent"
)

const (
	ErrTypeWithTx = "WithTx"
	// Lower level
	ErrTypeStartTx  = "start transaction"
	ErrTypeCommitTx = "commit transaction"
	ErrTypeCallback = "callback"
)

// Not used by this package. Return this error when you need to cancel the transaction and don't have an error
var ErrCancelTransaction = common.NewErrorWithCategories("cancel transaction")

var ErrWrapperStartTx = common.NewErrorWrapper(common.ErrTypeDbCommon, ErrTypeStartTx).
	SetChild(ErrWrapperDatabase)
var ErrWrapperCommitTx = common.NewErrorWrapper(common.ErrTypeDbCommon, ErrTypeCommitTx).
	SetChild(ErrWrapperDatabase)
var ErrWrapperCallback = common.NewErrorWrapper(common.ErrTypeDbCommon, ErrTypeCallback)
var ErrWrapperWithTx = common.NewErrorWrapper(common.ErrTypeDbCommon, ErrTypeWithTx)

var ErrWrapperDatabase = common.NewErrorWrapper(common.ErrTypeDbCommon).
	SetChild(common.ErrWrapperDatabase)

var ErrUnexpectedTransaction = ErrWrapperStartTx.Wrap(
	ErrWrapperDatabase.Wrap(
		errors.New("found transaction in context. nested transactions are not supported"),
	),
)

func IsUniqueConstraintError(stdErr error, tableNames ...string) bool {
	constraintErr, ok := errors.AsType[*ent.ConstraintError](stdErr)
	if !ok {
		return false
	}
	unwrapped := constraintErr.Unwrap()
	if unwrapped == nil {
		// This shouldn't happen, but best to avoid a panic here
		return false
	}
	msg := unwrapped.Error()
	if !strings.Contains(msg, "UNIQUE constraint failed") {
		return false
	}
	if len(tableNames) == 0 {
		return true
	}
	for _, tableName := range tableNames {
		if strings.Contains(msg, tableName) {
			return true
		}
	}
	return false
}
