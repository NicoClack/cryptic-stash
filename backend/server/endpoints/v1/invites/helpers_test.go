package invites_test

import (
	"crypto/sha256"
	"encoding/base64"
	"testing"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/testhelpers"
)

func createInvite(
	t *testing.T,
	app *testhelpers.App,
	email string,
	expiresAt time.Time,
) (*ent.Invite, string) {
	code := app.Core.RandomAuthCode()
	hashed := sha256.Sum256([]byte(code))
	now := app.Clock.Now()

	inviteOb := app.Database.Client().Invite.Create().
		SetEmail(email).
		SetHashedCode(hashed[:]).
		SetCreatedAt(now).
		SetUpdatedAt(now).
		SetExpiresAt(expiresAt).
		SaveX(t.Context())

	return inviteOb, base64.RawURLEncoding.EncodeToString(code)
}
