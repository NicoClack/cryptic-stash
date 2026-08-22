package middleware

import (
	"context"
	"net/http"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/common/dbcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/gin-gonic/gin"
)

func NewAdminProtected(
	core common.CoreService,
	db common.DatabaseService,
) gin.HandlerFunc {
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

		adminUserOb, stdErr := dbcommon.WithReadTx(
			ginCtx.Request.Context(), db,
			func(tx *ent.Tx, ctx context.Context) (*ent.User, error) {
				return tx.User.Get(ctx, core.AdminID())
			},
		)
		if stdErr != nil {
			ginCtx.Error(servercommon.NewError(stdErr))
			ginCtx.Abort()
			return
		}

		ginCtx.Set(userContextKey, adminUserOb) // TODO: will endpoints ever need more than an ID?
		ginCtx.Set(actorContextKey, &common.Actor{
			UserID:    adminUserOb.ID,
			IP:        ginCtx.ClientIP(),
			UserAgent: ginCtx.Request.UserAgent(),
		})
		ginCtx.Next()
	}
}
