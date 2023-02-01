# IBM Cloud Terratest wrapper

[![Incubating (Not yet consumable)](https://img.shields.io/badge/status-Incubating%20(Not%20yet%20consumable)-red)](https://terraform-ibm-modules.github.io/documentation/#/badge-status)
[![Build Status](https://github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/actions/workflows/ci.yml/badge.svg)](https://github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/actions/workflows/ci.yml)
[![pre-commit](https://img.shields.io/badge/pre--commit-enabled-brightgreen?logo=pre-commit&logoColor=white)](https://github.com/pre-commit/pre-commit)
[![latest release](https://img.shields.io/github/v/release/terraform-ibm-modules/terraform-ibm-module-template?logo=GitHub&sort=semver)](https://github.com/terraform-ibm-modules/terraform-ibm-module-template/releases/latest)
[![Renovate enabled](https://img.shields.io/badge/renovate-enabled-brightgreen.svg)](https://renovatebot.com/)
[![semantic-release](https://img.shields.io/badge/%20%20%F0%9F%93%A6%F0%9F%9A%80-semantic--release-e10079.svg)](https://github.com/semantic-release/semantic-release)

This Go module provides helper functions as a wrapper around the [Terratest](https://terratest.gruntwork.io/) library so that tests can be created quickly and consistently. 

For more information about the code, see the pkg.go.dev repository and the GitHub repo for `ibmcloud-terratest-wrapper` at https://pkg.go.dev/github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/.

## Setup

Use this Go project with IBM Cloud Terraform projects to simplify testing with the [Terratest](https://terratest.gruntwork.io/) library. Follow these steps to set up the tests:

1 . [Create a Go module](https://go.dev/doc/tutorial/create-module) in your Terraform project.
1.  [Import](https://go.dev/doc/tutorial/call-module-code) this ibmcloud-terratest-wrapper module into your new module.
1.  [Add a unit test](https://go.dev/doc/tutorial/add-a-test) in your Terraform Go module.
1.  Initialize a [testhelper/TestOptions](https://pkg.go.dev/github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testhelper#TestOptions) object with appropriate values.

    You can then configure the `TestOptions` object by using the [default constructor](https://pkg.go.dev/github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testhelper#TestOptionsDefault).
1.  Call one of the `RunTest...()` methods of the `TestOptions` object and check the results.


## Running tests

To run unit tests for all the packages in this module, use the `go test` command, either for a single package or all packages.

```bash
# run single package tests
go test -v ./cloudinfo
```

```bash
# run all packages tests, skipping template tests that exist in common-dev-assets
go test -v $(go list ./... | grep -v /common-dev-assets/)
```

## Region selection at run time

This test framework supports selecting an IBM region for your test at run time. Select a VPC-supported region that is available to your account and that contains the least number of active VPCs. You can access this feature in two ways:

- Use the [testhelper/GetBestVpcRegion()](https://pkg.go.dev/github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testhelper#GetBestVpcRegion).
- Use a [default constructor](https://pkg.go.dev/github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testhelper#TestOptionsDefault), which calls `GetBestVpcRegion()` and assigns a result to the `Region` field, if not already set.

### Dynamic region selection

All VPC regions that are available to your account are queried in a nonsequential order if the parameter `prefsFilePath` is not passed to the `GetBestVpcRegion()` function (in other words, is empty), or if the field `TestOptions.BestRegionYAMLPath` is not set when you use the default constructor.

To restrict the query and assign a priority to the regions, supply a YAML file to the function by using the `prefsFilePath` parameter. Use the following YAML format:

```yaml
---
- name: us-east
  useForTest: true
  testPriority: 1
- name: eu-de
  useForTest: true
  testPriority: 2
```

## Examples
### Example to check basic consistency

The following example checks the consistency of an example in the `examples/basic` directory:

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

### Run a test inside IBM Cloud Schematics

To run a test inside IBM Schematics, use the `testschematic` package. The setup is similar to a basic Terraform test, but with some differences that are related to schematics, such as how input variables are handled.

To set up a test in schematics, follow these steps:

1.  [Set up a unit test](#setup) as shown earlier.
1.  Initialize a [testschematic/TestSchematicOptions](https://pkg.go.dev/github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testschematic#TestSchematicOptions) object with appropriate values.

    You can configure TestSchematicOptions by using the [default constructor](https://pkg.go.dev/github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testschematic#TestSchematicOptionsDefault).
1.  Call the `RunSchematicTest()` method of the `TestSchematicOptions` object and check the results.

#### Example for IBM Schematics

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

### Test for version upgrade

You can run a special upgrade test that verifies that the tested code (usually your pull request branch) will not destroy infrastructure when applied to resources that were created by a previous version (for example, the `main` branch). Call this test by using the `RunTestUpgrade()` method.

This test is important for Terraform module projects because you generally don't want to destroy key resources in an upgrade, even if the resources are replaced.

The `RunTestUpgrade()` method completes the following steps:

1.  Copies the current project directory, including the hidden `.git` repository, into a temporary location.
1.  Stores the Git references of the checked out branch (usually a PR merge branch).
1.  Checks out the `main` branch.
1.  Runs `terraform apply` with a check to make sure that the module is idempotent.
1.  Checks out the original branch from the stored Git reference (for example, the PR branch).
1.  Runs `terraform plan`.
1.  Analyzes the plan file for consistency.

#### Example version upgrade test

```go
output, err := options.RunTestUpgrade()
if !options.UpgradeTestSkipped {
    assert.Nil(t, err, "Unexpected error")
    assert.NotNil(t, output, "Expected output")
}
```

### More examples

For more customization examples, see the pkg.go.dev repository and the GitHub repo for `ibmcloud-terratest-wrapper`.

- [Terratest examples](https://pkg.go.dev/github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testhelper#pkg-overview)
- [Schematics Workspace examples](https://pkg.go.dev/github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testschematic#pkg-overview)

<!-- Leave this section as is so that your module has a link to local development environment set up steps for contributors to follow -->
## Contributing

You can report issues and request features for this module in GitHub issues in the module repo. See [Report an issue or request a feature](https://github.com/terraform-ibm-modules/.github/blob/main/.github/SUPPORT.md).

To set up your local development environment, see [Local development setup](https://terraform-ibm-modules.github.io/documentation/#/local-dev-setup) in the project documentation.
<!-- Source for this readme file: https://github.com/terraform-ibm-modules/common-dev-assets/tree/main/module-assets/ci/module-template-automation -->
<!-- END CONTRIBUTING HOOK -->
