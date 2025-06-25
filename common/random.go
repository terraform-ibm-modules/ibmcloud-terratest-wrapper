package common

import (
	"bytes"
	"math/rand"
	"time"
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

	generator := newRand()
	for i := 0; i < idLength; i++ {
		out.WriteByte(base36chars[generator.Intn(len(base36chars))])
	}

	return out.String()
}

// newRand creates a new random number generator, seeding it with the current system time.
func newRand() *rand.Rand {
	return rand.New(rand.NewSource(time.Now().UnixNano()))
}
