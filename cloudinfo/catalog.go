package cloudinfo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/platform-services-go-sdk/catalogmanagementv1"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/common"
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
		if len(offering.Kinds) > 0 && len(offering.Kinds[0].Versions) > 0 {
			version = offering.Kinds[0].Versions[0]
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

// processComponentReferences recursively processes component references for an addon and its dependencies
// It returns a map of processed version locators to avoid circular dependencies
func (infoSvc *CloudInfoService) processComponentReferences(addonConfig *AddonConfig, processedLocators map[string]bool) error {
	// If we've already processed this version locator, skip it to avoid circular dependencies
	if processedLocators[addonConfig.VersionLocator] {
		return nil
	}

	// Mark this locator as processed
	processedLocators[addonConfig.VersionLocator] = true

	// Get component references for this addon
	componentsReferences, err := infoSvc.GetComponentReferences(addonConfig.VersionLocator)
	if err != nil {
		return fmt.Errorf("error getting component references for %s: %w", addonConfig.VersionLocator, err)
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

				// Process dependencies of this dependency recursively
				if err := infoSvc.processComponentReferences(&addonConfig.Dependencies[i], processedLocators); err != nil {
					return err
				}

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
				addonConfig.Dependencies[i].Prefix = addonConfig.Prefix
				addonConfig.Dependencies[i].OfferingFlavor = component.OfferingReference.Flavor.Name
				addonConfig.Dependencies[i].OfferingLabel = component.OfferingReference.Label
				found = true

				// Process dependencies of this dependency recursively
				if err := infoSvc.processComponentReferences(&addonConfig.Dependencies[i], processedLocators); err != nil {
					return err
				}

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
			Prefix:          addonConfig.Prefix,
			OfferingName:    component.OfferingReference.Name,
			OfferingLabel:   component.OfferingReference.Label,
			OfferingFlavor:  component.OfferingReference.Flavor.Name,
			VersionLocator:  component.OfferingReference.VersionLocator,
			ResolvedVersion: component.OfferingReference.Version,
			OnByDefault:     component.OfferingReference.OnByDefault,
			Enabled:         component.OfferingReference.OnByDefault, // Required components have been forced to enabled
			OfferingID:      component.OfferingReference.ID,
			Inputs:          make(map[string]interface{}),
			Dependencies:    []AddonConfig{}, // Initialize empty dependencies slice
		}

		// Process dependencies of this new dependency recursively
		if err := infoSvc.processComponentReferences(&newDependency, processedLocators); err != nil {
			return err
		}

		addonConfig.Dependencies = append(addonConfig.Dependencies, newDependency)
	}

	return nil
}

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
func (infoSvc *CloudInfoService) DeployAddonToProject(addonConfig *AddonConfig, projectConfig *ProjectsConfig) (*DeployedAddonsDetails, error) {
	// Initialize a map to track processed version locators and prevent circular dependencies
	processedLocators := make(map[string]bool)

	// Process the main addon and all its dependencies recursively
	if err := infoSvc.processComponentReferences(addonConfig, processedLocators); err != nil {
		return nil, err
	}

	// Create the request body
	addonDependencies := make([]map[string]string, 0)

	// Add the addon itself
	addonConfig.ConfigName = fmt.Sprintf("%s-%s", addonConfig.Prefix, addonConfig.OfferingName)
	addonDependencies = append(addonDependencies, map[string]string{
		"version_locator": addonConfig.VersionLocator,
		"name":            addonConfig.ConfigName,
	})

	// Collect all dependencies from the tree into a flat list
	flattenedDependencies := flattenDependencies(addonConfig)

	// Add all flattened dependencies
	for i, dep := range flattenedDependencies {
		if dep.Enabled {
			randomPostfix := strings.ToLower(random.UniqueId())
			flattenedDependencies[i].ConfigName = fmt.Sprintf("%s-%s", dep.OfferingName, randomPostfix)
			dependencyEntry := map[string]string{
				"version_locator": dep.VersionLocator,
				"name":            flattenedDependencies[i].ConfigName,
			}
			if dep.ExistingConfigID != "" {
				dependencyEntry["config_id"] = dep.ExistingConfigID
			}
			addonDependencies = append(addonDependencies, dependencyEntry)
		}
	}

	// Convert the addonDependencies to JSON
	jsonBody, err := json.Marshal(addonDependencies)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request body: %w", err)
	}

	// print the json body pretty
	var prettyJSON bytes.Buffer

	err = json.Indent(&prettyJSON, jsonBody, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error pretty printing json: %w", err)
	}

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
	startTime := time.Now()
	resp, err := client.Do(req)
	requestTime := time.Since(startTime)
	infoSvc.Logger.ShortInfo(fmt.Sprintf("Request completed in %v\n", requestTime))
	if err != nil {
		return nil, fmt.Errorf("error executing request: %w", err)
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			if closeErr := resp.Body.Close(); closeErr != nil {
				infoSvc.Logger.ShortInfo(fmt.Sprintf("Error closing response body: %v", closeErr))
			}
		}
	}()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var deployResponse *DeployedAddonsDetails
	if err := json.Unmarshal(body, &deployResponse); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	// If no configs were returned, return nil
	if len(deployResponse.Configs) == 0 {
		return nil, nil
	}

	// Update configuration information for main addon and all its dependencies
	updateConfigInfoFromResponse(addonConfig, flattenedDependencies, deployResponse)

	return deployResponse, nil
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
	defer func() {
		if resp != nil && resp.Body != nil {
			if closeErr := resp.Body.Close(); closeErr != nil {
				infoSvc.Logger.ShortInfo(fmt.Sprintf("Error closing response body: %v", closeErr))
			}
		}
	}()

	// Check if the response is nil
	if resp == nil {
		return nil, fmt.Errorf("response is nil")
	}

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

