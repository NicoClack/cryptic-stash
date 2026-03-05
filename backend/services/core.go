package services

import (
	"context"
	"sync"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/core"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/google/uuid"
)

type Core struct {
	App       *common.App
	adminCode *core.AdminCode
	mu        sync.Mutex
}

func NewCore(app *common.App) *Core {
	return &Core{
		App:       app,
		adminCode: common.Pointer(core.NewAdminCode(app.Clock)),
	}
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
func (service *Core) InvalidateUserDownloadSessions(userID uuid.UUID, ctx context.Context) common.WrappedError {
	return core.InvalidateUserDownloadSessions(userID, ctx, service.App.Clock)
}
func (service *Core) IsUserSufficientlyNotified(downloadSessionOb *ent.DownloadSession) bool {
	return core.IsUserSufficientlyNotified(
		downloadSessionOb,
		service.App.Messengers,
		service.App.Logger,
		service.App.Clock, service.App.Env,
	)
}
func (service *Core) IsUserLocked(userOb *ent.User) bool {
	return core.IsUserLocked(userOb, service.App.Clock)
}

func (service *Core) Encrypt(data []byte, encryptionKey []byte) ([]byte, []byte, common.WrappedError) {
	return core.Encrypt(data, encryptionKey)
}

func (service *Core) Decrypt(encrypted []byte, encryptionKey []byte, nonce []byte) ([]byte, common.WrappedError) {
	return core.Decrypt(encrypted, encryptionKey, nonce)
}

func (service *Core) GenerateSalt() []byte {
	return core.GenerateSalt()
}

func (service *Core) HashPassword(password string, salt []byte, settings *common.PasswordHashSettings) []byte {
	return core.HashPassword(password, salt, settings)
}
