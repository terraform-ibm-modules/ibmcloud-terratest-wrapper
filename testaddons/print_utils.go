package testaddons

import (
	"fmt"

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
			key := fmt.Sprintf("%s:%s:%s", dep.Name, dep.Version, dep.Flavor.Name)
			allDependencies[key] = true
		}
	}

	var rootAddon *cloudinfo.OfferingReferenceDetail
	for _, addon := range expectedDeployedList {
		key := fmt.Sprintf("%s:%s:%s", addon.Name, addon.Version, addon.Flavor.Name)
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
	addonKey := fmt.Sprintf("%s:%s:%s", addon.Name, addon.Version, addon.Flavor.Name)

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
		key := fmt.Sprintf("%s:%s:%s", deployed.Name, deployed.Version, deployed.Flavor.Name)
		deployedMap[key] = true
	}

	errorMap := make(map[string]cloudinfo.DependencyError)
	for _, depErr := range validationResult.DependencyErrors {
		key := fmt.Sprintf("%s:%s:%s", depErr.Addon.Name, depErr.Addon.Version, depErr.Addon.Flavor.Name)
		errorMap[key] = depErr
	}

	missingMap := make(map[string]bool)
	for _, missing := range validationResult.MissingConfigs {
		key := fmt.Sprintf("%s:%s:%s", missing.Name, missing.Version, missing.Flavor.Name)
		missingMap[key] = true
	}

	// Find the root addon (the one that's not a dependency of any other)
	allDependencies := make(map[string]bool)
	for _, deps := range graph {
		for _, dep := range deps {
			key := fmt.Sprintf("%s:%s:%s", dep.Name, dep.Version, dep.Flavor.Name)
			allDependencies[key] = true
		}
	}

	var rootAddon *cloudinfo.OfferingReferenceDetail
	for _, addon := range expectedDeployedList {
		key := fmt.Sprintf("%s:%s:%s", addon.Name, addon.Version, addon.Flavor.Name)
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

	// Create a unique key for this addon
	addonKey := fmt.Sprintf("%s:%s:%s", addon.Name, addon.Version, addon.Flavor.Name)

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
		options.Logger.ShortInfo(fmt.Sprintf("%s└── [circular reference - already shown above]", nextIndent))
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
		options.printAddonTreeWithStatus(dep, graph, nextIndent, isLastDep, visited, deployedMap, errorMap, missingMap)
	}

	// Remove from visited when we're done with this branch
	delete(visited, addonKey)
}
