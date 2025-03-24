package testaddons

import (
	"fmt"
	"regexp"
	"runtime"

	Core "github.com/IBM/go-sdk-core/v5/core"
	"github.com/stretchr/testify/assert"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/cloudinfo"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/common"
)

// RunAddonTest : Run the test for addons
// Creates a new catalog
// Imports an offering
// Creates a new project
// Adds a configuration
// Deploys the configuration
// Deletes the project
// Deletes the catalog
// Returns an error if any of the steps fail
func (options *TestAddonOptions) RunAddonTest() error {
	if !options.SkipTestTearDown {
		// ensure we always run the test tear down, even if a panic occurs
		defer func() {
			if r := recover(); r != nil {

				options.Testing.Fail()
				// Get the file and line number where the panic occurred
				_, file, line, ok := runtime.Caller(4)
				if ok {
					options.Logger.ShortError(fmt.Sprintf("Recovered from panic: %v\nOccurred at: %s:%d\n", r, file, line))
				} else {
					options.Logger.ShortError(fmt.Sprintf("Recovered from panic: %v", r))
				}
			}
			options.TestTearDown()
		}()
	}

	setupErr := options.testSetup()
	if !assert.NoError(options.Testing, setupErr) {
		options.Testing.Fail()
		return fmt.Errorf("test setup has failed:%w", setupErr)
	}

	return nil
}

func (options *TestAddonOptions) TestSetup() error {

	setupErr := options.testSetup()
	if !assert.NoError(options.Testing, setupErr) {
		options.Testing.Fail()
		return fmt.Errorf("test setup has failed:%w", setupErr)
	}

	return nil
}

