package login

import (
	"context"
	"encoding/base64"
	"net/http"

	"github.com/NicoClack/cryptic-stash/backend/auth"
	"github.com/NicoClack/cryptic-stash/backend/common/dbcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/protocol"
)

type LoginFinishPayload struct {
	protocol.CredentialAssertionResponse

	WebAuthnSessionID string `binding:"required,min=1,max=64" json:"webAuthnSessionID"`
}

type LoginFinishResponse struct {
	Errors []servercommon.ErrorDetail `json:"errors"`
	UserID string                     `json:"userID"`
	Token  string                     `json:"token"`
}

func FinishLogin(app *servercommon.ServerApp) gin.HandlerFunc {
	return servercommon.NewHandler(func(ginCtx *gin.Context) error {
		body := LoginFinishPayload{}
		if serverErr := servercommon.ParseBody(&body, ginCtx); serverErr != nil {
			return serverErr
		}
		parsedResponse, stdErr := body.CredentialAssertionResponse.Parse()
		if stdErr != nil {
			return servercommon.NewError(stdErr).
				SetStatus(http.StatusBadRequest).
				AddDetail(servercommon.ErrorDetail{
					Message: "malformed WebAuthn assertion response",
					Code:    "MALFORMED_CREDENTIAL_ASSERTION_RESPONSE",
				})
		}

		resp, stdErr := dbcommon.WithReadWriteTx(
			ginCtx.Request.Context(),
			app.Database,
			func(tx *ent.Tx, ctx context.Context) (*LoginFinishResponse, error) {
				sessionOb, token, wrappedErr := app.Auth.FinishLogin(
					body.WebAuthnSessionID,
					parsedResponse,
					ginCtx,
					tx,
				)
				if wrappedErr != nil {
					return nil, wrappedErr
				}

				return &LoginFinishResponse{
					Errors: []servercommon.ErrorDetail{},
					UserID: sessionOb.UserID.String(),
					Token:  base64.RawURLEncoding.EncodeToString(token),
				}, nil
			},
		)
		if stdErr != nil {
			return servercommon.ExpectError(
				stdErr, auth.ErrInvalidWebAuthnSessionID, http.StatusBadRequest,
				&servercommon.ErrorDetail{
					Message: "WebAuthn session missing or expired",
					Code:    "INVALID_WEBAUTHN_SESSION",
				},
			).Expect(
				auth.ErrInvalidCredential, http.StatusUnauthorized,
				nil,
			)
		}

		ginCtx.JSON(http.StatusOK, resp)
		return nil
	})
}
