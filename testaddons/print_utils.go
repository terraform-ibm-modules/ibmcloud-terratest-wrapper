package testaddons

import (
	"fmt"
	"strings"

	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/cloudinfo"
)

// printConsolidatedValidationSummary prints a clean, consolidated summary of dependency validation errors
// instead of scattered individual error messages throughout the output
func (options *TestAddonOptions) printConsolidatedValidationSummary(validationResult ValidationResult) {
	// Build the entire summary as a single string to prevent log interleaving
	var builder strings.Builder

	builder.WriteString("\n═══════════════════════════════════════════════════════════════\n")
	builder.WriteString("                  DEPENDENCY VALIDATION FAILED\n")
	builder.WriteString("═══════════════════════════════════════════════════════════════\n")

	// Summary counts
	dependencyCount := len(validationResult.DependencyErrors)
	unexpectedCount := len(validationResult.UnexpectedConfigs)
	missingCount := len(validationResult.MissingConfigs)
	warningsCount := len(validationResult.Warnings)

	if warningsCount > 0 {
		builder.WriteString(fmt.Sprintf("Summary: %d dependency errors, %d unexpected configs, %d missing configs, %d warnings\n",
			dependencyCount, unexpectedCount, missingCount, warningsCount))
	} else {
		builder.WriteString(fmt.Sprintf("Summary: %d dependency errors, %d unexpected configs, %d missing configs\n",
			dependencyCount, unexpectedCount, missingCount))
	}
	builder.WriteString("\n")

	// Dependency Errors Section
	if dependencyCount > 0 {
		builder.WriteString("🔗 DEPENDENCY ERRORS:\n")
		for i, depErr := range validationResult.DependencyErrors {
			// Show the dependency relationship in tree format
			builder.WriteString(fmt.Sprintf("  %d. %s (%s, %s)\n", i+1, depErr.Addon.Name, depErr.Addon.Version, depErr.Addon.Flavor.Name))
			builder.WriteString(fmt.Sprintf("     └── requires: %s (%s, %s) - ❌ NOT AVAILABLE\n",
				depErr.DependencyRequired.Name, depErr.DependencyRequired.Version, depErr.DependencyRequired.Flavor.Name))

			if len(depErr.DependenciesAvailable) > 0 {
				builder.WriteString("     └── Available alternatives:\n")
				for j, available := range depErr.DependenciesAvailable {
					symbol := "├──"
					if j == len(depErr.DependenciesAvailable)-1 {
						symbol = "└──"
					}
					builder.WriteString(fmt.Sprintf("         %s %s (%s, %s)\n", symbol, available.Name, available.Version, available.Flavor.Name))
				}
			} else {
				builder.WriteString("     └── ❌ No alternatives available\n")
			}
		}
		builder.WriteString("\n")
	}

	// Unexpected Configs Section - show in tree format
	if unexpectedCount > 0 {
		builder.WriteString("❌ UNEXPECTED CONFIGS ADDED TO PROJECT:\n")
		for i, unexpectedConfig := range validationResult.UnexpectedConfigs {
			builder.WriteString(fmt.Sprintf("  %d. %s (%s, %s) - should not be added to project\n",
				i+1, unexpectedConfig.Name, unexpectedConfig.Version, unexpectedConfig.Flavor.Name))
		}
		builder.WriteString("\n")
	}

	// Missing Configs Section - show in tree format
	if missingCount > 0 {
		builder.WriteString("📋 MISSING EXPECTED CONFIGS:\n")
		for i, missingConfig := range validationResult.MissingConfigs {
			builder.WriteString(fmt.Sprintf("  %d. %s (%s, %s) - expected but not deployed\n",
				i+1, missingConfig.Name, missingConfig.Version, missingConfig.Flavor.Name))
		}
		builder.WriteString("\n")
	}

	// Warnings Section
	if warningsCount > 0 {
		builder.WriteString("⚠️ WARNINGS:\n")
		for i, warning := range validationResult.Warnings {
			builder.WriteString(fmt.Sprintf("  %d. %s\n", i+1, warning))
		}
		builder.WriteString("\n")
	}

	builder.WriteString("═══════════════════════════════════════════════════════════════\n")
	if warningsCount > 0 && dependencyCount == 0 && unexpectedCount == 0 && missingCount == 0 {
		builder.WriteString("Review the warnings above but test will continue.\n")
	} else {
		builder.WriteString("Fix the above issues and retry the deployment.\n")
	}
	builder.WriteString("═══════════════════════════════════════════════════════════════")

	// Log the complete summary as a single entry
	options.Logger.ShortError(builder.String())
}

