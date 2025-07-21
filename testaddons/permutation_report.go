package testaddons

import (
	"fmt"
	"strings"
	"time"

	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/cloudinfo"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/common"
)

// GeneratePermutationReport creates a comprehensive final report for all permutation tests
func GeneratePermutationReport(results []PermutationTestResult) *PermutationTestReport {
	report := &PermutationTestReport{
		TotalTests:  len(results),
		PassedTests: 0,
		FailedTests: 0,
		Results:     results,
		StartTime:   time.Now(), // This should be set by caller
		EndTime:     time.Now(),
	}

	for _, result := range results {
		if result.Passed {
			report.PassedTests++
		} else {
			report.FailedTests++
		}
	}

	return report
}

// PrintPermutationReport outputs the final permutation test report in a readable format
func (report *PermutationTestReport) PrintPermutationReport(logger *common.SmartLogger) {
	if logger == nil {
		return
	}

	// Header with summary
	logger.ShortInfo("================================================================================")
	logger.ShortInfo("🧪 PERMUTATION TEST REPORT - Complete")
	logger.ShortInfo("================================================================================")

	successRate := float64(report.PassedTests) / float64(report.TotalTests) * 100.0
	summaryLine := fmt.Sprintf("📊 Summary: %d total tests | ✅ %d passed (%.1f%%) | ❌ %d failed (%.1f%%)",
		report.TotalTests, report.PassedTests, successRate, report.FailedTests, 100.0-successRate)
	logger.ShortInfo(summaryLine)
	logger.ShortInfo("")

	// Passing tests section (collapsed)
	if report.PassedTests > 0 {
		logger.ShortInfo(fmt.Sprintf("🎯 PASSING TESTS (%d) - Collapsed for brevity", report.PassedTests))
		passedCount := 0
		for _, result := range report.Results {
			if result.Passed && passedCount < 3 {
				logger.ShortInfo(fmt.Sprintf("├─ ✅ %s", result.Name))
				passedCount++
			}
		}
		if report.PassedTests > 3 {
			logger.ShortInfo(fmt.Sprintf("└─ ... %d more passing tests (expand with --verbose)", report.PassedTests-3))
		}
		logger.ShortInfo("")
	}

	// Failed tests section (detailed)
	if report.FailedTests > 0 {
		logger.ShortInfo(fmt.Sprintf("❌ FAILED TESTS (%d) - Complete Error Details", report.FailedTests))
		failureIndex := 1
		for _, result := range report.Results {
			if !result.Passed {
				report.printFailedTest(logger, result, failureIndex, report.FailedTests)
				failureIndex++
			}
		}
		logger.ShortInfo("")
	}

	// Failure patterns analysis
	if report.FailedTests > 0 {
		report.printFailurePatterns(logger)
		logger.ShortInfo("")
	}

	logger.ShortInfo("📁 Full test logs available if additional context needed")
	logger.ShortInfo("================================================================================")
}

