package core

import (
	"context"
	"fmt"
	"math"
	"slices"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/ent/downloadsession"
	"github.com/NicoClack/cryptic-stash/backend/ent/user"
	"github.com/google/uuid"
	"github.com/jonboulle/clockwork"
)

// Doubled because the bytes are represented as base64
const AuthCodeByteLength = 128

func RandomAuthCode() []byte {
	return common.CryptoRandomBytes(AuthCodeByteLength)
}

func SendActiveDownloadSessionReminders(
	ctx context.Context,
	clock clockwork.Clock,
	messengers common.MessengerService,
) common.WrappedError {
	tx := ent.TxFromContext(ctx)
	if tx == nil {
		return ErrWrapperSendActiveDownloadSessionReminders.Wrap(
			common.ErrNoTxInContext,
		)
	}

	userObs, stdErr := tx.User.Query().
		WithDownloadSessions(func(sessionQuery *ent.DownloadSessionQuery) {
			sessionQuery.
				Where(downloadsession.ValidUntilGT(clock.Now())).
				Order(ent.Asc(downloadsession.FieldValidFrom)).
				Limit(1)
		}).
		All(ctx)
	if stdErr != nil {
		return ErrWrapperSendActiveDownloadSessionReminders.Wrap(
			ErrWrapperDatabase.Wrap(stdErr),
		)
	}

	messages := make([]*common.Message, 0, len(userObs))
	for _, userOb := range userObs {
		downloadSessionObs := userOb.Edges.DownloadSessions
		if len(downloadSessionObs) == 0 {
			continue
		}
		downloadSessionOb := downloadSessionObs[0]
		downloadSessionIDs := make([]uuid.UUID, 0, len(downloadSessionObs))
		for _, downloadSessionOb := range downloadSessionObs {
			downloadSessionIDs = append(downloadSessionIDs, downloadSessionOb.ID)
		}

		messages = append(messages, &common.Message{
			Type:               common.MessageActiveDownloadSessionReminder,
			User:               userOb,
			Time:               downloadSessionOb.ValidFrom,
			DownloadSessionIDs: downloadSessionIDs,
		})
	}
	wrappedErr := messengers.SendBulk(messages, ctx)
	if wrappedErr != nil {
		return ErrWrapperSendActiveDownloadSessionReminders.Wrap(wrappedErr)
	}

	return nil
}

func DeleteExpiredDownloadSessions(ctx context.Context, clock clockwork.Clock) common.WrappedError {
	tx := ent.TxFromContext(ctx)
	if tx == nil {
		return ErrWrapperDeleteExpiredDownloadSessions.Wrap(common.ErrNoTxInContext)
	}

	_, stdErr := tx.DownloadSession.Delete().
		Where(downloadsession.ValidUntilLTE(clock.Now())).
		Exec(ctx)
	if stdErr != nil {
		return ErrWrapperDeleteExpiredDownloadSessions.Wrap(
			ErrWrapperDatabase.Wrap(stdErr),
		)
	}
	return nil
}

func InvalidateUserDownloadSessions(userID uuid.UUID, ctx context.Context, clock clockwork.Clock) common.WrappedError {
	tx := ent.TxFromContext(ctx)
	if tx == nil {
		return ErrWrapperInvalidateUserDownloadSessions.Wrap(common.ErrNoTxInContext)
	}

	_, stdErr := tx.DownloadSession.Delete().
		Where(downloadsession.HasUserWith(user.ID(userID))).
		Exec(ctx)
	if stdErr != nil {
		return ErrWrapperInvalidateUserDownloadSessions.Wrap(
			ErrWrapperDatabase.Wrap(stdErr),
		)
	}
	stdErr = tx.User.UpdateOneID(userID).
		SetDownloadSessionsValidFrom(clock.Now()).
		Exec(ctx)
	if stdErr != nil {
		return ErrWrapperInvalidateUserDownloadSessions.Wrap(
			ErrWrapperDatabase.Wrap(stdErr),
		)
	}
	return nil
}

