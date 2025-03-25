package cloudinfo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/platform-services-go-sdk/catalogmanagementv1"
	project "github.com/IBM/project-go-sdk/projectv1"
	"github.com/gruntwork-io/terratest/modules/random"
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

// Addon Functions:

// DeployAddonToProject deploys an addon to a project
// POST /api/v1-beta/deploy/projects/:projectID/container
// [
//
//	{
//	    "version_locator": "9212a6da-ac9b-4f3c-94d8-83a866e1a250.cb157ad2-0bf7-488c-bdd4-5c568d611423"
//	},
//	{
//	    "version_locator": "9212a6da-ac9b-4f3c-94d8-83a866e1a250.3a38fa8e-12ba-472b-be07-832fcb1ae914"
//	},
//	{
//	    "version_locator": "9212a6da-ac9b-4f3c-94d8-83a866e1a250.12fa081a-47f1-473c-9acc-70812f66c26b",
//	    "config_id": "",    // set this if reusing an existing config
//	    "name": "sm da"
//	}
//
// ]
func (infoSvc *CloudInfoService) DeployAddonToProject(addonConfig *AddonConfig, projectConfig *ProjectsConfig) (*project.ProjectConfig, error) {
	// get all the version locators for the addon
	componentsReferences, err := infoSvc.GetComponentReferences(addonConfig.VersionLocator)
	if err != nil {
		return nil, fmt.Errorf("error getting component references: %w", err)
	}

	// Create a map to track existing dependencies by name
	existingDependencies := make(map[string]bool)
	for _, dep := range addonConfig.Dependencies {
		existingDependencies[dep.OfferingName] = true
	}

	// Update existing dependencies and collect components to add
	var componentsToAdd []OfferingReferenceItem

	// Process required references first (they take precedence)
	for _, component := range componentsReferences.Required.OfferingReferences {
		found := false
		for i := range addonConfig.Dependencies {
			if addonConfig.Dependencies[i].OfferingName == component.Name {
				// Update the version locator for this dependency
				addonConfig.Dependencies[i].VersionLocator = component.OfferingReference.VersionLocator
				addonConfig.Dependencies[i].ResolvedVersion = component.OfferingReference.Version
				addonConfig.Dependencies[i].Enabled = true // Required components are always enabled
				found = true
				break
			}
		}

		if !found {
			componentsToAdd = append(componentsToAdd, component)
		}
	}

	// Process optional references
	for _, component := range componentsReferences.Optional.OfferingReferences {
		// Skip if this is already in required references (we processed those already)
		if existingDependencies[component.Name] {
			continue
		}

		found := false
		for i := range addonConfig.Dependencies {
			if addonConfig.Dependencies[i].OfferingName == component.Name {
				// Update the version locator for this dependency
				addonConfig.Dependencies[i].VersionLocator = component.OfferingReference.VersionLocator
				addonConfig.Dependencies[i].ResolvedVersion = component.OfferingReference.Version
				addonConfig.Dependencies[i].OnByDefault = component.OfferingReference.OnByDefault
				found = true
				break
			}
		}

		if !found {
			// set required to on by default true
			component.OfferingReference.OnByDefault = true
			componentsToAdd = append(componentsToAdd, component)
		}
	}

	// Add new dependencies that weren't found in the existing dependencies
	for _, component := range componentsToAdd {

		newDependency := AddonConfig{
			OfferingName:    component.OfferingReference.Name,
			OfferingLabel:   component.OfferingReference.Label,
			VersionLocator:  component.OfferingReference.VersionLocator,
			ResolvedVersion: component.OfferingReference.Version,
			OnByDefault:     component.OfferingReference.OnByDefault,
			Enabled:         component.OfferingReference.OnByDefault, // Required components have been forced to enabled
			OfferingID:      component.OfferingReference.ID,
			Inputs:          make(map[string]interface{}),
		}
		addonConfig.Dependencies = append(addonConfig.Dependencies, newDependency)
	}

	// Create the request body
	addonDependecies := make([]map[string]string, 0, len(addonConfig.Dependencies)+1)
	// Add the addon itself
	addonDependecies = append(addonDependecies, map[string]string{
		"version_locator": addonConfig.VersionLocator,
		"name":            fmt.Sprintf("%s-%s", addonConfig.Prefix, addonConfig.OfferingName),
	})

	// Add the dependencies
	for _, dep := range addonConfig.Dependencies {
		if dep.Enabled {
			randomPostfix := strings.ToLower(random.UniqueId())
			dependencyEntry := map[string]string{
				"version_locator": dep.VersionLocator,
				"name":            fmt.Sprintf("%s-%s", dep.OfferingName, randomPostfix),
			}
			if dep.ExistingConfigID != "" {
				dependencyEntry["config_id"] = dep.ExistingConfigID
			}
			addonDependecies = append(addonDependecies, dependencyEntry)
		}
	}

	// Convert the addonDependecies to JSON
	jsonBody, err := json.Marshal(addonDependecies)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request body: %w", err)
	}

	// print the json body pretty
	var prettyJSON bytes.Buffer

	err = json.Indent(&prettyJSON, jsonBody, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error pretty printing json: %w", err)
	}

	fmt.Println("Request Body: ", prettyJSON.String())

	// Create a new HTTP request with the JSON body
	url := fmt.Sprintf("https://cm.globalcatalog.cloud.ibm.com/api/v1-beta/deploy/projects/%s/container", projectConfig.ProjectID)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Add headers
	token, err := infoSvc.authenticator.GetToken()
	if err != nil {
		return nil, fmt.Errorf("error getting auth token: %w", err)
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Add("Content-Type", "application/json")

	// Execute the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil, nil
}

// GetComponentReferences gets the component references for a version locator
// /ui/v1/versions/:version_locator/componentsReferences
func (infoSvc *CloudInfoService) GetComponentReferences(versionLocator string) (*OfferingReferenceResponse, error) {

	// Build the request URL
	url := fmt.Sprintf("https://cm.globalcatalog.cloud.ibm.com/ui/v1/versions/%s/componentsReferences", versionLocator)

	// Create a new HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Add authorization header
	token, err := infoSvc.authenticator.GetToken()
	if err != nil {
		return nil, fmt.Errorf("error getting auth token: %w", err)
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Add("Content-Type", "application/json")

	// Execute the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Unmarshal the response into OfferingReferenceResponse
	var offeringReferences OfferingReferenceResponse
	if err := json.Unmarshal(body, &offeringReferences); err != nil {
		return nil, fmt.Errorf("error unmarshaling offering references: %w", err)
	}

	return &offeringReferences, nil
}
