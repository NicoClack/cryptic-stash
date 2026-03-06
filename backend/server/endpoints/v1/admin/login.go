package admin

import (
	"context"
	"net/http"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/common/dbcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/ent/user"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type LoginPayload struct {
	Password string `binding:"required,min=1"         json:"password"` // #nosec G117
	TotpCode string `binding:"required,len=6,numeric" json:"totpCode"`
}

type LoginResponse struct {
	Errors      []servercommon.ErrorDetail `binding:"required" json:"errors"`
	AdminCode   string                     `                   json:"adminCode"`
	AdminUserID string                     `                   json:"adminUserId"`
}

func Login(app *servercommon.ServerApp) gin.HandlerFunc {
	return servercommon.NewHandler(func(ginCtx *gin.Context) error {
		body := LoginPayload{}
		if ctxErr := servercommon.ParseBody(&body, ginCtx); ctxErr != nil {
			return ctxErr
		}

		adminCode, isValid := app.Core.GetAdminCode(body.Password, body.TotpCode)
		if !isValid {
			return servercommon.NewUnauthorizedError()
		}
		adminUserID, stdErr := dbcommon.WithReadTx(
			ginCtx.Request.Context(), app.Database,
			func(tx *ent.Tx, ctx context.Context) (uuid.UUID, error) {
				return tx.User.Query().
					Where(user.Username(common.AdminUsername)).
					OnlyID(ctx)
			},
		)
		if stdErr != nil {
			return stdErr
		}

		ginCtx.JSON(http.StatusOK, LoginResponse{
			Errors:      []servercommon.ErrorDetail{},
			AdminCode:   adminCode,
			AdminUserID: adminUserID.String(),
		})
		return nil
	})
}
