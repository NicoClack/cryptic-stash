package users

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/common/dbcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/ent/user"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type GetAuthorizationCodePayload struct {
	Username string `binding:"required,min=1,max=32"  json:"username"`
	Password string `binding:"required,min=8,max=256" json:"password"` // #nosec G117
}

type GetAuthorizationCodeResponse struct {
	Errors            []servercommon.ErrorDetail `binding:"required" json:"errors"`
	AuthorizationCode string                     `                   json:"authorizationCode"`
	ValidFrom         time.Time                  `                   json:"validFrom"`
	ValidUntil        time.Time                  `                   json:"validUntil"`
}

func GetAuthorizationCode(app *servercommon.ServerApp) gin.HandlerFunc {
	clock := app.Clock

	return servercommon.NewHandler(func(ginCtx *gin.Context) error {
		body := GetAuthorizationCodePayload{}
		if ctxErr := servercommon.ParseBody(&body, ginCtx); ctxErr != nil {
			return ctxErr
		}
		if serverErr := servercommon.ValidateUsername(body.Username); serverErr != nil {
			return serverErr
		}

		userOb, stdErr := dbcommon.WithReadTx(
			ginCtx.Request.Context(), app.Database,
			func(tx *ent.Tx, ctx context.Context) (*ent.User, error) {
				userOb, stdErr := tx.User.Query().
					Where(user.Username(body.Username)).
					WithMessengers().
					WithStash().
					Only(ctx)
				if stdErr != nil {
					return nil, servercommon.SendUnauthorizedIfNotFound(stdErr)
				}
				return userOb, nil
			},
		)
		if stdErr != nil {
			return stdErr
		}
		if app.Core.IsUserLocked(userOb) {
			return servercommon.NewUnauthorizedError()
		}

		stashOb := userOb.Edges.Stash
		if stashOb == nil {
			return servercommon.NewUnauthorizedError()
		}
		stashKek := app.Core.HashPassword(
			body.Password,
			stashOb.PasswordSalt,
			&common.PasswordHashSettings{
				Time:    stashOb.HashTime,
				Memory:  stashOb.HashMemory,
				Threads: stashOb.HashThreads,
			},
		)
		_, wrappedErr := app.Core.Decrypt(stashOb.EncryptionDataKey, stashKek)
		if wrappedErr != nil {
			return servercommon.NewUnauthorizedError()
		}

		resp, stdErr := dbcommon.WithReadWriteTx(
			ginCtx.Request.Context(), app.Database,
			func(tx *ent.Tx, ctx context.Context) (*GetAuthorizationCodeResponse, error) {
				now := clock.Now()
				authCode := app.Core.RandomAuthCode()
				validFrom := now.Add(app.Env.UNLOCK_TIME)
				validUntil := now.Add(app.Env.AUTH_CODE_VALID_FOR)
				hashedAuthCode := sha256.Sum256(authCode)

				downloadSessionOb, stdErr := tx.DownloadSession.Create().
					SetCreatedAt(now).
					SetUpdatedAt(now).
					SetUser(userOb).
					SetHashedAuthCode(hashedAuthCode[:]).
					SetValidFrom(validFrom).
					SetValidUntil(validUntil).
					SetUserAgent(ginCtx.Request.UserAgent()).
					SetIP(ginCtx.ClientIP()).
					Save(ctx)
				if stdErr != nil {
					return nil, stdErr
				}

				_, _, wrappedErr := app.Messengers.SendUsingAll(
					&common.Message{
						Type:               common.MessageLogin,
						User:               userOb,
						Time:               validFrom,
						DownloadSessionIDs: []uuid.UUID{downloadSessionOb.ID},
					},
					ctx,
				)
				if wrappedErr != nil {
					return nil, wrappedErr
				}

				return &GetAuthorizationCodeResponse{
					Errors:            []servercommon.ErrorDetail{},
					AuthorizationCode: base64.StdEncoding.EncodeToString(authCode),
					ValidFrom:         validFrom,
					ValidUntil:        validUntil,
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
