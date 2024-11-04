package testschematic

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/jinzhu/copier"
	"github.com/stretchr/testify/require"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/cloudinfo"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/common"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testhelper"
)

const defaultRegion = "us-south"
const defaultRegionYaml = "../common-dev-assets/common-go-assets/cloudinfo-region-vpc-gen2-prefs.yaml"
const ibmcloudApiKeyVar = "TF_VAR_ibmcloud_api_key"
const defaultGitUserEnvKey = "GIT_TOKEN_USER"
const defaultGitTokenEnvKey = "GIT_TOKEN"
const DefaultWaitJobCompleteMinutes = int16(120) // default 2 hrs wait time
const DefaultSchematicsApiURL = "https://schematics.cloud.ibm.com"

// TestSchematicOptions is the main data struct containing all options related to running a Terraform unit test wihtin IBM Schematics Workspaces
type TestSchematicOptions struct {
	// REQUIRED: a pointer to an initialized testing object.
	// Typically you would assign the test object used in the unit test.
	Testing *testing.T `copier:"-"`

	// Assign the list of file patterns, including directories, that you want included in the TAR file that will get uploaded
	// to the schematics workspace. Wildcards are valid.
	// Only files that match these patterns will be used as source for the Schematic Workspace in this test.
	// NOTE: all file paths are relative to the root of the current Git project.
	//
	// Examples:
	// "*.tf", "scripts/*.sh", "examples/basic/*", "data/my-data.yml"
	//
	// Default if not provided:
	// "*.tf" (all terraform files in project root only)
	TarIncludePatterns []string

	// The default constructors will use this map to check that all required environment variables are set properly.
	// If any are missing, the test will fail.
	RequiredEnvironmentVars map[string]string

	// Path to YAML file contaning preferences for how dynamic regions should be chosen.
	// See examples in cloudinfo/testdata for proper format.
	BestRegionYAMLPath string

	// Used with dynamic region selection, if any errors occur this will be the region used (fail-open)
	DefaultRegion string

	// If set during creation, this region will be used for test and dynamic region selection will be skipped.
	// If left empty, this will be populated by dynamic region selection by default constructor and can be referenced later.
	Region string

	// Only required if using the WithVars constructor, as this value will then populate the `resource_group` input variable.
	ResourceGroup string

	// REQUIRED: the string prefix that will be prepended to all resource names, typically sent in as terraform input variable.
	// Set this value in the default constructors and a unique 6-digit random string will be appended.
	// Can then be referenced after construction and used as unique variable.
	//
	// Example:
	// Supplied to constructor = `my-test`
	// After constructor = `my-test-xu5oby`
	Prefix string

	// This array will be used to construct a valid `Variablestore` configuration for the Schematics Workspace Template
	TerraformVars []TestSchematicTerraformVar

	// This value will set the `folder` attribute in the Schematics template, and will be used as the execution folder for terraform.
	// Defaults to root directory of source, "." if not supplied.
	//
	// Example: if you are testing a module source, and the execution is located in the subdirectory `examples/basic`, then set this
	// to that value.
	TemplateFolder string

	// Optional list of tags that will be applied to the Schematics Workspace instance
	Tags []string

	// Amount of time, in minutes, to wait for any schematics job to finish. Set to override the default.
	// Default: 120 (two hours)
	WaitJobCompleteMinutes int16

	// Base URL of the schematics REST API. Set to override default.
	// Default: https://schematics.cloud.ibm.com
	SchematicsApiURL string

	// Set this to true if you would like to delete the test Schematic Workspace if the test fails.
	// By default this will be false, and if a failure happens the workspace and logs will be preserved for analysis.
	DeleteWorkspaceOnFail bool

	// If you want to skip test teardown (both resource destroy and workspace deletion)
	SkipTestTearDown bool

	// This value is used to set the terraform version attribute for the workspace and template.
	// If left empty, an empty value will be set in the template which will cause the Schematic jobs to use the highest available version.
	//
	// Format: "terraform_v1.x"
	TerraformVersion string

	// Use this optional list to provide .netrc credentials that will be used by schematics to access any private git repos accessed by
	// the project.
	//
	// This data will be used to construct a special `__netrc__` environment variable in the template.
	NetrcSettings []NetrcCredential

	// Use this optional list to provide any additional ENV values for the template.
	// NOTE: these values are only used to configure the template, they are not set as environment variables outside of schematics.
	WorkspaceEnvVars []WorkspaceEnvironmentVariable // array of ENV variables to set inside workspace

	CloudInfoService  cloudinfo.CloudInfoServiceI // OPTIONAL: Supply if you need multiple tests to share info service and data
	SchematicsApiSvc  SchematicsApiSvcI           // OPTIONAL: service pointer for interacting with external schematics api
	schematicsTestSvc *SchematicsTestService      // internal property to specify pointer to test service, used for test mocking

	// For Consistency Checks: Specify terraform resource names to ignore for consistency checks.
	// You can ignore specific resources in both idempotent and upgrade consistency checks by adding their names to these
	// lists. There are separate lists for adds, updates, and destroys.
	//
	// This can be useful if you have resources like `null_resource` that are marked with a lifecycle that causes a refresh on every run.
	// Normally this would fail a consistency check but can be ignored by adding to one of these lists.
	//
	// Name format is terraform style, for example: `module.some_module.null_resource.foo`
	IgnoreAdds     testhelper.Exemptions
	IgnoreDestroys testhelper.Exemptions
	IgnoreUpdates  testhelper.Exemptions

	// These optional fields can be used to override the default retry settings for making Schematics API calls.
	// If SDK/API calls to Schematics result in errors, such as retrieving existing workspace details,
	// the test framework will retry those calls for a set number of times, with a wait time between calls.
	//
	// NOTE: these are pointers to int, so that we know they were purposly set (they are nil), as zero is a legitimate value
	//
	// Current Default: 5 retries, 5 second wait
	SchematicSvcRetryCount       *int
	SchematicSvcRetryWaitSeconds *int
}

