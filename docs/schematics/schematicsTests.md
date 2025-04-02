# Schematics Testing

## Overview
The `testschematic` package provides a framework for testing IBM Cloud Terraform modules using IBM Cloud Schematics Workspaces. This allows you to test your Terraform code in a fully managed IBM Cloud environment without needing to install and configure Terraform locally.

The general process for testing a Terraform module in Schematics that the framework handles for the user is as follows:

1. Creates a test workspace in IBM Cloud Schematics
2. Creates and uploads a TAR file of your Terraform project to the workspace
3. Configures the workspace with your test variables
4. Runs PLAN/APPLY/DESTROY steps on the workspace to provision and destroy resources
5. Checks consistency by running an additional PLAN after APPLY and checking for unexpected resource changes
6. Deletes the test workspace

The framework also supports upgrade testing, which allows you to verify that changes in a PR branch do not cause unexpected resource destruction when applied to existing infrastructure.

## Examples

### Example 1: Basic Schematics Test

This example shows how to run a basic test using the `testschematic` package. It sets up the test options, including the required file patterns and variables, and then runs the test.

```golang
package test

import (
 "testing"

 "github.com/stretchr/testify/assert"
 "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testschematic"
)

func TestRunBasicInSchematic(t *testing.T) {
 t.Parallel()

 options := testschematic.TestSchematicOptionsDefault(&testschematic.TestSchematicOptions{
  Testing:            t,                      // the test object for unit test
  Prefix:             "my-test",              // will have 6 char random string appended
  BestRegionYAMLPath: "location/of/yaml.yml", // YAML file to configure dynamic region selection
  // Supply filters in order to build TAR file to upload to schematics
  TarIncludePatterns: []string{"*.tf", "scripts/*.sh", "examples/basic/*.tf"},
  // Directory within the TAR where Terraform will execute
  TemplateFolder:    "examples/basic",
  // Delete the workspace if the test fails (false keeps it for debugging)
  DeleteWorkspaceOnFail: false,
 })

 // Set up the schematic workspace variables
 options.TerraformVars = []testschematic.TestSchematicTerraformVar{
  {Name: "ibmcloud_api_key", Value: options.RequiredEnvironmentVars["TF_VAR_ibmcloud_api_key"], DataType: "string", Secure: true},
  {Name: "ibm_region", Value: options.Region, DataType: "string"},
  {Name: "resource_group_name", Value: "default", DataType: "string"},
  {Name: "prefix", Value: options.Prefix, DataType: "string"},
  {Name: "do_something", Value: true, DataType: "bool"},
  {Name: "tags", Value: []string{"test", "schematic"}, DataType: "list(string)"},
 }

 // Run the test
 err := options.RunSchematicTest()
 assert.NoError(t, err, "Schematics test should complete without errors")
}
```

### Example 2: Adding Custom Hooks

This example demonstrates how to add custom hooks to perform actions before or after resource deployment and destruction:

```golang
func TestRunSchematicWithHooks(t *testing.T) {
 t.Parallel()

 options := testschematic.TestSchematicOptionsDefault(&testschematic.TestSchematicOptions{
  Testing:             t,
  Prefix:              "hook-test",
  TarIncludePatterns:  []string{"*.tf", "modules/**/*.tf"},
  TemplateFolder:      "examples/complex",

  // Define custom hooks
  PreApplyHook: func(options *testschematic.TestSchematicOptions) error {
   // Execute code before the APPLY step
   t.Log("Executing pre-apply setup...")
   return nil
  },

  PostApplyHook: func(options *testschematic.TestSchematicOptions) error {
   // Execute code after successful APPLY
   t.Log("Validating deployed resources...")

   // Access terraform outputs if needed
   outputs := options.LastTestTerraformOutputs
   if outputs != nil {
    t.Logf("Found output: %v", outputs["example_output"])
   }
   return nil
  },

  PreDestroyHook: func(options *testschematic.TestSchematicOptions) error {
   // Execute code before the DESTROY step
   t.Log("Preparing for resource teardown...")
   return nil
  },

  PostDestroyHook: func(options *testschematic.TestSchematicOptions) error {
   // Execute code after successful DESTROY
   t.Log("Performing post-destruction cleanup...")
   return nil
  },
 })

 options.TerraformVars = []testschematic.TestSchematicTerraformVar{
  {Name: "ibmcloud_api_key", Value: options.RequiredEnvironmentVars["TF_VAR_ibmcloud_api_key"], DataType: "string", Secure: true},
  {Name: "ibm_region", Value: options.Region, DataType: "string"},
 }

 err := options.RunSchematicTest()
 assert.NoError(t, err)
}
```

