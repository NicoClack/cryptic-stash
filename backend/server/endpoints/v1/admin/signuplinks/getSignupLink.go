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

// TODO: add properties to say if the link was used?
type GetSignupLinkAdminResponse struct {
	Errors    []servercommon.ErrorDetail `binding:"required" json:"errors"`
	ID        string                     `                   json:"id"`
	Name      string                     `                   json:"name"`
	CreatedAt time.Time                  `                   json:"createdAt"`
	ExpiresAt time.Time                  `                   json:"expiresAt"`
	UserID    string                     `                   json:"userId,omitempty"`
	IP        string                     `                   json:"ip"`
	UserAgent string                     `                   json:"userAgent"`
}

func Get(app *servercommon.ServerApp) gin.HandlerFunc {
	return servercommon.NewHandler(func(ginCtx *gin.Context) error {
		signupID, ctxErr := servercommon.ParseObjectID(ginCtx.Param("id"))
		if ctxErr != nil {
			return ctxErr
		}

		resp, stdErr := dbcommon.WithReadTx(
			ginCtx.Request.Context(), app.Database,
			func(tx *ent.Tx, ctx context.Context) (*GetSignupLinkAdminResponse, error) {
				signupOb, stdErr := tx.SignupLink.Query().
					Where(signuplink.ID(signupID)).
					Only(ctx)
				if stdErr != nil {
					return nil, stdErr
				}

				userID := ""
				if signupOb.UserID != uuid.Nil {
					userID = signupOb.UserID.String()
				}

				return &GetSignupLinkAdminResponse{
					Errors:    []servercommon.ErrorDetail{},
					ID:        signupOb.ID.String(),
					Name:      signupOb.Name,
					CreatedAt: signupOb.CreatedAt,
					ExpiresAt: signupOb.ExpiresAt,
					UserID:    userID,
					IP:        signupOb.IP,
					UserAgent: signupOb.UserAgent,
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
