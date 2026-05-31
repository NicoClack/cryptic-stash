package invites

import (
	"context"
	"crypto/sha256"
	"encoding/base64"

	"github.com/NicoClack/cryptic-stash/backend/common/dbcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/ent/invite"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func useInvite[T any](
	id uuid.UUID,
	ginCtx *gin.Context, app *servercommon.ServerApp,
	respFunc func(inviteOb *ent.Invite, tx *ent.Tx, ctx context.Context) (*T, error),
) (*T, error) {
	token, serverErr := servercommon.RequireAuthorizationScheme("Bearer", ginCtx)
	if serverErr != nil {
		return nil, serverErr
	}
	givenCodeBytes, stdErr := base64.RawURLEncoding.DecodeString(token)
	if stdErr != nil {
		return nil, servercommon.NewBadRequestError(
			"authorization",
			"malformed token",
			"MALFORMED_AUTHORIZATION_TOKEN",
		)
	}

	hashed := sha256.Sum256(givenCodeBytes)
	return dbcommon.WithReadWriteTx(
		ginCtx.Request.Context(), app.Database,
		func(tx *ent.Tx, ctx context.Context) (*T, error) {
			inviteOb, stdErr := tx.Invite.Query().
				Where(
					invite.ID(id),
					invite.HashedCode(hashed[:]),
				).
				Only(ctx)
			if stdErr != nil {
				return nil, servercommon.SendUnauthorizedIfNotFound(stdErr)
			}
			if inviteOb.UserID != uuid.Nil ||
				app.Clock.Now().After(inviteOb.ExpiresAt) ||
				inviteOb.ExpiredReason != nil {
				return nil, servercommon.NewUnauthorizedError()
			}

			return respFunc(inviteOb, tx, ctx)
		},
	)
}
