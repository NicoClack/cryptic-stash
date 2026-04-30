package invites

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/common/dbcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/ent/user"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/gin-gonic/gin"
)

type CreatePayload struct {
	Email         string `binding:"required,email"         json:"email"`
	InviteMessage string `binding:"required,min=1,max=500" json:"inviteMessage"`
	ExpiresIn     int64  `binding:"omitempty"              json:"expiresIn"`
}
type CreateResponse struct {
	Errors    []servercommon.ErrorDetail `binding:"required" json:"errors"`
	ID        string                     `                   json:"id"`
	Code      string                     `                   json:"code"`
	ExpiresAt time.Time                  `                   json:"expiresAt"`
}

func Create(app *servercommon.ServerApp) gin.HandlerFunc {
	clock := app.Clock
	emailMessengerType, emailMessengerVersion, _ := common.ParseVersionedType(app.Env.EMAIL_MESSENGER_TYPE)

	return servercommon.NewHandler(func(ginCtx *gin.Context) error {
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

		expiresIn := app.Env.INVITE_DEFAULT_EXPIRY
		if body.ExpiresIn > 0 {
			expiresIn = time.Duration(body.ExpiresIn) * time.Second
		}
		expiresIn = min(expiresIn, app.Env.INVITE_MAX_EXPIRY)

		resp, stdErr := dbcommon.WithReadWriteTx(
			ginCtx.Request.Context(), app.Database,
			func(tx *ent.Tx, ctx context.Context) (*CreateResponse, error) {
				code := app.Core.RandomAuthCode()
				hashed := sha256.Sum256(code)
				encodedCode := base64.RawURLEncoding.EncodeToString(code)
				now := clock.Now()
				expiresAt := now.Add(expiresIn)

				exists, stdErr := tx.User.Query().Where(user.Username(body.Email)).Exist(ctx)
				if stdErr != nil {
					return nil, stdErr
				}
				if exists {
					return nil, servercommon.NewBadRequestError(
						"email",
						"username already taken",
						"USERNAME_TAKEN",
					)
				}

				inviteOb, stdErr := tx.Invite.Create().
					SetCreatedAt(now).
					SetUpdatedAt(now).
					SetEmail(body.Email).
					SetHashedCode(hashed[:]).
					SetExpiresAt(expiresAt).
					Save(ctx)
				if stdErr != nil {
					return nil, stdErr
				}

				inMemoryUser, stdErr := newInMemoryUser(body.Email, emailMessengerType, emailMessengerVersion)
				if stdErr != nil {
					return nil, stdErr
				}

				wrappedErr := app.Messengers.Send(
					app.Env.EMAIL_MESSENGER_TYPE,
					&common.Message{
						Type:          common.MessageInvite,
						User:          inMemoryUser,
						InviteMessage: inviteMessage,
						URL: getInviteURL(
							inviteOb.ID.String(),
							encodedCode,
							app.Env.FRONTEND_BASE_URL,
						),
					},
					ctx,
				)
				if wrappedErr != nil {
					return nil, wrappedErr
				}

				return &CreateResponse{
					Errors:    []servercommon.ErrorDetail{},
					ID:        inviteOb.ID.String(),
					Code:      encodedCode,
					ExpiresAt: expiresAt,
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

func newInMemoryUser(
	email string,
	emailMessengerType string, emailMessengerVersion int,
) (*ent.User, error) {
	encodedOptions := json.RawMessage("{}")
	if emailMessengerType != "develop" {
		var stdErr error
		encodedOptions, stdErr = json.Marshal(map[string]string{
			"email": email,
		})
		if stdErr != nil {
			return nil, stdErr
		}
	}
	// TODO: validate options

	return &ent.User{
		Username: email,
		Edges: ent.UserEdges{
			Messengers: []*ent.UserMessenger{
				{
					Type:    emailMessengerType,
					Version: emailMessengerVersion,
					Enabled: true,
					Options: encodedOptions,
				},
			},
		},
	}, nil
}

func getInviteURL(
	inviteID string,
	code string,
	frontendBaseURL string,
) string {
	invitePath := fmt.Sprintf("/invites/%s/?code=%s", inviteID, url.QueryEscape(code))
	return frontendBaseURL + invitePath
}
