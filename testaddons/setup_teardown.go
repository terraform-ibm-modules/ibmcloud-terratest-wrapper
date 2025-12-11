package testaddons

import (
	"fmt"
	"regexp"
	"strings"

	Core "github.com/IBM/go-sdk-core/v5/core"
	project "github.com/IBM/project-go-sdk/projectv1"
	"github.com/stretchr/testify/assert"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/cloudinfo"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/common"
)

// TestSetup performs the initial setup for addon tests
func (options *TestAddonOptions) TestSetup() error {
	setupErr := options.testSetup()
	if !assert.NoError(options.Testing, setupErr) {
		options.Logger.CriticalError(fmt.Sprintf("test setup has failed: %v", setupErr))
		options.Testing.Fail()
		return fmt.Errorf("test setup has failed: %w", setupErr)
	}
	return nil
}

// testSetup performs required steps for new test
func (options *TestAddonOptions) testSetup() error {
	// setup logger
	if options.Logger == nil {
		options.Logger = common.CreateSmartAutoBufferingLogger(options.Testing.Name(), false)
	}

	// Set logger prefix - test context already provides test identification
	options.Logger.SetPrefix("")

	options.Logger.EnableDateTime(false)

	// change relative paths of configuration files to full path based on git root
	repoRoot, repoErr := common.GitRootPath(".")
	if repoErr != nil {
		repoRoot = "."
	}

	isChanges, files, err := common.ChangesToBePush(options.Testing, repoRoot)
	if err != nil {
		options.Logger.CriticalError(fmt.Sprintf("Error checking for local changes in the repository: %v", err))
		options.Testing.Fail()
		return fmt.Errorf("error checking for local changes in the repository: %w", err)
	}

	// remove ignored files
	if len(options.LocalChangesIgnorePattern) > 0 {
		filteredFiles := make([]string, 0)
		for _, file := range files {
			shouldKeep := true

			// Special case: always keep ibm_catalog.json files regardless of ignore patterns
			if strings.HasSuffix(file, "ibm_catalog.json") {
				filteredFiles = append(filteredFiles, file)
				continue
			}

			// ignore files are regex patterns
			for _, ignorePattern := range options.LocalChangesIgnorePattern {
				matched, err := regexp.MatchString(ignorePattern, file)
				if err != nil {
					options.Logger.ShortWarn(fmt.Sprintf("Error matching pattern %s: %v", ignorePattern, err))
					continue
				}
				if matched {
					shouldKeep = false
					break
				}
			}
			if shouldKeep {
				filteredFiles = append(filteredFiles, file)
			}
		}
		files = filteredFiles
		if len(files) == 0 {
			isChanges = false
		}
	}

	if isChanges {
		if !options.SkipLocalChangeCheck {
			// Filter out files with no diff content (applies to all environments)
			filteredFiles := make([]string, 0)
			for _, file := range files {
				if diff, err := common.GetFileDiff(repoRoot, file); err == nil && strings.TrimSpace(diff) != "" {
					filteredFiles = append(filteredFiles, file)
				}
			}
			files = filteredFiles
			if len(files) == 0 {
				isChanges = false
			}

			// Only proceed with error if there are still files with actual changes
			if isChanges {
				filesList := "\nFiles with changes:\n"
				diffDetails := "\nDiff details:\n"

				for _, file := range files {
					filesList += fmt.Sprintf("  %s\n", file)

					// Get diff output for this file
					if diff, err := common.GetFileDiff(repoRoot, file); err != nil {
						diffDetails += fmt.Sprintf("=== %s ===\nError getting diff: %v\n\n", file, err)
					} else if strings.TrimSpace(diff) != "" {
						diffDetails += fmt.Sprintf("=== %s ===\n%s\n", file, diff)
					} else {
						diffDetails += fmt.Sprintf("=== %s ===\n(No diff output - file may be untracked or binary)\n\n", file)
					}
				}

				options.Logger.CriticalError(fmt.Sprintf("Local changes found in the repository, please commit, push or stash the changes before running the test%s%s", filesList, diffDetails))
				options.Testing.Fail()
				return fmt.Errorf("local changes found in the repository")
			}
		} else {
			options.Logger.ShortWarn("Local changes found in the repository, but skipping the check")
			options.Logger.ShortWarn("Files with changes:")
			for _, file := range files {
				options.Logger.ShortWarn(fmt.Sprintf("  %s", file))
			}
		}
	}

	// create new CloudInfoService if not supplied
	if options.CloudInfoService == nil {
		cacheEnabled := true
		if options.CacheEnabled != nil {
			cacheEnabled = *options.CacheEnabled
		}
		cloudInfoSvc, err := cloudinfo.NewCloudInfoServiceFromEnv("TF_VAR_ibmcloud_api_key", cloudinfo.CloudInfoServiceOptions{
			Logger:       options.Logger,
			CacheEnabled: cacheEnabled,
			CacheTTL:     options.CacheTTL,
		})
		if err != nil {
			return err
		}
		options.CloudInfoService = cloudInfoSvc
	}

	// get current branch and repo url and validate branch exists for offering import
	// Use the cloudinfo helper to prepare offering import (validates branch exists)
	branchUrl, repo, branch, err := options.CloudInfoService.PrepareOfferingImport()
	if err != nil {
		options.Logger.CriticalError(fmt.Sprintf("Error preparing offering import: %v", err))
		options.Testing.Fail()
		return fmt.Errorf("error preparing offering import: %w", err)
	}

	options.currentBranch = &branch

	options.Logger.ShortInfo("Checking for local changes in the repository")

	options.currentBranchUrl = Core.StringPtr(branchUrl)
	options.Logger.ShortInfo(fmt.Sprintf("Current branch: %s", branch))
	options.Logger.ShortInfo(fmt.Sprintf("Current repo: %s", repo))
	options.Logger.ShortInfo(fmt.Sprintf("Current branch URL: %s", *options.currentBranchUrl))

	catalog, err := cloudinfo.SetupCatalog(cloudinfo.SetupCatalogOptions{
		CatalogUseExisting: options.CatalogUseExisting,
		Catalog:            options.catalog,
		CatalogName:        options.CatalogName,
		SharedCatalog:      options.SharedCatalog,
		CloudInfoService:   options.CloudInfoService,
		Logger:             options.Logger,
		Testing:            options.Testing,
		PostCreateDelay:    options.PostCreateDelay,
		IsAddonTest:        false,
	})

	if err != nil {
		return err
	} else {
		options.catalog = catalog
	}
	if err := options.setupOffering(); err != nil {
		return err
	}

	project, projectConfig, err := cloudinfo.SetupProject(cloudinfo.SetupProjectOptions{
		CurrentProject:           options.currentProject,
		CurrentProjectConfig:     options.currentProjectConfig,
		ProjectDestroyOnDelete:   options.ProjectDestroyOnDelete,
		ProjectAutoDeploy:        options.ProjectAutoDeploy,
		ProjectAutoDeployMode:    options.ProjectAutoDeployMode,
		ProjectMonitoringEnabled: options.ProjectMonitoringEnabled,
		ProjectEnvironments:      options.ProjectEnvironments,
		ProjectName:              options.ProjectName,
		ProjectDescription:       options.ProjectDescription,
		ProjectRetryConfig:       options.ProjectRetryConfig,
		ResourceGroup:            options.ResourceGroup,
		QuietMode:                options.QuietMode,
		PostCreateDelay:          options.PostCreateDelay,
		CloudInfoService:         options.CloudInfoService,
		Logger:                   options.Logger,
		Testing:                  options.Testing,
	})
	if err != nil {
		return err
	}
	options.currentProject = project
	options.currentProjectConfig = projectConfig

	return nil
}

