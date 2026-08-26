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

// Note: before using this on a DB column, could it be nullable with a min value > 0 instead?
func ZeroToPtr[T comparable](value T) *T {
	var defaultValue T
	if value == defaultValue {
		return nil
	}
	return new(value)
}
