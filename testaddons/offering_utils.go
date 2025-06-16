package testaddons

import (
	"fmt"
	"strings"
)

// SetOfferingDetails sets offering details for the addon and its dependencies
func SetOfferingDetails(options *TestAddonOptions) {

	// set top level offering required inputs
	var topLevelVersion string
	locatorParts := strings.Split(options.AddonConfig.VersionLocator, ".")
	if len(locatorParts) > 1 {
		topLevelVersion = locatorParts[1]
	} else {
		options.Logger.ShortError(fmt.Sprintf("Error, Could not parse VersionLocator: %s", options.AddonConfig.VersionLocator))
	}
	topLevelOffering, _, err := options.CloudInfoService.GetOffering(*options.offering.CatalogID, *options.offering.ID)
	if err != nil {
		options.Logger.ShortError(fmt.Sprintf("Error retrieving top level offering: %s from catalog: %s", *options.offering.ID, *options.offering.CatalogID))
	}
	if *topLevelOffering.Kinds[0].InstallKind != "terraform" {
		options.Logger.ShortError(fmt.Sprintf("Error, top level offering: %s, Expected Kind 'terraform' got '%s'", *options.offering.ID, *topLevelOffering.Kinds[0].InstallKind))
	}
	options.AddonConfig.OfferingInputs = options.CloudInfoService.GetOfferingInputs(topLevelOffering, topLevelVersion, *options.offering.ID)
	options.AddonConfig.VersionID = topLevelVersion
	options.AddonConfig.CatalogID = *options.offering.CatalogID

	// set dependency offerings required inputs
	for i, dependency := range options.AddonConfig.Dependencies {
		offeringDependencyVersionLocator := strings.Split(dependency.VersionLocator, ".")
		dependencyCatalogID := offeringDependencyVersionLocator[0]
		dependencyVersionID := offeringDependencyVersionLocator[1]
		if dependency.OfferingID == "" {
			options.Logger.ShortError(fmt.Sprintf("Error, dependency offering ID is not set for dependency: %s", dependency.OfferingName))
			options.Testing.Fail()
			return
		}
		myOffering, _, err := options.CloudInfoService.GetOffering(dependencyCatalogID, dependency.OfferingID)
		if err != nil {
			if myOffering == nil {
				options.Logger.ShortError(fmt.Sprintf("Error retrieving dependency offering: %s from catalog: %s, offering not found", dependency.OfferingID, dependencyCatalogID))
			} else {
				options.Logger.ShortError(fmt.Sprintf("Error retrieving dependency offering: %s from catalog: %s", *myOffering.ID, dependencyCatalogID))
			}
		}
		options.AddonConfig.Dependencies[i].OfferingInputs = options.CloudInfoService.GetOfferingInputs(myOffering, dependencyVersionID, dependencyCatalogID)
		options.AddonConfig.Dependencies[i].VersionID = dependencyVersionID
		options.AddonConfig.Dependencies[i].CatalogID = dependencyCatalogID
	}
}
