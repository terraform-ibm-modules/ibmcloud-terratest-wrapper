package cloudinfo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/platform-services-go-sdk/catalogmanagementv1"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/common"
)

// GetCatalogVersionByLocator gets a version by its locator using the Catalog Management service
func (infoSvc *CloudInfoService) GetCatalogVersionByLocator(versionLocator string) (*catalogmanagementv1.Version, error) {
	// Use new retry utility for catalog operations
	config := common.CatalogOperationRetryConfig()
	config.Logger = infoSvc.Logger
	config.OperationName = fmt.Sprintf("GetCatalogVersionByLocator '%s'", versionLocator)

	return common.RetryWithConfig(config, func() (*catalogmanagementv1.Version, error) {
		// Call the GetCatalogVersionByLocator method with the version locator and the context
		getVersionOptions := &catalogmanagementv1.GetVersionOptions{
			VersionLocID: &versionLocator,
		}

		offering, response, err := infoSvc.catalogService.GetVersion(getVersionOptions)
		if err != nil {
			return nil, err
		}

		// Handle rate limiting (429) and other temporary failures
		if response.StatusCode == 429 {
			return nil, fmt.Errorf("rate limited (status: %d)", response.StatusCode)
		} else if response.StatusCode >= 500 {
			return nil, fmt.Errorf("server error (status: %d)", response.StatusCode)
		}

		// Check if the response status code is 200 (success)
		if response.StatusCode == 200 {
			var version catalogmanagementv1.Version
			if len(offering.Kinds) > 0 && len(offering.Kinds[0].Versions) > 0 {
				version = offering.Kinds[0].Versions[0]
				return &version, nil
			}
			return nil, fmt.Errorf("version not found")
		} else {
			return nil, fmt.Errorf("failed to get version: %s", response.RawResult)
		}
	})
}

// CreateCatalog creates a new private catalog using the Catalog Management service
func (infoSvc *CloudInfoService) CreateCatalog(catalogName string) (*catalogmanagementv1.Catalog, error) {
	// Use new retry utility for catalog operations
	config := common.CatalogOperationRetryConfig()
	config.Logger = infoSvc.Logger
	config.OperationName = fmt.Sprintf("CreateCatalog '%s'", catalogName)

	return common.RetryWithConfig(config, func() (*catalogmanagementv1.Catalog, error) {
		// Call the CreateCatalog method with the catalog name and the context
		createCatalogOptions := &catalogmanagementv1.CreateCatalogOptions{
			Label: &catalogName,
		}

		catalog, response, err := infoSvc.catalogService.CreateCatalog(createCatalogOptions)
		if err != nil {
			return nil, err
		}

		// Handle rate limiting (429) and other temporary failures
		if response.StatusCode == 429 || response.StatusCode >= 500 {
			return nil, fmt.Errorf("temporary failure (status: %d)", response.StatusCode)
		}

		// Check if the response status code is 201 (created)
		if response.StatusCode == 201 {
			return catalog, nil
		}

		return nil, fmt.Errorf("failed to create catalog: %s", response.RawResult)
	})
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

	// Use new retry utility for catalog operations
	config := common.CatalogOperationRetryConfig()
	config.Logger = infoSvc.Logger
	config.OperationName = fmt.Sprintf("ImportOffering '%s'", offeringName)

	return common.RetryWithConfig(config, func() (*catalogmanagementv1.Offering, error) {
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

		// Handle rate limiting (429) and other temporary failures
		if response.StatusCode == 429 || response.StatusCode >= 500 {
			return nil, fmt.Errorf("temporary failure (status: %d)", response.StatusCode)
		}

		// Check if the response status code is 201 (created)
		if response.StatusCode == 201 {
			return offering, nil
		}

		return nil, fmt.Errorf("failed to import offering: %s", response.RawResult)
	})
}

