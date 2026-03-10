package signuplinks

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common/dbcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/ent/signuplink"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type GetSignupLinkResponse struct {
	Errors        []servercommon.ErrorDetail `binding:"required" json:"errors"`
	SuggestedName string                     `                   json:"suggestedName"`
	ExpiresAt     time.Time                  `                   json:"expiresAt"`
}

func GetSignupLink(app *servercommon.ServerApp) gin.HandlerFunc {
	clock := app.Clock

	return servercommon.NewHandler(func(ginCtx *gin.Context) error {
		signupID, ctxErr := servercommon.ParseObjectID(ginCtx.Param("id"))
		if ctxErr != nil {
			return ctxErr
		}

		token, serverErr := servercommon.RequireAuthorizationScheme("Bearer", ginCtx)
		if serverErr != nil {
			return serverErr
		}
		givenCodeBytes, stdErr := base64.StdEncoding.DecodeString(token)
		if stdErr != nil {
			return servercommon.NewBadRequestError(
				"authorization",
				"malformed token",
				"MALFORMED_AUTHORIZATION_TOKEN",
			)
		}

		hashed := sha256.Sum256(givenCodeBytes)
		resp, stdErr := dbcommon.WithReadTx(
			ginCtx.Request.Context(), app.Database,
			func(tx *ent.Tx, ctx context.Context) (*GetSignupLinkResponse, error) {
				signupOb, stdErr := tx.SignupLink.Query().
					Where(
						signuplink.ID(signupID),
						signuplink.HashedCode(hashed[:]),
					).
					Only(ctx)
				if stdErr != nil {
					return nil, servercommon.SendUnauthorizedIfNotFound(stdErr)
				}

				if clock.Now().After(signupOb.ExpiresAt) ||
					signupOb.UserID != uuid.Nil {
					return nil, servercommon.NewUnauthorizedError()
				}

				return &GetSignupLinkResponse{
					Errors:        []servercommon.ErrorDetail{},
					SuggestedName: signupOb.Name,
					ExpiresAt:     signupOb.ExpiresAt,
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
