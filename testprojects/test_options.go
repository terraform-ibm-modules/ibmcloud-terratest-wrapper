package testprojects

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/IBM/go-sdk-core/v5/core"

	project "github.com/IBM/project-go-sdk/projectv1"
	"github.com/jinzhu/copier"
	"github.com/stretchr/testify/require"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/cloudinfo"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/common"
)

const defaultRegion = "us-south"
const defaultRegionYaml = "../common-dev-assets/common-go-assets/cloudinfo-region-vpc-gen2-prefs.yaml"
const ibmcloudApiKeyVar = "TF_VAR_ibmcloud_api_key"

type TestProjectsOptions struct {
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

	// ProjectsApiURL Base URL of the schematics REST API. Set to override default.
	// Default: https://projects.api.cloud.ibm.com
	ProjectsApiURL string

	// ConfigrationPath Path to the configuration file that will be used to create the project.
	// Deprecated: Use StackConfigurationPath instead.
	ConfigrationPath string
	// StackConfigurationPath Path to the configuration file that will be used to create the stack.
	StackConfigurationPath string
	StackCatalogJsonPath   string

	// StackPollTimeSeconds The number of seconds to wait between polling the stack status. 0 is not valid and will default to 60 seconds.
	StackPollTimeSeconds int

	// StackAutoSync If set to true, when deploying or undeploying a member, a sync with Schematics will be executed if the member has not updated before the StackAutoSyncInterval.
	StackAutoSync bool
	// StackAutoSyncInterval The number of minutes to wait before syncing with Schematics if state has not updated. Default is 20 minutes.
	StackAutoSyncInterval int

	// Deprecated: Deploy order is now determined by the project.
	StackConfigurationOrder []string
	// Deprecated: Deploy groups are now determined by the project.
	StackUndeployOrder []string
	// Deprecated: Deploy groups are now determined by the project.
	stackUndeployGroups [][]string

	// StackAuthorizations The authorizations to use for the project.
	// If not set, the default will be to use the TF_VAR_ibmcloud_api_key environment variable.
	// Can be used to set Trusted Profile or API Key.
	StackAuthorizations *project.ProjectConfigAuth

	// StackMemberInputs [ "primary-da": {["input1": "value1", "input2": 2]}, "secondary-da": {["input1": "value1", "input2": 2]}]
	StackMemberInputs map[string]map[string]interface{}
	// StackInputs {"input1": "value1", "input2": 2}
	StackInputs map[string]interface{}

	// CatalogProductName The name of the product in the catalog. Defaults to the first product in the catalog.
	CatalogProductName string
	// CatalogFlavorName The name of the flavor in the catalog. Defaults to the first flavor in the catalog.
	CatalogFlavorName string

	// ParallelDeploy If set to true, the test will deploy the stack in parallel.
	// This will deploy the stack in batches of whatever is not waiting on a prerequisite to be deployed.
	// Note Undeploy will still be in serial.
	// Deprecated: All deploys are now parallel by default using projects built-in parallel deploy feature.
	ParallelDeploy bool

	// ValidationTimeoutMinutes The number of minutes to wait for the project to validate.
	// Deprecated: This is now handled by projects and we only use DeployTimeoutMinutes for the entire project.
	ValidationTimeoutMinutes int
	// DeployTimeoutMinutes The number of minutes to wait for the stack to deploy. Also used for undeploy. Default is 6 hours.
	DeployTimeoutMinutes int

	// If you want to skip teardown use this flag
	SkipTestTearDown  bool
	SkipUndeploy      bool
	SkipProjectDelete bool

	// PostCreateDelay is the delay to wait after creating resources before attempting to read them.
	// This helps with eventual consistency issues in IBM Cloud APIs.
	// Default: 1 second. Set to a pointer to 0 duration to disable delays explicitly.
	PostCreateDelay *time.Duration

	// internal use
	currentProject       *project.Project
	currentProjectConfig *cloudinfo.ProjectsConfig

	currentStack       *project.StackDefinition
	currentStackConfig *cloudinfo.ConfigDetails

	// Hooks These allow us to inject custom code into the test process
	// example to set a hook:
	// options.PreDeployHook = func(options *TestProjectsOptions) error {
	//     // do something
	//     return nil
	// }
	PreDeployHook    func(options *TestProjectsOptions) error // In upgrade tests, this hook will be called before the deploy
	PostDeployHook   func(options *TestProjectsOptions) error // In upgrade tests, this hook will be called after the deploy
	PreUndeployHook  func(options *TestProjectsOptions) error // If this fails, the undeploy will continue
	PostUndeployHook func(options *TestProjectsOptions) error

	Logger *common.TestLogger
}