// printFailedTest outputs detailed information for a single failed test
func (report *PermutationTestReport) printFailedTest(logger *common.SmartLogger, result PermutationTestResult, index int, total int) {
	// Test header box
	logger.ShortInfo("┌─────────────────────────────────────────────────────────────────────────────┐")
	logger.ShortInfo(fmt.Sprintf("│ %d/%d ❌ %-69s │", index, total, result.Name))
	logger.ShortInfo(fmt.Sprintf("│     📁 Prefix: %-59s │", result.Prefix))

	// Format addon configuration
	addonSummary := report.formatAddonConfiguration(result.AddonConfig)
	lines := report.wrapText(addonSummary, 63)
	for i, line := range lines {
		if i == 0 {
			logger.ShortInfo(fmt.Sprintf("│     🔧 Addons: %-59s │", line))
		} else {
			logger.ShortInfo(fmt.Sprintf("│            %-63s │", line))
		}
	}

	logger.ShortInfo("│                                                                             │")

	// Validation errors
	if result.ValidationResult != nil && (!result.ValidationResult.IsValid || len(result.ValidationResult.DependencyErrors) > 0) {
		logger.ShortInfo("│     🔴 VALIDATION ERRORS:                                                   │")
		for _, depError := range result.ValidationResult.DependencyErrors {
			errorMsg := fmt.Sprintf("• %s addon requires '%s' dependency but it's disabled", depError.Addon.Name, depError.DependencyRequired.Name)
			lines := report.wrapText(errorMsg, 67)
			for _, line := range lines {
				logger.ShortInfo(fmt.Sprintf("│     %-71s │", line))
			}
		}
		for _, msg := range result.ValidationResult.Messages {
			lines := report.wrapText("• "+msg, 67)
			for _, line := range lines {
				logger.ShortInfo(fmt.Sprintf("│     %-71s │", line))
			}
		}
		logger.ShortInfo("│                                                                             │")
	}

	// Deployment errors
	if len(result.DeploymentErrors) > 0 || len(result.UndeploymentErrors) > 0 {
		logger.ShortInfo("│     🔴 DEPLOYMENT ERRORS:                                                   │")
		allDeployErrors := append(result.DeploymentErrors, result.UndeploymentErrors...)
		for _, err := range allDeployErrors {
			lines := report.wrapText("• "+err.Error(), 67)
			for _, line := range lines {
				logger.ShortInfo(fmt.Sprintf("│     %-71s │", line))
			}
		}
		logger.ShortInfo("│                                                                             │")
	}

	// Configuration errors
	if len(result.ConfigurationErrors) > 0 {
		logger.ShortInfo("│     🔴 CONFIGURATION ERRORS:                                                │")
		for _, err := range result.ConfigurationErrors {
			lines := report.wrapText("• "+err, 67)
			for _, line := range lines {
				logger.ShortInfo(fmt.Sprintf("│     %-71s │", line))
			}
		}
		logger.ShortInfo("│                                                                             │")
	}

	// Runtime errors (panics, etc.)
	if len(result.RuntimeErrors) > 0 {
		logger.ShortInfo("│     🔴 RUNTIME ERRORS:                                                      │")
		for _, err := range result.RuntimeErrors {
			lines := report.wrapText("• "+err, 67)
			for _, line := range lines {
				logger.ShortInfo(fmt.Sprintf("│     %-71s │", line))
			}
		}
		logger.ShortInfo("│                                                                             │")
	}

	// Missing inputs
	if len(result.MissingInputs) > 0 {
		logger.ShortInfo("│     🔴 MISSING INPUTS:                                                      │")
		inputsMsg := "• Required inputs missing: ['" + strings.Join(result.MissingInputs, "', '") + "']"
		lines := report.wrapText(inputsMsg, 67)
		for _, line := range lines {
			logger.ShortInfo(fmt.Sprintf("│     %-71s │", line))
		}
		logger.ShortInfo("│                                                                             │")
	}

	logger.ShortInfo("└─────────────────────────────────────────────────────────────────────────────┘")
	logger.ShortInfo("")
}

// formatAddonConfiguration creates a human-readable summary of addon configuration
func (report *PermutationTestReport) formatAddonConfiguration(configs []cloudinfo.AddonConfig) string {
	if len(configs) == 0 {
		return "No addons configured"
	}

	enabled := []string{}
	disabled := []string{}

	for _, config := range configs {
		if config.Enabled != nil && *config.Enabled {
			enabled = append(enabled, config.OfferingName)
		} else {
			disabled = append(disabled, config.OfferingName)
		}
	}

	var parts []string
	if len(enabled) > 0 {
		parts = append(parts, fmt.Sprintf("%s=enabled", strings.Join(enabled, ", ")))
	}
	if len(disabled) > 0 {
		if len(disabled) > 5 {
			parts = append(parts, fmt.Sprintf("[%d others disabled]", len(disabled)))
		} else {
			parts = append(parts, fmt.Sprintf("%s=disabled", strings.Join(disabled, ", ")))
		}
	}

	return strings.Join(parts, ", ")
}

// wrapText wraps text to fit within specified width
func (report *PermutationTestReport) wrapText(text string, width int) []string {
	if len(text) <= width {
		return []string{text}
	}

	var lines []string
	words := strings.Fields(text)
	currentLine := ""

	for _, word := range words {
		if len(currentLine)+len(word)+1 <= width {
			if currentLine != "" {
				currentLine += " "
			}
			currentLine += word
		} else {
			if currentLine != "" {
				lines = append(lines, currentLine)
			}
			currentLine = word
		}
	}

	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return lines
}

// printFailurePatterns analyzes and displays common failure patterns
func (report *PermutationTestReport) printFailurePatterns(logger *common.SmartLogger) {
	patterns := map[string]int{
		"Dependency Issues":      0,
		"Deployment Errors":      0,
		"Configuration Problems": 0,
		"Runtime Panics":         0,
	}

	for _, result := range report.Results {
		if !result.Passed {
			if result.ValidationResult != nil && len(result.ValidationResult.DependencyErrors) > 0 {
				patterns["Dependency Issues"]++
			}
			if len(result.DeploymentErrors) > 0 || len(result.UndeploymentErrors) > 0 {
				patterns["Deployment Errors"]++
			}
			if len(result.ConfigurationErrors) > 0 || len(result.MissingInputs) > 0 {
				patterns["Configuration Problems"]++
			}
			if len(result.RuntimeErrors) > 0 {
				patterns["Runtime Panics"]++
			}
		}
	}

	logger.ShortInfo("🔍 FAILURE PATTERNS (for quick scanning)")
	for pattern, count := range patterns {
		if count > 0 {
			logger.ShortInfo(fmt.Sprintf("├─ %s: %d tests", pattern, count))
		}
	}
}
