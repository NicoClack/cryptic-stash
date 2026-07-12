package auth

import (
	"context"
	"errors"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/jonboulle/clockwork"
)

func StartElevation(
	sessionOb *ent.Session, // Must have Passkey preloaded
	userOb *ent.User, // Must have Passkeys preloaded
	webAuthnApp *webauthn.WebAuthn,
	tempKV common.TempKeyValueService,
	clock clockwork.Clock,
) (uuid.UUID, protocol.PublicKeyCredentialRequestOptions, common.WrappedError) {
	// Prevent infinitely resetting the session timeout
	if sessionOb.SuperUserMode {
		return uuid.Nil, protocol.PublicKeyCredentialRequestOptions{}, ErrWrapperStartElevation.Wrap(
			ErrSessionAlreadyElevated,
		)
	}

	eligiblePasskeys, wrappedErr := GetEligiblePasskeysForSuperUserMode(
		sessionOb,
		userOb,
	)
	if wrappedErr != nil {
		return uuid.Nil, protocol.PublicKeyCredentialRequestOptions{}, ErrWrapperStartElevation.Wrap(
			wrappedErr,
		)
	}
	if len(eligiblePasskeys) == 0 {
		return uuid.Nil, protocol.PublicKeyCredentialRequestOptions{}, ErrWrapperStartElevation.Wrap(
			ErrNoSuperEligiblePasskeys,
		)
	}

	allowedCredentials := make([]protocol.CredentialDescriptor, 0, len(eligiblePasskeys))
	for _, passkeyOb := range eligiblePasskeys {
		allowedCredentials = append(allowedCredentials, passkeyOb.Credential.Descriptor())
	}

	creation, sessionData, stdErr := webAuthnApp.BeginDiscoverableLogin(
		webauthn.WithAllowedCredentials(allowedCredentials),
	)
	if stdErr != nil {
		return uuid.Nil, protocol.PublicKeyCredentialRequestOptions{}, ErrWrapperStartElevation.Wrap(stdErr)
	}

	webAuthnSessionID := uuid.New()
	// TODO: what happens if parent transaction fails?
	tempKV.Set(WebAuthnSessionStoreName, webAuthnSessionID.String(), sessionData, sessionData.Expires)

	return webAuthnSessionID, creation.Response, nil
}

func FinishElevation(
	webAuthnSessionID uuid.UUID,
	parsedResponse *protocol.ParsedCredentialAssertionData,
	sessionOb *ent.Session, // Must have passkeys loaded
	ginCtx *gin.Context,
	webAuthnApp *webauthn.WebAuthn,
	tx *ent.Tx,
	tempKV common.TempKeyValueService,
	clock clockwork.Clock,
	logger common.Logger,
	sessionDuration time.Duration,
) common.WrappedError {
	userOb, passkeyOb, cloneWarning, wrappedErr := ValidateLogin(
		webAuthnSessionID,
		parsedResponse,
		ginCtx.Request.Context(),
		webAuthnApp,
		tx,
		tempKV,
		clock,
		logger,
	)
	if wrappedErr != nil {
		return ErrWrapperFinishElevation.Wrap(
			wrappedErr,
		)
	}
	if cloneWarning {
		return ErrWrapperFinishElevation.Wrap(
			errors.New("TODO: rework when proper clone warning handling is implemented"),
		)
	}

	// go-webauthn validates this since we pass .WithAllowedCredentials to BeginDiscoverableLogin,
	// but we'll double check in case the passkeys have changed during the session and to be safe.
	eligiblePasskeys, wrappedErr := GetEligiblePasskeysForSuperUserMode(sessionOb, userOb)
	if wrappedErr != nil {
		return ErrWrapperFinishElevation.Wrap(
			wrappedErr,
		)
	}
	foundMatch := false
	for _, p := range eligiblePasskeys {
		if p.ID == passkeyOb.ID {
			foundMatch = true
			break
		}
	}
	if !foundMatch {
		return ErrWrapperFinishElevation.Wrap(
			ErrNeitherPasskeySuperEligible,
		)
	}

	wrappedErr = ElevateSession(
		sessionOb.ID,
		clock.Now().Add(sessionDuration),
		tx,
		ginCtx.Request.Context(),
	)
	if wrappedErr != nil {
		return ErrWrapperFinishElevation.Wrap(
			wrappedErr,
		)
	}

	return nil
}

// Upgrades an existing session to superuser mode
func ElevateSession(
	sessionID uuid.UUID,
	expiresAt time.Time,
	tx *ent.Tx,
	ctx context.Context,
) common.WrappedError {
	stdErr := tx.Session.UpdateOneID(sessionID).
		SetSuperUserMode(true).
		SetExpiresAt(expiresAt).
		Exec(ctx)
	if stdErr != nil {
		return ErrWrapperElevateSession.Wrap(ErrWrapperDatabase.Wrap(stdErr))
	}
	return nil
}

// At least 1 of the 2 passkeys must allow superuser mode.
// If the user has any second group passkeys, the two passkeys must be from opposite groups.
// Otherwise, any passkey can be used, including the same one twice.
func GetEligiblePasskeysForSuperUserMode(
	sessionOb *ent.Session,
	userOb *ent.User,
) ([]*ent.Passkey, common.WrappedError) {
	if sessionOb.Edges.Passkey == nil {
		panic("GetEligiblePasskeysForSuperUserMode: sessionOb must have Passkey preloaded")
	}
	if userOb.Edges.Passkeys == nil {
		panic("GetEligiblePasskeysForSuperUserMode: userOb must have Passkeys preloaded")
	}

	passkeyObs := userOb.Edges.Passkeys
	hasSecondGroupPasskey := false
	for _, passkeyOb := range passkeyObs {
		if passkeyOb.IsSecondGroup {
			hasSecondGroupPasskey = true
			break
		}
	}

	var eligible []*ent.Passkey
	for _, passkeyOb := range passkeyObs {
		if !(passkeyOb.AllowSuperUser || sessionOb.Edges.Passkey.AllowSuperUser) {
			continue
		}
		if hasSecondGroupPasskey && passkeyOb.IsSecondGroup == sessionOb.Edges.Passkey.IsSecondGroup {
			continue
		}
		eligible = append(eligible, passkeyOb)
	}

	return eligible, nil
}
