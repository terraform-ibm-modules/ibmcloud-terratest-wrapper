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

	// Build the entire report as a single string
	var reportBuilder strings.Builder

	// Header with summary
	reportBuilder.WriteString("================================================================================\n")
	reportBuilder.WriteString("🧪 PERMUTATION TEST REPORT - Complete\n")
	reportBuilder.WriteString("================================================================================\n")

	var summaryLine string
	if report.TotalTests == 0 {
		summaryLine = fmt.Sprintf("📊 Summary: %d total tests | ✅ %d passed | ❌ %d failed",
			report.TotalTests, report.PassedTests, report.FailedTests)
	} else {
		successRate := float64(report.PassedTests) / float64(report.TotalTests) * 100.0
		failureRate := float64(report.FailedTests) / float64(report.TotalTests) * 100.0
		summaryLine = fmt.Sprintf("📊 Summary: %d total tests | ✅ %d passed (%.1f%%) | ❌ %d failed (%.1f%%)",
			report.TotalTests, report.PassedTests, successRate, report.FailedTests, failureRate)
	}
	reportBuilder.WriteString(summaryLine + "\n\n")

	// Passing tests section (summary only)
	if report.PassedTests > 0 {
		reportBuilder.WriteString(fmt.Sprintf("✅ PASSED: %d tests completed successfully\n\n", report.PassedTests))
	}

	// Failed tests section (detailed)
	if report.FailedTests > 0 {
		reportBuilder.WriteString(fmt.Sprintf("❌ FAILED TESTS (%d) - Complete Error Details\n", report.FailedTests))
		failureIndex := 1
		for _, result := range report.Results {
			if !result.Passed {
				failedTestReport := report.buildFailedTestReport(result, failureIndex, report.FailedTests)
				reportBuilder.WriteString(failedTestReport)
				failureIndex++
			}
		}
		reportBuilder.WriteString("\n")
	}

	reportBuilder.WriteString("📁 Full test logs available if additional context needed\n")
	reportBuilder.WriteString("================================================================================")

	// Output the entire report as a single log entry - bypasses QuietMode
	logger.ProgressSuccess("\n" + reportBuilder.String())
}

// buildFailedTestReport builds detailed information for a single failed test as a string
func (report *PermutationTestReport) buildFailedTestReport(result PermutationTestResult, index int, total int) string {
	var builder strings.Builder

	// Test header box
	builder.WriteString("┌─────────────────────────────────────────────────────────────────────────────┐\n")
	builder.WriteString(fmt.Sprintf("│ %d/%d ❌ %-67s │\n", index, total, result.Name))
	builder.WriteString(fmt.Sprintf("│     📁 Prefix: %-57s │\n", result.Prefix))

	// Format addon configuration
	addonSummary := report.formatAddonConfiguration(result.AddonConfig)
	lines := report.wrapText(addonSummary, 63)
	for i, line := range lines {
		if i == 0 {
			builder.WriteString(fmt.Sprintf("│     🔧 Addons: %-57s │\n", line))
		} else {
			builder.WriteString(fmt.Sprintf("│            %-63s │\n", line))
		}
	}

	builder.WriteString("│                                                                             │\n")

	// Validation errors
	if result.ValidationResult != nil && (!result.ValidationResult.IsValid || len(result.ValidationResult.DependencyErrors) > 0) {
		builder.WriteString("│     🔴 VALIDATION ERRORS:                                                   │\n")
		for _, depError := range result.ValidationResult.DependencyErrors {
			errorMsg := fmt.Sprintf("• %s addon requires '%s' dependency but it's disabled", depError.Addon.Name, depError.DependencyRequired.Name)
			lines := report.wrapText(errorMsg, 67)
			for _, line := range lines {
				builder.WriteString(fmt.Sprintf("│     %-71s │\n", line))
			}
		}
		for _, msg := range result.ValidationResult.Messages {
			lines := report.wrapText("• "+msg, 67)
			for _, line := range lines {
				builder.WriteString(fmt.Sprintf("│     %-71s │\n", line))
			}
		}
		builder.WriteString("│                                                                             │\n")
	}

	// Deployment errors
	if len(result.DeploymentErrors) > 0 || len(result.UndeploymentErrors) > 0 {
		builder.WriteString("│     🔴 DEPLOYMENT ERRORS:                                                   │\n")
		allDeployErrors := append(result.DeploymentErrors, result.UndeploymentErrors...)
		for _, err := range allDeployErrors {
			lines := report.wrapText("• "+err.Error(), 67)
			for _, line := range lines {
				builder.WriteString(fmt.Sprintf("│     %-71s │\n", line))
			}
		}
		builder.WriteString("│                                                                             │\n")
	}

	// Configuration errors
	if len(result.ConfigurationErrors) > 0 {
		builder.WriteString("│     🔴 CONFIGURATION ERRORS:                                                │\n")
		for _, err := range result.ConfigurationErrors {
			lines := report.wrapText("• "+err, 67)
			for _, line := range lines {
				builder.WriteString(fmt.Sprintf("│     %-71s │\n", line))
			}
		}
		builder.WriteString("│                                                                             │\n")
	}

	// Runtime errors (panics, etc.)
	if len(result.RuntimeErrors) > 0 {
		builder.WriteString("│     🔴 RUNTIME ERRORS:                                                      │\n")
		for _, err := range result.RuntimeErrors {
			lines := report.wrapText("• "+err, 67)
			for _, line := range lines {
				builder.WriteString(fmt.Sprintf("│     %-71s │\n", line))
			}
		}
		builder.WriteString("│                                                                             │\n")
	}

	// Missing inputs
	if len(result.MissingInputs) > 0 {
		builder.WriteString("│     🔴 MISSING INPUTS:                                                      │\n")
		inputsMsg := "• Required inputs missing: ['" + strings.Join(result.MissingInputs, "', '") + "']"
		lines := report.wrapText(inputsMsg, 67)
		for _, line := range lines {
			builder.WriteString(fmt.Sprintf("│     %-71s │\n", line))
		}
		builder.WriteString("│                                                                             │\n")
	}

	builder.WriteString("└─────────────────────────────────────────────────────────────────────────────┘\n")
	builder.WriteString("\n")

	return builder.String()
}

