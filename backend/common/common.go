package common

const (
	AdminUsername       = "admin"
	EncryptionKeyLength = 32  // Required by AES-256
	PasswordSaltLength  = 128 // Overkill but there shouldn't really be any downsides
)
