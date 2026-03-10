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
	passwordCipher, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return nil, ErrWrapperEncrypt.Wrap(err)
	}
	gcm, err := cipher.NewGCM(passwordCipher)
	if err != nil {
		return nil, ErrWrapperEncrypt.Wrap(err)
	}
	nonce := common.CryptoRandomBytes(GCMNonceSize)

	encrypted := gcm.Seal(nil, nonce, data, nil)
	return slices.Concat(nonce, encrypted), nil
}

func Decrypt(encrypted []byte, encryptionKey []byte) ([]byte, common.WrappedError) {
	passwordCipher, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return nil, ErrWrapperDecrypt.Wrap(err)
	}

	gcm, err := cipher.NewGCM(passwordCipher)
	if err != nil {
		return nil, ErrWrapperDecrypt.Wrap(err)
	}

	decrypted, err := gcm.Open(nil, encrypted[:GCMNonceSize], encrypted[GCMNonceSize:], nil)
	if err != nil {
		return nil, ErrWrapperDecrypt.Wrap(err)
	}
	return decrypted, nil
}
