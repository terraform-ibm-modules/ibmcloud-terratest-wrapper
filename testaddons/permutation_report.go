package testaddons

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/cloudinfo"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/common"
)

// MessageType defines the category of validation/error messages for structured classification
type MessageType int

const (
	MessageTypeUnexpectedConfig   MessageType = iota // unexpected configs deployed when disabled
	MessageTypeMissingConfig                         // missing configs that should be deployed
	MessageTypeDependencyError                       // dependency validation errors
	MessageTypeInputValidation                       // missing or invalid input validation
	MessageTypeCircularDependency                    // circular dependency errors
	MessageTypeSuccessMessage                        // success messages that should be filtered
	MessageTypeGeneral                               // other validation messages
)

// String returns the string representation of MessageType
func (mt MessageType) String() string {
	switch mt {
	case MessageTypeUnexpectedConfig:
		return "UnexpectedConfig"
	case MessageTypeMissingConfig:
		return "MissingConfig"
	case MessageTypeDependencyError:
		return "DependencyError"
	case MessageTypeInputValidation:
		return "InputValidation"
	case MessageTypeCircularDependency:
		return "CircularDependency"
	case MessageTypeSuccessMessage:
		return "SuccessMessage"
	case MessageTypeGeneral:
		return "General"
	default:
		return "Unknown"
	}
}

// classifyMessage categorizes a validation message into predefined types
// replacing fragile strings.Contains() checks with structured classification
func classifyMessage(msg string) MessageType {
	// Success messages - should be filtered out in most contexts
	if strings.Contains(msg, "actually deployed configs are same as expected deployed configs") {
		return MessageTypeSuccessMessage
	}

	// Configuration-related errors
	if strings.Contains(msg, "unexpected configs") {
		return MessageTypeUnexpectedConfig
	}
	if strings.Contains(msg, "missing configs") {
		return MessageTypeMissingConfig
	}

	// Circular dependency errors (check before general dependency errors)
	if strings.Contains(msg, "🔍 CIRCULAR DEPENDENCY DETECTED") {
		return MessageTypeCircularDependency
	}

	// Dependency validation errors
	if strings.Contains(msg, "dependency errors") {
		return MessageTypeDependencyError
	}

	// Input validation errors
	if strings.Contains(msg, "missing required inputs") || strings.Contains(msg, "input validation") {
		return MessageTypeInputValidation
	}

	// Default to general validation message
	return MessageTypeGeneral
}

// shouldFilterMessage determines if a message should be excluded from reports
// based on its classification type
func shouldFilterMessage(msg string) bool {
	messageType := classifyMessage(msg)
	return messageType == MessageTypeSuccessMessage
}

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

	// Add aggregated summary if there are failures
	if report.FailedTests > 0 {
		aggregatedSummary := report.generateAggregatedSummary()
		if aggregatedSummary != "" {
			reportBuilder.WriteString(aggregatedSummary)
			reportBuilder.WriteString("\n")
		}
	}

	reportBuilder.WriteString("📁 Full test logs available if additional context needed\n")
	reportBuilder.WriteString("================================================================================")

	// Output the entire report as a single log entry - bypasses QuietMode
	logger.ProgressSuccess("\n" + reportBuilder.String())
}