// flattenDependencies collects all dependencies from an addon config and its nested dependencies
// into a flat list, avoiding duplicates by tracking version locators
func flattenDependencies(addonConfig *AddonConfig) []AddonConfig {
	// Use a map to track unique dependencies by version locator
	uniqueDeps := make(map[string]AddonConfig)

	// Helper function to recursively collect dependencies
	var collectDependencies func(addon *AddonConfig)
	collectDependencies = func(addon *AddonConfig) {
		for _, dep := range addon.Dependencies {
			// Only add this dependency if we haven't seen it before
			if _, exists := uniqueDeps[dep.VersionLocator]; !exists {
				uniqueDeps[dep.VersionLocator] = dep

				// Recursively collect dependencies of this dependency
				collectDependencies(&dep)
			}
		}
	}

	// Start the collection process
	collectDependencies(addonConfig)

	// Convert the map to a slice
	result := make([]AddonConfig, 0, len(uniqueDeps))
	for _, dep := range uniqueDeps {
		result = append(result, dep)
	}

	return result
}

// updateConfigInfoFromResponse processes the deployment response and updates
// the configuration information for the main addon and its dependencies
func updateConfigInfoFromResponse(addonConfig *AddonConfig, dependencies []AddonConfig, response *DeployedAddonsDetails) {
	// Create a map for easier lookup by config name
	configMap := make(map[string]string)
	containerMap := make(map[string]string)

	for _, config := range response.Configs {
		// Check if this is a container config (name ends with " Container")
		isContainer := strings.HasSuffix(config.Name, " Container")

		if isContainer {
			// For container configs, extract the base name (without " Container")
			baseName := strings.TrimSuffix(config.Name, " Container")
			containerMap[baseName] = config.ConfigID
		} else {
			// For regular configs
			configMap[config.Name] = config.ConfigID
		}
	}

	// Update the main addon config
	if configID, exists := configMap[addonConfig.ConfigName]; exists {
		addonConfig.ConfigID = configID
	}

	// Update the main addon's container config
	if containerID, exists := containerMap[addonConfig.ConfigName]; exists {
		addonConfig.ContainerConfigID = containerID
		addonConfig.ContainerConfigName = addonConfig.ConfigName + " Container"
	}

	// Update all dependencies
	for i, dep := range dependencies {
		if configID, exists := configMap[dep.ConfigName]; exists {
			dependencies[i].ConfigID = configID
		}

		if containerID, exists := containerMap[dep.ConfigName]; exists {
			dependencies[i].ContainerConfigID = containerID
			dependencies[i].ContainerConfigName = dep.ConfigName + " Container"
		}
	}
}

