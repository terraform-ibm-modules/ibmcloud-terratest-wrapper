package testaddons

import (
	"testing"

	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/stretchr/testify/assert"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/cloudinfo"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/common"
)

// TestRequiredDependencyValidation tests that the centralized required dependency logic
// works consistently for both manual and permutation tests
func TestRequiredDependencyValidation(t *testing.T) {
	logger := common.CreateSmartAutoBufferingLogger(t.Name(), false)

	// Test 1: No dependencies are force-enabled when CloudInfoService unavailable (all treated as optional)
	t.Run("NoCloudInfoService_AllOptional", func(t *testing.T) {
		options := &TestAddonOptions{
			Testing:    t,
			Logger:     logger,
			StrictMode: core.BoolPtr(true), // Explicit strict mode
			AddonConfig: cloudinfo.AddonConfig{
				OfferingName:   "deploy-arch-ibm-event-notifications",
				OfferingFlavor: "fully-configurable",
				Dependencies: []cloudinfo.AddonConfig{
					{
						OfferingName: "deploy-arch-ibm-cloud-logs", // Should be optional when no CloudInfoService
						Enabled:      core.BoolPtr(false),          // User tries to disable it
						// Don't pre-set IsRequired - let the function determine from catalog
					},
					{
						OfferingName: "deploy-arch-ibm-cos", // This is optional
						Enabled:      core.BoolPtr(false),   // User can disable it
						IsRequired:   core.BoolPtr(false),   // Mock as optional
					},
				},
			},
		}

		// Test the validation logic
		err := options.validateAndProcessRequiredDependencies()
		assert.NoError(t, err, "Validation should not fail in strict mode")

		// Check that cloud-logs dependency remains disabled (since it's optional without catalog info)
		cloudLogsDep := findDependencyByName(options.AddonConfig.Dependencies, "deploy-arch-ibm-cloud-logs")
		assert.NotNil(t, cloudLogsDep, "Cloud logs dependency should exist")
		if cloudLogsDep.Enabled != nil {
			assert.False(t, *cloudLogsDep.Enabled, "Optional dependency should remain disabled")
		}

		// Check that the optional dependency remains disabled
		cosDep := findDependencyByName(options.AddonConfig.Dependencies, "deploy-arch-ibm-cos")
		assert.NotNil(t, cosDep, "COS dependency should exist")
		if cosDep.Enabled != nil {
			assert.False(t, *cosDep.Enabled, "Optional dependency should remain disabled")
		}
	})

	// Test 1b: StrictMode=true - should warn about force-enabled dependencies (using pre-set IsRequired)
	t.Run("StrictMode_True_WithPresetRequired", func(t *testing.T) {
		options := &TestAddonOptions{
			Testing:    t,
			Logger:     logger,
			StrictMode: core.BoolPtr(true), // Explicit strict mode
			AddonConfig: cloudinfo.AddonConfig{
				OfferingName:   "deploy-arch-ibm-event-notifications",
				OfferingFlavor: "fully-configurable",
				Dependencies: []cloudinfo.AddonConfig{
					{
						OfferingName: "deploy-arch-ibm-some-required-service", // Mock required dependency
						Enabled:      core.BoolPtr(false),                     // User tries to disable it
						IsRequired:   core.BoolPtr(true),                      // Pre-set as required
						RequiredBy:   []string{"deploy-arch-ibm-event-notifications"},
					},
					{
						OfferingName: "deploy-arch-ibm-cos", // This is optional
						Enabled:      core.BoolPtr(false),   // User can disable it
						IsRequired:   core.BoolPtr(false),   // Mock as optional
					},
				},
			},
		}

		// Test the validation logic
		err := options.validateAndProcessRequiredDependencies()
		assert.NoError(t, err, "Validation should not fail in strict mode")

		// Check that the required dependency was force-enabled
		requiredDep := findDependencyByName(options.AddonConfig.Dependencies, "deploy-arch-ibm-some-required-service")
		assert.NotNil(t, requiredDep, "Required dependency should exist")
		if requiredDep.Enabled != nil {
			assert.True(t, *requiredDep.Enabled, "Required dependency should be force-enabled")
		}
		if requiredDep.IsRequired != nil {
			assert.True(t, *requiredDep.IsRequired, "Should be marked as required")
		}

		// Check that the optional dependency remains disabled
		cosDep := findDependencyByName(options.AddonConfig.Dependencies, "deploy-arch-ibm-cos")
		assert.NotNil(t, cosDep, "COS dependency should exist")
		if cosDep.Enabled != nil {
			assert.False(t, *cosDep.Enabled, "Optional dependency should remain disabled")
		}
	})

	// Test 2: StrictMode=false - should silently force-enable required dependencies
	t.Run("StrictMode_False_SilentlyForceEnables", func(t *testing.T) {
		options := &TestAddonOptions{
			Testing:    t,
			Logger:     logger,
			StrictMode: core.BoolPtr(false), // Non-strict mode
			AddonConfig: cloudinfo.AddonConfig{
				OfferingName:   "deploy-arch-ibm-event-notifications",
				OfferingFlavor: "fully-configurable",
				Dependencies: []cloudinfo.AddonConfig{
					{
						OfferingName: "deploy-arch-ibm-cloud-logs", // Required dependency
						Enabled:      core.BoolPtr(false),          // User tries to disable it
						IsRequired:   core.BoolPtr(true),           // Mock as required
						RequiredBy:   []string{"deploy-arch-ibm-event-notifications"},
					},
				},
			},
		}

		// Test the validation logic
		err := options.validateAndProcessRequiredDependencies()
		assert.NoError(t, err, "Validation should not fail in non-strict mode")

		// Check that the required dependency was force-enabled
		cloudLogsDep := findDependencyByName(options.AddonConfig.Dependencies, "deploy-arch-ibm-cloud-logs")
		assert.NotNil(t, cloudLogsDep, "Cloud logs dependency should exist")
		if cloudLogsDep.Enabled != nil {
			assert.True(t, *cloudLogsDep.Enabled, "Required dependency should be force-enabled")
		}
		if cloudLogsDep.IsRequired != nil {
			assert.True(t, *cloudLogsDep.IsRequired, "Should be marked as required")
		}
	})

	// Test 3: No CloudInfoService - should skip validation gracefully
	t.Run("NoCloudInfoService_SkipsValidation", func(t *testing.T) {
		options := &TestAddonOptions{
			Testing:          t,
			Logger:           logger,
			CloudInfoService: nil, // No service available
			AddonConfig: cloudinfo.AddonConfig{
				OfferingName: "test-addon",
				Dependencies: []cloudinfo.AddonConfig{
					{
						OfferingName: "some-dependency",
						Enabled:      core.BoolPtr(false),
					},
				},
			},
		}

		// Should not error when CloudInfoService is nil
		err := options.validateAndProcessRequiredDependencies()
		assert.NoError(t, err, "Should handle missing CloudInfoService gracefully")
	})

	// Test 4: Already enabled dependencies - should not change them
	t.Run("AlreadyEnabled_NoChange", func(t *testing.T) {
		options := &TestAddonOptions{
			Testing: t,
			Logger:  logger,
			AddonConfig: cloudinfo.AddonConfig{
				OfferingName: "deploy-arch-ibm-event-notifications",
				Dependencies: []cloudinfo.AddonConfig{
					{
						OfferingName: "deploy-arch-ibm-cloud-logs",
						Enabled:      core.BoolPtr(true), // Already enabled
						IsRequired:   core.BoolPtr(true), // Mock as required
						RequiredBy:   []string{"deploy-arch-ibm-event-notifications"},
					},
				},
			},
		}

		// Test the validation logic
		err := options.validateAndProcessRequiredDependencies()
		assert.NoError(t, err, "Validation should not fail")

		// Check that the dependency remains enabled and unchanged
		cloudLogsDep := findDependencyByName(options.AddonConfig.Dependencies, "deploy-arch-ibm-cloud-logs")
		assert.NotNil(t, cloudLogsDep, "Cloud logs dependency should exist")
		if cloudLogsDep.Enabled != nil {
			assert.True(t, *cloudLogsDep.Enabled, "Dependency should remain enabled")
		}
	})
}

