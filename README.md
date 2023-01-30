# IBM Cloud Terratest wrapper
[![Build Status](https://github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/actions/workflows/ci.yml/badge.svg)](https://github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/actions/workflows/ci.yml)
[![semantic-release](https://img.shields.io/badge/%20%20%F0%9F%93%A6%F0%9F%9A%80-semantic--release-e10079.svg)](https://github.com/semantic-release/semantic-release)
[![pre-commit](https://img.shields.io/badge/pre--commit-enabled-brightgreen?logo=pre-commit&logoColor=white)](https://github.com/pre-commit/pre-commit)

This Go module provides helper functions as a wrapper around the [Terratest](https://terratest.gruntwork.io/) library so that tests can be created quickly and consistently.

## Contributing
To contribute to this project, read through the documentation: https://terraform-ibm-modules.github.io/documentation/#/

### Setting up your local development environment

This Go project uses submodules, pre-commit hooks, and some other tools that are common across all projects in this org. Before you start contributing to the project, follow these steps to set up your environment: https://terraform-ibm-modules.github.io/documentation/#/local-dev-setup

### Running tests

To run unit tests for all the packages in this module, you can use the `go test` command, either for a single package or all packages.

```bash
# run single package tests
go test -v ./cloudinfo
```

```bash
# run all packages tests, skipping template tests that exist in common-dev-assets
go test -v $(go list ./... | grep -v /common-dev-assets/)
```

### Publishing
Publishing is handled automatically by the merge pipeline and Semantic Versioning automation, which creates a new Github release.

## Usage

### Basic Usage Steps
This golang project can be used in IBM Cloud Terraform projects to simplify unit tests performed by the [Terratest](https://terratest.gruntwork.io/) library. This is typically done by performing the following steps:
1. [Create a golang module](https://go.dev/doc/tutorial/create-module) in your Terraform project
1. [Import](https://go.dev/doc/tutorial/call-module-code) this ibmcloud-terratest-wrapper module into your new module
1. [Add a unit test](https://go.dev/doc/tutorial/add-a-test) in your Terraform golang module
1. Initialize a [testhelper/TestOptions](https://pkg.go.dev/github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testhelper#TestOptions) object with appropriate values
1. Further configure TestOptions using [default constructor](https://pkg.go.dev/github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testhelper#TestOptionsDefault) (optional but recommended)
1. Call one of the `RunTest...()` methods of the TestOptions object and check the return

### Dynamic IBM Cloud Region Support
This test framework has support for selecting an IBM Region for your test dynamically, at run time. Currently this is done by selecting a VPC supported region available to your account that contains the least amount of currently active VPCs. You can access this feature in two ways:
1. Use the [testhelper/GetBestVpcRegion()](https://pkg.go.dev/github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testhelper#GetBestVpcRegion)
1. Use a [default constructor](https://pkg.go.dev/github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testhelper#TestOptionsDefault) which will call `GetBestVpcRegion()` and assign result to the `Region` field, if it had not been set previously.

#### Dynamic IBM Cloud Region Selection Configuration
If the parameter `prefsFilePath` is not passed (empty) to the `GetBestVpcRegion()` function, or the field `TestOptions.BestRegionYAMLPath` is not set when using the default constructor, all possible VPC regions available to the account will be queried in a non-sequenced order. In order to restrict the region list to query, and to assign a priority for selection, you can supply a YAML file to the function using `prefsFilePath`. The format of this YAML file is:
```yaml
---
- name: us-east
  useForTest: true
  testPriority: 1
- name: eu-de
  useForTest: true
  testPriority: 2
```

### Basic Consistency Example
Terraform unit test to check consistency of an example located in project under examples/basic directory:
```go
func TestRunBasic(t *testing.T) {
	t.Parallel()

	options := testhelper.TestOptionsDefault(&testhelper.TestOptions{
        Testing:            t,                      // the test object for unit test
        TerraformDir:       "examples/basic",       // location of example to test
        Prefix:             "my-test",              // will have 6 char random string appended
        BestRegionYAMLPath: "location/of/yaml.yml", // YAML file to configure dynamic region selection
        // Region: "us-south", // if you set Region, dynamic selection will be skipped
    })

    options.TerraformVars = map[string]interface{}{
        "variable_1":   "foo",
        "resource_prefix": options.Prefix,
        "ibm_region": options.Region,
    }

    // idempotent test
    output, err := options.RunTestConsistency()
    assert.Nil(t, err, "This should not have errored")
    assert.NotNil(t, output, "Expected some output")
}
```

### IBM Cloud Schematic Support
If you would like to run a Terraform unit test inside of the IBM Schematic Terraform service, this can be done in a similar manner using the `testschematic` package. The setup is similar to a basic Terraform test, but with some different options related to schematics, such as input variables being handled differently.

To set up a test in schematics, follow the basic steps above to set up a unit test, then:
1. Initialize a [testschematic/TestSchematicOptions](https://pkg.go.dev/github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testschematic#TestSchematicOptions) object with appropriate values
1. Further configure TestSchematicOptions using [default constructor](https://pkg.go.dev/github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testschematic#TestSchematicOptionsDefault) (optional but recommended)
1. Call `RunSchematicTest()` method of the TestSchematicOptions object and check the return

Example:
```go
func TestRunBasicInSchematic(t *testing.T) {
	t.Parallel()

	options := testschematic.TestSchematicOptionsDefault(&testschematic.TestSchematicOptions{
        Testing:            t,                      // the test object for unit test
        Prefix:             "my-test",              // will have 6 char random string appended
        BestRegionYAMLPath: "location/of/yaml.yml", // YAML file to configure dynamic region selection
        // Region: "us-south", // if you set Region, dynamic selection will be skipped
        TarIncludePatterns: []string{"*.tf", "scripts/*.sh", "examples/basic/*.tf"},
        TemplateFolder: "examples/basic",
    })

    options.TerraformVars = []testschematic.TestSchematicTerraformVar{
        {Name: "ibmcloud_api_key", Value: options.RequiredEnvironmentVars["TF_VAR_ibmcloud_api_key"], DataType: "string", Secure: true},
        {Name: "ibm_region", Value: options.Region, DataType: "string"},
    }

    // idempotent test
	output, err := options.RunSchematicTest()
	assert.Nil(t, err, "This should not have errored")
	assert.NotNil(t, output, "Expected some output")
}
```

### Running a Module Version Upgrade Test
You can run a special upgrade test that will verify that the current code being tested (usually your pull request branch) will not destroy infrastructure when applied to resources created by a previous version (usually master branch). You can call this type of test by using the `RunTestUpgrade()` method.

This kind of test can be important to run for Terraform Module projects, as you want to find out what the experience for consumers of that module will be when they upgrade from the previous version to the pending version. If key resources are destroyed during an upgrade, even if they are replaced, this could negatively impact consumer environments.

The `RunTestUpgrade()` will perform the following steps:
1. Copy current project directory, including ".git" checkout repository, into temporary location to be used by the test
1. Store the git ref of currently checked out branch (usually a PR merge branch)
1. Checkout main or master branch
1. Run Terraform Apply with idempotent check
1. Checkout original branch from original stored git ref (PR branch)
1. Run Terraform Plan
1. Analyze Plan file for consistency check


Example:
```go
output, err := options.RunTestUpgrade()
if !options.UpgradeTestSkipped {
    assert.Nil(t, err, "This should not have errored")
    assert.NotNil(t, output, "Expected some output")
}
```

### More Examples
More examples, including some advanced features, can be found in the official Go Pkg documentation:

[Basic Terraform Test Examples](https://pkg.go.dev/github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testhelper#pkg-examples)

[Schematic Terraform Test Examples](https://pkg.go.dev/github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testschematic#pkg-examples)