// PrepareOfferingImport handles the complete workflow of validating repository/branch
// and preparing parameters for offering import. This centralizes the logic for
// branch validation and repository URL conversion that's needed for catalog operations.
//
// Returns:
// - branchUrl: The formatted branch URL ready for catalog import
// - repo: The normalized repository URL
// - branch: The resolved branch name
// - error: Any error that occurred during preparation
func (infoSvc *CloudInfoService) PrepareOfferingImport() (branchUrl, repo, branch string, err error) {
	// Get repository info
	repo, branch, err = common.GetCurrentPrRepoAndBranch()
	if err != nil {
		infoSvc.Logger.ShortWarn("Error getting current branch and repo for offering import validation")
		return "", "", "", fmt.Errorf("failed to get repository info for offering import: %w", err)
	}

	// Resolve actual branch name in CI environments where git returns "HEAD"
	resolvedBranch := resolveCIBranchName(branch)
	if resolvedBranch != branch {
		infoSvc.Logger.ShortInfo(fmt.Sprintf("Resolved CI branch name from '%s' to '%s'", branch, resolvedBranch))
		branch = resolvedBranch
	}

	// Convert repository URL to HTTPS format for branch validation and catalog import
	repo = normalizeRepositoryURL(repo)

	// Validate that the branch exists in the remote repository (required for offering import)
	// Skip validation only if we're in a detached HEAD state and can't determine the actual branch
	if branch == "HEAD" {
		infoSvc.Logger.ShortInfo("Skipping branch validation as running in detached HEAD mode and unable to resolve actual branch name")
		infoSvc.Logger.ShortInfo("This is common in CI environments - catalog operations will use the commit hash instead")
	} else {
		infoSvc.Logger.ShortInfo(fmt.Sprintf("Validating that branch '%s' exists in remote repository before creating any resources", branch))
		branchExists, err := common.CheckRemoteBranchExists(repo, branch)
		if err != nil {
			infoSvc.Logger.ShortWarn(fmt.Sprintf("Error checking if branch exists in remote repository: %v", err))
			return "", "", "", fmt.Errorf("failed to validate branch exists for offering import: %w", err)
		}
		if !branchExists {
			infoSvc.Logger.ShortError(fmt.Sprintf("Required branch '%s' does not exist in repository '%s'", branch, repo))
			infoSvc.Logger.ShortError("This branch is required for offering import/catalog tests to work properly.")
			infoSvc.Logger.ShortError("Please ensure the branch exists in the remote repository before running the test.")
			return "", "", "", fmt.Errorf("required branch '%s' does not exist in repository '%s' (required for offering import)", branch, repo)
		}
		infoSvc.Logger.ShortInfo(fmt.Sprintf("Branch '%s' confirmed to exist in remote repository", branch))
	}

	// Format the branch URL for catalog import
	// URL encode only the branch name, not the entire URL
	branchUrl = fmt.Sprintf("%s/tree/%s", repo, url.PathEscape(branch))

	return branchUrl, repo, branch, nil
}

// ImportOfferingWithValidation performs the complete offering import workflow including
// branch validation and offering creation. This is the high-level function that most
// tests should use for importing offerings from the current repository.
func (infoSvc *CloudInfoService) ImportOfferingWithValidation(catalogID, offeringName, offeringFlavor, version string, installKind InstallKind) (*catalogmanagementv1.Offering, error) {
	branchUrl, _, _, err := infoSvc.PrepareOfferingImport()
	if err != nil {
		return nil, err
	}

	infoSvc.Logger.ShortInfo(fmt.Sprintf("Importing offering: %s from branch URL: %s as version: %s", offeringFlavor, branchUrl, version))

	offering, err := infoSvc.ImportOffering(
		catalogID,
		branchUrl,
		offeringName,
		offeringFlavor,
		version,
		installKind,
	)
	if err != nil {
		infoSvc.Logger.ShortWarn(fmt.Sprintf("Error importing offering: %v", err))
		return nil, fmt.Errorf("failed to import offering: %w", err)
	}

	if offering != nil && offering.Label != nil && offering.ID != nil {
		infoSvc.Logger.ShortInfo(fmt.Sprintf("Imported offering: %s with ID %s", *offering.Label, *offering.ID))
	} else {
		infoSvc.Logger.ShortWarn("Imported offering but offering details are incomplete")
	}

	return offering, nil
}

