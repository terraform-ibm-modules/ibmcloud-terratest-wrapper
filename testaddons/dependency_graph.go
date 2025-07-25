package testaddons

import (
	"fmt"
	"strings"

	"github.com/IBM/platform-services-go-sdk/catalogmanagementv1"
	"github.com/IBM/project-go-sdk/projectv1"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/cloudinfo"
)

// buildDependencyGraph builds the expected dependency graph and returns the results
// This version returns values instead of modifying input pointers for better clarity and testability
func (options *TestAddonOptions) buildDependencyGraph(catalogID string, offeringID string, versionLocator string, flavor string, addonConfig *cloudinfo.AddonConfig, existingVisited map[string]bool) (*DependencyGraphResult, error) {
	// Collect disabled offerings from the root addon config to propagate down the tree
	disabledOfferings := make(map[string]bool)
	for _, dep := range addonConfig.Dependencies {
		if dep.Enabled != nil && !*dep.Enabled {
			disabledOfferings[dep.OfferingName] = true
		}
	}

	return options.buildDependencyGraphWithDisabled(catalogID, offeringID, versionLocator, flavor, addonConfig, existingVisited, disabledOfferings)
}

// buildDependencyGraphWithDisabled is the internal implementation that carries disabled offerings through the recursion
func (options *TestAddonOptions) buildDependencyGraphWithDisabled(catalogID string, offeringID string, versionLocator string, flavor string, addonConfig *cloudinfo.AddonConfig, existingVisited map[string]bool, disabledOfferings map[string]bool) (*DependencyGraphResult, error) {
	// Initialize result with copies of existing state to avoid mutation
	result := &DependencyGraphResult{
		Graph:                make(map[string][]cloudinfo.OfferingReferenceDetail),
		ExpectedDeployedList: make([]cloudinfo.OfferingReferenceDetail, 0),
		Visited:              make(map[string]bool),
	}

	// Copy existing visited map to avoid modifying input
	for k, v := range existingVisited {
		result.Visited[k] = v
	}

	if result.Visited[versionLocator] {
		return result, nil
	}

	result.Visited[versionLocator] = true
	offering, _, err := options.CloudInfoService.GetOffering(catalogID, offeringID)
	if err != nil {
		options.Logger.ShortError(fmt.Sprintf("error: %v\n", err))
		return nil, err
	}

	var version catalogmanagementv1.Version
	found := false
	for _, kind := range offering.Kinds {
		if *kind.InstallKind == "terraform" {
			for _, v := range kind.Versions {
				if *v.VersionLocator == versionLocator {
					version = v
					found = true
					break
				}
			}
		}
		if found {
			break
		}
	}

	if !found {
		return nil, fmt.Errorf("version not found for version locator: %s", versionLocator)
	}

	offeringVersion := *version.Version
	offeringName := *offering.Name

	addon := cloudinfo.OfferingReferenceDetail{
		Name:    offeringName,
		Version: offeringVersion,
		Flavor:  cloudinfo.Flavor{Name: flavor},
	}
	result.ExpectedDeployedList = append(result.ExpectedDeployedList, addon)

	// Create a key for the graph map (using name:version:flavor as a unique identifier)
	addonKey := generateAddonKey(offeringName, offeringVersion, flavor)

	// Process catalog-defined dependencies that are on by default
	for _, dep := range version.SolutionInfo.Dependencies {
		if *dep.OnByDefault {
			// Check if this dependency has been disabled at the offering level
			// This applies globally across the entire dependency tree
			if disabledOfferings[*dep.Name] {
				options.Logger.ShortInfo(fmt.Sprintf("Skipping catalog dependency %s - disabled at offering level in dependency tree\n", *dep.Name))
				continue
			}

			depCatalogID := *dep.CatalogID
			depOfferingID := *dep.ID
			depFlavor := dep.Flavors[0]

			if dep.DefaultFlavor != nil && *dep.DefaultFlavor != "" {
				depFlavor = *dep.DefaultFlavor
			}

			depVersion, depVersionLocator, err := options.CloudInfoService.GetOfferingVersionLocatorByConstraint(depCatalogID, depOfferingID, *dep.Version, depFlavor)
			if err != nil {
				options.Logger.ShortError(fmt.Sprintf("error: %v\n", err))
				return nil, err
			}

			child := cloudinfo.OfferingReferenceDetail{
				Name:    *dep.Name,
				Version: depVersion,
				Flavor:  cloudinfo.Flavor{Name: depFlavor},
			}

			result.Graph[addonKey] = append(result.Graph[addonKey], child)

			// Find the corresponding manual dependency config for recursion
			var childAddonConfig *cloudinfo.AddonConfig
			for i := range addonConfig.Dependencies {
				if addonConfig.Dependencies[i].OfferingName == *dep.Name && addonConfig.Dependencies[i].OfferingFlavor == depFlavor {
					childAddonConfig = &addonConfig.Dependencies[i]
					break
				}
			}
			if childAddonConfig == nil {
				childAddonConfig = &cloudinfo.AddonConfig{
					OfferingName:   *dep.Name,
					OfferingFlavor: depFlavor,
					CatalogID:      depCatalogID,
					OfferingID:     depOfferingID,
					VersionLocator: depVersionLocator,
					Dependencies:   []cloudinfo.AddonConfig{},
				}
			}

			// Recursively process child dependencies
			childResult, err := options.buildDependencyGraphWithDisabled(depCatalogID, depOfferingID, depVersionLocator, depFlavor, childAddonConfig, result.Visited, disabledOfferings)
			if err != nil {
				return nil, err
			}

			// Merge child results into our result
			result.mergeResults(childResult)
		}
	}

	// Process manually enabled dependencies that might not be on by default
	for _, manualDep := range addonConfig.Dependencies {
		if manualDep.Enabled != nil && *manualDep.Enabled {
			// Check if this dependency has been disabled at the offering level
			if disabledOfferings[manualDep.OfferingName] {
				options.Logger.ShortInfo(fmt.Sprintf("Skipping manually enabled dependency %s - disabled at offering level in dependency tree\n", manualDep.OfferingName))
				continue
			}
			// Check if this dependency was already processed from catalog
			alreadyProcessed := false
			for _, catDep := range version.SolutionInfo.Dependencies {
				if *catDep.OnByDefault && *catDep.Name == manualDep.OfferingName {
					depFlavor := catDep.Flavors[0]
					if catDep.DefaultFlavor != nil && *catDep.DefaultFlavor != "" {
						depFlavor = *catDep.DefaultFlavor
					}
					if depFlavor == manualDep.OfferingFlavor {
						alreadyProcessed = true
						break
					}
				}
			}

			if !alreadyProcessed && !result.Visited[manualDep.VersionLocator] {
				options.Logger.ShortInfo(fmt.Sprintf("Processing manually enabled dependency %s for addon %s\n", manualDep.OfferingName, offeringName))

				child := cloudinfo.OfferingReferenceDetail{
					Name:    manualDep.OfferingName,
					Version: manualDep.ResolvedVersion,
					Flavor:  cloudinfo.Flavor{Name: manualDep.OfferingFlavor},
				}

				result.Graph[addonKey] = append(result.Graph[addonKey], child)

				childResult, err := options.buildDependencyGraphWithDisabled(manualDep.CatalogID, manualDep.OfferingID, manualDep.VersionLocator, manualDep.OfferingFlavor, &manualDep, result.Visited, disabledOfferings)
				if err != nil {
					return nil, err
				}

				result.mergeResults(childResult)
			}
		}
	}

	return result, nil
}

