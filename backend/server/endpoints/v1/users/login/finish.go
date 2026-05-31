package login

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"

	"github.com/NicoClack/cryptic-stash/backend/auth"
	"github.com/NicoClack/cryptic-stash/backend/common/dbcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/google/uuid"
)

type LoginFinishPayload struct {
	protocol.CredentialAssertionResponse

	WebAuthnSessionID uuid.UUID `binding:"required" json:"webAuthnSessionId"`
}

type LoginFinishResponse struct {
	Errors []servercommon.ErrorDetail `json:"errors"`
	UserID uuid.UUID                  `json:"userId"`
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
					UserID: sessionOb.UserID,
					Token:  base64.RawURLEncoding.EncodeToString(token),
				}, nil
			},
		)
		if stdErr != nil {
			var protoErr *protocol.Error
			if errors.As(stdErr, &protoErr) && protoErr.Type == protocol.ErrBadRequest.Type {
				return servercommon.NewError(stdErr).SetStatus(http.StatusBadRequest).
					AddDetail(servercommon.ErrorDetail{
						Message: "invalid credential",
						Code:    "INVALID_CREDENTIAL",
					}).
					DisableLogging()
			}

			return servercommon.ExpectError(
				stdErr, auth.ErrInvalidWebAuthnSessionID, http.StatusBadRequest,
				&servercommon.ErrorDetail{
					Message: "WebAuthn session missing or expired",
					Code:    "INVALID_WEBAUTHN_SESSION",
				},
			)
		}

		ginCtx.JSON(http.StatusOK, resp)
		return nil
	})
}
