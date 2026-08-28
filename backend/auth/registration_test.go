package auth_test

import (
	"errors"
	"testing"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/auth"
	"github.com/NicoClack/cryptic-stash/backend/common/testcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// Note: the happy paths are part of the endpoint tests instead

// This test is almost identical to TestRegisterStart, not sure if it's worth keeping
func TestStartRegisterPasskey_ExcludesUsersExistingPasskeys(t *testing.T) {
	t.Parallel()

	db := testcommon.CreateDB(t)
	dbClient := db.Client()
	userOb := createUser(t, dbClient)
	firstPasskey := createDummyPasskey(t, dbClient, userOb.ID, "first-passkey", true, false)
	secondPasskey := createDummyPasskey(t, dbClient, userOb.ID, "second-passkey", true, false)

	loadedUser := loadUserWithPasskeys(t, dbClient, userOb.ID)
	webAuthnApp := auth.NewWebAuthnApp(testcommon.DefaultEnv())

	options, _, wrappedErr := auth.StartRegisterPasskey(
		&auth.RealWebAuthnUser{User: loadedUser},
		webAuthnApp,
	)
	require.NoError(t, wrappedErr)
	require.Len(t, options.Challenge, 32)
	require.Equal(
		t,
		[]protocol.CredentialDescriptor{
			firstPasskey.Credential.Descriptor(),
			secondPasskey.Credential.Descriptor(),
			// ^ Shouldn't be able to reregister the existing passkeys
		},
		options.CredentialExcludeList,
	)
}

func TestFinishRegisterPasskey_GivenExpiredWebAuthnSession_ReturnsExpiredError(t *testing.T) {
	t.Parallel()

	db := testcommon.CreateDB(t)
	userOb := createUser(t, db.Client())
	webAuthnApp := auth.NewWebAuthnApp(testcommon.DefaultEnv())

	_, sessionData, wrappedErr := auth.StartRegisterPasskey(
		&auth.RealWebAuthnUser{User: userOb},
		webAuthnApp,
	)
	require.NoError(t, wrappedErr)
	sessionData.Expires = time.Now().Add(-time.Minute)

	tx := testcommon.StartWriteTx(t, db)
	_, wrappedErr = auth.FinishRegisterPasskey(
		"expired",
		false,
		false,
		userOb.Username,
		sessionData,
		nil, // Not needed because the session should be rejected before this is accessed
		webAuthnApp,
		tx,
		t.Context(),
		func(userID uuid.UUID, tx *ent.Tx) (*ent.User, error) {
			return nil, errors.New("FinishRegisterPasskey should fail before this is called")
		},
	)
	require.ErrorIs(t, wrappedErr, auth.ErrWebAuthnSessionExpired)
	require.NoError(t, tx.Rollback())
}

func TestFinishRegisterPasskey_GetUserCallbackError_ReturnsWrappedError(t *testing.T) {
	t.Parallel()

	db := testcommon.CreateDB(t)
	dbClient := db.Client()
	userOb := createUser(t, dbClient)
	webAuthnApp := auth.NewWebAuthnApp(testcommon.DefaultEnv())

	sessionData, parsedCredential, _, _ := startRegistrationCeremony(t, webAuthnApp, userOb)

	tx := testcommon.StartWriteTx(t, db)
	baseError := errors.New("get user callback failed")
	_, wrappedErr := auth.FinishRegisterPasskey(
		"callback-error",
		false,
		false,
		userOb.Username,
		sessionData,
		parsedCredential,
		webAuthnApp,
		tx,
		t.Context(),
		func(userID uuid.UUID, tx *ent.Tx) (*ent.User, error) {
			return nil, baseError
		},
	)
	require.ErrorIs(t, wrappedErr, baseError)
	require.True(t, auth.ErrWrapperFinishRegisterPasskey.HasWrapped(wrappedErr))
	require.True(t, auth.ErrWrapperGetUserCallback.HasWrapped(wrappedErr))
	require.NoError(t, tx.Rollback())
}

func TestFinishRegisterPasskey_RejectsInvalidSessionUserID(t *testing.T) {
	t.Parallel()

	db := testcommon.CreateDB(t)
	dbClient := db.Client()
	userOb := createUser(t, dbClient)
	webAuthnApp := auth.NewWebAuthnApp(testcommon.DefaultEnv())

	sessionData, parsedCredential, _, _ := startRegistrationCeremony(t, webAuthnApp, userOb)
	sessionData.UserID = []byte("invalid")

	tx := testcommon.StartWriteTx(t, db)
	_, wrappedErr := auth.FinishRegisterPasskey(
		"bad-user",
		false,
		false,
		userOb.Username,
		sessionData,
		parsedCredential,
		webAuthnApp,
		tx,
		t.Context(),
		func(userID uuid.UUID, tx *ent.Tx) (*ent.User, error) {
			return nil, errors.New("FinishRegisterPasskey should fail before this is called")
		},
	)
	uuidErr := errors.Unwrap(wrappedErr)
	// The UUID package doesn't use sentinel errors, but in the future,
	// FinishRegisterPasskey might remap this to its own sentinel
	require.Contains(t, uuidErr.Error(), "invalid UUID")
	require.NoError(t, tx.Rollback())
}
