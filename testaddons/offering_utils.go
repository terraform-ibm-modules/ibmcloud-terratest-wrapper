package testaddons

import (
	"fmt"
	"strings"
	"time"

	"github.com/IBM/platform-services-go-sdk/catalogmanagementv1"
)

// SetOfferingDetails sets offering details for the addon and its dependencies
func SetOfferingDetails(options *TestAddonOptions) {
	// Check if offering is nil
	if options.offering == nil {
		options.Logger.ShortError("Error: offering is nil, cannot set offering details")
		options.Testing.Fail()
		return
	}

	// Check if offering ID is nil
	if options.offering.ID == nil {
		options.Logger.ShortError("Error: offering ID is nil, cannot set offering details")
		options.Testing.Fail()
		return
	}

	// Check if offering CatalogID is nil
	if options.offering.CatalogID == nil {
		options.Logger.ShortError("Error: offering CatalogID is nil, cannot set offering details")
		options.Testing.Fail()
		return
	}

	options.Logger.ShortInfo(fmt.Sprintf("Setting offering details for offeringID='%s', catalogID='%s'", *options.offering.ID, *options.offering.CatalogID))

	// set top level offering required inputs
	var topLevelVersion string
	locatorParts := strings.Split(options.AddonConfig.VersionLocator, ".")
	if len(locatorParts) > 1 {
		topLevelVersion = locatorParts[1]
	} else {
		options.Logger.ShortError(fmt.Sprintf("Error, Could not parse VersionLocator: %s", options.AddonConfig.VersionLocator))
	}

	// Add retry logic for getting top level offering in parallel execution
	const maxRetries = 3
	const baseDelay = 2 * time.Second

	var topLevelOffering *catalogmanagementv1.Offering
	var err error

	// Retry getting the top level offering in case of timing issues in parallel tests
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(float64(baseDelay) * float64(attempt))
			options.Logger.ShortInfo(fmt.Sprintf("Retrying GetOffering after %v (attempt %d/%d)", delay, attempt+1, maxRetries))
			time.Sleep(delay)
		}

		topLevelOffering, _, err = options.CloudInfoService.GetOffering(*options.offering.CatalogID, *options.offering.ID)
		if err == nil && topLevelOffering != nil {
			break
		}

		if attempt == maxRetries-1 {
			options.Logger.ShortError(fmt.Sprintf("Error retrieving top level offering after %d attempts: %s from catalog: %s", maxRetries, *options.offering.ID, *options.offering.CatalogID))
			options.Testing.Fail()
			return
		}
	}
	if topLevelOffering == nil || len(topLevelOffering.Kinds) == 0 || topLevelOffering.Kinds[0].InstallKind == nil {
		options.Logger.ShortError(fmt.Sprintf("Error, top level offering: %s, install kind is nil or not available", *options.offering.ID))
		options.Testing.Fail()
		return
	}
	if *topLevelOffering.Kinds[0].InstallKind != "terraform" {
		options.Logger.ShortError(fmt.Sprintf("Error, top level offering: %s, Expected Kind 'terraform' got '%s'", *options.offering.ID, *topLevelOffering.Kinds[0].InstallKind))
	}
	options.AddonConfig.OfferingInputs = options.CloudInfoService.GetOfferingInputs(topLevelOffering, topLevelVersion, *options.offering.ID)
	options.AddonConfig.VersionID = topLevelVersion
	options.AddonConfig.CatalogID = *options.offering.CatalogID

	// Ensure OfferingID is set (it should have been set in setupOffering, but add safety check for race conditions in parallel tests)
	if options.AddonConfig.OfferingID == "" {
		options.Logger.ShortWarn("AddonConfig.OfferingID was empty, setting it from offering data (this may indicate a race condition in parallel tests)")
		options.AddonConfig.OfferingID = *options.offering.ID
	}

	// Confirm that critical values were set successfully
	options.Logger.ShortInfo(fmt.Sprintf("Successfully set AddonConfig: CatalogID='%s', OfferingID='%s', VersionLocator='%s'",
		options.AddonConfig.CatalogID, options.AddonConfig.OfferingID, options.AddonConfig.VersionLocator))

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

		// Add retry logic for dependency offerings as well
		var myOffering *catalogmanagementv1.Offering
		for attempt := 0; attempt < maxRetries; attempt++ {
			if attempt > 0 {
				delay := time.Duration(float64(baseDelay) * float64(attempt))
				options.Logger.ShortInfo(fmt.Sprintf("Retrying GetOffering for dependency after %v (attempt %d/%d)", delay, attempt+1, maxRetries))
				time.Sleep(delay)
			}

			myOffering, _, err = options.CloudInfoService.GetOffering(dependencyCatalogID, dependency.OfferingID)
			if err == nil && myOffering != nil {
				break
			}

			if attempt == maxRetries-1 {
				if myOffering == nil {
					options.Logger.ShortError(fmt.Sprintf("Error retrieving dependency offering after %d attempts: %s from catalog: %s, offering not found", maxRetries, dependency.OfferingID, dependencyCatalogID))
				} else {
					options.Logger.ShortError(fmt.Sprintf("Error retrieving dependency offering after %d attempts: %s from catalog: %s", maxRetries, *myOffering.ID, dependencyCatalogID))
				}
				options.Testing.Fail()
				return
			}
		}
		options.AddonConfig.Dependencies[i].OfferingInputs = options.CloudInfoService.GetOfferingInputs(myOffering, dependencyVersionID, dependencyCatalogID)
		options.AddonConfig.Dependencies[i].VersionID = dependencyVersionID
		options.AddonConfig.Dependencies[i].CatalogID = dependencyCatalogID
	}
}