// buildFailedTestReport builds detailed information for a single failed test as a string
func (report *PermutationTestReport) buildFailedTestReport(result PermutationTestResult, index int, total int) string {
	// No deduplication needed with simplified error categories

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

	// VALIDATION ERRORS: Configuration, dependency, and input validation issues
	if result.ValidationResult != nil && report.hasValidationErrors(result.ValidationResult) {
		builder.WriteString("│     🔴 VALIDATION ERRORS:                                                   │\n")

		// Show missing inputs (most common validation issue) - grouped by config
		groupedMissingInputs := report.groupMissingInputsByConfig(result.ValidationResult.MissingInputs)
		for configName, inputs := range groupedMissingInputs {
			// Display config header
			headerLine := fmt.Sprintf("• %s missing inputs:", configName)
			lines := report.wrapText(headerLine, 67)
			for _, line := range lines {
				builder.WriteString(fmt.Sprintf("│     %-71s │\n", line))
			}

			// Display each input as a separate bullet point
			for _, input := range inputs {
				inputLine := fmt.Sprintf("  - %s", input)
				inputLines := report.wrapText(inputLine, 67)
				for _, line := range inputLines {
					builder.WriteString(fmt.Sprintf("│     %-71s │\n", line))
				}
			}
		}

		// Show dependency errors (requires specific dependencies that are disabled)
		for _, depError := range result.ValidationResult.DependencyErrors {
			errorMsg := fmt.Sprintf("• %s addon requires '%s' dependency but it's disabled", depError.Addon.Name, depError.DependencyRequired.Name)
			lines := report.wrapText(errorMsg, 67)
			for _, line := range lines {
				builder.WriteString(fmt.Sprintf("│     %-71s │\n", line))
			}
		}

		// Show unexpected configs
		for _, unexpected := range result.ValidationResult.UnexpectedConfigs {
			errorMsg := fmt.Sprintf("• Unexpected: %s (%s, %s)", unexpected.Name, unexpected.Version, unexpected.Flavor.Name)
			lines := report.wrapText(errorMsg, 67)
			for _, line := range lines {
				builder.WriteString(fmt.Sprintf("│     %-71s │\n", line))
			}
		}

		// Show missing configs
		for _, missing := range result.ValidationResult.MissingConfigs {
			errorMsg := fmt.Sprintf("• Missing: %s (%s, %s)", missing.Name, missing.Version, missing.Flavor.Name)
			lines := report.wrapText(errorMsg, 67)
			for _, line := range lines {
				builder.WriteString(fmt.Sprintf("│     %-71s │\n", line))
			}
		}

		// Show configuration errors
		for _, configError := range result.ValidationResult.ConfigurationErrors {
			cleanedError := parseConfigurationError(configError)
			lines := report.wrapText("• "+cleanedError, 67)
			for _, line := range lines {
				builder.WriteString(fmt.Sprintf("│     %-71s │\n", line))
			}
		}

		// Show other validation messages (but skip success messages)
		for _, msg := range result.ValidationResult.Messages {
			if !shouldFilterMessage(msg) {
				lines := report.wrapText("• "+msg, 67)
				for _, line := range lines {
					builder.WriteString(fmt.Sprintf("│     %-71s │\n", line))
				}
			}
		}

		builder.WriteString("│                                                                             │\n")
	}

	// TRANSIENT ERRORS: API failures, timeouts, infrastructure issues (may resolve on retry)
	if len(result.TransientErrors) > 0 {
		builder.WriteString("│     🔴 TRANSIENT ERRORS:                                                    │\n")
		for _, err := range result.TransientErrors {
			cleanedError := parseJSONErrorMessage(err)
			lines := report.wrapText("• "+cleanedError, 67)
			for _, line := range lines {
				builder.WriteString(fmt.Sprintf("│     %-71s │\n", line))
			}
		}
		builder.WriteString("│                                                                             │\n")
	}

	// RUNTIME ERRORS: Go panics, nil pointers, code bugs (require code fixes)
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

	builder.WriteString("└─────────────────────────────────────────────────────────────────────────────┘\n")
	builder.WriteString("\n")

	return builder.String()
}

// hasValidationErrors checks if ValidationResult has any actual validation errors to display
func (report *PermutationTestReport) hasValidationErrors(validationResult *ValidationResult) bool {
	if validationResult == nil {
		return false
	}

	return !validationResult.IsValid ||
		len(validationResult.DependencyErrors) > 0 ||
		len(validationResult.UnexpectedConfigs) > 0 ||
		len(validationResult.MissingConfigs) > 0 ||
		len(validationResult.MissingInputs) > 0 ||
		len(validationResult.ConfigurationErrors) > 0 ||
		(len(validationResult.Messages) > 0 && !report.isOnlySuccessMessages(validationResult.Messages))
}

// isOnlySuccessMessages checks if the messages array contains only success messages
func (report *PermutationTestReport) isOnlySuccessMessages(messages []string) bool {
	for _, msg := range messages {
		if !shouldFilterMessage(msg) {
			return false
		}
	}
	return len(messages) > 0 // Only return true if there are messages and they're all success messages
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

// groupMissingInputsByConfig groups missing inputs by config name to avoid duplicate entries
func (report *PermutationTestReport) groupMissingInputsByConfig(missingInputs []string) map[string][]string {
	configInputs := make(map[string][]string)

	for _, missingInput := range missingInputs {
		// Extract config name and input from various formats
		configName, inputName := report.extractConfigAndInput(missingInput)
		if configName != "" && inputName != "" {
			// Add input to the config's list (avoid duplicates)
			inputs := configInputs[configName]
			found := false
			for _, existing := range inputs {
				if existing == inputName {
					found = true
					break
				}
			}
			if !found {
				configInputs[configName] = append(configInputs[configName], inputName)
			}
		}
	}

	return configInputs
}

// extractConfigAndInput extracts config name and input name from various missing input formats
func (report *PermutationTestReport) extractConfigAndInput(errorMsg string) (configName, inputName string) {
	// Handle format: "config-name (missing: input1, input2, ...)"
	if strings.Contains(errorMsg, " (missing: ") && strings.HasSuffix(errorMsg, ")") {
		parts := strings.Split(errorMsg, " (missing: ")
		if len(parts) == 2 {
			configName = strings.TrimSpace(parts[0])
			inputsPart := strings.TrimSuffix(parts[1], ")")

			// Get first input (we'll process multiple inputs from same config separately)
			inputs := strings.Split(inputsPart, ",")
			if len(inputs) > 0 {
				inputName = strings.TrimSpace(inputs[0])
				return configName, inputName
			}
		}
	}

	// Handle format: "missing required inputs: config-name (missing: input1)"
	if strings.Contains(errorMsg, "missing required inputs: ") && strings.Contains(errorMsg, " (missing: ") {
		parts := strings.Split(errorMsg, "missing required inputs: ")
		if len(parts) == 2 {
			remainder := strings.TrimSpace(parts[1])
			if idx := strings.Index(remainder, " (missing: "); idx != -1 {
				configName = strings.TrimSpace(remainder[:idx])
				inputsPart := remainder[idx+len(" (missing: "):]
				inputsPart = strings.TrimSuffix(inputsPart, ")")

				inputs := strings.Split(inputsPart, ",")
				if len(inputs) > 0 {
					inputName = strings.TrimSpace(inputs[0])
					return configName, inputName
				}
			}
		}
	}

	return "", ""
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
			result.WriteString("\n\n")
		}
		result.WriteString(fmt.Sprintf("%s missing required inputs:", configName))
		for _, input := range configInputs[configName] {
			result.WriteString(fmt.Sprintf("\n  - %s", input))
		}
	}

	return result.String()
}

