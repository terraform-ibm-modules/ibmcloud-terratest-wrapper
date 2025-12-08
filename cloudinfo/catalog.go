package cloudinfo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/platform-services-go-sdk/catalogmanagementv1"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/common"
)

// GetCatalogVersionByLocator gets a version by its locator using the Catalog Management service
// CACHED: Static catalog metadata - safe to cache as version locator data doesn't change
// NOTE: Cache can be bypassed with BYPASS_CACHE_FOR_VALIDATION=true for critical validation scenarios
func (infoSvc *CloudInfoService) GetCatalogVersionByLocator(versionLocator string) (*catalogmanagementv1.Version, error) {
	// Check cache first if caching is enabled and not bypassed for validation
	if infoSvc.apiCache != nil && !infoSvc.shouldBypassCache() {
		cacheKey := infoSvc.apiCache.generateCatalogVersionKey(versionLocator)

		infoSvc.apiCache.mutex.RLock()
		cached, exists := infoSvc.apiCache.catalogVersionCache[cacheKey]
		infoSvc.apiCache.mutex.RUnlock()

		if exists && !infoSvc.apiCache.isExpired(cached.Timestamp) {
			infoSvc.apiCache.mutex.Lock()
			infoSvc.apiCache.stats.CatalogVersionHits++
			infoSvc.apiCache.mutex.Unlock()

			infoSvc.Logger.ShortInfo(fmt.Sprintf("Cache HIT for catalog version: versionLocator='%s'", versionLocator))
			return cached.Version, cached.Error
		}

		// Cache miss
		infoSvc.apiCache.mutex.Lock()
		infoSvc.apiCache.stats.CatalogVersionMisses++
		infoSvc.apiCache.mutex.Unlock()
	}

	// Use new retry utility for catalog operations
	config := common.CatalogOperationRetryConfig()
	config.Logger = infoSvc.Logger
	config.OperationName = fmt.Sprintf("GetCatalogVersionByLocator '%s'", versionLocator)

	version, err := common.RetryWithConfig(config, func() (*catalogmanagementv1.Version, error) {
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
			return nil, fmt.Errorf("rate limited on GetVersion for versionLocator='%s' (status: %d) - this indicates high API load, retrying with backoff", versionLocator, response.StatusCode)
		} else if response.StatusCode >= 500 {
			return nil, fmt.Errorf("server error on GetVersion for versionLocator='%s' (status: %d)", versionLocator, response.StatusCode)
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
			return nil, fmt.Errorf("failed to get version (status %d): %s", response.StatusCode, response.RawResult)
		}
	})

	// Cache the result if caching is enabled
	if infoSvc.apiCache != nil {
		cacheKey := infoSvc.apiCache.generateCatalogVersionKey(versionLocator)
		cachedResult := &CachedCatalogVersion{
			Version:   version,
			Error:     err,
			Timestamp: time.Now(),
		}

		infoSvc.apiCache.mutex.Lock()
		infoSvc.apiCache.catalogVersionCache[cacheKey] = cachedResult
		infoSvc.apiCache.mutex.Unlock()
	}

	return version, err
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
		if response.StatusCode == 429 {
			return nil, fmt.Errorf("rate limited on CreateCatalog for catalogName='%s' (status: %d) - this indicates high API load, retrying with backoff", catalogName, response.StatusCode)
		} else if response.StatusCode >= 500 {
			return nil, fmt.Errorf("server error on CreateCatalog for catalogName='%s' (status: %d)", catalogName, response.StatusCode)
		}

		// Check if the response status code is 201 (created)
		if response.StatusCode == 201 {
			return catalog, nil
		}

		return nil, fmt.Errorf("failed to create catalog (status %d): %s", response.StatusCode, response.RawResult)
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

	return fmt.Errorf("failed to delete catalog (status %d): %s", response.StatusCode, response.RawResult)
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
		if response.StatusCode == 429 {
			return nil, fmt.Errorf("rate limited on ImportOffering for catalogID='%s', offeringName='%s' (status: %d) - this indicates high API load, retrying with backoff", catalogID, offeringName, response.StatusCode)
		} else if response.StatusCode >= 500 {
			return nil, fmt.Errorf("server error on ImportOffering for catalogID='%s', offeringName='%s' (status: %d)", catalogID, offeringName, response.StatusCode)
		}

		// Check if the response status code is 201 (created)
		if response.StatusCode == 201 {
			return offering, nil
		}

		return nil, fmt.Errorf("failed to import offering (status %d): %s", response.StatusCode, response.RawResult)
	})
}

