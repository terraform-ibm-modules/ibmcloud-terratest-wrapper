package testaddons

import (
	"fmt"
	"testing"
	"time"

	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/platform-services-go-sdk/catalogmanagementv1"

	project "github.com/IBM/project-go-sdk/projectv1"
	"github.com/jinzhu/copier"
	"github.com/stretchr/testify/require"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/cloudinfo"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/common"
)

const defaultRegion = "us-south"
const defaultRegionYaml = "../common-dev-assets/common-go-assets/cloudinfo-region-vpc-gen2-prefs.yaml"
const ibmcloudApiKeyVar = "TF_VAR_ibmcloud_api_key"

type TestAddonOptions struct {
	// REQUIRED: a pointer to an initialized testing object.
	// Typically you would assign the test object used in the unit test.
	Testing *testing.T `copier:"-"`

	// The default constructors will use this map to check that all required environment variables are set properly.
	// If any are missing, the test will fail.
	RequiredEnvironmentVars map[string]string

	// Only required if using the WithVars constructor, as this value will then populate the `resource_group` input variable.
	// This resource group will be used to create the project
	ResourceGroup string

	// REQUIRED: the string prefix that will be prepended to all resource names, typically sent in as terraform input variable.
	// Set this value in the default constructors and a unique 6-digit random string will be appended.
	// Can then be referenced after construction and used as unique variable.
	//
	// Example:
	// Supplied to constructor = `my-test`
	// After constructor = `my-test-xu5oby`
	Prefix string

	ProjectName              string
	ProjectDescription       string
	ProjectLocation          string
	ProjectDestroyOnDelete   *bool
	ProjectMonitoringEnabled *bool
	ProjectAutoDeploy        *bool
	ProjectEnvironments      []project.EnvironmentPrototype

	CloudInfoService cloudinfo.CloudInfoServiceI // OPTIONAL: Supply if you need multiple tests to share info service and data

	// CatalogUseExisting If set to true, the test will use an existing catalog.
	CatalogUseExisting bool
	// CatalogName The name of the catalog to create and deploy to.
	CatalogName string

	// SharedCatalog If set to true (default), catalogs and offerings will be shared across tests using the same TestOptions object.
	// When false, each test will create its own catalog and offering, which is useful for isolation but less efficient.
	// This applies to both individual tests and matrix tests.
	SharedCatalog *bool

	// Internal Use
	// catalog the catalog instance in use.
	catalog *catalogmanagementv1.Catalog

	// internal use
	// offering the offering created in the catalog.
	offering *catalogmanagementv1.Offering

	// AddonConfig The configuration for the addon to deploy.
	AddonConfig cloudinfo.AddonConfig

	// DeployTimeoutMinutes The number of minutes to wait for the stack to deploy. Also used for undeploy. Default is 6 hours.
	DeployTimeoutMinutes int

	// If you want to skip teardown use this flag
	SkipTestTearDown  bool
	SkipUndeploy      bool
	SkipProjectDelete bool

	// SkipInfrastructureDeployment If set to true, the test will skip the infrastructure deployment and undeploy operations.
	// All other validations and setup will still be performed.
	SkipInfrastructureDeployment bool

	// SkipLocalChangeCheck If set to true, the test will not check for local changes before deploying.
	SkipLocalChangeCheck bool
	// SkipRefValidation If set to true, the test will not check for reference validation before deploying.
	SkipRefValidation bool
	// SkipDependencyValidatio If set to true, the test will not check for dependency validation before deploying
	SkipDependencyValidation bool

	// InputValidationRetries The number of retry attempts for input validation (default: 3)
	// This handles timing issues where the backend database hasn't been updated yet after configuration changes
	InputValidationRetries int
	// InputValidationRetryDelay The delay between retry attempts for input validation (default: 2 seconds)
	InputValidationRetryDelay time.Duration

	// VerboseValidationErrors If set to true, shows detailed individual error messages instead of consolidated summary
	VerboseValidationErrors bool
	// EnhancedTreeValidationOutput If set to true, shows dependency tree with validation status annotations
	EnhancedTreeValidationOutput bool
	// LocalChangesIgnorePattern List of regex patterns to ignore files or directories when checking for local changes.
	LocalChangesIgnorePattern []string

	// TestCaseName Optional custom identifier for log messages. When specified, log output will show:
	// "[TestFunction - ADDON - TestCaseName]" instead of using the project name.
	// Matrix tests automatically set this using the AddonTestCase.Name field.
	TestCaseName string

	// internal use
	currentProject       *project.Project
	currentProjectConfig *cloudinfo.ProjectsConfig
	deployedConfigs      *cloudinfo.DeployedAddonsDetails // Store deployed configs for validation

	currentBranch    *string
	currentBranchUrl *string

	// Hooks These allow us to inject custom code into the test process
	// example to set a hook:
	// options.PreDeployHook = func(options *TestProjectsOptions) error {
	//     // do something
	//     return nil
	// }
	PreDeployHook    func(options *TestAddonOptions) error // In upgrade tests, this hook will be called before the deploy
	PostDeployHook   func(options *TestAddonOptions) error // In upgrade tests, this hook will be called after the deploy
	PreUndeployHook  func(options *TestAddonOptions) error // If this fails, the undeploy will continue
	PostUndeployHook func(options *TestAddonOptions) error

	Logger *common.TestLogger
}