// generateErrorDistribution creates a summary of error types across all failed tests
func (report *PermutationTestReport) generateErrorDistribution() string {
	var validationCount, transientCount, runtimeCount int

	// Count error types across all failed test results using simplified categories
	for _, result := range report.Results {
		if !result.Passed {
			if report.hasValidationErrors(result.ValidationResult) {
				validationCount++
			}
			if len(result.TransientErrors) > 0 {
				transientCount++
			}
			if len(result.RuntimeErrors) > 0 {
				runtimeCount++
			}
		}
	}

	// Build error distribution summary if there are any errors
	var errorTypes []string
	if validationCount > 0 {
		errorTypes = append(errorTypes, fmt.Sprintf("%d Validation errors", validationCount))
	}
	if transientCount > 0 {
		errorTypes = append(errorTypes, fmt.Sprintf("%d Transient errors", transientCount))
	}
	if runtimeCount > 0 {
		errorTypes = append(errorTypes, fmt.Sprintf("%d Runtime errors", runtimeCount))
	}

	if len(errorTypes) == 0 {
		return ""
	}

	return "🔍 Error Distribution: " + strings.Join(errorTypes, ", ")
}

// generateAggregatedSummary creates a comprehensive aggregated summary of error patterns
func (report *PermutationTestReport) generateAggregatedSummary() string {
	if report.FailedTests == 0 {
		return ""
	}

	var summary strings.Builder
	summary.WriteString("📊 AGGREGATED ERROR ANALYSIS\n")
	summary.WriteString("================================================================================\n\n")

	// Analyze configuration errors
	configPatterns := report.analyzeConfigurationErrors()
	if len(configPatterns) > 0 {
		summary.WriteString("🔧 CONFIGURATION ERRORS (Root Cause Analysis):\n\n")
		for _, pattern := range configPatterns {
			summary.WriteString(fmt.Sprintf("• %s missing required inputs: %s\n", pattern.ConfigPattern, pattern.InputName))
			if pattern.SuspectedRootCause != "" {
				summary.WriteString(fmt.Sprintf("  Seen %d times → ROOT CAUSE: %s\n", pattern.Count, pattern.SuspectedRootCause))

				// Check if this is a root offering error by examining if the config pattern appears in any disabled dependencies
				isRootOffering := report.isRootOfferingError(pattern.ConfigPattern)

				if isRootOffering {
					summary.WriteString(fmt.Sprintf("  💡 SOLUTION: Add input mapping within the root offering for when %s is disabled, or provide %s input to the test case configuration\n\n", extractDependencyName(pattern.SuspectedRootCause), pattern.InputName))
				} else {
					summary.WriteString(fmt.Sprintf("  💡 SOLUTION: Add input mapping for when %s is disabled\n\n", extractDependencyName(pattern.SuspectedRootCause)))
				}
			} else {
				summary.WriteString(fmt.Sprintf("  Seen %d times (requires further analysis)\n\n", pattern.Count))
			}
		}
	}

	// Analyze validation errors
	validationPatterns := report.analyzeValidationErrors()
	if len(validationPatterns) > 0 {
		summary.WriteString("🔍 VALIDATION ERRORS:\n\n")
		for _, pattern := range validationPatterns {
			// Use consistent terminology: "Seen in X tests" for systematic patterns
			testCount := pattern.Count
			testWord := "test"
			if testCount > 1 {
				testWord = "tests"
			}

			summary.WriteString(fmt.Sprintf("• %s: %s\n", pattern.ErrorType, pattern.Pattern))
			summary.WriteString(fmt.Sprintf("  Seen in %d %s → %s\n", testCount, testWord, report.getValidationInsight(pattern.ErrorType)))
			if solution := report.getValidationSolution(pattern.ErrorType); solution != "" {
				summary.WriteString(fmt.Sprintf("  💡 SOLUTION: %s\n", solution))
			}
			summary.WriteString("\n")
		}
	}

	// Analyze transient errors
	transientDetails := report.analyzeTransientErrors()
	if transientDetails.RuntimeCount > 0 || transientDetails.DeploymentCount > 0 {
		summary.WriteString("⚡ TRANSIENT ERRORS (Less Critical):\n")

		if transientDetails.RuntimeCount > 0 {
			// Use proper grammar: "1 occurrence" vs "X occurrences"
			occurrenceWord := "occurrence"
			if transientDetails.RuntimeCount > 1 {
				occurrenceWord = "occurrences"
			}
			summary.WriteString(fmt.Sprintf("• Runtime errors: %d %s\n", transientDetails.RuntimeCount, occurrenceWord))

			// Show sample error messages for debugging
			for i, sample := range transientDetails.RuntimeSamples {
				if i == 0 {
					summary.WriteString(fmt.Sprintf("  - %s\n", sample))
				} else if i == 1 && len(transientDetails.RuntimeSamples) > 1 {
					if transientDetails.RuntimeCount > 2 {
						summary.WriteString(fmt.Sprintf("  - %s (and %d more)\n", sample, transientDetails.RuntimeCount-2))
					} else {
						summary.WriteString(fmt.Sprintf("  - %s\n", sample))
					}
				}
			}
		}

		if transientDetails.DeploymentCount > 0 {
			// Use proper grammar: "1 occurrence" vs "X occurrences"
			occurrenceWord := "occurrence"
			if transientDetails.DeploymentCount > 1 {
				occurrenceWord = "occurrences"
			}
			summary.WriteString(fmt.Sprintf("• Deployment errors: %d %s\n", transientDetails.DeploymentCount, occurrenceWord))

			// Show sample error messages for debugging
			for i, sample := range transientDetails.DeploymentSamples {
				if i == 0 {
					summary.WriteString(fmt.Sprintf("  - %s\n", sample))
				} else if i == 1 && len(transientDetails.DeploymentSamples) > 1 {
					if transientDetails.DeploymentCount > 2 {
						summary.WriteString(fmt.Sprintf("  - %s (and %d more)\n", sample, transientDetails.DeploymentCount-2))
					} else {
						summary.WriteString(fmt.Sprintf("  - %s\n", sample))
					}
				}
			}
		}

		summary.WriteString("\n")
	}

	// Generate action items
	actionItems := report.generateActionItems(configPatterns, validationPatterns)
	if len(actionItems) > 0 {
		summary.WriteString("📋 ACTION ITEMS:\n")
		for i, item := range actionItems {
			summary.WriteString(fmt.Sprintf("%d. %s\n", i+1, item))
		}
	}

	// Add error accounting notice if analysis might be incomplete
	errorAccountingNotice := report.checkErrorAccounting(configPatterns, validationPatterns, transientDetails)
	if errorAccountingNotice != "" {
		summary.WriteString(errorAccountingNotice)
	}

	return summary.String()
}