// PrepareOfferingImport handles the complete workflow of validating repository/branch
// and preparing parameters for offering import. This centralizes the logic for
// branch validation and repository URL conversion that's needed for catalog operations.
//
// Returns:
// - commitUrl: The commit URL ready for catalog import
// - repo: The normalized repository URL
// - branch: The resolved branch name
// - error: Any error that occurred during preparation
func (infoSvc *CloudInfoService) PrepareOfferingImport() (commitUrl, repo, branch string, err error) {
	// Get repository info
	gitRoot, _ := common.GitRootPath(".")
	commitID, _ := common.GetLatestCommitID(gitRoot)
	repoName := filepath.Base(gitRoot)
	repoUrl, branch := common.GetBaseRepoAndBranch(repoName, "")
	doesCommitExistInRemote, err := common.CommitExistsInRemote(repoUrl, commitID)
	if err != nil {
		infoSvc.Logger.ShortWarn("Error getting current branch for offering import validation")
		return "", "", "", fmt.Errorf("failed to get repository info for offering import: %w", err)
	}
	if !doesCommitExistInRemote {
		infoSvc.Logger.ShortError(fmt.Sprintf("Required commit '%s' does not exist in repository '%s'.", commitID, repo))
		infoSvc.Logger.ShortError("Please ensure a PR has been opened against the remote repository before running the test.")
		return "", "", "", fmt.Errorf("failed to validate PR commit exists for offering import: %w", err)
	}

	// Convert repository URL to HTTPS format for branch validation and catalog import
	repo = normalizeRepositoryURL(repoUrl)

	// Resolve actual branch name in CI environments where git returns "HEAD"
	resolvedBranch := resolveCIBranchName(branch)
	if resolvedBranch != branch {
		infoSvc.Logger.ShortInfo(fmt.Sprintf("Resolved CI branch name from '%s' to '%s'", branch, resolvedBranch))
		branch = resolvedBranch
	}

	commitUrl = fmt.Sprintf("%s/commit/%s", repo, commitID)

	return commitUrl, repo, branch, nil
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
	// Collect disabled offerings from the root addon config to respect user disable choices
	disabledOfferings := make(map[string]bool)
	for _, dep := range addonConfig.Dependencies {
		if dep.Enabled != nil && !*dep.Enabled {
			disabledOfferings[dep.OfferingName] = true
		}
	}
	return infoSvc.processComponentReferencesWithGetter(addonConfig, processedLocators, disabledOfferings, infoSvc)
}

