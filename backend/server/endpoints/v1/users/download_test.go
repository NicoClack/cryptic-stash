package users_test

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common/dbcommon"
	"github.com/NicoClack/cryptic-stash/backend/common/testcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/server/endpoints/v1/users"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/NicoClack/cryptic-stash/backend/testhelpers"
	"github.com/gin-gonic/gin"
	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/require"
)

func TestDownload_SufficientlyNotifiedUser_AllowsDownload(t *testing.T) {
	t.Parallel()
	// TODO: assert messenger sent message, maybe improve the setup

	clock := clockwork.NewFakeClock()
	app := testhelpers.NewApp(t, &testhelpers.AppOptions{
		Clock: clock,
	})

	username := "alice"
	password := "password123456"
	fileContent := []byte("file content here")
	filename := "data.zip"

	passwordSalt := app.Core.GenerateSalt()
	stashKek := app.Core.HashPassword(password, passwordSalt, app.Env.PASSWORD_HASH_SETTINGS)
	stashDataKey := app.Core.GenerateEncryptionKey()

	encryptedContent, wrappedErr := app.Core.Encrypt(fileContent, stashDataKey)
	require.NoError(t, wrappedErr)
	encryptedFileName, wrappedErr := app.Core.Encrypt([]byte(filename), stashDataKey)
	require.NoError(t, wrappedErr)

	encryptedDataKey, wrappedErr := app.Core.Encrypt(stashDataKey, stashKek)
	require.NoError(t, wrappedErr)

	authCode := app.Core.RandomAuthCode()

	stdErr := dbcommon.WithWriteTx(
		t.Context(), app.Database,
		func(tx *ent.Tx, ctx context.Context) error {
			now := clock.Now()

			userOb, stdErr := tx.User.Create().
				SetUsername(username).
				SetCreatedAt(now).
				SetUpdatedAt(now).
				SetDownloadSessionsValidFrom(now).
				SetLockedUntil(now). // Has just expired
				Save(ctx)
			if stdErr != nil {
				return stdErr
			}

			userMessengerOb, stdErr := tx.UserMessenger.
				Create().
				SetCreatedAt(now).
				SetUpdatedAt(now).
				SetType(app.MockMessenger.Name).
				SetVersion(1).
				SetUserID(userOb.ID).
				SetOptions(nil).
				SetEnabled(true).
				Save(ctx)
			if stdErr != nil {
				return stdErr
			}
			stdErr = tx.Stash.Create().
				SetCreatedAt(now).
				SetUpdatedAt(now).
				SetContent(encryptedContent).
				SetFileName(encryptedFileName).
				SetEncryptionDataKey(encryptedDataKey).
				SetPasswordSalt(passwordSalt).
				SetHashTime(app.Env.PASSWORD_HASH_SETTINGS.Time).
				SetHashMemory(app.Env.PASSWORD_HASH_SETTINGS.Memory).
				SetHashThreads(app.Env.PASSWORD_HASH_SETTINGS.Threads).
				SetUser(userOb).
				Exec(ctx)
			if stdErr != nil {
				return stdErr
			}

			hashedAuthCode := sha256.Sum256(authCode)
			validUntil := now.Add(24 * time.Hour)
			downloadSessionOb, stdErr := tx.DownloadSession.Create().
				SetCreatedAt(now).
				SetUpdatedAt(now).
				SetUser(userOb).
				SetHashedAuthCode(hashedAuthCode[:]).
				SetValidFrom(now).
				SetValidUntil(validUntil).
				SetUserAgent("test-agent").
				SetIP("127.0.0.1").
				Save(ctx)
			if stdErr != nil {
				return stdErr
			}

			stdErr = tx.LoginAlert.Create().
				SetCreatedAt(now).
				SetUpdatedAt(now).
				SetDownloadSession(downloadSessionOb).
				SetSentAt(now).
				SetUserMessenger(userMessengerOb).
				SetConfirmed(true).
				Exec(ctx)
			return stdErr
		},
	)
	require.NoError(t, stdErr)

	respRecorder := testcommon.Post(
		t, app.Server,
		"/api/v1/users/download/",
		users.DownloadPayload{
			Username:          username,
			Password:          password,
			AuthorizationCode: base64.StdEncoding.EncodeToString(authCode),
		},
	)
	testcommon.AssertJSONResponse(
		t, respRecorder,
		http.StatusOK,
		&users.DownloadResponse{
			Errors:                      []servercommon.ErrorDetail{},
			AuthorizationCodeValidFrom:  nil,
			AuthorizationCodeValidUntil: nil,
			Content:                     fileContent,
			Filename:                    filename,
		},
	)
}