// TestAddonsOptionsDefault Default constructor for TestAddonOptions
// This function will accept an existing instance of
// TestAddonOptions values, and return a new instance of TestAddonOptions with the original values set along with appropriate
// default values for any properties that were not set in the original options.
// Summary of default values:
// - Prefix: original prefix with a unique 6-digit random string appended
func TestAddonsOptionsDefault(originalOptions *TestAddonOptions) *TestAddonOptions {
	newOptions, err := originalOptions.Clone()
	require.NoError(originalOptions.Testing, err)

	// Handle empty prefix case to avoid leading hyphen
	if newOptions.Prefix == "" {
		newOptions.Prefix = common.UniqueId()
	} else {
		newOptions.Prefix = fmt.Sprintf("%s-%s", newOptions.Prefix, common.UniqueId())
	}
	newOptions.AddonConfig.Prefix = newOptions.Prefix

	// Verify required environment variables are set - better to do this now rather than retry and fail with every attempt
	// Only check if RequiredEnvironmentVars hasn't been explicitly set (for unit tests that don't need env vars)
	if newOptions.RequiredEnvironmentVars == nil {
		checkVariables := []string{ibmcloudApiKeyVar}
		newOptions.RequiredEnvironmentVars = common.GetRequiredEnvVars(newOptions.Testing, checkVariables)
	}

	if newOptions.CatalogName == "" {
		newOptions.CatalogName = fmt.Sprintf("addon-test-catalog-%s", newOptions.Prefix)
	}
	if newOptions.ProjectName == "" {
		newOptions.ProjectName = fmt.Sprintf("addon-%s", newOptions.Prefix)
	}
	if newOptions.ProjectDescription == "" {
		newOptions.ProjectDescription = fmt.Sprintf("Testing %s-addon", newOptions.Prefix)
	}

	if newOptions.ResourceGroup == "" {
		newOptions.ResourceGroup = "Default"
	}

	if newOptions.DeployTimeoutMinutes == 0 {
		newOptions.DeployTimeoutMinutes = 6 * 60
	}
	if newOptions.ProjectDestroyOnDelete == nil {
		newOptions.ProjectDestroyOnDelete = core.BoolPtr(true)
	}
	if newOptions.ProjectMonitoringEnabled == nil {
		newOptions.ProjectMonitoringEnabled = core.BoolPtr(true)
	}
	if newOptions.ProjectAutoDeploy == nil {
		newOptions.ProjectAutoDeploy = core.BoolPtr(true)
	}

	// We need to handle the bool default properly - default SharedCatalog to false for individual tests
	// Matrix tests will override this to true and handle cleanup automatically
	if newOptions.SharedCatalog == nil {
		newOptions.SharedCatalog = core.BoolPtr(false)
	}

	// Set default retry configuration for input validation
	if newOptions.InputValidationRetries <= 0 {
		newOptions.InputValidationRetries = 3
	}
	if newOptions.InputValidationRetryDelay <= 0 {
		newOptions.InputValidationRetryDelay = 2 * time.Second
	}

	// Always include default ignore patterns and append user patterns if provided
	defaultIgnorePatterns := []string{
		"^common-dev-assets$",   // Ignore submodule pointer changes for common-dev-assets
		"^common-dev-assets/.*", // Ignore changes in common-dev-assets directory
		"^tests/.*",             // Ignore changes in tests directory
		".*\\.json$",            // Ignore JSON files
		".*\\.out$",             // Ignore output files
	}

	if newOptions.LocalChangesIgnorePattern == nil {
		newOptions.LocalChangesIgnorePattern = defaultIgnorePatterns
	} else {
		// Append user patterns to default patterns
		newOptions.LocalChangesIgnorePattern = append(defaultIgnorePatterns, newOptions.LocalChangesIgnorePattern...)
	}

	return newOptions
}

