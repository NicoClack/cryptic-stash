package twofactoractions

import (
	"context"
	"net/http"

	"github.com/NicoClack/cryptic-stash/backend/common/dbcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/NicoClack/cryptic-stash/backend/twofactoractions"
	"github.com/gin-gonic/gin"
)

type ConfirmPayload struct {
	Code string `binding:"required,min=9,max=9,alphanum,lowercase" json:"code"`
}

type ConfirmResponse struct {
	Errors []servercommon.ErrorDetail `binding:"required" json:"errors"`
}

func Confirm(app *servercommon.ServerApp) gin.HandlerFunc {
	return servercommon.NewHandler(func(ginCtx *gin.Context) error {
		body := ConfirmPayload{}
		if serverErr := servercommon.ParseBody(&body, ginCtx); serverErr != nil {
			return serverErr
		}
		actionID, serverErr := servercommon.ParseObjectID(ginCtx.Param("id"))
		if serverErr != nil {
			return serverErr
		}

		return dbcommon.WithWriteTx(
			ginCtx.Request.Context(), app.Database,
			func(tx *ent.Tx, ctx context.Context) error {
				_, wrappedErr := app.TwoFactorActions.Confirm(actionID, body.Code, ctx)
				if wrappedErr != nil {
					return servercommon.ExpectAnyOfErrors(
						wrappedErr,
						[]error{
							twofactoractions.ErrNotFound,
							twofactoractions.ErrExpired,
							twofactoractions.ErrWrongCode,
						},
						http.StatusUnauthorized, nil,
					)
				}
				ginCtx.JSON(http.StatusOK, ConfirmResponse{
					Errors: []servercommon.ErrorDetail{},
				})
				return nil
			},
		)
	})
}
