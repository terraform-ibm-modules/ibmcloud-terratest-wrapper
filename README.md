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
If you are using a passphrase-protected SSH key, set the `SSH_PASSPHRASE` environment variable to the actual passphrase used to protect the SSH key..
___
### More examples

For more customization, see the `ibmcloud-terratest-wrapper` reference at pkg.go.dev, including the following examples:

- [Terratest examples](https://pkg.go.dev/github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testhelper#pkg-overview)
- [IBM Schematics Workspace examples](https://pkg.go.dev/github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testschematic#pkg-overview)

---
## IBM Cloud Projects

This section explains how to use the test wrapper for testing IBM Cloud Projects.

### Example Usage

The following example demonstrates how to use the `TestProjectsOptions` from the `testprojects` package to run a full test for IBM Cloud Projects.

```go
package tests

import (
 "fmt"
 "github.com/stretchr/testify/assert"
 "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testprojects"
 "os"
 "testing"
)

func TestProjectsFullTest(t *testing.T) {
 options := testprojects.TestProjectOptionsDefault(&testprojects.TestProjectsOptions{
  Testing:              t,
  Prefix:               "mock-stack",
  ProjectLocation:      "us-south", // By default, a random region is selected
  ProjectDestroyOnDelete: core.BoolPtr(true), // By default, the members are destroyed on Project delete
  ProjectMonitoringEnabled: core.BoolPtr(true), // By default, monitoring is enabled
  StackConfigurationPath: "stack_definition.json", // Path to the stack configuration file, by default it is stack_definition.json
  StackCatalogJsonPath: "ibm_catalog.json", // Path to the stack catalog JSON file, by default it is ibm_catalog.json
  StackAutoSync:        true, // By default, auto sync is disabled. This is for emergencies only. If set to true, a sync with Schematics will be executed if the member has not updated before the `StackAutoSyncInterval`.
  StackAutoSyncInterval: 20,  // By default, the interval is 20 minutes
  StackAuthorizations: &project.ProjectConfigAuth{ // By default, the API Key is used with the TF_VAR_ibmcloud_api_key environment variable
   ApiKey: core.StringPtr(os.Getenv("TF_VAR_ibmcloud_api_key")),
   Method: core.StringPtr(project.ProjectConfigAuth_Method_ApiKey),
  },
  DeployTimeoutMinutes: 360, // By default, the timeout is 6 hours
  PreDeployHook: func(options *testprojects.TestProjectsOptions) error {
   // Custom code to run before stack deploy
   return nil
  },
  PostDeployHook: func(options *testprojects.TestProjectsOptions) error {
   // Custom code to run after stack deploy
   return nil
  },
  PreUndeployHook: func(options *testprojects.TestProjectsOptions) error {
   // Custom code to run before stack undeploy
   return nil
  },
  PostUndeployHook: func(options *testprojects.TestProjectsOptions) error {
   // Custom code to run after stack undeploy
   return nil
  },
 })

 options.StackMemberInputs = map[string]map[string]interface{}{
  "primary-da": {
   "prefix": fmt.Sprintf("p%s", options.Prefix),
  },
  "secondary-da": {
   "prefix": fmt.Sprintf("s%s", options.Prefix),
  },
 }
 options.StackInputs = map[string]interface{}{
  "resource_group_name": "default",
  "ibmcloud_api_key":    os.Getenv("TF_VAR_ibmcloud_api_key"),
 }

 err := options.RunProjectsTest()
 if assert.NoError(t, err) {
  t.Log("TestProjectsFullTest Passed")
 } else {
  t.Error("TestProjectsFullTest Failed")
 }
}
```

## Explanation of `TestProjectsOptions`

#### Prefix
- **Type**: `string`
- **Description**: This prefix is added to resource names to easily identify the project.


#### ProjectLocation
- **Type**: `string`
- **Description**: The location of the project. If not set, a random location will be selected this is recommended.
- **Default**: Random location is selected

#### ProjectDestroyOnDelete
- **Type**: `*bool`
- **Description**: If set to `true`, the project will be destroyed when deleted.
- **Default**: `true`

#### ProjectMonitoringEnabled
- **Type**: `*bool`
- **Description**: If set to `true`, monitoring will be enabled for the project.
- **Default**: `true`

#### ProjectAutoDeploy
- **Type**: `*bool`
- **Description**: If set to `true`, the project will be automatically deployed. It is recommended to set this to `true` (default) for most tests.
- **Default**: `true`

#### StackConfigurationPath
- **Type**: `string`
- **Description**: Path to the configuration file that will be used to create the stack.
- **Default**: `stack_definition.json`

#### StackCatalogJsonPath
- **Type**: `string`
- **Description**: Path to the JSON file containing the stack catalog.
- **Default**: `ibm_catalog.json`

#### StackAutoSync
- **Type**: `bool`
- **Description**: If set to `true`, a sync with Schematics will be executed if the member has not updated before the `StackAutoSyncInterval`. This is for emergencies only and not recommended.
- **Default**: `false`

#### StackAutoSyncInterval
- **Type**: `int`
- **Description**: The number of minutes to wait before syncing with Schematics if the state has not updated. Default is 20 minutes.
- **Default**: 20

#### StackAuthorizations
- **Type**: `*project.ProjectConfigAuth`
- **Description**: The authorizations to use for the project. If not set, the default will be to use the `TF_VAR_ibmcloud_api_key` environment variable. Can be used to set Trusted Profile or API Key.
- **Default**: API Key with the value from environment variable `TF_VAR_ibmcloud_api_key`

### Stack and Member Inputs

The `StackInputs` and `StackMemberInputs` maps allow you to specify configuration variables for the stack and its members, respectively.
The input precedence follows a specific order to ensure that the most relevant values are applied.
The order of precedence is as follows:
 - Inputs from the current stack configuration `StackInputs` and `StackMemberInputs`
 - Default values from the `ibm_catalog.json`
 - Default values from the `stack_definition.json`

This means that if a variable is defined in the current stack configuration, it will take precedence over any default values.
If it is not defined, the default value from the `ibm_catalog.json` will be used.
If neither is available, the default value from the `stack_definition.json` will be applied.
This hierarchy ensures that the most specific and relevant configuration is always used.

#### StackMemberInputs
- **Type**: `map[string]map[string]interface{}`
- **Description**: A map where each key represents a stack member, and the value is another map containing the variables for that member.
- **Example**:
    ```go
    options.StackMemberInputs = map[string]map[string]interface{}{
        "primary-da": {
            "input1": "value1",
            "input2": 2,
        },
        "secondary-da": {
            "input1": "value1",
            "input2": 2,
        },
    }
    ```

#### StackInputs
- **Type**: `map[string]interface{}`
- **Description**: A map containing variables that apply at the top stack level.
- **Example**:
    ```go
    options.StackInputs = map[string]interface{}{
        "input1": "value1",
        "input2": 2,
    }
    ```

#### DeployTimeoutMinutes
- **Type**: `int`
- **Description**: The number of minutes to wait for the stack to deploy. Also used for undeploy. Default is 6 hours. This should be set to a reasonable value for the test, the lowest maximum deploy time.
- **Recommended**: Yes

### Hooks

Hooks allow you to inject custom code into the test process. Here are the available hooks:

- **PreDeployHook**: Called before the deploy.
- **PostDeployHook**: Called after the deploy.
- **PreUndeployHook**: Called before the undeploy. If this fails, the undeploy will continue.
- **PostUndeployHook**: Called after the undeploy.

#### Example of Setting a Hook
```go
options.PreDeployHook = func(options *TestProjectsOptions) error {
    // Custom code to run before deploy
    return nil
}
```
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