// checkErrorAccounting verifies if all failed tests are accounted for in the aggregated analysis
func (report *PermutationTestReport) checkErrorAccounting(configPatterns []ConfigurationErrorPattern, validationPatterns []ValidationErrorPattern, transientDetails TransientErrorDetails) string {
	// Count total errors across all patterns
	var accountedFailures int

	// Configuration errors
	for _, pattern := range configPatterns {
		accountedFailures += pattern.Count
	}

	// Validation errors
	for _, pattern := range validationPatterns {
		accountedFailures += pattern.Count
	}

	// Transient errors (simplified - each test can have multiple error types)
	// Count unique tests that had transient errors but weren't already counted
	transientFailures := make(map[string]bool)
	for _, result := range report.Results {
		if !result.Passed {
			if len(result.TransientErrors) > 0 {
				// Only count this if it's not already counted in validation errors
				hasValidationError := report.hasValidationErrors(result.ValidationResult)

				if !hasValidationError {
					transientFailures[result.Name] = true
				}
			}
		}
	}
	accountedFailures += len(transientFailures)

	// Check if we're missing any failures
	unaccountedFailures := report.FailedTests - accountedFailures

	if unaccountedFailures > 0 {
		var notice strings.Builder
		notice.WriteString("\n")
		notice.WriteString("⚠️  ERROR ANALYSIS INCOMPLETE\n")
		notice.WriteString("=====================================\n")

		// Be specific about what might be missing
		if unaccountedFailures == 1 {
			notice.WriteString("1 failure may not be fully represented in this summary.\n")
		} else {
			notice.WriteString(fmt.Sprintf("%d failures may not be fully represented in this summary.\n", unaccountedFailures))
		}

		notice.WriteString("This can happen when:\n")
		notice.WriteString("• Error messages don't match expected patterns\n")
		notice.WriteString("• New types of validation failures occur\n")
		notice.WriteString("• Complex error combinations aren't fully parsed\n\n")
		notice.WriteString("📋 RECOMMENDATION: Review the complete test logs above for additional validation details.\n")

		return notice.String()
	}

	return ""
}

