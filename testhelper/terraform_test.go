package testhelper

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetTerraformOutputs(t *testing.T) {
	t.Parallel()
	t.Run("All outputs exist", func(t *testing.T) {
		// Generate a map of static key-value pairs for testing purposes.
		outputs := map[string]interface{}{
			"test1": "1234",
			"test2": "5678",
			"test3": "91011",
			"test4": "98427",
		}

		// Extract a slice of the keys for use in the test.
		expectedKeys := []string{"test1", "test2", "test3", "test4"}

		missingKeys, err := ValidateTerraformOutputs(outputs, expectedKeys...)
		assert.Empty(t, missingKeys)
		assert.NoError(t, err)
	})

	t.Run("All outputs exist, but with nil value", func(t *testing.T) {
		// Generate a map of static key-value pairs for testing purposes.
		outputs := map[string]interface{}{
			"test1": "1234",
			"test2": "5678",
			"test3": "91011",
			"test4": nil,
		}

		// Extract a slice of the keys for use in the test.
		expectedKeys := []string{"test1", "test2", "test3", "test4"}

		missingKeys, err := ValidateTerraformOutputs(outputs, expectedKeys...)
		assert.Contains(t, missingKeys, "test4")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Output test4 was not expected to be nil")
	})

	t.Run("Not all outputs exist", func(t *testing.T) {
		// Generate a map of static key-value pairs for testing purposes.
		outputs := map[string]interface{}{
			"test1": "1234",
			"test2": "5678",
			"test3": "91011",
		}

		// Extract a slice of the keys for use in the test.
		expectedKeys := []string{"test1", "test2", "test3", "test4"}

		missingKeys, err := ValidateTerraformOutputs(outputs, expectedKeys...)
		assert.Contains(t, missingKeys, "test4")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Output test4 was not found")
	})
}
