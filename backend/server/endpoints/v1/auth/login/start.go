package login

import (
	"net/http"

	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/google/uuid"
)

type LoginStartResponse struct {
	Errors            []servercommon.ErrorDetail                 `json:"errors"`
	WebAuthnSessionID uuid.UUID                                  `json:"webAuthnSessionId"`
	PublicKey         protocol.PublicKeyCredentialRequestOptions `json:"publicKey"`
}

func LoginStart(app *servercommon.ServerApp) gin.HandlerFunc {
	// Note: currently username enumeration isn't possible because even the server doesn't know
	// who is logging in when this endpoint is called.
	// If in the future this endpoint accepts an email in order to get a list of AllowedCredentials, we'll need to
	// send some plausible credentials for an invalid username, using a hash of the email
	// with a secret key so the randomness is deterministic.
	// It might make sense to make this behaviour configurable in case an admin has an idea of how many credentials
	// their users might have. Also keep in mind that the credential IDs have a wide range of lengths
	// and those need to be believable too.
	return servercommon.NewHandler(func(ginCtx *gin.Context) error {
		sessionID, options, wrappedErr := app.Auth.StartLogin(ginCtx.Request.Context())
		if wrappedErr != nil {
			return wrappedErr
		}

		ginCtx.JSON(http.StatusOK, &LoginStartResponse{
			Errors:            []servercommon.ErrorDetail{},
			WebAuthnSessionID: sessionID,
			PublicKey:         options,
		})
		return nil
	})
}
