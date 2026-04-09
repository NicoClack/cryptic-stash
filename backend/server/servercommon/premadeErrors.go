package servercommon

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/common/dbcommon"
)

const (
	ErrTypeParseBodyJson = "parse body json"
	ErrTypeParseObjectID = "parse object ID"
	ErrTypeBadRequest    = "bad request"
)

var ErrCancelTransaction = NewError(dbcommon.ErrCancelTransaction).DisableLogging()

var ErrWrapperParseBodyJson = common.NewErrorWrapper(
	common.ErrTypeServerCommon,
	ErrTypeParseBodyJson, common.ErrTypeClient,
)
var ErrWrapperParseObjectID = common.NewErrorWrapper(
	common.ErrTypeServerCommon,
	ErrTypeParseObjectID, common.ErrTypeClient,
)

var ErrUnauthorized = NewError(common.NewErrorWithCategories(
	"unauthorized", common.ErrTypeServerCommon, common.ErrTypeClient,
)).SetStatus(http.StatusUnauthorized)
var ErrNotFound = NewError(common.NewErrorWithCategories(
	"not found", common.ErrTypeServerCommon, common.ErrTypeClient,
)).SetStatus(http.StatusNotFound).DisableLogging()

// Mostly when "admin" is passed to a non-admin endpoint
var ErrInvalidUserEmail = NewError(common.NewErrorWithCategories(
	"invalid user email", common.ErrTypeServerCommon, common.ErrTypeClient,
)).
	SetStatus(http.StatusBadRequest).
	AddDetail(ErrorDetail{
		Code:    "INVALID_EMAIL",
		Message: "Invalid email",
	})

var ErrWrapperBadRequest = common.NewErrorWrapper(common.ErrTypeServerCommon, ErrTypeBadRequest, common.ErrTypeClient)

func NewUnauthorizedError() *Error {
	return ErrUnauthorized.Clone()
}
func NewNotFoundError() *Error {
	return ErrNotFound.Clone()
}
func NewInvalidUserEmailError() *Error {
	return ErrInvalidUserEmail.Clone()
}

func NewBadRequestError(fieldName string, message string, errorCode string) *Error {
	fullMessage := fmt.Sprintf("%v: %v", fieldName, message)
	return NewError(ErrWrapperBadRequest.Wrap(errors.New(fullMessage))).
		SetStatus(http.StatusBadRequest).
		AddDetail(ErrorDetail{
			Message: fullMessage,
			Code:    errorCode,
		}).DisableLogging()
}