// analyzeConfigurationErrors groups configuration errors by pattern and identifies root causes
func (report *PermutationTestReport) analyzeConfigurationErrors() []ConfigurationErrorPattern {
	// Group errors by missing input patterns
	inputPatterns := make(map[string][]AggregatedTestInfo)

	for _, result := range report.Results {
		if !result.Passed && report.hasValidationErrors(result.ValidationResult) {
			testInfo := AggregatedTestInfo{
				Name:         result.Name,
				Prefix:       result.Prefix,
				EnabledDeps:  extractEnabledDependencies(result.AddonConfig),
				DisabledDeps: extractDisabledDependencies(result.AddonConfig),
			}

			// Create a mapping from config names to offering names for this test result
			configToOffering := make(map[string]string)
			for _, addon := range result.AddonConfig {
				configToOffering[addon.ConfigName] = addon.OfferingName
			}

			// Check missing inputs and configuration errors in ValidationResult
			if result.ValidationResult != nil {
				// Check missing inputs - handle both formats
				for _, missingInput := range result.ValidationResult.MissingInputs {
					// Handle full error format: "missing required inputs: config (missing: input)"
					if strings.Contains(missingInput, "missing required inputs:") {
						cleanedError := parseConfigurationError(missingInput)
						inputName, offeringName := ExtractInputPatternWithOffering(cleanedError, configToOffering)
						if inputName != "" && offeringName != "" {
							key := fmt.Sprintf("%s|%s", offeringName, inputName)
							inputPatterns[key] = append(inputPatterns[key], testInfo)
						}
					} else {
						// Handle individual format: "config (missing: input)" or any other format
						inputName, offeringName := ExtractInputPatternWithOffering(missingInput, configToOffering)
						if inputName != "" && offeringName != "" {
							key := fmt.Sprintf("%s|%s", offeringName, inputName)
							inputPatterns[key] = append(inputPatterns[key], testInfo)
						}
					}
				}

				// Check configuration errors
				for _, configError := range result.ValidationResult.ConfigurationErrors {
					if strings.Contains(configError, "missing required inputs:") {
						cleanedError := parseConfigurationError(configError)
						inputName, offeringName := ExtractInputPatternWithOffering(cleanedError, configToOffering)
						if inputName != "" && offeringName != "" {
							key := fmt.Sprintf("%s|%s", offeringName, inputName)
							inputPatterns[key] = append(inputPatterns[key], testInfo)
						}
					}
				}
			}
		}
	}

	// Convert to pattern list and analyze root causes
	var patterns []ConfigurationErrorPattern
	for key, tests := range inputPatterns {
		parts := strings.Split(key, "|")
		configPattern := parts[0]
		inputName := parts[1]

		pattern := ConfigurationErrorPattern{
			ConfigPattern: configPattern,
			InputName:     inputName,
			Count:         len(tests),
		}

		// Identify root cause
		commonDisabled := findCommonDisabledDependencies(tests)
		if len(commonDisabled) == 1 {
			pattern.SuspectedRootCause = commonDisabled[0] + " (disabled in all cases)"
			pattern.ConfidenceLevel = "HIGH"
		} else if len(commonDisabled) > 1 {
			// Try to identify the most likely cause based on input name correlation
			mostLikely := findMostLikelyRootCause(commonDisabled, inputName)
			if mostLikely != "" {
				pattern.SuspectedRootCause = mostLikely + " (disabled in all cases)"
				pattern.ConfidenceLevel = "HIGH"
			}
		}

		patterns = append(patterns, pattern)
	}

	// Sort by count (most common first)
	for i := 0; i < len(patterns); i++ {
		for j := i + 1; j < len(patterns); j++ {
			if patterns[i].Count < patterns[j].Count {
				patterns[i], patterns[j] = patterns[j], patterns[i]
			}
		}
	}

	return patterns
}

// isRootOfferingError determines if a configuration error is for the root offering
// Root offerings are those that don't appear in any disabled dependencies list
func (report *PermutationTestReport) isRootOfferingError(configPattern string) bool {
	// Check all test results to see if this offering name ever appears as a disabled dependency
	for _, result := range report.Results {
		disabledDeps := extractDisabledDependencies(result.AddonConfig)
		for _, disabled := range disabledDeps {
			if disabled == configPattern {
				return false // Found in disabled deps, so it's not a root offering
			}
		}
	}
	return true // Never found in disabled deps, so it's likely the root offering
}

