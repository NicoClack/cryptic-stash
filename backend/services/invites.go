package services

import (
	"context"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/invites"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/google/uuid"
)

type Invites struct {
	app *common.App
}

func NewInvites(app *common.App) *Invites {
	return &Invites{
		app: app,
	}
}

func (service *Invites) DeleteExpiredInvites(tx *ent.Tx, ctx context.Context) common.WrappedError {
	return invites.DeleteExpiredInvites(tx, ctx, service.app.Clock)
}

func (service *Invites) CreateInvite(
	email string,
	inviteMessage string,
	expiresIn time.Duration,
	actor *common.Actor,
	tx *ent.Tx,
	ctx context.Context,
) (*ent.Invite, string, common.WrappedError) {
	inviteOb, encodedCode, wrappedErr := invites.CreateInvite(
		email, inviteMessage, expiresIn,
		tx, ctx,
		service.app.Messengers, service.app.Clock, service.app.Env,
	)
	if wrappedErr != nil {
		return nil, "", wrappedErr
	}

	service.app.Logger.Info(
		"created invite",
		"inviteID", inviteOb.ID,
		"expiresAt", inviteOb.ExpiresAt,
		"actorID", actor.UserID,
		"actorIP", actor.IP,
		"actorUserAgent", actor.UserAgent,
	)

	return inviteOb, encodedCode, nil
}

func (service *Invites) GetInvite(
	id uuid.UUID,
	code []byte,
	tx *ent.Tx,
	ctx context.Context,
) (*ent.Invite, common.WrappedError) {
	return invites.GetInvite(id, code, tx, ctx, service.app.Clock)
}

func (service *Invites) GenerateOptions(
	id uuid.UUID,
	code []byte,
	actor *common.Actor,
	tx *ent.Tx,
	ctx context.Context,
) (protocol.PublicKeyCredentialCreationOptions, common.WrappedError) {
	options, wrappedErr := invites.GenerateOptions(id, code, tx, ctx, service.app.Auth, service.app.Clock)
	if wrappedErr != nil {
		return protocol.PublicKeyCredentialCreationOptions{}, wrappedErr
	}

	service.app.Logger.Info(
		"generated invite registration options",
		"inviteID", id,
		"actorID", actor.UserID,
		"actorIP", actor.IP,
		"actorUserAgent", actor.UserAgent,
	)

	return options, nil
}

func (service *Invites) CreateUser(
	id uuid.UUID,
	code []byte,
	credentialName string,
	parsedCredential *protocol.ParsedCredentialCreationData,
	actor *common.Actor,
	tx *ent.Tx,
	ctx context.Context,
) (*ent.User, *ent.Passkey, *ent.Session, []byte, common.WrappedError) {
	userOb, passkeyOb, sessionOb, token, wrappedErr := invites.CreateUser(
		id,
		code,
		credentialName,
		parsedCredential,
		actor,
		tx,
		ctx,
		service.app.Auth, service.app.Clock,
	)
	if wrappedErr != nil {
		return nil, nil, nil, nil, wrappedErr
	}

	service.app.Logger.Info(
		"created user from invite",
		"inviteID", id,
		"userID", userOb.ID,
		"actorID", actor.UserID,
		"actorIP", actor.IP,
		"actorUserAgent", actor.UserAgent,
	)

	return userOb, passkeyOb, sessionOb, token, nil
}
