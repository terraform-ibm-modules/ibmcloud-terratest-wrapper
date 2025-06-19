# Stack Testing Examples

This guide provides comprehensive examples for different stack testing scenarios using the `testprojects` package.

## Basic Examples

### Example 1: Basic Stack Test

This example shows how to run a basic test using the `testprojects` package:

```golang
package test

import (
    "fmt"
    "os"
    "testing"
    "github.com/IBM/go-sdk-core/v5/core"
    "github.com/stretchr/testify/assert"
    "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testprojects"
)

func TestStackDeployment(t *testing.T) {
    t.Parallel()

    options := testprojects.TestProjectOptionsDefault(&testprojects.TestProjectsOptions{
        Testing:                  t,
        Prefix:                   "test-stack",
        ProjectLocation:          "us-south",
        ProjectDestroyOnDelete:   core.BoolPtr(true),
        ProjectMonitoringEnabled: core.BoolPtr(true),
        StackConfigurationPath:   "stack_definition.json",
        StackCatalogJsonPath:     "ibm_catalog.json",
        DeployTimeoutMinutes:     120,
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

### Example 2: Stack Test with Hooks

This example demonstrates how to add custom hooks for additional validation:

```golang
func TestStackWithHooks(t *testing.T) {
    t.Parallel()

    options := testprojects.TestProjectOptionsDefault(&testprojects.TestProjectsOptions{
        Testing:                t,
        Prefix:                 "hook-stack",
        StackConfigurationPath: "stack_definition.json",
        StackCatalogJsonPath:   "ibm_catalog.json",
    })

    // Define custom hooks
    options.PreDeployHook = func(options *testprojects.TestProjectsOptions) error {
        t.Log("Executing pre-deploy setup...")
        return nil
    }

    options.PostDeployHook = func(options *testprojects.TestProjectsOptions) error {
        t.Log("Validating deployed resources...")
        // Access the current project using options.currentProject
        // Access the current stack using options.currentStack
        return nil
    }

    options.PreUndeployHook = func(options *testprojects.TestProjectsOptions) error {
        t.Log("Preparing for undeploy...")
        return nil
    }

    options.PostUndeployHook = func(options *testprojects.TestProjectsOptions) error {
        t.Log("Cleanup verification...")
        return nil
    }

    err := options.RunProjectsTest()
    assert.NoError(t, err, "Stack test with hooks should succeed")
}
```

## Advanced Examples

### Example 3: Stack Test with Custom Validation

```golang
func TestStackWithValidation(t *testing.T) {
    t.Parallel()

    options := testprojects.TestProjectOptionsDefault(&testprojects.TestProjectsOptions{
        Testing:                t,
        Prefix:                 "validation-stack",
        StackConfigurationPath: "stack_definition.json",
        StackCatalogJsonPath:   "ibm_catalog.json",
    })

    options.PostDeployHook = func(options *testprojects.TestProjectsOptions) error {
        // Validate specific resources were created
        if err := validateDatabaseCreated(); err != nil {
            return fmt.Errorf("database validation failed: %w", err)
        }

        if err := validateComputeInstances(); err != nil {
            return fmt.Errorf("compute validation failed: %w", err)
        }

        return nil
    }

    err := options.RunProjectsTest()
    assert.NoError(t, err, "Stack validation should succeed")
}

func validateDatabaseCreated() error {
    // Add your database validation logic
    return nil
}

func validateComputeInstances() error {
    // Add your compute validation logic
    return nil
}
```

### Example 4: Stack Test with Skip Options

```golang
func TestStackValidationOnly(t *testing.T) {
    t.Parallel()

    options := testprojects.TestProjectOptionsDefault(&testprojects.TestProjectsOptions{
        Testing:                t,
        Prefix:                 "validation-only",
        StackConfigurationPath: "stack_definition.json",
        StackCatalogJsonPath:   "ibm_catalog.json",
        SkipUndeploy:           true, // Skip undeploy for investigation
        SkipProjectDelete:      true, // Keep project for manual inspection
    })

    err := options.RunProjectsTest()
    assert.NoError(t, err, "Stack validation test should succeed")
}
```