// processComponentReferencesWithGetter is the internal implementation that accepts a ComponentReferenceGetter
// This function uses a correct tree-building approach:
// 1. Start with user's direct dependencies in AddonConfig.Dependencies
// 2. Use API flat list to fill in metadata (version locators, IDs) for direct dependencies
// 3. Recursively build sub-trees for each enabled dependency
func (infoSvc *CloudInfoService) processComponentReferencesWithGetter(addonConfig *AddonConfig, processedLocators map[string]bool, disabledOfferings map[string]bool, getter ComponentReferenceGetter) error {
	// If we've already processed this version locator, skip processing to avoid circular dependencies
	if processedLocators[addonConfig.VersionLocator] {
		return nil
	}
	// Mark this locator as processed
	processedLocators[addonConfig.VersionLocator] = true

	// PHASE 1: Get API flat list and fill in metadata for user's direct dependencies
	componentsReferences, err := getter.GetComponentReferences(addonConfig.VersionLocator)
	if err != nil {
		return fmt.Errorf("error getting component references for %s: %w", addonConfig.VersionLocator, err)
	}

	// Build a lookup map of all API results by name (handling multiple flavors)
	apiDependencies := make(map[string][]*OfferingReferenceItem) // name -> list of flavors

	// Add required dependencies
	for _, component := range componentsReferences.Required.OfferingReferences {
		componentCopy := component
		apiDependencies[component.Name] = append(apiDependencies[component.Name], &componentCopy)
	}

	// Add optional dependencies
	for _, component := range componentsReferences.Optional.OfferingReferences {
		componentCopy := component
		apiDependencies[component.Name] = append(apiDependencies[component.Name], &componentCopy)
	}

	// Check if any required dependencies are explicitly disabled - this is an error
	for _, component := range componentsReferences.Required.OfferingReferences {
		for _, dep := range addonConfig.Dependencies {
			if dep.OfferingName == component.Name && dep.Enabled != nil && !*dep.Enabled {
				return fmt.Errorf("required dependency %s cannot be disabled - it is required by %s",
					component.Name, addonConfig.OfferingName)
			}
		}
	}

	// Fill in metadata for each user-defined dependency
	for i := range addonConfig.Dependencies {
		dep := &addonConfig.Dependencies[i]
		depName := dep.OfferingName

		// Find matching API result(s) for this dependency name
		apiResults, exists := apiDependencies[depName]
		if !exists {
			continue
		}

		// Handle multiple flavors - match by flavor if user specified one
		var matchedAPI *OfferingReferenceItem = nil
		if dep.OfferingFlavor != "" {
			// User specified a flavor, find exact match
			for _, apiResult := range apiResults {
				if apiResult.OfferingReference.Flavor.Name == dep.OfferingFlavor {
					matchedAPI = apiResult
					break
				}
			}
		} else {
			// User didn't specify flavor, use first one (or default if available)
			for _, apiResult := range apiResults {
				if apiResult.OfferingReference.DefaultFlavor == "" ||
					apiResult.OfferingReference.DefaultFlavor == apiResult.OfferingReference.Flavor.Name {
					matchedAPI = apiResult
					break
				}
			}
			if matchedAPI == nil && len(apiResults) > 0 {
				matchedAPI = apiResults[0] // Fallback to first one
			}
		}

		if matchedAPI == nil {
			continue
		}

		// Fill in metadata from API
		dep.VersionLocator = matchedAPI.OfferingReference.VersionLocator
		dep.ResolvedVersion = matchedAPI.OfferingReference.Version
		dep.CatalogID = matchedAPI.OfferingReference.CatalogID
		dep.OfferingID = matchedAPI.OfferingReference.ID
		dep.OfferingFlavor = matchedAPI.OfferingReference.Flavor.Name
		dep.OfferingLabel = matchedAPI.OfferingReference.Label
		dep.Prefix = addonConfig.Prefix

		// Set OnByDefault if not already set
		if dep.OnByDefault == nil {
			dep.OnByDefault = core.BoolPtr(matchedAPI.OfferingReference.OnByDefault)
		}

		// Set Enabled if not already set by user (use OnByDefault)
		if dep.Enabled == nil {
			dep.Enabled = core.BoolPtr(matchedAPI.OfferingReference.OnByDefault)
		}

		// Initialize inputs if nil
		if dep.Inputs == nil {
			dep.Inputs = make(map[string]interface{})
		}

		// Check if this is a required dependency
		for _, reqComp := range componentsReferences.Required.OfferingReferences {
			if reqComp.Name == depName && reqComp.OfferingReference.VersionLocator == dep.VersionLocator {
				// Required dependencies must be enabled
				dep.Enabled = core.BoolPtr(true)
				dep.IsRequired = core.BoolPtr(true)
				dep.RequiredBy = []string{addonConfig.OfferingName}
				break
			}
		}

	}

	// PHASE 1.5: Add missing on_by_default dependencies that user didn't configure

	// Create a map of user-configured dependencies for quick lookup
	userConfiguredDeps := make(map[string]bool)
	for _, dep := range addonConfig.Dependencies {
		userConfiguredDeps[dep.OfferingName] = true
	}

	// Query the catalog to get the direct dependencies for this specific addon
	// This replaces the hardcoded list and works programmatically for any addon
	var catalogDirectDependencies map[string]bool
	if addonConfig.VersionLocator != "" {
		version, err := infoSvc.GetCatalogVersionByLocator(addonConfig.VersionLocator)
		if err != nil {
			catalogDirectDependencies = make(map[string]bool) // Empty map as fallback
		} else if version != nil && version.SolutionInfo != nil && version.SolutionInfo.Dependencies != nil {
			catalogDirectDependencies = make(map[string]bool)
			for _, catalogDep := range version.SolutionInfo.Dependencies {
				if catalogDep.Name != nil {
					catalogDirectDependencies[*catalogDep.Name] = true
				}
			}
		} else {
			catalogDirectDependencies = make(map[string]bool) // Empty map as fallback
		}
	} else {
		catalogDirectDependencies = make(map[string]bool) // Empty map as fallback
	}

	for _, component := range componentsReferences.Optional.OfferingReferences {
		if component.OfferingReference.OnByDefault && !userConfiguredDeps[component.Name] {
			// Only process if this is a direct dependency according to the catalog
			if !catalogDirectDependencies[component.Name] {
				continue
			}

			// Check if this dependency is globally disabled
			if disabledOfferings[component.Name] {
				continue
			}

			// This is an on_by_default direct dependency that the user didn't configure

			newDependency := AddonConfig{
				OfferingName:    component.Name,
				OfferingFlavor:  component.OfferingReference.Flavor.Name,
				VersionLocator:  component.OfferingReference.VersionLocator,
				OfferingID:      component.OfferingReference.ID,
				CatalogID:       component.OfferingReference.CatalogID,
				ResolvedVersion: component.OfferingReference.Version,
				Enabled:         core.BoolPtr(true), // on_by_default means enabled by default
				OnByDefault:     core.BoolPtr(true),
				Dependencies:    []AddonConfig{}, // Initialize empty dependencies slice
			}

			addonConfig.Dependencies = append(addonConfig.Dependencies, newDependency)
		}
	}

	for _, component := range componentsReferences.Required.OfferingReferences {
		if !userConfiguredDeps[component.Name] {
			// Only process if this is a direct dependency according to the catalog
			if !catalogDirectDependencies[component.Name] {
				continue
			}

			// Check if this dependency is globally disabled
			if disabledOfferings[component.Name] {
				continue
			}

			// This is a required direct dependency that the user didn't configure

			newDependency := AddonConfig{
				OfferingName:    component.Name,
				OfferingFlavor:  component.OfferingReference.Flavor.Name,
				VersionLocator:  component.OfferingReference.VersionLocator,
				OfferingID:      component.OfferingReference.ID,
				CatalogID:       component.OfferingReference.CatalogID,
				ResolvedVersion: component.OfferingReference.Version,
				Enabled:         core.BoolPtr(true), // on_by_default means enabled by default
				OnByDefault:     core.BoolPtr(true),
				Dependencies:    []AddonConfig{}, // Initialize empty dependencies slice
			}

			addonConfig.Dependencies = append(addonConfig.Dependencies, newDependency)
		}
	}
	// PHASE 2: Recursively build sub-trees for enabled dependencies

	for i := range addonConfig.Dependencies {
		dep := &addonConfig.Dependencies[i]

		if dep.Enabled != nil && *dep.Enabled {
			// This dependency is enabled, get its children
			if dep.VersionLocator != "" {

				// Recursively process this dependency's children
				err := infoSvc.processComponentReferencesWithGetter(dep, processedLocators, disabledOfferings, getter)
				if err != nil {
					return fmt.Errorf("error processing children of %s: %w", dep.OfferingName, err)
				}
			} else {
			}
		} else {
			// Make sure Dependencies is empty for disabled deps
			dep.Dependencies = []AddonConfig{}
		}
	}

	return nil
}

