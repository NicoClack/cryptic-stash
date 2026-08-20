package auth_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/auth"
	"github.com/NicoClack/cryptic-stash/backend/auth/testdata"
	"github.com/NicoClack/cryptic-stash/backend/common/testcommon"
	"github.com/descope/virtualwebauthn"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func createAssertion(
	t *testing.T,
	options protocol.PublicKeyCredentialRequestOptions,
	vAuthenticator virtualwebauthn.Authenticator,
	credential virtualwebauthn.Credential,
) *protocol.ParsedCredentialAssertionData {
	t.Helper()

	assertionResponse := virtualwebauthn.CreateAssertionResponse(
		testcommon.NewWebAuthnRelyingParty(testcommon.DefaultEnv()),
		vAuthenticator,
		credential,
		virtualwebauthn.AssertionOptions{
			Challenge: options.Challenge,
		},
	)

	var parsedAssertion protocol.CredentialAssertionResponse
	stdErr := json.Unmarshal([]byte(assertionResponse), &parsedAssertion)
	require.NoError(t, stdErr)
	parsedResponse, stdErr := parsedAssertion.Parse()
	require.NoError(t, stdErr)
	return parsedResponse
}

func TestStartLogin_StoresUniqueWebAuthnSessions(t *testing.T) {
	t.Parallel()

	webAuthnApp := auth.NewWebAuthnApp(testcommon.DefaultEnv())
	tempKV := newMinimalTempKeyValueService(t)

	sessionID1, options1, wrappedErr := auth.StartLogin(webAuthnApp, tempKV)
	require.NoError(t, wrappedErr)
	sessionID2, options2, wrappedErr := auth.StartLogin(webAuthnApp, tempKV)
	require.NoError(t, wrappedErr)

	require.NotEqual(t, sessionID1, sessionID2)
	require.NotEmpty(t, options1.Challenge)
	require.NotEmpty(t, options2.Challenge)

	var sessionData *webauthn.SessionData
	require.True(t, tempKV.Get(auth.WebAuthnSessionStoreName, sessionID1.String(), &sessionData))
	require.True(t, tempKV.Get(auth.WebAuthnSessionStoreName, sessionID2.String(), &sessionData))
}

func TestValidateLogin_RejectsUnknownWebAuthnSessionID(t *testing.T) {
	t.Parallel()

	db := testcommon.CreateDB(t)
	tx := testcommon.StartWriteTx(t, db)
	_, _, _, wrappedErr := auth.ValidateLogin(
		uuid.New(),
		nil,
		t.Context(),
		auth.NewWebAuthnApp(testcommon.DefaultEnv()),
		tx,
		newMinimalTempKeyValueService(t),
		testcommon.NewTestLogger(t),
	)
	require.ErrorIs(t, wrappedErr, auth.ErrInvalidWebAuthnSessionID)
	require.NoError(t, tx.Rollback())
}

func TestValidateLogin_UnknownUser(t *testing.T) {
	t.Parallel()

	db := testcommon.CreateDB(t)
	webAuthnApp := auth.NewWebAuthnApp(testcommon.DefaultEnv())
	tempKV := newMinimalTempKeyValueService(t)

	vAuthenticator, credential := newVirtualAuthenticator(
		uuid.New(), // Unknown user
	)

	sessionID, options, wrappedErr := auth.StartLogin(webAuthnApp, tempKV)
	require.NoError(t, wrappedErr)
	parsedAssertionResp := createAssertion(t, options, vAuthenticator, credential)

	tx := testcommon.StartWriteTx(t, db)
	_, _, _, wrappedErr = auth.ValidateLogin(
		sessionID,
		parsedAssertionResp,
		t.Context(),
		webAuthnApp,
		tx,
		tempKV,
		testcommon.NewTestLogger(t),
	)
	require.ErrorIs(t, wrappedErr, auth.ErrWebAuthnUserNotFound)
	require.NoError(t, tx.Rollback())
}

