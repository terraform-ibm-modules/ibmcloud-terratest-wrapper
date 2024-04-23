package testprojects

import (
	"fmt"
	project "github.com/IBM/project-go-sdk/projectv1"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/jinzhu/copier"
	"github.com/stretchr/testify/require"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/cloudinfo"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/common"
	"strings"
	"testing"
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

	ProjectName        string
	ProjectDescription string

	CloudInfoService cloudinfo.CloudInfoServiceI // OPTIONAL: Supply if you need multiple tests to share info service and data

	// ProjectsApiURL Base URL of the schematics REST API. Set to override default.
	// Default: https://projects.api.cloud.ibm.com
	ProjectsApiURL string

	// ConfigrationPath Path to the configuration file that will be used to create the project.
	ConfigrationPath string
	// StackConfigurationPath Path to the configuration file that will be used to create the stack.
	StackConfigurationPath  string
	StackLocatorID          string
	StackCatalogJsonPath    string
	StackConfigurationOrder []string
	// StackMemberInputs [ "primary-da": {["input1": "value1", "input2": 2]}, "secondary-da": {["input1": "value1", "input2": 2]}]
	StackMemberInputs map[string]map[string]interface{}
	// StackInputs {"input1": "value1", "input2": 2}
	StackInputs map[string]interface{}

	ValidationTimeoutMinutes int
	DeployTimeoutMinutes     int

	// If you want to skip teardown use this flag
	SkipTestTearDown bool

	// internal use
	currentProject *project.Project
	currentStack   *project.StackDefinition
}

// TestProjectOptionsDefault Default constructor for TestProjectsOptions
// This function will accept an existing instance of
// TestProjectOptions values, and return a new instance of TestProjectOptions with the original values set along with appropriate
// default values for any properties that were not set in the original options.
// Summary of default values:
// - Prefix: original prefix with a unique 6-digit random string appended
// - DefaultRegion: if not set, will be determined by dynamic region selection
// - Region: if not set, will be determined by dynamic region selection
// - BestRegionYAMLPath: if not set, will use the default region YAML path
func TestProjectOptionsDefault(originalOptions *TestProjectsOptions) *TestProjectsOptions {
	newOptions, err := originalOptions.Clone()
	require.NoError(originalOptions.Testing, err)

	newOptions.Prefix = fmt.Sprintf("%s-%s", newOptions.Prefix, strings.ToLower(random.UniqueId()))

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
	// TODO: default stack configuration path and catalog path to repo root stack_definition.json and ibm_catalog.json
	repoRoot, repoErr := common.GitRootPath(".")
	if repoErr != nil {
		repoRoot = "."
	}
	if newOptions.StackConfigurationPath == "" {
		newOptions.StackConfigurationPath = fmt.Sprintf("%s/stack_definition.json", repoRoot)
	}
	if newOptions.StackCatalogJsonPath == "" {
		newOptions.StackCatalogJsonPath = fmt.Sprintf("%s/ibm_catalog.json", repoRoot)
	}
	if newOptions.ValidationTimeoutMinutes == 0 {
		newOptions.ValidationTimeoutMinutes = 60
	}
	if newOptions.DeployTimeoutMinutes == 0 {
		newOptions.DeployTimeoutMinutes = 6 * 60
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
