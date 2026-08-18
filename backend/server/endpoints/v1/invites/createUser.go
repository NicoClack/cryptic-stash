package invites

import (
	"context"
	"encoding/base64"
	"net/http"

	"github.com/NicoClack/cryptic-stash/backend/auth"
	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/ent/user"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/google/uuid"
)

type CreateUserPayload struct {
	protocol.CredentialCreationResponse
	CredentialName string `json:"credentialName" binding:"required,min=1,max=64"`
}
type CreateUserResponse struct {
	Errors   []servercommon.ErrorDetail `json:"errors"`
	UserID   uuid.UUID                  `json:"userId"`
	Token    string                     `json:"token"`
	Username string                     `json:"username"`
	IsSudo   bool                       `json:"isSudo"`
}

func CreateUser(app *servercommon.ServerApp) gin.HandlerFunc {
	clock := app.Clock

	return servercommon.NewObjectIDHandler(func(id uuid.UUID, ginCtx *gin.Context) error {
		body := CreateUserPayload{}
		if serverErr := servercommon.ParseBody(&body, ginCtx); serverErr != nil {
			return serverErr
		}

		parsedCredential, stdErr := body.Parse()
		if stdErr != nil {
			return servercommon.NewError(stdErr).
				SetStatus(http.StatusBadRequest).
				AddDetail(servercommon.ErrorDetail{
					Message: "invalid WebAuthn credential",
					Code:    "INVALID_CREDENTIAL",
				})
		}

		isUsernameTaken := false
		resp, stdErr := useInvite(
			id, ginCtx, app,
			func(inviteOb *ent.Invite, tx *ent.Tx, ctx context.Context) (*CreateUserResponse, error) {
				exists, stdErr := tx.User.Query().Where(user.Username(inviteOb.Email)).Exist(ctx)
				if stdErr != nil {
					return nil, stdErr
				}
				if exists {
					// It doesn't matter if this leaks the existence of the account
					// as the invite should have only been sent to the owner of this email.
					stdErr = tx.Invite.UpdateOneID(inviteOb.ID).
						SetExpiredReason("username_taken").Exec(ctx)
					if stdErr != nil {
						return nil, stdErr
					}
					isUsernameTaken = true // So the transaction commits
					return nil, nil
				}

				if inviteOb.WebAuthnSession == nil {
					return nil, servercommon.NewBadRequestError(
						"credential",
						"no active WebAuthn session, please refresh the page",
						"NO_WEBAUTHN_SESSION",
					)
				}

				passkeyOb, wrappedErr := app.Auth.FinishRegisterPasskey(
					body.CredentialName,
					true,
					false,
					inviteOb.Email,
					inviteOb.WebAuthnSession,
					parsedCredential,
					tx,
					ctx,
					func(pendingUserID uuid.UUID, tx *ent.Tx) (*ent.User, error) {
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
						_, stdErr = tx.Invite.UpdateOneID(inviteOb.ID).
							SetUser(createdUserOb).
							SetWebAuthnSession(nil).
							// ^ We don't need this anymore and the user edge prevents the invite being used twice
							SetUserAgent(new(ginCtx.Request.UserAgent())).
							SetIP(new(ginCtx.ClientIP())).
							Save(ctx)
						if stdErr != nil {
							return nil, stdErr
						}
						return createdUserOb, nil
					},
				)
				if wrappedErr != nil {
					return nil, servercommon.ExpectError(
						wrappedErr, auth.ErrWebAuthnSessionExpired, http.StatusBadRequest,
						&servercommon.ErrorDetail{
							Message: "WebAuthn session expired, please refresh the page",
							Code:    "WEBAUTHN_SESSION_EXPIRED",
						},
					).Expect(
						auth.ErrInvalidAAGUIDLength, http.StatusBadRequest,
						&servercommon.ErrorDetail{
							Message: "AAGUID must be 16 bytes",
							Code:    "INVALID_AAGUID_LENGTH",
						},
					)
				}

				sessionOb, token, wrappedErr := app.Auth.CreateSession(
					true, // TODO: this session is unrealistic
					passkeyOb.UserID,
					passkeyOb.ID,
					ginCtx.Request.UserAgent(),
					ginCtx.ClientIP(),
					tx,
					ctx,
				)
				if wrappedErr != nil {
					return nil, wrappedErr
				}

				return &CreateUserResponse{
					Errors:   []servercommon.ErrorDetail{},
					UserID:   passkeyOb.UserID,
					Token:    base64.RawURLEncoding.EncodeToString(token),
					Username: inviteOb.Email,
					IsSudo:   sessionOb.IsSudo,
				}, nil
			},
		)
		if stdErr != nil {
			return stdErr
		}
		if isUsernameTaken {
			return servercommon.NewUnauthorizedError().
				SetChild(
					common.NewErrorWithCategories("username already taken", common.ErrTypeClient),
				)
		}

		ginCtx.JSON(http.StatusCreated, resp)
		return nil
	})
}
