package services_test

import (
	"context"
	"crypto/sha256"
	"testing"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common/dbcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/ent/invite"
	"github.com/NicoClack/cryptic-stash/backend/testhelpers"
	"github.com/google/uuid"
	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/require"
)

func createInvite(
	t *testing.T,
	dbClient *ent.Client,
	email string,
	hashedCode []byte,
	expiresAt time.Time,
	userID *uuid.UUID,
) {
	t.Helper()
	createdAt := expiresAt.Add(-time.Hour)
	_, stdErr := dbClient.Invite.Create().
		SetEmail(email).
		SetHashedCode(hashedCode).
		SetExpiresAt(expiresAt).
		SetNillableUserID(userID).
		SetCreatedAt(createdAt).
		SetUpdatedAt(createdAt).
		Save(t.Context())
	require.NoError(t, stdErr)
}

func TestInvitesService_DeleteExpiredInvites(t *testing.T) {
	t.Parallel()

	clock := clockwork.NewFakeClock()
	app := testhelpers.NewApp(t, &testhelpers.AppOptions{Clock: clock})
	now := clock.Now()
	dbClient := app.Database.Client()

	formatEmail := func(suffix string) string {
		return "test-" + suffix + "@example.com"
	}
	getHash := func(suffix string) []byte {
		h := sha256.Sum256([]byte("code-" + suffix))
		return h[:]
	}

	// Expired invite, should be deleted
	createInvite(t, dbClient, formatEmail("expired"), getHash("expired"), now.Add(-time.Hour), nil)
	// Valid and unused, shouldn't be deleted
	createInvite(t, dbClient, formatEmail("valid"), getHash("valid"), now.Add(time.Hour), nil)

	// Expired but used invite, shouldn't be deleted
	userOb := dbClient.User.Create().
		SetUsername(formatEmail("used")).
		SetCreatedAt(now.Add(-2 * time.Hour)).
		SetUpdatedAt(now.Add(-2 * time.Hour)).
		SaveX(t.Context())
	createInvite(t, dbClient, formatEmail("used"), getHash("used"), now.Add(-time.Hour), new(userOb.ID))

	// Another expired
	createInvite(t, dbClient, formatEmail("expired-2"), getHash("expired-2"), now.Add(-30*time.Minute), nil)

	stdErr := dbcommon.WithWriteTx(
		t.Context(), app.Database,
		func(tx *ent.Tx, ctx context.Context) error {
			return app.Invites.DeleteExpiredInvites(tx, ctx)
		},
	)
	require.NoError(t, stdErr)

	remainingCount := dbClient.Invite.Query().CountX(t.Context())
	require.Equal(t, 2, remainingCount, "only valid and used invites should remain")

	validInvite := dbClient.Invite.Query().
		Where(invite.Email(formatEmail("valid"))).
		OnlyX(t.Context())
	require.NotNil(t, validInvite)

	usedInvite := dbClient.Invite.Query().
		Where(invite.Email(formatEmail("used"))).
		OnlyX(t.Context())
	require.NotNil(t, usedInvite)
	require.NotNil(t, usedInvite.UserID)

	clock.Advance(time.Hour) // The exact time validInvite expires
	stdErr = dbcommon.WithWriteTx(
		t.Context(), app.Database,
		func(tx *ent.Tx, ctx context.Context) error {
			return app.Invites.DeleteExpiredInvites(tx, ctx)
		},
	)
	require.NoError(t, stdErr)

	remainingCount = dbClient.Invite.Query().CountX(t.Context())
	require.Equal(t, 1, remainingCount, "only the used invite should remain after the valid one expires")

	usedInvite = dbClient.Invite.Query().
		Where(invite.Email(formatEmail("used"))).
		OnlyX(t.Context())
	require.NotNil(t, usedInvite)
	require.NotNil(t, usedInvite.UserID)
}
