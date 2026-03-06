package users

import (
	"context"
	"encoding/base64"
	"net/http"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/common/dbcommon"
	"github.com/NicoClack/cryptic-stash/backend/core"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/ent/downloadsession"
	"github.com/NicoClack/cryptic-stash/backend/ent/user"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/gin-gonic/gin"
)

type DownloadPayload struct {
	Username          string `binding:"required,min=1,max=32,alphanum,lowercase" json:"username"`
	Password          string `binding:"required,min=8,max=256"                   json:"password"` // #nosec G117
	AuthorizationCode string `binding:"required,min=128,max=256"                 json:"authorizationCode"`
	// ^ I think the length can vary because of the base64 encoding?
}

type DownloadResponse struct {
	Errors                      []servercommon.ErrorDetail `binding:"required" json:"errors"`
	AuthorizationCodeValidFrom  *time.Time                 `                   json:"authorizationCodeValidFrom"`
	AuthorizationCodeValidUntil *time.Time                 `                   json:"authorizationCodeValidUntil"`
	Content                     []byte                     `                   json:"content"`
	Filename                    string                     `                   json:"filename"`
	Mime                        string                     `                   json:"mime"`
}

func Download(app *servercommon.ServerApp) gin.HandlerFunc {
	clock := app.Clock

	return servercommon.NewHandler(func(ginCtx *gin.Context) error {
		body := DownloadPayload{}
		if ctxErr := servercommon.ParseBody(&body, ginCtx); ctxErr != nil {
			return ctxErr
		}
		if body.Username == common.AdminUsername {
			return servercommon.NewInvalidUsernameError()
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

		downloadSessionOb, stdErr := dbcommon.WithReadWriteTx(
			ginCtx.Request.Context(), app.Database,
			func(tx *ent.Tx, ctx context.Context) (*ent.DownloadSession, error) {
				downloadSessionOb, stdErr := tx.DownloadSession.Query().
					Where(downloadsession.And(
						downloadsession.HasUserWith(user.Username(body.Username)),
						downloadsession.Code(givenAuthCodeBytes),
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
		encryptionKey := app.Core.HashPassword(
			body.Password,
			stashOb.KeySalt,
			&common.PasswordHashSettings{
				Time:    stashOb.HashTime,
				Memory:  stashOb.HashMemory,
				Threads: stashOb.HashThreads,
			},
		)
		decrypted, wrappedErr := app.Core.Decrypt(stashOb.Content, encryptionKey, stashOb.Nonce)
		if wrappedErr != nil {
			return servercommon.ExpectError(
				wrappedErr, core.ErrIncorrectPassword,
				http.StatusUnauthorized, nil,
			)
		}

		if !app.Core.IsUserSufficientlyNotified(downloadSessionOb) {
			return servercommon.NewUnauthorizedError()
		}

		return dbcommon.WithWriteTx(
			ginCtx.Request.Context(), app.Database,
			func(tx *ent.Tx, ctx context.Context) error {
				stdErr := tx.DownloadSession.UpdateOneID(downloadSessionOb.ID).
					SetValidUntil(clock.Now().Add(app.Env.USED_AUTH_CODE_VALID_FOR)).
					Exec(ctx)
				if stdErr != nil {
					return stdErr
				}
				_, _, wrappedErr := app.Messengers.SendUsingAll(
					&common.Message{
						Type: common.MessageDownload,
						User: userOb,
					},
					ctx,
				)
				if wrappedErr != nil {
					return wrappedErr
				}

				ginCtx.JSON(http.StatusOK, DownloadResponse{
					Errors: []servercommon.ErrorDetail{},
					// TODO: set these?
					AuthorizationCodeValidFrom:  nil,
					AuthorizationCodeValidUntil: nil,
					Content:                     decrypted,
					Filename:                    stashOb.FileName,
					Mime:                        stashOb.Mime,
				})
				return nil
			},
		)
	})
}
