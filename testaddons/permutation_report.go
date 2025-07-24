package testaddons

import (
	"encoding/json"
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
	reportBuilder.WriteString(summaryLine + "\n")

	// Add error distribution summary if there are failures
	if report.FailedTests > 0 {
		errorDistribution := report.generateErrorDistribution()
		if errorDistribution != "" {
			reportBuilder.WriteString(errorDistribution + "\n")
		}
	}
	reportBuilder.WriteString("\n")

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

		// Show dependency errors (requires specific dependencies that are disabled)
		for _, depError := range result.ValidationResult.DependencyErrors {
			errorMsg := fmt.Sprintf("• %s addon requires '%s' dependency but it's disabled", depError.Addon.Name, depError.DependencyRequired.Name)
			lines := report.wrapText(errorMsg, 67)
			for _, line := range lines {
				builder.WriteString(fmt.Sprintf("│     %-71s │\n", line))
			}
		}

		// Show unexpected configs with detailed information
		for _, unexpected := range result.ValidationResult.UnexpectedConfigs {
			errorMsg := fmt.Sprintf("• Unexpected: %s (v%s, %s)", unexpected.Name, unexpected.Version, unexpected.Flavor.Name)
			lines := report.wrapText(errorMsg, 67)
			for _, line := range lines {
				builder.WriteString(fmt.Sprintf("│     %-71s │\n", line))
			}
		}

		// Show missing configs with detailed information
		for _, missing := range result.ValidationResult.MissingConfigs {
			errorMsg := fmt.Sprintf("• Missing: %s (v%s, %s)", missing.Name, missing.Version, missing.Flavor.Name)
			lines := report.wrapText(errorMsg, 67)
			for _, line := range lines {
				builder.WriteString(fmt.Sprintf("│     %-71s │\n", line))
			}
		}

		// Show any other generic validation messages (fallback)
		for _, msg := range result.ValidationResult.Messages {
			// Skip generic messages that we've already covered with detailed info
			if !strings.Contains(msg, "unexpected configs") && !strings.Contains(msg, "missing configs") {
				lines := report.wrapText("• "+msg, 67)
				for _, line := range lines {
					builder.WriteString(fmt.Sprintf("│     %-71s │\n", line))
				}
			}
		}

		builder.WriteString("│                                                                             │\n")
	}

	// Deployment errors
	if len(result.DeploymentErrors) > 0 || len(result.UndeploymentErrors) > 0 {
		builder.WriteString("│     🔴 DEPLOYMENT ERRORS:                                                   │\n")
		allDeployErrors := append(result.DeploymentErrors, result.UndeploymentErrors...)
		for _, err := range allDeployErrors {
			cleanedError := parseJSONErrorMessage(err.Error())
			lines := report.wrapText("• "+cleanedError, 67)
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
			// First try JSON parsing, then configuration-specific parsing
			cleanedError := parseJSONErrorMessage(err)
			formattedError := parseConfigurationError(cleanedError)
			lines := report.wrapText("• "+formattedError, 67)
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
			cleanedError := parseJSONErrorMessage(err)
			lines := report.wrapText("• "+cleanedError, 67)
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

// parseJSONErrorMessage attempts to parse JSON error structures and extract readable messages
// If parsing fails, returns the original message to preserve all error content
func parseJSONErrorMessage(errorMsg string) string {
	// First, check if the message contains JSON anywhere (not just at the start)
	// Look for JSON patterns that contain message or description fields
	if !strings.Contains(errorMsg, `"message":`) && !strings.Contains(errorMsg, `"description":`) {
		return errorMsg
	}

	// Try to extract JSON from the error message
	jsonStart := strings.Index(errorMsg, "{")
	if jsonStart == -1 {
		return errorMsg
	}

	// Find the matching closing brace for the JSON
	jsonPart := errorMsg[jsonStart:]
	braceCount := 0
	jsonEnd := -1
	for i, char := range jsonPart {
		if char == '{' {
			braceCount++
		} else if char == '}' {
			braceCount--
			if braceCount == 0 {
				jsonEnd = i + 1
				break
			}
		}
	}

	if jsonEnd == -1 {
		return errorMsg
	}

	jsonStr := jsonPart[:jsonEnd]

	// Try to parse the extracted JSON
	var errorData map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &errorData); err != nil {
		// If JSON parsing fails, return original message
		return errorMsg
	}

	// Extract message field first
	if message, ok := errorData["message"]; ok {
		if messageStr, ok := message.(string); ok {
			// Check if the message itself is nested JSON
			if strings.HasPrefix(strings.TrimSpace(messageStr), "{") {
				var nestedData map[string]interface{}
				if err := json.Unmarshal([]byte(messageStr), &nestedData); err == nil {
					// Extract nested message or description
					if nestedMsg, ok := nestedData["message"].(string); ok && nestedMsg != "" {
						return nestedMsg
					}
					if description, ok := nestedData["description"].(string); ok && description != "" {
						return description
					}
				}
			}
			// Return the outer message if no nested parsing worked
			return messageStr
		}
	}

	// If no message field found, try description
	if description, ok := errorData["description"]; ok {
		if descStr, ok := description.(string); ok {
			return descStr
		}
	}

	// If nothing else worked, return original message
	return errorMsg
}

// parseConfigurationError reformats configuration error messages for better readability
// Converts "missing required inputs: config-name (missing: input1); config-name (missing: input2)"
// To "config-name missing required inputs:\n  - input1\n  - input2"
func parseConfigurationError(errorMsg string) string {
	// Check if this is a missing required inputs error
	if !strings.Contains(errorMsg, "missing required inputs:") {
		return errorMsg
	}

	// Remove the "missing required inputs: " prefix
	content := strings.TrimPrefix(errorMsg, "missing required inputs: ")

	// Split by semicolon to handle multiple config errors
	configErrors := strings.Split(content, "; ")

	// Group errors by config name
	configInputs := make(map[string][]string)

	for _, configError := range configErrors {
		configError = strings.TrimSpace(configError)

		// Parse format: "config-name (missing: input1, input2, ...)"
		if strings.Contains(configError, " (missing: ") && strings.HasSuffix(configError, ")") {
			parts := strings.Split(configError, " (missing: ")
			if len(parts) == 2 {
				configName := strings.TrimSpace(parts[0])
				inputsPart := strings.TrimSuffix(parts[1], ")")

				// Split inputs by comma and clean them
				inputs := strings.Split(inputsPart, ",")
				for _, input := range inputs {
					input = strings.TrimSpace(input)
					if input != "" {
						configInputs[configName] = append(configInputs[configName], input)
					}
				}
			}
		}
	}

	// If we couldn't parse it properly, return original
	if len(configInputs) == 0 {
		return errorMsg
	}

	// Build the formatted output
	var result strings.Builder
	configNames := make([]string, 0, len(configInputs))
	for configName := range configInputs {
		configNames = append(configNames, configName)
	}

	// Sort for consistent output
	for i := 0; i < len(configNames); i++ {
		for j := i + 1; j < len(configNames); j++ {
			if configNames[i] > configNames[j] {
				configNames[i], configNames[j] = configNames[j], configNames[i]
			}
		}
	}

	for i, configName := range configNames {
		if i > 0 {
			result.WriteString("; ")
		}
		result.WriteString(fmt.Sprintf("%s missing required inputs: %s", configName, strings.Join(configInputs[configName], ", ")))
	}

	return result.String()
}

// generateErrorDistribution creates a summary of error types across all failed tests
func (report *PermutationTestReport) generateErrorDistribution() string {
	var runtimeCount, configCount, deploymentCount, validationCount int

	// Count error types across all failed test results
	for _, result := range report.Results {
		if !result.Passed {
			if len(result.RuntimeErrors) > 0 {
				runtimeCount++
			}
			if len(result.ConfigurationErrors) > 0 {
				configCount++
			}
			if len(result.DeploymentErrors) > 0 || len(result.UndeploymentErrors) > 0 {
				deploymentCount++
			}
			if result.ValidationResult != nil && !result.ValidationResult.IsValid {
				validationCount++
			}
		}
	}

	// Build error distribution summary if there are any errors
	var errorTypes []string
	if runtimeCount > 0 {
		errorTypes = append(errorTypes, fmt.Sprintf("%d Runtime errors", runtimeCount))
	}
	if configCount > 0 {
		errorTypes = append(errorTypes, fmt.Sprintf("%d Configuration errors", configCount))
	}
	if deploymentCount > 0 {
		errorTypes = append(errorTypes, fmt.Sprintf("%d Deployment errors", deploymentCount))
	}
	if validationCount > 0 {
		errorTypes = append(errorTypes, fmt.Sprintf("%d Validation errors", validationCount))
	}

	if len(errorTypes) == 0 {
		return ""
	}

	return "🔍 Error Distribution: " + strings.Join(errorTypes, ", ")
}
