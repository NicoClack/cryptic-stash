package middleware

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
)

func NewTimeout() gin.HandlerFunc {
	return func(ginCtx *gin.Context) {
		// If this times out, the response will be sent by the error handling middleware
		// TODO: seems to result in an error in Railway
		ctx, cancel := context.WithTimeout(ginCtx.Request.Context(), (9*time.Second)+(900*time.Millisecond))
		defer cancel()

		ginCtx.Request = ginCtx.Request.WithContext(ctx)
		ginCtx.Next()
	}
}
