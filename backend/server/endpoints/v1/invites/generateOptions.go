package invites

import (
	"context"
	"net/http"

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
	webAuthnApp, _ := newWebAuthnApp(app)

	return servercommon.NewHandler(func(ginCtx *gin.Context) error {
		resp, stdErr := useInvite(
			ginCtx, app,
			func(inviteOb *ent.Invite, tx *ent.Tx, ctx context.Context) (*GenerateOptionsResponse, error) {
				pendingUserID := uuid.New()
				creation, session, stdErr := webAuthnApp.BeginRegistration(&webAuthnUser{
					id:          pendingUserID[:],
					name:        inviteOb.Email,
					displayName: inviteOb.Email,
				})
				if stdErr != nil {
					return nil, stdErr
				}

				_, stdErr = tx.Invite.UpdateOneID(inviteOb.ID).
					SetWebAuthnSession(session).
					Save(ctx)
				if stdErr != nil {
					return nil, stdErr
				}

				return &GenerateOptionsResponse{
					Errors:    []servercommon.ErrorDetail{},
					PublicKey: creation.Response,
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
