package common

import (
	"crypto/rand"
	"math/big"
)

const base36chars = "0123456789abcdefghijklmnopqrstuvwxyz"
const base64chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"

// UniqueId returns a unique (ish) id we can attach to prefix variable passed in terraform configuration
// length of the random string to be appended can be controlled by length argument passed to the function
func UniqueId(length ...int) string {

	// Set default length to 3 characters
	idLength := 3

	// Override default if valid length parameter provided
	if len(length) > 0 && length[0] > 0 {
		idLength = length[0]
	}

	return RandomStringGenerator(idLength, base36chars)
}

// GetRandomPasswordWithPrefix generates a random password string that always starts
// with a required prefix.
//
// If a prefix is provided, it is used as the prefix; otherwise the default
// prefix "A1a" is used (handy for satisfying common complexity rules like upper/lower/digit).
//
// The returned password is the prefix plus 12 additional cryptographically secure
// random characters chosen from base64chars (via RandomStringGenerator).
//
// Note: the total length of the returned string is len(prefix) + 12.
func GetRandomPasswordWithPrefix(prefix ...string) string {

	// Default prefix to satisfy common complexity rules
	defaultPrefix := "A1a"

	// Override default prefix if provided
	if len(prefix) > 0 && prefix[0] != "" {
		defaultPrefix = prefix[0]
	}

	length := 12
	randomPass := defaultPrefix + RandomStringGenerator(length, base64chars)

	return randomPass
}

// RandomStringGenerator returns a cryptographically-secure random string of the given
// length, where each character is sampled uniformly at random from characterSet.
//
// If length <= 0 or characterSet is empty, it returns an empty string.
//
// Randomness is sourced from CryptoIntn, which uses crypto/rand,
// making it suitable for passwords, tokens, and other security-sensitive values.
func RandomStringGenerator(length int, characterSet string) string {
	if length <= 0 || len(characterSet) == 0 {
		return ""
	}

	b := make([]byte, length)
	for i := 0; i < length; i++ {
		b[i] = characterSet[CryptoIntn(len(characterSet))]
	}
	return string(b)
}

// CryptoIntn returns a cryptographically-secure random integer n in the range [0, max).
//
// It uses crypto/rand.Int with a uniform distribution over the specified bound.
// This function panics if the underlying random read fails.
//
// Callers must ensure max > 0; passing max <= 0 will cause big.NewInt to panic or
// crypto/rand.Int to error.
func CryptoIntn(max int) int {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		panic(err)
	}
	return int(n.Int64())
}