type TestSchematicTerraformVar struct {
	Name     string      // name of variable
	Value    interface{} // value of variable
	DataType string      // the TERRAFORM DATA TYPE of the varialbe (not golang type)
	Secure   bool        // true if value should be hidden
}

type NetrcCredential struct {
	Host     string // hostname or machine name of the entry
	Username string // user name
	Password string // password or token
}

type WorkspaceEnvironmentVariable struct {
	Key    string // key name to set in workspace
	Value  string // value of env var
	Hidden bool   // metadata to hide this env var
	Secure bool   // metadata to mark value as sensitive
}

// To support consistency check options interface
func (options *TestSchematicOptions) GetCheckConsistencyOptions() *testhelper.CheckConsistencyOptions {
	return &testhelper.CheckConsistencyOptions{
		Testing:        options.Testing,
		IgnoreAdds:     options.IgnoreAdds,
		IgnoreDestroys: options.IgnoreDestroys,
		IgnoreUpdates:  options.IgnoreUpdates,
		IsUpgradeTest:  false,
	}
}

// TestSchematicOptionsDefault is a constructor for struct TestSchematicOptions. This function will accept an existing instance of
// TestSchematicOptions values, and return a new instance of TestSchematicOptions with the original values set along with appropriate
// default values for any properties that were not set in the original options.
//
// Summary of properties changed:
// * appends unique 6-char string to end of original prefix
// * checks that certain required environment variables are set
// * computes best dynamic region for test, if Region is not supplied
// * sets various other properties to sensible defaults if not supplied
func TestSchematicOptionsDefault(originalOptions *TestSchematicOptions) *TestSchematicOptions {

	newOptions, err := originalOptions.Clone()
	require.NoError(originalOptions.Testing, err)

	newOptions.Prefix = fmt.Sprintf("%s-%s", newOptions.Prefix, strings.ToLower(random.UniqueId()))

	if newOptions.DefaultRegion == "" {
		newOptions.DefaultRegion = defaultRegion
	}
	// Verify required environment variables are set - better to do this now rather than retry and fail with every attempt
	checkVariables := []string{ibmcloudApiKeyVar}
	newOptions.RequiredEnvironmentVars = common.GetRequiredEnvVars(newOptions.Testing, checkVariables)

	if newOptions.Region == "" {
		// Get the best region
		// Programmatically determine region to use based on availability
		// Set OS environment variable FORCE_TEST_REGION to force a specific region
		regionOptions := &testhelper.TesthelperTerraformOptions{
			CloudInfoService:              newOptions.CloudInfoService,
			ExcludeActivityTrackerRegions: false,
		}
		if newOptions.BestRegionYAMLPath != "" {
			newOptions.Region, _ = testhelper.GetBestVpcRegionO(newOptions.RequiredEnvironmentVars[ibmcloudApiKeyVar], newOptions.BestRegionYAMLPath, newOptions.DefaultRegion, *regionOptions)
		} else {
			newOptions.Region, _ = testhelper.GetBestVpcRegionO(newOptions.RequiredEnvironmentVars[ibmcloudApiKeyVar], defaultRegionYaml, newOptions.DefaultRegion, *regionOptions)
		}
	}

	if newOptions.WaitJobCompleteMinutes <= 0 {
		newOptions.WaitJobCompleteMinutes = DefaultWaitJobCompleteMinutes
	}

	if len(newOptions.SchematicsApiURL) == 0 {
		newOptions.SchematicsApiURL = DefaultSchematicsApiURL
	}

	return newOptions

}

