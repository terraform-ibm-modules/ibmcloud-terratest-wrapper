package common

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

const base36chars = "0123456789abcdefghijklmnopqrstuvwxyz"

// UniqueId returns a unique (ish) id we can attach to prefix variable passed in terraform configuration
// length of the random string to be appended can be controlled by length argument passed to the function
func UniqueId(length ...int) string {

	// Set default length to 3 characters
	idLength := 3

	// Override default if valid length parameter provided
	if len(length) > 0 && length[0] > 0 {
		idLength = length[0]
	}
	var out bytes.Buffer

	for i := 0; i < idLength; i++ {
		out.WriteByte(base36chars[CryptoIntn(len(base36chars))])
	}

	return out.String()
}

func CryptoIntn(max int) int {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		panic(err)
	}
	return int(n.Int64())
}

// GetRandomAdminPassword generates a random admin password for use in tests
func GetRandomAdminPassword(t *testing.T) string {
	// Generate a 15 char long random string for the admin_pass
	randomBytes := make([]byte, 13)
	_, randErr := rand.Read(randomBytes)
	require.Nil(t, randErr) // do not proceed if we can't gen a random password

	randomPass := "A1a" + base64.URLEncoding.EncodeToString(randomBytes)[:12]

	return randomPass
}
