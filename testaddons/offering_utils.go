package testaddons

import (
	"fmt"
	"strings"

	"github.com/IBM/platform-services-go-sdk/catalogmanagementv1"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/common"
)

// SetOfferingDetails sets offering details for the addon and its dependencies
func SetOfferingDetails(options *TestAddonOptions) {
	// Check if offering is nil
	if options.offering == nil {
		options.Logger.ShortError("Error: offering is nil, cannot set offering details")
		options.Logger.MarkFailed()
		options.Logger.FlushOnFailure()
		options.Testing.Fail()
		return
	}

	// Check if offering ID is nil
	if options.offering.ID == nil {
		options.Logger.ShortError("Error: offering ID is nil, cannot set offering details")
		options.Logger.MarkFailed()
		options.Logger.FlushOnFailure()
		options.Testing.Fail()
		return
	}

	// Check if offering CatalogID is nil
	if options.offering.CatalogID == nil {
		options.Logger.ShortError("Error: offering CatalogID is nil, cannot set offering details")
		options.Logger.MarkFailed()
		options.Logger.FlushOnFailure()
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

	// Use common retry utility for getting top level offering in parallel execution
	config := common.CatalogOperationRetryConfig()
	if options.CatalogRetryConfig != nil {
		config = *options.CatalogRetryConfig
	}
	config.Logger = options.Logger.GetUnderlyingLogger()
	config.OperationName = fmt.Sprintf("GetOffering catalogID='%s', offeringID='%s'", *options.offering.CatalogID, *options.offering.ID)
	config.MaxRetries = 3 // Keep same retry count as before

	topLevelOffering, err := common.RetryWithConfig(config, func() (*catalogmanagementv1.Offering, error) {
		offering, _, err := options.CloudInfoService.GetOffering(*options.offering.CatalogID, *options.offering.ID)
		if err != nil {
			return nil, err
		}
		if offering == nil {
			return nil, fmt.Errorf("offering is nil")
		}
		return offering, nil
	})

	if err != nil {
		options.Logger.ShortError(fmt.Sprintf("Error retrieving top level offering: %s from catalog: %s - %v", *options.offering.ID, *options.offering.CatalogID, err))
		options.Logger.MarkFailed()
		options.Logger.FlushOnFailure()
		options.Testing.Fail()
		return
	}
	if topLevelOffering == nil || len(topLevelOffering.Kinds) == 0 || topLevelOffering.Kinds[0].InstallKind == nil {
		options.Logger.ShortError(fmt.Sprintf("Error, top level offering: %s, install kind is nil or not available", *options.offering.ID))
		options.Logger.MarkFailed()
		options.Logger.FlushOnFailure()
		options.Testing.Fail()
		return
	}
	if *topLevelOffering.Kinds[0].InstallKind != "terraform" {
		options.Logger.ShortError(fmt.Sprintf("Error, top level offering: %s, Expected Kind 'terraform' got '%s'", *options.offering.ID, *topLevelOffering.Kinds[0].InstallKind))
	}
	options.AddonConfig.OfferingInputs = options.CloudInfoService.GetOfferingInputs(topLevelOffering, topLevelVersion, *options.offering.ID)
	options.AddonConfig.VersionID = topLevelVersion

	// Defensive check for CatalogID to handle race conditions in parallel tests
	if options.offering.CatalogID == nil {
		options.Logger.ShortWarn("Offering CatalogID is nil - attempting to recover from catalog object (this may indicate a race condition in parallel test execution)")
		// Try to get catalog ID from the catalog object as fallback
		if options.catalog != nil && options.catalog.ID != nil {
			options.AddonConfig.CatalogID = *options.catalog.ID
			options.Logger.ShortInfo(fmt.Sprintf("Recovered CatalogID from catalog object: %s", options.AddonConfig.CatalogID))
		} else {
			options.Logger.ShortError("Cannot recover CatalogID - both offering.CatalogID and catalog.ID are nil")
			options.Logger.MarkFailed()
			options.Logger.FlushOnFailure()
			options.Testing.Fail()
			return
		}
	} else {
		options.AddonConfig.CatalogID = *options.offering.CatalogID
	}

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
			options.Logger.MarkFailed()
			options.Logger.FlushOnFailure()
			options.Testing.Fail()
			return
		}

		// Use common retry utility for dependency offerings as well
		dependencyConfig := common.CatalogOperationRetryConfig()
		if options.CatalogRetryConfig != nil {
			dependencyConfig = *options.CatalogRetryConfig
		}
		dependencyConfig.Logger = options.Logger.GetUnderlyingLogger()
		dependencyConfig.OperationName = fmt.Sprintf("GetOffering dependency catalogID='%s', offeringID='%s'", dependencyCatalogID, dependency.OfferingID)
		dependencyConfig.MaxRetries = 3 // Keep same retry count as before

		myOffering, err := common.RetryWithConfig(dependencyConfig, func() (*catalogmanagementv1.Offering, error) {
			offering, _, err := options.CloudInfoService.GetOffering(dependencyCatalogID, dependency.OfferingID)
			if err != nil {
				return nil, err
			}
			if offering == nil {
				return nil, fmt.Errorf("dependency offering not found")
			}
			return offering, nil
		})

		if err != nil {
			options.Logger.ShortError(fmt.Sprintf("Error retrieving dependency offering: %s from catalog: %s - %v", dependency.OfferingID, dependencyCatalogID, err))
			options.Logger.MarkFailed()
			options.Logger.FlushOnFailure()
			options.Testing.Fail()
			return
		}
		options.AddonConfig.Dependencies[i].OfferingInputs = options.CloudInfoService.GetOfferingInputs(myOffering, dependencyVersionID, dependencyCatalogID)
		options.AddonConfig.Dependencies[i].VersionID = dependencyVersionID
		options.AddonConfig.Dependencies[i].CatalogID = dependencyCatalogID
	}
}
