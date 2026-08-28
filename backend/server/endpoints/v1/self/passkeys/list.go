package passkeys

import (
	"context"
	"net/http"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common/dbcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/ent/passkey"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ListInfo struct {
	ID              uuid.UUID  `json:"id"`
	CreatedAt       time.Time  `json:"createdAt"`
	LastUsedAt      *time.Time `json:"lastUsedAt"`
	Name            string     `json:"name"`
	AllowSudo       bool       `json:"allowSudo"`
	IsSessionFirst  bool       `json:"isSessionFirst"`
	IsSessionSecond bool       `json:"isSessionSecond"`
}
type ListResponse struct {
	Errors              []servercommon.ErrorDetail `json:"errors"`
	FirstGroupPasskeys  []ListInfo                 `json:"firstGroupPasskeys"`
	SecondGroupPasskeys []ListInfo                 `json:"secondGroupPasskeys"`
}

func List(app *servercommon.ServerApp) gin.HandlerFunc {
	return servercommon.NewHandler(func(ginCtx *gin.Context) error {
		sessionOb := ginCtx.MustGet("session").(*ent.Session)
		userID := ginCtx.MustGet("user").(*ent.User).ID

		resp, stdErr := dbcommon.WithReadTx(
			ginCtx.Request.Context(),
			app.Database,
			func(tx *ent.Tx, ctx context.Context) (*ListResponse, error) {
				passkeyObs, stdErr := tx.Passkey.Query().
					Where(passkey.UserID(userID)).
					All(ctx)
				if stdErr != nil {
					return nil, stdErr
				}

				firstGroup, secondGroup := []*ent.Passkey{}, []*ent.Passkey{}
				for _, passkeyOb := range passkeyObs {
					if passkeyOb.IsSecondGroup {
						secondGroup = append(secondGroup, passkeyOb)
					} else {
						firstGroup = append(firstGroup, passkeyOb)
					}
				}

				getInfos := func(passkeyObs []*ent.Passkey) []ListInfo {
					items := make([]ListInfo, 0, len(passkeyObs))
					for _, passkeyOb := range passkeyObs {
						items = append(items, ListInfo{
							ID:             passkeyOb.ID,
							CreatedAt:      passkeyOb.CreatedAt,
							LastUsedAt:     passkeyOb.LastUsedAt,
							Name:           passkeyOb.Name,
							AllowSudo:      passkeyOb.AllowSudo,
							IsSessionFirst: sessionOb.PasskeyID == passkeyOb.ID,
							IsSessionSecond: sessionOb.ElevationPasskeyID != nil &&
								*sessionOb.ElevationPasskeyID == passkeyOb.ID,
						})
					}
					return items
				}

				return &ListResponse{
					Errors:              []servercommon.ErrorDetail{},
					FirstGroupPasskeys:  getInfos(firstGroup),
					SecondGroupPasskeys: getInfos(secondGroup),
				}, nil
			},
		)
		if stdErr != nil {
			return stdErr
		}

		ginCtx.JSON(http.StatusOK, resp)
		return nil
	})
}
