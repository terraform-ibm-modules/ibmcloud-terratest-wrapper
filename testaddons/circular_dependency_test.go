package testaddons

import (
	"testing"

	core "github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/project-go-sdk/projectv1"
	"github.com/stretchr/testify/assert"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/common"
)

// TestCircularDependencyDetection tests the circular dependency detection logic
func TestCircularDependencyDetection(t *testing.T) {
	logger := common.CreateSmartAutoBufferingLogger(t.Name(), false)

	// Create test options
	options := &TestAddonOptions{
		Testing: t,
		Logger:  logger,
	}

	t.Run("DetectSimpleCircularDependency", func(t *testing.T) {
		// Simulate two configs that depend on each other
		awaitingConfigs := []ConfigDependencyInfo{
			{
				ID:   "config-a-id",
				Name: "deploy-arch-ibm-event-notifications",
				InputReferences: []string{
					"ref:/configs/config-b-id/outputs/some_output",
				},
				InputFieldReferences: map[string]string{
					"cloud_logs_instance_name": "ref:/configs/config-b-id/outputs/some_output",
				},
			},
			{
				ID:   "config-b-id",
				Name: "deploy-arch-ibm-cloud-logs",
				InputReferences: []string{
					"ref:/configs/config-a-id/outputs/another_output",
				},
				InputFieldReferences: map[string]string{
					"existing_cloud_logs_instance_crn": "ref:/configs/config-a-id/outputs/another_output",
				},
			},
		}

		cycles := options.detectCircularDependencies(awaitingConfigs)

		assert.Len(t, cycles, 1, "Should detect exactly one circular dependency")
		assert.Contains(t, cycles[0], "deploy-arch-ibm-event-notifications", "Should include event notifications in cycle")
		assert.Contains(t, cycles[0], "deploy-arch-ibm-cloud-logs", "Should include cloud logs in cycle")
		assert.Contains(t, cycles[0], "→", "Should show dependency direction")

		// Verify actual field names are shown instead of "unknown_output"
		assert.Contains(t, cycles[0], "cloud_logs_instance_name", "Should show actual input field name")
		assert.Contains(t, cycles[0], "existing_cloud_logs_instance_crn", "Should show actual input field name")
		assert.NotContains(t, cycles[0], "unknown_output", "Should not show unknown_output anymore")
		assert.NotContains(t, cycles[0], "unknown_input", "Should not show unknown_input anymore")

		// Verify that the error message is well-formatted and informative
		assert.Contains(t, cycles[0], "🔍 CIRCULAR DEPENDENCY DETECTED", "Should have clear header")
		assert.Contains(t, cycles[0], "💡 RESOLUTION OPTIONS", "Should provide resolution guidance")
	})

	t.Run("NoCircularDependency", func(t *testing.T) {
		// Simulate configs with no circular dependencies
		awaitingConfigs := []ConfigDependencyInfo{
			{
				ID:   "config-a-id",
				Name: "deploy-arch-ibm-event-notifications",
				InputReferences: []string{
					"ref:/configs/external-config/outputs/some_output",
				},
				InputFieldReferences: map[string]string{
					"some_input": "ref:/configs/external-config/outputs/some_output",
				},
			},
			{
				ID:   "config-b-id",
				Name: "deploy-arch-ibm-cloud-logs",
				InputReferences: []string{
					"ref:/configs/another-external-config/outputs/another_output",
				},
				InputFieldReferences: map[string]string{
					"another_input": "ref:/configs/another-external-config/outputs/another_output",
				},
			},
		}

		cycles := options.detectCircularDependencies(awaitingConfigs)

		assert.Len(t, cycles, 0, "Should not detect any circular dependencies")
	})

	t.Run("ParseConfigIDFromReference", func(t *testing.T) {
		tests := []struct {
			reference string
			expected  string
		}{
			{"ref:/configs/abc-123/outputs/test_output", "abc-123"},
			{"ref:/configs/config-with-dashes/outputs/name", "config-with-dashes"},
			{"ref:/configs/simple/outputs/value", "simple"},
			{"invalid-reference", ""},
			{"ref:/something/else", ""},
		}

		for _, test := range tests {
			result := options.parseConfigIDFromReference(test.reference)
			assert.Equal(t, test.expected, result, "Should correctly parse config ID from reference: %s", test.reference)
		}
	})

	t.Run("ParseReferenceDetails", func(t *testing.T) {
		tests := []struct {
			reference        string
			expectedValid    bool
			expectedConfigID string
			expectedType     string
			expectedField    string
		}{
			{"ref:/configs/config-b-id/outputs/some_output", true, "config-b-id", "outputs", "some_output"},
			{"ref:/configs/config-a-id/outputs/another_output", true, "config-a-id", "outputs", "another_output"},
			{"ref:/configs/config-c-id/inputs/input_field", true, "config-c-id", "inputs", "input_field"},
			{"invalid-reference", false, "", "", ""},
		}

		for _, test := range tests {
			result := options.parseReferenceDetails(test.reference)
			assert.Equal(t, test.expectedValid, result.IsValid, "IsValid should match for reference: %s", test.reference)
			if test.expectedValid {
				assert.Equal(t, test.expectedConfigID, result.ConfigID, "ConfigID should match for reference: %s", test.reference)
				assert.Equal(t, test.expectedType, result.ReferenceType, "ReferenceType should match for reference: %s", test.reference)
				assert.Equal(t, test.expectedField, result.FieldName, "FieldName should match for reference: %s", test.reference)
			}
		}
	})

	t.Run("FindInputFieldNameFromReference", func(t *testing.T) {
		testConfig := ConfigDependencyInfo{
			ID:   "test-config",
			Name: "test-config-name",
			InputFieldReferences: map[string]string{
				"cloud_logs_instance_name":         "ref:/configs/config-b-id/outputs/some_output",
				"existing_cloud_logs_instance_crn": "ref:/configs/config-a-id/outputs/another_output",
			},
		}

		tests := []struct {
			reference string
			expected  string
		}{
			{"ref:/configs/config-b-id/outputs/some_output", "cloud_logs_instance_name"},
			{"ref:/configs/config-a-id/outputs/another_output", "existing_cloud_logs_instance_crn"},
			{"ref:/configs/unknown/outputs/unknown", "unknown_input"},
		}

		for _, test := range tests {
			result := options.findInputFieldNameFromReference(testConfig, test.reference)
			assert.Equal(t, test.expected, result, "Should correctly find input field name for reference: %s", test.reference)
		}
	})

	t.Run("EmptyAwaitingConfigs", func(t *testing.T) {
		cycles := options.detectCircularDependencies([]ConfigDependencyInfo{})
		assert.Nil(t, cycles, "Should return nil for empty config list")
	})

	t.Run("StrictModeTrue_CircularDependencyFails", func(t *testing.T) {
		// Test that StrictMode=true (default) causes circular dependency to fail
		options.StrictMode = core.BoolPtr(true)

		awaitingConfigs := []ConfigDependencyInfo{
			{
				ID:   "config-a-id",
				Name: "deploy-arch-ibm-event-notifications",
				InputReferences: []string{
					"ref:/configs/config-b-id/outputs/some_output",
				},
				InputFieldReferences: map[string]string{
					"cloud_logs_instance_name": "ref:/configs/config-b-id/outputs/some_output",
				},
			},
			{
				ID:   "config-b-id",
				Name: "deploy-arch-ibm-cloud-logs",
				InputReferences: []string{
					"ref:/configs/config-a-id/outputs/another_output",
				},
				InputFieldReferences: map[string]string{
					"existing_cloud_logs_instance_crn": "ref:/configs/config-a-id/outputs/another_output",
				},
			},
		}

		cycles := options.detectCircularDependencies(awaitingConfigs)
		assert.Len(t, cycles, 1, "Should detect circular dependency in strict mode")
		assert.Contains(t, cycles[0], "cloud_logs_instance_name", "Should show actual field names")
	})

	t.Run("ActualScenarioInputToInputReferences", func(t *testing.T) {
		// Test the actual scenario from the logs where we have input-to-input references
		awaitingConfigs := []ConfigDependencyInfo{
			{
				ID:   "activity-tracker-id",
				Name: "deploy-arch-ibm-activity-tracker",
				InputReferences: []string{
					"ref:/configs/cloud-logs-id/outputs/cloud_logs_crn",
				},
				InputFieldReferences: map[string]string{
					"existing_cloud_logs_instance_crn": "ref:/configs/cloud-logs-id/outputs/cloud_logs_crn",
				},
			},
			{
				ID:   "cloud-logs-id",
				Name: "deploy-arch-ibm-cloud-logs",
				InputReferences: []string{
					"ref:/configs/activity-tracker-id/inputs/cloud_logs_instance_name",
				},
				InputFieldReferences: map[string]string{
					"cloud_logs_instance_name": "ref:/configs/activity-tracker-id/inputs/cloud_logs_instance_name",
				},
			},
		}

		cycles := options.detectCircularDependencies(awaitingConfigs)

		assert.Len(t, cycles, 1, "Should detect exactly one circular dependency")
		assert.Contains(t, cycles[0], "deploy-arch-ibm-activity-tracker", "Should include activity tracker in cycle")
		assert.Contains(t, cycles[0], "deploy-arch-ibm-cloud-logs", "Should include cloud logs in cycle")
		assert.Contains(t, cycles[0], "→", "Should show dependency direction")

		// Verify that it correctly shows input-to-input references
		assert.Contains(t, cycles[0], "cloud_logs_instance_name", "Should show actual input field name")
		assert.Contains(t, cycles[0], "existing_cloud_logs_instance_crn", "Should show actual input field name")
		assert.Contains(t, cycles[0], ".input:", "Should show input reference type")
		assert.Contains(t, cycles[0], ".output:", "Should show output reference type")
		assert.Contains(t, cycles[0], "💡 RESOLUTION OPTIONS", "Should provide resolution guidance")
	})

	t.Run("StrictModeDetectionConsistency", func(t *testing.T) {
		// Test that detectCircularDependencies works consistently regardless of StrictMode
		// (StrictMode handling is done by the caller, not by detectCircularDependencies itself)

		awaitingConfigs := []ConfigDependencyInfo{
			{
				ID:   "config-a-id",
				Name: "deploy-arch-ibm-event-notifications",
				InputReferences: []string{
					"ref:/configs/config-b-id/outputs/some_output",
				},
				InputFieldReferences: map[string]string{
					"cloud_logs_instance_name": "ref:/configs/config-b-id/outputs/some_output",
				},
			},
			{
				ID:   "config-b-id",
				Name: "deploy-arch-ibm-cloud-logs",
				InputReferences: []string{
					"ref:/configs/config-a-id/outputs/another_output",
				},
				InputFieldReferences: map[string]string{
					"existing_cloud_logs_instance_crn": "ref:/configs/config-a-id/outputs/another_output",
				},
			},
		}

		// Test with StrictMode=true
		options.StrictMode = core.BoolPtr(true)
		cyclesStrict := options.detectCircularDependencies(awaitingConfigs)

		// Test with StrictMode=false
		options.StrictMode = core.BoolPtr(false)
		cyclesNonStrict := options.detectCircularDependencies(awaitingConfigs)

		// detectCircularDependencies should return the same result regardless of StrictMode
		assert.Len(t, cyclesStrict, 1, "Should detect circular dependency in strict mode")
		assert.Len(t, cyclesNonStrict, 1, "Should detect circular dependency in non-strict mode")

		// Both results should contain the key elements (order may vary)
		assert.Contains(t, cyclesStrict[0], "cloud_logs_instance_name", "Should show actual field names")
		assert.Contains(t, cyclesStrict[0], "existing_cloud_logs_instance_crn", "Should show actual field names")
		assert.Contains(t, cyclesNonStrict[0], "cloud_logs_instance_name", "Should show actual field names")
		assert.Contains(t, cyclesNonStrict[0], "existing_cloud_logs_instance_crn", "Should show actual field names")

		// Both should contain the circular dependency marker
		assert.Contains(t, cyclesStrict[0], "🔍 CIRCULAR DEPENDENCY DETECTED", "Should have circular dependency header")
		assert.Contains(t, cyclesNonStrict[0], "🔍 CIRCULAR DEPENDENCY DETECTED", "Should have circular dependency header")
	})

	t.Run("FindUnresolvedReferences", func(t *testing.T) {
		// Mock existing configs
		existingConfigs := []projectv1.ProjectConfigSummary{
			{ID: stringPtr("existing-config-1")},
			{ID: stringPtr("existing-config-2")},
		}

		// Mock awaiting configs with some unresolved references
		awaitingConfigs := []ConfigDependencyInfo{
			{
				ID:   "config-a",
				Name: "deploy-arch-ibm-event-notifications",
				InputReferences: []string{
					"ref:/configs/existing-config-1/outputs/valid_output", // This should resolve
					"ref:/configs/non-existent-config/outputs/bad_output", // This should not resolve
				},
				InputFieldReferences: map[string]string{
					"valid_input":   "ref:/configs/existing-config-1/outputs/valid_output",
					"invalid_input": "ref:/configs/non-existent-config/outputs/bad_output",
				},
			},
		}

		unresolvedRefs := options.findUnresolvedReferences(awaitingConfigs, existingConfigs)

		assert.Len(t, unresolvedRefs, 1, "Should find exactly one unresolved reference")
		assert.Contains(t, unresolvedRefs[0], "deploy-arch-ibm-event-notifications", "Should include config name")
		assert.Contains(t, unresolvedRefs[0], "non-existent-config", "Should include non-existent config ID")
	})
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}
