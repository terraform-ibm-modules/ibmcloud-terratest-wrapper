package testhelper

import (
	"github.com/stretchr/testify/assert"
	"io/fs"
	"os"
	"syscall"
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

type ExpectedValues struct {
	Variables map[string]interface{}
	Err       error
}

func TestGetTerraformVariableInfo(t *testing.T) {
	tests := []struct {
		Name         string
		TerraformDir string
		Expected     ExpectedValues
	}{
		{
			Name:         "Empty Directory",
			TerraformDir: "sample/temp_empty",
			Expected: ExpectedValues{
				Variables: map[string]interface{}{}, // No variables expected
				Err:       nil,                      // No error expected
			},
		},
		{
			Name:         "Valid Directory",
			TerraformDir: "sample/terraform/sample_vars",
			Expected: ExpectedValues{
				Variables: map[string]interface{}{
					"var1": map[string]interface{}{
						"type":        "string",
						"description": "Description for var1",
						"default":     "default1",
						"sensitive":   true,
					},
					"var2": map[string]interface{}{
						"type":        "number",
						"description": "",
						"default":     nil,
						"sensitive":   false,
					},
					"var3": map[string]interface{}{
						"type":        "bool",
						"description": "Description for var3",
						"default":     nil,
						"sensitive":   true,
					},
				},
				Err: nil, // No error expected
			},
		},
		{
			Name:         "Commented Variables",
			TerraformDir: "sample/terraform/sample_vars_commented",
			Expected: ExpectedValues{
				Variables: map[string]interface{}{
					"var2": map[string]interface{}{
						"type":        "number",
						"description": "Description for var2",
						"default":     nil,
						"sensitive":   false,
					},
					"var3": map[string]interface{}{
						"type":        "number",
						"description": "",
						"default":     nil,
						"sensitive":   false,
					},
				},
				Err: nil, // No error expected
			},
		},
		{
			Name:         "Invalid Directory",
			TerraformDir: "sample/nonexistent",
			Expected: ExpectedValues{
				Variables: nil, // No variables expected
				Err: &fs.PathError{
					Op:   "open",
					Path: "sample/nonexistent",
					Err:  syscall.Errno(2), // syscall error code 2 is "no such file or directory"
				}, // Expected error
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			// Create a temporary empty directory if the test is "Empty Directory"
			if test.Name == "Empty Directory" {
				dirName, err := os.MkdirTemp(".", "temp")
				assert.NoError(t, err, "Error creating a temporary directory")
				test.TerraformDir = dirName
				defer os.Remove(dirName) // Remove the temporary directory after the test
			}
			variableInfo, err := GetTerraformVariableInfo(test.TerraformDir)

			assert.Equal(t, test.Expected.Variables, variableInfo, "Variables mismatch")
			assert.Equal(t, test.Expected.Err, err, "Error mismatch")
		})
	}
}

type ExpectedOutput struct {
	Outputs map[string]interface{}
	Err     error
}

func TestGetTerraformOutputInfo(t *testing.T) {
	tests := []struct {
		Name         string
		TerraformDir string
		Expected     ExpectedOutput
	}{
		{
			Name:         "Empty Directory",
			TerraformDir: "sample/temp_empty",
			Expected: ExpectedOutput{
				Outputs: map[string]interface{}{}, // No outputs expected
				Err:     nil,                      // No error expected
			},
		},
		{
			Name:         "Valid Directory",
			TerraformDir: "sample/terraform/sample_vars",
			Expected: ExpectedOutput{
				Outputs: map[string]interface{}{
					"output1": map[string]interface{}{
						"description": "Description for output1",
						"value":       "output1_value",
						"sensitive":   false,
					},
					"output2": map[string]interface{}{
						"description": "",
						"value":       "output2_value",
						"sensitive":   true,
					},
					"output3": map[string]interface{}{
						"description": "",
						"value":       "output3_value",
						"sensitive":   false,
					},
				},
				Err: nil, // No error expected
			},
		},
		{
			Name:         "Commented Outputs",
			TerraformDir: "sample/terraform/sample_vars_commented",
			Expected: ExpectedOutput{
				Outputs: map[string]interface{}{},
				Err:     nil, // No error expected
			},
		},
		{
			Name:         "Invalid Directory",
			TerraformDir: "sample/nonexistent",
			Expected: ExpectedOutput{
				Outputs: nil, // No outputs expected
				Err: &fs.PathError{
					Op:   "open",
					Path: "sample/nonexistent",
					Err:  syscall.Errno(2), // syscall error code 2 is "no such file or directory"
				}, // Expected error
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			// Create a temporary empty directory if the test is "Empty Directory"
			if test.Name == "Empty Directory" {
				dirName, err := os.MkdirTemp(".", "temp")
				assert.NoError(t, err, "Error creating a temporary directory")
				test.TerraformDir = dirName
				defer os.Remove(dirName) // Remove the temporary directory after the test
			}
			outputInfo, err := GetTerraformOutputInfo(test.TerraformDir)

			assert.Equal(t, test.Expected.Outputs, outputInfo, "Outputs mismatch")
			assert.Equal(t, test.Expected.Err, err, "Error mismatch")
		})
	}
}