func TestDownload_UndeletedInvalidSession_ReturnsUnauthorizedError(t *testing.T) {
	t.Parallel()

	clock := clockwork.NewFakeClock()
	app := testhelpers.NewApp(t, &testhelpers.AppOptions{
		Clock: clock,
	})

	username := "bob"
	password := "password123456"
	fileContent := []byte("file content here")
	filename := "data.zip"

	passwordSalt := app.Core.GenerateSalt()
	stashKek := app.Core.HashPassword(password, passwordSalt, app.Env.PASSWORD_HASH_SETTINGS)
	stashDataKey := app.Core.GenerateEncryptionKey()

	encryptedContent, wrappedErr := app.Core.Encrypt(fileContent, stashDataKey)
	require.NoError(t, wrappedErr)
	encryptedFileName, wrappedErr := app.Core.Encrypt([]byte(filename), stashDataKey)
	require.NoError(t, wrappedErr)

	encryptedDataKey, wrappedErr := app.Core.Encrypt(stashDataKey, stashKek)
	require.NoError(t, wrappedErr)

	authCode := app.Core.RandomAuthCode()
	stdErr := dbcommon.WithWriteTx(
		t.Context(), app.Database,
		func(tx *ent.Tx, ctx context.Context) error {
			now := clock.Now()
			// Set SessionsValidFrom to be in the future
			sessionsValidFrom := now.Add(1 * time.Hour)

			userOb, stdErr := tx.User.Create().
				SetUsername(username).
				SetCreatedAt(now).
				SetUpdatedAt(now).
				SetDownloadSessionsValidFrom(sessionsValidFrom).
				Save(ctx)
			if stdErr != nil {
				return stdErr
			}
			userMessengerOb, stdErr := tx.UserMessenger.
				Create().
				SetCreatedAt(now).
				SetUpdatedAt(now).
				SetType(app.MockMessenger.Name).
				SetVersion(1).
				SetUserID(userOb.ID).
				SetOptions(nil).
				SetEnabled(true).
				Save(ctx)
			if stdErr != nil {
				return stdErr
			}
			stdErr = tx.Stash.Create().
				SetCreatedAt(now).
				SetUpdatedAt(now).
				SetContent(encryptedContent).
				SetFileName(encryptedFileName).
				SetEncryptionDataKey(encryptedDataKey).
				SetPasswordSalt(passwordSalt).
				SetHashTime(app.Env.PASSWORD_HASH_SETTINGS.Time).
				SetHashMemory(app.Env.PASSWORD_HASH_SETTINGS.Memory).
				SetHashThreads(app.Env.PASSWORD_HASH_SETTINGS.Threads).
				SetUser(userOb).
				Exec(ctx)
			if stdErr != nil {
				return stdErr
			}

			hashedAuthCode := sha256.Sum256(authCode)
			validUntil := now.Add(24 * time.Hour)

			downloadSessionOb, stdErr := tx.DownloadSession.Create().
				SetCreatedAt(now).
				SetUpdatedAt(now).
				SetUser(userOb).
				SetHashedAuthCode(hashedAuthCode[:]).
				SetValidFrom(now).
				SetValidUntil(validUntil).
				SetUserAgent("test-agent").
				SetIP("127.0.0.1").
				Save(ctx)
			if stdErr != nil {
				return stdErr
			}

			return tx.LoginAlert.Create().
				SetCreatedAt(now).
				SetUpdatedAt(now).
				SetDownloadSession(downloadSessionOb).
				SetSentAt(now).
				SetUserMessenger(userMessengerOb).
				SetConfirmed(true).
				Exec(ctx)
		},
	)
	require.NoError(t, stdErr)

	respRecorder := testcommon.Post(
		t, app.Server,
		"/api/v1/users/download/",
		users.DownloadPayload{
			Username:          username,
			Password:          password,
			AuthorizationCode: base64.StdEncoding.EncodeToString(authCode),
		},
	)
	testcommon.AssertJSONResponse(
		t, respRecorder,
		http.StatusUnauthorized,
		&gin.H{
			"errors": []servercommon.ErrorDetail{},
		},
	)
}

