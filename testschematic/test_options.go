package testschematic

import (
	"fmt"
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

type TestSchematicOptions struct {
	TarIncludePatterns      []string
	BestRegionYAMLPath      string                       // BestRegionYAMLPath Path to the yaml containing regions and weights
	DefaultRegion           string                       // DefaultRegion default region if automatic detection fails
	ResourceGroup           string                       // ResourceGroup IBM Cloud resource group to use
	Region                  string                       // Region to use
	RequiredEnvironmentVars map[string]string            // RequiredEnvironmentVars
	TerraformVars           map[string]interface{}       // TerraformVars variables to pass to terraform
	Tags                    []string                     // Tags optional tags to add
	Prefix                  string                       // Prefix to use when creating resources
	Testing                 *testing.T                   `copier:"-"` // Testing The current test object
	CloudInfoService        testhelper.CloudInfoServiceI // Supply if you need multiple tests to share info service and data
}

func TestOptionsDefaultWithVars(originalOptions *TestSchematicOptions) *TestSchematicOptions {

	newOptions := TestOptionsDefault(originalOptions)

	// Vars to pass into module
	varsMap := make(map[string]interface{})

	testhelper.ConditionalAdd(varsMap, "prefix", newOptions.Prefix, "")
	testhelper.ConditionalAdd(varsMap, "region", newOptions.Region, "")
	testhelper.ConditionalAdd(varsMap, "resource_group", newOptions.ResourceGroup, "")

	varsMap["resource_tags"] = testhelper.GetTagsFromTravis()

	// Vars to pass into module
	newOptions.TerraformVars = testhelper.MergeMaps(varsMap, newOptions.TerraformVars)

	return newOptions

}

func TestOptionsDefault(originalOptions *TestSchematicOptions) *TestSchematicOptions {

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