// printDetailedValidationErrors prints detailed individual validation error messages
// This is the original verbose behavior for backward compatibility
func (options *TestAddonOptions) printDetailedValidationErrors(validationResult ValidationResult) {
	var builder strings.Builder
	builder.WriteString("\n") // Add newline at start for proper alignment

	for _, depErr := range validationResult.DependencyErrors {
		errormsg := fmt.Sprintf(
			"Addon %s (version %s, flavor %s) requires %s (version %s, flavor %s), but it's not available.\n",
			depErr.Addon.Name, depErr.Addon.Version, depErr.Addon.Flavor.Name,
			depErr.DependencyRequired.Name, depErr.DependencyRequired.Version, depErr.DependencyRequired.Flavor.Name,
		)
		builder.WriteString(errormsg)

		if len(depErr.DependenciesAvailable) > 0 {
			builder.WriteString("Available alternatives:\n")
			for _, available := range depErr.DependenciesAvailable {
				builder.WriteString(fmt.Sprintf("  - %s (version %s, flavor %s)\n", available.Name, available.Version, available.Flavor.Name))
			}
		} else {
			builder.WriteString("No alternatives are available\n")
		}
		builder.WriteString("\n")
	}

	for _, unexpectedConfig := range validationResult.UnexpectedConfigs {
		builder.WriteString(fmt.Sprintf("Unexpected config deployed: %s (version %s, flavor %s)\n", unexpectedConfig.Name, unexpectedConfig.Version, unexpectedConfig.Flavor.Name))
	}

	for _, missingConfig := range validationResult.MissingConfigs {
		builder.WriteString(fmt.Sprintf("Missing expected config: %s (version %s, flavor %s)\n", missingConfig.Name, missingConfig.Version, missingConfig.Flavor.Name))
	}

	if output := strings.TrimSuffix(builder.String(), "\n"); output != "" {
		options.Logger.Error(output)
	}
}

// PrintDependencyTree prints the dependency graph in a clear tree format
func (options *TestAddonOptions) PrintDependencyTree(graph map[string][]cloudinfo.OfferingReferenceDetail, expectedDeployedList []cloudinfo.OfferingReferenceDetail) {
	if len(expectedDeployedList) == 0 {
		options.Logger.ShortInfo("  No dependencies found")
		return
	}

	// Find the root addon (the one that's not a dependency of any other)
	allDependencies := make(map[string]bool)
	for _, deps := range graph {
		for _, dep := range deps {
			key := generateAddonKeyFromDetail(dep)
			allDependencies[key] = true
		}
	}

	var rootAddon *cloudinfo.OfferingReferenceDetail
	for _, addon := range expectedDeployedList {
		key := generateAddonKeyFromDetail(addon)
		if !allDependencies[key] {
			rootAddon = &addon
			break
		}
	}

	if rootAddon == nil && len(expectedDeployedList) > 0 {
		// If no clear root found, use the first addon
		rootAddon = &expectedDeployedList[0]
	}

	if rootAddon != nil {
		// Build entire tree as string to prevent log interleaving
		var builder strings.Builder
		builder.WriteString("\n") // Add newline at start for proper alignment
		options.printAddonTree(*rootAddon, graph, "", true, make(map[string]bool), &builder)
		// Log the complete tree as a single entry
		if treeStr := builder.String(); treeStr != "" {
			options.Logger.ShortInfo(strings.TrimSuffix(treeStr, "\n"))
		}
	}
}