// DeployAddonToProject deploys an addon and its dependencies to a project
// POST /api/v1-beta/deploy/projects/:projectID/container
// NOT CACHED: Deployment operations must always be executed fresh

// This function handles dependency tree hierarchy by ensuring:
// 1. Each offering (version_locator) appears only once in the deployment list
// 2. The topmost instance in the dependency hierarchy takes precedence
// 3. The main addon is always deployed first, followed by dependencies in hierarchy order
// 4. Only enabled dependencies are included in the deployment

// Example request body:
// [

// 	{
// 	    "version_locator": "9212a6da-ac9b-4f3c-94d8-83a866e1a250.cb157ad2-0bf7-488c-bdd4-5c568d611423"
// 	},
// 	{
// 	    "version_locator": "9212a6da-ac9b-4f3c-94d8-83a866e1a250.3a38fa8e-12ba-472b-be07-832fcb1ae914"
// 	},
// 	{
// 	    "version_locator": "9212a6da-ac9b-4f3c-94d8-83a866e1a250.12fa081a-47f1-473c-9acc-70812f66c26b",
// 	    "config_id": "",    // set this if reusing an existing config
// 	    "name": "sm da"
// 	}

// ]

func (infoSvc *CloudInfoService) DeployAddonToProject(addonConfig *AddonConfig, projectConfig *ProjectsConfig) (*DeployedAddonsDetails, error) {
	// Initialize a map to track processed version locators and prevent circular dependencies
	processedLocators := make(map[string]bool)

	// Process the main addon and all its dependencies recursively
	if err := infoSvc.processComponentReferences(addonConfig, processedLocators); err != nil {
		return nil, err
	}

	// Validate that the main addon has a version locator after processing
	if addonConfig.VersionLocator == "" {
		return nil, fmt.Errorf("main addon %s has empty VersionLocator after processing component references", addonConfig.OfferingName)
	}

	// Build a hierarchical deployment list ensuring each offering appears only once
	// The topmost instance in the dependency hierarchy takes precedence
	addonDependencies := buildHierarchicalDeploymentList(addonConfig)

	// Validate each dependency before deployment to catch empty version locators
	for _, dep := range addonDependencies {
		if dep.VersionLocator == "" {
			return nil, fmt.Errorf("dependency %s has empty VersionLocator", dep.OfferingName)
		}
	}

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

		// Read the response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("error reading response body: %w", err)
		}

		// Handle rate limiting (429) with retry
		if resp.StatusCode == 429 {
			return nil, fmt.Errorf("rate limited on DeployAddonToProject for projectID='%s', offeringName='%s' (status: %d) - this indicates high API load, retrying with backoff", projectConfig.ProjectID, addonConfig.OfferingName, resp.StatusCode)
		}

		// Check other error status codes
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			// Handle ISB064E "already exists" as a special case - treat as success
			if resp.StatusCode == 409 && (strings.Contains(string(body), "ISB064E") || strings.Contains(string(body), "already exists in project")) {
				infoSvc.Logger.ShortInfo(fmt.Sprintf("Config already exists in project %s, treating as success", projectConfig.ProjectID))
				// For ISB064E, we need to query the current project state and return that
				// This ensures the caller gets accurate information about existing configs

				// Query the project to get actual deployed configs
				projectConfigs, err := infoSvc.GetProjectConfigs(projectConfig.ProjectID)
				if err != nil {
					// If we can't get project configs, log the error but return an empty valid response
					infoSvc.Logger.ShortWarn(fmt.Sprintf("Could not retrieve project configs after 409: %v", err))
					// Return an empty but valid DeployedAddonsDetails structure
					emptyResponse := &DeployedAddonsDetails{
						ProjectID: projectConfig.ProjectID,
						Configs: []struct {
							Name     string `json:"name"`
							ConfigID string `json:"config_id"`
						}{},
					}
					emptyBody, _ := json.Marshal(emptyResponse)
					return emptyBody, nil
				}

				// Build DeployedAddonsDetails from project configs
				deployedResponse := &DeployedAddonsDetails{
					ProjectID: projectConfig.ProjectID,
					Configs: make([]struct {
						Name     string `json:"name"`
						ConfigID string `json:"config_id"`
					}, 0, len(projectConfigs)),
				}

				for _, config := range projectConfigs {
					if config.ID != nil && config.Definition != nil && config.Definition.Name != nil {
						deployedResponse.Configs = append(deployedResponse.Configs, struct {
							Name     string `json:"name"`
							ConfigID string `json:"config_id"`
						}{
							Name:     *config.Definition.Name,
							ConfigID: *config.ID,
						})
					}
				}

				// Marshal the response to JSON
				responseBody, err := json.Marshal(deployedResponse)
				if err != nil {
					return nil, fmt.Errorf("error marshaling deployed configs response: %w", err)
				}

				return responseBody, nil
			}

			// Debug logging for failed requests - log API URL and request body
			infoSvc.Logger.ShortError(fmt.Sprintf("API request failed - URL: %s", url))
			infoSvc.Logger.ShortError(fmt.Sprintf("Request body: %s", string(jsonBody)))
			return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
		}

		// Only log success message after confirming the request succeeded
		infoSvc.Logger.ShortInfo(fmt.Sprintf("Configuration added to project %s: %s", projectConfig.ProjectName, projectConfig.ProjectID))

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

