package testaddons

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/IBM/go-sdk-core/v5/core"
	Core "github.com/IBM/go-sdk-core/v5/core"
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

	// Set logger prefix based on available identifiers (in order of preference)
	if options.TestCaseName != "" {
		// Use TestCaseName for clear test identification (preferred for matrix tests and custom scenarios)
		options.Logger.SetPrefix(fmt.Sprintf("ADDON - %s", options.TestCaseName))
	} else if options.ProjectName != "" {
		// For single tests, include project name in prefix
		options.Logger.SetPrefix(fmt.Sprintf("ADDON - %s", options.ProjectName))
	} else {
		// For tests without project name, use simple prefix
		options.Logger.SetPrefix("ADDON")
	}

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
		cloudInfoSvc, err := cloudinfo.NewCloudInfoServiceFromEnv("TF_VAR_ibmcloud_api_key", cloudinfo.CloudInfoServiceOptions{
			Logger: options.Logger,
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

	if err := options.setupCatalog(); err != nil {
		return err
	}

	if err := options.setupOffering(); err != nil {
		return err
	}

	if err := options.setupProject(); err != nil {
		return err
	}

	return nil
}

// setupCatalog handles catalog creation or reuse based on configuration
func (options *TestAddonOptions) setupCatalog() error {
	if !options.CatalogUseExisting {
		// Check if catalog sharing is enabled and if catalog already exists
		if options.catalog != nil {
			if options.catalog.Label != nil && options.catalog.ID != nil {
				options.Logger.ShortInfo(fmt.Sprintf("Using existing catalog: %s with ID %s", *options.catalog.Label, *options.catalog.ID))
			} else {
				options.Logger.ShortWarn("Using existing catalog but catalog details are incomplete")
			}
		} else if options.SharedCatalog != nil && *options.SharedCatalog {
			// For shared catalogs, only create if no shared catalog exists yet
			// Individual tests with SharedCatalog=true should not create new catalogs
			options.Logger.ShortInfo("SharedCatalog=true but no existing shared catalog available - this may indicate a setup issue")
			options.Logger.ShortInfo("Creating catalog anyway to avoid test failure, but consider using matrix tests for proper catalog sharing")
			catalog, err := options.CloudInfoService.CreateCatalog(options.CatalogName)
			if err != nil {
				options.Logger.CriticalError(fmt.Sprintf("Error creating catalog for shared use: %v", err))
				options.Testing.Fail()
				return fmt.Errorf("error creating catalog for shared use: %w", err)
			}
			options.catalog = catalog
			if options.catalog != nil && options.catalog.Label != nil && options.catalog.ID != nil {
				options.Logger.ShortInfo(fmt.Sprintf("Created catalog for shared use: %s with ID %s", *options.catalog.Label, *options.catalog.ID))
			} else {
				options.Logger.ShortWarn("Created catalog for shared use but catalog details are incomplete")
			}
		} else {
			// Create new catalog only for non-shared usage
			options.Logger.ShortInfo(fmt.Sprintf("Creating a new catalog: %s", options.CatalogName))
			catalog, err := options.CloudInfoService.CreateCatalog(options.CatalogName)
			if err != nil {
				options.Logger.CriticalError(fmt.Sprintf("Error creating a new catalog: %v", err))
				options.Testing.Fail()
				return fmt.Errorf("error creating a new catalog: %w", err)
			}
			options.catalog = catalog
			if options.catalog != nil && options.catalog.Label != nil && options.catalog.ID != nil {
				options.Logger.ShortInfo(fmt.Sprintf("Created a new catalog: %s with ID %s", *options.catalog.Label, *options.catalog.ID))
			} else {
				options.Logger.ShortWarn("Created catalog but catalog details are incomplete")
			}
		}
	} else {
		options.Logger.ShortInfo("Using existing catalog")
		options.Logger.ShortWarn("Not implemented yet")
		// TODO: lookup the catalog ID no api for this
	}
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
	if options.SharedCatalog != nil && *options.SharedCatalog && options.offering != nil {
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

// setupProject handles project creation
func (options *TestAddonOptions) setupProject() error {
	// Create a new project (only if not already created)
	if options.currentProject == nil {
		options.Logger.ShortInfo("Creating project for test")
		if options.ProjectDestroyOnDelete == nil {
			options.ProjectDestroyOnDelete = core.BoolPtr(true)
		}
		if options.ProjectAutoDeploy == nil {
			options.ProjectAutoDeploy = core.BoolPtr(false)
		}
		if options.ProjectMonitoringEnabled == nil {
			options.ProjectMonitoringEnabled = core.BoolPtr(false)
		}
		options.currentProjectConfig = &cloudinfo.ProjectsConfig{
			Location:           options.ProjectLocation,
			ProjectName:        options.ProjectName,
			ProjectDescription: options.ProjectDescription,
			ResourceGroup:      options.ResourceGroup,
			DestroyOnDelete:    *options.ProjectDestroyOnDelete,
			MonitoringEnabled:  *options.ProjectMonitoringEnabled,
			AutoDeploy:         *options.ProjectAutoDeploy,
			Environments:       options.ProjectEnvironments,
		}
		prj, resp, err := options.CloudInfoService.CreateProjectFromConfig(options.currentProjectConfig)
		if err != nil {
			errorMsg := fmt.Sprintf("Error creating a new project: %v\nResponse: %v", err, resp)
			options.Logger.CriticalError(errorMsg)
			options.Testing.Fail()
			return fmt.Errorf("error creating a new project: %w", err)
		}
		options.currentProject = prj
		options.currentProjectConfig.ProjectID = *options.currentProject.ID
		options.Logger.ShortInfo(fmt.Sprintf("Created a new project: %s with ID %s", options.ProjectName, options.currentProjectConfig.ProjectID))
		projectURL := fmt.Sprintf("https://cloud.ibm.com/projects/%s/configurations", options.currentProjectConfig.ProjectID)
		options.Logger.ShortInfo(fmt.Sprintf("Project URL: %s", projectURL))
		region := options.currentProjectConfig.Location
		if region == "" {
			region = "unknown"
		}
		options.Logger.ShortInfo(fmt.Sprintf("Project Region: %s", region))
	} else {
		// Using existing project
		options.Logger.ShortInfo(fmt.Sprintf("Using existing project: %s with ID %s", options.ProjectName, *options.currentProject.ID))
		// Ensure currentProjectConfig is set up properly for existing projects
		if options.currentProjectConfig == nil {
			options.currentProjectConfig = &cloudinfo.ProjectsConfig{
				ProjectID: *options.currentProject.ID,
			}
		}
	}
	return nil
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
	// Show teardown progress in quiet mode
	if options.QuietMode {
		options.Logger.ProgressStage("Cleaning up resources")
	}

	// Flush buffered logs if test failed to show debug information during cleanup
	options.Logger.FlushOnFailure()

	// perform the test teardown
	options.Logger.ShortInfo("Performing test teardown")

	// Project cleanup logic: always clean up projects since we're not sharing them
	if options.currentProject != nil && options.currentProject.ID != nil {
		options.Logger.ShortInfo(fmt.Sprintf("Deleting the project %s with ID %s", options.ProjectName, *options.currentProject.ID))
		_, resp, err := options.CloudInfoService.DeleteProject(*options.currentProject.ID)
		if assert.NoError(options.Testing, err) {
			if assert.Equal(options.Testing, 202, resp.StatusCode) {
				options.Logger.ShortInfo(fmt.Sprintf("Deleted Test Project: %s", options.currentProjectConfig.ProjectName))
			} else {
				options.Logger.ShortWarn(fmt.Sprintf("Failed to delete Test Project, response code: %d", resp.StatusCode))
			}
		} else {
			options.Logger.ShortWarn(fmt.Sprintf("Error deleting Test Project: %s", err))
		}
	} else {
		options.Logger.ShortInfo("No project ID found to delete")
	}

	// Catalog cleanup logic:
	// - Individual tests with SharedCatalog=false: clean up their own catalogs
	// - Individual tests with SharedCatalog=true: keep catalog for potential reuse
	if options.catalog != nil && (options.SharedCatalog == nil || !*options.SharedCatalog) {
		options.Logger.ShortInfo(fmt.Sprintf("Deleting the catalog %s with ID %s (SharedCatalog=false)", *options.catalog.Label, *options.catalog.ID))
		err := options.CloudInfoService.DeleteCatalog(*options.catalog.ID)
		if err != nil {
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
