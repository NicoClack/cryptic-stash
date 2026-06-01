package servercommon

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

func ParseBody(pointer any, ginCtx *gin.Context) *Error {
	stdErr := ginCtx.ShouldBindJSON(pointer)
	if stdErr != nil {
		validationErrs := validator.ValidationErrors{}
		if !errors.As(stdErr, &validationErrs) {
			// We don't know much about these errors and what we can return to the client

			// The uuid package doesn't provide sentinel errors and we don't know the field :(
			if strings.Contains(stdErr.Error(), "UUID") || strings.Contains(stdErr.Error(), "urn") {
				return NewError(ErrWrapperParseBodyJson.Wrap(stdErr)).
					SetStatus(http.StatusBadRequest).
					AddDetail(ErrorDetail{
						Message: "malformed UUID in JSON body",
						Code:    "MALFORMED_BODY_JSON_UUID",
					}).
					DisableLogging()
			}

			return NewError(ErrWrapperParseBodyJson.Wrap(stdErr)).
				SetStatus(http.StatusBadRequest).
				AddDetail(ErrorDetail{
					Message: "unknown error in JSON body",
					Code:    "UNKNOWN_BODY_JSON_ERROR",
				}).
				EnableLogging()
		}

		var builder strings.Builder
		for _, validationErr := range validationErrs {
			// TODO: these errors have incorrect casing:
			// TotpCode: condition failed: required
			fmt.Fprintf(&builder, "%v: condition failed: %v", validationErr.Field(), validationErr.Tag())
		}

		return NewError(ErrWrapperParseBodyJson.Wrap(stdErr)).
			SetStatus(http.StatusBadRequest).
			AddDetail(ErrorDetail{
				Message: builder.String(),
				Code:    "MALFORMED_BODY_JSON",
			}).
			DisableLogging()
	}
	return nil
}