### Example 3: Upgrade Testing

This example shows how to run an upgrade test to verify that changes in a PR branch don't cause unexpected resource destruction:

```golang
func TestRunSchematicUpgrade(t *testing.T) {
 t.Parallel()

 options := testschematic.TestSchematicOptionsDefault(&testschematic.TestSchematicOptions{
  Testing:              t,
  Prefix:               "upgrade-test",
  TarIncludePatterns:   []string{"*.tf", "modules/**/*.tf"},
  TemplateFolder:       "examples/basic",
  // If true, will run 'apply' after upgrade plan consistency check
  CheckApplyResultForUpgrade: true,
 })

 options.TerraformVars = []testschematic.TestSchematicTerraformVar{
  {Name: "ibmcloud_api_key", Value: options.RequiredEnvironmentVars["TF_VAR_ibmcloud_api_key"], DataType: "string", Secure: true},
  {Name: "ibm_region", Value: options.Region, DataType: "string"},
  {Name: "prefix", Value: options.Prefix, DataType: "string"},
 }

 err := options.RunSchematicUpgradeTest()
 if !options.UpgradeTestSkipped {
  assert.NoError(t, err, "Upgrade test should complete without errors")
 }
}
```

### Example 4: Private Git Repository Access

If your Terraform code references modules in private Git repositories, you can provide netrc credentials for authentication:

```golang
func TestWithPrivateRepoAccess(t *testing.T) {
 t.Parallel()

 options := testschematic.TestSchematicOptionsDefault(&testschematic.TestSchematicOptions{
  Testing:            t,
  Prefix:             "private-repo-test",
  TarIncludePatterns: []string{"*.tf"},
 })

 // Add credentials for private Git repos
 options.AddNetrcCredential("github.com", "github-username", options.RequiredEnvironmentVars["GITHUB_TOKEN"])
 options.AddNetrcCredential("bitbucket.com", "bit-username", options.RequiredEnvironmentVars["BITBUCKET_TOKEN"])

 options.TerraformVars = []testschematic.TestSchematicTerraformVar{
  {Name: "ibmcloud_api_key", Value: options.RequiredEnvironmentVars["TF_VAR_ibmcloud_api_key"], DataType: "string", Secure: true},
  {Name: "ibm_region", Value: options.Region, DataType: "string"},
 }

 err := options.RunSchematicTest()
 assert.NoError(t, err)
}
```

## Configuration Options

The `TestSchematicOptions` structure provides several configuration options that you can use to customize your Schematics tests:

### Basic Configuration
- `Testing` - Required testing.T object
- `Prefix` - A unique prefix for all resources created during testing (a random string will be appended)
- `RequiredEnvironmentVars` - Environment variables required for testing (TF_VAR_ibmcloud_api_key is required by default)
- `TarIncludePatterns` - List of file patterns to include in the TAR file uploaded to Schematics (defaults to "*.tf" in project root)

### Workspace Configuration
- `TemplateFolder` - Directory within the TAR file where Terraform should execute (defaults to ".")
- `TerraformVars` - List of Terraform variables to set in the Schematics workspace
- `TerraformVersion` - Specific Terraform version to use in the workspace (format: "terraform_v1.x")
- `WorkspaceLocation` - Region for the Schematics workspace (if not set, a random location will be selected)
- `Tags` - List of tags to apply to the workspace
- `WorkspaceEnvVars` - Additional environment variables to set in the workspace
- `SchematicsApiURL` - Base URL of the Schematics REST API (defaults to appropriate endpoint for the chosen region)
- `DeleteWorkspaceOnFail` - Whether to delete the workspace if the test fails (defaults to false)
- `PrintAllSchematicsLogs` - Whether to print all Schematics job logs regardless of success/failure (defaults to false)