// TestProjectOptionsDefault Default constructor for TestProjectsOptions
// This function will accept an existing instance of
// TestProjectOptions values, and return a new instance of TestProjectOptions with the original values set along with appropriate
// default values for any properties that were not set in the original options.
// Summary of default values:
// - Prefix: original prefix with a unique 6-digit random string appended
func TestProjectOptionsDefault(originalOptions *TestProjectsOptions) *TestProjectsOptions {
	newOptions, err := originalOptions.Clone()
	require.NoError(originalOptions.Testing, err)

	newOptions.Prefix = fmt.Sprintf("%s-%s", newOptions.Prefix, common.UniqueId())

	// Verify required environment variables are set - better to do this now rather than retry and fail with every attempt
	checkVariables := []string{ibmcloudApiKeyVar}
	newOptions.RequiredEnvironmentVars = common.GetRequiredEnvVars(newOptions.Testing, checkVariables)

	if newOptions.ProjectName == "" {
		newOptions.ProjectName = fmt.Sprintf("project%s", newOptions.Prefix)
	}
	if newOptions.ProjectDescription == "" {
		newOptions.ProjectDescription = fmt.Sprintf("Testing %s-project", newOptions.Prefix)
	}

	if newOptions.ResourceGroup == "" {
		newOptions.ResourceGroup = "Default"
	}

	if newOptions.StackConfigurationPath == "" {
		newOptions.StackConfigurationPath = "stack_definition.json"
	}
	if newOptions.StackCatalogJsonPath == "" {
		newOptions.StackCatalogJsonPath = "ibm_catalog.json"
	}
	if newOptions.ValidationTimeoutMinutes == 0 {
		newOptions.ValidationTimeoutMinutes = 60
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

	if newOptions.StackAutoSyncInterval == 0 {
		newOptions.StackAutoSyncInterval = 20
	}

	if newOptions.StackPollTimeSeconds == 0 {
		newOptions.StackPollTimeSeconds = 60
	}
	// if newOptions.ProjectLocation == ""
	// a random location will be selected at project creation time in CreateProjectFromConfig

	if newOptions.StackAuthorizations == nil {
		newOptions.StackAuthorizations = &project.ProjectConfigAuth{
			ApiKey: core.StringPtr(os.Getenv(ibmcloudApiKeyVar)),
			Method: core.StringPtr(project.ProjectConfigAuth_Method_ApiKey),
		}
	}

	// Set default post-creation delay if not already set
	if newOptions.PostCreateDelay == nil {
		delay := 1 * time.Second
		newOptions.PostCreateDelay = &delay
	}

	return newOptions
}

// Clone makes a deep copy of most fields on the Options object and returns it.
//
// NOTE: options.SshAgent and options.Logger CANNOT be deep copied (e.g., the SshAgent struct contains channels and
// listeners that can't be meaningfully copied), so the original values are retained.
func (options *TestProjectsOptions) Clone() (*TestProjectsOptions, error) {
	newOptions := &TestProjectsOptions{}
	if err := copier.Copy(newOptions, options); err != nil {
		return nil, err
	}

	// the Copy library does not handle pointer of struct very well so we want to manually take care of our
	// pointers to other complex structs
	newOptions.Testing = options.Testing

	return newOptions, nil
}

func (options *TestProjectsOptions) SetCurrentStackConfig(currentStackConfig *cloudinfo.ConfigDetails) {
	options.currentStackConfig = currentStackConfig
}

func (options *TestProjectsOptions) SetCurrentProjectConfig(currentProjectConfig *cloudinfo.ProjectsConfig) {
	options.currentProjectConfig = currentProjectConfig
}
