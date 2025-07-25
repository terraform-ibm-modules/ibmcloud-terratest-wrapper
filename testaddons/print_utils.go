package testaddons

import (
	"fmt"
	"strings"

	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/cloudinfo"
)

// printConsolidatedValidationSummary prints a clean, consolidated summary of dependency validation errors
// instead of scattered individual error messages throughout the output
func (options *TestAddonOptions) printConsolidatedValidationSummary(validationResult ValidationResult) {
	options.Logger.ShortError("═══════════════════════════════════════════════════════════════")
	options.Logger.ShortError("                  DEPENDENCY VALIDATION FAILED")
	options.Logger.ShortError("═══════════════════════════════════════════════════════════════")

	// Summary counts
	dependencyCount := len(validationResult.DependencyErrors)
	unexpectedCount := len(validationResult.UnexpectedConfigs)
	missingCount := len(validationResult.MissingConfigs)

	options.Logger.ShortError(fmt.Sprintf("Summary: %d dependency errors, %d unexpected configs, %d missing configs",
		dependencyCount, unexpectedCount, missingCount))
	options.Logger.ShortError("")

	// Dependency Errors Section
	if dependencyCount > 0 {
		options.Logger.ShortError("🔗 DEPENDENCY ERRORS:")
		for i, depErr := range validationResult.DependencyErrors {
			// Show the dependency relationship in tree format
			options.Logger.ShortError(fmt.Sprintf("  %d. %s (%s, %s)", i+1, depErr.Addon.Name, depErr.Addon.Version, depErr.Addon.Flavor.Name))
			options.Logger.ShortError(fmt.Sprintf("     └── requires: %s (%s, %s) - ❌ NOT AVAILABLE",
				depErr.DependencyRequired.Name, depErr.DependencyRequired.Version, depErr.DependencyRequired.Flavor.Name))

			if len(depErr.DependenciesAvailable) > 0 {
				options.Logger.ShortInfo("     └── Available alternatives:")
				for j, available := range depErr.DependenciesAvailable {
					symbol := "├──"
					if j == len(depErr.DependenciesAvailable)-1 {
						symbol = "└──"
					}
					options.Logger.ShortInfo(fmt.Sprintf("         %s %s (%s, %s)", symbol, available.Name, available.Version, available.Flavor.Name))
				}
			} else {
				options.Logger.ShortError("     └── ❌ No alternatives available")
			}
		}
		options.Logger.ShortError("")
	}

	// Unexpected Configs Section - show in tree format
	if unexpectedCount > 0 {
		options.Logger.ShortError("❌ UNEXPECTED CONFIGS DEPLOYED:")
		for i, unexpectedConfig := range validationResult.UnexpectedConfigs {
			options.Logger.ShortError(fmt.Sprintf("  %d. %s (%s, %s) - should not be deployed",
				i+1, unexpectedConfig.Name, unexpectedConfig.Version, unexpectedConfig.Flavor.Name))
		}
		options.Logger.ShortError("")
	}

	// Missing Configs Section - show in tree format
	if missingCount > 0 {
		options.Logger.ShortError("📋 MISSING EXPECTED CONFIGS:")
		for i, missingConfig := range validationResult.MissingConfigs {
			options.Logger.ShortError(fmt.Sprintf("  %d. %s (%s, %s) - expected but not deployed",
				i+1, missingConfig.Name, missingConfig.Version, missingConfig.Flavor.Name))
		}
		options.Logger.ShortError("")
	}

	options.Logger.ShortError("═══════════════════════════════════════════════════════════════")
	options.Logger.ShortError("Fix the above issues and retry the deployment.")
	options.Logger.ShortError("═══════════════════════════════════════════════════════════════")
}