// Clone makes a deep copy of most fields on the Options object and returns it.
//
// NOTE: options.SshAgent and options.Logger CANNOT be deep copied (e.g., the SshAgent struct contains channels and
// listeners that can't be meaningfully copied), so the original values are retained.
func (options *TestSchematicOptions) Clone() (*TestSchematicOptions, error) {
	newOptions := &TestSchematicOptions{}
	if err := copier.Copy(newOptions, options); err != nil {
		return nil, err
	}

	// the Copy library does not handle pointer of struct very well so we want to manually take care of our
	// pointers to other complex structs
	newOptions.Testing = options.Testing

	return newOptions, nil
}

// AddNetrcCredential is a helper function for TestSchematicOptions that will append a new netrc credential struct to the appropriate array
// in the options struct
func (options *TestSchematicOptions) AddNetrcCredential(hostname string, username string, password string) {
	options.NetrcSettings = append(options.NetrcSettings, NetrcCredential{
		Host:     hostname,
		Username: username,
		Password: password,
	})
}

// AddNetrcCredential is a helper function for TestSchematicOptions that will append a new netrc credential struct to the appropriate options array,
// by retrieving the username and password by lookup up supplied environment variable keys.
// error returned if any keys are missing in local environment
func (options *TestSchematicOptions) AddNetrcCredentialByEnv(hostname string, usernameEnvKey string, passwordEnvKey string) error {
	user, userSet := os.LookupEnv(usernameEnvKey)
	pass, passSet := os.LookupEnv(passwordEnvKey)

	if !userSet {
		return fmt.Errorf("netrc username environment variable [%s] has not been set", usernameEnvKey)
	}

	if !passSet {
		return fmt.Errorf("netrc password environment variable [%s] has not been set", passwordEnvKey)
	}

	options.AddNetrcCredential(hostname, user, pass)

	return nil
}

// AddNetrcCredentialByEnvDefault is a helper function for TestSchematicOptions that will append a new netrc credential struct to the appropriate options array,
// by retrieving the username and password by looking up the default environment variable keys
// error returned if any keys are missing in local environment
func (options *TestSchematicOptions) AddNetrcCredentialByEnvDefault(hostname string) error {
	return options.AddNetrcCredentialByEnv(hostname, defaultGitUserEnvKey, defaultGitTokenEnvKey)
}

// AddWorkspaceEnvVar is a helper function for TestSchematicOptions that will append a new ENV entry to options that will be added to the workspace during the test
func (options *TestSchematicOptions) AddWorkspaceEnvVar(key string, value string, hidden bool, secure bool) {
	options.WorkspaceEnvVars = append(options.WorkspaceEnvVars, WorkspaceEnvironmentVariable{
		Key:    key,
		Value:  value,
		Hidden: hidden,
		Secure: secure,
	})
}

// AddWorkspaceEnvVarFromLocalEnv is a helper function for TestSchematicOptions that will append a new ENV entry to options that will be added to the workspace during the test.
// The value for the environment variable will be queried from the local OS under the same env var key.
// error returned if the key is not set in local environment.
func (options *TestSchematicOptions) AddWorkspaceEnvVarFromLocalEnv(key string, hidden bool, secure bool) error {
	val, valSet := os.LookupEnv(key)
	if !valSet {
		return fmt.Errorf("local environment variable [%s] is not set", key)
	}

	options.AddWorkspaceEnvVar(key, val, hidden, secure)

	return nil
}
