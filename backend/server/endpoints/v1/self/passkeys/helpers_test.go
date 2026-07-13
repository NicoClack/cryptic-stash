package passkeys_test

import (
	"context"
	"encoding/base64"
	"testing"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common/dbcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/testhelpers"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/stretchr/testify/require"
)

func createDummyPasskey(
	t *testing.T,
	ctx context.Context,
	userOb *ent.User,
	db *ent.Client,
) *ent.Passkey {
	t.Helper()

	credential := webauthn.Credential{
		ID:        []byte("dummy-credential-id"),
		PublicKey: []byte("dummy-public-key"),
	}
	passkeyOb, stdErr := db.Passkey.Create().
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		SetUserID(userOb.ID).
		SetName("Test passkey").
		SetAllowSudo(false).
		SetIsSecondGroup(false).
		SetCredentialID(credential.ID).
		SetCredential(credential).
		Save(ctx)
	require.NoError(t, stdErr)
	return passkeyOb
}

func createSuperSession(t *testing.T, app *testhelpers.App, passkeyOb *ent.Passkey) string {
	t.Helper()

	token, stdErr := dbcommon.WithReadWriteTx(
		t.Context(),
		app.Database,
		func(tx *ent.Tx, ctx context.Context) ([]byte, error) {
			_, token, stdErr := app.Auth.CreateSession(
				true,
				passkeyOb.UserID,
				passkeyOb.ID,
				"Mozilla/5.0",
				"127.0.0.1",
				tx,
				ctx,
			)
			return token, stdErr
		},
	)
	require.NoError(t, stdErr)

	return base64.RawURLEncoding.EncodeToString(token)
}
