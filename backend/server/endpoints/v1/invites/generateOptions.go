package invites

import (
	"context"
	"net/http"

	"github.com/NicoClack/cryptic-stash/backend/auth"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/google/uuid"
)

type GenerateOptionsResponse struct {
	Errors    []servercommon.ErrorDetail                  `json:"errors"`
	PublicKey protocol.PublicKeyCredentialCreationOptions `json:"publicKey"`
}

func GenerateOptions(app *servercommon.ServerApp) gin.HandlerFunc {
	return servercommon.NewObjectIDHandler(func(id uuid.UUID, ginCtx *gin.Context) error {
		resp, stdErr := useInvite(
			id, ginCtx, app,
			func(inviteOb *ent.Invite, tx *ent.Tx, ctx context.Context) (*GenerateOptionsResponse, error) {
				pendingUserID := uuid.New()
				options, sessionData, wrappedErr := app.Auth.StartRegisterPasskey(
					&auth.TempWebAuthnUser{
						ID:          pendingUserID[:],
						Name:        inviteOb.Email,
						DisplayName: inviteOb.Email,
					},
					ctx,
				)
				if wrappedErr != nil {
					return nil, wrappedErr
				}

				_, stdErr := tx.Invite.UpdateOneID(inviteOb.ID).
					SetWebAuthnSession(sessionData).
					Save(ctx)
				if stdErr != nil {
					return nil, stdErr
				}

				return &GenerateOptionsResponse{
					Errors:    []servercommon.ErrorDetail{},
					PublicKey: options,
				}, nil
			},
		)
		if stdErr != nil {
			return stdErr
		}

		ginCtx.JSON(http.StatusOK, resp)
		return nil
	})
}