// Helper function to find a dependency by name
func findDependencyByName(dependencies []cloudinfo.AddonConfig, name string) *cloudinfo.AddonConfig {
	for i, dep := range dependencies {
		if dep.OfferingName == name {
			return &dependencies[i]
		}
	}
	return nil
}

// TestPermutationAndManualConsistency demonstrates that both test types now behave consistently
func TestPermutationAndManualConsistency(t *testing.T) {
	// This test demonstrates that both manual and permutation tests now apply
	// the same required dependency business logic

	logger := common.CreateSmartAutoBufferingLogger(t.Name(), false)

	// Simulate what a manual test would do (like TestENCustom)
	manualConfig := cloudinfo.AddonConfig{
		OfferingName:   "deploy-arch-ibm-event-notifications",
		OfferingFlavor: "fully-configurable",
		Dependencies: []cloudinfo.AddonConfig{
			{
				OfferingName: "deploy-arch-ibm-cloud-logs",
				Enabled:      core.BoolPtr(false), // User tries to disable
				IsRequired:   core.BoolPtr(true),  // Mock as required
				RequiredBy:   []string{"deploy-arch-ibm-event-notifications"},
			},
		},
	}

	// Simulate what a permutation test would discover
	permutationConfig := cloudinfo.AddonConfig{
		OfferingName:   "deploy-arch-ibm-event-notifications",
		OfferingFlavor: "fully-configurable",
		Dependencies: []cloudinfo.AddonConfig{
			{
				OfferingName: "deploy-arch-ibm-cloud-logs",
				IsRequired:   core.BoolPtr(true), // Discovered as required
				RequiredBy:   []string{"deploy-arch-ibm-event-notifications"},
				Enabled:      core.BoolPtr(false), // Permutation tries to disable
			},
		},
	}

	// Both should behave the same way after processing
	manualOptions := &TestAddonOptions{
		Testing:     t,
		Logger:      logger,
		StrictMode:  core.BoolPtr(false),
		AddonConfig: manualConfig,
	}

	permutationOptions := &TestAddonOptions{
		Testing:     t,
		Logger:      logger,
		StrictMode:  core.BoolPtr(false),
		AddonConfig: permutationConfig,
	}

	// Process both
	err1 := manualOptions.validateAndProcessRequiredDependencies()
	err2 := permutationOptions.validateAndProcessRequiredDependencies()

	assert.NoError(t, err1, "Manual config processing should succeed")
	assert.NoError(t, err2, "Permutation config processing should succeed")

	// Both should have the dependency force-enabled
	manualDep := findDependencyByName(manualOptions.AddonConfig.Dependencies, "deploy-arch-ibm-cloud-logs")
	permutationDep := findDependencyByName(permutationOptions.AddonConfig.Dependencies, "deploy-arch-ibm-cloud-logs")

	assert.NotNil(t, manualDep, "Manual dependency should exist")
	assert.NotNil(t, permutationDep, "Permutation dependency should exist")

	if manualDep.Enabled != nil {
		assert.True(t, *manualDep.Enabled, "Manual dependency should be force-enabled")
	}
	if permutationDep.Enabled != nil {
		assert.True(t, *permutationDep.Enabled, "Permutation dependency should be force-enabled")
	}

	if manualDep.IsRequired != nil {
		assert.True(t, *manualDep.IsRequired, "Manual dependency should be marked as required")
	}
	if permutationDep.IsRequired != nil {
		assert.True(t, *permutationDep.IsRequired, "Permutation dependency should be marked as required")
	}

	t.Logf("✅ Both manual and permutation tests now apply consistent required dependency logic")
}