// Clone makes a deep copy of most fields on the Options object and returns it.
//
// NOTE: options.SshAgent and options.Logger CANNOT be deep copied (e.g., the SshAgent struct contains channels and
// listeners that can't be meaningfully copied), so the original values are retained.
func (options *TestAddonOptions) Clone() (*TestAddonOptions, error) {
	newOptions := &TestAddonOptions{}
	if err := copier.Copy(newOptions, options); err != nil {
		return nil, err
	}

	// the Copy library does not handle pointer of struct very well so we want to manually take care of our
	// pointers to other complex structs
	newOptions.Testing = options.Testing

	return newOptions, nil
}

// copy creates a deep copy of TestAddonOptions for use in matrix tests
// This allows BaseOptions to be safely shared across test cases
// copyBoolPointer creates a deep copy of a bool pointer
func copyBoolPointer(original *bool) *bool {
	if original == nil {
		return nil
	}
	copied := *original
	return &copied
}

func (options *TestAddonOptions) copy() *TestAddonOptions {
	if options == nil {
		return nil
	}

	copied := &TestAddonOptions{
		Testing:                      options.Testing, // Will be overridden per test case
		RequiredEnvironmentVars:      options.RequiredEnvironmentVars,
		ResourceGroup:                options.ResourceGroup,
		Prefix:                       options.Prefix,
		ProjectName:                  options.ProjectName,
		ProjectDescription:           options.ProjectDescription,
		ProjectLocation:              options.ProjectLocation,
		ProjectDestroyOnDelete:       options.ProjectDestroyOnDelete,
		ProjectMonitoringEnabled:     options.ProjectMonitoringEnabled,
		ProjectAutoDeploy:            options.ProjectAutoDeploy,
		ProjectEnvironments:          options.ProjectEnvironments,
		CloudInfoService:             options.CloudInfoService,
		CatalogUseExisting:           options.CatalogUseExisting,
		CatalogName:                  options.CatalogName,
		SharedCatalog:                copyBoolPointer(options.SharedCatalog),
		AddonConfig:                  options.AddonConfig, // Note: shallow copy, will be overridden
		DeployTimeoutMinutes:         options.DeployTimeoutMinutes,
		SkipTestTearDown:             options.SkipTestTearDown,
		SkipUndeploy:                 options.SkipUndeploy,
		SkipProjectDelete:            options.SkipProjectDelete,
		SkipInfrastructureDeployment: options.SkipInfrastructureDeployment,
		SkipLocalChangeCheck:         options.SkipLocalChangeCheck,
		SkipRefValidation:            options.SkipRefValidation,
		SkipDependencyValidation:     options.SkipDependencyValidation,
		VerboseValidationErrors:      options.VerboseValidationErrors,
		EnhancedTreeValidationOutput: options.EnhancedTreeValidationOutput,
		LocalChangesIgnorePattern:    options.LocalChangesIgnorePattern,
		TestCaseName:                 options.TestCaseName,
		InputValidationRetries:       options.InputValidationRetries,
		InputValidationRetryDelay:    options.InputValidationRetryDelay,
		PreDeployHook:                options.PreDeployHook,
		PostDeployHook:               options.PostDeployHook,
		PreUndeployHook:              options.PreUndeployHook,
		PostUndeployHook:             options.PostUndeployHook,
		Logger:                       options.Logger,

		// These fields are not copied as they are managed per test instance
		catalog:              nil,
		offering:             nil,
		currentProject:       nil,
		currentProjectConfig: nil,
		deployedConfigs:      nil,
		currentBranch:        nil,
		currentBranchUrl:     nil,
	}

	return copied
}

// CleanupSharedResources cleans up shared catalog and offering resources
// This method is useful for cleaning up shared catalogs when using SharedCatalog=true with individual tests.
// For matrix tests, cleanup happens automatically and you don't need to call this method.
//
// Example usage:
//
//	options := testaddons.TestAddonsOptionsDefault(&testaddons.TestAddonOptions{
//	    Testing: t,
//	    Prefix: "shared-test",
//	    ResourceGroup: "my-rg",
//	    SharedCatalog: core.BoolPtr(true),
//	})
//	defer options.CleanupSharedResources() // Ensure cleanup happens
//
//	// Run multiple tests that share the catalog
//	err1 := options.RunAddonTest()
//	err2 := options.RunAddonTest()
func (options *TestAddonOptions) CleanupSharedResources() {
	if options.catalog != nil {
		options.Logger.ShortInfo(fmt.Sprintf("Deleting the shared catalog %s with ID %s", *options.catalog.Label, *options.catalog.ID))
		err := options.CloudInfoService.DeleteCatalog(*options.catalog.ID)
		if err != nil {
			options.Logger.ShortError(fmt.Sprintf("Error deleting the shared catalog: %v", err))
		} else {
			options.Logger.ShortInfo(fmt.Sprintf("Deleted the shared catalog %s with ID %s", *options.catalog.Label, *options.catalog.ID))
		}
	}
}
