package login

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"

	"github.com/NicoClack/cryptic-stash/backend/auth"
	"github.com/NicoClack/cryptic-stash/backend/common"
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
	Errors          []servercommon.ErrorDetail `json:"errors"`
	UserID          uuid.UUID                  `json:"userId"`
	Token           string                     `json:"token"`
	Username        string                     `json:"username"`
	IsSuperUserMode bool                       `json:"isSuperUserMode"`
	IsSecondGroup   bool                       `json:"isSecondGroup"`
}

func FinishLogin(app *servercommon.ServerApp) gin.HandlerFunc {
	// Note: username enumeration shouldn't be possible in the future for this endpoint.
	// It's might be possible to detect if a credential ID or a user ID is valid, but that's not useful information
	// because I don't ever plan to add an endpoint that returns information about another user.
	// An endpoint that confirms if a credential ID is valid can be used to confirm if a user is registered on this site,
	// but they're likely one step away from being pwned at that point e.g:
	// 1. Physical access (but no PIN access) to a security key that leaks metadata
	//    -> Only needs PIN to access account, decent chance the key has other vulnerabilities.
	//       Mitigated if passkey is revoked from the user's account.
	// 2. Malware gets credential IDs from a password manager that doesn't secure metadata
	//    -> Probably also has browsing history, can session steal.
	return servercommon.NewHandler(func(ginCtx *gin.Context) error {
		body := LoginFinishPayload{}
		if serverErr := servercommon.ParseBody(&body, ginCtx); serverErr != nil {
			return serverErr
		}
		parsedResponse, stdErr := body.Parse()
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
				userOb, passkeyOb, sessionOb, token, wrappedErr := app.Auth.FinishLogin(
					body.WebAuthnSessionID,
					parsedResponse,
					ginCtx,
					tx,
				)
				if wrappedErr != nil {
					return nil, wrappedErr
				}

				return &LoginFinishResponse{
					Errors:          []servercommon.ErrorDetail{},
					UserID:          sessionOb.UserID,
					Token:           base64.RawURLEncoding.EncodeToString(token),
					Username:        userOb.Username,
					IsSuperUserMode: sessionOb.SuperUserMode,
					IsSecondGroup:   passkeyOb.IsSecondGroup,
				}, nil
			},
		)
		if stdErr != nil {
			if common.IsErrorType[*protocol.Error](stdErr) ||
				errors.Is(stdErr, auth.ErrWebAuthnUserNotFound) {
				return servercommon.NewError(stdErr).
					SetStatus(http.StatusBadRequest).
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
