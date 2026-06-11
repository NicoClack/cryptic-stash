package core

import (
	"crypto/aes"
	"crypto/cipher"
	"slices"

	"github.com/NicoClack/cryptic-stash/backend/common"
)

const (
	GCMNonceSize = 12
)

// Adapted from: https://tutorialedge.net/golang/go-encrypt-decrypt-aes-tutorial/
func Encrypt(data []byte, encryptionKey []byte) ([]byte, common.WrappedError) {
	keyCipher, stdErr := aes.NewCipher(encryptionKey)
	if stdErr != nil {
		return nil, ErrWrapperEncrypt.Wrap(stdErr)
	}
	gcm, stdErr := cipher.NewGCM(keyCipher)
	if stdErr != nil {
		return nil, ErrWrapperEncrypt.Wrap(stdErr)
	}
	nonce := common.CryptoRandomBytes(GCMNonceSize)

	encrypted := gcm.Seal(nil, nonce, data, nil)
	return slices.Concat(nonce, encrypted), nil
}

func Decrypt(encrypted []byte, encryptionKey []byte) ([]byte, common.WrappedError) {
	keyCipher, stdErr := aes.NewCipher(encryptionKey)
	if stdErr != nil {
		return nil, ErrWrapperDecrypt.Wrap(stdErr)
	}

	gcm, stdErr := cipher.NewGCM(keyCipher)
	if stdErr != nil {
		return nil, ErrWrapperDecrypt.Wrap(stdErr)
	}

	decrypted, stdErr := gcm.Open(nil, encrypted[:GCMNonceSize], encrypted[GCMNonceSize:], nil)
	if stdErr != nil {
		return nil, ErrWrapperDecrypt.Wrap(stdErr)
	}
	return decrypted, nil
}
