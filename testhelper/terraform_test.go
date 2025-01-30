package testhelper

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/stretchr/testify/assert"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/logger"
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

func TestSanitizeSensitiveData(t *testing.T) {
	t.Parallel()

	// Define a sensitive variable
	sensitiveValue := "super-secret-password"
	sanitizedValue := "[SENSITIVE]"

	// Create a TestOptions object with the sensitive variable
	options := &TestOptions{
		Testing:      t,
		TerraformDir: "examples/basic",
		TerraformVars: map[string]interface{}{
			"admin_pass": sensitiveValue,
		},
		SensitiveVars: []string{"admin_pass"},
	}

	// Simulate a Terraform plan
	planStruct := &terraform.PlanStruct{
		ResourceChangesMap: map[string]*tfjson.ResourceChange{
			"module.example.aws_instance.example": {
				Address: "module.example.aws_instance.example",
				Change: &tfjson.Change{
					BeforeSensitive: map[string]interface{}{
						"admin_pass": sensitiveValue,
					},
					AfterSensitive: map[string]interface{}{
						"admin_pass": sensitiveValue,
					},
				},
			},
		},
	}

	// Run the CheckConsistency function to sanitize the output
	CheckConsistency(planStruct, options)

	// Verify that the sensitive value is masked in the output
	output := logger.GetLogOutput(t)
	assert.NotContains(t, output, sensitiveValue, "Sensitive value should not be present in the output")
	assert.Contains(t, output, sanitizedValue, "Sanitized value should be present in the output")
}
