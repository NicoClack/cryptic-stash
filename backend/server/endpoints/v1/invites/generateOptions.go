package invites

import (
	"context"
	"encoding/base64"
	"net/http"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/google/uuid"
)

const challengeExpiry = 5 * time.Minute

type GenerateOptionsResponse struct {
	Errors    []servercommon.ErrorDetail                  `json:"errors"`
	PublicKey protocol.PublicKeyCredentialCreationOptions `json:"publicKey"`
}

func GenerateOptions(app *servercommon.ServerApp) gin.HandlerFunc {
	webAuthnApp, _ := newWebAuthnApp(app)
	pendingUserID := uuid.New()

	return servercommon.NewHandler(func(ginCtx *gin.Context) error {
		resp, stdErr := useInvite(
			ginCtx, app,
			func(inviteOb *ent.Invite, tx *ent.Tx, ctx context.Context) (*GenerateOptionsResponse, error) {
				creation, sessionOb, stdErr := webAuthnApp.BeginRegistration(&webAuthnUser{
					id:          pendingUserID[:],
					name:        inviteOb.Email,
					displayName: inviteOb.Email,
				})
				if stdErr != nil {
					return nil, stdErr
				}

				challengeBytes, stdErr := base64.RawURLEncoding.DecodeString(sessionOb.Challenge)
				if stdErr != nil {
					return nil, stdErr
				}

				_, stdErr = tx.Invite.UpdateOneID(inviteOb.ID).
					SetPendingUserID(pendingUserID).
					SetWebAuthnChallenge(challengeBytes).
					SetChallengeExpiresAt(app.Clock.Now().Add(challengeExpiry)).
					Save(ctx)
				if stdErr != nil {
					return nil, stdErr
				}

				return &GenerateOptionsResponse{
					Errors:    []servercommon.ErrorDetail{},
					PublicKey: creation.Response,
				}, nil
			},
		)
		if stdErr != nil {
			return stdErr
		}

		ginCtx.JSON(http.StatusOK, resp)
		return nil
	},
	)

	// inviteID, ctxErr := servercommon.ParseObjectID(ginCtx.Param("id"))
	// if ctxErr != nil {
	// 	return ctxErr
	// }

	// token, serverErr := servercommon.RequireAuthorizationScheme("Bearer", ginCtx)
	// if serverErr != nil {
	// 	return serverErr
	// }
	// givenCodeBytes, stdErr := base64.RawURLEncoding.DecodeString(token)
	// if stdErr != nil {
	// 	return servercommon.NewBadRequestError(
	// 		"authorization",
	// 		"malformed token",
	// 		"MALFORMED_AUTHORIZATION_TOKEN",
	// 	)
	// }

	// hashed := sha256.Sum256(givenCodeBytes)
	// pendingUserID := uuid.New()
	// resp, stdErr := dbcommon.WithReadWriteTx(
	// 	ginCtx.Request.Context(), app.Database,
	// 	func(tx *ent.Tx, ctx context.Context) (*GenerateOptionsResponse, error) {
	// 		inviteOb, stdErr := tx.Invite.Query().
	// 			Where(
	// 				invite.ID(inviteID),
	// 				invite.HashedCode(hashed[:]),
	// 			).
	// 			Only(ctx)
	// 		if stdErr != nil {
	// 			return nil, servercommon.SendUnauthorizedIfNotFound(stdErr)
	// 		}
	// 		if inviteOb.UserID != uuid.Nil ||
	// 			clock.Now().After(inviteOb.ExpiresAt) ||
	// 			inviteOb.ExpiredReason != nil {
	// 			return nil, servercommon.NewUnauthorizedError()
	// 		}

	// 		creation, sessionOb, stdErr := webAuthnApp.BeginRegistration(&webAuthnUser{
	// 			id:          pendingUserID[:],
	// 			name:        inviteOb.Email,
	// 			displayName: inviteOb.Email,
	// 		})
	// 		if stdErr != nil {
	// 			return nil, stdErr
	// 		}

	// 		challengeBytes, stdErr := base64.RawURLEncoding.DecodeString(sessionOb.Challenge)
	// 		if stdErr != nil {
	// 			return nil, stdErr
	// 		}

	// 		now := clock.Now()
	// 		_, stdErr = tx.Invite.UpdateOneID(inviteID).
	// 			SetPendingUserID(pendingUserID).
	// 			SetWebAuthnChallenge(challengeBytes).
	// 			SetChallengeExpiresAt(now.Add(challengeExpiry)).
	// 			Save(ctx)
	// 		if stdErr != nil {
	// 			return nil, stdErr
	// 		}

	// 		return &GenerateOptionsResponse{
	// 			Errors:    []servercommon.ErrorDetail{},
	// 			PublicKey: creation.Response,
	// 		}, nil
	// 	},
	// )
	// if stdErr != nil {
	// 	return stdErr
	// }

	// ginCtx.JSON(http.StatusOK, resp)
	// return nil
	// })
}
