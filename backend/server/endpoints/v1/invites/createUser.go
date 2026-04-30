package invites

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/ent/user"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/google/uuid"
)

var ErrUsernameTaken = servercommon.NewUnauthorizedError().
	SetChild(
		common.NewErrorWithCategories("username already taken", common.ErrTypeClient),
	)

type CreateUserPayload struct {
	Credential     json.RawMessage `json:"credential"     binding:"required"`
	CredentialName string          `json:"credentialName" binding:"required,min=1,max=64"`
}
type CreateUserResponse struct {
	Errors []servercommon.ErrorDetail `binding:"required" json:"errors"`
}

func CreateUser(app *servercommon.ServerApp) gin.HandlerFunc {
	clock := app.Clock
	webAuthnApp, _ := newWebAuthnApp(app)

	return servercommon.NewHandler(func(ginCtx *gin.Context) error {
		body := CreateUserPayload{}
		if serverErr := servercommon.ParseBody(&body, ginCtx); serverErr != nil {
			return serverErr
		}
		parsedCredential, stdErr := protocol.ParseCredentialCreationResponseBytes(body.Credential)
		if stdErr != nil {
			return servercommon.NewBadRequestError(
				"credential",
				"malformed WebAuthn credential",
				"MALFORMED_CREDENTIAL",
			)
		}

		resp, stdErr := useInvite(
			ginCtx, app,
			func(inviteOb *ent.Invite, tx *ent.Tx, ctx context.Context) (*CreateUserResponse, error) {
				exists, stdErr := tx.User.Query().Where(user.Username(inviteOb.Email)).Exist(ctx)
				if stdErr != nil {
					return nil, stdErr
				}
				if exists {
					// It doesn't matter if this leaks the existence of the account as the invite should have only
					// been sent to the owner of this email.
					stdErr = tx.Invite.UpdateOneID(inviteOb.ID).
						SetExpiredReason("username_taken").Exec(ctx)
					if stdErr != nil {
						return nil, stdErr
					}
					return nil, ErrUsernameTaken.Clone()
				}

				webAuthnSession := inviteOb.WebAuthnSession
				if webAuthnSession == nil {
					return nil, servercommon.NewBadRequestError(
						"credential",
						"no active WebAuthn session, please refresh the page",
						"NO_WEBAUTHN_SESSION",
					)
				}
				if !webAuthnSession.Expires.IsZero() && clock.Now().After(webAuthnSession.Expires) {
					return nil, servercommon.NewBadRequestError(
						"credential",
						"WebAuthn session expired, please refresh the page",
						"WEBAUTHN_SESSION_EXPIRED",
					)
				}
				credentialOb, stdErr := webAuthnApp.CreateCredential(
					&webAuthnUser{
						id:          webAuthnSession.UserID,
						name:        inviteOb.Email,
						displayName: inviteOb.Email,
					},
					*webAuthnSession,
					parsedCredential,
				)
				if stdErr != nil {
					return nil, servercommon.NewBadRequestError(
						"credential",
						"invalid WebAuthn credential",
						"INVALID_CREDENTIAL",
					)
				}

				aaguid := credentialOb.Authenticator.AAGUID
				if len(aaguid) == 0 {
					aaguid = make([]byte, 16)
				} else if len(aaguid) != 16 {
					return nil, servercommon.NewBadRequestError(
						"credential",
						"AAGUID must be 16 bytes",
						"INVALID_AAGUID_LENGTH",
					)
				}
				pendingUserID, stdErr := uuid.FromBytes(webAuthnSession.UserID)
				if stdErr != nil {
					return nil, stdErr
				}
				now := clock.Now()

				createdUserOb, stdErr := tx.User.Create().
					SetID(pendingUserID).
					SetUsername(inviteOb.Email).
					SetCreatedAt(now).
					SetUpdatedAt(now).
					SetInviteID(inviteOb.ID).
					Save(ctx)
				if stdErr != nil {
					return nil, stdErr
				}
				_, stdErr = tx.Passkey.Create().
					SetUserID(createdUserOb.ID).
					SetName(body.CredentialName).
					SetCredentialID(credentialOb.ID).
					SetPublicKey(credentialOb.PublicKey).
					SetAaguid(aaguid).
					SetSignCount(credentialOb.Authenticator.SignCount).
					SetCreatedAt(now).
					SetUpdatedAt(now).
					Save(ctx)
				if stdErr != nil {
					return nil, stdErr
				}

				_, stdErr = tx.Invite.UpdateOneID(inviteOb.ID).
					SetUser(createdUserOb).
					ClearWebAuthnSession().
					SetUserAgent(ginCtx.Request.UserAgent()).
					SetIP(ginCtx.ClientIP()).
					Save(ctx)
				if stdErr != nil {
					return nil, stdErr
				}

				return &CreateUserResponse{
					Errors: []servercommon.ErrorDetail{},
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