// printAddonTree recursively prints an addon and its dependencies in tree format
func (options *TestAddonOptions) printAddonTree(addon cloudinfo.OfferingReferenceDetail, graph map[string][]cloudinfo.OfferingReferenceDetail, indent string, isLast bool, visited map[string]bool, builder *strings.Builder) {
	// Create a unique key for this addon
	addonKey := generateAddonKeyFromDetail(addon)

	// Print the current addon
	symbol := options.getTreeSymbol(isLast)
	builder.WriteString(fmt.Sprintf("%s%s %s (%s, %s)\n", indent, symbol, addon.Name, addon.Version, addon.Flavor.Name))

	// Check if we've already visited this addon to avoid infinite loops
	if visited[addonKey] {
		nextIndent := options.getIndentString(indent, isLast)
		builder.WriteString(fmt.Sprintf("%s%s (already shown above)\n", nextIndent, "└── [circular reference]"))
		return
	}

	// Mark this addon as visited
	visited[addonKey] = true

	// Get dependencies for this addon
	dependencies, hasDeps := graph[addonKey]
	if !hasDeps || len(dependencies) == 0 {
		// Remove from visited when we're done with this branch
		delete(visited, addonKey)
		return
	}

	// Print dependencies
	nextIndent := options.getIndentString(indent, isLast)
	for i, dep := range dependencies {
		isLastDep := i == len(dependencies)-1
		options.printAddonTree(dep, graph, nextIndent, isLastDep, visited, builder)
	}

	// Remove from visited when we're done with this branch
	delete(visited, addonKey)
}

// getTreeSymbol returns the appropriate tree symbol based on position
func (options *TestAddonOptions) getTreeSymbol(isLast bool) string {
	if isLast {
		return "└──"
	}
	return "├──"
}

// getIndentString returns the appropriate indentation string
func (options *TestAddonOptions) getIndentString(currentIndent string, isLast bool) string {
	if isLast {
		return currentIndent + "    "
	}
	return currentIndent + "│   "
}

// printDependencyTreeWithValidationStatus prints the dependency tree with validation status annotations
func (options *TestAddonOptions) printDependencyTreeWithValidationStatus(graph map[string][]cloudinfo.OfferingReferenceDetail,
	expectedDeployedList []cloudinfo.OfferingReferenceDetail,
	actuallyDeployedList []cloudinfo.OfferingReferenceDetail,
	validationResult ValidationResult) {

	if len(expectedDeployedList) == 0 {
		options.Logger.ShortInfo("  No dependencies found")
		return
	}

	// Create maps for quick lookup of deployed configs and validation issues
	deployedMap := make(map[string]bool)
	for _, deployed := range actuallyDeployedList {
		key := generateAddonKeyFromDetail(deployed)
		deployedMap[key] = true
	}

	errorMap := make(map[string]cloudinfo.DependencyError)
	for _, depErr := range validationResult.DependencyErrors {
		key := generateAddonKeyFromDependencyError(depErr)
		errorMap[key] = depErr
	}

	missingMap := make(map[string]bool)
	for _, missing := range validationResult.MissingConfigs {
		key := generateAddonKeyFromDetail(missing)
		missingMap[key] = true
	}

	// Find the root addon (the one that's not a dependency of any other)
	allDependencies := make(map[string]bool)
	for _, deps := range graph {
		for _, dep := range deps {
			key := generateAddonKeyFromDetail(dep)
			allDependencies[key] = true
		}
	}

	var rootAddon *cloudinfo.OfferingReferenceDetail
	for _, addon := range expectedDeployedList {
		key := generateAddonKeyFromDetail(addon)
		if !allDependencies[key] {
			rootAddon = &addon
			break
		}
	}

	if rootAddon == nil && len(expectedDeployedList) > 0 {
		// If no clear root found, use the first addon
		rootAddon = &expectedDeployedList[0]
	}

	// Build consolidated output to prevent log interleaving
	var builder strings.Builder

	builder.WriteString("\n═══════════════════════════════════════════════════════════════\n")
	builder.WriteString("                  DEPENDENCY VALIDATION FAILED\n")
	builder.WriteString("═══════════════════════════════════════════════════════════════\n")

	// Show expected tree first
	builder.WriteString("Expected dependency tree:\n")
	if rootAddon != nil {
		options.printAddonTree(*rootAddon, graph, "", true, make(map[string]bool), &builder)
	}
	builder.WriteString("\n")

	// Show actual deployment status tree (comprehensive: expected + unexpected)
	builder.WriteString("Actual deployment status:\n")
	// Build a comprehensive tree that includes unexpected configurations to show where they fit
	allDeployedTree := options.buildComprehensiveDeploymentTree(actuallyDeployedList, graph, validationResult)
	if len(allDeployedTree) > 0 {
		// Find a suitable root among all deployed configs (one that isn't a dependency of another)
		var rootConfig *cloudinfo.OfferingReferenceDetail
		for _, cfg := range allDeployedTree {
			isRoot := true
			for _, other := range allDeployedTree {
				if deps, exists := graph[generateAddonKeyFromDetail(other)]; exists {
					for _, dep := range deps {
						if dep.Name == cfg.Name && dep.Version == cfg.Version && dep.Flavor.Name == cfg.Flavor.Name {
							isRoot = false
							break
						}
					}
				}
				if !isRoot {
					break
				}
			}
			if isRoot {
				rootConfig = &cfg
				break
			}
		}
		if rootConfig == nil {
			// Fallback: use first config if we couldn't determine a clear root
			rootConfig = &allDeployedTree[0]
		}
		options.printComprehensiveTreeWithStatus(*rootConfig, allDeployedTree, graph, "", true, make(map[string]bool), validationResult, &builder)
	} else if rootAddon != nil {
		// Fallback to expected-only tree if no comprehensive data available
		options.printAddonTreeWithStatus(*rootAddon, graph, "", true, make(map[string]bool), deployedMap, errorMap, missingMap, &builder)
	}
	builder.WriteString("\n")

	// Short error summary
	dependencyCount := len(validationResult.DependencyErrors)
	unexpectedCount := len(validationResult.UnexpectedConfigs)
	missingCount := len(validationResult.MissingConfigs)

	builder.WriteString("Summary:\n")
	if dependencyCount > 0 {
		builder.WriteString(fmt.Sprintf("  ❌ %d dependency version mismatches\n", dependencyCount))
	}
	if missingCount > 0 {
		builder.WriteString(fmt.Sprintf("  📋 %d missing expected components\n", missingCount))
	}
	if unexpectedCount > 0 {
		builder.WriteString(fmt.Sprintf("  ⚠️  %d unexpected components deployed\n", unexpectedCount))
		// List unexpected components explicitly for clarity
		for _, u := range validationResult.UnexpectedConfigs {
			builder.WriteString(fmt.Sprintf("    • Unexpected: %s (%s, %s)\n", u.Name, u.Version, u.Flavor.Name))
		}
	}

	builder.WriteString("═══════════════════════════════════════════════════════════════")

	// Log the complete output as a single entry
	options.Logger.ShortError(builder.String())
}

