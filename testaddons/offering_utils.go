package testaddons

import (
	"fmt"
	"strings"
)

// SetOfferingDetails sets offering details for the addon and its dependencies
func SetOfferingDetails(options *TestAddonOptions) error {
	// Check if offering is nil
	if options.offering == nil {
		options.Logger.ShortError("Error: offering is nil, cannot set offering details")
		options.Logger.MarkFailed()
		options.Logger.FlushOnFailure()
		options.Testing.Fail()
		return fmt.Errorf("offering is nil, cannot set offering details")
	}

	// Check if offering ID is nil
	if options.offering.ID == nil {
		options.Logger.ShortError("Error: offering ID is nil, cannot set offering details")
		options.Logger.MarkFailed()
		options.Logger.FlushOnFailure()
		options.Testing.Fail()
		return fmt.Errorf("offering ID is nil, cannot set offering details")
	}

	// Check if offering CatalogID is nil
	if options.offering.CatalogID == nil {
		options.Logger.ShortError("Error: offering CatalogID is nil, cannot set offering details")
		options.Logger.MarkFailed()
		options.Logger.FlushOnFailure()
		options.Testing.Fail()
		return fmt.Errorf("offering CatalogID is nil, cannot set offering details")
	}

	options.Logger.ShortInfo(fmt.Sprintf("Setting offering details for offeringID='%s', catalogID='%s'", *options.offering.ID, *options.offering.CatalogID))

	// Seed critical IDs BEFORE any network calls so validation can proceed even if we hit 429s
	if options.AddonConfig.OfferingID == "" {
		options.AddonConfig.OfferingID = *options.offering.ID
		options.Logger.ShortInfo("Seeded AddonConfig.OfferingID from offering data")
	}
	if options.AddonConfig.CatalogID == "" {
		// Prefer offering.CatalogID, fall back to catalog.ID if available
		if options.offering.CatalogID != nil {
			options.AddonConfig.CatalogID = *options.offering.CatalogID
			options.Logger.ShortInfo("Seeded AddonConfig.CatalogID from offering data")
		} else if options.catalog != nil && options.catalog.ID != nil {
			options.AddonConfig.CatalogID = *options.catalog.ID
			options.Logger.ShortInfo("Seeded AddonConfig.CatalogID from catalog object")
		}
	}

	// set top level offering required inputs
	var topLevelVersion string
	locatorParts := strings.Split(options.AddonConfig.VersionLocator, ".")
	if len(locatorParts) > 1 {
		topLevelVersion = locatorParts[1]
	} else {
		options.Logger.ShortError(fmt.Sprintf("Error, Could not parse VersionLocator: %s", options.AddonConfig.VersionLocator))
	}

	// Get top level offering - GetOffering already has retry logic built in
	topLevelOffering, _, err := options.CloudInfoService.GetOffering(*options.offering.CatalogID, *options.offering.ID)
	if topLevelOffering == nil && err == nil {
		err = fmt.Errorf("offering is nil")
	}

	if err != nil {
		options.Logger.ShortError(fmt.Sprintf("Error retrieving top level offering: %s from catalog: %s - %v", *options.offering.ID, *options.offering.CatalogID, err))
		options.Logger.MarkFailed()
		options.Logger.FlushOnFailure()
		options.Testing.Fail()
		return fmt.Errorf("error retrieving top level offering %s from catalog %s: %w", *options.offering.ID, *options.offering.CatalogID, err)
	}
	if topLevelOffering == nil || len(topLevelOffering.Kinds) == 0 || topLevelOffering.Kinds[0].InstallKind == nil {
		options.Logger.ShortError(fmt.Sprintf("Error, top level offering: %s, install kind is nil or not available", *options.offering.ID))
		options.Logger.MarkFailed()
		options.Logger.FlushOnFailure()
		options.Testing.Fail()
		return fmt.Errorf("top level offering %s install kind is nil or not available", *options.offering.ID)
	}
	if *topLevelOffering.Kinds[0].InstallKind != "terraform" {
		options.Logger.ShortError(fmt.Sprintf("Error, top level offering: %s, Expected Kind 'terraform' got '%s'", *options.offering.ID, *topLevelOffering.Kinds[0].InstallKind))
	}
	options.AddonConfig.OfferingInputs = options.CloudInfoService.GetOfferingInputs(topLevelOffering, topLevelVersion, *options.offering.ID)
	options.AddonConfig.VersionID = topLevelVersion

	// Ensure we log the final seeded values for traceability
	options.Logger.ShortInfo(fmt.Sprintf("Confirmed AddonConfig IDs: CatalogID='%s', OfferingID='%s'",
		options.AddonConfig.CatalogID, options.AddonConfig.OfferingID))

	// Confirm that critical values were set successfully
	options.Logger.ShortInfo(fmt.Sprintf("Successfully set AddonConfig: CatalogID='%s', OfferingID='%s', VersionLocator='%s'",
		options.AddonConfig.CatalogID, options.AddonConfig.OfferingID, options.AddonConfig.VersionLocator))

	// set dependency offerings required inputs
	for i, dependency := range options.AddonConfig.Dependencies {
		if dependency.VersionLocator == "" {
			options.Logger.ShortError(fmt.Sprintf("Error, could not find version for offering: %s %s", dependency.OfferingName, dependency.OfferingFlavor))
			options.Logger.MarkFailed()
			options.Logger.FlushOnFailure()
			options.Testing.Fail()
			return fmt.Errorf("could not find version for offering: %s %s", dependency.OfferingName, dependency.OfferingFlavor)
		}
		offeringDependencyVersionLocator := strings.Split(dependency.VersionLocator, ".")
		dependencyCatalogID := offeringDependencyVersionLocator[0]
		dependencyVersionID := offeringDependencyVersionLocator[1]
		if dependency.OfferingID == "" {
			options.Logger.ShortError(fmt.Sprintf("Error, dependency offering ID is not set for dependency: %s", dependency.OfferingName))
			options.Logger.MarkFailed()
			options.Logger.FlushOnFailure()
			options.Testing.Fail()
			return fmt.Errorf("dependency offering ID is not set for dependency: %s", dependency.OfferingName)
		}

		// Get dependency offering - GetOffering already has retry logic built in
		myOffering, _, err := options.CloudInfoService.GetOffering(dependencyCatalogID, dependency.OfferingID)
		if myOffering == nil && err == nil {
			err = fmt.Errorf("dependency offering not found")
		}

		if err != nil {
			options.Logger.ShortError(fmt.Sprintf("Error retrieving dependency offering: %s from catalog: %s - %v", dependency.OfferingID, dependencyCatalogID, err))
			options.Logger.MarkFailed()
			options.Logger.FlushOnFailure()
			options.Testing.Fail()
			return fmt.Errorf("error retrieving dependency offering %s from catalog %s: %w", dependency.OfferingID, dependencyCatalogID, err)
		}
		options.AddonConfig.Dependencies[i].OfferingInputs = options.CloudInfoService.GetOfferingInputs(myOffering, dependencyVersionID, dependencyCatalogID)
		options.AddonConfig.Dependencies[i].VersionID = dependencyVersionID
		options.AddonConfig.Dependencies[i].CatalogID = dependencyCatalogID
	}
	return nil
}
