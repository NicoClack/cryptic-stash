package common

import (
	"context"
	"errors"
	"slices"
	"time"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

var ErrWrapperDatabase = NewDynamicErrorWrapper(func(err error) WrappedError {
	wrappedErr := WrapErrorWithCategories(err)
	if wrappedErr == nil {
		return nil
	}

	if errors.Is(err, context.DeadlineExceeded) {
		wrappedErr.AddCategoriesMut(ErrTypeTimeout, ErrTypeDatabase)
		return wrappedErr
	}
	sqliteErr := &sqlite.Error{}
	if errors.As(err, &sqliteErr) {
		// Code() returns the extended result primaryCode (e.g. "database table is
		// locked (262)" is SQLITE_LOCKED_SHAREDCACHE = SQLITE_LOCKED | (1<<8)).
		// So we use a mask to get the primary primaryCode.
		primaryCode := sqliteErr.Code() & 0xff
		if slices.Index([]int{
			sqlite3.SQLITE_FULL,
			sqlite3.SQLITE_AUTH,
			sqlite3.SQLITE_READONLY,
			sqlite3.SQLITE_BUSY,
			sqlite3.SQLITE_CANTOPEN,
			sqlite3.SQLITE_IOERR,
			sqlite3.SQLITE_LOCKED,
			sqlite3.SQLITE_NOMEM,
		}, primaryCode) != -1 {
			wrappedErr.ConfigureRetriesMut(10, 50*time.Millisecond, 2)
			if primaryCode == sqlite3.SQLITE_NOMEM {
				wrappedErr.AddCategoriesMut(ErrTypeMemory, ErrTypeDatabase)
			} else {
				wrappedErr.AddCategoriesMut(ErrTypeDisk, ErrTypeDatabase)
			}
			return wrappedErr
		}
	}

	wrappedErr.AddCategoriesMut(ErrTypeOther, ErrTypeDatabase)
	return wrappedErr
})
var ErrWrapperAPI = NewErrorWrapper(ErrTypeAPI)

var ErrNoTxInContext = NewErrorWithCategories("no db transaction found in context")
var ErrNotImplemented = NewErrorWithCategories("not implemented")

// Helper functions
var ErrWrapperNewPublicJSONSchema = NewErrorWrapper(ErrTypeCommon, "new public json schema")
