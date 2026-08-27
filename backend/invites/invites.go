package invites

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/auth"
	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/core"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/ent/invite"
	"github.com/NicoClack/cryptic-stash/backend/ent/user"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/google/uuid"
	"github.com/jonboulle/clockwork"
)

func DeleteExpiredInvites(
	tx *ent.Tx,
	ctx context.Context,
	clock clockwork.Clock,
) common.WrappedError {
	_, stdErr := tx.Invite.Delete().
		Where(
			invite.ExpiresAtLTE(clock.Now()),
			invite.UserIDIsNil(),
		).
		Exec(ctx)
	if stdErr != nil {
		return ErrWrapperDeleteExpiredInvites.Wrap(
			ErrWrapperDatabase.Wrap(stdErr),
		)
	}
	return nil
}

func CreateInvite(
	email string,
	inviteMessage string,
	expiresIn time.Duration,
	tx *ent.Tx,
	ctx context.Context,
	messengers common.MessengerService,
	clock clockwork.Clock,
	env *common.Env,
) (*ent.Invite, string, common.WrappedError) {
	expiresIn = min(expiresIn, env.INVITE_MAX_EXPIRY)

	code := core.RandomAuthCode()
	hashed := sha256.Sum256(code)
	encodedCode := base64.RawURLEncoding.EncodeToString(code)
	now := clock.Now()
	expiresAt := now.Add(expiresIn)

	exists, stdErr := tx.User.Query().Where(user.Username(email)).Exist(ctx)
	if stdErr != nil {
		return nil, "", ErrWrapperCreateInvite.Wrap(ErrWrapperDatabase.Wrap(stdErr))
	}
	if exists {
		return nil, "", ErrWrapperCreateInvite.Wrap(ErrUsernameTaken)
	}

	inviteOb, stdErr := tx.Invite.Create().
		SetCreatedAt(now).
		SetUpdatedAt(now).
		SetEmail(email).
		SetHashedCode(hashed[:]).
		SetExpiresAt(expiresAt).
		Save(ctx)
	if stdErr != nil {
		return nil, "", ErrWrapperCreateInvite.Wrap(ErrWrapperDatabase.Wrap(stdErr))
	}

	emailMessengerType, emailMessengerVersion, _ := common.ParseVersionedType(env.EMAIL_MESSENGER_TYPE)
	inMemoryUser, stdErr := newInMemoryUser(email, emailMessengerType, emailMessengerVersion)
	if stdErr != nil {
		return nil, "", ErrWrapperCreateInvite.Wrap(common.AutoWrapError(stdErr))
	}

	wrappedErr := messengers.Send(
		env.EMAIL_MESSENGER_TYPE,
		&common.Message{
			Type:          common.MessageInvite,
			User:          inMemoryUser,
			InviteMessage: inviteMessage,
			URL: getInviteURL(
				inviteOb.ID,
				encodedCode,
				env.FRONTEND_BASE_URL,
			),
		},
		ctx,
	)
	if wrappedErr != nil {
		return nil, "", ErrWrapperCreateInvite.Wrap(wrappedErr)
	}

	return inviteOb, encodedCode, nil
}

func GetInvite(
	id uuid.UUID,
	code []byte,
	tx *ent.Tx,
	ctx context.Context,
	clock clockwork.Clock,
) (*ent.Invite, common.WrappedError) {
	hashed := sha256.Sum256(code)
	inviteOb, stdErr := tx.Invite.Query().
		Where(
			invite.ID(id),
			invite.HashedCode(hashed[:]),
		).
		Only(ctx)
	if stdErr != nil {
		if ent.IsNotFound(stdErr) {
			return nil, ErrWrapperGetInvite.Wrap(ErrInviteNotFound)
		}
		return nil, ErrWrapperGetInvite.Wrap(ErrWrapperDatabase.Wrap(stdErr))
	}

	if inviteOb.UserID != nil || inviteOb.ExpiredReason != nil {
		return nil, ErrWrapperGetInvite.Wrap(ErrInviteUsed)
	}
	if clock.Now().After(inviteOb.ExpiresAt) {
		return nil, ErrWrapperGetInvite.Wrap(ErrInviteExpired)
	}

	return inviteOb, nil
}

func GenerateOptions(
	id uuid.UUID,
	code []byte,
	tx *ent.Tx,
	ctx context.Context,
	authService common.AuthService,
	clock clockwork.Clock,
) (protocol.PublicKeyCredentialCreationOptions, common.WrappedError) {
	inviteOb, wrappedErr := GetInvite(id, code, tx, ctx, clock)
	if wrappedErr != nil {
		return protocol.PublicKeyCredentialCreationOptions{}, ErrWrapperGenerateOptions.Wrap(wrappedErr)
	}

	pendingUserID := uuid.New()
	options, sessionData, wrappedErr := authService.StartRegisterPasskey(
		&auth.TempWebAuthnUser{
			ID:          pendingUserID[:],
			Name:        inviteOb.Email,
			DisplayName: inviteOb.Email,
		},
		ctx,
	)
	if wrappedErr != nil {
		return protocol.PublicKeyCredentialCreationOptions{}, ErrWrapperGenerateOptions.Wrap(wrappedErr)
	}

	_, stdErr := tx.Invite.UpdateOneID(inviteOb.ID).
		SetWebAuthnSession(sessionData).
		Save(ctx)
	if stdErr != nil {
		return protocol.PublicKeyCredentialCreationOptions{}, ErrWrapperGenerateOptions.Wrap(
			ErrWrapperDatabase.Wrap(stdErr),
		)
	}

	return options, nil
}

