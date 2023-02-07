package testschematic_test

import (
	"testing"

	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testschematic"
)

func Example_default() {
	// (here for compile only)
	// this is the test object supplied in a unit test call
	t := &testing.T{}

	t.Run("schematic unit test", func(t *testing.T) {
		// create TestOptions using Default contructor. This will do several things for you:
		// * Prefix will have random string added to end
		// * Validate required OS Environment variables are set
		// * Dynamically choose best region for test, if region not supplied
		// * Set other option fields to their sensible defaults
		//
		// You call this constructor by passing it an existing TestOptions with minimal data and it will return
		// a new altered object.
		options := testschematic.TestSchematicOptionsDefault(&testschematic.TestSchematicOptions{
			Testing:            t,                      // the test object for unit test
			Prefix:             "my-test",              // will have 6 char random string appended, can be used as variable to prefix test resources
			BestRegionYAMLPath: "location/of/yaml.yml", // YAML file to configure dynamic region selection (see cloudinfo/RegionData type)
			// Region: "us-south", // if you set Region, dynamic selection will be skipped
			// Supply filters in order to build TAR file to upload to schematics, if not supplied will default to "*.tf" in project root
			TarIncludePatterns: []string{"*.tf", "scripts/*.sh", "examples/basic/*.tf"},
			// If test fails, determine if schematic workspace is also deleted
			DeleteWorkspaceOnFail: false,
			TemplateFolder:        "examples/basic",
		})

		// Set up the schematic workspace Variablestore, including values to use
		options.TerraformVars = []testschematic.TestSchematicTerraformVar{
			{Name: "ibmcloud_api_key", Value: options.RequiredEnvironmentVars["TF_VAR_ibmcloud_api_key"], DataType: "string", Secure: true},
			{Name: "ibm_region", Value: options.Region, DataType: "string"},
			{Name: "do_something", Value: true, DataType: "bool"},
			{Name: "tags", Value: []string{"test", "schematic"}, DataType: "list(string)"},
		}

		// If needed, Git credentials can be supplied to create netrc entries in schematics to pull terraform modules from private repos
		// NOTE: token should be retrieved from trusted source and not statically coded here
		options.AddNetrcCredential("github.com", "some-user", "some-token")
		options.AddNetrcCredential("bitbucket.com", "bit-user", "bit-token")

		// run the test
		err := options.RunSchematicTest()
		if err != nil {
			t.Fail()
		}
	})
}