// mergeResults merges the child dependency graph results into the parent result
func (result *DependencyGraphResult) mergeResults(childResult *DependencyGraphResult) {
	// Merge graph entries
	for key, deps := range childResult.Graph {
		result.Graph[key] = append(result.Graph[key], deps...)
	}

	// Merge expected deployed list
	result.ExpectedDeployedList = append(result.ExpectedDeployedList, childResult.ExpectedDeployedList...)

	// Merge visited map
	for k, v := range childResult.Visited {
		result.Visited[k] = v
	}
}

// buildActuallyDeployedListFromResponse creates the actually deployed list directly from the deployment response
// This uses the actual deployed configurations returned by the deployment API, which is the source of truth
func (options *TestAddonOptions) buildActuallyDeployedListFromResponse(deployedConfigs *cloudinfo.DeployedAddonsDetails) BuildActuallyDeployedResult {
	result := BuildActuallyDeployedResult{
		ActuallyDeployedList: make([]cloudinfo.OfferingReferenceDetail, 0),
		Warnings:             make([]string, 0),
		Errors:               make([]string, 0),
	}

	if deployedConfigs == nil {
		result.Errors = append(result.Errors, "deployed configs is nil")
		return result
	}

	for _, config := range deployedConfigs.Configs {
		// For each deployed config, we need to get its offering details to create OfferingReferenceDetail
		// Get the config details to extract offering information
		configDetails, _, err := options.CloudInfoService.GetConfig(&cloudinfo.ConfigDetails{
			ProjectID: deployedConfigs.ProjectID,
			ConfigID:  config.ConfigID,
		})
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Could not get config details for %s (%s): %v", config.Name, config.ConfigID, err))
			continue
		}

		// Extract version locator from the config definition
		if defResponse, ok := configDetails.Definition.(*projectv1.ProjectConfigDefinitionResponse); ok && defResponse.LocatorID != nil {
			versionLocator := *defResponse.LocatorID

			// Get the actual version information from the catalog using the version locator
			catalogVersion, err := options.CloudInfoService.GetCatalogVersionByLocator(versionLocator)
			if err != nil {
				result.Warnings = append(result.Warnings, fmt.Sprintf("Could not get catalog version for config %s (locator: %s): %v", config.Name, versionLocator, err))
				continue
			}

			if catalogVersion == nil || catalogVersion.Version == nil {
				result.Warnings = append(result.Warnings, fmt.Sprintf("Invalid catalog version for config %s (locator: %s)", config.Name, versionLocator))
				continue
			} // Extract flavor information from the catalog version
			var flavorName string
			if catalogVersion.Flavor != nil && catalogVersion.Flavor.Name != nil {
				flavorName = *catalogVersion.Flavor.Name
			}

			// Use the catalog ID and offering ID directly from the catalog version
			if catalogVersion.CatalogID == nil {
				result.Errors = append(result.Errors, fmt.Sprintf("CatalogID is nil for config %s (locator: %s)", config.Name, versionLocator))
				continue
			}

			if catalogVersion.OfferingID == nil {
				result.Errors = append(result.Errors, fmt.Sprintf("OfferingID is nil for config %s (locator: %s)", config.Name, versionLocator))
				continue
			}

			catalogID := *catalogVersion.CatalogID
			rawOfferingID := *catalogVersion.OfferingID

			// Parse offering ID from format "<sha>:o:<offering id>"
			offeringID := rawOfferingID
			if strings.Contains(rawOfferingID, ":o:") {
				parts := strings.Split(rawOfferingID, ":o:")
				if len(parts) == 2 {
					offeringID = parts[1]
				} else {
					result.Errors = append(result.Errors, fmt.Sprintf("Invalid offering ID format for config %s: %s (expected format: <sha>:o:<offering_id>)", config.Name, rawOfferingID))
					continue
				}
			}

			// Get the offering details to retrieve the offering name
			offering, _, err := options.CloudInfoService.GetOffering(catalogID, offeringID)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("Could not get offering details for config %s (catalog: %s, offering: %s): %v", config.Name, catalogID, offeringID, err))
				continue
			}

			if offering.Name == nil {
				result.Errors = append(result.Errors, fmt.Sprintf("Offering name is nil for config %s (catalog: %s, offering: %s)", config.Name, catalogID, offeringID))
				continue
			}

			offeringName := *offering.Name

			// Create OfferingReferenceDetail from the deployed config using actual version string
			offeringDetail := cloudinfo.OfferingReferenceDetail{
				Name:    offeringName,            // Use the offering name from catalog
				Version: *catalogVersion.Version, // Use the actual version string from catalog
				Flavor:  cloudinfo.Flavor{Name: flavorName},
			}

			result.ActuallyDeployedList = append(result.ActuallyDeployedList, offeringDetail)
		} else {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Could not get locator ID for config %s", config.Name))
		}
	}

	return result
}

