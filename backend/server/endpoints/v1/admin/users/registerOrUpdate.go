package users

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

type RegisterPayload struct {
	Username string `binding:"required,min=1,max=32,alphanum,lowercase" json:"username"`
	Password string `binding:"required,min=8,max=256"                   json:"password"` // #nosec G117
	Content  string `binding:"required,min=1,max=100000000"             json:"content"`  // 100 MB but base64 encoded
	Filename string `binding:"required,min=1,max=256"                   json:"filename"`
}
type RegisterOrUpdateResponse struct {
	Errors []servercommon.ErrorDetail `binding:"required" json:"errors"`
}

// TODO: rename and create new update endpoint
// TODO: split creating/updating stash into own endpoint?
func RegisterOrUpdate(app *servercommon.ServerApp) gin.HandlerFunc {
	hashSettings := app.Env.PASSWORD_HASH_SETTINGS

	return servercommon.NewHandler(func(ginCtx *gin.Context) error {
		body := RegisterPayload{}
		if ctxErr := servercommon.ParseBody(&body, ginCtx); ctxErr != nil {
			return ctxErr
		}
		if body.Username == common.AdminUsername {
			return servercommon.NewInvalidUsernameError()
		}
		contentBytes, stdErr := base64.StdEncoding.DecodeString(body.Content)
		if stdErr != nil {
			return servercommon.NewError(stdErr).
				SetStatus(http.StatusBadRequest).
				AddDetail(servercommon.ErrorDetail{
					Message: "content is not valid base64",
					Code:    "MALFORMED_CONTENT",
				}).
				DisableLogging()
		}

		salt := app.Core.GenerateSalt()
		encryptionKey := app.Core.HashPassword(body.Password, salt, hashSettings)
		stashDataKey := app.Core.GenerateSalt()
		encrypted, wrappedErr := app.Core.Encrypt(contentBytes, stashDataKey)
		if wrappedErr != nil {
			return wrappedErr
		}
		encryptedDataKey, wrappedErr := app.Core.Encrypt(stashDataKey, encryptionKey)
		if wrappedErr != nil {
			return wrappedErr
		}

		return dbcommon.WithWriteTx(
			ginCtx.Request.Context(), app.Database,
			func(tx *ent.Tx, ctx context.Context) error {
				userOb, stdErr := tx.User.Create().
					SetUsername(body.Username).
					SetDownloadSessionsValidFrom(app.Clock.Now()).
					Save(ctx)
				if stdErr != nil {
					return stdErr
				}
				stdErr = tx.Stash.Create().
					SetContent(encrypted).
					SetFileName([]byte(body.Filename)).
					SetEncryptionDataKey(encryptedDataKey).
					SetPasswordSalt(salt).
					SetHashTime(hashSettings.Time).
					SetHashMemory(hashSettings.Memory).
					SetHashThreads(hashSettings.Threads).
					SetUser(userOb).
					Exec(ctx)
				if stdErr != nil {
					return stdErr
				}

				wrappedErr := app.Core.InvalidateUserDownloadSessions(userOb.ID, ctx)
				if wrappedErr != nil {
					return wrappedErr
				}

				_, _, wrappedErr = app.Messengers.SendUsingAll(
					&common.Message{
						Type: common.MessageUserUpdate,
						User: userOb,
					},
					ctx,
				)
				if wrappedErr != nil {
					return wrappedErr
				}

				ginCtx.JSON(http.StatusCreated, RegisterOrUpdateResponse{
					Errors: []servercommon.ErrorDetail{},
				})
				return nil
			},
		)
	})
}
