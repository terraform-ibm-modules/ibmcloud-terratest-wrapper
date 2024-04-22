# IBM Cloud Terratest wrapper

[![pre-commit](https://img.shields.io/badge/pre--commit-enabled-brightgreen?logo=pre-commit&logoColor=white)](https://github.com/pre-commit/pre-commit)
[![latest release](https://img.shields.io/github/v/release/terraform-ibm-modules/ibmcloud-terratest-wrapper?logo=GitHub&sort=semver)](https://github.com/terraform-ibm-modules/terraform-ibm-module-template/releases/latest)
[![Renovate enabled](https://img.shields.io/badge/renovate-enabled-brightgreen.svg)](https://renovatebot.com/)
[![semantic-release](https://img.shields.io/badge/%20%20%F0%9F%93%A6%F0%9F%9A%80-semantic--release-e10079.svg)](https://github.com/semantic-release/semantic-release)
[![Go reference](https://pkg.go.dev/badge/github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/.svg)](https://pkg.go.dev/github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper#section-directories)

This Go module provides helper functions as a wrapper around the [Terratest](https://terratest.gruntwork.io/) library.

The project helps to simplify and standardize your Terratest unit tests. It is used by default by Terraform modules in this GitHub organization. For more information about how the tests are used in the IBM Cloud Terraform modules project, see [validation tests](https://terraform-ibm-modules.github.io/documentation/#/tests) in the project docs.

## Test your own projects

You can also use this Go project with your own Terraform projects for IBM Cloud.

<a name="setup"></a>

### Adding this wrapper to your project


The following procedure is a typical way to add this wrapper to your Terraform module for IBM Cloud.

1.  [Create a Go module](https://go.dev/doc/tutorial/create-module) in your Terraform project.
1.  [Import](https://go.dev/doc/tutorial/call-module-code) this ibmcloud-terratest-wrapper module into your new module.
1.  [Add a unit test](https://go.dev/doc/tutorial/add-a-test) in your Terraform Go module.
1.  Initialize a [testhelper/TestOptions](https://pkg.go.dev/github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testhelper#TestOptions) object with appropriate values.

    You can then configure the `TestOptions` object by using the [default constructor](https://pkg.go.dev/github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testhelper#TestOptionsDefault).
1.  Call one of the `RunTest...()` methods of the `TestOptions` object and check the results.

## Region selection at run time

This test framework supports runtime selection of an IBM region for your test. Select a VPC-supported region that is available to your account and that contains the least number of active VPCs.

You can access this feature in two ways:

- Use the [testhelper/GetBestVpcRegion()](https://pkg.go.dev/github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testhelper#GetBestVpcRegion).
- Use a [default constructor](https://pkg.go.dev/github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testhelper#TestOptionsDefault), which calls `GetBestVpcRegion()` and assigns a result to the `Region` field, if not already set.

### Configuring runtime region selection

All VPC regions that are available to your account are queried in a nonsequential order if the parameter `prefsFilePath` is not passed to the `GetBestVpcRegion()` function (in other words, is empty), or if the field `TestOptions.BestRegionYAMLPath` is not set when you use the default constructor.

To restrict the query and assign a priority to the regions, supply a YAML file to the function by using the `prefsFilePath` parameter. Use the following format:

```yaml
---
- name: us-east
  useForTest: true
  testPriority: 1
- name: eu-de
  useForTest: true
  testPriority: 2
```
___
## Examples

<a name="testrunbasic"></a>

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
___

### Example Handling Terraform Outputs

After the test completes and teardown occurs, the state file no longer contains the outputs. To handle this situation, the last test to execute stores its outputs in `LastTestTerraformOutputs`. Use the helper function called `ValidateTerraformOutputs` to validate that the outputs exist. The function returns a list of output keys that are missing and an error message with details.

The following example checks if the output exists and contains a certain value.

```go
outputs := options.LastTestTerraformOutputs
expectedOutputs := []string{"output1", "output2"}
_, outputErr := testhelper.ValidateTerraformOutputs(outputs, expectedOutputs...)
if assert.NoErrorf(t, outputErr, "Some outputs not found or nil.") {
    assert.Equal(t, outputs["output1"].(string), "output 1")
    assert.Equal(t, outputs["output2"].(string), "output 2")
}
```
---

### OpenTofu

Enable OpenTofu with the TestOptions, then OpenTofu on the systems path will be used for the test.
```go
func TestRunBasicTofu(t *testing.T) {
	t.Parallel()

	options := testhelper.TestOptionsDefault(&testhelper.TestOptions{
        Testing:            t,                      // the test object for unit test
        EnableOpenTofu:     true,                   // enable open Tofu
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
The `TerraformBinary` can also be set directly if Terrform/OpenTofu is not in the system path. If this is set the `EnableOpenTofu` option will be ignored.
```go
func TestRunBasicTerraformBinary(t *testing.T) {
	t.Parallel()

	options := testhelper.TestOptionsDefault(&testhelper.TestOptions{
        Testing:            t,                      // the test object for unit test
        TerraformBinary:    "/custom/path/tofu",    // set the path to the Terraform binary
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
___

### Run in IBM Cloud Schematics

The code to run a test inside IBM Schematics is similar to the [basic example](#testrunbasic), but uses the `testschematic` package.

1.  Complete the steps shown earlier to [add this wrapper](#setup) to your project.
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
___
### Test a module upgrade

When a new version of your Terraform module is released, you can test whether the upgrade destroys resources. Consumers of your module might not want key resources deleted in an upgrade, even if the resources are replaced.

The following test verifies that the tested code (usually your pull request branch) will not destroy infrastructure when applied to existing resources (for example, in the `main` branch). Call this test by using the `RunTestUpgrade()` method.

The `RunTestUpgrade()` method completes the following steps:

1.  Copies the current project directory, including the hidden `.git` repository, into a temporary location.
1.  Stores the Git references of the checked out branch (usually a PR merge branch).
1.  Clones the `main` branch from the target base repository.
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


#### Notes:
**Skipping the test**

The upgrade Test checks the current commit messages and if `BREAKING CHANGE` OR `SKIP UPGRADE TEST` string found in commit messages then it will skip the upgrade test.
If the message `UNSKIP UPGRADE TEST` is found in the commit messages, it will not skip the upgrade test and will not be possible to skip the test again.

**Base repo and branch**

The upgrade test needs to pull the latest changes from the default branch of the base repo to apply them. If you are using a fork it will attempt to figure out the base repo and base branch.
If this fails in your environment, you can manually set the base repo and branch by setting the environment variables `BASE_TERRAFORM_REPO` and `BASE_TERRAFORM_BRANCH`.

**Authentication**

If authentication is required to access the base repo, the code tries to automatically figure it out, by default it will try unauthenticated for HTTPS repositories and trie use the default SSH key located at `~/.ssh/id_rsa` for SSH repositories.
If this fails it will try unauthenticated. You can manually set the `SSH_PRIVATE_KEY` environment variable to the value of your SSH private key. For HTTPS repositories, set the `GIT_TOKEN` environment variable to your Personal Access Token (PAT).
___
### More examples

For more customization, see the `ibmcloud-terratest-wrapper` reference at pkg.go.dev, including the following examples:

- [Terratest examples](https://pkg.go.dev/github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testhelper#pkg-overview)
- [IBM Schematics Workspace examples](https://pkg.go.dev/github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testschematic#pkg-overview)

---
## IBM Cloud Projects
IBM Cloud Projects support has been added but should be considered alpha. It is using pre-release APIs and may change in the future.
It is not recommended to use this feature. Breaking changes are expected and upgrade paths are not guaranteed.
___
## Contributing

You can report issues and request features for this module in [issues](/issues/new/choose) in this repo. Changes that are accepted and merged are published to the pkg.go.dev reference by the merge pipeline and semantic versioning automation, which creates a new GitHub release.

If you work at IBM, you can talk with us in the #project-goldeneye Slack channel in the IBM Cloud Platform workspace.

### Setting up your local development environment

This Go project uses submodules, pre-commit hooks, and other tools that are common across all projects in this GitHub org. Follow the steps in [Local development setup](https://terraform-ibm-modules.github.io/documentation/#/local-dev-setup) to set up your local development environment.

### Running tests

To run unit tests for all the packages in this module, use the `go test` command, either for a single package or all packages.

```bash
# run single package tests
go test -v ./cloudinfo
```

```bash
# run all packages tests, skipping template tests that exist in common-dev-assets
go test -v $(go list ./... | grep -v /common-dev-assets/)
```
