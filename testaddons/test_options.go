package testaddons

import (
	"fmt"
	"strings"
	"testing"

	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/platform-services-go-sdk/catalogmanagementv1"

	project "github.com/IBM/project-go-sdk/projectv1"
	"github.com/gruntwork-io/terratest/modules/random"
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
	// This resource group will be used to create the project and stack.
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

	// SkipLocalChangeCheck If set to true, the test will not check for local changes before deploying.
	SkipLocalChangeCheck bool
	// LocalChangesIgnorePattern List of regex patterns to ignore files or directories when checking for local changes.
	LocalChangesIgnorePattern []string

	// internal use
	currentProject       *project.Project
	currentProjectConfig *cloudinfo.ProjectsConfig

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

	newOptions.Prefix = fmt.Sprintf("%s-%s", newOptions.Prefix, strings.ToLower(random.UniqueId()))
	newOptions.AddonConfig.Prefix = newOptions.Prefix

	// Verify required environment variables are set - better to do this now rather than retry and fail with every attempt
	checkVariables := []string{ibmcloudApiKeyVar}
	newOptions.RequiredEnvironmentVars = common.GetRequiredEnvVars(newOptions.Testing, checkVariables)

	if newOptions.CatalogName == "" {
		newOptions.CatalogName = fmt.Sprintf("dev-addon-test-%s", newOptions.Prefix)
	}
	if newOptions.ProjectName == "" {
		newOptions.ProjectName = fmt.Sprintf("addon%s", newOptions.Prefix)
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
	if newOptions.LocalChangesIgnorePattern == nil {
		// ignore tests directory by default
		newOptions.LocalChangesIgnorePattern = []string{"^tests/.*"}
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
