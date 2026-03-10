package signuplinks

import (
	"context"
	"net/http"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common/dbcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/ent/signuplink"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ListSignupLinksResponse struct {
	Errors      []servercommon.ErrorDetail `binding:"required" json:"errors"`
	SignupLinks []*SignupLink              `binding:"required" json:"signupLinks"`
}
type SignupLink struct {
	ID        string    `binding:"required" json:"id"`
	Name      string    `                   json:"name"`
	CreatedAt time.Time `                   json:"createdAt"`
	ExpiresAt time.Time `                   json:"expiresAt"`
	UserID    string    `                   json:"userId,omitempty"`
	IP        string    `                   json:"ip"`
	UserAgent string    `                   json:"userAgent"`
}

func List(app *servercommon.ServerApp) gin.HandlerFunc {
	return servercommon.NewHandler(func(ginCtx *gin.Context) error {
		name := ginCtx.Query("name")

		signupObs, stdErr := dbcommon.WithReadTx(
			ginCtx.Request.Context(), app.Database,
			func(tx *ent.Tx, ctx context.Context) ([]*ent.SignupLink, error) {
				signupQuery := tx.SignupLink.Query()
				if name != "" {
					signupQuery = signupQuery.Where(signuplink.NameHasPrefix(name))
				}
				return signupQuery.Select(
					signuplink.FieldID,
					signuplink.FieldName,
					signuplink.FieldCreatedAt,
					signuplink.FieldExpiresAt,
					signuplink.FieldIP,
					signuplink.FieldUserAgent,
					signuplink.FieldUserID,
				).All(ctx)
			},
		)
		if stdErr != nil {
			return stdErr
		}

		resp := make([]*SignupLink, 0, len(signupObs))
		for _, signupOb := range signupObs {
			userID := ""
			if signupOb.UserID != uuid.Nil {
				userID = signupOb.UserID.String()
			}
			resp = append(resp, &SignupLink{
				ID:        signupOb.ID.String(),
				Name:      signupOb.Name,
				CreatedAt: signupOb.CreatedAt,
				ExpiresAt: signupOb.ExpiresAt,
				UserID:    userID,
				IP:        signupOb.IP,
				UserAgent: signupOb.UserAgent,
			})
		}

		ginCtx.JSON(http.StatusOK, ListSignupLinksResponse{
			Errors:      []servercommon.ErrorDetail{},
			SignupLinks: resp,
		})
		return nil
	})
}
