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
			"test5": []string{"substring", "substring"},
		}

		// Extract a slice of the keys for use in the test.
		expectedKeys := []string{"test1", "test2", "test3", "test4", "test5"}

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
		assert.Equal(t, "Output: \x1b[1;34m'test4'\x1b[0m was not expected to be nil\n", err.Error())
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
		assert.Equal(t, "Output: \x1b[1;34m'test4'\x1b[0m was not found\n", err.Error())
	})

	t.Run("Mixed errors", func(t *testing.T) {
		// Generate a map of static key-value pairs for testing purposes.
		outputs := map[string]interface{}{
			"test1": "1234",
			"test2": "5678",
			"test3": "    ",
			"test6": nil,
			"test7": "",
		}

		// Extract a slice of the keys for use in the test.
		expectedKeys := []string{"test1", "test2", "test3", "test4", "test5", "test6", "test7"}

		missingKeys, err := ValidateTerraformOutputs(outputs, expectedKeys...)
		assert.Contains(t, missingKeys, "test4")
		assert.Error(t, err)
		assert.Equal(t, "Output: \u001B[1;34m'test3'\u001B[0m was not expected to be blank string\nOutput: \x1b[1;34m'test4'\x1b[0m was not found\nOutput: \x1b[1;34m'test5'\x1b[0m was not found\nOutput: \x1b[1;34m'test6'\x1b[0m was not expected to be nil\nOutput: \u001B[1;34m'test7'\u001B[0m was not expected to be blank string\n", err.Error())
	})
}
