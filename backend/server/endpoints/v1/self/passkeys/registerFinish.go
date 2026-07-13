package passkeys

import (
	"context"
	"net/http"

	"github.com/NicoClack/cryptic-stash/backend/auth"
	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/common/dbcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/ent/user"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
)

type RegisterFinishPayload struct {
	protocol.CredentialCreationResponse

	WebAuthnSessionID uuid.UUID `binding:"required" json:"webAuthnSessionId"`
	Name              string    `binding:"required" json:"name"`
	AllowSudo         bool      `                   json:"allowSudo"`
	IsSecondGroup     bool      `                   json:"isSecondGroup"`
}

type RegisterFinishResponse struct {
	Errors []servercommon.ErrorDetail `json:"errors"`
}

func RegisterFinish(app *servercommon.ServerApp) gin.HandlerFunc {
	return servercommon.NewHandler(func(ginCtx *gin.Context) error {
		userID := ginCtx.MustGet("user").(*ent.User).ID

		body := RegisterFinishPayload{}
		if serverErr := servercommon.ParseBody(&body, ginCtx); serverErr != nil {
			return serverErr
		}
		parsedCredential, stdErr := body.Parse()
		if stdErr != nil {
			return servercommon.NewError(stdErr).
				SetStatus(http.StatusBadRequest).
				AddDetail(servercommon.ErrorDetail{
					Message: "invalid WebAuthn credential",
					Code:    "INVALID_CREDENTIAL",
				})
		}

		var sessionData *webauthn.SessionData
		if !app.TempKeyValue.Get(auth.WebAuthnSessionStoreName, body.WebAuthnSessionID.String(), &sessionData) {
			return servercommon.NewBadRequestError(
				"WebAuthnSessionID",
				"missing or expired",
				"INVALID_WEBAUTHN_SESSION",
			)
		}
		resp, stdErr := dbcommon.WithReadWriteTx(
			ginCtx.Request.Context(),
			app.Database,
			func(tx *ent.Tx, ctx context.Context) (*RegisterFinishResponse, error) {
				userOb, stdErr := tx.User.Query().
					Where(user.ID(userID)).
					WithPasskeys(). // The user loaded by the auth middleware doesn't include this edge
					Only(ctx)
				if stdErr != nil {
					return nil, stdErr
				}

				_, wrappedErr := app.Auth.FinishRegisterPasskey(
					body.Name,
					body.AllowSudo,
					body.IsSecondGroup,
					userOb.Username,
					sessionData,
					parsedCredential,
					tx,
					ctx,
					func(userID uuid.UUID, tx *ent.Tx) (*ent.User, error) {
						return userOb, nil
					},
				)
				if wrappedErr != nil {
					return nil, wrappedErr
				}

				return &RegisterFinishResponse{
					Errors: []servercommon.ErrorDetail{},
				}, nil
			},
		)
		if stdErr != nil {
			if common.IsErrorType[*protocol.Error](stdErr) {
				return servercommon.NewError(stdErr).
					SetStatus(http.StatusBadRequest).
					AddDetail(servercommon.ErrorDetail{
						Message: "invalid credential",
						Code:    "INVALID_CREDENTIAL",
					}).
					DisableLogging()
			}

			if dbcommon.IsUniqueConstraintError(stdErr, "passkeys") {
				return servercommon.NewError(stdErr).
					SetStatus(http.StatusConflict).
					AddDetail(servercommon.ErrorDetail{
						Message: "passkey name already exists",
						Code:    "DUPLICATE_PASSKEY_NAME",
					}).
					DisableLogging()
			}

			return stdErr
		}
		app.TempKeyValue.Delete(auth.WebAuthnSessionStoreName, body.WebAuthnSessionID.String())

		ginCtx.JSON(http.StatusCreated, resp)
		return nil
	})
}
