package middleware

import (
	"context"
	"encoding/base64"
	"net/http"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/common/dbcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/gin-gonic/gin"
)

const (
	sessionContextKey = "session"
	userContextKey    = "user"
)

type SessionAuthOptions struct {
	RequireSuperuser bool
	AdminProtected   bool
	AllowAnonymous   bool
}

// By default, any user can call the endpoint
func NewSessionAuth(
	auth common.AuthService,
	db common.DatabaseService,
	options *SessionAuthOptions,
) gin.HandlerFunc {
	if options == nil {
		options = &SessionAuthOptions{}
	}

	return func(ginCtx *gin.Context) {
		givenTokenStr, serverErr := servercommon.RequireAuthorizationScheme("Session", ginCtx)
		if serverErr != nil {
			if options.AllowAnonymous {
				ginCtx.Next()
				return
			}
			ginCtx.Error(serverErr)
			ginCtx.Abort()
			return
		}

		givenTokenBytes, stdErr := base64.RawURLEncoding.DecodeString(givenTokenStr)
		if stdErr != nil {
			ginCtx.Error(
				servercommon.NewError(stdErr).
					SetStatus(http.StatusBadRequest).
					AddDetail(servercommon.ErrorDetail{
						Message: "session token is not valid raw URL base64",
						Code:    "MALFORMED_SESSION_TOKEN",
					}).
					DisableLogging(),
			)
			ginCtx.Abort()
			return
		}

		sessionOb, stdErr := dbcommon.WithReadTx(
			ginCtx.Request.Context(), db,
			func(tx *ent.Tx, ctx context.Context) (*ent.Session, error) {
				sessionOb, wrappedErr := auth.ValidateSession(givenTokenBytes, tx, ctx)
				if wrappedErr != nil {
					return nil, wrappedErr
				}
				return sessionOb, nil
			},
		)
		if stdErr != nil {
			ginCtx.Error(stdErr)
			ginCtx.Abort()
			return
		}

		if options.AdminProtected {
			if sessionOb.Edges.User.Username != common.AdminUsername {
				ginCtx.Error(servercommon.NewForbiddenError().
					AddDetail(servercommon.ErrorDetail{
						Message: "only admin user can use this endpoint",
						Code:    "ADMIN_REQUIRED",
					}),
				)
				ginCtx.Abort()
				return
			}
		}
		if options.RequireSuperuser || options.AdminProtected {
			if !sessionOb.SuperUserMode {
				ginCtx.Error(servercommon.NewForbiddenError().
					AddDetail(servercommon.ErrorDetail{
						Message: "superuser mode required",
						Code:    "SUPERUSER_MODE_REQUIRED",
					}),
				)
				ginCtx.Abort()
				return
			}
		}

		ginCtx.Set(sessionContextKey, sessionOb)
		ginCtx.Set(userContextKey, sessionOb.Edges.User)
		ginCtx.Next()
	}
}
