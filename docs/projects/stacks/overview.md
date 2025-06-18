# Stack Testing

## Overview
The `testprojects` package provides a framework for testing IBM Cloud Projects stacks. It allows you to run tests against different stack configurations and validate their behavior.
The general process for testing a stack that the framework handles for the user is as follows:

1. Creates a test project in IBM Cloud with configurations based on your test options
2. Creates a stack from your stack definition file and catalog file
3. Applies any custom input configuration provided in your test
4. Deploys the stack and all its member configurations
5. Runs validation steps during and/or after deployment
6. Undeploys the stack if needed
7. Cleans up by deleting the project

During this process, the framework will log detailed information about each step, including the project ID, configuration statuses, and deployment progress.

## Examples

### Example 1: Basic Stack Test

This example shows how to run a basic test using the `testprojects` package. It sets up the test options, including the prefix and resource group, and then runs the stack test.

```golang
package test

import (
 "fmt"
 "os"
 "testing"

 "github.com/IBM/go-sdk-core/v5/core"
 project "github.com/IBM/project-go-sdk/projectv1"
 "github.com/stretchr/testify/assert"
 "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testprojects"
)

func TestStackDeployment(t *testing.T) {
 t.Parallel()

 options := testprojects.TestProjectOptionsDefault(&testprojects.TestProjectsOptions{
  Testing:              t,
  Prefix:               "test-stack",
  ProjectLocation:      "us-south", // Optional, a random region will be selected if not specified
  ProjectDestroyOnDelete: core.BoolPtr(true), // Will destroy resources when project is deleted
  ProjectMonitoringEnabled: core.BoolPtr(true), // Enable monitoring
  StackConfigurationPath: "stack_definition.json", // Path to the stack configuration file
  StackCatalogJsonPath: "ibm_catalog.json", // Path to the catalog JSON file
  DeployTimeoutMinutes: 120, // Timeout for deployment (2 hours)
 })

 // Set stack-level inputs
 options.StackInputs = map[string]interface{}{
  "resource_group_name": "default",
  "ibmcloud_api_key":    os.Getenv("TF_VAR_ibmcloud_api_key"),
 }

 // Set member-specific inputs
 options.StackMemberInputs = map[string]map[string]interface{}{
  "database": {
   "prefix": fmt.Sprintf("db-%s", options.Prefix),
   "plan":   "standard",
  },
  "compute": {
   "prefix": fmt.Sprintf("comp-%s", options.Prefix),
   "region": "us-south",
  },
 }

 err := options.RunProjectsTest()
 assert.NoError(t, err, "Stack deployment should succeed")
}
```

### Example 2: Adding Custom Hooks

This example demonstrates how to add custom hooks to perform actions before or after deployment and undeployment:

```golang
func TestStackWithHooks(t *testing.T) {
 t.Parallel()

 options := testprojects.TestProjectOptionsDefault(&testprojects.TestProjectsOptions{
  Testing:              t,
  Prefix:               "hook-stack",
  StackConfigurationPath: "stack_definition.json",
  StackCatalogJsonPath: "ibm_catalog.json",

  // Define custom hooks
  PreDeployHook: func(options *testprojects.TestProjectsOptions) error {
   // Do pre-deployment setup, like creating dependent resources
   t.Log("Executing pre-deploy setup...")
   return nil
  },

  PostDeployHook: func(options *testprojects.TestProjectsOptions) error {
   // Validate deployed resources
   t.Log("Validating deployed resources...")
   // Access the current project using options.currentProject
   // Access the current stack using options.currentStack
   return nil
  },

  PreUndeployHook: func(options *testprojects.TestProjectsOptions) error {
   // Do something before undeploy
   t.Log("Preparing for undeployment...")
   return nil
  },

  PostUndeployHook: func(options *testprojects.TestProjectsOptions) error {
   // Final cleanup after undeployment
   t.Log("Performing post-undeployment tasks...")
   return nil
  },
 })

 // Add stack inputs
 options.StackInputs = map[string]interface{}{
  "resource_group_name": "default",
 }

 err := options.RunProjectsTest()
 assert.NoError(t, err, "Stack test with hooks should succeed")
}
```

### Example 3: Using Custom Authorization

This example shows how to use a trusted profile for authorization instead of the default API key:

