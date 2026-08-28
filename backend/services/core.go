package services

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/common/dbcommon"
	"github.com/NicoClack/cryptic-stash/backend/core"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/ent/user"
	"github.com/google/uuid"
)

type Core struct {
	App       *common.App
	adminCode *core.AdminCode
	adminID   uuid.UUID
	mu        sync.Mutex
}

func NewCore(app *common.App) *Core {
	return &Core{
		App:       app,
		adminCode: new(core.NewAdminCode(app.Clock)),
	}
}

func (service *Core) Init() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	adminID, stdErr := dbcommon.WithReadTx(
		ctx, service.App.Database,
		func(tx *ent.Tx, ctx context.Context) (uuid.UUID, error) {
			return tx.User.Query().
				Where(user.Username(common.AdminUsername)).
				OnlyID(ctx)
		},
	)
	cancel()
	if stdErr != nil {
		log.Fatalf("failed to get admin user ID. error:\n%v", stdErr.Error())
	}
	service.adminID = adminID
}

func (service *Core) AdminID() uuid.UUID {
	if service.adminID == uuid.Nil {
		panic("Core.AdminID: adminID is uuid.Nil, was Core.Init() called?")
	}
	return service.adminID
}

func (service *Core) maybeRotateAdminCode() {
	service.adminCode.MaybeRotate(service.App.Clock.Now(), service.App.Env.ADMIN_CODE_ROTATION_INTERVAL)
}
func (service *Core) CheckAdminCode(givenCode string) bool {
	service.mu.Lock()
	defer service.mu.Unlock()

	service.maybeRotateAdminCode()
	return core.CheckAdminCode(givenCode, *service.adminCode, service.App.Logger)
}
func (service *Core) CheckAdminCredentials(password string, totpCode string) bool {
	return core.CheckAdminCredentials(
		password,
		totpCode,
		service.App.Env.ADMIN_PASSWORD_HASH,
		service.App.Env.ADMIN_PASSWORD_SALT,
		service.App.Env.ADMIN_PASSWORD_HASH_SETTINGS,
		service.App.Env.ADMIN_TOTP_SECRET,
	)
}
func (service *Core) GetAdminCode(password string, totpCode string) (string, bool) {
	if !service.CheckAdminCredentials(password, totpCode) {
		return "", false
	}

	service.mu.Lock()
	defer service.mu.Unlock()

	service.maybeRotateAdminCode()
	return service.adminCode.String(), true
}

func (service *Core) RandomAuthCode() []byte {
	return core.RandomAuthCode()
}
func (service *Core) SendActiveDownloadSessionReminders(ctx context.Context) common.WrappedError {
	return core.SendActiveDownloadSessionReminders(
		ctx, service.App.Clock, service.App.Messengers,
	)
}
func (service *Core) DeleteExpiredDownloadSessions(ctx context.Context) common.WrappedError {
	return core.DeleteExpiredDownloadSessions(ctx, service.App.Clock)
}
func (service *Core) InvalidateDownloadSessionsForStash(stashID uuid.UUID, ctx context.Context) common.WrappedError {
	return core.InvalidateDownloadSessionsForStash(stashID, ctx, service.App.Clock)
}
func (service *Core) IsUserSufficientlyNotified(downloadSessionOb *ent.DownloadSession) bool {
	return core.IsUserSufficientlyNotified(
		downloadSessionOb,
		service.App.Messengers,
		service.App.Logger,
		service.App.Clock, service.App.Env,
	)
}
func (service *Core) IsStashLocked(stashOb *ent.Stash) bool {
	return core.IsStashLocked(stashOb, service.App.Clock)
}

func (service *Core) Encrypt(data []byte, encryptionKey []byte) ([]byte, common.WrappedError) {
	if len(service.App.Env.STASH_ENCRYPTION_KEY) > 0 {
		innerEncrypted, wrappedErr := core.Encrypt(data, service.App.Env.STASH_ENCRYPTION_KEY)
		if wrappedErr != nil {
			return nil, wrappedErr
		}
		data = innerEncrypted
	}

	return core.Encrypt(data, encryptionKey)
}

func (service *Core) Decrypt(encrypted []byte, encryptionKey []byte) ([]byte, common.WrappedError) {
	innerEncrypted, wrappedErr := core.Decrypt(encrypted, encryptionKey)
	if wrappedErr != nil {
		return nil, wrappedErr
	}

	if len(service.App.Env.STASH_ENCRYPTION_KEY) < 1 {
		return innerEncrypted, nil
	}

	if len(innerEncrypted) < core.GCMNonceSize {
		return nil, ErrWrapperDecrypt.Wrap(
			ErrMissingInnerEncryptionNonce,
		)
	}
	return core.Decrypt(
		innerEncrypted,
		service.App.Env.STASH_ENCRYPTION_KEY,
	)
}

func (service *Core) GenerateSalt() []byte {
	return core.GenerateSalt()
}
func (service *Core) GenerateEncryptionKey() []byte {
	return core.GenerateEncryptionKey()
}

func (service *Core) HashPassword(password string, salt []byte, settings *common.PasswordHashSettings) []byte {
	return core.HashPassword(password, salt, settings)
}
