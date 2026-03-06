package setup

import (
	"net/http"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/gin-gonic/gin"
)

type GenerateAdminEnvVarsPayload struct {
	Password string `binding:"required,min=8,max=256" json:"password"` // #nosec G117
}

type GenerateAdminEnvVarsResponse struct {
	Errors  []servercommon.ErrorDetail `binding:"required" json:"errors"`
	EnvVars *common.AdminAuthEnvVars   `                   json:"envVars"`
	TotpURL string                     `                   json:"totpUrl"`
}

func GenerateAdminEnvVars(app *servercommon.ServerApp) gin.HandlerFunc {
	return servercommon.NewHandler(func(ginCtx *gin.Context) error {
		body := GenerateAdminEnvVarsPayload{}
		if ctxErr := servercommon.ParseBody(&body, ginCtx); ctxErr != nil {
			return ctxErr
		}

		envVars, totpURL, wrappedErr := app.Setup.GenerateAdminSetupConstants(body.Password)
		if wrappedErr != nil {
			return wrappedErr
		}

		ginCtx.JSON(http.StatusOK, GenerateAdminEnvVarsResponse{
			Errors:  []servercommon.ErrorDetail{},
			EnvVars: envVars,
			TotpURL: totpURL,
		})
		return nil
	})
}
