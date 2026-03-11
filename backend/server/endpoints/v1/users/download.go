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
	"github.com/NicoClack/cryptic-stash/backend/ent/downloadsession"
	"github.com/NicoClack/cryptic-stash/backend/ent/user"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/gin-gonic/gin"
)

type DownloadPayload struct {
	Username          string `binding:"required,min=1,max=32"  json:"username"`
	Password          string `binding:"required,min=8,max=256" json:"password"` // #nosec G117
	AuthorizationCode string `binding:"required,len=44"        json:"authorizationCode"`
}

type DownloadResponse struct {
	Errors                      []servercommon.ErrorDetail `binding:"required" json:"errors"`
	AuthorizationCodeValidFrom  *time.Time                 `                   json:"authorizationCodeValidFrom"`
	AuthorizationCodeValidUntil *time.Time                 `                   json:"authorizationCodeValidUntil"`
	Content                     []byte                     `                   json:"content"`
	Filename                    string                     `                   json:"filename"`
}

func Download(app *servercommon.ServerApp) gin.HandlerFunc {
	clock := app.Clock

	return servercommon.NewHandler(func(ginCtx *gin.Context) error {
		body := DownloadPayload{}
		if ctxErr := servercommon.ParseBody(&body, ginCtx); ctxErr != nil {
			return ctxErr
		}
		if serverErr := servercommon.ValidateUsername(body.Username); serverErr != nil {
			return serverErr
		}
		givenAuthCodeBytes, stdErr := base64.StdEncoding.DecodeString(body.AuthorizationCode)
		if stdErr != nil {
			return servercommon.NewError(stdErr).
				SetStatus(http.StatusBadRequest).
				AddDetail(servercommon.ErrorDetail{
					Message: "auth code is not valid base64",
					Code:    "MALFORMED_AUTH_CODE",
				}).
				DisableLogging()
		}
		hashedAuthCode := sha256.Sum256(givenAuthCodeBytes)

		downloadSessionOb, stdErr := dbcommon.WithReadWriteTx(
			ginCtx.Request.Context(), app.Database,
			func(tx *ent.Tx, ctx context.Context) (*ent.DownloadSession, error) {
				downloadSessionOb, stdErr := tx.DownloadSession.Query().
					Where(downloadsession.And(
						downloadsession.HasUserWith(user.Username(body.Username)),
						downloadsession.HashedAuthCode(hashedAuthCode[:]),
					)).
					WithUser(func(userQuery *ent.UserQuery) {
						userQuery.WithStash()
						userQuery.WithMessengers()
						userQuery.WithStash()
					}).
					WithLoginAlerts(func(laQuery *ent.LoginAlertQuery) {
						laQuery.WithUserMessenger()
					}).
					First(ctx)
				if stdErr != nil {
					return nil, servercommon.SendUnauthorizedIfNotFound(stdErr)
				}
				if clock.Now().After(downloadSessionOb.ValidUntil) ||
					downloadSessionOb.Edges.User.DownloadSessionsValidFrom.After(downloadSessionOb.CreatedAt) {
					stdErr := tx.DownloadSession.DeleteOneID(downloadSessionOb.ID).Exec(ctx)
					if stdErr != nil {
						return nil, stdErr
					}
					return nil, servercommon.NewUnauthorizedError()
				}
				return downloadSessionOb, nil
			},
		)
		if stdErr != nil {
			return stdErr
		}
		if clock.Now().Before(downloadSessionOb.ValidFrom) {
			ginCtx.JSON(http.StatusBadRequest, DownloadResponse{
				Errors: []servercommon.ErrorDetail{
					{
						Message: "authorization code is not valid yet",
						Code:    "CODE_NOT_VALID_YET",
					},
				},
				AuthorizationCodeValidFrom:  &downloadSessionOb.ValidFrom,
				AuthorizationCodeValidUntil: &downloadSessionOb.ValidUntil,
			})
			return nil
		}
		if app.Core.IsUserLocked(downloadSessionOb.Edges.User) {
			return servercommon.NewUnauthorizedError()
		}

		userOb := downloadSessionOb.Edges.User
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
		stashDataKey, wrappedErr := app.Core.Decrypt(stashOb.EncryptionDataKey, stashKek)
		if wrappedErr != nil {
			return servercommon.NewUnauthorizedError().SetChild(wrappedErr)
		}

		if !app.Core.IsUserSufficientlyNotified(downloadSessionOb) {
			return servercommon.NewUnauthorizedError().SetChild(wrappedErr)
		}
		stashContent, wrappedErr := app.Core.Decrypt(stashOb.Content, stashDataKey)
		if wrappedErr != nil {
			return servercommon.NewUnauthorizedError().SetChild(wrappedErr)
		}
		stashFileName, wrappedErr := app.Core.Decrypt(stashOb.FileName, stashDataKey)
		if wrappedErr != nil {
			return servercommon.NewUnauthorizedError().SetChild(wrappedErr)
		}

		resp, stdErr := dbcommon.WithReadWriteTx(
			ginCtx.Request.Context(), app.Database,
			func(tx *ent.Tx, ctx context.Context) (*DownloadResponse, error) {
				now := clock.Now()
				authCodeValidUntil := now.Add(app.Env.USED_AUTH_CODE_VALID_FOR)
				stdErr := tx.DownloadSession.UpdateOneID(downloadSessionOb.ID).
					SetValidUntil(authCodeValidUntil).
					Exec(ctx)
				if stdErr != nil {
					return nil, stdErr
				}
				now = clock.Now()
				stdErr = tx.Stash.UpdateOneID(stashOb.ID).
					SetUpdatedAt(now).
					SetLastDownloadAt(now).
					Exec(ctx)
				if stdErr != nil {
					return nil, stdErr
				}
				_, _, wrappedErr := app.Messengers.SendUsingAll(
					&common.Message{
						Type: common.MessageDownload,
						User: userOb,
					},
					ctx,
				)
				if wrappedErr != nil {
					return nil, wrappedErr
				}

				return &DownloadResponse{
					Errors: []servercommon.ErrorDetail{},
					// TODO: set these?
					AuthorizationCodeValidFrom:  nil,
					AuthorizationCodeValidUntil: nil,
					Content:                     stashContent,
					Filename:                    string(stashFileName),
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
