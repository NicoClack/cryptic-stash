package services

import (
	"context"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/auth"
	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
)

type Auth struct {
	webAuthnApp *webauthn.WebAuthn
	app         *common.App
}

// Note: due to the go-webauthn dependency, this service uses real time instead of app.Clock
func NewAuth(app *common.App) *Auth {
	return &Auth{
		webAuthnApp: auth.NewWebAuthnApp(app.Env),
		app:         app,
	}
}

func (service *Auth) StartLogin(
	ctx context.Context,
) (
	uuid.UUID,
	protocol.PublicKeyCredentialRequestOptions,
	common.WrappedError,
) {
	return auth.StartLogin(service.webAuthnApp, service.app.TempKeyValue)
}

func (service *Auth) FinishLogin(
	sessionID uuid.UUID,
	parsedResponse *protocol.ParsedCredentialAssertionData,
	actor *common.Actor,
	tx *ent.Tx,
	ctx context.Context,
) (*ent.User, *ent.Passkey, *ent.Session, []byte, common.WrappedError) {
	userOb, passkeyOb, sessionOb, sessionToken, wrappedErr := auth.FinishLogin(
		sessionID,
		parsedResponse,
		actor,
		service.webAuthnApp,
		tx,
		service.app.TempKeyValue,
		service.app.Logger,
		service.app.Env.SESSION_DURATION,
		ctx,
	)
	if wrappedErr != nil {
		return nil, nil, nil, nil, wrappedErr
	}

	service.app.Logger.Info(
		"finished login",
		"userID", userOb.ID,
		"passkeyID", passkeyOb.ID,
		"sessionID", sessionOb.ID,
		"actorID", actor.UserID,
		"actorIP", actor.IP,
		"actorUserAgent", actor.UserAgent,
	)

	return userOb, passkeyOb, sessionOb, sessionToken, nil
}

func (service *Auth) GetEligiblePasskeysForSudo(sessionOb *ent.Session, userOb *ent.User) (
	[]*ent.Passkey,
	common.WrappedError,
) {
	return auth.GetEligiblePasskeysForSudo(sessionOb, userOb)
}
func (service *Auth) StartElevation(
	sessionOb *ent.Session, // Must have Passkey preloaded
	userOb *ent.User, // Must have Passkeys preloaded
) (uuid.UUID, protocol.PublicKeyCredentialRequestOptions, common.WrappedError) {
	return auth.StartElevation(
		sessionOb,
		userOb,
		service.webAuthnApp,
		service.app.TempKeyValue,
	)
}

func (service *Auth) FinishElevation(
	webAuthnSessionID uuid.UUID,
	parsedResponse *protocol.ParsedCredentialAssertionData,
	sessionOb *ent.Session, // Must have Passkey preloaded
	actor *common.Actor,
	tx *ent.Tx,
	ctx context.Context,
) common.WrappedError {
	wrappedErr := auth.FinishElevation(
		webAuthnSessionID,
		parsedResponse,
		sessionOb,
		service.webAuthnApp,
		tx,
		service.app.TempKeyValue,
		service.app.Logger,
		service.app.Env.SESSION_DURATION,
		ctx,
	)
	if wrappedErr != nil {
		return wrappedErr
	}

	service.app.Logger.Info(
		"finished elevation",
		"sessionID", sessionOb.ID,
		"actorID", actor.UserID,
		"actorIP", actor.IP,
		"actorUserAgent", actor.UserAgent,
	)

	return nil
}

func (service *Auth) StartRegisterPasskey(
	user webauthn.User,
	ctx context.Context,
) (
	protocol.PublicKeyCredentialCreationOptions,
	*webauthn.SessionData,
	common.WrappedError,
) {
	return auth.StartRegisterPasskey(user, service.webAuthnApp)
}

func (service *Auth) FinishRegisterPasskey(
	credentialName string,
	allowSudo bool,
	isSecondGroup bool,
	username string,
	session *webauthn.SessionData,
	parsedCredential *protocol.ParsedCredentialCreationData,
	tx *ent.Tx,
	ctx context.Context,
	getUser func(userID uuid.UUID, tx *ent.Tx) (*ent.User, error),
) (*ent.Passkey, common.WrappedError) {
	return auth.FinishRegisterPasskey(
		credentialName,
		allowSudo,
		isSecondGroup,
		username,
		session,
		parsedCredential,
		service.webAuthnApp,
		tx,
		ctx,
		getUser,
	)
}

func (service *Auth) CreateSession(
	userID uuid.UUID,
	passkeyID uuid.UUID,
	elevationPasskeyID *uuid.UUID,
	actor *common.Actor,
	tx *ent.Tx,
	ctx context.Context,
) (*ent.Session, []byte, common.WrappedError) {
	return auth.CreateSession(
		userID,
		passkeyID,
		elevationPasskeyID,
		actor,
		tx,
		service.app.Env.SESSION_DURATION,
		ctx,
	)
}

func (service *Auth) ValidateSession(
	token []byte,
	tx *ent.Tx,
	ctx context.Context,
) (*ent.Session, common.WrappedError) {
	return auth.ValidateSession(token, tx, ctx)
}

func (service *Auth) ElevateSession(
	sessionOb *ent.Session,
	elevationPasskeyID uuid.UUID,
	tx *ent.Tx,
	ctx context.Context,
) common.WrappedError {
	return auth.ElevateSession(
		sessionOb.ID,
		elevationPasskeyID,
		time.Now().Add(service.app.Env.SESSION_DURATION),
		tx,
		ctx,
	)
}

func (service *Auth) RenamePasskey(
	passkeyID uuid.UUID,
	newName string,
	actor *common.Actor,
	tx *ent.Tx,
	ctx context.Context,
) common.WrappedError {
	return auth.RenamePasskey(passkeyID, newName, actor, tx, ctx)
}

func (service *Auth) SetPasskeyAllowSudo(
	targetPasskeyID uuid.UUID,
	sessionPasskeyID uuid.UUID,
	sessionElevationPasskeyID *uuid.UUID,
	newAllowSudo bool,
	actor *common.Actor,
	tx *ent.Tx,
	ctx context.Context,
) common.WrappedError {
	return auth.SetPasskeyAllowSudo(
		targetPasskeyID,
		sessionPasskeyID, sessionElevationPasskeyID,
		newAllowSudo, actor, tx, ctx,
		service.app.Logger,
	)
}

func (service *Auth) MovePasskeyGroup(
	targetPasskeyID uuid.UUID,
	targetUserID uuid.UUID,
	sessionPasskeyID uuid.UUID,
	sessionElevationPasskeyID *uuid.UUID,
	newIsSecondGroup bool,
	actor *common.Actor,
	tx *ent.Tx,
	ctx context.Context,
) common.WrappedError {
	return auth.MovePasskeyGroup(
		targetPasskeyID,
		targetUserID, sessionPasskeyID, sessionElevationPasskeyID,
		newIsSecondGroup,
		actor, tx, ctx,
		service.app.Logger,
	)
}

func (service *Auth) DeletePasskey(
	passkeyID uuid.UUID,
	sessionID uuid.UUID,
	actor *common.Actor,
	tx *ent.Tx,
	ctx context.Context,
) common.WrappedError {
	return auth.DeletePasskey(passkeyID, sessionID, actor, tx, ctx)
}

func (service *Auth) DisableTwoGroupAuth(
	userID uuid.UUID,
	actor *common.Actor,
	tx *ent.Tx,
	ctx context.Context,
) common.WrappedError {
	return auth.DisableTwoGroupAuth(userID, actor, tx, ctx)
}
