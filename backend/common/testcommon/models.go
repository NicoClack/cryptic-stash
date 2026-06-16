package testcommon

import (
	"context"
	"fmt"

	"github.com/NicoClack/cryptic-stash/backend/core"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/ent/schema"
	"github.com/jonboulle/clockwork"
)

func NewDummyUser(counter int, dbClient *ent.Client, ctx context.Context, clock clockwork.Clock) *ent.User {
	now := clock.Now()
	userOb := dbClient.User.Create().
		SetUsername(fmt.Sprintf("user%v", counter)).
		SetCreatedAt(now).
		SetUpdatedAt(now).
		SaveX(ctx)
	dbClient.Stash.Create().
		SetCreatedAt(now).
		SetUpdatedAt(now).
		SetPublicName("dummy stash").
		SetContent([]byte{1}).
		SetFileName([]byte{1}).
		SetEncryptionDataKey(schema.EncryptedField[[]byte]{
			Decrypted: core.GenerateEncryptionKey(),
			// ^ Normally this would be encrypted first using a password but it doesn't matter for this
			KeyName: "stash_1",
		}).
		SetPasswordSalt(core.GenerateSalt()).
		SetHashTime(0).SetHashMemory(0).SetHashThreads(0).
		SetUser(userOb).
		SetDownloadSessionsValidFrom(now).
		SaveX(ctx)
	return userOb
}
