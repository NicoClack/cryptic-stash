package middleware

import (
	"net/http"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/gin-gonic/gin"
)

func NewLogger(logger common.Logger) gin.HandlerFunc {
	return func(ginCtx *gin.Context) {
		ginCtx.Set(servercommon.LoggerKey, logger.With(
			// TODO: I think gin might already be providing this through the context?
			"url", ginCtx.Request.URL,
			"method", ginCtx.Request.Method,
		))

		ginCtx.Next()
	}
}

// TODO: handle panics and stop the default error handler from being registered
func NewError() gin.HandlerFunc {
	return func(ginCtx *gin.Context) {
		logger := servercommon.GetLogger(ginCtx)
		ginCtx.Next()

		statusCode := -1
		mergedDetails := []servercommon.ErrorDetail{}
		for _, ginError := range ginCtx.Errors {
			serverErr := servercommon.NewError(ginError.Err)
			if serverErr.Status() != -1 {
				if statusCode == -1 {
					statusCode = serverErr.Status()
				} else {
					logger.Warn(
						"server errors have different status codes",
						"previousStatusCode", statusCode,
						"newStatusCode", serverErr.Status(),
						"errors", ginCtx.Errors,
					)
					statusCode = http.StatusInternalServerError
				}
			}
			mergedDetails = append(mergedDetails, serverErr.Details()...)

			if serverErr.ShouldLog() {
				logger := logger.With("error", serverErr, "statusCode", statusCode)
				if statusCode >= 500 || statusCode == -1 {
					logger.Error("an internal server error occurred")
				} else {
					logger.Info("a HTTP 4xx was returned to a client")
				}
			}
		}

		if len(ginCtx.Errors) > 0 {
			if statusCode == -1 {
				statusCode = http.StatusInternalServerError
			}
			if ginCtx.Writer.Written() {
				logger.Warn(
					"couldn't write status from serverErr to response because the response has already been written",
					"statusCode", statusCode,
					"existingStatusCode", ginCtx.Writer.Status(),
				)
			} else {
				ginCtx.JSON(statusCode, gin.H{
					"errors": mergedDetails,
				})
			}
		}
	}
}
