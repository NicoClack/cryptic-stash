package schema

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"slices"
	"sync"

	"golang.org/x/crypto/hkdf"
)

const (
	GCMNonceSize = 12
)

var keyNames = []string{
	"auth_1",
	"stash_1",
	"job_1",
	"user_messenger_1",
	"security_pii_logging_1", // e.g IPs and user agents
}
var encryptionKeys = map[string][]byte{}
var initOnce sync.Once

func Init(baseEncryptionKey []byte) {
	if len(baseEncryptionKey) != 32 {
		log.Fatal("Base encryption key must be 32 bytes")
	}

	initOnce.Do(func() {
		for _, keyName := range keyNames {
			hkdf := hkdf.New(sha256.New, baseEncryptionKey, nil, []byte(keyName))
			key := make([]byte, 32)
			_, stdErr := io.ReadFull(hkdf, key)
			if stdErr != nil {
				panic(fmt.Sprintf("failed to derive key for %s: %v", keyName, stdErr))
			}

			encryptionKeys[keyName] = key
		}
	})
}

// TODO: dedupe this encryption logic somewhere, the common package can't be imported because of circular deps
// Maybe common won't need to import the schema package once the service interface types are moved to their own package?
func cryptoRandomBytes(length int) []byte {
	salt := make([]byte, length)
	_, stdErr := rand.Read(salt)
	if stdErr != nil {
		panic(fmt.Sprintf("schema.cryptoRandomBytes: couldn't get random byte. error:\n%v", stdErr))
	}
	return salt
}
func Encrypt(data []byte, encryptionKey []byte) ([]byte, error) {
	keyCipher, stdErr := aes.NewCipher(encryptionKey)
	if stdErr != nil {
		return nil, stdErr
	}
	gcm, stdErr := cipher.NewGCM(keyCipher)
	if stdErr != nil {
		return nil, stdErr
	}
	nonce := cryptoRandomBytes(GCMNonceSize)

	encrypted := gcm.Seal(nil, nonce, data, nil)
	return slices.Concat(nonce, encrypted), nil
}
func Decrypt(encrypted []byte, encryptionKey []byte) ([]byte, error) {
	keyCipher, stdErr := aes.NewCipher(encryptionKey)
	if stdErr != nil {
		return nil, stdErr
	}

	gcm, stdErr := cipher.NewGCM(keyCipher)
	if stdErr != nil {
		return nil, stdErr
	}

	decrypted, stdErr := gcm.Open(nil, encrypted[:GCMNonceSize], encrypted[GCMNonceSize:], nil)
	if stdErr != nil {
		return nil, stdErr
	}
	return decrypted, nil
}

type EncryptedField[T any] struct {
	KeyName string
}

func (encryptedField EncryptedField[T]) Value(v any) (driver.Value, error) {
	if v == nil {
		// There isn't much point in encrypting nils because they're easy to guess based on their length.
		// Plus they're easy to modify by just reusing an encrypted nil with the same key name.
		return nil, nil
	}

	val, ok := v.(T)
	if !ok {
		var zero T
		return nil, fmt.Errorf("EncryptedField.Value: unexpected type %T, expected %T", v, zero)
	}

	encryptionKey, ok := encryptionKeys[encryptedField.KeyName]
	if !ok {
		panic("EncryptedField.Value: invalid key name " + encryptedField.KeyName)
	}

	var plaintextBytes []byte

	// Special handling for string and []byte to avoid JSON overhead
	switch any(val).(type) {
	case string:
		plaintextBytes = []byte(any(val).(string))
	case []byte:
		plaintextBytes = any(val).([]byte)
	default:
		var stdErr error
		plaintextBytes, stdErr = json.Marshal(val)
		if stdErr != nil {
			return nil, fmt.Errorf("EncryptedField.Value: failed to marshal JSON data: %w", stdErr)
		}
	}

	encryptedBytes, stdErr := Encrypt(plaintextBytes, encryptionKey)
	if stdErr != nil {
		return nil, fmt.Errorf("EncryptedField.Value: failed to encrypt data: %w", stdErr)
	}

	return encryptedBytes, nil
}

func (encryptedField EncryptedField[T]) Scan(src any) (any, error) {
	if src == nil {
		return nil, nil
	}

	encryptedBytes, ok := src.([]byte)
	if !ok {
		return nil, fmt.Errorf("EncryptedField.Scan: unexpected type %T", src)
	}

	encryptionKey, ok := encryptionKeys[encryptedField.KeyName]
	if !ok {
		panic("EncryptedField.Scan: invalid key name " + encryptedField.KeyName)
	}

	plaintextBytes, stdErr := Decrypt(encryptedBytes, encryptionKey)
	if stdErr != nil {
		return nil, fmt.Errorf("EncryptedField.Scan: failed to decrypt data: %w", stdErr)
	}

	// Special handling for string and []byte to avoid JSON overhead
	var val T
	switch any(&val).(type) {
	case *string:
		return string(plaintextBytes), nil
	case *[]byte:
		return plaintextBytes, nil
	default:
		if stdErr := json.Unmarshal(plaintextBytes, &val); stdErr != nil {
			return nil, fmt.Errorf("EncryptedField.Scan: failed to unmarshal JSON data: %w", stdErr)
		}
		return val, nil
	}
}
