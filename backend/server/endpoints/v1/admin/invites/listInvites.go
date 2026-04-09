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

type ListInvitesResponse struct {
	Errors  []servercommon.ErrorDetail `binding:"required" json:"errors"`
	Invites []*Invite                  `binding:"required" json:"invites"`
}
type Invite struct {
	ID        string    `binding:"required" json:"id"`
	Email     string    `                   json:"email"`
	CreatedAt time.Time `                   json:"createdAt"`
	ExpiresAt time.Time `                   json:"expiresAt"`
	UserID    string    `                   json:"userId,omitempty"`
	IP        string    `                   json:"ip"`
	UserAgent string    `                   json:"userAgent"`
}

func ListInvites(app *servercommon.ServerApp) gin.HandlerFunc {
	return servercommon.NewHandler(func(ginCtx *gin.Context) error {
		email := ginCtx.Query("email")

		inviteObs, stdErr := dbcommon.WithReadTx(
			ginCtx.Request.Context(), app.Database,
			func(tx *ent.Tx, ctx context.Context) ([]*ent.Invite, error) {
				inviteQuery := tx.Invite.Query()
				if email != "" {
					inviteQuery = inviteQuery.Where(invite.EmailHasPrefix(email))
				}
				return inviteQuery.Select(
					invite.FieldID,
					invite.FieldEmail,
					invite.FieldCreatedAt,
					invite.FieldExpiresAt,
					invite.FieldIP,
					invite.FieldUserAgent,
					invite.FieldUserID,
				).Order(ent.Desc(invite.FieldCreatedAt)).All(ctx)
			},
		)
		if stdErr != nil {
			return stdErr
		}

		resp := make([]*Invite, 0, len(inviteObs))
		for _, inviteOb := range inviteObs {
			userID := ""
			if inviteOb.UserID != uuid.Nil {
				userID = inviteOb.UserID.String()
			}
			resp = append(resp, &Invite{
				ID:        inviteOb.ID.String(),
				Email:     inviteOb.Email,
				CreatedAt: inviteOb.CreatedAt,
				ExpiresAt: inviteOb.ExpiresAt,
				UserID:    userID,
				IP:        inviteOb.IP,
				UserAgent: inviteOb.UserAgent,
			})
		}

		ginCtx.JSON(http.StatusOK, ListInvitesResponse{
			Errors:  []servercommon.ErrorDetail{},
			Invites: resp,
		})
		return nil
	})
}