// Perform required steps for new test
func (options *TestAddonOptions) testSetup() error {

	// setup logger
	if options.Logger == nil {
		options.Logger = common.NewTestLogger(options.Testing.Name())
	}

	if options.ProjectName != "" {
		options.Logger.SetPrefix(fmt.Sprintf("ADDON - %s", options.ProjectName))
	} else {
		options.Logger.SetPrefix("ADDON")
	}

	options.Logger.EnableDateTime(false)

	// change relative paths of configuration files to full path based on git root
	repoRoot, repoErr := common.GitRootPath(".")
	if repoErr != nil {
		repoRoot = "."
	}

	options.Logger.ShortInfo("Checking for local changes in the repository")

	isChanges, files, err := common.ChangesToBePush(options.Testing, repoRoot)
	if err != nil {
		options.Logger.ShortError("Error checking for local changes in the repository")
		options.Testing.Fail()
		return fmt.Errorf("error checking for local changes in the repository: %w", err)
	}
	// remove ignored files
	if len(options.IgnorePattern) > 0 {
		filteredFiles := make([]string, 0)
		for _, file := range files {
			shouldKeep := true
			// ignore files are regex patterns
			for _, ignorePattern := range options.IgnorePattern {
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
			options.Logger.ShortError("Local changes found in the repository, please commit or stash the changes before running the test")
			options.Logger.ShortError("Files with changes:")
			for _, file := range files {
				options.Logger.ShortError(fmt.Sprintf("  %s", file))
			}
			options.Testing.Fail()
			return fmt.Errorf("local changes found in the repository")
		} else {
			options.Logger.ShortWarn("Local changes found in the repository, but skipping the check")
			options.Logger.ShortWarn("Files with changes:")
			for _, file := range files {
				options.Logger.ShortWarn(fmt.Sprintf("  %s", file))
			}
		}
	}

	// get current branch and repo url
	repo, branch, repoErr := common.GetCurrentPrRepoAndBranch()
	if repoErr != nil {
		options.Logger.ShortError("Error getting current branch and repo")
		options.Testing.Fail()
		return fmt.Errorf("error getting current branch and repo: %w", repoErr)
	}
	options.currentBranch = &branch
	options.currentBranchUrl = Core.StringPtr(fmt.Sprintf("%s/tree/%s", repo, branch))
	options.Logger.ShortInfo(fmt.Sprintf("Current branch: %s", branch))
	options.Logger.ShortInfo(fmt.Sprintf("Current repo: %s", repo))
	options.Logger.ShortInfo(fmt.Sprintf("Current branch URL: %s", *options.currentBranchUrl))

	// create new CloudInfoService if not supplied
	if options.CloudInfoService == nil {
		cloudInfoSvc, err := cloudinfo.NewCloudInfoServiceFromEnv("TF_VAR_ibmcloud_api_key", cloudinfo.CloudInfoServiceOptions{})
		if err != nil {
			return err
		}
		options.CloudInfoService = cloudInfoSvc
	}

	if !options.CatalogUseExisting {
		options.Logger.ShortInfo(fmt.Sprintf("Creating a new catalog: %s", options.CatalogName))
		catalog, err := options.CloudInfoService.CreateCatalog(options.CatalogName)
		if err != nil {
			options.Logger.ShortError(fmt.Sprintf("Error creating a new catalog: %v", err))
			options.Testing.Fail()
			return fmt.Errorf("error creating a new catalog: %w", err)
		}
		options.catalog = catalog
		options.Logger.ShortInfo(fmt.Sprintf("Created a new catalog: %s with ID %s", *options.catalog.Label, *options.catalog.ID))
	} else {
		options.Logger.ShortInfo("Using existing catalog")
		options.Logger.ShortWarn("Not implemented yet")
		// TODO: lookup the catalog ID no api for this
	}

	// import the offering
	// ensure install kind is set or return an error
	if !options.OfferingInstallKind.Valid() {
		options.Logger.ShortError(fmt.Sprintf("'%s' is not valid for OfferingInstallKind", options.OfferingInstallKind))
		options.Testing.Fail()
		return fmt.Errorf("'%s' is not valid for OfferingInstallKind", options.OfferingInstallKind)
	}
	// check offering name set or fail
	if options.OfferingName == "" {
		options.Logger.ShortError("OfferingName is not set")
		options.Testing.Fail()
		return fmt.Errorf("OfferingName is not set")
	}
	version := fmt.Sprintf("v0.0.1-dev-%s", options.Prefix)
	options.Logger.ShortInfo(fmt.Sprintf("Importing the offering flavor: %s from branch: %s as version: %s", options.OfferingFlavorName, *options.currentBranchUrl, version))
	offering, err := options.CloudInfoService.ImportOffering(*options.catalog.ID, *options.currentBranchUrl, options.OfferingName, options.OfferingFlavorName, version, options.OfferingInstallKind)
	if err != nil {
		options.Logger.ShortError(fmt.Sprintf("Error importing the offering: %v", err))
		options.Testing.Fail()
		return fmt.Errorf("error importing the offering: %w", err)
	}
	options.offering = offering
	options.Logger.ShortInfo(fmt.Sprintf("Imported flavor: %s with version: %s to %s", *options.offering.Label, version, *options.catalog.Label))

	return nil
}

func (options *TestAddonOptions) TestTearDown() {

	if !options.SkipTestTearDown {
		// if we are not skipping the test teardown, execute it
		options.testTearDown()
	}

}

func (options *TestAddonOptions) testTearDown() {
	// perform the test teardown
	options.Logger.ShortInfo("Performing test teardown")
	// Delete Catalog
	if options.catalog != nil {
		options.Logger.ShortInfo(fmt.Sprintf("Deleting the catalog %s with ID %s", *options.catalog.Label, *options.catalog.ID))
		err := options.CloudInfoService.DeleteCatalog(*options.catalog.ID)
		if err != nil {
			options.Logger.ShortError(fmt.Sprintf("Error deleting the catalog: %v", err))
			options.Testing.Fail()
		} else {
			options.Logger.ShortInfo(fmt.Sprintf("Deleted the catalog %s with ID %s", *options.catalog.Label, *options.catalog.ID))
		}
	} else {
		options.Logger.ShortInfo("No catalog to delete")
	}
}
