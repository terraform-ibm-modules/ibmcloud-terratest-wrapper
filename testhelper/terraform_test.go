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
			"test4": []string{"12", "123"},
		}

		// Extract a slice of the keys for use in the test.
		outputKeys := []string{"test1", "test2", "test3", "test4"}

		foundValues, missingKeys := GetTerraformOutputs(t, outputs, outputKeys...)
		assert.Equal(t, len(outputKeys), len(foundValues))
		assert.Empty(t, missingKeys)
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
		outputKeys := []string{"test1", "test2", "test3", "test4"}

		failedT := new(testing.T)
		foundValues, missingKeys := GetTerraformOutputs(failedT, outputs, outputKeys...)
		assert.True(t, failedT.Failed(), "test4 should have caused an error because it is nil")

		assert.Contains(t, missingKeys, "test4")
		assert.Equal(t, len(outputKeys)-1, len(foundValues))
	})

	t.Run("Not all outputs exist", func(t *testing.T) {
		// Generate a map of static key-value pairs for testing purposes.
		outputs := map[string]interface{}{
			"test1": "1234",
			"test2": "5678",
			"test3": "91011",
		}

		// Extract a slice of the keys for use in the test.
		outputKeys := []string{"test1", "test2", "test3", "test4"}

		failedT := new(testing.T)
		foundValues, missingKeys := GetTerraformOutputs(failedT, outputs, outputKeys...)
		assert.True(t, failedT.Failed(), "test4 should have caused an error because it is missing")

		assert.Equal(t, len(foundValues), len(outputKeys)-1)
		assert.Contains(t, missingKeys, "test4")
		assert.NotContains(t, missingKeys, "test1")
		assert.NotContains(t, missingKeys, "test2")
		assert.NotContains(t, missingKeys, "test3")
	})
}
