package signuplinks

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/NicoClack/cryptic-stash/backend/common/dbcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/ent/signuplink"
	"github.com/NicoClack/cryptic-stash/backend/ent/user"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type CreateUserPayload struct {
	Username string `binding:"required,min=1,max=32"       json:"username"`
	Password string `binding:"required,min=8,max=256"      json:"password"` // #nosec G117
	Content  string `binding:"required,min=1,max=10000000" json:"content"`  // 10 MB but base64 encoded
	Filename string `binding:"required,min=1,max=256"      json:"filename"`
}
type CreateUserResponse struct {
	Errors []servercommon.ErrorDetail `binding:"required" json:"errors"`
}

// TODO: prevent user enumeration.
// Maybe just limiting the number of failed attempts per link would be enough?
// Can cancelling the request be used to bypass that?
func CreateUser(app *servercommon.ServerApp) gin.HandlerFunc {
	clock := app.Clock
	hashSettings := app.Env.PASSWORD_HASH_SETTINGS

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

		body := CreateUserPayload{}
		if ctxErr := servercommon.ParseBody(&body, ginCtx); ctxErr != nil {
			return ctxErr
		}
		if serverErr := servercommon.ValidateUsername(body.Username); serverErr != nil {
			return serverErr
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

		hashed := sha256.Sum256(givenCodeBytes)
		signupOb, stdErr := dbcommon.WithReadTx(
			ginCtx.Request.Context(), app.Database,
			func(tx *ent.Tx, ctx context.Context) (*ent.SignupLink, error) {
				signupOb, stdErr := tx.SignupLink.Query().
					Where(
						signuplink.ID(signupID),
						signuplink.HashedCode(hashed[:]),
					).
					Only(ctx)
				if stdErr != nil {
					return nil, servercommon.SendUnauthorizedIfNotFound(stdErr)
				}
				if signupOb.UserID != uuid.Nil || clock.Now().After(signupOb.ExpiresAt) {
					return nil, servercommon.NewUnauthorizedError()
				}

				exists, stdErr := tx.User.Query().Where(user.Username(body.Username)).Exist(ctx)
				if stdErr != nil {
					return nil, stdErr
				}
				if exists {
					return nil, servercommon.NewBadRequestError(
						"username",
						"username already exists",
						"USERNAME_ALREADY_EXISTS",
					)
				}

				return signupOb, nil
			},
		)
		if stdErr != nil {
			return stdErr
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

		resp, stdErr := dbcommon.WithReadWriteTx(
			ginCtx.Request.Context(), app.Database,
			func(tx *ent.Tx, ctx context.Context) (*CreateUserResponse, error) {
				now := clock.Now()
				userOb, stdErr := tx.User.Create().
					SetUsername(body.Username).
					SetCreatedAt(now).
					SetUpdatedAt(now).
					SetDownloadSessionsValidFrom(now).
					SetSignupLinkID(signupOb.ID).
					Save(ctx)
				if stdErr != nil {
					if ent.IsConstraintError(stdErr) && strings.Contains(stdErr.Error(), "username") {
						return nil, servercommon.NewBadRequestError(
							"username",
							"username already exists",
							"USERNAME_ALREADY_EXISTS",
						)
					}
					return nil, stdErr
				}
				stdErr = tx.Stash.Create().
					SetCreatedAt(now).
					SetUpdatedAt(now).
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
					return nil, stdErr
				}

				_, stdErr = tx.SignupLink.UpdateOneID(signupID).
					SetUser(userOb).
					SetUserAgent(ginCtx.Request.UserAgent()).
					SetIP(ginCtx.ClientIP()).
					Save(ctx)
				if stdErr != nil {
					return nil, stdErr
				}

				return &CreateUserResponse{Errors: []servercommon.ErrorDetail{}}, nil
			},
		)
		if stdErr != nil {
			return stdErr
		}

		ginCtx.JSON(http.StatusCreated, resp)
		return nil
	})
}
