package services

import (
	"context"

	"github.com/NicoClack/cryptic-stash/backend/auth"
	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
)

type Auth struct {
	webAuthnApp *webauthn.WebAuthn
	app         *common.App
}

func NewAuth(app *common.App) *Auth {
	return &Auth{
		webAuthnApp: auth.NewWebAuthnApp(app.Env),
		app:         app,
	}
}

func (service *Auth) WebAuthn() *webauthn.WebAuthn {
	return service.webAuthnApp
}

func (service *Auth) StartLogin(
	ctx context.Context,
) (
	uuid.UUID,
	protocol.PublicKeyCredentialRequestOptions,
	common.WrappedError,
) {
	return auth.StartLogin(service.webAuthnApp, service.app.TempKeyValue, service.app.Clock)
}

func (service *Auth) FinishLogin(
	sessionID uuid.UUID,
	parsedResponse *protocol.ParsedCredentialAssertionData,
	ginCtx *gin.Context,
	tx *ent.Tx,
) (*ent.Session, []byte, common.WrappedError) {
	return auth.FinishLogin(
		sessionID,
		parsedResponse,
		ginCtx,
		service.webAuthnApp,
		tx,
		service.app.TempKeyValue,
		service.app.Clock,
		service.app.Logger,
		service.app.Env.SESSION_DURATION,
	)
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
	session *webauthn.SessionData,
	username string,
	parsedCredential *protocol.ParsedCredentialCreationData,
	credentialName string,
	tx *ent.Tx,
	ctx context.Context,
	getUser func(uuid.UUID, *ent.Tx) (*ent.User, error),
) (*ent.Passkey, common.WrappedError) {
	return auth.FinishRegisterPasskey(
		session,
		username,
		parsedCredential,
		credentialName,
		service.webAuthnApp,
		tx,
		service.app.Clock,
		ctx,
		getUser,
	)
}

func (service *Auth) CreateSession(
	userID uuid.UUID,
	passkeyID uuid.UUID,
	userAgent string,
	ip string,
	tx *ent.Tx,
	ctx context.Context,
) (*ent.Session, []byte, common.WrappedError) {
	return auth.CreateSession(
		userID,
		passkeyID,
		userAgent,
		ip,
		tx,
		service.app.Clock,
		service.app.Env.SESSION_DURATION,
		ctx,
	)
}

func (service *Auth) ValidateSession(
	token []byte,
	tx *ent.Tx,
	ctx context.Context,
) (*ent.Session, common.WrappedError) {
	return auth.ValidateSession(token, tx, service.app.Clock, ctx)
}
