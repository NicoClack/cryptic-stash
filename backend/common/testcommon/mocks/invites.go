package mocks

import (
	"context"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/google/uuid"
)

type EmptyInviteService struct{}

func NewEmptyInviteService() *EmptyInviteService {
	return &EmptyInviteService{}
}

func (m *EmptyInviteService) DeleteExpiredInvites(tx *ent.Tx, ctx context.Context) common.WrappedError {
	return nil
}

func (m *EmptyInviteService) CreateInvite(
	email string,
	inviteMessage string,
	expiresIn time.Duration,
	actor *common.Actor,
	tx *ent.Tx,
	ctx context.Context,
) (*ent.Invite, string, common.WrappedError) {
	return nil, "", nil
}

func (m *EmptyInviteService) GetInvite(
	id uuid.UUID,
	code []byte,
	tx *ent.Tx,
	ctx context.Context,
) (*ent.Invite, common.WrappedError) {
	//nolint:nilnil // this is a limited mock
	return nil, nil
}

func (m *EmptyInviteService) GenerateOptions(
	id uuid.UUID,
	code []byte,
	actor *common.Actor,
	tx *ent.Tx,
	ctx context.Context,
) (protocol.PublicKeyCredentialCreationOptions, common.WrappedError) {
	return protocol.PublicKeyCredentialCreationOptions{}, nil
}

func (m *EmptyInviteService) CreateUser(
	id uuid.UUID,
	code []byte,
	credentialName string,
	parsedCredential *protocol.ParsedCredentialCreationData,
	actor *common.Actor,
	tx *ent.Tx,
	ctx context.Context,
) (*ent.User, *ent.Passkey, *ent.Session, []byte, common.WrappedError) {
	return nil, nil, nil, nil, nil
}