// resolveCIBranchName attempts to resolve the actual branch name in CI environments
// where git may return "HEAD" due to detached HEAD state
func resolveCIBranchName(currentBranch string) string {
	// If not in detached HEAD mode, return original branch
	if currentBranch != "HEAD" {
		return currentBranch
	}

	// GitHub Actions
	if githubHeadRef := os.Getenv("GITHUB_HEAD_REF"); githubHeadRef != "" {
		// This is set for pull request events
		return githubHeadRef
	}
	if githubRefName := os.Getenv("GITHUB_REF_NAME"); githubRefName != "" {
		// This is set for push events and other events
		return githubRefName
	}

	// Travis CI
	if travisBranch := os.Getenv("TRAVIS_BRANCH"); travisBranch != "" {
		return travisBranch
	}

	// GitLab CI
	if gitlabBranch := os.Getenv("CI_COMMIT_REF_NAME"); gitlabBranch != "" {
		return gitlabBranch
	}

	// Jenkins
	if jenkinsBranch := os.Getenv("BRANCH_NAME"); jenkinsBranch != "" {
		return jenkinsBranch
	}

	// IBM Cloud Tekton
	if pipelineRunID := os.Getenv("PIPELINE_RUN_ID"); pipelineRunID != "" {
		// Try various environment variables that might contain branch info in Tekton
		if tektonBranch := os.Getenv("BRANCH"); tektonBranch != "" {
			return tektonBranch
		}
		if gitBranch := os.Getenv("GIT_BRANCH"); gitBranch != "" {
			return gitBranch
		}
		if gitRef := os.Getenv("GIT_REF"); gitRef != "" {
			return gitRef
		}
	}

	// Return original branch if no CI environment detected or resolved
	return currentBranch
}

// normalizeRepositoryURL converts various repository URL formats to HTTPS format
// suitable for catalog operations
func normalizeRepositoryURL(repo string) string {
	if strings.HasPrefix(repo, "git@") {
		repo = strings.Replace(repo, ":", "/", 1)
		repo = strings.Replace(repo, "git@", "https://", 1)
		repo = strings.TrimSuffix(repo, ".git")
	} else if strings.HasPrefix(repo, "git://") {
		repo = strings.Replace(repo, "git://", "https://", 1)
		repo = strings.TrimSuffix(repo, ".git")
	} else if strings.HasPrefix(repo, "https://") {
		repo = strings.TrimSuffix(repo, ".git")
	}
	return repo
}

// ComponentReferenceGetter interface for getting component references
type ComponentReferenceGetter interface {
	GetComponentReferences(versionLocator string) (*OfferingReferenceResponse, error)
}

// processComponentReferences recursively processes component references for an addon and its dependencies
// It populates metadata from component references while respecting user-defined settings.
//
// User Override Strategy:
// - Metadata fields (VersionLocator, OfferingID, ResolvedVersion, etc.) are always updated from component references
// - User preference fields are preserved when explicitly set by users:
//   - Enabled: Use pointer to distinguish nil (unset), &false (user disabled), &true (user enabled)
//   - OnByDefault: Use pointer to distinguish nil (unset), &false (user disabled), &true (user enabled)
//   - Inputs: Preserve existing user-defined input map, only initialize if nil
//
// - Required dependencies always have Enabled=true (business rule override)
// - Auto-discovered dependencies get all fields from component references
//
// Examples:
//  1. User sets Dependencies: [{OfferingName: "db", Enabled: &false, Inputs: {"size": "small"}}]
//     → Framework populates VersionLocator, OfferingID but preserves Enabled: false and Inputs
//  2. Framework auto-discovers dependency from component references
//     → Framework sets all fields including Enabled, OnByDefault from component reference
func (infoSvc *CloudInfoService) processComponentReferences(addonConfig *AddonConfig, processedLocators map[string]bool) error {
	return infoSvc.processComponentReferencesWithGetter(addonConfig, processedLocators, infoSvc)
}

