package common

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUniqueId(t *testing.T) {
	t.Run("DefaultLength", func(t *testing.T) {
		id := UniqueId()
		assert.Equal(t, 3, len(id), "Expected default UniqueId length to be 3")
		for _, char := range id {
			assert.True(t, strings.ContainsRune(base36chars, char),
				"Character not in base36chars set")
		}
	})

	t.Run("CustomLength", func(t *testing.T) {
		customLength := 8
		id := UniqueId(customLength)
		assert.Equal(t, customLength, len(id), "Expected UniqueId length to match custom length")
		for _, char := range id {
			assert.True(t, strings.ContainsRune(base36chars, char),
				"Character not in base36chars set")
		}
	})
}

func TestGetRandomPasswordWithPrefix(t *testing.T) {
	t.Run("DefaultPrefix", func(t *testing.T) {
		defaultPrefix := "A1a"
		password := GetRandomPasswordWithPrefix()
		assert.True(t, strings.HasPrefix(password, defaultPrefix), "Expected password to start with default prefix 'A1a'")
		assert.Equal(t, 15, len(password), "Expected password length to be 15 (3 prefix + 12 random)")
		for _, char := range password[3:] {
			assert.True(t, strings.ContainsRune(base64chars, char),
				"Character not in base64chars set")
		}
	})

	t.Run("CustomPrefix", func(t *testing.T) {
		customPrefix := "B2b"
		password := GetRandomPasswordWithPrefix(customPrefix)
		assert.True(t, strings.HasPrefix(password, customPrefix),
			"Expected password to start with custom prefix")
		assert.Equal(t, len(customPrefix)+12, len(password),
			"Expected password length to be 15 (len(customPrefix) + 12 random)")
		for _, char := range password[len(customPrefix):] {
			assert.True(t, strings.ContainsRune(base64chars, char),
				"Character not in base64chars set")
		}
	})

	t.Run("WithEmptyPrefix", func(t *testing.T) {
		password := GetRandomPasswordWithPrefix("")
		defaultPrefix := "A1a"
		assert.True(t, strings.HasPrefix(password, defaultPrefix),
			"Expected password to start with default prefix 'A1a' when empty prefix provided")
		assert.Equal(t, 15, len(password),
			"Expected password length to be 15 (3 prefix + 12 random)")
		for _, char := range password[3:] {
			assert.True(t, strings.ContainsRune(base64chars, char),
				"Character not in base64chars set")
		}
	})
}