// printAddonTreeWithStatus recursively prints an addon and its dependencies with validation status
func (options *TestAddonOptions) printAddonTreeWithStatus(addon cloudinfo.OfferingReferenceDetail,
	graph map[string][]cloudinfo.OfferingReferenceDetail,
	indent string, isLast bool, visited map[string]bool,
	deployedMap map[string]bool,
	errorMap map[string]cloudinfo.DependencyError,
	missingMap map[string]bool,
	builder *strings.Builder) {

	options.printAddonTreeWithStatusAndPath(addon, graph, indent, isLast, visited, deployedMap, errorMap, missingMap, []string{}, builder)
}

// printAddonTreeWithStatusAndPath recursively prints an addon and its dependencies with validation status and circular reference detection
func (options *TestAddonOptions) printAddonTreeWithStatusAndPath(addon cloudinfo.OfferingReferenceDetail,
	graph map[string][]cloudinfo.OfferingReferenceDetail,
	indent string, isLast bool, visited map[string]bool,
	deployedMap map[string]bool,
	errorMap map[string]cloudinfo.DependencyError,
	missingMap map[string]bool,
	path []string,
	builder *strings.Builder) {

	// Create a unique key for this addon
	addonKey := generateAddonKeyFromDetail(addon)

	// Determine status symbol and log method
	statusSymbol := ""

	if missingMap[addonKey] {
		statusSymbol = " ❌ MISSING" // Missing completely
	} else if deployedMap[addonKey] {
		if _, hasError := errorMap[addonKey]; hasError {
			statusSymbol = " ✅ ADDED_TO_PROJECT (dependency issue)" // Added to project but with dependency issues
		} else {
			statusSymbol = " ✅ ADDED_TO_PROJECT" // Added to project correctly
		}
	} else {
		statusSymbol = " ❓ UNKNOWN STATUS" // Status unclear
	}

	// Print the current addon with status
	symbol := options.getTreeSymbol(isLast)
	builder.WriteString(fmt.Sprintf("%s%s %s (%s, %s)%s\n", indent, symbol, addon.Name, addon.Version, addon.Flavor.Name, statusSymbol))

	// Check if we've already visited this addon to avoid infinite loops
	if visited[addonKey] {
		nextIndent := options.getIndentString(indent, isLast)
		// Show the circular reference with the path
		cycle := options.findCycleInPath(path, addonKey)
		if len(cycle) > 0 {
			builder.WriteString(fmt.Sprintf("%s└── 🔄 CIRCULAR REFERENCE: %s\n", nextIndent, strings.Join(cycle, " → ")))
		} else {
			builder.WriteString(fmt.Sprintf("%s└── 🔄 CIRCULAR REFERENCE: %s (already shown above)\n", nextIndent, addon.Name))
		}
		return
	}

	// Mark this addon as visited and add to path
	visited[addonKey] = true
	newPath := append(path, addon.Name)

	// Get dependencies for this addon
	dependencies, hasDeps := graph[addonKey]
	if !hasDeps || len(dependencies) == 0 {
		// Remove from visited when we're done with this branch
		delete(visited, addonKey)
		return
	}

	// Print dependencies
	nextIndent := options.getIndentString(indent, isLast)
	for i, dep := range dependencies {
		isLastDep := i == len(dependencies)-1
		options.printAddonTreeWithStatusAndPath(dep, graph, nextIndent, isLastDep, visited, deployedMap, errorMap, missingMap, newPath, builder)
	}

	// Remove from visited when we're done with this branch
	delete(visited, addonKey)
}

