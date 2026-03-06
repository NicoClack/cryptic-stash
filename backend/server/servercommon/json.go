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
			return NewError(ErrWrapperParseBodyJson.Wrap(stdErr))
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
				Code:    "INVALID_BODY_JSON",
			}).
			DisableLogging()
	}
	return nil
}
