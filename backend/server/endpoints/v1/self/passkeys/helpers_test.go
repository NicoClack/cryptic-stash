package passkeys_test

import (
	"context"
	"encoding/base64"
	"testing"
	"time"

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

func createPasskey(
	t *testing.T,
	name string,
	allowSudo bool,
	isSecondGroup bool,
	userID uuid.UUID,
	db *ent.Client,
) *ent.Passkey {
	t.Helper()

	credentialID := common.CryptoRandomBytes(16)
	passkeyOb, stdErr := db.Passkey.Create().
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		SetUserID(userID).
		SetName(name).
		SetAllowSudo(allowSudo).
		SetIsSecondGroup(isSecondGroup).
		SetCredentialID(credentialID).
		SetCredential(webauthn.Credential{
			ID:              credentialID,
			PublicKey:       common.CryptoRandomBytes(32),
			AttestationType: "",
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
		Save(t.Context())
	require.NoError(t, stdErr)
	return passkeyOb
}

// Note: this creates unrealistic sudo sessions unlike createSessionWithElevationPasskey
func createSession(
	t *testing.T,
	isSudo bool,
	userID uuid.UUID,
	passkeyID uuid.UUID,
	app *testhelpers.App,
) string {
	t.Helper()

	token, stdErr := dbcommon.WithReadWriteTx(
		t.Context(), app.Database,
		func(tx *ent.Tx, ctx context.Context) ([]byte, error) {
			_, token, stdErr := app.Auth.CreateSession(
				isSudo,
				userID,
				passkeyID,
				"test-agent",
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

func createSessionWithElevationPasskey(
	t *testing.T,
	userID uuid.UUID,
	passkeyID uuid.UUID,
	elevationPasskeyID uuid.UUID,
	app *testhelpers.App,
) string {
	t.Helper()

	sessionToken := createSession(t, true, userID, passkeyID, app)
	app.Database.Client().Session.Update().
		Where(session.HashedToken(testcommon.HashSessionToken(t, sessionToken))).
		SetElevationPasskeyID(elevationPasskeyID).
		ExecX(t.Context())
	return sessionToken
}
