package testhelper

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/jinzhu/copier"
	"github.com/stretchr/testify/require"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/cloudinfo"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/common"
)

const defaultRegion = "eu-gb"
const defaultRegionYaml = "../common-dev-assets/common-go-assets/cloudinfo-region-vpc-gen2-prefs.yaml"
const ibmcloudApiKeyVar = "TF_VAR_ibmcloud_api_key"

type TestOptions struct {
	// REQUIRED: a pointer to an initialized testing object.
	// Typically, you would assign the test object used in the unit test.
	Testing *testing.T `copier:"-"`

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

	// This map contains key-value pairs that will be used as variables for the test terraform run.
	// NOTE: when using the `...WithVars()` constructor, this map will be appended with the
	// standard test variables.
	TerraformVars map[string]interface{}

	// This is the subdirectory of the project that contains the terraform to run for the test.
	// This value is relative to the root directory of the project.
	// Defaults to root directory of project if not supplied.
	//
	// Example: if you are testing a module source, and the execution is located in the subdirectory `examples/basic`, then set this
	// to that value.
	TerraformDir string

	// Specify additional Terraform Options for Terratest using this variable.
	// see: https://pkg.go.dev/github.com/gruntwork-io/terratest/modules/terraform#Options
	TerraformOptions *terraform.Options `copier:"-"`

	// Use these options to have terratest execute using a terraform "workspace".
	UseTerraformWorkspace bool
	WorkspaceName         string
	WorkspacePath         string

	// Use these options to specify a base terraform repo and branch to use for upgrade tests.
	// If not supplied, the default logic will be used to determine the base repo and branch.
	// Will be overridden by environment variables BASE_TERRAFORM_REPO and BASE_TERRAFORM_BRANCH if set.
	//
	// For repositories that require authentication:
	// - For HTTPS repositories, set the GIT_TOKEN environment variable to your Personal Access Token (PAT).
	// - For SSH repositories, set the SSH_PRIVATE_KEY environment variable to your SSH private key.
	//   If the SSH_PRIVATE_KEY environment variable is not set, the default SSH key located at ~/.ssh/id_rsa will be used.
	//   Ensure that the appropriate public key is added to the repository's list of authorized keys.
	//
	// BaseTerraformRepo:   The URL of the base Terraform repository.
	BaseTerraformRepo string
	// BaseTerraformBranch: The branch within the base Terraform repository to use for upgrade tests.
	BaseTerraformBranch string

	// Resource tags to use for tests.
	// NOTE: when using `...WithVars()` constructor, this value will be automatically added to the appropriate
	// TerraformVars entries for tags.
	Tags []string

	// For Consistency Checks: Specify terraform resource names to ignore for consistency checks.
	// You can ignore specific resources in both idempotent and upgrade consistency checks by adding their names to these
	// lists. There are separate lists for adds, updates, and destroys.
	//
	// This can be useful if you have resources like `null_resource` that are marked with a lifecycle that causes a refresh on every run.
	// Normally this would fail a consistency check but can be ignored by adding to one of these lists.
	//
	// Name format is terraform style, for example: `module.some_module.null_resource.foo`
	IgnoreAdds     Exemptions
	IgnoreDestroys Exemptions
	IgnoreUpdates  Exemptions

	// Implicit Destroy can be used to speed up the `terraform destroy` action of the test, by removing resources from the state file
	// before the destroy process is executed.
	//
	// Use this for resources that are destroyed as part of a parent resource and do not need to be destroyed on their own.
	// For example: most helm releases inside of an OCP instance do not need to be individually destroyed, they will be destroyed
	// when the OCP instance is destroyed.
	//
	// Name format is terraform style, for example: `module.some_module.null_resource.foo`
	// NOTE: can specify at any layer of name, all children will also be removed, for example: `module.some_module` will remove all resources for that module.
	ImplicitDestroy []string

	//  If true the test will fail if any resources in ImplicitDestroy list fails to be removed from the state file
	ImplicitRequired bool

	// Set to true if using the `TestOptionsDefault` constructors with dynamic region selection, and you wish to exclude any regions that already
	// contain an Activity Tracker.
	ExcludeActivityTrackerRegions bool

	// Optional instance of a CloudInfoService to be used for any dynamic region selections.
	// If you wish parallel unit tests to use the same instance of CloudInfoService so that they do not pick the same region, you can initialize
	// this service in a variable that is shared amongst all unit tests and supply the pointer here.
	CloudInfoService cloudinfo.CloudInfoServiceI

	// Set to true if you wish for an Upgrade test to do a final `terraform apply` after the consistency check on the new (not main) branch.
	CheckApplyResultForUpgrade bool

	// If you want to skip test setup and teardown use these
	SkipTestSetup    bool
	SkipTestTearDown bool

	// These properties are considered READ ONLY and are used internally in the service to keep track of certain data elements.
	// Some of these properties are public, and can be used after the test is run to determine specific outcomes.
	IsUpgradeTest      bool   // Identifies if current test is an UPGRADE test, used for special processing
	UpgradeTestSkipped bool   // Informs the calling test that conditions were met to skip the upgrade test
	baseTempWorkingDir string // INTERNAL variable to store the base level of temporary working directory

}