// findCycleInPath finds the circular reference in the dependency path and returns the cycle
func (options *TestAddonOptions) findCycleInPath(path []string, currentAddon string) []string {
	// Extract just the addon name from the current addon key (before the first colon)
	currentName := currentAddon
	if idx := strings.Index(currentAddon, ":"); idx != -1 {
		currentName = currentAddon[:idx]
	}

	// Find where the cycle starts in the path
	for i, pathItem := range path {
		if pathItem == currentName {
			// Found the start of the cycle - return the cycle path
			cycle := make([]string, len(path[i:])+1)
			copy(cycle, path[i:])
			cycle[len(cycle)-1] = currentName // Complete the cycle
			return cycle
		}
	}

	// If not found in path, just return the current addon as a self-reference
	return []string{currentName, currentName}
}

// buildComprehensiveDeploymentTree builds a tree that includes all deployed configurations
// (both expected and unexpected) to help with debugging dependency issues
func (options *TestAddonOptions) buildComprehensiveDeploymentTree(actuallyDeployedList []cloudinfo.OfferingReferenceDetail, graph map[string][]cloudinfo.OfferingReferenceDetail, validationResult ValidationResult) []cloudinfo.OfferingReferenceDetail {
	// Start with all actually deployed configurations
	allConfigs := make([]cloudinfo.OfferingReferenceDetail, len(actuallyDeployedList))
	copy(allConfigs, actuallyDeployedList)

	// Add any missing configurations that should have been deployed
	for _, missing := range validationResult.MissingConfigs {
		allConfigs = append(allConfigs, missing)
	}

	return allConfigs
}

// printComprehensiveTreeWithStatus prints a comprehensive tree that shows all configurations
// (expected, unexpected, missing) with their proper status indicators
func (options *TestAddonOptions) printComprehensiveTreeWithStatus(rootConfig cloudinfo.OfferingReferenceDetail,
	allDeployedConfigs []cloudinfo.OfferingReferenceDetail,
	graph map[string][]cloudinfo.OfferingReferenceDetail,
	indent string, isLast bool, visited map[string]bool,
	validationResult ValidationResult,
	builder *strings.Builder) {

	options.printComprehensiveTreeWithStatusAndPath(rootConfig, allDeployedConfigs, graph, indent, isLast, visited, validationResult, []string{}, builder)
}

