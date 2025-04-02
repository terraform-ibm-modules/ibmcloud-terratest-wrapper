# Addon Testing

## Overview
The `testaddons` package provides a framework for testing IBM Cloud add-ons. It allows you to run tests against different add-on configurations and validate their behavior.
The general process for testing an add-on that the framework handles for the user is as follows:

1. Checks for local changes to be committed or pushed. This can be disabled with the `TestAddonOptions.SkipLocalChangeCheck` flag to `true`
2. Automatically determines the repository URL and branch from the current Git context (no need to specify repository URL or branch in your test configuration)
3. Creates a temporary catalog in the IBM Cloud account
4. Imports the offering/addon from the current branch to the temporary catalog
5. Creates a test project in IBM Cloud
6. Deploys the addon configuration to the project
7. Deploys dependent configurations that are marked as `on by default` unless explicitly configured otherwise through the `Enabled` flag
8. Updates the configurations with proper API key authentication using the `TF_VAR_ibmcloud_api_key` environment variable
9. Updates input configurations based on values provided in the test options
10. Runs a deploy operation and waits for completion
11. Runs an undeploy operation and waits for completion
12. Cleans up by deleting the project and catalog

During this process, the framework will log detailed information about each step, including the repository URL and branch, catalog and project IDs, and the dependencies that are deployed.

## Examples

### Example 1: Basic Test with Terraform Add-on

This example shows how to run a basic test using the `testaddons` package with a Terraform add-on. It sets up the test options, including the prefix and resource group, and then runs the addon test.

```golang
package test

import (
 "testing"

 "github.com/stretchr/testify/assert"
 "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/cloudinfo"
 "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testaddons"
)

func setupAddonOptions(t *testing.T, prefix string) *testaddons.TestAddonOptions {
 options := testaddons.TestAddonsOptionsDefault(&testaddons.TestAddonOptions{
  Testing:              t,
  Prefix:               prefix,
  ResourceGroup:        "my-project-rg",
 })

 return options
}

func TestRunTerraformAddon(t *testing.T) {
 t.Parallel()

 options := setupAddonOptions(t, "test-terraform-addon")

 // Using the specialized Terraform helper function
 options.AddonConfig = cloudinfo.NewAddonConfigTerraform(
  options.Prefix,        // prefix for unique resource naming
  "test-addon",           // offering name
  "test-flavor",          // offering flavor
  map[string]interface{}{ // inputs
   "prefix": options.Prefix,
  },
 )

 err := options.RunAddonTest()
 assert.Nil(t, err, "This should not have errored")
}
```

### Example 2: Using Stack Add-on

This example demonstrates how to configure a test with a Stack add-on.

```golang
func TestRunStackAddon(t *testing.T) {
 t.Parallel()

 options := setupAddonOptions(t, "test-stack-addon")

 // Using the specialized Stack helper function
 options.AddonConfig = cloudinfo.NewAddonConfigStack(
  options.Prefix,        // prefix for unique resource naming
  "test-addon",          // offering name
  "test-flavor",         // offering flavor
  map[string]interface{}{ // inputs
   "prefix": options.Prefix,
   "region": "us-south",
  },
 )

 err := options.RunAddonTest()
 assert.Nil(t, err, "This should not have errored")
}
```

### Example 3: Managing Dependencies

This example shows how to correctly configure dependencies of an addon by directly setting the `Dependencies` array:

```golang
func TestRunAddonWithCustomDependencyConfig(t *testing.T) {
 t.Parallel()

 options := setupAddonOptions(t, "custom-dependency-config")

 // Create the base addon config
 options.AddonConfig = cloudinfo.NewAddonConfigStack(
  "example-app",
  "standard",
  map[string]interface{}{
   "prefix": options.Prefix,
   "region": "us-south",
  },
 )

 // Set dependencies is by directly assigning to the Dependencies array
 options.AddonConfig.Dependencies = []cloudinfo.AddonConfig{
  {
   // First dependency
   OfferingName:   "database",
   OfferingFlavor: "postgresql",
   Inputs: map[string]interface{}{
    "prefix": options.Prefix,
    "plan":   "standard",
   },
   Enabled: true, // explicitly enable this dependency
  },
  {
   // Second dependency
   OfferingName:   "monitoring",
   OfferingFlavor: "basic",
   Enabled: false, // explicitly disable this dependency
  }
 }

 err := options.RunAddonTest()
 assert.Nil(t, err, "This should not have errored")
}
```


## Advanced Configuration Options

The `TestAddonOptions` structure provides several configuration options that you can use to customize your addon tests:

### Basic Configuration
- `Testing` - Required testing.T object
- `Prefix` - A unique prefix for all resources created during testing
- `ResourceGroup` - The resource group where resources will be created
- `RequiredEnvironmentVars` - Environment variables required for testing (TF_VAR_ibmcloud_api_key is required by default)

### Project Configuration
- `ProjectName` - The name of the test project (defaults to "addon" + prefix)
- `ProjectDescription` - Description of the test project
- `ProjectLocation` - The location for the project
- `ProjectDestroyOnDelete` - Whether to destroy resources when deleting the project
- `ProjectMonitoringEnabled` - Whether project monitoring is enabled
- `ProjectAutoDeploy` - Whether to automatically deploy configurations
- `ProjectEnvironments` - Define custom environments

### Catalog Configuration
- `CatalogUseExisting` - Whether to use an existing catalog
- `CatalogName` - The name of the catalog to create/use

### Testing Options
- `DeployTimeoutMinutes` - Max time to wait for deployment (defaults to 6 hours)
- `SkipTestTearDown` - Skip cleanup after tests
- `SkipUndeploy` - Skip the undeploy step
- `SkipProjectDelete` - Don't delete the project after testing
- `SkipLocalChangeCheck` - Skip checking for uncommitted local changes
- `LocalChangesIgnorePattern` - Regex patterns for files to ignore in local change check

### Hooks
The framework provides several hook points where you can inject custom code:

- `PreDeployHook` - Executed before deployment
- `PostDeployHook` - Executed after successful deployment
- `PreUndeployHook` - Executed before undeployment
- `PostUndeployHook` - Executed after successful undeployment

Example of using hooks:
```golang
options.PreDeployHook = func(options *testaddons.TestAddonOptions) error {
    // Do some extra pre configuration before deploying
    return nil
}

options.PostDeployHook = func(options *testaddons.TestAddonOptions) error {
    // Validate deployed resources or perform tests against them
    return nil
}
```