// processComponentReferencesWithGetter is the internal implementation that accepts a ComponentReferenceGetter
func (infoSvc *CloudInfoService) processComponentReferencesWithGetter(addonConfig *AddonConfig, processedLocators map[string]bool, getter ComponentReferenceGetter) error {
	// If we've already processed this version locator, skip it to avoid circular dependencies
	if processedLocators[addonConfig.VersionLocator] {
		return nil
	}

	// Mark this locator as processed
	processedLocators[addonConfig.VersionLocator] = true

	// Get component references for this addon
	componentsReferences, err := getter.GetComponentReferences(addonConfig.VersionLocator)
	if err != nil {
		return fmt.Errorf("error getting component references for %s: %w", addonConfig.VersionLocator, err)
	}

	// Update existing dependencies and collect components to add
	var componentsToAdd []OfferingReferenceItem
	processedInThisCall := make(map[string]bool) // Track dependencies processed in this function call

	// Process required references first (they take precedence)
	for _, component := range componentsReferences.Required.OfferingReferences {
		// Skip if this version locator has already been processed in the recursive call tree
		if processedLocators[component.OfferingReference.VersionLocator] {
			continue
		}

		found := false
		for i := range addonConfig.Dependencies {
			if addonConfig.Dependencies[i].OfferingName == component.Name && (component.OfferingReference.DefaultFlavor == "" || component.OfferingReference.DefaultFlavor == component.OfferingReference.Flavor.Name) {
				// Update metadata fields (these should always be populated from component references)
				addonConfig.Dependencies[i].VersionLocator = component.OfferingReference.VersionLocator
				addonConfig.Dependencies[i].ResolvedVersion = component.OfferingReference.Version
				addonConfig.Dependencies[i].CatalogID = component.OfferingReference.CatalogID
				addonConfig.Dependencies[i].OfferingID = component.OfferingReference.ID
				addonConfig.Dependencies[i].Prefix = addonConfig.Prefix
				addonConfig.Dependencies[i].OfferingFlavor = component.OfferingReference.Flavor.Name
				addonConfig.Dependencies[i].OfferingLabel = component.OfferingReference.Label
				// Required components are always enabled (business rule - override user setting for required deps)
				addonConfig.Dependencies[i].Enabled = core.BoolPtr(true)

				// Preserve user-defined inputs - only initialize if nil
				if addonConfig.Dependencies[i].Inputs == nil {
					addonConfig.Dependencies[i].Inputs = make(map[string]interface{})
				}

				found = true
				processedInThisCall[component.Name] = true // Mark as processed

				// OPTIMIZATION: Always process required dependencies recursively, regardless of enabled status
				// Required dependencies override user preferences and must be processed
				if err := infoSvc.processComponentReferencesWithGetter(&addonConfig.Dependencies[i], processedLocators, getter); err != nil {
					return err
				}

				break
			}
		}

		if !found && (component.OfferingReference.DefaultFlavor == "" || component.OfferingReference.DefaultFlavor == component.OfferingReference.Flavor.Name) {
			componentsToAdd = append(componentsToAdd, component)
			processedInThisCall[component.Name] = true // Mark as processed
		}
	}

	// Process optional references
	for _, component := range componentsReferences.Optional.OfferingReferences {
		// Skip if already processed in required references within this call
		if processedInThisCall[component.Name] {
			continue
		}

		// Skip if this version locator has already been processed in the recursive call tree
		if processedLocators[component.OfferingReference.VersionLocator] {
			continue
		}

		found := false
		for i := range addonConfig.Dependencies {
			if addonConfig.Dependencies[i].OfferingName == component.Name && (component.OfferingReference.DefaultFlavor == "" || component.OfferingReference.DefaultFlavor == component.OfferingReference.Flavor.Name) {
				// Update metadata fields (these should always be populated from component references)
				addonConfig.Dependencies[i].VersionLocator = component.OfferingReference.VersionLocator
				addonConfig.Dependencies[i].OfferingID = component.OfferingReference.ID
				addonConfig.Dependencies[i].CatalogID = component.OfferingReference.CatalogID
				addonConfig.Dependencies[i].ResolvedVersion = component.OfferingReference.Version
				addonConfig.Dependencies[i].Prefix = addonConfig.Prefix
				addonConfig.Dependencies[i].OfferingFlavor = component.OfferingReference.Flavor.Name
				addonConfig.Dependencies[i].OfferingLabel = component.OfferingReference.Label
				// Only update OnByDefault if user hasn't explicitly set it (for optional deps)
				if addonConfig.Dependencies[i].OnByDefault == nil {
					addonConfig.Dependencies[i].OnByDefault = core.BoolPtr(component.OfferingReference.OnByDefault)
				}

				// Only update Enabled if user hasn't explicitly set it
				// Note: For optional dependencies, respect user choice; for required, they're forced enabled
				if addonConfig.Dependencies[i].Enabled == nil {
					addonConfig.Dependencies[i].Enabled = core.BoolPtr(component.OfferingReference.OnByDefault)
				}

				// Preserve user-defined inputs - only initialize if nil
				if addonConfig.Dependencies[i].Inputs == nil {
					addonConfig.Dependencies[i].Inputs = make(map[string]interface{})
				}

				found = true

				// OPTIMIZATION: Only process dependencies of enabled optional dependencies
				// This is safe because disabled optional dependencies won't be deployed anyway
				if addonConfig.Dependencies[i].Enabled != nil && *addonConfig.Dependencies[i].Enabled {
					if err := infoSvc.processComponentReferencesWithGetter(&addonConfig.Dependencies[i], processedLocators, getter); err != nil {
						return err
					}
				}

				break
			}
		}

		if !found && (component.OfferingReference.DefaultFlavor == "" || component.OfferingReference.DefaultFlavor == component.OfferingReference.Flavor.Name) && (component.OfferingReference.OnByDefault) {
			// set required to on by default true
			component.OfferingReference.OnByDefault = true
			componentsToAdd = append(componentsToAdd, component)
		}
	}

	// Add new dependencies that weren't found in the existing dependencies
	for _, component := range componentsToAdd {
		onByDefault := component.OfferingReference.OnByDefault
		enabled := component.OfferingReference.OnByDefault // For new components, enabled follows onByDefault

		newDependency := AddonConfig{
			Prefix:          addonConfig.Prefix,
			OfferingName:    component.OfferingReference.Name,
			OfferingLabel:   component.OfferingReference.Label,
			CatalogID:       component.OfferingReference.CatalogID,
			OfferingFlavor:  component.OfferingReference.Flavor.Name,
			VersionLocator:  component.OfferingReference.VersionLocator,
			ResolvedVersion: component.OfferingReference.Version,
			OnByDefault:     &onByDefault,
			Enabled:         &enabled,
			OfferingID:      component.OfferingReference.ID,
			Inputs:          make(map[string]interface{}),
			Dependencies:    []AddonConfig{}, // Initialize empty dependencies slice
		}

		// OPTIMIZATION: Only process dependencies of enabled new dependencies
		// This is safe because disabled dependencies won't be deployed anyway
		if enabled {
			if err := infoSvc.processComponentReferencesWithGetter(&newDependency, processedLocators, getter); err != nil {
				return err
			}
		}

		addonConfig.Dependencies = append(addonConfig.Dependencies, newDependency)
	}

	return nil
}

