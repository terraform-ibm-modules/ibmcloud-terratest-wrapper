package testhelper_test

import (
	"testing"

	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testhelper"
)

func Example_minimal() {
	// (here for compile only)
	// this is the test object supplied in a unit test call
	t := &testing.T{}

	t.Run("unit test", func(t *testing.T) {
		// set up a TestOptions data struct from scratch
		options := &testhelper.TestOptions{
			Testing:      t,                // the test object for unit test
			TerraformDir: "examples/basic", // location of example to test, relative to root of project
			TerraformVars: map[string]interface{}{
				"variable_1":   "foo",
				"variable_2":   "bar",
				"is_something": true,
				"tags":         []string{"tag1", "tag2"},
			},
		}

		// RunTestConsistency will init/apply, then plan again to verify idempotent
		terratestPlanStruct, err := options.RunTestConsistency()
		if err != nil {
			t.Fail()
		}
		if terratestPlanStruct == nil {
			t.Fail()
		}
	})
}

func Example_default() {
	// (here for compile only)
	// this is the test object supplied in a unit test call
	t := &testing.T{}

	t.Run("unit test", func(t *testing.T) {
		// create TestOptions using Default contructor. This will do several things for you:
		// * Prefix will have random string added to end
		// * Validate required OS Environment variables are set
		// * Dynamically choose best region for test, if region not supplied
		// * Set other option fields to their sensible defaults
		//
		// You call this constructor by passing it an existing TestOptions with minimal data and it will return
		// a new altered object.
		options := testhelper.TestOptionsDefault(&testhelper.TestOptions{
			Testing:            t,                      // the test object for unit test
			TerraformDir:       "examples/basic",       // location of example to test, relative to root of project
			Prefix:             "my-test",              // will have 6 char random string appended, can be used as variable to prefix test resources
			BestRegionYAMLPath: "location/of/yaml.yml", // YAML file to configure dynamic region selection (see cloudinfo/RegionData type)
			// Region: "us-south", // if you set Region, dynamic selection will be skipped
		})

		// You can set your Terraform variables using generated prefix and region from options defaults
		options.TerraformVars = map[string]interface{}{
			"variable_1":   "foo",
			"variable_2":   "bar",
			"is_something": true,
			"tags":         []string{"tag1", "tag2"},
			"region":       options.Region, // use dynamic region from Default constructor
			"prefix":       options.Prefix, // use unique prefix + random 6 character from Default constructor
		}

		// RunTestConsistency will init/apply, then plan again to verify idempotent
		terratestPlanStruct, err := options.RunTestConsistency()
		if err != nil {
			t.Fail()
		}
		if terratestPlanStruct == nil {
			t.Fail()
		}
	})
}

func Example_standard_inputs() {
	// (here for compile only)
	// this is the test object supplied in a unit test call
	t := &testing.T{}

	t.Run("unit test", func(t *testing.T) {
		// create TestOptions using Default contructor, and including some "standard" terraform input variables to your supplied list.
		// This will do several things for you:
		// * Prefix will have random string added to end
		// * Validate required OS Environment variables are set
		// * Dynamically choose best region for test, if region not supplied
		// * Set other option fields to their sensible defaults
		// * append the following standard terraform variables to the supplied array:
		//   - prefix: value is unique Prefix field
		//   - resource_group: value taken from field ResourceGroup
		//   - region: value from field Region (dynamically chosen if not supplied)
		//
		// You call this constructor by passing it an existing TestOptions with minimal data and Terraform input variables, and it will return
		// a new altered object.
		options := testhelper.TestOptionsDefaultWithVars(&testhelper.TestOptions{
			Testing:            t,                      // the test object for unit test
			TerraformDir:       "examples/basic",       // location of example to test, relative to root of project
			Prefix:             "my-test",              // will have 6 char random string appended, can be used as variable to prefix test resources
			BestRegionYAMLPath: "location/of/yaml.yml", // YAML file to configure dynamic region selection (see cloudinfo/RegionData type)
			// Region: "us-south", // if you set Region, dynamic selection will be skipped
			// TerraformVars is optional, supply here if you have extra variables beyond the standard ones,
			// and standard vars will be appended to list
			TerraformVars: map[string]interface{}{
				"extra_var_1": "foo",
				"extra_var_2": "bar",
			},
		})

		// RunTestConsistency will init/apply, then plan again to verify idempotent
		terratestPlanStruct, err := options.RunTestConsistency()
		if err != nil {
			t.Fail()
		}
		if terratestPlanStruct == nil {
			t.Fail()
		}
	})
}

func Example_upgrade_test() {
	// (here for compile only)
	// this is the test object supplied in a unit test call
	t := &testing.T{}

	t.Run("upgrade unit test", func(t *testing.T) {
		// Perform Upgrade test by first using default constructor.
		// This will do several things for you:
		// * Prefix will have random string added to end
		// * Validate required OS Environment variables are set
		// * Dynamically choose best region for test, if region not supplied
		// * Set other option fields to their sensible defaults
		//
		// You call this constructor by passing it an existing TestOptions with minimal data and it will return
		// a new altered object.
		options := testhelper.TestOptionsDefault(&testhelper.TestOptions{
			Testing:            t,                      // the test object for unit test
			TerraformDir:       "examples/basic",       // location of example to test, relative to root of project
			Prefix:             "my-test",              // will have 6 char random string appended, can be used as variable to prefix test resources
			BestRegionYAMLPath: "location/of/yaml.yml", // YAML file to configure dynamic region selection (see cloudinfo/RegionData type)
			// Region: "us-south", // if you set Region, dynamic selection will be skipped
		})

		// You can set your Terraform variables using generated prefix and region from options defaults
		options.TerraformVars = map[string]interface{}{
			"variable_1":   "foo",
			"variable_2":   "bar",
			"is_something": true,
			"tags":         []string{"tag1", "tag2"},
			"region":       options.Region, // use dynamic region from Default constructor
			"prefix":       options.Prefix, // use unique prefix + random 6 character from Default constructor
		}

		// Run upgrade test, which will do the following:
		// 1. checkout main branch
		// 2. terraform apply
		// 3. checkout original test branch
		// 4. terraform plan
		// If the plan identifies resources that would be DESTROYED, it will fail the test
		terratestPlanStruct, err := options.RunTestUpgrade()

		// there are factors in a CI pipeline run that would cause an Upgrade test to be skipped.
		// you can access the UpgradeTestSkipped boolean to find out if test was run
		if !options.UpgradeTestSkipped {
			if err != nil {
				t.Fail()
			}
			if terratestPlanStruct == nil {
				t.Fail()
			}
		}
	})
}