// Loads a pregenerated passkey to test that the format hasn't changed and so previously created passkeys still work
func TestValidateLogin_ExistingPasskey(t *testing.T) {
	t.Parallel()

	db := testcommon.CreateDB(t)
	dbClient := db.Client()
	webAuthnApp := auth.NewWebAuthnApp(testcommon.DefaultEnv())
	tempKV := newMinimalTempKeyValueService(t)

	var clientCredential virtualwebauthn.Credential
	require.NoError(t, json.Unmarshal([]byte(testdata.ExistingClientCredentialJSON), &clientCredential))

	var serverCredential webauthn.Credential
	require.NoError(t, json.Unmarshal([]byte(testdata.ExistingServerCredentialJSON), &serverCredential))

	userOb := createUser(t, dbClient)
	passkeyOb := dbClient.Passkey.Create().
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		SetName("existing-passkey").
		SetAllowSudo(true).
		SetCredentialID(serverCredential.ID).
		SetCredential(serverCredential).
		SetIsSecondGroup(false).
		SetUserID(userOb.ID).
		SaveX(t.Context())

	vAuthenticator := virtualwebauthn.NewAuthenticator()
	vAuthenticator.Options.UserHandle = userOb.ID[:]
	vAuthenticator.AddCredential(clientCredential)

	sessionID, options, wrappedErr := auth.StartLogin(webAuthnApp, tempKV)
	require.NoError(t, wrappedErr)
	parsedAssertionResp := createAssertion(t, options, vAuthenticator, clientCredential)

	tx := testcommon.StartWriteTx(t, db)
	returnedUser, returnedPasskey, _, wrappedErr := auth.ValidateLogin(
		sessionID,
		parsedAssertionResp,
		t.Context(),
		webAuthnApp,
		tx,
		tempKV,
		testcommon.NewTestLogger(t),
	)
	require.NoError(t, wrappedErr)
	require.Equal(t, userOb.ID, returnedUser.ID)
	require.Equal(t, passkeyOb.ID, returnedPasskey.ID)
	require.NoError(t, tx.Commit())
}

// This integration test can't go at the HTTP layer
// because we need to assert exactly when the WebAuthn session is deleted
func TestValidateLogin_UpdatesCredentialSignCount(t *testing.T) {
	t.Parallel()

	db := testcommon.CreateDB(t)
	dbClient := db.Client()
	webAuthnApp := auth.NewWebAuthnApp(testcommon.DefaultEnv())
	tempKV := newMinimalTempKeyValueService(t)

	userOb := createUser(t, dbClient)
	passkeyOb, credential, vAuthenticator := registerPasskey(t, webAuthnApp, userOb, "login-passkey", false, false, db)
	signCountBefore := passkeyOb.Credential.Authenticator.SignCount

	sessionID, options, wrappedErr := auth.StartLogin(webAuthnApp, tempKV)
	require.NoError(t, wrappedErr)
	// virtualwebauthn doesn't increment the counter itself, so set it explicitly to
	// verify that ValidateLogin persists the authenticator's new counter.
	credential.Counter = 42
	parsedAssertionResp := createAssertion(t, options, vAuthenticator, credential)

	tx := testcommon.StartWriteTx(t, db)
	returnedUser, returnedPasskey, _, wrappedErr := auth.ValidateLogin(
		sessionID,
		parsedAssertionResp,
		t.Context(),
		webAuthnApp,
		tx,
		tempKV,
		testcommon.NewTestLogger(t),
	)
	require.NoError(t, wrappedErr)
	require.Equal(t, userOb.ID, returnedUser.ID)
	require.Equal(t, passkeyOb.ID, returnedPasskey.ID)

	// The WebAuthn ceremony should only deleted once the transaction commits
	var sessionData *webauthn.SessionData
	require.True(t, tempKV.Get(auth.WebAuthnSessionStoreName, sessionID.String(), &sessionData))

	require.NoError(t, tx.Commit())
	require.False(t, tempKV.Get(auth.WebAuthnSessionStoreName, sessionID.String(), &sessionData))

	signCountAfter := dbClient.Passkey.GetX(t.Context(), passkeyOb.ID).Credential.Authenticator.SignCount
	require.Greater(t, signCountAfter, signCountBefore)
	require.Equal(t, uint32(42), signCountAfter)
}
