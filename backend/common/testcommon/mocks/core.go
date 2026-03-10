package mocks

import (
	"context"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/google/uuid"
)

type EmptyCoreService struct{}

func NewEmptyCoreService() *EmptyCoreService {
	return &EmptyCoreService{}
}

func (m *EmptyCoreService) CheckAdminCode(givenCode string) bool {
	return false
}
func (m *EmptyCoreService) CheckAdminCredentials(password string, totpCode string) bool {
	return false
}
func (m *EmptyCoreService) GetAdminCode(password string, totpCode string) (string, bool) {
	return "", false
}
func (m *EmptyCoreService) RandomAuthCode() []byte {
	return []byte{}
}
func (m *EmptyCoreService) SendActiveDownloadSessionReminders(ctx context.Context) common.WrappedError {
	return nil
}
func (m *EmptyCoreService) DeleteExpiredDownloadSessions(ctx context.Context) common.WrappedError {
	return nil
}
func (m *EmptyCoreService) InvalidateUserDownloadSessions(userID uuid.UUID, ctx context.Context) common.WrappedError {
	return nil
}
func (m *EmptyCoreService) IsUserSufficientlyNotified(downloadSessionOb *ent.DownloadSession) bool {
	return false
}
func (m *EmptyCoreService) IsUserLocked(userOb *ent.User) bool {
	return false
}
func (m *EmptyCoreService) Encrypt(data []byte, encryptionKey []byte) ([]byte, common.WrappedError) {
	return []byte{}, nil
}
func (m *EmptyCoreService) Decrypt(encrypted []byte, encryptionKey []byte) ([]byte, common.WrappedError) {
	return []byte{}, nil
}
func (m *EmptyCoreService) GenerateSalt() []byte {
	return []byte{}
}
func (m *EmptyCoreService) GenerateEncryptionKey() []byte {
	return []byte{}
}
func (m *EmptyCoreService) HashPassword(password string, salt []byte, settings *common.PasswordHashSettings) []byte {
	return []byte{}
}
