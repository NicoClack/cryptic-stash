package invites

import (
	"context"
	"net/http"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common/dbcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/invites"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type GetInviteResponse struct {
	Errors    []servercommon.ErrorDetail `binding:"required" json:"errors"`
	Email     string                     `                   json:"email"`
	ExpiresAt time.Time                  `                   json:"expiresAt"`
}

func GetInvite(app *servercommon.ServerApp) gin.HandlerFunc {
	return newInviteTokenHandler(func(id uuid.UUID, code []byte, ginCtx *gin.Context) error {
		resp, stdErr := dbcommon.WithReadTx(
			ginCtx.Request.Context(), app.Database,
			func(tx *ent.Tx, ctx context.Context) (*GetInviteResponse, error) {
				inviteOb, wrappedErr := app.Invites.GetInvite(id, code, tx, ctx)
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
				return &GetInviteResponse{
					Errors:    []servercommon.ErrorDetail{},
					Email:     inviteOb.Email,
					ExpiresAt: inviteOb.ExpiresAt,
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
