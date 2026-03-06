package core

import (
	"github.com/NicoClack/cryptic-stash/backend/common"
	"golang.org/x/crypto/argon2"
)

func GenerateSalt() []byte {
	return common.CryptoRandomBytes(common.PasswordSaltLength)
}

// Returns an encryption key
func HashPassword(password string, salt []byte, settings *common.PasswordHashSettings) []byte {
	return argon2.IDKey(
		[]byte(password), salt,
		settings.Time, settings.Memory,
		settings.Threads, common.EncryptionKeyLength,
	)
}
