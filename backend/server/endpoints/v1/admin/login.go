package admin

import (
	"net/http"

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
	AdminUserID uuid.UUID                  `                   json:"adminUserId"`
}

func Login(app *servercommon.ServerApp) gin.HandlerFunc {
	return servercommon.NewHandler(func(ginCtx *gin.Context) error {
		body := LoginPayload{}
		if serverErr := servercommon.ParseBody(&body, ginCtx); serverErr != nil {
			return serverErr
		}

		adminCode, isValid := app.Core.GetAdminCode(body.Password, body.TotpCode)
		if !isValid {
			return servercommon.NewUnauthorizedError()
		}

		ginCtx.JSON(http.StatusOK, LoginResponse{
			Errors:      []servercommon.ErrorDetail{},
			AdminCode:   adminCode,
			AdminUserID: app.Core.AdminID(),
		})
		return nil
	})
}