// analyzeValidationErrors groups validation errors by type
func (report *PermutationTestReport) analyzeValidationErrors() []ValidationErrorPattern {
	patternCounts := make(map[string]int)

	for _, result := range report.Results {
		if !result.Passed && result.ValidationResult != nil && !result.ValidationResult.IsValid {
			// Count dependency errors
			for _, depError := range result.ValidationResult.DependencyErrors {
				pattern := fmt.Sprintf("%s addon requires '%s' dependency but it's disabled", depError.Addon.Name, depError.DependencyRequired.Name)
				patternCounts["Missing dependency|"+pattern]++
			}

			// Count unexpected configs with clear description
			for _, unexpected := range result.ValidationResult.UnexpectedConfigs {
				pattern := fmt.Sprintf("%s (%s, %s) deployed when disabled", unexpected.Name, unexpected.Version, unexpected.Flavor.Name)
				patternCounts["Unexpected config|"+pattern]++
			}

			// Count missing configs with clear description
			for _, missing := range result.ValidationResult.MissingConfigs {
				pattern := fmt.Sprintf("%s (%s, %s) expected but not deployed", missing.Name, missing.Version, missing.Flavor.Name)
				patternCounts["Missing config|"+pattern]++
			}

			// Process generic validation messages (fallback for validation errors not in specific arrays)
			for _, msg := range result.ValidationResult.Messages {
				messageType := classifyMessage(msg)

				switch messageType {
				case MessageTypeCircularDependency:
					// Extract the circular dependency chain from the message
					// Format: "🔍 CIRCULAR DEPENDENCY DETECTED: configA → configB → configA"
					if strings.Contains(msg, "🔍 CIRCULAR DEPENDENCY DETECTED:") {
						// Extract just the chain part after the prefix
						chainPart := strings.TrimPrefix(msg, "🔍 CIRCULAR DEPENDENCY DETECTED: ")
						patternCounts["Circular dependency|"+chainPart]++
					}
				case MessageTypeGeneral:
					// Only process general messages that aren't covered by specific arrays
					// and aren't success messages that should be filtered
					patternCounts["Generic validation|"+msg]++
				}
			}
		}
	}

	var patterns []ValidationErrorPattern
	for key, count := range patternCounts {
		parts := strings.Split(key, "|")
		errorType := parts[0]
		pattern := parts[1]

		patterns = append(patterns, ValidationErrorPattern{
			ErrorType: errorType,
			Pattern:   pattern,
			Count:     count,
		})
	}

	// Sort by count (most common first)
	for i := 0; i < len(patterns); i++ {
		for j := i + 1; j < len(patterns); j++ {
			if patterns[i].Count < patterns[j].Count {
				patterns[i], patterns[j] = patterns[j], patterns[i]
			}
		}
	}

	return patterns
}

// analyzeTransientErrors analyzes transient error types and collects sample messages
func (report *PermutationTestReport) analyzeTransientErrors() TransientErrorDetails {
	details := TransientErrorDetails{}

	for _, result := range report.Results {
		if !result.Passed {
			// Collect runtime errors and samples
			if len(result.RuntimeErrors) > 0 {
				details.RuntimeCount++
				// Collect first few samples for debugging
				for _, runtimeErr := range result.RuntimeErrors {
					if len(details.RuntimeSamples) < 2 { // Limit to first 2 samples
						cleanedError := parseJSONErrorMessage(runtimeErr)
						details.RuntimeSamples = append(details.RuntimeSamples, cleanedError)
					}
				}
			}

			// Collect transient errors and samples (simplified)
			if len(result.TransientErrors) > 0 {
				details.DeploymentCount++ // Keep the same field name for backward compatibility in reporting
				// Collect transient error samples
				for _, transientErr := range result.TransientErrors {
					if len(details.DeploymentSamples) < 2 { // Limit to first 2 samples
						cleanedError := parseJSONErrorMessage(transientErr)
						details.DeploymentSamples = append(details.DeploymentSamples, cleanedError)
					}
				}
			}
		}
	}

	return details
}