// setupOffering handles offering import based on configuration
func (options *TestAddonOptions) setupOffering() error {
	// import the offering
	// ensure install kind is set or return an error
	if !options.AddonConfig.OfferingInstallKind.Valid() {
		options.Logger.ErrorWithContext(fmt.Sprintf("'%s' is not valid for OfferingInstallKind", options.AddonConfig.OfferingInstallKind.String()))
		options.Testing.Fail()
		return fmt.Errorf("'%s' is not valid for OfferingInstallKind", options.AddonConfig.OfferingInstallKind.String())
	}
	// check offering name set or fail
	if options.AddonConfig.OfferingName == "" {
		options.Logger.ErrorWithContext("AddonConfig.OfferingName is not set")
		options.Testing.Fail()
		return fmt.Errorf("AddonConfig.OfferingName is not set")
	}
	// Import the offering - check sharing settings
	if options.SharedCatalog != nil && *options.SharedCatalog && options.offering != nil &&
		options.offering.Label != nil && options.offering.ID != nil && options.offering.Name != nil {
		options.Logger.ShortInfo(fmt.Sprintf("Using existing shared offering: %s with ID %s", *options.offering.Label, *options.offering.ID))

		// Set offering details for addon config from existing offering
		newVersionLocator := ""
		if options.offering.Kinds != nil && len(options.offering.Kinds) > 0 &&
			len(options.offering.Kinds[0].Versions) > 0 {
			newVersionLocator = *options.offering.Kinds[0].Versions[0].VersionLocator
		}
		options.AddonConfig.OfferingName = *options.offering.Name
		options.AddonConfig.OfferingID = *options.offering.ID
		options.AddonConfig.VersionLocator = newVersionLocator
		options.AddonConfig.OfferingLabel = *options.offering.Label

		// Set the resolved version from the existing offering
		if options.offering.Kinds != nil && len(options.offering.Kinds) > 0 &&
			len(options.offering.Kinds[0].Versions) > 0 &&
			options.offering.Kinds[0].Versions[0].Version != nil {
			options.AddonConfig.ResolvedVersion = *options.offering.Kinds[0].Versions[0].Version
		}

		options.Logger.ShortInfo(fmt.Sprintf("Using shared offering Version Locator: %s", options.AddonConfig.VersionLocator))
	} else if options.SharedCatalog != nil && *options.SharedCatalog && options.offering != nil {
		// Shared offering is incomplete - log warning and fall back to creating new offering
		options.Logger.ShortWarn("Shared offering is nil or incomplete - offering import may have failed")
	} else {
		// Create new offering if sharing is disabled or no existing offering
		version := fmt.Sprintf("v0.0.1-dev-%s", options.Prefix)
		options.AddonConfig.ResolvedVersion = version
		options.Logger.ShortInfo(fmt.Sprintf("Importing the offering flavor: %s from branch: %s as version: %s", options.AddonConfig.OfferingFlavor, *options.currentBranchUrl, version))
		offering, err := options.CloudInfoService.ImportOffering(*options.catalog.ID, *options.currentBranchUrl, options.AddonConfig.OfferingName, options.AddonConfig.OfferingFlavor, version, options.AddonConfig.OfferingInstallKind)
		if err != nil {
			options.Logger.CriticalError(fmt.Sprintf("Error importing the offering: %v", err))
			options.Testing.Fail()
			return fmt.Errorf("error importing the offering: %w", err)
		}
		options.offering = offering
		options.Logger.ShortInfo(fmt.Sprintf("Imported flavor: %s with version: %s to %s", *options.offering.Label, version, *options.catalog.Label))

		// Set offering details for addon config
		newVersionLocator := ""
		if options.offering.Kinds != nil {
			newVersionLocator = *options.offering.Kinds[0].Versions[0].VersionLocator
		}
		options.AddonConfig.OfferingName = *options.offering.Name
		options.AddonConfig.OfferingID = *options.offering.ID
		options.AddonConfig.VersionLocator = newVersionLocator
		options.AddonConfig.OfferingLabel = *options.offering.Label

		options.Logger.ShortInfo(fmt.Sprintf("Offering Version Locator: %s", options.AddonConfig.VersionLocator))
	}
	return nil
}

