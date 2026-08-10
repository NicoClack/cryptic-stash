package superuser_test

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/common/dbcommon"
	"github.com/NicoClack/cryptic-stash/backend/common/testcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/ent/session"
	"github.com/NicoClack/cryptic-stash/backend/testhelpers"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// Note: the returned passkey is a dummy that can't be used for authentication,
// but is sufficient for passing auth and testing the start-elevation endpoint.
func createSessionAndPasskey(
	t *testing.T,
	isSudo bool,
	userOb *ent.User,
	app *testhelpers.App,
) (string, *ent.Passkey) {
	t.Helper()
	dbClient := app.Database.Client()
	now := app.Clock.Now()
	credentialID := common.CryptoRandomBytes(16)

	passkeyOb := dbClient.Passkey.Create().
		SetCreatedAt(now).
		SetUpdatedAt(now).
		SetUserID(userOb.ID).
		SetName("test-passkey").
		SetAllowSudo(true).
		SetCredentialID(credentialID).
		// Hardware key
		SetCredential(webauthn.Credential{
			ID:              credentialID,
			PublicKey:       common.CryptoRandomBytes(32),
			AttestationType: "",
			// ^ "packed" would be the most realistic, but this item is excluded in the AllowedCredentials sent to the client,
			// so it makes asserts easier if we don't set it
			Flags: webauthn.CredentialFlags{
				UserPresent:  true,
				UserVerified: true,
			},
			Authenticator: webauthn.Authenticator{
				AAGUID:    common.CryptoRandomBytes(16),
				SignCount: 1,
			},
			Transport: []protocol.AuthenticatorTransport{
				protocol.USB,
				protocol.NFC,
			},
		}).
		SaveX(t.Context())

	return createSession(t, isSudo, userOb.ID, passkeyOb.ID, app), passkeyOb
}
func createSession(
	t *testing.T,
	isSudo bool,
	userID uuid.UUID,
	passkeyID uuid.UUID,
	app *testhelpers.App,
) string {
	t.Helper()

	sessionToken, stdErr := dbcommon.WithReadWriteTx(
		t.Context(), app.Database,
		func(tx *ent.Tx, ctx context.Context) ([]byte, error) {
			_, sessionToken, stdErr := app.Auth.CreateSession(
				isSudo,
				userID,
				passkeyID,
				"test-agent",
				"127.0.0.1",
				tx,
				ctx,
			)
			return sessionToken, stdErr
		},
	)
	require.NoError(t, stdErr)
	return base64.RawURLEncoding.EncodeToString(sessionToken)
}

func assertSessionElevationPasskey(
	t *testing.T,
	sessionToken string,
	expectedElevationPasskeyID uuid.UUID,
	app *testhelpers.App,
) {
	t.Helper()

	sessionOb := app.Database.Client().Session.Query().
		Where(session.HashedToken(testcommon.HashSessionToken(t, sessionToken))).
		OnlyX(t.Context())
	require.True(t, sessionOb.IsSudo)
	require.NotNil(t, sessionOb.ElevationPasskeyID)
	require.Equal(t, expectedElevationPasskeyID, *sessionOb.ElevationPasskeyID)
}