// DeployAddonToProject deploys an addon and its dependencies to a project
// POST /api/v1-beta/deploy/projects/:projectID/container
//
// This function handles dependency tree hierarchy by ensuring:
// 1. Each offering (version_locator) appears only once in the deployment list
// 2. The topmost instance in the dependency hierarchy takes precedence
// 3. The main addon is always deployed first, followed by dependencies in hierarchy order
// 4. Only enabled dependencies are included in the deployment
//
// Example request body:
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

	// Build a hierarchical deployment list ensuring each offering appears only once
	// The topmost instance in the dependency hierarchy takes precedence
	addonDependencies := buildHierarchicalDeploymentList(addonConfig)

	// Convert each addon config to the deployment format
	deploymentList := make([]map[string]string, 0)
	for _, addon := range addonDependencies {
		dependencyEntry := map[string]string{
			"version_locator": addon.VersionLocator,
			"name":            addon.ConfigName,
		}
		if addon.ExistingConfigID != "" {
			dependencyEntry["config_id"] = addon.ExistingConfigID
		}
		deploymentList = append(deploymentList, dependencyEntry)
	}

	// Convert the addonDependencies to JSON
	jsonBody, err := json.Marshal(deploymentList)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request body: %w", err)
	}

	// print the json body pretty
	var prettyJSON bytes.Buffer

	err = json.Indent(&prettyJSON, jsonBody, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error pretty printing json: %w", err)
	}

	// Deploy with retry logic to handle rate limiting
	url := fmt.Sprintf("https://cm.globalcatalog.cloud.ibm.com/api/v1-beta/deploy/projects/%s/container", projectConfig.ProjectID)

	// Use new retry utility for deployment operation
	config := common.CatalogOperationRetryConfig()
	config.Logger = infoSvc.Logger
	config.OperationName = fmt.Sprintf("DeployAddonToProject for project %s", projectConfig.ProjectID)
	config.MaxRetries = 10 // More retries for deployment

	body, err := common.RetryWithConfig(config, func() ([]byte, error) {
		// Create a new HTTP request with the JSON body
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
		infoSvc.Logger.ShortInfo(fmt.Sprintf("Request completed in %v", requestTime))

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

		// Handle rate limiting (429) with retry
		if resp.StatusCode == 429 {
			return nil, fmt.Errorf("rate limited")
		}

		// Check other error status codes
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
		}

		return body, nil
	})

	if err != nil {
		return nil, err
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
	updateConfigInfoFromResponse(addonConfig, deployResponse)

	return deployResponse, nil
}