func TestDownload_TemporarilyLockedUser_ReturnsUnauthorizedError(t *testing.T) {
	t.Parallel()
	// TODO: assert messenger sent message, maybe improve the setup

	clock := clockwork.NewFakeClock()
	app := testhelpers.NewApp(t, &testhelpers.AppOptions{
		Clock: clock,
	})

	username := "alice"
	password := "password123456"
	fileContent := []byte("file content here")
	filename := "data.zip"

	passwordSalt := app.Core.GenerateSalt()
	stashKek := app.Core.HashPassword(password, passwordSalt, app.Env.PASSWORD_HASH_SETTINGS)
	stashDataKey := app.Core.GenerateEncryptionKey()

	encryptedContent, wrappedErr := app.Core.Encrypt(fileContent, stashDataKey)
	require.NoError(t, wrappedErr)
	encryptedFileName, wrappedErr := app.Core.Encrypt([]byte(filename), stashDataKey)
	require.NoError(t, wrappedErr)

	encryptedDataKey, wrappedErr := app.Core.Encrypt(stashDataKey, stashKek)
	require.NoError(t, wrappedErr)

	authCode := app.Core.RandomAuthCode()
	stdErr := dbcommon.WithWriteTx(
		t.Context(), app.Database,
		func(tx *ent.Tx, ctx context.Context) error {
			now := clock.Now()

			userOb, stdErr := tx.User.Create().
				SetUsername(username).
				SetCreatedAt(now).
				SetUpdatedAt(now).
				SetDownloadSessionsValidFrom(now).
				SetLockedUntil(now.Add((24 * time.Hour) + time.Nanosecond)).
				Save(ctx)
			if stdErr != nil {
				return stdErr
			}
			userMessengerOb, stdErr := tx.UserMessenger.
				Create().
				SetCreatedAt(now).
				SetUpdatedAt(now).
				SetType(app.MockMessenger.Name).
				SetVersion(1).
				SetUserID(userOb.ID).
				SetOptions(nil).
				SetEnabled(true).
				Save(ctx)
			if stdErr != nil {
				return stdErr
			}
			stdErr = tx.Stash.Create().
				SetCreatedAt(now).
				SetUpdatedAt(now).
				SetContent(encryptedContent).
				SetFileName(encryptedFileName).
				SetEncryptionDataKey(encryptedDataKey).
				SetPasswordSalt(passwordSalt).
				SetHashTime(app.Env.PASSWORD_HASH_SETTINGS.Time).
				SetHashMemory(app.Env.PASSWORD_HASH_SETTINGS.Memory).
				SetHashThreads(app.Env.PASSWORD_HASH_SETTINGS.Threads).
				SetUser(userOb).
				Exec(ctx)
			if stdErr != nil {
				return stdErr
			}

			hashedAuthCode := sha256.Sum256(authCode)
			validUntil := now.Add(2 * 24 * time.Hour) // Lasts until after the user is unlocked

			// This download session shouldn't exist, but let's say an attacker managed to somehow create it
			// at the exact time the user was locked
			// Even though both things should happen in the same transaction
			downloadSessionOb, stdErr := tx.DownloadSession.Create().
				SetCreatedAt(now).
				SetUpdatedAt(now).
				SetUser(userOb).
				SetHashedAuthCode(hashedAuthCode[:]).
				SetValidFrom(now).
				SetValidUntil(validUntil).
				SetUserAgent("test-agent").
				SetIP("127.0.0.1").
				Save(ctx)
			if stdErr != nil {
				return stdErr
			}

			stdErr = tx.LoginAlert.Create().
				SetCreatedAt(now).
				SetUpdatedAt(now).
				SetDownloadSession(downloadSessionOb).
				SetSentAt(now).
				SetUserMessenger(userMessengerOb).
				SetConfirmed(true).
				Exec(ctx)
			if stdErr != nil {
				return stdErr
			}

			// Slightly unrealistic but it's easiest to create both alerts here
			// This is needed otherwise core.IsUserSufficientlyNotified thinks the jobs are failing and prevents the login
			return tx.LoginAlert.Create().
				SetCreatedAt(now).
				SetUpdatedAt(now).
				SetDownloadSession(downloadSessionOb).
				SetSentAt(now.Add(24 * time.Hour)).
				SetUserMessenger(userMessengerOb).
				SetConfirmed(true).
				Exec(ctx)
		},
	)
	require.NoError(t, stdErr)

	makeRequest := func() *httptest.ResponseRecorder {
		return testcommon.Post(
			t, app.Server,
			"/api/v1/users/download/",
			users.DownloadPayload{
				Username:          username,
				Password:          password,
				AuthorizationCode: base64.StdEncoding.EncodeToString(authCode),
			},
		)
	}
	respRecorder := makeRequest()
	testcommon.AssertJSONResponse(
		t, respRecorder,
		http.StatusUnauthorized,
		&gin.H{
			"errors": []servercommon.ErrorDetail{},
		},
	)

	clock.Advance(24 * time.Hour) // 1ns before the user is unlocked
	respRecorder = makeRequest()
	testcommon.AssertJSONResponse(
		t, respRecorder,
		http.StatusUnauthorized,
		&gin.H{
			"errors": []servercommon.ErrorDetail{},
		},
	)

	clock.Advance(time.Nanosecond)
	respRecorder = makeRequest()
	// Unfortunately if this did actually happen,
	// we wouldn't have a way to know to reject this request after the temporary lock expired
	testcommon.AssertJSONResponse(
		t, respRecorder,
		http.StatusOK,
		&users.DownloadResponse{
			Errors:                      []servercommon.ErrorDetail{},
			AuthorizationCodeValidFrom:  nil,
			AuthorizationCodeValidUntil: nil,
			Content:                     fileContent,
			Filename:                    filename,
		},
	)
}

