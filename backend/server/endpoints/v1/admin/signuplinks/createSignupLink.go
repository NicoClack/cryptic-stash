package signuplinks

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common/dbcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/gin-gonic/gin"
)

type CreatePayload struct {
	Name      string `binding:"omitempty,min=1,max=32" json:"name"`
	ExpiresIn int64  `binding:"omitempty"              json:"expiresIn"`
}
type CreateResponse struct {
	Errors    []servercommon.ErrorDetail `binding:"required" json:"errors"`
	ID        string                     `                   json:"id"`
	Code      string                     `                   json:"code"`
	ExpiresAt time.Time                  `                   json:"expiresAt"`
}

func Create(app *servercommon.ServerApp) gin.HandlerFunc {
	clock := app.Clock

	return servercommon.NewHandler(func(ginCtx *gin.Context) error {
		body := CreatePayload{}
		if ctxErr := servercommon.ParseBody(&body, ginCtx); ctxErr != nil {
			return ctxErr
		}
		if body.Name != "" {
			if serverErr := servercommon.ValidateUsername(body.Name); serverErr != nil {
				return serverErr
			}
		}

		expiresIn := app.Env.SIGNUP_LINK_DEFAULT_EXPIRY
		if body.ExpiresIn > 0 {
			expiresIn = time.Duration(body.ExpiresIn) * time.Second
		}
		expiresIn = min(expiresIn, app.Env.SIGNUP_LINK_MAX_EXPIRY)

		resp, stdErr := dbcommon.WithReadWriteTx(
			ginCtx.Request.Context(), app.Database,
			func(tx *ent.Tx, ctx context.Context) (*CreateResponse, error) {
				code := app.Core.RandomAuthCode()
				hashed := sha256.Sum256(code)
				now := clock.Now()
				expiresAt := now.Add(expiresIn)

				signupOb, stdErr := tx.SignupLink.Create().
					SetCreatedAt(now).
					SetUpdatedAt(now).
					SetName(body.Name).
					SetHashedCode(hashed[:]).
					SetExpiresAt(expiresAt).
					Save(ctx)
				if stdErr != nil {
					return nil, stdErr
				}

				return &CreateResponse{
					Errors:    []servercommon.ErrorDetail{},
					ID:        signupOb.ID.String(),
					Code:      base64.StdEncoding.EncodeToString(code),
					ExpiresAt: expiresAt,
				}, nil
			},
		)
		if stdErr != nil {
			return stdErr
		}

		ginCtx.JSON(http.StatusCreated, resp)
		return nil
	})
}