// GetComponentReferences gets the component references for a version locator returns a flat list of all components(dependencies and sub-dependencies) for the given version locator including the inital component.
// /ui/v1/versions/:version_locator/componentsReferences
// CACHED: Static dependency tree metadata - safe to cache as component references don't change for a version
func (infoSvc *CloudInfoService) GetComponentReferences(versionLocator string) (*OfferingReferenceResponse, error) {
	// Check cache first if caching is enabled
	if infoSvc.apiCache != nil {
		cacheKey := infoSvc.apiCache.generateComponentReferencesKey(versionLocator)

		infoSvc.apiCache.mutex.RLock()
		cached, exists := infoSvc.apiCache.componentReferencesCache[cacheKey]
		infoSvc.apiCache.mutex.RUnlock()

		if exists && !infoSvc.apiCache.isExpired(cached.Timestamp) {
			infoSvc.apiCache.mutex.Lock()
			infoSvc.apiCache.stats.ComponentReferencesHits++
			infoSvc.apiCache.mutex.Unlock()

			infoSvc.Logger.ShortInfo(fmt.Sprintf("Cache HIT for component references: versionLocator='%s'", versionLocator))
			return cached.References, cached.Error
		}

		// Cache miss
		infoSvc.apiCache.mutex.Lock()
		infoSvc.apiCache.stats.ComponentReferencesMisses++
		infoSvc.apiCache.mutex.Unlock()
	}

	// Use new retry utility for catalog operations
	config := common.CatalogOperationRetryConfig()
	config.Logger = infoSvc.Logger
	config.OperationName = fmt.Sprintf("GetComponentReferences for '%s'", versionLocator)
	config.MaxRetries = 10 // More retries for component references

	result, err := common.RetryWithConfig(config, func() (*OfferingReferenceResponse, error) {
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
			return nil, fmt.Errorf("rate limited on GetComponentReferences for versionLocator='%s' (status: %d) - this indicates high API load, retrying with backoff", versionLocator, resp.StatusCode)
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

	// Cache the result if caching is enabled
	if infoSvc.apiCache != nil {
		cacheKey := infoSvc.apiCache.generateComponentReferencesKey(versionLocator)
		cachedResult := &CachedComponentReferences{
			References: result,
			Error:      err,
			Timestamp:  time.Now(),
		}

		infoSvc.apiCache.mutex.Lock()
		infoSvc.apiCache.componentReferencesCache[cacheKey] = cachedResult
		infoSvc.apiCache.mutex.Unlock()
	}

	return result, err
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

	// Build global disabled dependencies set - scan entire tree first
	globallyDisabled := make(map[string]bool)
	buildGlobalDisabledSet(mainAddon, globallyDisabled)

	// Create offering identity key for deduplication based on catalog+offering+flavor
	// This ensures we don't deploy the same offering multiple times even if it appears
	// in different parts of the dependency tree
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
	} else {
	}

	// Update the main addon's container config
	if containerID, exists := containerMap[addonConfig.ConfigName]; exists {
		addonConfig.ContainerConfigID = containerID
		addonConfig.ContainerConfigName = addonConfig.ConfigName + " Container"
	} else {
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
// CACHED: Static catalog metadata - safe to cache as offering details don't change
func (infoSvc *CloudInfoService) GetOffering(catalogID string, offeringID string) (result *catalogmanagementv1.Offering, response *core.DetailedResponse, err error) {

	// Add validation and debugging for empty parameters
	if catalogID == "" {
		return nil, nil, fmt.Errorf("catalogID cannot be empty - this may indicate a race condition or uninitialized catalog in parallel test execution")
	}
	if offeringID == "" {
		return nil, nil, fmt.Errorf("offeringID cannot be empty - this may indicate an uninitialized offering ID")
	}

	// Check cache first if caching is enabled
	if infoSvc.apiCache != nil {
		cacheKey := infoSvc.apiCache.generateOfferingKey(catalogID, offeringID)

		infoSvc.apiCache.mutex.RLock()
		cached, exists := infoSvc.apiCache.offeringCache[cacheKey]
		infoSvc.apiCache.mutex.RUnlock()

		if exists && !infoSvc.apiCache.isExpired(cached.Timestamp) {
			infoSvc.apiCache.mutex.Lock()
			infoSvc.apiCache.stats.OfferingHits++
			infoSvc.apiCache.mutex.Unlock()

			infoSvc.Logger.ShortInfo(fmt.Sprintf("Cache HIT for offering: catalogID='%s', offeringID='%s'", catalogID, offeringID))
			return cached.Offering, cached.Response, cached.Error
		}

		// Cache miss
		infoSvc.apiCache.mutex.Lock()
		infoSvc.apiCache.stats.OfferingMisses++
		infoSvc.apiCache.mutex.Unlock()
	}

	infoSvc.Logger.ShortInfo(fmt.Sprintf("Getting offering details: catalogID='%s', offeringID='%s'", catalogID, offeringID))

	// Use singleflight to prevent duplicate concurrent requests for the same offering
	singleflightKey := fmt.Sprintf("offering:%s:%s", catalogID, offeringID)

	type GetOfferingResult struct {
		offering *catalogmanagementv1.Offering
		response *core.DetailedResponse
	}

	// Wrap the entire retry operation in singleflight to deduplicate concurrent requests
	singleflightResult, err, shared := infoSvc.offeringSingleflight.Do(singleflightKey, func() (interface{}, error) {
		// Use new retry utility for catalog operations
		config := common.CatalogOperationRetryConfig()
		config.Logger = infoSvc.Logger
		config.OperationName = fmt.Sprintf("GetOffering catalogID='%s', offeringID='%s'", catalogID, offeringID)

		return common.RetryWithConfig(config, func() (*GetOfferingResult, error) {
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
			if response.StatusCode == 429 {
				return nil, fmt.Errorf("rate limited on GetOffering for catalogID='%s', offeringID='%s' (status: %d) - this indicates high API load, retrying with backoff", catalogID, offeringID, response.StatusCode)
			} else if response.StatusCode >= 500 {
				return nil, fmt.Errorf("server error on GetOffering for catalogID='%s', offeringID='%s' (status: %d)", catalogID, offeringID, response.StatusCode)
			}

			// Check if the response status code is not 200
			if response.StatusCode != 200 {
				return nil, fmt.Errorf("failed to get offering from catalog '%s' with offering ID '%s' (status: %d): %s", catalogID, offeringID, response.StatusCode, response.RawResult)
			}

			// Success - return the result
			return &GetOfferingResult{offering: offering, response: response}, nil
		})
	})

	if shared {
		infoSvc.Logger.ShortInfo(fmt.Sprintf("Deduplication: sharing GetOffering result for catalogID='%s', offeringID='%s'", catalogID, offeringID))
	}

	if err != nil {
		// Cache the error result if caching is enabled
		if infoSvc.apiCache != nil {
			cacheKey := infoSvc.apiCache.generateOfferingKey(catalogID, offeringID)
			cachedResult := &CachedOffering{
				Error:     err,
				Timestamp: time.Now(),
			}

			infoSvc.apiCache.mutex.Lock()
			infoSvc.apiCache.offeringCache[cacheKey] = cachedResult
			infoSvc.apiCache.mutex.Unlock()
		}
		return nil, nil, err
	}

	// Type assert the result back to our expected type
	resultValue, ok := singleflightResult.(*GetOfferingResult)
	if !ok {
		return nil, nil, fmt.Errorf("internal error: unexpected result type from singleflight")
	}

	// Cache the successful result if caching is enabled
	if infoSvc.apiCache != nil {
		cacheKey := infoSvc.apiCache.generateOfferingKey(catalogID, offeringID)
		cachedResult := &CachedOffering{
			Offering:  resultValue.offering,
			Response:  resultValue.response,
			Timestamp: time.Now(),
		}

		infoSvc.apiCache.mutex.Lock()
		infoSvc.apiCache.offeringCache[cacheKey] = cachedResult
		infoSvc.apiCache.mutex.Unlock()
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
				// Check critical fields that are required for deployment
				if configuration.Key == nil {
					// Safe to skip: A configuration without a key is unusable for deployment.
					// The key is required to identify which input this configuration represents.
					// We continue processing other configurations that might be valid, allowing
					// the deployment to proceed with the valid configurations found.
					if offering.ID != nil {
						infoSvc.Logger.ShortError(fmt.Sprintf("Error: configuration Key is nil for offering %s, version %s", *offering.ID, VersionID))
					} else {
						infoSvc.Logger.ShortError(fmt.Sprintf("Error: configuration Key is nil for offering with nil ID, version %s", VersionID))
					}
					continue
				}

				// Determine configuration type with fallback logic
				var configType string
				if configuration.Type != nil {
					configType = *configuration.Type
				} else {
					// Check for type in custom_config as fallback
					var foundInCustomConfig bool
					if configuration.CustomConfig != nil && configuration.CustomConfig.Type != nil {
						configType = *configuration.CustomConfig.Type
						foundInCustomConfig = true
						infoSvc.Logger.ShortInfo(fmt.Sprintf("Using type from custom_config for key %s: %s", *configuration.Key, configType))
					}

					if !foundInCustomConfig {
						// Default to string as last resort
						configType = "string"
						infoSvc.Logger.ShortWarn(fmt.Sprintf("Warning: no type information found for key %s, defaulting to 'string'", *configuration.Key))
					}
				}

				// Extract type_metadata if present
				var typeMetadata string
				if configuration.TypeMetadata != nil {
					typeMetadata = *configuration.TypeMetadata
				}

				// Handle optional fields with safe defaults
				required := false
				if configuration.Required != nil {
					required = *configuration.Required
				}

				description := ""
				if configuration.Description != nil {
					description = *configuration.Description
				}
				input := CatalogInput{
					Key:          *configuration.Key,
					Type:         configType,
					TypeMetadata: typeMetadata,
					DefaultValue: configuration.DefaultValue,
					Required:     required,
					Description:  description,
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
// CACHED: Static version resolution - safe to cache as version constraints resolve to the same locator
func (infoSvc *CloudInfoService) GetOfferingVersionLocatorByConstraint(catalogID string, offeringID string, version_constraint string, flavor string) (string, string, error) {

	// Add validation and debugging for empty parameters
	if catalogID == "" {
		return "", "", fmt.Errorf("catalogID cannot be empty when getting offering version locator - this may indicate a race condition or uninitialized catalog in parallel test execution")
	}
	if offeringID == "" {
		return "", "", fmt.Errorf("offeringID cannot be empty when getting offering version locator - this may indicate an uninitialized offering ID")
	}

	// Check cache first if caching is enabled
	if infoSvc.apiCache != nil {
		cacheKey := infoSvc.apiCache.generateVersionLocatorKey(catalogID, offeringID, version_constraint, flavor)

		infoSvc.apiCache.mutex.RLock()
		cached, exists := infoSvc.apiCache.versionLocatorCache[cacheKey]
		infoSvc.apiCache.mutex.RUnlock()

		if exists && !infoSvc.apiCache.isExpired(cached.Timestamp) {
			infoSvc.apiCache.mutex.Lock()
			infoSvc.apiCache.stats.VersionLocatorHits++
			infoSvc.apiCache.mutex.Unlock()

			infoSvc.Logger.ShortInfo(fmt.Sprintf("Cache HIT for version locator: catalogID='%s', offeringID='%s', constraint='%s', flavor='%s'", catalogID, offeringID, version_constraint, flavor))
			return cached.Version, cached.Locator, cached.Error
		}

		// Cache miss
		infoSvc.apiCache.mutex.Lock()
		infoSvc.apiCache.stats.VersionLocatorMisses++
		infoSvc.apiCache.mutex.Unlock()
	}

	infoSvc.Logger.ShortInfo(fmt.Sprintf("Getting offering version locator: catalogID='%s', offeringID='%s', constraint='%s', flavor='%s'", catalogID, offeringID, version_constraint, flavor))

	offering, _, err := infoSvc.GetOffering(catalogID, offeringID)
	if err != nil {
		// Cache the error if caching is enabled
		if infoSvc.apiCache != nil {
			cacheKey := infoSvc.apiCache.generateVersionLocatorKey(catalogID, offeringID, version_constraint, flavor)
			cachedResult := &CachedVersionLocator{
				Error:     fmt.Errorf("unable to get the dependency offering with catalogID='%s', offeringID='%s', constraint='%s', flavor='%s': %w", catalogID, offeringID, version_constraint, flavor, err),
				Timestamp: time.Now(),
			}

			infoSvc.apiCache.mutex.Lock()
			infoSvc.apiCache.versionLocatorCache[cacheKey] = cachedResult
			infoSvc.apiCache.mutex.Unlock()
		}
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
		err := fmt.Errorf("could not find a matching version for dependency %s ", *offering.Name)

		// Cache the error if caching is enabled
		if infoSvc.apiCache != nil {
			cacheKey := infoSvc.apiCache.generateVersionLocatorKey(catalogID, offeringID, version_constraint, flavor)
			cachedResult := &CachedVersionLocator{
				Error:     err,
				Timestamp: time.Now(),
			}

			infoSvc.apiCache.mutex.Lock()
			infoSvc.apiCache.versionLocatorCache[cacheKey] = cachedResult
			infoSvc.apiCache.mutex.Unlock()
		}

		return "", "", err
	}

	versionLocator := versionLocatorMap[bestVersion]

	// Cache the successful result if caching is enabled
	if infoSvc.apiCache != nil {
		cacheKey := infoSvc.apiCache.generateVersionLocatorKey(catalogID, offeringID, version_constraint, flavor)
		cachedResult := &CachedVersionLocator{
			Version:   bestVersion,
			Locator:   versionLocator,
			Timestamp: time.Now(),
		}

		infoSvc.apiCache.mutex.Lock()
		infoSvc.apiCache.versionLocatorCache[cacheKey] = cachedResult
		infoSvc.apiCache.mutex.Unlock()
	}

	return bestVersion, versionLocator, nil

}

// buildGlobalDisabledSet recursively scans the dependency tree and identifies all dependencies
// that are explicitly disabled. These should remain disabled even if they appear as enabled
// transitive dependencies elsewhere in the tree.
func buildGlobalDisabledSet(addon *AddonConfig, globallyDisabled map[string]bool) {
	// Check all direct dependencies of this addon
	for _, dep := range addon.Dependencies {
		// If a dependency is explicitly disabled, mark it as globally disabled
		if dep.Enabled != nil && !*dep.Enabled {
			globallyDisabled[dep.OfferingName] = true
		}

		// Recursively check sub-dependencies
		buildGlobalDisabledSet(&dep, globallyDisabled)
	}
}

type SetupCatalogOptions struct {
	CatalogUseExisting bool
	Catalog            *catalogmanagementv1.Catalog
	CatalogName        string
	SharedCatalog      *bool
	CloudInfoService   CloudInfoServiceI
	Logger             common.Logger
	Testing            *testing.T
	PostCreateDelay    *time.Duration
	AddonConfig        AddonConfig
	TestType           string
}

// setupCatalog handles catalog creation or reuse based on configuration
func SetupCatalog(options SetupCatalogOptions) (*catalogmanagementv1.Catalog, error) {
	createCatalog := func() (*catalogmanagementv1.Catalog, error) {
		catalog, err := options.CloudInfoService.CreateCatalog(options.CatalogName)
		if err != nil {
			options.Logger.CriticalError(fmt.Sprintf("Error creating catalog: %v", err))
			options.Testing.Fail()
			return nil, fmt.Errorf("error creating catalog: %w", err)
		}

		// Add post-creation delay for eventual consistency
		if options.PostCreateDelay != nil && *options.PostCreateDelay > 0 {
			options.Logger.ShortInfo(fmt.Sprintf("Waiting %v for catalog to be available...", *options.PostCreateDelay))
			time.Sleep(*options.PostCreateDelay)
		}

		if options.Catalog != nil && options.Catalog.Label != nil && options.Catalog.ID != nil {
			options.Logger.ShortInfo(fmt.Sprintf("Created catalog: %s with ID %s", *options.Catalog.Label, *options.Catalog.ID))
			// Seed root AddonConfig CatalogID immediately after creation
			if options.TestType == "addons" && options.AddonConfig.CatalogID == "" {
				options.AddonConfig.CatalogID = *options.Catalog.ID
				options.Logger.ShortInfo(fmt.Sprintf("Seeded AddonConfig.CatalogID from newly created catalog: %s", options.AddonConfig.CatalogID))
			}
		} else {
			options.Logger.ShortWarn("Created catalog but catalog details are incomplete")
		}
		return catalog, nil
	}

	if !options.CatalogUseExisting {
		// Check if catalog sharing is enabled and if catalog already exists
		if options.Catalog != nil {
			if options.Catalog.Label != nil && options.Catalog.ID != nil {
				options.Logger.ShortInfo(fmt.Sprintf("Using existing catalog: %s with ID %s", *options.Catalog.Label, *options.Catalog.ID))
				// Seed root AddonConfig CatalogID from the shared/existing catalog to avoid later
				// recovery paths that depend on network calls (helps under 429 rate limits)

				if options.TestType == "addons" && options.AddonConfig.CatalogID == "" {
					options.AddonConfig.CatalogID = *options.Catalog.ID
					options.Logger.ShortInfo(fmt.Sprintf("Seeded AddonConfig.CatalogID from existing catalog: %s", options.AddonConfig.CatalogID))
				}
			} else {
				options.Logger.ShortWarn("Using existing catalog but catalog details are incomplete")
			}
			return options.Catalog, nil
		} else if options.SharedCatalog != nil && *options.SharedCatalog {
			// For shared catalogs, only create if no shared catalog exists yet
			// Individual tests with SharedCatalog=true should not create new catalogs
			options.Logger.ShortInfo("SharedCatalog=true but no existing shared catalog available - this may indicate a setup issue")
			options.Logger.ShortInfo("Creating catalog anyway to avoid test failure, but consider using matrix tests for proper catalog sharing")
			return createCatalog()
		} else {
			// Create new catalog only for non-shared usage
			return createCatalog()
		}
	} else {
		options.Logger.ShortInfo("Using existing catalog")
		options.Logger.ShortWarn("Not implemented yet")
		// TODO: lookup the catalog ID no api for this
	}
	return nil, nil
}
