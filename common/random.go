package common

import (
	"bytes"
	"crypto/rand"
	"math/big"
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
