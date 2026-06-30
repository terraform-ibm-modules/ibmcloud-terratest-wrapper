package testhelper

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestIsSanitizationSensitiveValue unit-tests the isSanitizationSensitiveValue helper directly.
func TestIsSanitizationSensitiveValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    interface{}
		expected bool
	}{
		{
			name:     "bool is sensitive",
			value:    true,
			expected: true,
		},
		{
			name:     "non-empty map is sensitive",
			value:    map[string]interface{}{"field": true},
			expected: true,
		},
		{
			name:     "empty map is not sensitive",
			value:    map[string]interface{}{},
			expected: false,
		},
		{
			// This is the exact shape Terraform emits for event_notifications / object_storage
			// in ibm_scc_instance_settings: [{}] means the block exists but no sub-field is sensitive.
			name:     "slice of empty maps is not sensitive",
			value:    []interface{}{map[string]interface{}{}},
			expected: false,
		},
		{
			name:     "slice with non-empty map is sensitive",
			value:    []interface{}{map[string]interface{}{"password": true}},
			expected: true,
		},
		{
			name:     "slice with mixed maps — any non-empty makes it sensitive",
			value:    []interface{}{map[string]interface{}{}, map[string]interface{}{"field": true}},
			expected: true,
		},
		{
			name:     "empty slice is not sensitive",
			value:    []interface{}{},
			expected: false,
		},
		{
			name:     "slice with non-map element takes safe route",
			value:    []interface{}{"unexpected"},
			expected: true,
		},
		{
			name:     "unknown type takes safe route",
			value:    42,
			expected: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, isSanitizationSensitiveValue(tc.value))
		})
	}
}

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
		assert.Equal(t, "output: \x1b[1;34m'test4'\x1b[0m was not expected to be nil", err.Error())
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
		assert.Equal(t, "output: \x1b[1;34m'test4'\x1b[0m was not found", err.Error())
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
		assert.Equal(t, "output: \u001B[1;34m'test3'\u001B[0m was not expected to be blank string\noutput: \x1b[1;34m'test4'\x1b[0m was not found\noutput: \x1b[1;34m'test5'\x1b[0m was not found\noutput: \x1b[1;34m'test6'\x1b[0m was not expected to be nil\noutput: \u001B[1;34m'test7'\u001B[0m was not expected to be blank string", err.Error())
	})
}