// generateActionItems creates actionable recommendations based on error patterns
func (report *PermutationTestReport) generateActionItems(configPatterns []ConfigurationErrorPattern, validationPatterns []ValidationErrorPattern) []string {
	var actions []string

	// Handle configuration error patterns
	for _, pattern := range configPatterns {
		if pattern.SuspectedRootCause != "" && (pattern.ConfidenceLevel == "HIGH" || pattern.ConfidenceLevel == "MEDIUM") {
			depName := extractDependencyName(pattern.SuspectedRootCause)
			targetComponent := extractComponentFromPattern(pattern.ConfigPattern)
			action := fmt.Sprintf("Add input mapping to %s for %s disabled scenarios → Fix missing %s input (fixes %d tests)",
				targetComponent, depName, pattern.InputName, pattern.Count)
			actions = append(actions, action)
		}
	}

	// Handle validation error patterns including circular dependencies
	for _, pattern := range validationPatterns {
		if pattern.ErrorType == "Circular dependency" {
			// Extract the config names from the circular dependency chain
			configNames := extractConfigNamesFromCircularChain(pattern.Pattern)
			if len(configNames) >= 2 {
				action := fmt.Sprintf("Resolve circular dependency: %s ↔ %s → Use existing resources or restructure deployment (affects %d tests)",
					configNames[0], configNames[1], pattern.Count)
				actions = append(actions, action)
			}
		}
	}

	return actions
}

// Helper functions

// extractConfigNamesFromCircularChain extracts config names from a circular dependency chain
// Input format: "configA → configB → configC → configA" or similar
func extractConfigNamesFromCircularChain(chain string) []string {
	// Split by arrow separator and clean up names
	parts := strings.Split(chain, " → ")
	var configNames []string

	// Use a map to deduplicate config names (since circular chains repeat the first config at the end)
	seen := make(map[string]bool)

	for _, part := range parts {
		// Extract just the config name part (before any parentheses with details)
		configName := strings.TrimSpace(part)
		if parenIndex := strings.Index(configName, " ("); parenIndex != -1 {
			configName = configName[:parenIndex]
		}

		// Add to results if not already seen
		if configName != "" && !seen[configName] {
			configNames = append(configNames, configName)
			seen[configName] = true
		}
	}

	return configNames
}

// extractEnabledDependencies extracts enabled dependency names from addon config
func extractEnabledDependencies(configs []cloudinfo.AddonConfig) []string {
	var enabled []string
	for _, config := range configs {
		if config.Enabled != nil && *config.Enabled {
			enabled = append(enabled, config.OfferingName)
		}
	}
	return enabled
}

// extractDisabledDependencies extracts disabled dependency names from addon config
func extractDisabledDependencies(configs []cloudinfo.AddonConfig) []string {
	var disabled []string
	for _, config := range configs {
		if config.Enabled == nil || !*config.Enabled {
			disabled = append(disabled, config.OfferingName)
		}
	}
	return disabled
}

// ExtractInputPatternWithOffering extracts input name and offering name from error messages
// using the config-to-offering mapping instead of parsing config names
func ExtractInputPatternWithOffering(errorMsg string, configToOffering map[string]string) (inputName, offeringName string) {
	// Parse format: "config-name missing required inputs: input1, input2"
	if strings.Contains(errorMsg, " missing required inputs: ") {
		parts := strings.Split(errorMsg, " missing required inputs: ")
		if len(parts) == 2 {
			configName := strings.TrimSpace(parts[0])
			inputs := strings.Split(strings.TrimSpace(parts[1]), ", ")
			if len(inputs) > 0 {
				inputName = strings.TrimSpace(inputs[0])
				// Look up offering name from config name
				if offering, exists := configToOffering[configName]; exists {
					offeringName = offering
				} else {
					// Fallback: use config name as offering name if not found in mapping
					offeringName = configName
				}
			}
		}
		return inputName, offeringName
	}

	// Handle format: "missing required inputs: config-name (missing: input1, input2)"
	if strings.Contains(errorMsg, "missing required inputs: ") && strings.Contains(errorMsg, " (missing: ") {
		// Extract config name and inputs from full error format
		parts := strings.Split(errorMsg, "missing required inputs: ")
		if len(parts) == 2 {
			remainder := strings.TrimSpace(parts[1])
			// Split on " (missing: " to separate config name from inputs
			if idx := strings.Index(remainder, " (missing: "); idx != -1 {
				configName := strings.TrimSpace(remainder[:idx])
				inputsPart := remainder[idx+len(" (missing: "):]
				// Remove trailing ")"
				inputsPart = strings.TrimSuffix(inputsPart, ")")

				// Split inputs by comma and get first one
				inputs := strings.Split(inputsPart, ",")
				if len(inputs) > 0 {
					inputName = strings.TrimSpace(inputs[0])
					// Look up offering name from config name
					if offering, exists := configToOffering[configName]; exists {
						offeringName = offering
					} else {
						// Fallback: use config name as offering name if not found in mapping
						offeringName = configName
					}
					return inputName, offeringName
				}
			}
		}
	}

	// Handle direct config format: "config-name (missing: input1, input2)"
	if strings.Contains(errorMsg, " (missing: ") && !strings.Contains(errorMsg, "missing required inputs: ") {
		// Extract config name and inputs directly
		if idx := strings.Index(errorMsg, " (missing: "); idx != -1 {
			configName := strings.TrimSpace(errorMsg[:idx])
			inputsPart := errorMsg[idx+len(" (missing: "):]
			// Remove trailing ")"
			inputsPart = strings.TrimSuffix(inputsPart, ")")

			// Split inputs by comma and get first one
			inputs := strings.Split(inputsPart, ",")
			if len(inputs) > 0 {
				inputName = strings.TrimSpace(inputs[0])
				// Look up offering name from config name
				if offering, exists := configToOffering[configName]; exists {
					offeringName = offering
				} else {
					// Fallback: use config name as offering name if not found in mapping
					offeringName = configName
				}
				return inputName, offeringName
			}
		}
	}

	// Handle simple missing input patterns
	if strings.Contains(errorMsg, "missing: ") {
		parts := strings.Split(errorMsg, "missing: ")
		if len(parts) == 2 {
			inputs := strings.Split(strings.TrimSpace(parts[1]), ", ")
			if len(inputs) > 0 {
				inputName = strings.TrimSpace(inputs[0])
				offeringName = "Unknown" // Generic pattern when config name is unclear
			}
		}
	}

	return inputName, offeringName
}

