package passkeys

import (
	"context"
	"net/http"

	"github.com/NicoClack/cryptic-stash/backend/auth"
	"github.com/NicoClack/cryptic-stash/backend/common/dbcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/ent/user"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/google/uuid"
)

type RegisterStartResponse struct {
	Errors            []servercommon.ErrorDetail                  `json:"errors"`
	WebAuthnSessionID uuid.UUID                                   `json:"webAuthnSessionId"`
	PublicKey         protocol.PublicKeyCredentialCreationOptions `json:"publicKey"`
}

func RegisterStart(app *servercommon.ServerApp) gin.HandlerFunc {
	return servercommon.NewHandler(func(ginCtx *gin.Context) error {
		userID := ginCtx.MustGet("user").(*ent.User).ID

		userOb, stdErr := dbcommon.WithReadTx(
			ginCtx.Request.Context(),
			app.Database,
			func(tx *ent.Tx, ctx context.Context) (*ent.User, error) {
				return tx.User.Query().
					Where(user.ID(userID)).
					WithPasskeys().
					Only(ctx)
			},
		)
		if stdErr != nil {
			return stdErr
		}

		options, sessionData, wrappedErr := app.Auth.StartRegisterPasskey(
			&auth.RealWebAuthnUser{User: userOb},
			ginCtx.Request.Context(),
		)
		if wrappedErr != nil {
			return wrappedErr
		}

		webAuthnSessionID := uuid.New()
		app.TempKeyValue.Set(
			auth.WebAuthnSessionStoreName,
			webAuthnSessionID.String(),
			sessionData,
			sessionData.Expires,
		)

		ginCtx.JSON(http.StatusOK, &RegisterStartResponse{
			Errors:            []servercommon.ErrorDetail{},
			WebAuthnSessionID: webAuthnSessionID,
			PublicKey:         options,
		})
		return nil
	})
}