// GetOffering gets the details of an Offering from a specified Catalog
func (infoSvc *CloudInfoService) GetOffering(catalogID string, offeringID string) (result *catalogmanagementv1.Offering, response *core.DetailedResponse, err error) {

	options := &catalogmanagementv1.GetOfferingOptions{
		CatalogIdentifier: &catalogID,
		OfferingID:        &offeringID,
	}

	offering, response, err := infoSvc.catalogService.GetOffering(options)
	if err != nil {
		return nil, nil, fmt.Errorf("error getting offering: %w", err)
	}

	// Check if the response status code is not 200
	if response.StatusCode != 200 {
		return nil, nil, fmt.Errorf("failed to get offering: %s", response.RawResult)
	}

	return offering, response, err
}

func (infoSvc *CloudInfoService) GetOfferingInputs(offering *catalogmanagementv1.Offering, VersionID string, OfferingID string) []CatalogInput {
	for _, version := range offering.Kinds[0].Versions {
		if version.ID != nil && *version.ID == VersionID {
			inputs := []CatalogInput{}
			for _, configuration := range version.Configuration {
				input := CatalogInput{
					Key:          *configuration.Key,
					Type:         *configuration.Type,
					DefaultValue: configuration.DefaultValue,
					Required:     *configuration.Required,
					Description:  *configuration.Description,
				}
				inputs = append(inputs, input)
			}
			return inputs
		}
	}

	if offering.ID != nil {
		infoSvc.Logger.ShortInfo(fmt.Sprintf("Error, version not found for offering: %s", *offering.ID))
	} else {
		infoSvc.Logger.ShortInfo("Error, version not found for offering with nil ID")
	}
	return nil
}

// This function is going to return the Version Locator of the dependency which will be further used
// in the buildDependencyGraph function to build the expected graph
// Here version could a pinned version like(v1.0.3) , unpinned version like(^v2.1.4 or ~v1.5.6)
// range based matching is also supported >=v1.1.2,<=v4.3.1 or <=v3.1.4,>=v1.1.0
// It uses MatchVersion function in common package to find the suitable version available in case it is not pinned
func (infoSvc *CloudInfoService) GetOfferingVersionLocatorByConstraint(catalogID string, offeringID string, version string, flavor string) (string, string, error) {

	_, response, err := infoSvc.GetOffering(catalogID, offeringID)
	if err != nil {
		return "", "", fmt.Errorf("unable to get the dependency offering %s", err)
	}

	offering, ok := response.Result.(*catalogmanagementv1.Offering)
	versionList := make([]string, 0)
	if ok {

		for _, kind := range offering.Kinds {

			if *kind.InstallKind == "terraform" {

				for _, v := range kind.Versions {

					versionList = append(versionList, *v.Version)
				}
			}
		}
	}

	bestVersion := common.MatchVersion(versionList, version)
	if bestVersion == "" {
		return "", "", fmt.Errorf("could not find a matching version for dependency %s ", *offering.Name)
	}

	versionLocator := ""

	for _, kind := range offering.Kinds {

		if *kind.InstallKind == "terraform" {

			for _, v := range kind.Versions {

				if *v.Version == bestVersion && *v.Flavor.Name == flavor {
					versionLocator = *v.VersionLocator
					break
				}
			}
		}
	}

	return bestVersion, versionLocator, nil

}
