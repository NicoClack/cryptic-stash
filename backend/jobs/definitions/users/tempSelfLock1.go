package users

import (
	"context"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/common/dbcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/jobs"
)

type TempSelfLock1Body struct {
	Username string    `binding:"required" json:"username"`
	Until    time.Time `binding:"required" json:"until"`
}

func TempSelfLock1(app *common.App) *jobs.Definition {
	return &jobs.Definition{
		ID:            "TEMP_SELF_LOCK",
		Version:       1,
		Priority:      jobs.HighPriority,
		Weight:        1,
		NoParallelize: true,
		BodyType:      &TempSelfLock1Body{},
		Handler: func(jobCtx *jobs.Context) error {
			body := &TempSelfLock1Body{}
			jobErr := jobCtx.Decode(body)
			if jobErr != nil {
				return jobErr
			}

			return dbcommon.WithWriteTx(
				jobCtx.Context, app.Database,
				func(tx *ent.Tx, ctx context.Context) error {
					panic("not implemented")
					return nil
				},
			)
		},
	}
}
