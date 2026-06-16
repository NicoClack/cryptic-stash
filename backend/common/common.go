package common

const (
	AdminUsername       = "admin"
	EncryptionKeyLength = 32 // 256 bits, required by AES-256
	PasswordSaltLength  = 32 // 256 bits
)

func Deref[T any](ptr *T, defaultValue T) T {
	if ptr == nil {
		return defaultValue
	}
	return *ptr
}
