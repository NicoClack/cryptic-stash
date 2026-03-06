package setup

import (
	"net/http"

	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/gin-gonic/gin"
)

type CheckTotpPayload struct {
	Code   string `binding:"required,len=6,numeric" json:"code"`
	Secret string `binding:"required,min=1"         json:"secret"` // #nosec G117
}

type CheckTotpResponse struct {
	Errors  []servercommon.ErrorDetail `binding:"required" json:"errors"`
	IsValid bool                       `                   json:"isValid"`
}

func CheckTotp(app *servercommon.ServerApp) gin.HandlerFunc {
	return servercommon.NewHandler(func(ginCtx *gin.Context) error {
		body := CheckTotpPayload{}
		if ctxErr := servercommon.ParseBody(&body, ginCtx); ctxErr != nil {
			return ctxErr
		}

		ginCtx.JSON(http.StatusOK, CheckTotpResponse{
			Errors:  []servercommon.ErrorDetail{},
			IsValid: app.Setup.CheckTotpCode(body.Code, body.Secret),
		})
		return nil
	})
}
