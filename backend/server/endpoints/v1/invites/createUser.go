package invites

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/common/dbcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/ent/invite"
	"github.com/NicoClack/cryptic-stash/backend/ent/user"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
)

var ErrUsernameTaken = servercommon.NewUnauthorizedError().
	SetChild(
		common.NewErrorWithCategories("username already taken", common.ErrTypeClient),
	)

type CreateUserPayload struct {
	Credential json.RawMessage `json:"credential" binding:"required"`
}
type CreateUserResponse struct {
	Errors []servercommon.ErrorDetail `binding:"required" json:"errors"`
}

func CreateUser(app *servercommon.ServerApp) gin.HandlerFunc {
	clock := app.Clock
	webAuthnOb, rpID := newWebAuthnApp(app)

	return servercommon.NewHandler(func(ginCtx *gin.Context) error {
		inviteID, ctxErr := servercommon.ParseObjectID(ginCtx.Param("id"))
		if ctxErr != nil {
			return ctxErr
		}

		token, serverErr := servercommon.RequireAuthorizationScheme("Bearer", ginCtx)
		if serverErr != nil {
			return serverErr
		}
		givenCodeBytes, stdErr := base64.RawURLEncoding.DecodeString(token)
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

		hashed := sha256.Sum256(givenCodeBytes)
		resp, stdErr := dbcommon.WithReadWriteTx(
			ginCtx.Request.Context(), app.Database,
			func(tx *ent.Tx, ctx context.Context) (*CreateUserResponse, error) {
				inviteOb, stdErr := tx.Invite.Query().
					Where(
						invite.ID(inviteID),
						invite.HashedCode(hashed[:]),
					).
					Only(ctx)
				if stdErr != nil {
					return nil, servercommon.SendUnauthorizedIfNotFound(stdErr)
				}
				if inviteOb.UserID != uuid.Nil || // Already used
					clock.Now().After(inviteOb.ExpiresAt) ||
					inviteOb.ExpiredReason != nil { // Expired for another reason
					return nil, servercommon.NewUnauthorizedError()
				}

				exists, stdErr := tx.User.Query().Where(user.Username(inviteOb.Email)).Exist(ctx)
				if stdErr != nil {
					return nil, stdErr
				}
				if exists {
					// It doesn't matter if this leaks the existence of the account as the invite should have only
					// been sent to the owner of this email.
					stdErr = tx.Invite.UpdateOneID(inviteID).
						SetExpiredReason("username_taken").Exec(ctx)
					if stdErr != nil {
						return nil, stdErr
					}
					return nil, ErrUsernameTaken.Clone()
				}

				if inviteOb.WebAuthnChallenge == nil || inviteOb.ChallengeExpiresAt == nil {
					return nil, servercommon.NewBadRequestError(
						"credential",
						"no active WebAuthn challenge; request registration options first",
						"NO_WEBAUTHN_CHALLENGE",
					)
				}
				if inviteOb.PendingUserID == nil {
					return nil, servercommon.NewBadRequestError(
						"credential",
						"no pending user ID; request registration options first",
						"NO_PENDING_USER_ID",
					)
				}
				if clock.Now().After(*inviteOb.ChallengeExpiresAt) {
					return nil, servercommon.NewBadRequestError(
						"credential",
						"WebAuthn challenge expired; request new registration options",
						"WEBAUTHN_CHALLENGE_EXPIRED",
					)
				}

				pendingUserID := *inviteOb.PendingUserID
				sessionOb := webauthn.SessionData{
					Challenge:        base64.RawURLEncoding.EncodeToString(*inviteOb.WebAuthnChallenge),
					RelyingPartyID:   rpID,
					UserID:           pendingUserID[:],
					UserVerification: protocol.VerificationPreferred,
				}

				parsedCredential, stdErr := protocol.ParseCredentialCreationResponseBytes(body.Credential)
				if stdErr != nil {
					return nil, servercommon.NewBadRequestError(
						"credential",
						"invalid WebAuthn credential",
						"INVALID_WEBAUTHN_CREDENTIAL",
					)
				}

				registrationUserOb := &webAuthnUser{
					id:          pendingUserID[:],
					name:        inviteOb.Email,
					displayName: inviteOb.Email,
				}
				credentialOb, stdErr := webAuthnOb.CreateCredential(registrationUserOb, sessionOb, parsedCredential)
				if stdErr != nil {
					return nil, servercommon.NewBadRequestError(
						"credential",
						"WebAuthn credential verification failed",
						"WEBAUTHN_VERIFICATION_FAILED",
					)
				}

				aaguid := credentialOb.Authenticator.AAGUID
				if len(aaguid) != 16 {
					aaguid = make([]byte, 16)
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
					if ent.IsConstraintError(stdErr) {
						return nil, ErrUsernameTaken.Clone()
					}
					return nil, stdErr
				}

				_, stdErr = tx.Passkey.Create().
					SetUserID(createdUserOb.ID).
					SetName("Passkey").
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

				_, stdErr = tx.Invite.UpdateOneID(inviteID).
					SetUser(createdUserOb).
					ClearWebAuthnChallenge().
					ClearChallengeExpiresAt().
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