### Git Repository Access
- `NetrcSettings` - List of credentials for accessing private Git repositories
- `BaseTerraformRepo` - The URL of the origin git repository for upgrade tests
- `BaseTerraformBranch` - The branch name of the main origin branch for upgrade tests

### Testing Control
- `WaitJobCompleteMinutes` - Minutes to wait for Schematics jobs to complete (defaults to 120)
- `SkipTestTearDown` - Skip test teardown (both resource destroy and workspace deletion)
- `BestRegionYAMLPath` - Path to YAML file configuring dynamic region selection
- `Region` - Specific region to use (if set, dynamic selection will be skipped)
- `CheckApplyResultForUpgrade` - For upgrade tests, whether to perform a final apply after consistency check

### Consistency Checking
- `IgnoreAdds` - List of resource names to ignore when checking for added resources in consistency checks
- `IgnoreUpdates` - List of resource names to ignore when checking for updated resources in consistency checks
- `IgnoreDestroys` - List of resource names to ignore when checking for destroyed resources in consistency checks

### Hooks
The framework provides several hook points where you can inject custom code:

- `PreApplyHook` - Executed before the APPLY step
- `PostApplyHook` - Executed after successful APPLY
- `PreDestroyHook` - Executed before the DESTROY step
- `PostDestroyHook` - Executed after successful DESTROY

## Working with Variables

The `TerraformVars` field accepts a list of `TestSchematicTerraformVar` objects with the following properties:

- `Name` - The name of the Terraform variable
- `Value` - The value to set for the variable
- `DataType` - The Terraform data type of the variable (e.g., "string", "bool", "list(string)", "map(any)")
- `Secure` - Whether the variable should be hidden in logs and UIs

Example of setting different variable types:

```go
options.TerraformVars = []testschematic.TestSchematicTerraformVar{
 {Name: "string_var", Value: "hello", DataType: "string", Secure: false},
 {Name: "bool_var", Value: true, DataType: "bool", Secure: false},
 {Name: "number_var", Value: 42, DataType: "number", Secure: false},
 {Name: "list_var", Value: []string{"item1", "item2"}, DataType: "list(string)", Secure: false},
 {Name: "map_var", Value: map[string]interface{}{
  "key1": "value1",
  "key2": 42,
  "key3": true,
 }, DataType: "map(any)", Secure: false},
 {Name: "api_key", Value: "sensitive-value", DataType: "string", Secure: true},
}
```

## Upgrade Testing Details

The `RunSchematicUpgradeTest` method performs the following steps:

1. Creates a test workspace in Schematics
2. Configures it for the main branch of the repo with your test variables
3. Runs PLAN/APPLY steps to provision resources using the main branch code
4. Switches the workspace to your current PR branch
5. Runs a PLAN to check for resource changes
6. Analyzes the plan for unexpected resource destruction
7. Optionally applies the changes (if `CheckApplyResultForUpgrade` is true)
8. Cleans up resources and deletes the workspace

Upgrade tests are automatically skipped if commits in the PR include "BREAKING CHANGE" or "SKIP UPGRADE TEST" in their messages. This behavior can be overridden by including "UNSKIP UPGRADE TEST" in a commit message.

Base repo and branch detection is handled automatically for most cases, but you can manually set them with environment variables:
- `BASE_TERRAFORM_REPO` - URL of the base repository
- `BASE_TERRAFORM_BRANCH` - Branch name of the main branch

## Error Handling and Debugging

When a test fails, several debugging features are available:

1. Set `DeleteWorkspaceOnFail` to `false` to keep the workspace after test failure for manual inspection
2. Set `PrintAllSchematicsLogs` to `true` to see all logs from Schematics jobs
3. Use the environment variable `DO_NOT_DESTROY_ON_FAILURE=true` to keep the resources deployed after failure

For successful tests, the Terraform outputs are available in the `LastTestTerraformOutputs` field of the options object, making them accessible to your test validation code.