// findCommonDisabledDependencies finds dependencies disabled in ALL tests
func findCommonDisabledDependencies(tests []AggregatedTestInfo) []string {
	if len(tests) == 0 {
		return nil
	}

	// Start with disabled deps from first test
	common := make(map[string]bool)
	for _, dep := range tests[0].DisabledDeps {
		common[dep] = true
	}

	// Keep only deps that are disabled in ALL tests
	for i := 1; i < len(tests); i++ {
		testDisabled := make(map[string]bool)
		for _, dep := range tests[i].DisabledDeps {
			testDisabled[dep] = true
		}

		// Remove any dep not disabled in this test
		for dep := range common {
			if !testDisabled[dep] {
				delete(common, dep)
			}
		}
	}

	var result []string
	for dep := range common {
		result = append(result, dep)
	}
	return result
}

// findMostLikelyRootCause analyzes multiple disabled deps to find most likely cause
func findMostLikelyRootCause(disabledDeps []string, inputName string) string {
	// Simple heuristic: look for name correlations
	inputLower := strings.ToLower(inputName)

	for _, dep := range disabledDeps {
		depLower := strings.ToLower(dep)
		// Look for substring matches
		if strings.Contains(inputLower, "cos") && strings.Contains(depLower, "cos") {
			return dep
		}
		if strings.Contains(inputLower, "logs") && strings.Contains(depLower, "logs") {
			return dep
		}
		if strings.Contains(inputLower, "kms") && strings.Contains(depLower, "kms") {
			return dep
		}
		if strings.Contains(inputLower, "monitoring") && strings.Contains(depLower, "monitoring") {
			return dep
		}
	}

	return ""
}

// extractDependencyName extracts clean dependency name from root cause string
func extractDependencyName(rootCause string) string {
	// Extract from format: "deploy-arch-ibm-cos (disabled in all cases)"
	if strings.Contains(rootCause, " (") {
		return strings.Split(rootCause, " (")[0]
	}
	return rootCause
}

// extractComponentFromPattern extracts the target component name from ConfigPattern
func extractComponentFromPattern(configPattern string) string {
	// Extract from format: "deploy-arch-ibm-activity-tracker-*" -> "deploy-arch-ibm-activity-tracker"
	if strings.HasSuffix(configPattern, "-*") {
		return strings.TrimSuffix(configPattern, "-*")
	}
	return configPattern
}

// getValidationInsight provides actionable insight about what each validation error type means
func (report *PermutationTestReport) getValidationInsight(errorType string) string {
	switch errorType {
	case "Unexpected config":
		return "ISSUE: Configuration deployed despite being disabled in test setup"
	case "Missing config":
		return "ISSUE: Required dependency not deployed when expected"
	case "Missing dependency":
		return "ISSUE: Addon requires dependency that is disabled"
	case "Circular dependency":
		return "ISSUE: Circular dependency between configurations"
	case "Generic validation":
		return "ISSUE: Dependency validation failed"
	default:
		return "ISSUE: Validation constraint violated"
	}
}

// getValidationSolution provides actionable solution for each validation error type
func (report *PermutationTestReport) getValidationSolution(errorType string) string {
	switch errorType {
	case "Unexpected config":
		return "Review dependency resolution logic for disabled components"
	case "Missing config":
		return "Check dependency requirements and deployment logic"
	case "Missing dependency":
		return "Enable required dependency or remove dependent addon"
	case "Circular dependency":
		return "Use existing resources or restructure deployment order"
	case "Generic validation":
		return "Review dependency validation rules and configuration"
	default:
		return ""
	}
}