// Default constructor for TestOptions struct. This constructor takes in an existing TestOptions object with minimal values set, and returns
// a new object that has amended or new values set.
//
// This version of the constructor will call `TestOptionsDefault` to set up most values, and then will add specific common variables to the
// `TerraformVars` map using supplied values from the existing TestOptions sent in.
//
// Common TerraformVars added:
// * prefix
// * region
// * resource_group
// * resource_tags
//
// DO NOT USE this constructor if your terraform test does not have these common variables as inputs, this is a convenience function for tests that
// support these common variables.
//
// NOTE: the variables are merged into the existing TerraformVars map, so it is best practice to have additional vars already set in the TestOptions
// object that is supplied.
func TestOptionsDefaultWithVars(originalOptions *TestOptions) *TestOptions {

	newOptions := TestOptionsDefault(originalOptions)

	// Vars to pass into module
	varsMap := make(map[string]interface{})

	common.ConditionalAdd(varsMap, "prefix", newOptions.Prefix, "")
	common.ConditionalAdd(varsMap, "region", newOptions.Region, "")
	common.ConditionalAdd(varsMap, "resource_group", newOptions.ResourceGroup, "")

	varsMap["resource_tags"] = common.GetTagsFromTravis()

	// Vars to pass into module
	newOptions.TerraformVars = common.MergeMaps(varsMap, newOptions.TerraformVars)

	return newOptions

}

// Default constructor for TestOptions struct. This constructor takes in an existing TestOptions object with minimal values set, and returns
// a new object that has amended or new values set, based on standard defaults.
//
// Summary of properties changed:
// * appends unique 6-char string to end of original prefix
// * checks that certain required environment variables are set
// * computes best dynamic region for test, if Region is not supplied
// * sets various other properties to sensible defaults
func TestOptionsDefault(originalOptions *TestOptions) *TestOptions {

	newOptions, err := originalOptions.Clone()
	require.NoError(originalOptions.Testing, err)

	newOptions.Prefix = fmt.Sprintf("%s-%s", newOptions.Prefix, strings.ToLower(random.UniqueId()))

	if newOptions.DefaultRegion == "" {
		newOptions.DefaultRegion = defaultRegion
	}
	// Verify required environment variables are set - better to do this now rather than retry and fail with every attempt
	checkVariables := []string{ibmcloudApiKeyVar}
	newOptions.RequiredEnvironmentVars = common.GetRequiredEnvVars(newOptions.Testing, checkVariables)

	newOptions.UseTerraformWorkspace = false

	if newOptions.Region == "" {
		// Get the best region
		// Programmatically determine region to use based on availability
		// Set OS environment variable FORCE_TEST_REGION to force a specific region
		regionOptions := &TesthelperTerraformOptions{
			CloudInfoService:              newOptions.CloudInfoService,
			ExcludeActivityTrackerRegions: newOptions.ExcludeActivityTrackerRegions,
		}
		if newOptions.BestRegionYAMLPath != "" {
			newOptions.Region, _ = GetBestVpcRegionO(newOptions.RequiredEnvironmentVars[ibmcloudApiKeyVar], newOptions.BestRegionYAMLPath, newOptions.DefaultRegion, *regionOptions)
		} else {
			newOptions.Region, _ = GetBestVpcRegionO(newOptions.RequiredEnvironmentVars[ibmcloudApiKeyVar], defaultRegionYaml, newOptions.DefaultRegion, *regionOptions)
		}
	}
	newOptions.SkipTestSetup = false
	newOptions.SkipTestTearDown = false

	newOptions.TerraformOptions = nil

	newOptions.IsUpgradeTest = false

	return newOptions

}

// Clone makes a deep copy of most fields on the Options object and returns it.
//
// NOTE: options.SshAgent and options.Logger CANNOT be deep copied (e.g., the SshAgent struct contains channels and
// listeners that can't be meaningfully copied), so the original values are retained.
func (options *TestOptions) Clone() (*TestOptions, error) {
	newOptions := &TestOptions{}
	if err := copier.Copy(newOptions, options); err != nil {
		return nil, err
	}

	// the Copy library does not handle pointer of struct very well so we want to manually take care of our
	// pointers to other complex structs
	newOptions.Testing = options.Testing
	newOptions.TerraformOptions = options.TerraformOptions

	return newOptions, nil
}
