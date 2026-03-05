package testcommon

import (
	"context"
	"fmt"

	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/jonboulle/clockwork"
)

func NewDummyUser(counter int, dbClient *ent.Client, ctx context.Context, clock clockwork.Clock) *ent.User {
	userOb := dbClient.User.Create().
		SetUsername(fmt.Sprintf("user%v", counter)).
		SetDownloadSessionsValidFrom(clock.Now()).
		SaveX(ctx)
	dbClient.Stash.Create().
		SetContent([]byte{1}).
		SetFileName("file.zip").
		SetMime("application/zip").
		SetNonce([]byte{1}).
		SetKeySalt([]byte{1}).
		SetHashTime(0).SetHashMemory(0).SetHashThreads(0).
		SetUser(userOb).
		SaveX(ctx)
	return userOb
}
