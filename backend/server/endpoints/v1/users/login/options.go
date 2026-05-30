package login

import (
	"net/http"

	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/protocol"
)

type LoginOptionsResponse struct {
	Errors            []servercommon.ErrorDetail                 `json:"errors"`
	WebAuthnSessionID string                                     `json:"webAuthnSessionID"`
	PublicKey         protocol.PublicKeyCredentialRequestOptions `json:"publicKey"`
}

func LoginOptions(app *servercommon.ServerApp) gin.HandlerFunc {
	return servercommon.NewHandler(func(ginCtx *gin.Context) error {
		sessionID, options, wrappedErr := app.Auth.StartLogin(ginCtx.Request.Context())
		if wrappedErr != nil {
			return wrappedErr
		}

		ginCtx.JSON(http.StatusOK, &LoginOptionsResponse{
			Errors:            []servercommon.ErrorDetail{},
			WebAuthnSessionID: sessionID,
			PublicKey:         options,
		})
		return nil
	})
}
