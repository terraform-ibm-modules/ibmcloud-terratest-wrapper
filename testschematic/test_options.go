package testschematic

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/jinzhu/copier"
	"github.com/stretchr/testify/require"
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
	Testing                 *testing.T `copier:"-"` // Testing The current test object
	TarIncludePatterns      []string
	BestRegionYAMLPath      string                         // BestRegionYAMLPath Path to the yaml containing regions and weights
	DefaultRegion           string                         // DefaultRegion default region if automatic detection fails
	ResourceGroup           string                         // ResourceGroup IBM Cloud resource group to use
	Region                  string                         // Region to use
	RequiredEnvironmentVars map[string]string              // RequiredEnvironmentVars
	TerraformVars           []TestSchematicTerraformVar    // TerraformVars variables to pass to terraform
	TemplateFolder          string                         // folder that contains terraform template, defaults to "."
	Tags                    []string                       // Tags optional tags to add
	Prefix                  string                         // Prefix to use when creating resources
	WaitJobCompleteMinutes  int16                          // number of minutes to wait for schematic job completions
	SchematicsApiURL        string                         // OPTIONAL: base URL for schematics API
	DeleteWorkspaceOnFail   bool                           // if there is a failure, should test delete the workspace and logs, default of false
	TerraformVersion        string                         // OPTIONAL: Schematics terraform version to use for template. If not supplied will be determined from required version in project
	NetrcSettings           []NetrcCredential              // array of .netrc credentials that will be set for schematics workspace
	WorkspaceEnvVars        []WorkspaceEnvironmentVariable // array of ENV variables to set inside workspace
	CloudInfoService        testhelper.CloudInfoServiceI   // OPTIONAL: Supply if you need multiple tests to share info service and data
	SchematicsApiSvc        SchematicsApiSvcI              // OPTIONAL: service pointer for interacting with external schematics api
	schematicsTestSvc       *SchematicsTestService         // OPTIONAL: internal property to specify pointer to test service, used for test mocking
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

// TestSchematicOptionsDefault is a constructor for struct TestSchematicOptions. This function will accept an existing instance of
// TestSchematicOptions values, and return a new instance of TestSchematicOptions with the original values set along with appropriate
// default values for any properties that were not set in the original options.
func TestSchematicOptionsDefault(originalOptions *TestSchematicOptions) *TestSchematicOptions {

	newOptions, err := originalOptions.Clone()
	require.NoError(originalOptions.Testing, err)

	newOptions.Prefix = fmt.Sprintf("%s-%s", newOptions.Prefix, strings.ToLower(random.UniqueId()))

	if newOptions.DefaultRegion == "" {
		newOptions.DefaultRegion = defaultRegion
	}
	// Verify required environment variables are set - better to do this now rather than retry and fail with every attempt
	checkVariables := []string{ibmcloudApiKeyVar}
	newOptions.RequiredEnvironmentVars = testhelper.GetRequiredEnvVars(newOptions.Testing, checkVariables)

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
