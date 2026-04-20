package invites

import (
	"context"
	"net/http"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/gin-gonic/gin"
)

type GetInviteResponse struct {
	Errors    []servercommon.ErrorDetail `binding:"required" json:"errors"`
	Email     string                     `                   json:"email"`
	ExpiresAt time.Time                  `                   json:"expiresAt"`
}

func GetInvite(app *servercommon.ServerApp) gin.HandlerFunc {
	return servercommon.NewHandler(func(ginCtx *gin.Context) error {
		resp, stdErr := useInvite(
			ginCtx, app,
			func(inviteOb *ent.Invite, tx *ent.Tx, ctx context.Context) (*GetInviteResponse, error) {
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
