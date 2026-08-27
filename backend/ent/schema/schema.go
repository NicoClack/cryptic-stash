package schema

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"reflect"
	"slices"
	"sync"

	"entgo.io/ent/schema/field"
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

// Nil initialised so that the encryption/decryption logic knows if encryption was initialised,
// since it won't be in setup mode
var encryptionKeys map[string][]byte
var initOnce sync.Once
var encryptionKeysMu sync.RWMutex

var ErrEncryptionUnavailable = errors.New(
	"database encryption is unavailable because the base encryption key is not set, is the server in setup mode?",
)

func Init(baseEncryptionKey []byte) {
	if len(baseEncryptionKey) != 32 {
		log.Fatal("Base encryption key must be 32 bytes")
	}

	initOnce.Do(func() {
		keys := make(map[string][]byte)
		for _, keyName := range keyNames {
			hkdf := hkdf.New(sha256.New, baseEncryptionKey, nil, []byte(keyName))
			key := make([]byte, 32)
			_, stdErr := io.ReadFull(hkdf, key)
			if stdErr != nil {
				panic(fmt.Sprintf("failed to derive key for %s: %v", keyName, stdErr))
			}

			keys[keyName] = key
		}

		encryptionKeysMu.Lock()
		encryptionKeys = keys
		encryptionKeysMu.Unlock()
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
	if len(encrypted) < GCMNonceSize {
		return nil, fmt.Errorf(
			"schema.Decrypt: encrypted value is shorter than GCMNonceSize. length: %d",
			len(encrypted),
		)
	}

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
	// These run before encryption
	Validators []EncryptedValidator[T]
}

func (encryptedField EncryptedField[T]) Value(val T) (driver.Value, error) {
	reflectedValue := reflect.ValueOf(val)
	if !reflectedValue.IsValid() {
		return nil, nil //nolint: nilnil // nil is a valid SQL value
	}
	//nolint:exhaustive // Using as a guard clause
	switch reflectedValue.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		if reflectedValue.IsNil() {
			// There isn't much point in encrypting nils because they're easy to guess based on their length.
			// Plus they're easy to modify by just reusing an encrypted nil with the same key name.
			return nil, nil //nolint: nilnil // nil is a valid SQL value
		}
	}

	for _, validator := range encryptedField.Validators {
		stdErr := validator(val)
		if stdErr != nil {
			return nil, fmt.Errorf("EncryptedField.Value: validation failed: %w", stdErr)
		}
	}

	encryptionKeysMu.RLock()
	initialized := encryptionKeys != nil
	encryptionKey, ok := encryptionKeys[encryptedField.KeyName]
	encryptionKeysMu.RUnlock()
	if !initialized {
		var zero T
		return zero, ErrEncryptionUnavailable
	}
	if !ok {
		panic("EncryptedField.Value: invalid key name " + encryptedField.KeyName)
	}

	var plaintextBytes []byte

	// Special handling for string and []byte (and pointers to them) to avoid JSON overhead
	switch v := any(val).(type) {
	case string:
		plaintextBytes = []byte(v)
	case *string:
		plaintextBytes = []byte(*v)
	case []byte:
		plaintextBytes = v
	case *[]byte:
		plaintextBytes = *v
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

type binaryScanner struct {
	val []byte
}

func (bScanner *binaryScanner) Scan(src any) error {
	switch val := src.(type) {
	case nil:
		bScanner.val = nil
	case []byte:
		bScanner.val = slices.Clone(val)
	case string:
		bScanner.val = []byte(val)
	default:
		return fmt.Errorf("binaryScanner.Scan: unexpected type %T", src)
	}
	return nil
}

func (bScanner *binaryScanner) Value() (driver.Value, error) {
	return bScanner.val, nil
}

func (encryptedField EncryptedField[T]) ScanValue() field.ValueScanner {
	return &binaryScanner{}
}

func (encryptedField EncryptedField[T]) FromValue(rawValue driver.Value) (T, error) {
	if rawValue == nil {
		var zero T
		return zero, nil
	}

	var encryptedBytes []byte
	switch v := rawValue.(type) {
	case []byte:
		encryptedBytes = v
	case *binaryScanner:
		encryptedBytes = v.val
	case string:
		encryptedBytes = []byte(v)
	default:
		var zero T
		return zero, fmt.Errorf("EncryptedField.FromValue: unexpected type %T", rawValue)
	}
	if encryptedBytes == nil {
		var zero T
		return zero, nil
	}

	encryptionKeysMu.RLock()
	initialized := encryptionKeys != nil
	encryptionKey, ok := encryptionKeys[encryptedField.KeyName]
	encryptionKeysMu.RUnlock()
	if !initialized {
		var zero T
		return zero, ErrEncryptionUnavailable
	}
	if !ok {
		panic("EncryptedField.FromValue: invalid key name " + encryptedField.KeyName)
	}

	plaintextBytes, stdErr := Decrypt(encryptedBytes, encryptionKey)
	if stdErr != nil {
		var zero T
		return zero, fmt.Errorf("EncryptedField.FromValue: failed to decrypt data: %w", stdErr)
	}

	// Special handling for string and []byte (including pointers) to avoid JSON overhead
	var val T
	switch v := any(&val).(type) {
	case *string:
		*v = string(plaintextBytes)
		return val, nil
	case **string:
		s := string(plaintextBytes)
		*v = &s
		return val, nil
	case *[]byte:
		*v = plaintextBytes
		return val, nil
	case **[]byte:
		b := plaintextBytes
		*v = &b
		return val, nil
	default:
		if stdErr := json.Unmarshal(plaintextBytes, &val); stdErr != nil {
			var zero T
			return zero, fmt.Errorf("EncryptedField.FromValue: failed to unmarshal JSON data: %w", stdErr)
		}
		return val, nil
	}
}
