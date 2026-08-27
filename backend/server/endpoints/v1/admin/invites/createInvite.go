package invites

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/common/dbcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/invites"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type CreatePayload struct {
	Email         string `binding:"required,email,max=128" json:"email"`
	InviteMessage string `binding:"required,min=1,max=500" json:"inviteMessage"`
	ExpiresIn     int64  `binding:"required,gt=0"          json:"expiresIn"`
}
type CreateResponse struct {
	Errors    []servercommon.ErrorDetail `binding:"required" json:"errors"`
	ID        uuid.UUID                  `                   json:"id"`
	Code      string                     `                   json:"code"`
	ExpiresAt time.Time                  `                   json:"expiresAt"`
}

func Create(app *servercommon.ServerApp) gin.HandlerFunc {
	return servercommon.NewHandler(func(ginCtx *gin.Context) error {
		actor := ginCtx.MustGet("actor").(*common.Actor)

		body := CreatePayload{}
		if serverErr := servercommon.ParseBody(&body, ginCtx); serverErr != nil {
			return serverErr
		}
		if serverErr := servercommon.ValidateUserEmail(body.Email); serverErr != nil {
			return serverErr
		}
		inviteMessage := strings.TrimSpace(body.InviteMessage)
		if inviteMessage == "" {
			return servercommon.NewBadRequestError(
				"inviteMessage",
				"invite message is required",
				"INVITE_MESSAGE_REQUIRED",
			)
		}

		resp, stdErr := dbcommon.WithReadWriteTx(
			ginCtx.Request.Context(), app.Database,
			func(tx *ent.Tx, ctx context.Context) (*CreateResponse, error) {
				inviteOb, encodedCode, wrappedErr := app.Invites.CreateInvite(
					body.Email,
					inviteMessage,
					time.Duration(body.ExpiresIn)*time.Second,
					actor,
					tx,
					ctx,
				)
				if wrappedErr != nil {
					return nil, servercommon.ExpectError(
						wrappedErr, invites.ErrUsernameTaken,
						http.StatusBadRequest,
						&servercommon.ErrorDetail{
							Message: "email: username already taken",
							Code:    "USERNAME_TAKEN",
						},
					)
				}

				return &CreateResponse{
					Errors:    []servercommon.ErrorDetail{},
					ID:        inviteOb.ID,
					Code:      encodedCode,
					ExpiresAt: inviteOb.ExpiresAt,
				}, nil
			},
		)
		if stdErr != nil {
			return stdErr
		}

		ginCtx.JSON(http.StatusCreated, resp)
		return nil
	})
}
