package users

import (
	"time"

	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/gin-gonic/gin"
)

const MAX_SELF_LOCK_DURATION = 14 * (24 * time.Hour)

type SelfLockPayload struct {
	Username string    `binding:"required,min=1,max=32"  json:"username"`
	Password string    `binding:"required,min=8,max=256" json:"password"` // #nosec G117
	Until    time.Time `binding:"required"               json:"until"`
}
type SelfLockResponse struct {
	Errors            []servercommon.ErrorDetail `binding:"required" json:"errors"`
	TwoFactorActionID string                     `                   json:"twoFactorActionId"`
}

func SelfLock(app *servercommon.ServerApp) gin.HandlerFunc {
	return servercommon.NewHandler(func(ginCtx *gin.Context) error {
		body := SelfLockPayload{}
		if ctxErr := servercommon.ParseBody(&body, ginCtx); ctxErr != nil {
			return ctxErr
		}
		if serverErr := servercommon.ValidateUserEmail(body.Username); serverErr != nil {
			return serverErr
		}
		// until := clock.Now().Add(
		// 	min(
		// 		body.Until.Sub(clock.Now()), // Convert to duration
		// 		MAX_SELF_LOCK_DURATION,
		// 	),
		// )

		panic("not implemented")
		return nil
	})
}