// formatAddonConfiguration creates a complete human-readable summary of addon configuration for debugging
func (report *PermutationTestReport) formatAddonConfiguration(configs []cloudinfo.AddonConfig) string {
	if len(configs) == 0 {
		return "No addons configured"
	}

	// First entry is always the main addon (always enabled)
	mainAddon := configs[0]
	var summary strings.Builder

	// Show main addon
	summary.WriteString(fmt.Sprintf("Main: %s (enabled)", mainAddon.OfferingName))

	// Process dependencies if any
	if len(configs) > 1 {
		dependencies := configs[1:] // Skip the main addon
		enabled := []string{}
		disabled := []string{}

		for _, config := range dependencies {
			if config.Enabled != nil && *config.Enabled {
				enabled = append(enabled, config.OfferingName)
			} else {
				disabled = append(disabled, config.OfferingName)
			}
		}

		// Add dependency summary on new line
		summary.WriteString(fmt.Sprintf(" | Dependencies: %d enabled, %d disabled", len(enabled), len(disabled)))

		// Add enabled dependencies on new line if any
		if len(enabled) > 0 {
			summary.WriteString(fmt.Sprintf(" | ✅ Enabled: %s", strings.Join(enabled, ", ")))
		}

		// Add disabled dependencies on new line if any
		if len(disabled) > 0 {
			summary.WriteString(fmt.Sprintf(" | ❌ Disabled: %s", strings.Join(disabled, ", ")))
		}
	} else {
		summary.WriteString(" | No dependencies")
	}

	return summary.String()
}

// wrapText wraps text to fit within specified width, with special handling for " | " separators
func (report *PermutationTestReport) wrapText(text string, width int) []string {
	if len(text) <= width {
		return []string{text}
	}

	// First split on " | " to handle logical sections
	sections := strings.Split(text, " | ")
	var lines []string

	for i, section := range sections {
		// For sections after the first, add appropriate indentation
		if i > 0 {
			section = strings.TrimSpace(section)
		}

		// Wrap each section individually
		sectionLines := report.wrapSection(section, width)
		lines = append(lines, sectionLines...)
	}

	return lines
}

// wrapSection wraps a single section of text within the specified width
func (report *PermutationTestReport) wrapSection(text string, width int) []string {
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
