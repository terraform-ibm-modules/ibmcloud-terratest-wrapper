package testhelper

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/jinzhu/copier"
	"github.com/stretchr/testify/require"
)

const defaultRegion = "eu-gb"
const defaultRegionYaml = "../common-dev-assets/common-go-assets/cloudinfo-region-vpc-gen2-prefs.yaml"
const ibmcloudApiKeyVar = "TF_VAR_ibmcloud_api_key"

type TestOptions struct {
	BestRegionYAMLPath            string                 // BestRegionYAMLPath Path to the yaml containing regions and weights
	DefaultRegion                 string                 // DefaultRegion default region if automatic detection fails
	ResourceGroup                 string                 // ResourceGroup IBM Cloud resource group to use
	Region                        string                 // Region to use
	TerraformVars                 map[string]interface{} // TerraformVars variables to pass to terraform
	TerraformDir                  string                 // TerraformDir Directory Terraform files are in
	TerraformOptions              *terraform.Options     `copier:"-"` // TerraformOptions Terraform options to use
	UseTerraformWorkspace         bool                   // UseTerraformWorkspace Use a Terraform workspace
	WorkspaceName                 string                 // WorkspaceName name of the workspace
	WorkspacePath                 string                 // WorkspacePath path to workspace
	RequiredEnvironmentVars       map[string]string      // RequiredEnvironmentVars
	Tags                          []string               // Tags optional tags to add
	Prefix                        string                 // Prefix to use when creating resources
	IgnoreAdds                    Exemptions             // IgnoreAdds ignore adds (creates) to these resources in Consistency and Upgrade tests
	IgnoreDestroys                Exemptions             // IgnoreDestroys ignore destroys to these resources in Consistency and Upgrade tests
	IgnoreUpdates                 Exemptions             // IgnoreUpdates ignore updates to these resources in Consistency and Upgrade tests
	ImplicitDestroy               []string               // ImplicitDestroy Remove these resources from the State file to allow implicit destroy
	ImplicitRequired              bool                   // ImplicitRequired If true the test will fail if the resource fails to be removed from the state file
	Testing                       *testing.T             `copier:"-"` // Testing The current test object
	IsUpgradeTest                 bool                   // Identifies if current test is an UPGRADE test, used for special processing
	UpgradeTestSkipped            bool                   // Informs the calling test that conditions were met to skip the upgrade test
	baseTempWorkingDir            string                 // INTERNAL variable to store the base level of temporary working directory
	ExcludeActivityTrackerRegions bool                   // Will exclude any VPC regions that already contain an Activity Tracker
	CloudInfoService              cloudInfoServiceI      // Supply if you need multiple tests to share info service and data
}

func TestOptionsDefaultWithVars(originalOptions *TestOptions) *TestOptions {

	newOptions := TestOptionsDefault(originalOptions)

	// Vars to pass into module
	varsMap := make(map[string]interface{})

	conditionalAdd(varsMap, "prefix", newOptions.Prefix, "")
	conditionalAdd(varsMap, "region", newOptions.Region, "")
	conditionalAdd(varsMap, "resource_group", newOptions.ResourceGroup, "")

	varsMap["resource_tags"] = GetTagsFromTravis()

	// Vars to pass into module
	newOptions.TerraformVars = mergeMaps(varsMap, newOptions.TerraformVars)

	return newOptions

}

func TestOptionsDefault(originalOptions *TestOptions) *TestOptions {

	newOptions, err := originalOptions.Clone()
	require.NoError(originalOptions.Testing, err)

	newOptions.Prefix = fmt.Sprintf("%s-%s", newOptions.Prefix, strings.ToLower(random.UniqueId()))

	if newOptions.DefaultRegion == "" {
		newOptions.DefaultRegion = defaultRegion
	}
	// Verify required environment variables are set - better to do this now rather than retry and fail with every attempt
	checkVariables := []string{ibmcloudApiKeyVar}
	newOptions.RequiredEnvironmentVars = GetRequiredEnvVars(newOptions.Testing, checkVariables)

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

// overwriting duplicate keys
func mergeMaps(maps ...map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}

// Adds value to map[key] only if value != compareValue
func conditionalAdd(amap map[string]interface{}, key string, value string, compareValue string) {
	if value != compareValue {
		amap[key] = value
	}
}
