package invites

import (
	"encoding/base64"

	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func newInviteTokenHandler(
	handler func(id uuid.UUID, code []byte, ginCtx *gin.Context) error,
) gin.HandlerFunc {
	return servercommon.NewObjectIDHandler(func(id uuid.UUID, ginCtx *gin.Context) error {
		token, serverErr := servercommon.RequireAuthorizationScheme("Bearer", ginCtx)
		if serverErr != nil {
			return serverErr
		}
		code, stdErr := base64.RawURLEncoding.DecodeString(token)
		if stdErr != nil {
			return servercommon.NewBadRequestError(
				"authorization",
				"malformed token",
				"MALFORMED_AUTHORIZATION_TOKEN",
			)
		}
		return handler(id, code, ginCtx)
	})
}
