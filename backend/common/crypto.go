package common

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
)

func CryptoRandomBytes(length int) []byte {
	salt := make([]byte, length)
	_, stdErr := rand.Read(salt)
	if stdErr != nil {
		panic(fmt.Sprintf("common.CryptoRandomBytes: couldn't get random byte. error:\n%v", stdErr))
	}
	return salt
}

func CryptoRandomInt(maxValue int64) int64 {
	n, stdErr := rand.Int(rand.Reader, big.NewInt(maxValue))
	if stdErr != nil {
		panic(fmt.Sprintf("CryptoRandomInt: couldn't get random int. error:\n%v", stdErr))
	}
	return n.Int64()
}

func CryptoRandomAlphaNum(length int) string {
	characters := []rune("abcdefghijklmnopqrstuvwxyz0123456789")

	var builder strings.Builder
	builder.Grow(length)
	for range length {
		builder.WriteRune(characters[CryptoRandomInt(int64(len(characters)))])
	}
	return builder.String()
}
