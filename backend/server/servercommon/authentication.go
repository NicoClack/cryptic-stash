package servercommon

import (
	"strings"

	"github.com/gin-gonic/gin"
)

func ParseAuthorizationHeader(ginCtx *gin.Context) (string, string, *Error) {
	headerValue := ginCtx.GetHeader("authorization")
	if headerValue == "" {
		return "", "", NewBadRequestError("authorization", "header is required", "MISSING_AUTHORIZATION_HEADER")
	}

	parts := strings.SplitN(headerValue, " ", 2)
	isMalformed := len(parts) != 2
	var scheme, token string
	if !isMalformed {
		scheme = parts[0]
		token = parts[1]
		if scheme == "" || token == "" {
			isMalformed = true
		}
	}
	if isMalformed {
		return "", "", NewBadRequestError(
			"authorization",
			"malformed authorization header",
			"MALFORMED_AUTHORIZATION_HEADER",
		)
	}

	return scheme, token, nil
}
func RequireAuthorizationScheme(expectedScheme string, ginCtx *gin.Context) (string, *Error) {
	scheme, token, serverErr := ParseAuthorizationHeader(ginCtx)
	if serverErr != nil {
		return "", serverErr
	}
	if scheme != expectedScheme {
		return "", NewBadRequestError(
			"authorization",
			"unsupported authorization scheme",
			"UNSUPPORTED_AUTHORIZATION_SCHEME",
		)
	}
	return token, nil
}