// GetComponentReferences gets the component references for a version locator
// /ui/v1/versions/:version_locator/componentsReferences
func (infoSvc *CloudInfoService) GetComponentReferences(versionLocator string) (*OfferingReferenceResponse, error) {
	// Use new retry utility for catalog operations
	config := common.CatalogOperationRetryConfig()
	config.Logger = infoSvc.Logger
	config.OperationName = fmt.Sprintf("GetComponentReferences for '%s'", versionLocator)
	config.MaxRetries = 10 // More retries for component references

	return common.RetryWithConfig(config, func() (*OfferingReferenceResponse, error) {
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

		// Handle rate limiting (429) with retry
		if resp.StatusCode == 429 {
			return nil, fmt.Errorf("rate limited")
		}

		// Check other error status codes
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
		}

		// Unmarshal the response into OfferingReferenceResponse
		var offeringReferences OfferingReferenceResponse
		if err := json.Unmarshal(body, &offeringReferences); err != nil {
			return nil, fmt.Errorf("error unmarshaling offering references: %w", err)
		}

		return &offeringReferences, nil
	})
}

// buildHierarchicalDeploymentList creates a deployment list respecting dependency hierarchy.
// Each offering appears only once, with the topmost instance in the hierarchy taking precedence.
// The main addon is always first in the deployment list, followed by its dependencies.
//
// Hierarchy Rules:
// 1. The main addon (root of the tree) is always deployed first
// 2. If an offering appears multiple times in the dependency tree, only the topmost occurrence is deployed
// 3. Deduplication is based on offering identity (catalog + offering + flavor), not version locator
// 4. Dependencies are processed in depth-first order to respect hierarchy
// 5. Only enabled dependencies are included in the deployment list
//
// Future considerations:
// - This function can be extended to support custom deployment ordering
// - Additional validation can be added for version conflicts between hierarchy levels
func buildHierarchicalDeploymentList(mainAddon *AddonConfig) []AddonConfig {
	deploymentList := make([]AddonConfig, 0)
	processedOfferings := make(map[string]bool) // Track by offering identity instead of version locator

	// Create offering identity key for deduplication
	getOfferingKey := func(addon *AddonConfig) string {
		return fmt.Sprintf("%s|%s|%s", addon.CatalogID, addon.OfferingID, addon.OfferingFlavor)
	}

	// Always add the main addon first with its config name (generate if not set)
	if mainAddon.ConfigName == "" {
		mainAddon.ConfigName = fmt.Sprintf("%s-%s", mainAddon.Prefix, mainAddon.OfferingName)
	}
	deploymentList = append(deploymentList, *mainAddon)
	processedOfferings[getOfferingKey(mainAddon)] = true

	// Recursively process dependencies in hierarchy order
	// This ensures topmost instances take precedence over deeper occurrences
	var processDependencies func(addon *AddonConfig)
	processDependencies = func(addon *AddonConfig) {
		for _, dep := range addon.Dependencies {
			offeringKey := getOfferingKey(&dep)

			// Only process enabled dependencies that haven't been seen before (by offering identity)
			if dep.Enabled != nil && *dep.Enabled && !processedOfferings[offeringKey] {
				// Generate a unique config name for this dependency if not already set
				if dep.ConfigName == "" {
					randomPostfix := strings.ToLower(random.UniqueId())
					dep.ConfigName = fmt.Sprintf("%s-%s", dep.OfferingName, randomPostfix)
				}

				// Add to deployment list and mark as processed
				deploymentList = append(deploymentList, dep)
				processedOfferings[offeringKey] = true

				// Recursively process this dependency's dependencies
				processDependencies(&dep)
			}
		}
	}

	// Start processing from the main addon's dependencies
	processDependencies(mainAddon)

	return deploymentList
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
func updateConfigInfoFromResponse(addonConfig *AddonConfig, response *DeployedAddonsDetails) {
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

	// Recursively update all dependencies in the original structure
	updateDependencyConfigIDs(addonConfig.Dependencies, configMap, containerMap)
}

// updateDependencyConfigIDs recursively updates ConfigIDs for dependencies
func updateDependencyConfigIDs(dependencies []AddonConfig, configMap map[string]string, containerMap map[string]string) {
	for i, dep := range dependencies {
		if configID, exists := configMap[dep.ConfigName]; exists {
			dependencies[i].ConfigID = configID
		}

		if containerID, exists := containerMap[dep.ConfigName]; exists {
			dependencies[i].ContainerConfigID = containerID
			dependencies[i].ContainerConfigName = dep.ConfigName + " Container"
		}

		// Recursively update nested dependencies
		updateDependencyConfigIDs(dependencies[i].Dependencies, configMap, containerMap)
	}
}

// GetOffering gets the details of an Offering from a specified Catalog
func (infoSvc *CloudInfoService) GetOffering(catalogID string, offeringID string) (result *catalogmanagementv1.Offering, response *core.DetailedResponse, err error) {

	// Add validation and debugging for empty parameters
	if catalogID == "" {
		return nil, nil, fmt.Errorf("catalogID cannot be empty - this may indicate a race condition or uninitialized catalog in parallel test execution")
	}
	if offeringID == "" {
		return nil, nil, fmt.Errorf("offeringID cannot be empty - this may indicate an uninitialized offering ID")
	}

	infoSvc.Logger.ShortInfo(fmt.Sprintf("Getting offering details: catalogID='%s', offeringID='%s'", catalogID, offeringID))

	// Use new retry utility for catalog operations
	config := common.CatalogOperationRetryConfig()
	config.Logger = infoSvc.Logger
	config.OperationName = fmt.Sprintf("GetOffering catalogID='%s', offeringID='%s'", catalogID, offeringID)

	type GetOfferingResult struct {
		offering *catalogmanagementv1.Offering
		response *core.DetailedResponse
	}

	resultValue, err := common.RetryWithConfig(config, func() (*GetOfferingResult, error) {
		options := &catalogmanagementv1.GetOfferingOptions{
			CatalogIdentifier: &catalogID,
			OfferingID:        &offeringID,
		}

		offering, response, err := infoSvc.catalogService.GetOffering(options)
		if err != nil {
			// Provide much more detailed error information
			return nil, fmt.Errorf("error getting offering from catalog '%s' with offering ID '%s': %w", catalogID, offeringID, err)
		}

		// Handle rate limiting (429) and other temporary failures
		if response.StatusCode == 429 || response.StatusCode >= 500 {
			return nil, fmt.Errorf("temporary failure (status: %d)", response.StatusCode)
		}

		// Check if the response status code is not 200
		if response.StatusCode != 200 {
			return nil, fmt.Errorf("failed to get offering from catalog '%s' with offering ID '%s' (status: %d): %s", catalogID, offeringID, response.StatusCode, response.RawResult)
		}

		// Success - return the result
		return &GetOfferingResult{offering: offering, response: response}, nil
	})

	if err != nil {
		return nil, nil, err
	}

	return resultValue.offering, resultValue.response, nil
}

func (infoSvc *CloudInfoService) GetOfferingInputs(offering *catalogmanagementv1.Offering, VersionID string, OfferingID string) []CatalogInput {
	// Add null checks to prevent panic
	if offering == nil {
		infoSvc.Logger.ShortInfo("Error, offering is nil")
		return nil
	}

	if len(offering.Kinds) == 0 {
		if offering.ID != nil {
			infoSvc.Logger.ShortInfo(fmt.Sprintf("Error, no kinds found for offering: %s", *offering.ID))
		} else {
			infoSvc.Logger.ShortInfo("Error, no kinds found for offering with nil ID")
		}
		return nil
	}

	if len(offering.Kinds[0].Versions) == 0 {
		if offering.ID != nil {
			infoSvc.Logger.ShortInfo(fmt.Sprintf("Error, no versions found for offering: %s", *offering.ID))
		} else {
			infoSvc.Logger.ShortInfo("Error, no versions found for offering with nil ID")
		}
		return nil
	}

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
// Here version_constraint could a pinned version like(v1.0.3) , unpinned version like(^v2.1.4 or ~v1.5.6)
// range based matching is also supported >=v1.1.2,<=v4.3.1 or <=v3.1.4,>=v1.1.0
// It uses MatchVersion function in common package to find the suitable version available in case it is not pinned
func (infoSvc *CloudInfoService) GetOfferingVersionLocatorByConstraint(catalogID string, offeringID string, version_constraint string, flavor string) (string, string, error) {

	// Add validation and debugging for empty parameters
	if catalogID == "" {
		return "", "", fmt.Errorf("catalogID cannot be empty when getting offering version locator - this may indicate a race condition or uninitialized catalog in parallel test execution")
	}
	if offeringID == "" {
		return "", "", fmt.Errorf("offeringID cannot be empty when getting offering version locator - this may indicate an uninitialized offering ID")
	}

	infoSvc.Logger.ShortInfo(fmt.Sprintf("Getting offering version locator: catalogID='%s', offeringID='%s', constraint='%s', flavor='%s'", catalogID, offeringID, version_constraint, flavor))

	offering, _, err := infoSvc.GetOffering(catalogID, offeringID)
	if err != nil {
		return "", "", fmt.Errorf("unable to get the dependency offering with catalogID='%s', offeringID='%s', constraint='%s', flavor='%s': %w", catalogID, offeringID, version_constraint, flavor, err)
	}

	versionList := make([]string, 0)
	versionLocatorMap := make(map[string]string)
	for _, kind := range offering.Kinds {

		if *kind.InstallKind == "terraform" {

			for _, v := range kind.Versions {

				if *v.Flavor.Name == flavor {
					versionList = append(versionList, *v.Version)
					versionLocatorMap[*v.Version] = *v.VersionLocator
				}
			}
		}
	}

	bestVersion := common.GetLatestVersionByConstraint(versionList, version_constraint)
	if bestVersion == "" {
		return "", "", fmt.Errorf("could not find a matching version for dependency %s ", *offering.Name)
	}

	versionLocator := versionLocatorMap[bestVersion]
	return bestVersion, versionLocator, nil

}