func CreateUser(
	id uuid.UUID,
	code []byte,
	credentialName string,
	parsedCredential *protocol.ParsedCredentialCreationData,
	actor *common.Actor,
	tx *ent.Tx,
	ctx context.Context,
	authService common.AuthService,
	clock clockwork.Clock,
) (*ent.User, *ent.Passkey, *ent.Session, []byte, common.WrappedError) {
	inviteOb, wrappedErr := GetInvite(id, code, tx, ctx, clock)
	if wrappedErr != nil {
		return nil, nil, nil, nil, ErrWrapperCreateUser.Wrap(wrappedErr)
	}

	exists, stdErr := tx.User.Query().Where(user.Username(inviteOb.Email)).Exist(ctx)
	if stdErr != nil {
		return nil, nil, nil, nil, ErrWrapperCreateUser.Wrap(ErrWrapperDatabase.Wrap(stdErr))
	}
	if exists {
		// It doesn't matter if this leaks the existence of the account
		// as the invite should have only been sent to the owner of this email.
		stdErr = tx.Invite.UpdateOneID(inviteOb.ID).
			SetExpiredReason("username_taken").Exec(ctx)
		if stdErr != nil {
			return nil, nil, nil, nil, ErrWrapperCreateUser.Wrap(ErrWrapperDatabase.Wrap(stdErr))
		}
		return nil, nil, nil, nil, ErrWrapperCreateUser.Wrap(ErrUsernameTaken)
	}

	if inviteOb.WebAuthnSession == nil {
		return nil, nil, nil, nil, ErrWrapperCreateUser.Wrap(ErrNoWebAuthnSession)
	}

	var createdUserOb *ent.User
	passkeyOb, wrappedErr := authService.FinishRegisterPasskey(
		credentialName,
		true,
		false,
		inviteOb.Email,
		inviteOb.WebAuthnSession,
		parsedCredential,
		tx,
		ctx,
		func(pendingUserID uuid.UUID, tx *ent.Tx) (*ent.User, error) {
			now := clock.Now()
			createdUserOb, stdErr = tx.User.Create().
				SetID(pendingUserID).
				SetUsername(inviteOb.Email).
				SetCreatedAt(now).
				SetUpdatedAt(now).
				SetInviteID(inviteOb.ID).
				Save(ctx)
			if stdErr != nil {
				return nil, stdErr
			}
			_, stdErr = tx.Invite.UpdateOneID(inviteOb.ID).
				SetUser(createdUserOb).
				SetWebAuthnSession(nil).
				// ^ We don't need this anymore and the user edge prevents the invite being used twice
				SetUserAgent(new(actor.UserAgent)).
				SetIP(new(actor.IP)).
				Save(ctx)
			if stdErr != nil {
				return nil, stdErr
			}
			return createdUserOb, nil
		},
	)
	if wrappedErr != nil {
		return nil, nil, nil, nil, ErrWrapperCreateUser.Wrap(wrappedErr)
	}

	sessionOb, token, wrappedErr := authService.CreateSession(
		passkeyOb.UserID,
		passkeyOb.ID,
		new(passkeyOb.ID), // Elevate the session to sudo using the passkey that was just registered
		actor,
		tx,
		ctx,
	)
	if wrappedErr != nil {
		return nil, nil, nil, nil, ErrWrapperCreateUser.Wrap(wrappedErr)
	}

	return createdUserOb, passkeyOb, sessionOb, token, nil
}

func newInMemoryUser(
	email string,
	emailMessengerType string, emailMessengerVersion int,
) (*ent.User, error) {
	encodedOptions := json.RawMessage("{}")
	if emailMessengerType != "develop" {
		var stdErr error
		encodedOptions, stdErr = json.Marshal(map[string]string{
			"email": email,
		})
		if stdErr != nil {
			return nil, stdErr
		}
	}
	// TODO: validate options

	return &ent.User{
		Username: email,
		Edges: ent.UserEdges{
			Messengers: []*ent.UserMessenger{
				{
					Type:      emailMessengerType,
					Version:   emailMessengerVersion,
					IsEnabled: true,
					Options:   encodedOptions,
				},
			},
		},
	}, nil
}

func getInviteURL(
	inviteID uuid.UUID,
	code string,
	frontendBaseURL *url.URL,
) string {
	rel := &url.URL{
		Path:     fmt.Sprintf("/invites/%s/", inviteID.String()),
		Fragment: code,
	}
	return frontendBaseURL.ResolveReference(rel).String()
}
