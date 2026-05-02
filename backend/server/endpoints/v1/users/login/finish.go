package login

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"

	authpkg "github.com/NicoClack/cryptic-stash/backend/auth"
	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/common/dbcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/gin-gonic/gin"
)

type LoginFinishPayload struct {
	SessionID string `binding:"required,min=1,max=64" json:"sessionId"`
}

type LoginFinishResponse struct {
	Errors       []servercommon.ErrorDetail `json:"errors"`
	Session      *ent.Session               `json:"session"`
	SessionToken string                     `json:"sessionToken"`
}

func FinishLogin(app *servercommon.ServerApp) gin.HandlerFunc {
	return servercommon.NewHandler(func(ginCtx *gin.Context) error {
		body := LoginFinishPayload{}
		if serverErr := servercommon.ParseBody(&body, ginCtx); serverErr != nil {
			return serverErr
		}

		type result struct {
			session *ent.Session
			token   []byte
		}
		res, txErr := dbcommon.WithReadWriteTx(
			ginCtx.Request.Context(),
			app.Database,
			func(tx *ent.Tx, ctx context.Context) (*result, error) {
				sessionOb, token, wrappedErr := app.Auth.FinishLogin(body.SessionID, ginCtx, tx)
				if wrappedErr != nil {
					return nil, wrappedErr
				}
				return &result{session: sessionOb, token: token}, nil
			},
		)
		if txErr != nil {
			// Map common auth errors to specific HTTP responses
			if errors.Is(txErr, authpkg.ErrInvalidCeremonyID) {
				return servercommon.NewBadRequestError(
					"sessionId", "login session is missing or expired", "INVALID_LOGIN_SESSION",
				)
			}
			if errors.Is(txErr, authpkg.ErrInvalidCredential) {
				return servercommon.NewUnauthorizedError()
			}
			return servercommon.NewError(txErr.(common.WrappedError))
		}
		sessionOb := res.session
		sessionToken := base64.StdEncoding.EncodeToString(res.token)

		ginCtx.JSON(http.StatusOK, &LoginFinishResponse{
			Errors:       []servercommon.ErrorDetail{},
			Session:      sessionOb,
			SessionToken: sessionToken,
		})
		return nil
	})
}
