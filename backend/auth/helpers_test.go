package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/auth"
	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/common/dbcommon"
	"github.com/NicoClack/cryptic-stash/backend/common/testcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/tempkeyvalue"
	"github.com/descope/virtualwebauthn"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func createUser(t *testing.T, dbClient *ent.Client) *ent.User {
	t.Helper()

	now := time.Now()
	return dbClient.User.Create().
		SetUsername("user1").
		SetCreatedAt(now).
		SetUpdatedAt(now).
		SaveX(t.Context())
}

func createDummyPasskey(
	t *testing.T,
	dbClient *ent.Client,
	userID uuid.UUID,
	name string,
	allowSudo bool,
	isSecondGroup bool,
) *ent.Passkey {
	t.Helper()

	credentialID := common.CryptoRandomBytes(16)
	return dbClient.Passkey.Create().
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
		SaveX(t.Context())
}

// Note: the returned session can't be validated
func createDummySession(
	t *testing.T,
	dbClient *ent.Client,
	userID uuid.UUID,
	passkeyID uuid.UUID,
) *ent.Session {
	t.Helper()

	return dbClient.Session.Create().
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		SetUserID(userID).
		SetPasskeyID(passkeyID).
		SetIsSudo(false).
		SetHashedToken(common.CryptoRandomBytes(32)).
		SetExpiresAt(time.Now().Add(time.Hour)).
		SetUserAgent("test-agent").
		SetIP("127.0.0.1").
		SaveX(t.Context())
}

type minimalTempKeyValueService struct {
	registry *tempkeyvalue.Registry
}

// Creates a key/value service with only the definitions that the auth package needs
func newMinimalTempKeyValueService(t *testing.T) *minimalTempKeyValueService {
	t.Helper()

	registry := tempkeyvalue.NewRegistry()
	registry.Register(&tempkeyvalue.Definition{
		Name: auth.WebAuthnSessionStoreName,
		Type: &webauthn.SessionData{},
	})
	return &minimalTempKeyValueService{registry: registry}
}

func (tempKV *minimalTempKeyValueService) Get(storeName string, key string, ptr any) bool {
	return tempKV.registry.Get(storeName, key, ptr, time.Now())
}
func (tempKV *minimalTempKeyValueService) Set(storeName string, key string, value any, expiresAt time.Time) {
	tempKV.registry.Set(storeName, key, value, expiresAt)
}
func (tempKV *minimalTempKeyValueService) Delete(storeName string, key string) {
	tempKV.registry.Delete(storeName, key)
}
func (tempKV *minimalTempKeyValueService) Prune(storeName string) {
	tempKV.registry.Prune(storeName, time.Now())
}
func (tempKV *minimalTempKeyValueService) PruneAll() {
	tempKV.registry.PruneAll(time.Now())
}

// newVirtualAuthenticator returns a virtual authenticator holding a single
// WebAuthn credential, associated with the given user handle.
func newVirtualAuthenticator(userID uuid.UUID) (
	virtualwebauthn.Authenticator,
	virtualwebauthn.Credential,
) {
	vAuthenticator := virtualwebauthn.NewAuthenticator()
	vAuthenticator.Options.Transports = []virtualwebauthn.Transport{
		virtualwebauthn.TransportUSB,
		virtualwebauthn.TransportNFC,
	}
	vAuthenticator.Options.UserHandle = userID[:]
	credential := virtualwebauthn.NewCredential(virtualwebauthn.KeyTypeEC2)
	vAuthenticator.AddCredential(credential)
	return vAuthenticator, credential
}

// Calls auth.StartRegisterPasskey and sets up a virtual authenticator to use in a FinishRegisterPasskey call
func startRegistrationCeremony(
	t *testing.T,
	webAuthnApp *webauthn.WebAuthn,
	userOb *ent.User,
) (
	*webauthn.SessionData,
	*protocol.ParsedCredentialCreationData,
	virtualwebauthn.Authenticator,
	virtualwebauthn.Credential,
) {
	t.Helper()

	vAuthenticator, credential := newVirtualAuthenticator(userOb.ID)

	options, sessionData, wrappedErr := auth.StartRegisterPasskey(
		&auth.RealWebAuthnUser{User: userOb},
		webAuthnApp,
	)
	require.NoError(t, wrappedErr)

	credentialJSON := virtualwebauthn.CreateAttestationResponse(
		testcommon.NewWebAuthnRelyingParty(testcommon.DefaultEnv()),
		vAuthenticator,
		credential,
		virtualwebauthn.AttestationOptions{
			Challenge: options.Challenge,
		},
	)
	parsedCredential, stdErr := protocol.ParseCredentialCreationResponseBytes([]byte(credentialJSON))
	require.NoError(t, stdErr)

	return sessionData, parsedCredential, vAuthenticator, credential
}

// Registers a passkey using a complete WebAuthn ceremony.
// Used for integration-ish tests
func registerPasskey(
	t *testing.T,
	webAuthnApp *webauthn.WebAuthn,
	userOb *ent.User,
	name string,
	allowSudo bool,
	isSecondGroup bool,
	db *testcommon.TestDatabase,
) (*ent.Passkey, virtualwebauthn.Credential, virtualwebauthn.Authenticator) {
	t.Helper()

	sessionData, parsedCredential, vAuthenticator, credential := startRegistrationCeremony(
		t, webAuthnApp, userOb,
	)

	passkeyOb, stdErr := dbcommon.WithReadWriteTx(
		t.Context(), db,
		func(tx *ent.Tx, ctx context.Context) (*ent.Passkey, error) {
			return auth.FinishRegisterPasskey(
				name,
				allowSudo,
				isSecondGroup,
				userOb.Username,
				sessionData,
				parsedCredential,
				webAuthnApp,
				tx,
				ctx,
				func(userID uuid.UUID, tx *ent.Tx) (*ent.User, error) {
					return userOb, nil
				},
			)
		},
	)
	require.NoError(t, stdErr)

	return passkeyOb, credential, vAuthenticator
}
