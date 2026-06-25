package superuser

import (
	"context"
	"net/http"

	"github.com/NicoClack/cryptic-stash/backend/auth"
	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/common/dbcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/ent/session"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/google/uuid"
)

type FinishElevationPayload struct {
	protocol.CredentialAssertionResponse

	WebAuthnSessionID uuid.UUID `binding:"required" json:"webAuthnSessionId"`
}

type FinishElevationResponse struct {
	Errors []servercommon.ErrorDetail `json:"errors"`
}

func FinishElevation(app *servercommon.ServerApp) gin.HandlerFunc {
	return servercommon.NewHandler(func(ginCtx *gin.Context) error {
		sessionID := ginCtx.MustGet("session").(*ent.Session).ID

		body := FinishElevationPayload{}
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

		_, stdErr = dbcommon.WithReadWriteTx(
			ginCtx.Request.Context(),
			app.Database,
			func(tx *ent.Tx, ctx context.Context) (*struct{}, error) {
				sessionOb, stdErr := tx.Session.Query().
					Where(session.ID(sessionID)).
					WithPasskey(). // Not loaded by the auth middleware
					Only(ctx)
				if stdErr != nil {
					return nil, stdErr
				}

				wrappedErr := app.Auth.FinishElevation(
					body.WebAuthnSessionID,
					parsedResponse,
					sessionOb,
					ginCtx,
					tx,
				)
				if wrappedErr != nil {
					return nil, wrappedErr
				}

				return nil, nil
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

			return servercommon.ExpectError(
				stdErr, auth.ErrInvalidWebAuthnSessionID, http.StatusBadRequest,
				&servercommon.ErrorDetail{
					Message: "WebAuthn session missing or expired",
					Code:    "INVALID_WEBAUTHN_SESSION",
				},
			).Expect(
				auth.ErrNeitherPasskeySuperEligible,
				http.StatusForbidden,
				&servercommon.ErrorDetail{
					Message: "neither passkey is eligible for superuser mode",
					Code:    "NEITHER_PASSKEY_SUPER_ELIGIBLE",
				},
			)
		}

		ginCtx.JSON(http.StatusOK, &FinishElevationResponse{
			Errors: []servercommon.ErrorDetail{},
		})
		return nil
	})
}
