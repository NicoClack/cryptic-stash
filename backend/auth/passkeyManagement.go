package auth

import (
	"context"
	"slices"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/common/dbcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/ent/passkey"
	"github.com/NicoClack/cryptic-stash/backend/ent/predicate"
	"github.com/NicoClack/cryptic-stash/backend/ent/session"
	"github.com/google/uuid"
	"github.com/jonboulle/clockwork"
)

func updateOwnPasskey(
	passkeyID uuid.UUID,
	actor *common.Actor,
	tx *ent.Tx,
	extraPredicates ...predicate.Passkey,
) *ent.PasskeyUpdate {
	predicates := slices.Concat(
		[]predicate.Passkey{
			passkey.ID(passkeyID),
			dbcommon.MaybePredicate(
				actor.UserID != uuid.Nil,
				passkey.UserID(actor.UserID),
			),
		},
		extraPredicates,
	)

	return tx.Passkey.Update().Where(predicates...)
}

func RenamePasskey(
	passkeyID uuid.UUID,
	newName string,
	actor *common.Actor,
	tx *ent.Tx,
	ctx context.Context,
	clock clockwork.Clock,
) common.WrappedError {
	count, stdErr := updateOwnPasskey(passkeyID, actor, tx).
		SetUpdatedAt(clock.Now()).
		SetName(newName).
		Save(ctx)
	if stdErr != nil {
		return ErrWrapperRenamePasskey.Wrap(ErrWrapperDatabase.Wrap(stdErr))
	}
	if count == 0 {
		return ErrWrapperRenamePasskey.Wrap(ErrPasskeyNotFound)
	}
	return nil
}

// Constraint: can't disable sudo on both passkeys the session uses.
func SetPasskeyAllowSudo(
	targetPasskeyID uuid.UUID,
	sessionPasskeyID uuid.UUID,
	sessionElevationPasskeyID *uuid.UUID,
	newAllowSudo bool,
	actor *common.Actor,
	tx *ent.Tx,
	ctx context.Context,
	clock clockwork.Clock,
	logger common.Logger,
) common.WrappedError {
	isTargetUsedAsSessionFirst := targetPasskeyID == sessionPasskeyID
	isTargetUsedAsSessionElevation := sessionElevationPasskeyID != nil &&
		targetPasskeyID == *sessionElevationPasskeyID
	if !newAllowSudo && (isTargetUsedAsSessionFirst || isTargetUsedAsSessionElevation) {
		var nonTargetPasskeyID *uuid.UUID
		if !isTargetUsedAsSessionFirst {
			nonTargetPasskeyID = new(sessionPasskeyID)
		} else if !isTargetUsedAsSessionElevation {
			nonTargetPasskeyID = sessionElevationPasskeyID
		}

		if nonTargetPasskeyID == nil {
			// The same passkey was used twice or the session doesn't have an elevation passkey
			return ErrWrapperSetPasskeyAllowSudo.Wrap(ErrPasskeySudoConstraint)
		}
		nonTargetPasskeyOb, stdErr := tx.Passkey.Get(ctx, *nonTargetPasskeyID)
		if stdErr != nil {
			return ErrWrapperSetPasskeyAllowSudo.Wrap(ErrWrapperDatabase.Wrap(stdErr))
		}
		if !nonTargetPasskeyOb.AllowSudo {
			return ErrWrapperSetPasskeyAllowSudo.Wrap(ErrPasskeySudoConstraint)
		}
	}
	// ^ There's a TOCTOU race with this check, but that just means users can theoretically demote
	// both of their passkeys if they spam the API, which doesn't decrease security

	count, stdErr := updateOwnPasskey(targetPasskeyID, actor, tx).
		SetUpdatedAt(clock.Now()).
		SetAllowSudo(newAllowSudo).
		Save(ctx)
	if stdErr != nil {
		return ErrWrapperSetPasskeyAllowSudo.Wrap(ErrWrapperDatabase.Wrap(stdErr))
	}
	if count == 0 {
		return ErrWrapperSetPasskeyAllowSudo.Wrap(ErrPasskeyNotFound)
	}

	if !newAllowSudo {
		count, wrappedErr := demoteInvalidSudoSessions(actor.UserID, tx, ctx)
		if wrappedErr != nil {
			return ErrWrapperSetPasskeyAllowSudo.Wrap(wrappedErr)
		}
		message := "demoted session(s) for user due to passkey demotion"
		if count == 0 {
			message = "no sessions to demote for user after passkey demotion"
		}
		logger.Info(
			message,
			"sessionDemotionCount", count,
			"userID", actor.UserID,
			"targetPasskeyID", targetPasskeyID,
			"isTargetUsedAsSessionFirst", isTargetUsedAsSessionFirst,
			"isTargetUsedAsSessionElevation", isTargetUsedAsSessionElevation,
			"sessionPasskeyID", sessionPasskeyID,
			"sessionElevationPasskeyID", sessionElevationPasskeyID,
			"actorID", actor.UserID,
			"actorIP", actor.IP,
			"actorUserAgent", actor.UserAgent,
		)
	}

	return nil
}

