package auth

import (
	"context"
	"crypto/sha256"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/ent/session"
	"github.com/google/uuid"
	"github.com/jonboulle/clockwork"
)

func CreateSession(
	userID uuid.UUID,
	userAgent string,
	ip string,
	tx *ent.Tx,
	clock clockwork.Clock,
	sessionDuration time.Duration,
	ctx context.Context,
) (*ent.Session, []byte, common.WrappedError) {
	sessionToken := common.CryptoRandomBytes(SessionTokenLength)
	hashedToken := sha256.Sum256(sessionToken)
	now := clock.Now()
	expiresAt := now.Add(sessionDuration)

	sessionOb, stdErr := tx.Session.Create().
		SetCreatedAt(now).
		SetUpdatedAt(now).
		SetUserID(userID).
		SetHashedToken(hashedToken[:]).
		SetExpiresAt(expiresAt).
		SetUserAgent(userAgent).
		SetIP(ip).
		Save(ctx)
	if stdErr != nil {
		return nil, nil, ErrWrapperCreateSession.Wrap(ErrWrapperDatabase.Wrap(stdErr))
	}

	return sessionOb, sessionToken, nil
}

func ValidateSession(
	token []byte,
	tx *ent.Tx,
	clock clockwork.Clock,
	ctx context.Context,
) (*ent.Session, common.WrappedError) {
	hashedToken := sha256.Sum256(token)

	sessionOb, stdErr := tx.Session.Query().
		Where(
			session.HashedToken(hashedToken[:]),
			session.ExpiresAtGT(clock.Now()),
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