```golang
func TestStackWithTrustedProfile(t *testing.T) {
 t.Parallel()

 options := testprojects.TestProjectOptionsDefault(&testprojects.TestProjectsOptions{
  Testing:              t,
  Prefix:               "tp-stack",
  StackConfigurationPath: "stack_definition.json",
  StackCatalogJsonPath: "ibm_catalog.json",

  // Use trusted profile for authorization
  StackAuthorizations: &project.ProjectConfigAuth{
   Method: core.StringPtr(project.ProjectConfigAuth_Method_TrustedProfile),
   TrustedProfileId: core.StringPtr("profile-id-12345"),
   TrustedProfileTargetAccountId: core.StringPtr("account-id-12345"),
  },
 })

 options.StackInputs = map[string]interface{}{
  "resource_group_name": "default",
 }

 err := options.RunProjectsTest()
 assert.NoError(t, err, "Stack test with trusted profile should succeed")
}
```

## Configuration Options

The `TestProjectsOptions` structure provides several configuration options that you can use to customize your stack tests:

### Basic Configuration
- `Testing` - Required testing.T object
- `Prefix` - A unique prefix for all resources created during testing (a random string will be appended)
- `ResourceGroup` - The resource group where resources will be created (defaults to "Default")
- `RequiredEnvironmentVars` - Environment variables required for testing (TF_VAR_ibmcloud_api_key is required by default)

### Project Configuration
- `ProjectName` - The name of the test project (defaults to "project" + prefix)
- `ProjectDescription` - Description of the test project
- `ProjectLocation` - The location for the project (if not set, a random location will be selected)
- `ProjectDestroyOnDelete` - Whether to destroy resources when deleting the project (defaults to true)
- `ProjectMonitoringEnabled` - Whether project monitoring is enabled (defaults to true)
- `ProjectAutoDeploy` - Whether to automatically deploy configurations (defaults to true)
- `ProjectEnvironments` - Define custom environments

### Stack Configuration
- `StackConfigurationPath` - Path to the stack definition JSON file (defaults to "stack_definition.json")
- `StackCatalogJsonPath` - Path to the catalog JSON file (defaults to "ibm_catalog.json")
- `StackPollTimeSeconds` - Seconds to wait between polling stack status (defaults to 60)
- `StackAutoSync` - Whether to sync with Schematics if a member hasn't updated (defaults to false)
- `StackAutoSyncInterval` - Minutes to wait before syncing with Schematics (defaults to 20)
- `StackAuthorizations` - Authentication configuration (defaults to using API key from environment)
- `CatalogProductName` - The name of the product in the catalog (defaults to first product)
- `CatalogFlavorName` - The name of the flavor in the catalog (defaults to first flavor)

### Stack and Member Inputs
The `StackInputs` and `StackMemberInputs` maps allow you to specify configuration variables for the stack and its members, respectively. The input precedence follows a specific order to ensure that the most relevant values are applied:

1. Inputs from the current stack configuration (`StackInputs` and `StackMemberInputs`)
2. Default values from the `ibm_catalog.json`
3. Default values from the `stack_definition.json`

This means that if a variable is defined in the current stack configuration, it will take precedence over any default values. If not defined, the default value from the `ibm_catalog.json` will be used. If neither is available, the default value from the `stack_definition.json` will be applied.

#### StackMemberInputs
- **Type**: `map[string]map[string]interface{}`
- **Description**: A map where each key represents a stack member, and the value is another map containing the variables for that member.
- **Example**:
```go
options.StackMemberInputs = map[string]map[string]interface{}{
    "database": {
        "prefix": "db-test",
        "plan": "standard",
    },
    "application": {
        "instance_count": 2,
        "enable_monitoring": true,
    },
}
```

#### StackInputs
- **Type**: `map[string]interface{}`
- **Description**: A map containing variables that apply at the top stack level.
- **Example**:
```go
options.StackInputs = map[string]interface{}{
    "resource_group_name": "default",
    "region": "us-south",
    "tags": []string{"test", "stack"},
}
```

### Testing Options
- `DeployTimeoutMinutes` - Max time to wait for deployment (defaults to 6 hours)
- `SkipTestTearDown` - Skip cleanup after tests
- `SkipUndeploy` - Skip the undeploy step
- `SkipProjectDelete` - Don't delete the project after testing

### Hooks
The framework provides several hook points where you can inject custom code:

- `PreDeployHook` - Executed before deployment
- `PostDeployHook` - Executed after successful deployment
- `PreUndeployHook` - Executed before undeployment
- `PostUndeployHook` - Executed after successful undeployment

Example of using hooks:
```golang
options.PreDeployHook = func(options *testprojects.TestProjectsOptions) error {
    // Do some extra pre configuration before deploying
    return nil
}

options.PostDeployHook = func(options *testprojects.TestProjectsOptions) error {
    // Validate deployed resources or perform tests against them
    return nil
}
```