// TestCloudLogsOptionalBehavior specifically tests that deploy-arch-ibm-cloud-logs
// is correctly identified as optional (not required) when CloudInfoService is unavailable
func TestCloudLogsOptionalBehavior(t *testing.T) {
	logger := common.CreateSmartAutoBufferingLogger(t.Name(), false)

	// Simulate TestENCustom configuration
	options := &TestAddonOptions{
		Testing: t,
		Logger:  logger,
		AddonConfig: cloudinfo.AddonConfig{
			OfferingName:   "deploy-arch-ibm-event-notifications",
			OfferingFlavor: "fully-configurable",
			Dependencies: []cloudinfo.AddonConfig{
				{
					Enabled:      core.BoolPtr(true),
					OfferingName: "deploy-arch-ibm-kms",
				},
				{
					Enabled:      core.BoolPtr(true),
					OfferingName: "deploy-arch-ibm-activity-tracker",
				},
				{
					Enabled:      core.BoolPtr(true),
					OfferingName: "deploy-arch-ibm-account-infra-base",
				},
				{
					Enabled:      core.BoolPtr(false),
					OfferingName: "deploy-arch-ibm-cloud-monitoring",
				},
				{
					Enabled:      core.BoolPtr(false),
					OfferingName: "deploy-arch-ibm-cos",
				},
				{
					Enabled:      core.BoolPtr(false),
					OfferingName: "deploy-arch-ibm-cloud-logs", // This should remain disabled
				},
			},
		},
	}

	// Test the validation logic
	err := options.validateAndProcessRequiredDependencies()
	assert.NoError(t, err, "Validation should not fail")

	// Verify that deploy-arch-ibm-cloud-logs remains disabled (not force-enabled)
	cloudLogsDep := findDependencyByName(options.AddonConfig.Dependencies, "deploy-arch-ibm-cloud-logs")
	assert.NotNil(t, cloudLogsDep, "Cloud logs dependency should exist")
	assert.False(t, *cloudLogsDep.Enabled, "Cloud logs should remain disabled (not force-enabled as required)")

	// Verify it's not marked as required
	if cloudLogsDep.IsRequired != nil {
		assert.False(t, *cloudLogsDep.IsRequired, "Cloud logs should not be marked as required")
	}

	t.Logf("✅ deploy-arch-ibm-cloud-logs correctly identified as optional and remains disabled")
}