// Demotes any sudo sessions whose passkeys are now both non-sudo, for a given user
func demoteInvalidSudoSessions(
	userID uuid.UUID,
	tx *ent.Tx,
	ctx context.Context,
) (int, common.WrappedError) {
	count, stdErr := tx.Session.Update().
		Where(
			session.UserID(userID),
			session.IsSudo(true),
			session.And(
				session.HasPasskeyWith(passkey.AllowSudo(false)),
				session.Or(
					session.ElevationPasskeyIDIsNil(),
					session.HasElevationPasskeyWith(passkey.AllowSudo(false)),
				),
			),
		).
		SetIsSudo(false).
		ClearElevationPasskeyID().
		Save(ctx)
	if stdErr != nil {
		return 0, ErrWrapperDatabase.Wrap(stdErr)
	}
	return count, nil
}

// Constraints:
//
// When hasExistingSecondGroup is false: passkey must be used by the session
// and the session must be from 2 different passkeys
//
// When hasExistingSecondGroup is true: can't move a passkey used by the session
func MovePasskeyGroup(
	targetPasskeyID uuid.UUID,
	userID uuid.UUID,
	sessionPasskeyID uuid.UUID,
	sessionElevationPasskeyID *uuid.UUID,
	newIsSecondGroup bool,
	actor *common.Actor,
	tx *ent.Tx,
	ctx context.Context,
	clock clockwork.Clock,
) common.WrappedError {
	if actor.UserID != uuid.Nil && actor.UserID != userID {
		return ErrWrapperMovePasskeyGroup.Wrap(ErrUnauthorizedToModifyUser)
	}
	hasExistingSecondGroup, stdErr := tx.Passkey.Query().
		Where(passkey.UserID(userID), passkey.IsSecondGroup(true)).
		Exist(ctx)
	if stdErr != nil {
		return ErrWrapperMovePasskeyGroup.Wrap(ErrWrapperDatabase.Wrap(stdErr))
	}

	if hasExistingSecondGroup {
		if targetPasskeyID == sessionPasskeyID ||
			(sessionElevationPasskeyID != nil && *sessionElevationPasskeyID == targetPasskeyID) {
			return ErrWrapperMovePasskeyGroup.Wrap(ErrPasskeyGroupMoveConstraint)
		}
	} else {
		usedBySession := targetPasskeyID == sessionPasskeyID ||
			(sessionElevationPasskeyID != nil && *sessionElevationPasskeyID == targetPasskeyID)
		if !usedBySession ||
			sessionElevationPasskeyID == nil ||
			*sessionElevationPasskeyID == sessionPasskeyID {
			return ErrWrapperMovePasskeyGroup.Wrap(ErrPasskeyGroupMoveConstraint)
		}
	}
	// ^ There's a TOCTOU race with this check, but that just means users can theoretically put their
	// passkeys in group combinations they haven't proved they can satisfy, it doesn't decrease security.
	// e.g they might be able to enable second group auth with their main passkey and a second lost passkey
	// (which they didn't use for auth)

	count, stdErr := updateOwnPasskey(targetPasskeyID, actor, tx).
		SetUpdatedAt(clock.Now()).
		SetIsSecondGroup(newIsSecondGroup).
		Save(ctx)
	if stdErr != nil {
		return ErrWrapperMovePasskeyGroup.Wrap(ErrWrapperDatabase.Wrap(stdErr))
	}
	if count == 0 {
		return ErrWrapperMovePasskeyGroup.Wrap(ErrPasskeyNotFound)
	}
	return nil
}

// Constraint: can't delete either passkey the session uses.
// TODO: add frontend prompt when sidevation is implemented for passkeys
// whose deletion would make the session unelevateable.
func DeletePasskey(
	passkeyID uuid.UUID,
	sessionID uuid.UUID,
	actor *common.Actor,
	tx *ent.Tx,
	ctx context.Context,
) common.WrappedError {
	// Note: cascade deletes sessions
	count, stdErr := tx.Passkey.Delete().Where(
		passkey.ID(passkeyID),
		dbcommon.MaybePredicate(
			actor.UserID != uuid.Nil,
			passkey.UserID(actor.UserID),
		),
		passkey.Not(
			passkey.Or(
				passkey.HasSessionsWith(session.ID(sessionID)),
				passkey.HasElevatedSessionsWith(session.ID(sessionID)),
			),
		),
	).Exec(ctx)
	if stdErr != nil {
		return ErrWrapperDeletePasskey.Wrap(ErrWrapperDatabase.Wrap(stdErr))
	}
	if count == 0 {
		return ErrWrapperDeletePasskey.Wrap(ErrPasskeyNotFound)
	}
	return nil
}

func DisableTwoGroupAuth(
	userID uuid.UUID,
	actor *common.Actor,
	tx *ent.Tx,
	ctx context.Context,
	clock clockwork.Clock,
) common.WrappedError {
	if actor.UserID != uuid.Nil && actor.UserID != userID {
		return ErrWrapperDisableTwoGroupAuth.Wrap(ErrUnauthorizedToModifyUser)
	}

	stdErr := tx.Passkey.Update().
		Where(
			passkey.UserID(userID),
			passkey.IsSecondGroup(true),
		).
		SetUpdatedAt(clock.Now()).
		SetIsSecondGroup(false).
		Exec(ctx)
	if stdErr != nil {
		return ErrWrapperDisableTwoGroupAuth.Wrap(ErrWrapperDatabase.Wrap(stdErr))
	}
	return nil
}
