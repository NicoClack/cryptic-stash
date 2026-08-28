package invites

import (
	"context"
	"net/http"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/common/dbcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/invites"
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
	return newInviteTokenHandler(func(id uuid.UUID, code []byte, ginCtx *gin.Context) error {
		actor := &common.Actor{
			IP:        ginCtx.ClientIP(),
			UserAgent: ginCtx.Request.UserAgent(),
		}

		resp, stdErr := dbcommon.WithReadWriteTx(
			ginCtx.Request.Context(), app.Database,
			func(tx *ent.Tx, ctx context.Context) (*GenerateOptionsResponse, error) {
				options, wrappedErr := app.Invites.GenerateOptions(id, code, actor, tx, ctx)
				if wrappedErr != nil {
					return nil, servercommon.ExpectAnyOfErrors(
						wrappedErr,
						[]error{
							invites.ErrInviteNotFound,
							invites.ErrInviteUsed,
							invites.ErrInviteExpired,
						},
						http.StatusUnauthorized,
						nil,
					)
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