// printComprehensiveTreeWithStatusAndPath prints a comprehensive tree with circular reference detection
func (options *TestAddonOptions) printComprehensiveTreeWithStatusAndPath(rootConfig cloudinfo.OfferingReferenceDetail,
	allDeployedConfigs []cloudinfo.OfferingReferenceDetail,
	graph map[string][]cloudinfo.OfferingReferenceDetail,
	indent string, isLast bool, visited map[string]bool,
	validationResult ValidationResult,
	path []string,
	builder *strings.Builder) {

	// Create a unique key for this config
	configKey := generateAddonKeyFromDetail(rootConfig)

	// Check if we've already visited this config to avoid infinite loops
	if visited[configKey] {
		nextIndent := options.getIndentString(indent, isLast)
		// Show the circular reference with the path
		cycle := options.findCycleInPath(path, configKey)
		if len(cycle) > 0 {
			builder.WriteString(fmt.Sprintf("%s└── 🔄 CIRCULAR REFERENCE: %s\n", nextIndent, strings.Join(cycle, " → ")))
		} else {
			builder.WriteString(fmt.Sprintf("%s└── 🔄 CIRCULAR REFERENCE: %s (already shown above)\n", nextIndent, rootConfig.Name))
		}
		return
	}

	// Mark this config as visited and add to path
	visited[configKey] = true
	newPath := append(path, rootConfig.Name)

	// Determine status symbol
	statusSymbol, _ := options.getConfigStatus(rootConfig, validationResult)

	// Print the current config with status
	symbol := options.getTreeSymbol(isLast)
	builder.WriteString(fmt.Sprintf("%s%s %s (%s, %s)%s\n", indent, symbol, rootConfig.Name, rootConfig.Version, rootConfig.Flavor.Name, statusSymbol))

	// Get dependencies for this config
	dependencies, hasDeps := graph[configKey]

	// If no dependencies in expected graph, check if any deployed configs might be dependencies
	if !hasDeps || len(dependencies) == 0 {
		// Look for any deployed configs that might be dependencies of this one
		// This helps show unexpected dependencies in the tree
		deployedDependencies := options.findDeployedDependencies(rootConfig, allDeployedConfigs, validationResult)
		if len(deployedDependencies) > 0 {
			nextIndent := options.getIndentString(indent, isLast)
			for i, dep := range deployedDependencies {
				isLastDep := i == len(deployedDependencies)-1
				options.printComprehensiveTreeWithStatusAndPath(dep, allDeployedConfigs, graph, nextIndent, isLastDep, visited, validationResult, newPath, builder)
			}
		}
		// Remove from visited when we're done with this branch
		delete(visited, configKey)
		return
	}

	// Print expected dependencies
	nextIndent := options.getIndentString(indent, isLast)
	for i, dep := range dependencies {
		isLastDep := i == len(dependencies)-1
		options.printComprehensiveTreeWithStatusAndPath(dep, allDeployedConfigs, graph, nextIndent, isLastDep, visited, validationResult, newPath, builder)
	}

	// Remove from visited when we're done with this branch
	delete(visited, configKey)
}

// getConfigStatus determines the status symbol and log method for a configuration
func (options *TestAddonOptions) getConfigStatus(config cloudinfo.OfferingReferenceDetail, validationResult ValidationResult) (string, func(string)) {
	configKey := generateAddonKeyFromDetail(config)

	// Check if it's missing
	for _, missing := range validationResult.MissingConfigs {
		missingKey := generateAddonKeyFromDetail(missing)
		if configKey == missingKey {
			return " ❌ MISSING", options.Logger.ShortError
		}
	}

	// Check if it's unexpected
	for _, unexpected := range validationResult.UnexpectedConfigs {
		unexpectedKey := generateAddonKeyFromDetail(unexpected)
		if configKey == unexpectedKey {
			return " ❌ UNEXPECTED", options.Logger.ShortError
		}
	}

	// Check if it has dependency errors
	for _, depErr := range validationResult.DependencyErrors {
		errorKey := generateAddonKeyFromDependencyError(depErr)
		if configKey == errorKey {
			return " ✅ ADDED_TO_PROJECT (dependency issue)", options.Logger.ShortWarn
		}
	}

	// Default - added to project correctly
	return " ✅ ADDED_TO_PROJECT", options.Logger.ShortInfo
}

// findDeployedDependencies finds any deployed configurations that might be dependencies
// This helps show unexpected dependencies in the tree structure
func (options *TestAddonOptions) findDeployedDependencies(parent cloudinfo.OfferingReferenceDetail, allDeployedConfigs []cloudinfo.OfferingReferenceDetail, validationResult ValidationResult) []cloudinfo.OfferingReferenceDetail {
	var dependencies []cloudinfo.OfferingReferenceDetail

	// Only add unexpected configs as dependencies if they're not the same as the parent
	// This prevents fake circular references
	parentKey := generateAddonKeyFromDetail(parent)

	for _, unexpected := range validationResult.UnexpectedConfigs {
		unexpectedKey := generateAddonKeyFromDetail(unexpected)

		// Don't add self as dependency (prevents fake circular references)
		if unexpectedKey != parentKey {
			dependencies = append(dependencies, unexpected)
		}
	}

	return dependencies
}
