package cloudinfo

import (
	"fmt"

	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/platform-services-go-sdk/catalogmanagementv1"
)

// GetCatalogVersionByLocator gets a version by its locator using the Catalog Management service
func (infoSvc *CloudInfoService) GetCatalogVersionByLocator(versionLocator string) (*catalogmanagementv1.Version, error) {
	// Call the GetCatalogVersionByLocator method with the version locator and the context
	getVersionOptions := &catalogmanagementv1.GetVersionOptions{
		VersionLocID: &versionLocator,
	}

	offering, response, err := infoSvc.catalogService.GetVersion(getVersionOptions)
	if err != nil {
		return nil, err
	}

	// Check if the response status code is 200 (success)
	if response.StatusCode == 200 {
		var version catalogmanagementv1.Version
		if offering.Kinds != nil {
			version = offering.Kinds[0].Versions[0]
		}
		if &version != nil {
			return &version, nil
		}
		return nil, fmt.Errorf("version not found")
	}

	return nil, fmt.Errorf("failed to get version: %s", response.RawResult)
}

// CreateCatalog creates a new private catalog using the Catalog Management service
func (infoSvc *CloudInfoService) CreateCatalog(catalogName string) (*catalogmanagementv1.Catalog, error) {
	// Call the CreateCatalog method with the catalog name and the context
	createCatalogOptions := &catalogmanagementv1.CreateCatalogOptions{
		Label: &catalogName,
	}

	catalog, response, err := infoSvc.catalogService.CreateCatalog(createCatalogOptions)
	if err != nil {
		return nil, err
	}

	// Check if the response status code is 201 (created)
	if response.StatusCode == 201 {
		return catalog, nil
	}

	return nil, fmt.Errorf("failed to create catalog: %s", response.RawResult)
}

// DeleteCatalog deletes a private catalog using the Catalog Management service
func (infoSvc *CloudInfoService) DeleteCatalog(catalogID string) error {
	// Call the DeleteCatalog method with the catalog ID and the context
	deleteCatalogOptions := &catalogmanagementv1.DeleteCatalogOptions{
		CatalogIdentifier: &catalogID,
	}

	response, err := infoSvc.catalogService.DeleteCatalog(deleteCatalogOptions)
	if err != nil {
		return err
	}

	// Check if the response status code is 200 (Successful Result)
	if response.StatusCode == 200 {
		return nil
	}

	return fmt.Errorf("failed to delete catalog: %s", response.RawResult)
}

// ImportOffering Import a new offering using the Catalog Management service
// catalogID: The ID of the catalog to import the offering to
// zipUrl: The URL of the zip file containing the offering or url to the branch
// offeringName: The name of the offering to import
// flavorName: The name of the flavor to import Note: programatic name not label
// version: The version of the offering
// installKind: The kind of install to use
func (infoSvc *CloudInfoService) ImportOffering(catalogID string, zipUrl string, offeringName string, flavorName string, version string, installKind InstallKind) (*catalogmanagementv1.Offering, error) {

	flavorInstance := &catalogmanagementv1.Flavor{
		Name: &flavorName,
	}

	// Call the ImportOffering method with the catalog ID, offering ID, and the context
	importOfferingOptions := &catalogmanagementv1.ImportOfferingOptions{
		CatalogIdentifier: &catalogID,
		Zipurl:            &zipUrl,
		TargetVersion:     &version,
		InstallType:       core.StringPtr("fullstack"),
		Name:              core.StringPtr(offeringName),
		Flavor:            flavorInstance,
		ProductKind:       core.StringPtr("solution"),
		FormatKind:        core.StringPtr(installKind.String()),
	}

	offering, response, err := infoSvc.catalogService.ImportOffering(importOfferingOptions)
	if err != nil {
		return nil, err
	}

	// Check if the response status code is 201 (created)
	if response.StatusCode == 201 {
		return offering, nil
	}

	return nil, fmt.Errorf("failed to import offering: %s", response.RawResult)
}
