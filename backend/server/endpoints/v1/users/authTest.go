package users

import (
	"context"
	"encoding/base64"
	"net/http"

	"github.com/NicoClack/cryptic-stash/backend/common/dbcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AuthTestResponse struct {
	Errors    []servercommon.ErrorDetail `json:"errors"`
	SessionID uuid.UUID                  `json:"sessionId"`
	UserID    uuid.UUID                  `json:"userId"`
	Username  string                     `json:"username"`
}

func AuthTest(app *servercommon.ServerApp) gin.HandlerFunc {
	return servercommon.NewHandler(func(ginCtx *gin.Context) error {
		givenTokenStr, serverErr := servercommon.RequireAuthorizationScheme("Session", ginCtx)
		if serverErr != nil {
			return serverErr
		}
		givenTokenBytes, stdErr := base64.RawURLEncoding.DecodeString(givenTokenStr)
		if stdErr != nil {
			return servercommon.NewError(stdErr).
				SetStatus(http.StatusBadRequest).
				AddDetail(servercommon.ErrorDetail{
					Message: "session token is not valid raw URL base64",
					Code:    "MALFORMED_SESSION_TOKEN",
				}).
				DisableLogging()
		}

		resp, stdErr := dbcommon.WithReadTx(
			ginCtx.Request.Context(), app.Database,
			func(tx *ent.Tx, ctx context.Context) (*AuthTestResponse, error) {
				sessionOb, wrappedErr := app.Auth.ValidateSession(givenTokenBytes, tx, ctx)
				if wrappedErr != nil {
					return nil, servercommon.NewUnauthorizedError()
				}
				userOb := sessionOb.Edges.User

				return &AuthTestResponse{
					Errors:    []servercommon.ErrorDetail{},
					SessionID: sessionOb.ID,
					UserID:    userOb.ID,
					Username:  userOb.Username,
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