// validateDependencies compares expected dependency graph and actually deployed configurations
// Returns a ValidationResult containing all validation issues found instead of failing on first error
func (options *TestAddonOptions) validateDependencies(graph map[string][]cloudinfo.OfferingReferenceDetail, expectedDeployedList []cloudinfo.OfferingReferenceDetail, actuallyDeployedList []cloudinfo.OfferingReferenceDetail) ValidationResult {
	result := ValidationResult{
		IsValid:           true,
		DependencyErrors:  make([]cloudinfo.DependencyError, 0),
		UnexpectedConfigs: make([]cloudinfo.OfferingReferenceDetail, 0),
		MissingConfigs:    make([]cloudinfo.OfferingReferenceDetail, 0),
		Messages:          make([]string, 0),
	}

	// Check for missing dependencies in the graph
	for addonKey, dependencies := range graph {
		// Parse addon info from the key format "name:version:flavor"
		keyParts := strings.Split(addonKey, ":")
		if len(keyParts) != 3 {
			options.Logger.ShortWarn(fmt.Sprintf("Invalid addon key format: %s", addonKey))
			continue
		}

		addon := cloudinfo.OfferingReferenceDetail{
			Name:    keyParts[0],
			Version: keyParts[1],
			Flavor:  cloudinfo.Flavor{Name: keyParts[2]},
		}

		for _, dep := range dependencies {
			found := false
			for _, dep2 := range actuallyDeployedList {
				if dep.Name == dep2.Name && dep.Version == dep2.Version && dep.Flavor.Name == dep2.Flavor.Name {
					found = true
					break
				}
			}

			if !found {
				availableVersions := make([]cloudinfo.OfferingReferenceDetail, 0)
				for _, dep2 := range actuallyDeployedList {
					if dep2.Name == dep.Name {
						availableVersions = append(availableVersions, cloudinfo.OfferingReferenceDetail{
							Name:    dep2.Name,
							Version: dep2.Version,
							Flavor:  cloudinfo.Flavor{Name: dep2.Flavor.Name},
						})
					}
				}
				result.DependencyErrors = append(result.DependencyErrors, cloudinfo.DependencyError{
					Addon:                 addon,
					DependencyRequired:    dep,
					DependenciesAvailable: availableVersions,
				})
				result.IsValid = false
			}
		}
	}

	// Check for unexpected configs (deployed but not expected)
	for _, actualConfig := range actuallyDeployedList {
		found := false
		for _, expectedConfig := range expectedDeployedList {
			if actualConfig.Name == expectedConfig.Name && actualConfig.Version == expectedConfig.Version && actualConfig.Flavor.Name == expectedConfig.Flavor.Name {
				found = true
				break
			}
		}
		if !found {
			result.UnexpectedConfigs = append(result.UnexpectedConfigs, actualConfig)
			result.IsValid = false
		}
	}

	// Check for missing configs (expected but not deployed)
	for _, expectedConfig := range expectedDeployedList {
		found := false
		for _, actualConfig := range actuallyDeployedList {
			if expectedConfig.Name == actualConfig.Name && expectedConfig.Version == actualConfig.Version && expectedConfig.Flavor.Name == actualConfig.Flavor.Name {
				found = true
				break
			}
		}
		if !found {
			result.MissingConfigs = append(result.MissingConfigs, expectedConfig)
			result.IsValid = false
		}
	}

	// Check if lengths differ
	if len(expectedDeployedList) != len(actuallyDeployedList) {
		result.IsValid = false
	}

	// Generate summary messages
	// Only add success message for truly successful validations - don't add it during failed tests
	// The success message should only appear when the overall test passes, not just when validation passes
	if result.IsValid {
		result.Messages = append(result.Messages, "actually deployed configs are same as expected deployed configs")
	} else {
		if len(result.DependencyErrors) > 0 {
			result.Messages = append(result.Messages, fmt.Sprintf("found %d dependency errors", len(result.DependencyErrors)))
		}
		if len(result.UnexpectedConfigs) > 0 {
			result.Messages = append(result.Messages, fmt.Sprintf("found %d unexpected configs", len(result.UnexpectedConfigs)))
		}
		if len(result.MissingConfigs) > 0 {
			result.Messages = append(result.Messages, fmt.Sprintf("found %d missing expected configs", len(result.MissingConfigs)))
		}
	}

	return result
}