// Function to determine if test resources should be destroyed
//
// Conditions for teardown:
// - The `SkipUndeploy` option is false (if true will override everything else)
// - Test failed and DO_NOT_DESTROY_ON_FAILURE was not set or false
// - Test completed with success (and `SkipUndeploy` was false)
func (options *TestAddonOptions) executeResourceTearDown() bool {

	// assume we will execute
	execute := true

	// if skipundeploy is true, short circuit we are done
	if options.SkipUndeploy {
		options.Logger.ShortInfo("SkipUndeploy is set")
		execute = false
	}

	if options.Testing.Failed() && common.DoNotDestroyOnFailure() {
		options.Logger.ShortInfo("DO_NOT_DESTROY_ON_FAILURE is set")
		options.Logger.ShortInfo(fmt.Sprintf("Test Passed: %t", !options.Testing.Failed()))
		execute = false
	}

	// if test failed and we are not executing, add a log line stating this
	if options.Testing.Failed() && !execute {
		options.Logger.ShortError("Terratest failed. Debug the Test and delete resources manually.")
	}

	if execute {
		options.Logger.ShortInfo("Executing resource teardown")

	}
	return execute
}

// Function to determine if the project or stack steps (and their schematics workspaces) should be destroyed
//
// Conditions for teardown:
// - Test completed with success and `SkipProjectDelete` is false
func (options *TestAddonOptions) executeProjectTearDown() bool {

	// assume we will execute
	execute := true

	// if SkipProjectDelete then short circuit we are done
	if options.SkipProjectDelete {
		execute = false
	}

	if options.Testing.Failed() {
		execute = false
	}
	// skip teardown if no project was created
	if options.currentProject == nil {
		execute = false
	}

	// if test failed and we are not executing, add a log line stating this
	if options.Testing.Failed() && !execute {
		if options.currentProject == nil {
			options.Logger.ShortError("Terratest failed. No project to delete.")
		} else {
			options.Logger.ShortError("Terratest failed. Debug the Test and delete the project manually.")
		}
	}

	return execute
}

