package sudo

import (
	"context"
	"net/http"

	"github.com/NicoClack/cryptic-stash/backend/auth"
	"github.com/NicoClack/cryptic-stash/backend/common/dbcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/ent/session"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/google/uuid"
)

type StartElevationResponse struct {
	Errors            []servercommon.ErrorDetail                 `json:"errors"`
	WebAuthnSessionID uuid.UUID                                  `json:"webAuthnSessionId"`
	PublicKey         protocol.PublicKeyCredentialRequestOptions `json:"publicKey"`
}

func StartElevation(app *servercommon.ServerApp) gin.HandlerFunc {
	return servercommon.NewHandler(func(ginCtx *gin.Context) error {
		sessionID := ginCtx.MustGet("session").(*ent.Session).ID

		resp, stdErr := dbcommon.WithReadTx(
			ginCtx.Request.Context(),
			app.Database,
			func(tx *ent.Tx, ctx context.Context) (*StartElevationResponse, error) {
				// We need some edges that aren't loaded by the auth middleware
				sessionOb, stdErr := tx.Session.Query().
					Where(session.ID(sessionID)).
					WithPasskey().
					WithUser(func(userQuery *ent.UserQuery) {
						userQuery.WithPasskeys()
					}).
					Only(ctx)
				if stdErr != nil {
					return nil, stdErr
				}

				sessionID, options, wrappedErr := app.Auth.StartElevation(sessionOb, sessionOb.Edges.User)
				if wrappedErr != nil {
					return nil, wrappedErr
				}

				return &StartElevationResponse{
					Errors:            []servercommon.ErrorDetail{},
					WebAuthnSessionID: sessionID,
					PublicKey:         options,
				}, nil
			},
		)
		if stdErr != nil {
			// TODO: handle ErrNoSudoEligiblePasskeys and trigger sidevation? Require a sudo mode passkey from any group,
			// then have them elevate again with a passkey from the other group.
			// Currently a session can be unelevatable if a non-sudo passkey was used for login
			// and the only other sudo passkey is in the same group.
			return servercommon.ExpectError(
				stdErr, auth.ErrSessionAlreadyElevated,
				http.StatusConflict,
				&servercommon.ErrorDetail{
					Message: "session is already in sudo mode",
					Code:    "SESSION_ALREADY_ELEVATED",
				},
			)
		}

		ginCtx.JSON(http.StatusOK, resp)
		return nil
	})
}
