package core

import (
	"crypto/subtle"
	"encoding/base64"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/jonboulle/clockwork"
	"github.com/pquerna/otp/totp"
)

// Doubled because the bytes are represented as base64
const AdminCodeByteLength = 128

// TODO: mutex?
//
//nolint:recvcheck
type AdminCode struct {
	Current       []byte
	LastRotatedAt time.Time
}

func NewAdminCode(clock clockwork.Clock) AdminCode {
	return AdminCode{
		Current:       common.CryptoRandomBytes(AdminCodeByteLength),
		LastRotatedAt: clock.Now(),
	}
}
func (adminCode AdminCode) String() string {
	return base64.StdEncoding.EncodeToString(adminCode.Current)
}
func (adminCode *AdminCode) MaybeRotate(now time.Time, rotationInterval time.Duration) {
	if now.Sub(adminCode.LastRotatedAt) < rotationInterval {
		return
	}
	adminCode.Current = common.CryptoRandomBytes(AdminCodeByteLength)
	adminCode.LastRotatedAt = now
}

func CheckAdminCode(givenCode string, expected AdminCode, logger common.Logger) bool {
	if len(expected.Current) != AdminCodeByteLength { // Failsafe in case this is somehow unset or only partly written
		logger.Error(
			"current admin code is the wrong length, this should not happen!",
			"length", len(expected.Current),
			"expectedLength", AdminCodeByteLength,
		)
		return false
	}
	givenBytes, stdErr := base64.StdEncoding.DecodeString(givenCode)
	if stdErr != nil {
		return false
	}
	return subtle.ConstantTimeCompare(givenBytes, expected.Current) == 1
}

func CheckAdminCredentials(
	password string,
	totpCode string,
	expectedHash []byte,
	salt []byte,
	settings *common.PasswordHashSettings,
	totpSecret string,
) bool {
	if len(expectedHash) == 0 || len(salt) == 0 || totpSecret == "" {
		return false
	}

	encryptionKey := HashPassword(password, salt, settings)
	isHashValid := subtle.ConstantTimeCompare(encryptionKey, expectedHash)
	isTotpValidBool := totp.Validate(totpCode, totpSecret)
	isTotpValid := 0
	if isTotpValidBool {
		isTotpValid = 1
	}

	return isHashValid&isTotpValid == 1
}