// TestTearDown performs cleanup after addon tests
func (options *TestAddonOptions) TestTearDown() {
	if !options.SkipTestTearDown {
		// if we are not skipping the test teardown, execute it
		options.testTearDown()
	}
}

// testTearDown performs the test teardown
func (options *TestAddonOptions) testTearDown() {
	// Flush buffered logs if test failed to show debug information during cleanup
	options.Logger.FlushOnFailure()

	// perform the test teardown
	options.Logger.ShortInfo("Performing test teardown")

	// Always show project URL if project exists, regardless of teardown path
	if options.currentProject != nil && options.currentProject.ID != nil {
		projectURL := fmt.Sprintf("https://cloud.ibm.com/projects/%s/configurations", *options.currentProject.ID)
		options.Logger.ShortInfo(fmt.Sprintf("Project URL: %s", projectURL))
	}

	if options.executeResourceTearDown() {
		err := options.RunPreUndeployHook()
		if err != nil {
			options.Logger.ShortWarn(fmt.Sprintf("Pre Undeploy hook failed: %s", err))
		}

		err = options.Undeploy()
		if err != nil {
			options.Logger.ShortWarn(fmt.Sprintf("Undeploy resources failed: %s", err))
			postHookErr := options.RunPostUndeployHook()
			if postHookErr != nil {
				options.Logger.ShortWarn(fmt.Sprintf("Post Undeploy hook failed: %s", postHookErr))
			}
		}
	}

	if options.executeProjectTearDown() {
		// Project cleanup logic: always clean up projects since we're not sharing them
		if options.currentProject != nil && options.currentProject.ID != nil {
			options.Logger.ShortInfo(fmt.Sprintf("Deleting the project %s with ID %s", options.ProjectName, *options.currentProject.ID))

			// Delete project with retry logic to handle transient database errors
			retryConfig := common.ProjectOperationRetryConfig()
			if options.ProjectRetryConfig != nil {
				retryConfig = *options.ProjectRetryConfig
			}
			retryConfig.Logger = options.Logger
			retryConfig.OperationName = "project deletion"

			_, err := common.RetryWithConfig(retryConfig, func() (*project.ProjectDeleteResponse, error) {
				result, resp, err := options.CloudInfoService.DeleteProject(*options.currentProject.ID)
				if err != nil {
					options.Logger.ShortWarn(fmt.Sprintf("Project deletion attempt failed: %v (will retry if retryable)", err))

					// Check if project was actually deleted despite the error
					if common.StringContainsIgnoreCase(err.Error(), "not found") || common.StringContainsIgnoreCase(err.Error(), "does not exist") {
						options.Logger.ShortInfo("Project deletion returned 'not found' error - this indicates the project was successfully deleted on a previous attempt")

						// The "not found" error means the deletion succeeded - the project doesn't exist
						// This is the desired end state for deletion
						if resp != nil && resp.StatusCode == 404 { // 404 Not Found
							options.Logger.ShortInfo("Treating 'not found' response as successful project deletion")
							return &project.ProjectDeleteResponse{}, nil
						}

						// Even without a 404 response, "not found" in error message indicates successful deletion
						options.Logger.ShortInfo("Project deleted successfully despite API error response")
						return &project.ProjectDeleteResponse{}, nil
					}

					return nil, err
				}

				// Check for successful deletion (HTTP 202)
				if resp.StatusCode != 202 {
					options.Logger.ShortWarn(fmt.Sprintf("Project deletion returned unexpected status code: %d", resp.StatusCode))
					return nil, fmt.Errorf("unexpected response code: %d", resp.StatusCode)
				}

				return result, nil
			})

			if assert.NoError(options.Testing, err) {
				options.Logger.ShortInfo(fmt.Sprintf("Deleted Test Project: %s", options.currentProjectConfig.ProjectName))
			} else {
				errorMsg := fmt.Sprintf("Project deletion failed: %v", err)
				options.lastTeardownErrors = append(options.lastTeardownErrors, errorMsg)
				projectURL := fmt.Sprintf("https://cloud.ibm.com/projects/%s/configurations", *options.currentProject.ID)
				options.Logger.ShortWarn(fmt.Sprintf("Error deleting Test Project: %s\nProject Console: %s", err, projectURL))
			}
		} else {
			options.Logger.ShortInfo("No project ID found to delete")
		}
	}

	// Catalog cleanup logic:
	// - Individual tests with SharedCatalog=false: clean up their own catalogs
	// - Individual tests with SharedCatalog=true: keep catalog for potential reuse
	if options.catalog != nil && (options.SharedCatalog == nil || !*options.SharedCatalog) {
		options.Logger.ShortInfo(fmt.Sprintf("Deleting the catalog %s with ID %s (SharedCatalog=false)", *options.catalog.Label, *options.catalog.ID))
		err := options.CloudInfoService.DeleteCatalog(*options.catalog.ID)
		if err != nil {
			errorMsg := fmt.Sprintf("Catalog deletion failed: %v", err)
			options.lastTeardownErrors = append(options.lastTeardownErrors, errorMsg)
			options.Logger.ErrorWithContext(fmt.Sprintf("Error deleting the catalog: %v", err))
			options.Testing.Fail()
		} else {
			options.Logger.ShortInfo(fmt.Sprintf("Deleted the catalog %s with ID %s", *options.catalog.Label, *options.catalog.ID))
		}
	} else {
		if options.SharedCatalog != nil && *options.SharedCatalog {
			options.Logger.ShortInfo("Shared catalog retained for potential reuse (SharedCatalog=true)")
		} else {
			options.Logger.ShortInfo("No catalog to delete")
		}
	}
}