func IsUserSufficientlyNotified(
	downloadSessionOb *ent.DownloadSession,
	messengers common.MessengerService,
	logger common.Logger,
	clock clockwork.Clock, env *common.Env,
) bool {
	logger = logger.With(
		"downloadSessionID", downloadSessionOb.ID,
		"userID", downloadSessionOb.Edges.User.ID,
	)

	allLoginAlerts := slices.Clone(downloadSessionOb.Edges.LoginAlerts)
	groupedLoginAlerts := make(map[string][]*ent.LoginAlert)
	for _, loginAlert := range allLoginAlerts {
		if loginAlert.Edges.UserMessenger == nil {
			panic(
				fmt.Sprintf(
					"IsUserSufficientlyNotified: LoginAlert missing UserMessenger edge (loginAlertID=%s)",
					loginAlert.ID,
				),
			)
		}
		versionedType := common.GetVersionedType(
			loginAlert.Edges.UserMessenger.Type,
			loginAlert.Edges.UserMessenger.Version,
		)
		groupedLoginAlerts[versionedType] = append(groupedLoginAlerts[versionedType], loginAlert)
	}
	messengerTypes := messengers.GetConfiguredMessengerTypes(downloadSessionOb.Edges.User)
	earliestValidTime := clock.Now().Add(-env.ACTIVE_DOWNLOAD_SESSION_REMINDER_INTERVAL)
	successfulMessengerTypes := []string{}
	// Ignore the supplemental messengers when assessing this
	coreMessengerTypeCount := 0
	for _, messengerType := range messengerTypes {
		messengerDef, ok := messengers.GetPublicDefinition(messengerType)
		if !ok {
			panic(fmt.Sprintf("IsUserSufficientlyNotified: no messenger definition for %s", messengerType))
		}
		if messengerDef.IsSupplemental {
			continue
		}
		coreMessengerTypeCount++

		loginAlerts := groupedLoginAlerts[messengerType]
		confirmedLoginAlerts := []*ent.LoginAlert{}
		for _, alert := range loginAlerts {
			if alert.Confirmed {
				confirmedLoginAlerts = append(confirmedLoginAlerts, alert)
			}
		}

		if len(confirmedLoginAlerts) < env.MIN_SUCCESSFUL_MESSAGE_COUNT {
			logger.Warn(
				"user was not sufficiently notified by one of their configured messengers because it "+
					"didn't successfully send and confirm enough login alerts",
				"messengerType",
				messengerType,
				"loginAlertCount",
				len(loginAlerts),
				"confirmedLoginAlertCount",
				len(confirmedLoginAlerts),
			)
			continue
		}
		mostRecentConfirmedAlert := &ent.LoginAlert{}
		for _, alert := range confirmedLoginAlerts {
			if alert.SentAt.After(mostRecentConfirmedAlert.SentAt) {
				mostRecentConfirmedAlert = alert
			}
		}
		if mostRecentConfirmedAlert.SentAt.Before(earliestValidTime) {
			logger.Warn(
				"user was not sufficiently notified by one of their configured messengers because "+
					"its most recent confirmed alert was too old. are jobs still running? are some messengers failing?",
				"messengerType",
				messengerType,
				"mostRecentAlertTime",
				mostRecentConfirmedAlert.SentAt,
				"earliestValidTime",
				earliestValidTime,
			)
			continue
		}

		successfulMessengerTypes = append(successfulMessengerTypes, messengerType)
	}

	minSuccessfulMessengers := max(int(
		math.Ceil(float64(coreMessengerTypeCount)/float64(2)),
	), 1)
	if len(successfulMessengerTypes) < minSuccessfulMessengers {
		logger.Warn(
			"user was not sufficiently notified because not enough of their core configured messengers "+
				"successfully sent login alerts",
			"configuredMessengerTypes",
			messengerTypes,
			"successfulMessengerTypes",
			successfulMessengerTypes,
			"minSuccessfulMessengers",
			minSuccessfulMessengers,
		)
		return false
	}

	logger.Info(
		"user was sufficiently notified",
		"configuredMessengerTypes", messengerTypes,
		"successfulMessengerTypes", successfulMessengerTypes,
		"minSuccessfulMessengers", minSuccessfulMessengers,
	)
	return true
}

func IsUserLocked(userOb *ent.User, clock clockwork.Clock) bool {
	if userOb.Locked {
		return true
	}
	if userOb.LockedUntil == nil {
		return false
	}
	return clock.Now().Before(*userOb.LockedUntil)
}
