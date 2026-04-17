package users

import (
	"context"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/common/dbcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/jobs"
)

type TempSelfUnlock1Body struct {
	Username string `binding:"required" json:"username"`
}

func TempSelfUnlock1(app *common.App) *jobs.Definition {
	return &jobs.Definition{
		ID:            "TEMP_SELF_UNLOCK",
		Version:       1,
		Weight:        1,
		NoParallelize: true,
		BodyType:      &TempSelfUnlock1Body{},
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
