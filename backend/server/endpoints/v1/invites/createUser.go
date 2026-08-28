package invites

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"

	"github.com/NicoClack/cryptic-stash/backend/auth"
	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/common/dbcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/invites"
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
	return newInviteTokenHandler(func(id uuid.UUID, code []byte, ginCtx *gin.Context) error {
		actor := &common.Actor{
			IP:        ginCtx.ClientIP(),
			UserAgent: ginCtx.Request.UserAgent(),
		}

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
		resp, stdErr := dbcommon.WithReadWriteTx(
			ginCtx.Request.Context(), app.Database,
			func(tx *ent.Tx, ctx context.Context) (*CreateUserResponse, error) {
				userOb, _, sessionOb, token, wrappedErr := app.Invites.CreateUser(
					id, code, body.CredentialName, parsedCredential, actor, tx, ctx,
				)
				if wrappedErr != nil {
					if errors.Is(wrappedErr, invites.ErrUsernameTaken) {
						isUsernameTaken = true // So the transaction commits
						return nil, nil
					}
					return nil, servercommon.ExpectAnyOfErrors(
						wrappedErr,
						[]error{
							// TODO: merge these errors in the service?
							auth.ErrWebAuthnSessionExpired,
							invites.ErrNoWebAuthnSession,
						},
						http.StatusBadRequest,
						&servercommon.ErrorDetail{
							Message: "WebAuthn session expired, please refresh the page",
							Code:    "NO_WEBAUTHN_SESSION",
						},
					).Expect(
						auth.ErrInvalidAAGUIDLength,
						http.StatusBadRequest,
						&servercommon.ErrorDetail{
							Message: "AAGUID must be 16 bytes",
							Code:    "INVALID_AAGUID_LENGTH",
						},
					).ExpectAnyOf(
						[]error{
							invites.ErrInviteNotFound,
							invites.ErrInviteUsed,
							invites.ErrInviteExpired,
						},
						http.StatusUnauthorized,
						nil,
					)
				}

				return &CreateUserResponse{
					Errors:   []servercommon.ErrorDetail{},
					UserID:   userOb.ID,
					Token:    base64.RawURLEncoding.EncodeToString(token),
					Username: userOb.Username,
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