// printDetailedValidationErrors prints detailed individual validation error messages
// This is the original verbose behavior for backward compatibility
func (options *TestAddonOptions) printDetailedValidationErrors(validationResult ValidationResult) {
	for _, depErr := range validationResult.DependencyErrors {
		errormsg := fmt.Sprintf(
			"Addon %s (version %s, flavor %s) requires %s (version %s, flavor %s), but it's not available.",
			depErr.Addon.Name, depErr.Addon.Version, depErr.Addon.Flavor.Name,
			depErr.DependencyRequired.Name, depErr.DependencyRequired.Version, depErr.DependencyRequired.Flavor.Name,
		)
		options.Logger.Error(errormsg)

		if len(depErr.DependenciesAvailable) > 0 {
			options.Logger.ShortInfo("Available alternatives:")
			for _, available := range depErr.DependenciesAvailable {
				options.Logger.ShortInfo(fmt.Sprintf("  - %s (version %s, flavor %s)", available.Name, available.Version, available.Flavor.Name))
			}
		} else {
			options.Logger.ShortError("No alternatives are available")
		}
	}

	for _, unexpectedConfig := range validationResult.UnexpectedConfigs {
		options.Logger.ShortError(fmt.Sprintf("Unexpected config deployed: %s (version %s, flavor %s)", unexpectedConfig.Name, unexpectedConfig.Version, unexpectedConfig.Flavor.Name))
	}

	for _, missingConfig := range validationResult.MissingConfigs {
		options.Logger.ShortError(fmt.Sprintf("Missing expected config: %s (version %s, flavor %s)", missingConfig.Name, missingConfig.Version, missingConfig.Flavor.Name))
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
		options.printAddonTree(*rootAddon, graph, "", true, make(map[string]bool))
	}
}

