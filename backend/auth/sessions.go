package auth

import (
	"context"
	"crypto/sha256"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/ent/session"
	"github.com/google/uuid"
)

func CreateSession(
	userID uuid.UUID,
	passkeyID uuid.UUID,
	elevationPasskeyID *uuid.UUID,
	actor *common.Actor,
	tx *ent.Tx,
	sessionDuration time.Duration,
	ctx context.Context,
) (*ent.Session, []byte, common.WrappedError) {
	// Instead of allowing an admin to log in as another user,
	// create an admin endpoint to enable the admin to make the changes themself
	if actor.UserID != uuid.Nil {
		panic(
			"CreateSession: actor.UserID must be uuid.Nil, sessions should never be created on behalf of another user",
		)
	}

	sessionToken := common.CryptoRandomBytes(SessionTokenLength)
	hashedToken := sha256.Sum256(sessionToken)
	now := time.Now()
	expiresAt := now.Add(sessionDuration)

	sessionOb, stdErr := tx.Session.Create().
		SetCreatedAt(now).
		SetUpdatedAt(now).
		SetUserID(userID).
		SetPasskeyID(passkeyID).
		SetNillableElevationPasskeyID(elevationPasskeyID).
		SetIsSudo(elevationPasskeyID != nil).
		SetHashedToken(hashedToken[:]).
		SetExpiresAt(expiresAt).
		SetUserAgent(actor.UserAgent).
		SetIP(actor.IP).
		Save(ctx)
	if stdErr != nil {
		return nil, nil, ErrWrapperCreateSession.Wrap(ErrWrapperDatabase.Wrap(stdErr))
	}

	return sessionOb, sessionToken, nil
}

func ValidateSession(
	token []byte,
	tx *ent.Tx,
	ctx context.Context,
) (*ent.Session, common.WrappedError) {
	hashedToken := sha256.Sum256(token)

	sessionOb, stdErr := tx.Session.Query().
		Where(
			session.HashedToken(hashedToken[:]),
			session.ExpiresAtGT(time.Now()),
		).
		WithUser().
		Only(ctx)
	if stdErr != nil {
		if ent.IsNotFound(stdErr) {
			return nil, ErrWrapperValidateSession.Wrap(ErrInvalidSession)
		}
		return nil, ErrWrapperValidateSession.Wrap(ErrWrapperDatabase.Wrap(stdErr))
	}
	return sessionOb, nil
}