func TestDownload_PermanentlyLockedUser_ReturnsUnauthorizedError(t *testing.T) {
	t.Parallel()
	// TODO: assert messenger sent message, maybe improve the setup

	clock := clockwork.NewFakeClock()
	app := testhelpers.NewApp(t, &testhelpers.AppOptions{
		Clock: clock,
	})

	username := "alice"
	password := "password123456"
	fileContent := []byte("file content here")
	filename := "data.zip"

	passwordSalt := app.Core.GenerateSalt()
	stashKek := app.Core.HashPassword(password, passwordSalt, app.Env.PASSWORD_HASH_SETTINGS)
	stashDataKey := app.Core.GenerateEncryptionKey()

	encryptedContent, wrappedErr := app.Core.Encrypt(fileContent, stashDataKey)
	require.NoError(t, wrappedErr)
	encryptedFileName, wrappedErr := app.Core.Encrypt([]byte(filename), stashDataKey)
	require.NoError(t, wrappedErr)

	encryptedDataKey, wrappedErr := app.Core.Encrypt(stashDataKey, stashKek)
	require.NoError(t, wrappedErr)

	authCode := app.Core.RandomAuthCode()
	stdErr := dbcommon.WithWriteTx(
		t.Context(), app.Database,
		func(tx *ent.Tx, ctx context.Context) error {
			now := clock.Now()

			userOb, stdErr := tx.User.Create().
				SetCreatedAt(now).
				SetUpdatedAt(now).
				SetUsername(username).
				SetDownloadSessionsValidFrom(now).
				SetLockedUntil(now.Add(-time.Hour)). // Expired a little while ago
				SetLocked(true).                     // But this takes priority
				Save(ctx)
			if stdErr != nil {
				return stdErr
			}
			userMessengerOb, stdErr := tx.UserMessenger.Create().
				SetCreatedAt(now).
				SetUpdatedAt(now).
				SetType(app.MockMessenger.Name).
				SetVersion(1).
				SetUserID(userOb.ID).
				SetOptions(nil).
				SetEnabled(true).
				Save(ctx)
			if stdErr != nil {
				return stdErr
			}
			stdErr = tx.Stash.Create().
				SetCreatedAt(now).
				SetUpdatedAt(now).
				SetContent(encryptedContent).
				SetFileName(encryptedFileName).
				SetEncryptionDataKey(encryptedDataKey).
				SetPasswordSalt(passwordSalt).
				SetHashTime(app.Env.PASSWORD_HASH_SETTINGS.Time).
				SetHashMemory(app.Env.PASSWORD_HASH_SETTINGS.Memory).
				SetHashThreads(app.Env.PASSWORD_HASH_SETTINGS.Threads).
				SetUser(userOb).
				Exec(ctx)
			if stdErr != nil {
				return stdErr
			}

			hashedAuthCode := sha256.Sum256(authCode)
			validUntil := now.Add(24 * time.Hour)

			// This download session shouldn't exist, but let's say an attacker managed to somehow create it
			// at the exact time the user was locked
			// Even though both things should happen in the same transaction
			downloadSessionOb, stdErr := tx.DownloadSession.Create().
				SetCreatedAt(now).
				SetUpdatedAt(now).
				SetUser(userOb).
				SetHashedAuthCode(hashedAuthCode[:]).
				SetValidFrom(now).
				SetValidUntil(validUntil).
				SetUserAgent("test-agent").
				SetIP("127.0.0.1").
				Save(ctx)
			if stdErr != nil {
				return stdErr
			}

			return tx.LoginAlert.Create().
				SetCreatedAt(now).
				SetUpdatedAt(now).
				SetDownloadSession(downloadSessionOb).
				SetSentAt(now).
				SetUserMessenger(userMessengerOb).
				SetConfirmed(true).
				Exec(ctx)
		},
	)
	require.NoError(t, stdErr)

	respRecorder := testcommon.Post(
		t, app.Server,
		"/api/v1/users/download/",
		users.DownloadPayload{
			Username:          username,
			Password:          password,
			AuthorizationCode: base64.StdEncoding.EncodeToString(authCode),
		},
	)
	testcommon.AssertJSONResponse(
		t, respRecorder,
		http.StatusUnauthorized,
		&gin.H{
			"errors": []servercommon.ErrorDetail{},
		},
	)
}
