package middleware

import (
	"net/http"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/gin-gonic/gin"
)

func NewAdminProtected(core common.CoreService) gin.HandlerFunc {
	return func(ginCtx *gin.Context) {
		token, serverErr := servercommon.RequireAuthorizationScheme("AdminCode", ginCtx)
		if serverErr != nil {
			ginCtx.AbortWithStatusJSON(serverErr.Status(), gin.H{
				"errors": serverErr.Details(),
			})
			return
		}

		if !core.CheckAdminCode(token) {
			ginCtx.AbortWithStatusJSON(
				http.StatusUnauthorized,
				gin.H{
					"errors": []servercommon.ErrorDetail{
						{
							Message: "invalid admin code",
							Code:    "INVALID_ADMIN_CODE",
						},
					},
				},
			)
			return
		}
		ginCtx.Next()
	}
}
