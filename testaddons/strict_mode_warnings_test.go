package testaddons

import (
	"testing"

	core "github.com/IBM/go-sdk-core/v5/core"
	"github.com/stretchr/testify/assert"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/cloudinfo"
)

// TestStrictModeWarningsInReport tests that strict mode warnings are properly displayed in the final report
func TestStrictModeWarningsInReport(t *testing.T) {
	t.Run("StrictModeWarningsDisplayed", func(t *testing.T) {

		// Create test results with strict mode warnings
		results := []PermutationTestResult{
			{
				Name:   "test-circular-dependency",
				Prefix: "test-cd",
				AddonConfig: []cloudinfo.AddonConfig{
					{OfferingName: "main-addon"},
				},
				Passed:     true,
				StrictMode: core.BoolPtr(false), // Non-strict mode
				StrictModeWarnings: []string{
					"Circular dependency: deploy-arch-ibm-activity-tracker → deploy-arch-ibm-cloud-logs → deploy-arch-ibm-activity-tracker",
				},
			},
			{
				Name:   "test-force-enabled",
				Prefix: "test-fe",
				AddonConfig: []cloudinfo.AddonConfig{
					{OfferingName: "main-addon"},
				},
				Passed:     true,
				StrictMode: core.BoolPtr(false), // Non-strict mode
				StrictModeWarnings: []string{
					"Required dependency deploy-arch-ibm-kms was force-enabled despite being disabled (required by deploy-arch-ibm-event-notifications)",
				},
			},
			{
				Name:   "test-strict-mode-enabled",
				Prefix: "test-strict",
				AddonConfig: []cloudinfo.AddonConfig{
					{OfferingName: "main-addon"},
				},
				Passed:     true,
				StrictMode: core.BoolPtr(true), // Strict mode - should not show in warnings
			},
		}

		// Generate report
		report := GeneratePermutationReport(results)

		// Verify basic counts
		assert.Equal(t, 3, report.TotalTests)
		assert.Equal(t, 3, report.PassedTests)
		assert.Equal(t, 0, report.FailedTests)

		// Test the strict mode warnings section generation
		warningsSection := report.generateStrictModeWarningsSection()

		// Should have warnings section since we have StrictMode=false tests with warnings
		assert.NotEmpty(t, warningsSection, "Should generate strict mode warnings section")

		// Check for circular dependency warning
		assert.Contains(t, warningsSection, "Circular Dependencies Detected (1 test)", "Should mention circular dependency")
		assert.Contains(t, warningsSection, "test-circular-dependency", "Should include test name")
		assert.Contains(t, warningsSection, "deploy-arch-ibm-activity-tracker → deploy-arch-ibm-cloud-logs", "Should show circular dependency chain")

		// Check for force-enabled dependency warning
		assert.Contains(t, warningsSection, "Required Dependencies Force-Enabled (1 test)", "Should mention force-enabled dependency")
		assert.Contains(t, warningsSection, "test-force-enabled", "Should include test name")
		assert.Contains(t, warningsSection, "deploy-arch-ibm-kms was force-enabled despite being disabled", "Should show force-enabled warning")

		// Should not include the strict mode enabled test
		assert.NotContains(t, warningsSection, "test-strict-mode-enabled", "Should not include strict mode tests")

		t.Logf("Generated warnings section:\n%s", warningsSection)
	})

	t.Run("NoStrictModeWarnings", func(t *testing.T) {
		// Test with no strict mode warnings
		results := []PermutationTestResult{
			{
				Name:   "test-normal",
				Prefix: "test-n",
				AddonConfig: []cloudinfo.AddonConfig{
					{OfferingName: "main-addon"},
				},
				Passed:     true,
				StrictMode: core.BoolPtr(true), // Strict mode
			},
		}

		report := GeneratePermutationReport(results)
		warningsSection := report.generateStrictModeWarningsSection()

		// Should be empty since no strict mode warnings
		assert.Empty(t, warningsSection, "Should not generate warnings section when no warnings exist")
	})
}
