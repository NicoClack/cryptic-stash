package invites

import (
	"context"
	"net/http"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common/dbcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/ent/invite"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TODO: add properties to say if the link was used?
type GetInviteResponse struct {
	Errors    []servercommon.ErrorDetail `binding:"required" json:"errors"`
	ID        string                     `                   json:"id"`
	Email     string                     `                   json:"email"`
	CreatedAt time.Time                  `                   json:"createdAt"`
	ExpiresAt time.Time                  `                   json:"expiresAt"`
	UserID    string                     `                   json:"userId,omitempty"`
	IP        string                     `                   json:"ip"`
	UserAgent string                     `                   json:"userAgent"`
}

func GetInvite(app *servercommon.ServerApp) gin.HandlerFunc {
	return servercommon.NewHandler(func(ginCtx *gin.Context) error {
		inviteID, serverErr := servercommon.ParseObjectID(ginCtx.Param("id"))
		if serverErr != nil {
			return serverErr
		}

		resp, stdErr := dbcommon.WithReadTx(
			ginCtx.Request.Context(), app.Database,
			func(tx *ent.Tx, ctx context.Context) (*GetInviteResponse, error) {
				inviteOb, stdErr := tx.Invite.Query().
					Where(invite.ID(inviteID)).
					Only(ctx)
				if stdErr != nil {
					return nil, stdErr
				}

				userID := ""
				if inviteOb.UserID != uuid.Nil {
					userID = inviteOb.UserID.String()
				}

				return &GetInviteResponse{
					Errors:    []servercommon.ErrorDetail{},
					ID:        inviteOb.ID.String(),
					Email:     inviteOb.Email,
					CreatedAt: inviteOb.CreatedAt,
					ExpiresAt: inviteOb.ExpiresAt,
					UserID:    userID,
					IP:        inviteOb.IP,
					UserAgent: inviteOb.UserAgent,
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