// printAddonTree recursively prints an addon and its dependencies in tree format
func (options *TestAddonOptions) printAddonTree(addon cloudinfo.OfferingReferenceDetail, graph map[string][]cloudinfo.OfferingReferenceDetail, indent string, isLast bool, visited map[string]bool) {
	// Create a unique key for this addon
	addonKey := generateAddonKeyFromDetail(addon)

	// Print the current addon
	symbol := options.getTreeSymbol(isLast)
	options.Logger.ShortInfo(fmt.Sprintf("%s%s %s (%s, %s)", indent, symbol, addon.Name, addon.Version, addon.Flavor.Name))

	// Check if we've already visited this addon to avoid infinite loops
	if visited[addonKey] {
		nextIndent := options.getIndentString(indent, isLast)
		options.Logger.ShortInfo(fmt.Sprintf("%s%s (already shown above)", nextIndent, "└── [circular reference]"))
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
		options.printAddonTree(dep, graph, nextIndent, isLastDep, visited)
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

	options.Logger.ShortError("═══════════════════════════════════════════════════════════════")
	options.Logger.ShortError("                  DEPENDENCY VALIDATION FAILED")
	options.Logger.ShortError("═══════════════════════════════════════════════════════════════")

	// Show expected tree first
	options.Logger.ShortInfo("Expected dependency tree:")
	if rootAddon != nil {
		options.printAddonTree(*rootAddon, graph, "", true, make(map[string]bool))
	}
	options.Logger.ShortInfo("")

	// Show actual deployment status tree
	options.Logger.ShortError("Actual deployment status:")
	if rootAddon != nil {
		options.printAddonTreeWithStatus(*rootAddon, graph, "", true, make(map[string]bool), deployedMap, errorMap, missingMap)
	}
	options.Logger.ShortError("")

	// Short error summary
	dependencyCount := len(validationResult.DependencyErrors)
	unexpectedCount := len(validationResult.UnexpectedConfigs)
	missingCount := len(validationResult.MissingConfigs)

	options.Logger.ShortError("Summary:")
	if dependencyCount > 0 {
		options.Logger.ShortError(fmt.Sprintf("  ❌ %d dependency version mismatches", dependencyCount))
	}
	if missingCount > 0 {
		options.Logger.ShortError(fmt.Sprintf("  📋 %d missing expected components", missingCount))
	}
	if unexpectedCount > 0 {
		options.Logger.ShortError(fmt.Sprintf("  ⚠️  %d unexpected components deployed", unexpectedCount))
	}

	options.Logger.ShortError("═══════════════════════════════════════════════════════════════")
}

// printAddonTreeWithStatus recursively prints an addon and its dependencies with validation status
func (options *TestAddonOptions) printAddonTreeWithStatus(addon cloudinfo.OfferingReferenceDetail,
	graph map[string][]cloudinfo.OfferingReferenceDetail,
	indent string, isLast bool, visited map[string]bool,
	deployedMap map[string]bool,
	errorMap map[string]cloudinfo.DependencyError,
	missingMap map[string]bool) {

	options.printAddonTreeWithStatusAndPath(addon, graph, indent, isLast, visited, deployedMap, errorMap, missingMap, []string{})
}

// printAddonTreeWithStatusAndPath recursively prints an addon and its dependencies with validation status and circular reference detection
func (options *TestAddonOptions) printAddonTreeWithStatusAndPath(addon cloudinfo.OfferingReferenceDetail,
	graph map[string][]cloudinfo.OfferingReferenceDetail,
	indent string, isLast bool, visited map[string]bool,
	deployedMap map[string]bool,
	errorMap map[string]cloudinfo.DependencyError,
	missingMap map[string]bool,
	path []string) {

	// Create a unique key for this addon
	addonKey := generateAddonKeyFromDetail(addon)

	// Determine status symbol and log method
	statusSymbol := ""
	logMethod := options.Logger.ShortInfo

	if missingMap[addonKey] {
		statusSymbol = " ❌ MISSING" // Missing completely
		logMethod = options.Logger.ShortError
	} else if deployedMap[addonKey] {
		if _, hasError := errorMap[addonKey]; hasError {
			statusSymbol = " ✅ DEPLOYED (dependency issue)" // Deployed but with dependency issues
			logMethod = options.Logger.ShortWarn
		} else {
			statusSymbol = " ✅ DEPLOYED" // Deployed correctly
			logMethod = options.Logger.ShortInfo
		}
	} else {
		statusSymbol = " ❓ UNKNOWN STATUS" // Status unclear
		logMethod = options.Logger.ShortWarn
	}

	// Print the current addon with status
	symbol := options.getTreeSymbol(isLast)
	logMethod(fmt.Sprintf("%s%s %s (%s, %s)%s", indent, symbol, addon.Name, addon.Version, addon.Flavor.Name, statusSymbol))

	// Check if we've already visited this addon to avoid infinite loops
	if visited[addonKey] {
		nextIndent := options.getIndentString(indent, isLast)
		// Show the circular reference with the path
		cycle := options.findCycleInPath(path, addonKey)
		if len(cycle) > 0 {
			options.Logger.ShortWarn(fmt.Sprintf("%s└── 🔄 CIRCULAR REFERENCE: %s", nextIndent, strings.Join(cycle, " → ")))
		} else {
			options.Logger.ShortWarn(fmt.Sprintf("%s└── 🔄 CIRCULAR REFERENCE: %s (already shown above)", nextIndent, addon.Name))
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
		options.printAddonTreeWithStatusAndPath(dep, graph, nextIndent, isLastDep, visited, deployedMap, errorMap, missingMap, newPath)
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
	validationResult ValidationResult) {

	options.printComprehensiveTreeWithStatusAndPath(rootConfig, allDeployedConfigs, graph, indent, isLast, visited, validationResult, []string{})
}

// printComprehensiveTreeWithStatusAndPath prints a comprehensive tree with circular reference detection
func (options *TestAddonOptions) printComprehensiveTreeWithStatusAndPath(rootConfig cloudinfo.OfferingReferenceDetail,
	allDeployedConfigs []cloudinfo.OfferingReferenceDetail,
	graph map[string][]cloudinfo.OfferingReferenceDetail,
	indent string, isLast bool, visited map[string]bool,
	validationResult ValidationResult,
	path []string) {

	// Create a unique key for this config
	configKey := generateAddonKeyFromDetail(rootConfig)

	// Check if we've already visited this config to avoid infinite loops
	if visited[configKey] {
		nextIndent := options.getIndentString(indent, isLast)
		// Show the circular reference with the path
		cycle := options.findCycleInPath(path, configKey)
		if len(cycle) > 0 {
			options.Logger.ShortWarn(fmt.Sprintf("%s└── 🔄 CIRCULAR REFERENCE: %s", nextIndent, strings.Join(cycle, " → ")))
		} else {
			options.Logger.ShortWarn(fmt.Sprintf("%s└── 🔄 CIRCULAR REFERENCE: %s (already shown above)", nextIndent, rootConfig.Name))
		}
		return
	}

	// Mark this config as visited and add to path
	visited[configKey] = true
	newPath := append(path, rootConfig.Name)

	// Determine status symbol and log method
	statusSymbol, logMethod := options.getConfigStatus(rootConfig, validationResult)

	// Print the current config with status
	symbol := options.getTreeSymbol(isLast)
	logMethod(fmt.Sprintf("%s%s %s (%s, %s)%s", indent, symbol, rootConfig.Name, rootConfig.Version, rootConfig.Flavor.Name, statusSymbol))

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
				options.printComprehensiveTreeWithStatusAndPath(dep, allDeployedConfigs, graph, nextIndent, isLastDep, visited, validationResult, newPath)
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
		options.printComprehensiveTreeWithStatusAndPath(dep, allDeployedConfigs, graph, nextIndent, isLastDep, visited, validationResult, newPath)
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
			return " ✅ DEPLOYED (dependency issue)", options.Logger.ShortWarn
		}
	}

	// Default - deployed correctly
	return " ✅ DEPLOYED", options.Logger.ShortInfo
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
