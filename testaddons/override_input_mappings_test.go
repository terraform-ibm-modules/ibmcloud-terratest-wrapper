package testaddons

import (
	"testing"

	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/stretchr/testify/assert"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/cloudinfo"
)

// TestOverrideInputMappingsFlag tests the new OverrideInputMappings functionality
func TestOverrideInputMappingsFlag(t *testing.T) {
	t.Run("DefaultBehaviorPreservesReferences", func(t *testing.T) {
		// Test default behavior (OverrideInputMappings = false)
		options := TestAddonsOptionsDefault(&TestAddonOptions{
			Testing: t,
			Prefix:  "test-preserve",
			AddonConfig: cloudinfo.AddonConfig{
				ConfigID: "test-config-id",
			},
		})

		// Verify default is false (preserve references)
		assert.NotNil(t, options.OverrideInputMappings)
		assert.False(t, *options.OverrideInputMappings, "Default should preserve references")

		// Simulate cached reference information
		options.configInputReferences = map[string]map[string]string{
			"test-config-id": {
				"reference_input": "ref:/configs/other-config/outputs/some_value",
				"regular_input":   "", // Non-reference input
			},
		}

		// Test input merging with cached references
		testCase := AddonTestCase{
			Name: "TestCase",
			Inputs: map[string]interface{}{
				"reference_input": "should_be_ignored",
				"regular_input":   "should_be_applied",
				"new_input":       "should_be_added",
			},
		}

		// Initialize AddonConfig.Inputs
		options.AddonConfig.Inputs = make(map[string]interface{})

		// Simulate the input merging logic
		if testCase.Inputs != nil && len(testCase.Inputs) > 0 {
			if options.AddonConfig.Inputs == nil {
				options.AddonConfig.Inputs = make(map[string]interface{})
			}

			if options.OverrideInputMappings != nil && !*options.OverrideInputMappings {
				configReferences := options.configInputReferences[options.AddonConfig.ConfigID]
				for key, newValue := range testCase.Inputs {
					if referenceValue, isReference := configReferences[key]; isReference && referenceValue != "" {
						// Keep the existing reference value
						options.AddonConfig.Inputs[key] = referenceValue
					} else {
						// Safe to override - not a reference
						options.AddonConfig.Inputs[key] = newValue
					}
				}
			}
		}

		// Verify results
		assert.Equal(t, "ref:/configs/other-config/outputs/some_value", options.AddonConfig.Inputs["reference_input"], "Reference value should be preserved")
		assert.Equal(t, "should_be_applied", options.AddonConfig.Inputs["regular_input"], "Regular input should be applied")
		assert.Equal(t, "should_be_added", options.AddonConfig.Inputs["new_input"], "New input should be added")
	})

	t.Run("ExplicitOverrideBehavior", func(t *testing.T) {
		// Test explicit override behavior (OverrideInputMappings = true)
		options := TestAddonsOptionsDefault(&TestAddonOptions{
			Testing:               t,
			Prefix:                "test-override",
			OverrideInputMappings: core.BoolPtr(true), // Explicitly enable overriding
			AddonConfig: cloudinfo.AddonConfig{
				ConfigID: "test-config-id",
			},
		})

		// Verify override is enabled
		assert.NotNil(t, options.OverrideInputMappings)
		assert.True(t, *options.OverrideInputMappings, "Should override all inputs")

		// Simulate cached reference information (should be ignored)
		options.configInputReferences = map[string]map[string]string{
			"test-config-id": {
				"reference_input": "ref:/configs/other-config/outputs/some_value",
			},
		}

		// Test input merging with override enabled
		testCase := AddonTestCase{
			Name: "TestCase",
			Inputs: map[string]interface{}{
				"reference_input": "should_override_reference",
				"regular_input":   "should_be_applied",
			},
		}

		// Initialize AddonConfig.Inputs
		options.AddonConfig.Inputs = make(map[string]interface{})

		// Simulate the input merging logic
		if testCase.Inputs != nil && len(testCase.Inputs) > 0 {
			if options.AddonConfig.Inputs == nil {
				options.AddonConfig.Inputs = make(map[string]interface{})
			}

			if options.OverrideInputMappings != nil && !*options.OverrideInputMappings {
				// Reference preservation logic (should not execute)
				t.Error("Should not execute reference preservation logic when OverrideInputMappings is true")
			} else {
				// Current behavior - override all inputs
				for key, value := range testCase.Inputs {
					options.AddonConfig.Inputs[key] = value
				}
			}
		}

		// Verify results - all inputs should be overridden
		assert.Equal(t, "should_override_reference", options.AddonConfig.Inputs["reference_input"], "Reference should be overridden when flag is true")
		assert.Equal(t, "should_be_applied", options.AddonConfig.Inputs["regular_input"], "Regular input should be applied")
	})

	t.Run("CopyMethodPreservesFlag", func(t *testing.T) {
		// Test that the copy method preserves the OverrideInputMappings flag
		original := &TestAddonOptions{
			Testing:               &testing.T{},
			OverrideInputMappings: core.BoolPtr(true),
		}

		copied := original.copy()

		assert.NotNil(t, copied.OverrideInputMappings)
		assert.True(t, *copied.OverrideInputMappings, "Copy should preserve OverrideInputMappings flag")

		// Verify they are different pointers
		assert.NotSame(t, original.OverrideInputMappings, copied.OverrideInputMappings, "Should be different pointer instances")
	})

	t.Run("CacheInitializationAndStorage", func(t *testing.T) {
		// Test that configInputReferences cache is properly initialized and used
		options := &TestAddonOptions{
			Testing: t,
			AddonConfig: cloudinfo.AddonConfig{
				ConfigID: "test-config",
			},
		}

		// Cache should be nil initially
		assert.Nil(t, options.configInputReferences)

		// Simulate cache initialization (as done in the GetConfig loop)
		if options.configInputReferences == nil {
			options.configInputReferences = make(map[string]map[string]string)
		}

		// Add reference data
		fieldReferences := map[string]string{
			"input1": "ref:/configs/config-a/outputs/output1",
			"input2": "ref:/configs/config-b/outputs/output2",
		}
		options.configInputReferences["test-config"] = fieldReferences

		// Verify cache content
		assert.NotNil(t, options.configInputReferences)
		assert.Equal(t, fieldReferences, options.configInputReferences["test-config"])
		assert.Equal(t, "ref:/configs/config-a/outputs/output1", options.configInputReferences["test-config"]["input1"])
	})
}
