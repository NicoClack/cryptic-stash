package testcommon

import (
	"context"
	"fmt"

	"github.com/NicoClack/cryptic-stash/backend/core"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/jonboulle/clockwork"
)

func NewDummyUser(counter int, dbClient *ent.Client, ctx context.Context, clock clockwork.Clock) *ent.User {
	now := clock.Now()
	userOb := dbClient.User.Create().
		SetUsername(fmt.Sprintf("user%v", counter)).
		SetCreatedAt(now).
		SetUpdatedAt(now).
		SetDownloadSessionsValidFrom(now).
		SaveX(ctx)
	dbClient.Stash.Create().
		SetCreatedAt(now).
		SetUpdatedAt(now).
		SetContent([]byte{1}).
		SetFileName([]byte{1}).
		SetEncryptionDataKey(core.GenerateEncryptionKey()).
		SetPasswordSalt(core.GenerateSalt()).
		SetHashTime(0).SetHashMemory(0).SetHashThreads(0).
		SetUser(userOb).
		SaveX(ctx)
	return userOb
}
