package users_test

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common/dbcommon"
	"github.com/NicoClack/cryptic-stash/backend/common/testcommon"
	"github.com/NicoClack/cryptic-stash/backend/core"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/server/endpoints/testhelpers"
	"github.com/NicoClack/cryptic-stash/backend/server/endpoints/v1/users"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
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
	mimeType := "application/zip"

	keySalt := app.Core.GenerateSalt()
	encryptionKey := app.Core.HashPassword(password, keySalt, app.Env.PASSWORD_HASH_SETTINGS)

	encrypted, nonce, wrappedErr := app.Core.Encrypt(fileContent, encryptionKey)
	require.NoError(t, wrappedErr)

	sessionOb, stdErr := dbcommon.WithReadWriteTx(
		t.Context(), app.Database,
		func(tx *ent.Tx, ctx context.Context) (*ent.Session, error) {
			now := clock.Now()

			userOb, stdErr := tx.User.Create().
				SetUsername(username).
				SetSessionsValidFrom(now).
				SetLockedUntil(now). // Has just expired
				Save(ctx)
			if stdErr != nil {
				return nil, stdErr
			}
			stdErr = tx.UserMessenger.
				Create().
				SetType(app.MockMessenger.Name).
				SetVersion(1).
				SetUserID(userOb.ID).
				SetOptions(nil).
				SetEnabled(true).
				Exec(ctx)
			if stdErr != nil {
				return nil, stdErr
			}
			stdErr = tx.Stash.Create().
				SetContent(encrypted).
				SetFileName(filename).
				SetMime(mimeType).
				SetNonce(nonce).
				SetKeySalt(keySalt).
				SetHashTime(app.Env.PASSWORD_HASH_SETTINGS.Time).
				SetHashMemory(app.Env.PASSWORD_HASH_SETTINGS.Memory).
				SetHashThreads(app.Env.PASSWORD_HASH_SETTINGS.Threads).
				SetUser(userOb).
				Exec(ctx)
			if stdErr != nil {
				return nil, stdErr
			}

			authCode := app.Core.RandomAuthCode()
			validUntil := now.Add(24 * time.Hour)

			sessionOb, stdErr := tx.Session.Create().
				SetUser(userOb).
				SetCreatedAt(now).
				SetCode(authCode).
				SetValidFrom(now).
				SetValidUntil(validUntil).
				SetUserAgent("test-agent").
				SetIP("127.0.0.1").
				Save(ctx)
			if stdErr != nil {
				return sessionOb, stdErr
			}

			_, stdErr = tx.LoginAlert.Create().
				SetSession(sessionOb).
				SetSentAt(now).
				SetVersionedMessengerType(app.MockMessenger.VersionedName()).
				SetConfirmed(true).
				Save(ctx)
			return sessionOb, stdErr
		},
	)
	require.NoError(t, stdErr)

	respRecorder := testcommon.Post(
		t, app.Server,
		"/api/v1/users/download/",
		users.DownloadPayload{
			Username:          username,
			Password:          password,
			AuthorizationCode: base64.StdEncoding.EncodeToString(sessionOb.Code),
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
			Mime:                        mimeType,
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
	mimeType := "application/zip"

	keySalt := core.GenerateSalt()
	encryptionKey := core.HashPassword(password, keySalt, app.Env.PASSWORD_HASH_SETTINGS)

	encrypted, nonce, wrappedErr := core.Encrypt(fileContent, encryptionKey)
	require.NoError(t, wrappedErr)

	sessionOb, stdErr := dbcommon.WithReadWriteTx(
		t.Context(), app.Database,
		func(tx *ent.Tx, ctx context.Context) (*ent.Session, error) {
			now := clock.Now()
			// Set SessionsValidFrom to be in the future
			sessionsValidFrom := now.Add(1 * time.Hour)

			userOb, stdErr := tx.User.Create().
				SetUsername(username).
				SetSessionsValidFrom(sessionsValidFrom).
				Save(ctx)
			if stdErr != nil {
				return nil, stdErr
			}
			stdErr = tx.UserMessenger.
				Create().
				SetType(app.MockMessenger.Name).
				SetVersion(1).
				SetUserID(userOb.ID).
				SetOptions(nil).
				SetEnabled(true).
				Exec(ctx)
			if stdErr != nil {
				return nil, stdErr
			}
			stdErr = tx.Stash.Create().
				SetContent(encrypted).
				SetFileName(filename).
				SetMime(mimeType).
				SetNonce(nonce).
				SetKeySalt(keySalt).
				SetHashTime(app.Env.PASSWORD_HASH_SETTINGS.Time).
				SetHashMemory(app.Env.PASSWORD_HASH_SETTINGS.Memory).
				SetHashThreads(app.Env.PASSWORD_HASH_SETTINGS.Threads).
				SetUser(userOb).
				Exec(ctx)
			if stdErr != nil {
				return nil, stdErr
			}

			authCode := core.RandomAuthCode()
			validUntil := now.Add(24 * time.Hour)

			sessionOb, stdErr := tx.Session.Create().
				SetUser(userOb).
				SetCreatedAt(now).
				SetCode(authCode).
				SetValidFrom(now).
				SetValidUntil(validUntil).
				SetUserAgent("test-agent").
				SetIP("127.0.0.1").
				Save(ctx)
			if stdErr != nil {
				return sessionOb, stdErr
			}

			_, stdErr = tx.LoginAlert.Create().
				SetSession(sessionOb).
				SetSentAt(now).
				SetVersionedMessengerType(app.MockMessenger.VersionedName()).
				SetConfirmed(true).
				Save(ctx)
			return sessionOb, stdErr
		},
	)
	require.NoError(t, stdErr)

	respRecorder := testcommon.Post(
		t, app.Server,
		"/api/v1/users/download/",
		users.DownloadPayload{
			Username:          username,
			Password:          password,
			AuthorizationCode: base64.StdEncoding.EncodeToString(sessionOb.Code),
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
	mimeType := "application/zip"

	keySalt := app.Core.GenerateSalt()
	encryptionKey := app.Core.HashPassword(password, keySalt, app.Env.PASSWORD_HASH_SETTINGS)

	encrypted, nonce, wrappedErr := app.Core.Encrypt(fileContent, encryptionKey)
	require.NoError(t, wrappedErr)

	sessionOb, stdErr := dbcommon.WithReadWriteTx(
		t.Context(), app.Database,
		func(tx *ent.Tx, ctx context.Context) (*ent.Session, error) {
			now := clock.Now()

			userOb, stdErr := tx.User.Create().
				SetUsername(username).
				SetSessionsValidFrom(now).
				SetLockedUntil(now.Add((24 * time.Hour) + time.Nanosecond)).
				Save(ctx)
			if stdErr != nil {
				return nil, stdErr
			}
			stdErr = tx.UserMessenger.
				Create().
				SetType(app.MockMessenger.Name).
				SetVersion(1).
				SetUserID(userOb.ID).
				SetOptions(nil).
				SetEnabled(true).
				Exec(ctx)
			if stdErr != nil {
				return nil, stdErr
			}
			stdErr = tx.Stash.Create().
				SetContent(encrypted).
				SetFileName(filename).
				SetMime(mimeType).
				SetNonce(nonce).
				SetKeySalt(keySalt).
				SetHashTime(app.Env.PASSWORD_HASH_SETTINGS.Time).
				SetHashMemory(app.Env.PASSWORD_HASH_SETTINGS.Memory).
				SetHashThreads(app.Env.PASSWORD_HASH_SETTINGS.Threads).
				SetUser(userOb).
				Exec(ctx)
			if stdErr != nil {
				return nil, stdErr
			}

			authCode := app.Core.RandomAuthCode()
			validUntil := now.Add(2 * 24 * time.Hour) // Lasts until after the user is unlocked

			// This session shouldn't exist, but let's say an attacker managed to somehow create it
			// at the exact time the user was locked
			// Even though both things should happen in the same transaction
			sessionOb, stdErr := tx.Session.Create().
				SetUser(userOb).
				SetCreatedAt(now).
				SetCode(authCode).
				SetValidFrom(now).
				SetValidUntil(validUntil).
				SetUserAgent("test-agent").
				SetIP("127.0.0.1").
				Save(ctx)
			if stdErr != nil {
				return sessionOb, stdErr
			}

			_, stdErr = tx.LoginAlert.Create().
				SetSession(sessionOb).
				SetSentAt(now).
				SetVersionedMessengerType(app.MockMessenger.VersionedName()).
				SetConfirmed(true).
				Save(ctx)
			if stdErr != nil {
				return sessionOb, stdErr
			}

			// Slightly unrealistic but it's easiest to create both alerts here
			// This is needed otherwise core.IsUserSufficientlyNotified thinks the jobs are failing and prevents the login
			_, stdErr = tx.LoginAlert.Create().
				SetSession(sessionOb).
				SetSentAt(now.Add(24 * time.Hour)).
				SetVersionedMessengerType(app.MockMessenger.VersionedName()).
				SetConfirmed(true).
				Save(ctx)
			if stdErr != nil {
				return sessionOb, stdErr
			}

			return sessionOb, nil
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
				AuthorizationCode: base64.StdEncoding.EncodeToString(sessionOb.Code),
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
			Mime:                        mimeType,
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
	mimeType := "application/zip"

	keySalt := app.Core.GenerateSalt()
	encryptionKey := app.Core.HashPassword(password, keySalt, app.Env.PASSWORD_HASH_SETTINGS)

	encrypted, nonce, wrappedErr := app.Core.Encrypt(fileContent, encryptionKey)
	require.NoError(t, wrappedErr)

	sessionOb, stdErr := dbcommon.WithReadWriteTx(
		t.Context(), app.Database,
		func(tx *ent.Tx, ctx context.Context) (*ent.Session, error) {
			now := clock.Now()

			userOb, stdErr := tx.User.Create().
				SetUsername(username).
				SetSessionsValidFrom(now).
				SetLockedUntil(now.Add(-time.Hour)). // Expired a little while ago
				SetLocked(true).                     // But this takes priority
				Save(ctx)
			if stdErr != nil {
				return nil, stdErr
			}
			stdErr = tx.UserMessenger.
				Create().
				SetType(app.MockMessenger.Name).
				SetVersion(1).
				SetUserID(userOb.ID).
				SetOptions(nil).
				SetEnabled(true).
				Exec(ctx)
			if stdErr != nil {
				return nil, stdErr
			}
			stdErr = tx.Stash.Create().
				SetContent(encrypted).
				SetFileName(filename).
				SetMime(mimeType).
				SetNonce(nonce).
				SetKeySalt(keySalt).
				SetHashTime(app.Env.PASSWORD_HASH_SETTINGS.Time).
				SetHashMemory(app.Env.PASSWORD_HASH_SETTINGS.Memory).
				SetHashThreads(app.Env.PASSWORD_HASH_SETTINGS.Threads).
				SetUser(userOb).
				Exec(ctx)
			if stdErr != nil {
				return nil, stdErr
			}

			authCode := app.Core.RandomAuthCode()
			validUntil := now.Add(24 * time.Hour)

			// This session shouldn't exist, but let's say an attacker managed to somehow create it
			// at the exact time the user was locked
			// Even though both things should happen in the same transaction
			sessionOb, stdErr := tx.Session.Create().
				SetUser(userOb).
				SetCreatedAt(now).
				SetCode(authCode).
				SetValidFrom(now).
				SetValidUntil(validUntil).
				SetUserAgent("test-agent").
				SetIP("127.0.0.1").
				Save(ctx)
			if stdErr != nil {
				return sessionOb, stdErr
			}

			_, stdErr = tx.LoginAlert.Create().
				SetSession(sessionOb).
				SetSentAt(now).
				SetVersionedMessengerType(app.MockMessenger.VersionedName()).
				SetConfirmed(true).
				Save(ctx)
			return sessionOb, stdErr
		},
	)
	require.NoError(t, stdErr)

	respRecorder := testcommon.Post(
		t, app.Server,
		"/api/v1/users/download/",
		users.DownloadPayload{
			Username:          username,
			Password:          password,
			AuthorizationCode: base64.StdEncoding.EncodeToString(sessionOb.Code),
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
