# Addon Testing Examples

This guide provides comprehensive examples for different addon testing scenarios. Each example demonstrates best practices and common patterns.

## Basic Examples

### Example 1: Simple Terraform Addon Test (Recommended)

This is the standard approach for most addon testing scenarios:

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
        Testing:       t,
        Prefix:        prefix,
        ResourceGroup: "my-project-rg",
    })
    return options
}

func TestRunTerraformAddon(t *testing.T) {
    t.Parallel()

    options := setupAddonOptions(t, "test-terraform-addon")

    // Using the standard Terraform helper function (recommended for most use cases)
    options.AddonConfig = cloudinfo.NewAddonConfigTerraform(
        options.Prefix,        // prefix for unique resource naming
        "test-addon",          // offering name
        "test-flavor",         // offering flavor
        map[string]interface{}{ // inputs
            "prefix": options.Prefix,
        },
    )

    err := options.RunAddonTest()
    assert.NoError(t, err, "Addon Test had an unexpected error")
}
```

### Example 2: Stack Addon Test (Advanced/Rare Use Case)

This demonstrates how to test a Stack addon. **Note: This is an advanced use case that most users won't need.**

```golang
func TestRunStackAddon(t *testing.T) {
    t.Parallel()

    options := setupAddonOptions(t, "test-stack-addon")

    // Using the specialized Stack helper function (rarely needed)
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
    assert.NoError(t, err, "Addon Test had an unexpected error")
}
```

## Dependency Management Examples

### Example 3: Automatic Dependency Discovery

By default, the framework automatically discovers and processes dependencies:

```golang
func TestRunAddonWithAutoDependencies(t *testing.T) {
    t.Parallel()

    options := setupAddonOptions(t, "auto-dependencies")

    // Create the base addon config - dependencies will be auto-discovered
    options.AddonConfig = cloudinfo.NewAddonConfigTerraform(
        options.Prefix,
        "example-app",
        "standard",
        map[string]interface{}{
            "prefix": options.Prefix,
            "region": "us-south",
        },
    )

    err := options.RunAddonTest()
    assert.NoError(t, err, "Addon Test had an unexpected error")
}
```

### Example 4: Manual Dependency Configuration

You can override automatic dependency discovery by explicitly setting dependencies:

```golang
func TestRunAddonWithCustomDependencyConfig(t *testing.T) {
    t.Parallel()

    options := setupAddonOptions(t, "custom-dependency-config")

    // Create the base addon config
    options.AddonConfig = cloudinfo.NewAddonConfigTerraform(
        options.Prefix,
        "example-app",
        "standard",
        map[string]interface{}{
            "prefix": options.Prefix,
            "region": "us-south",
        },
    )

    // Override dependencies by directly assigning to the Dependencies array
    options.AddonConfig.Dependencies = []cloudinfo.AddonConfig{
        {
            // First dependency - explicitly enable
            OfferingName:   "database",
            OfferingFlavor: "postgresql",
            Inputs: map[string]interface{}{
                "prefix": options.Prefix,
                "plan":   "standard",
            },
            Enabled: core.BoolPtr(true), // explicitly enable this dependency
        },
        {
            // Second dependency - explicitly disable
            OfferingName:   "monitoring",
            OfferingFlavor: "basic",
            Enabled:        core.BoolPtr(false), // explicitly disable this dependency
        },
    }

    err := options.RunAddonTest()
    assert.NoError(t, err, "Addon Test had an unexpected error")
}
```

## Parallel Testing Examples

### Example 5: Manual Matrix Testing

```golang
func TestRunAddonTests(t *testing.T) {
    t.Parallel()

    testCases := []struct {
        name         string
        prefix       string
        dependencies []cloudinfo.AddonConfig
    }{
        {
            name:   "Defaults",
            prefix: "kmsadd",
        },
        {
            name:   "ResourceGroupOnly",
            prefix: "kmsadd",
            dependencies: []cloudinfo.AddonConfig{
                {
                    OfferingName:   "deploy-arch-ibm-account-infra-base",
                    OfferingFlavor: "resource-group-only",
                    Enabled:        core.BoolPtr(true),
                },
            },
        },
        {
            name:   "ResourceGroupWithAccountSettings",
            prefix: "kmsadd",
            dependencies: []cloudinfo.AddonConfig{
                {
                    OfferingName:   "deploy-arch-ibm-account-infra-base",
                    OfferingFlavor: "resource-groups-with-account-settings",
                    Enabled:        core.BoolPtr(true),
                },
            },
        },
    }

    for _, tc := range testCases {
        tc := tc // Capture loop variable for parallel execution
        t.Run(tc.name, func(t *testing.T) {
            t.Parallel()

            options := setupAddonOptions(t, tc.prefix)

            options.AddonConfig = cloudinfo.NewAddonConfigTerraform(
                options.Prefix,        // prefix for unique resource naming
                "deploy-arch-ibm-kms", // offering name
                "fully-configurable",  // offering flavor
                map[string]interface{}{ // inputs
                    "prefix": options.Prefix,
                    "region": "us-south",
                },
            )

            // Set dependencies if provided
            if tc.dependencies != nil {
                options.AddonConfig.Dependencies = tc.dependencies
            }

            err := options.RunAddonTest()
            assert.NoError(t, err, "Addon Test had an unexpected error")
        })
    }
}
```

### Example 6: Framework Matrix Testing

Using the framework's built-in matrix testing utilities:

```golang
func TestRunAddonTestsWithFramework(t *testing.T) {
    matrix := testaddons.AddonTestMatrix{
        TestCases: []testaddons.AddonTestCase{
            {
                Name:   "Defaults",
                Prefix: "kmsadd",
            },
            {
                Name:   "ResourceGroupOnly",
                Prefix: "kmsadd",
                Dependencies: []cloudinfo.AddonConfig{
                    {
                        OfferingName:   "deploy-arch-ibm-account-infra-base",
                        OfferingFlavor: "resource-group-only",
                        Enabled:        core.BoolPtr(true),
                    },
                },
            },
            {
                Name:   "CustomInputs",
                Prefix: "kmsadd",
                Inputs: map[string]interface{}{
                    "region": "eu-gb",
                    "plan":   "standard",
                },
            },
        },
        BaseSetupFunc: func(testCase testaddons.AddonTestCase) *testaddons.TestAddonOptions {
            return testaddons.TestAddonsOptionsDefault(&testaddons.TestAddonOptions{
                Testing:       t,
                Prefix:        testCase.Prefix,
                ResourceGroup: "my-resource-group",
            })
        },
        AddonConfigFunc: func(options *testaddons.TestAddonOptions, testCase testaddons.AddonTestCase) cloudinfo.AddonConfig {
            return cloudinfo.NewAddonConfigTerraform(
                options.Prefix,
                "deploy-arch-ibm-kms",
                "fully-configurable",
                map[string]interface{}{
                    "prefix": options.Prefix,
                    "region": "us-south",
                },
            )
        },
    }

    testaddons.RunAddonTestMatrix(t, matrix)
}
```

## Hook Examples

### Example 7: Using Pre-Deploy Hook

```golang
func TestAddonWithPreDeployHook(t *testing.T) {
    t.Parallel()

    options := setupAddonOptions(t, "pre-deploy-hook")

    options.AddonConfig = cloudinfo.NewAddonConfigTerraform(
        options.Prefix,
        "my-addon",
        "standard",
        map[string]interface{}{
            "prefix": options.Prefix,
        },
    )

    // Pre-deployment configuration
    options.PreDeployHook = func(options *testaddons.TestAddonOptions) error {
        // Configure additional environment variables
        os.Setenv("CUSTOM_CONFIG", "value")

        // Validate custom prerequisites
        if err := validateCustomPrerequisites(); err != nil {
            return fmt.Errorf("custom prerequisites failed: %w", err)
        }

        options.Logger.ShortInfo("Custom pre-deployment configuration completed")
        return nil
    }

    err := options.RunAddonTest()
    assert.NoError(t, err, "Addon Test had an unexpected error")
}

func validateCustomPrerequisites() error {
    // Add your custom validation logic here
    return nil
}
```

### Example 8: Using Post-Deploy Hook for Validation

```golang
func TestAddonWithPostDeployValidation(t *testing.T) {
    t.Parallel()

    options := setupAddonOptions(t, "post-deploy-validation")

    options.AddonConfig = cloudinfo.NewAddonConfigTerraform(
        options.Prefix,
        "my-addon",
        "standard",
        map[string]interface{}{
            "prefix": options.Prefix,
        },
    )

    // Post-deployment testing and validation
    options.PostDeployHook = func(options *testaddons.TestAddonOptions) error {
        // Test custom endpoints or services
        if err := testCustomEndpoints(options.AddonConfig); err != nil {
            return fmt.Errorf("custom endpoint tests failed: %w", err)
        }

        // Validate deployed resources meet custom requirements
        if err := validateDeployedResources(options.ProjectID); err != nil {
            return fmt.Errorf("resource validation failed: %w", err)
        }

        options.Logger.ShortInfo("Custom post-deployment validation passed")
        return nil
    }

    err := options.RunAddonTest()
    assert.NoError(t, err, "Addon Test had an unexpected error")
}

func testCustomEndpoints(config cloudinfo.AddonConfig) error {
    // Add your endpoint testing logic here
    return nil
}

func validateDeployedResources(projectID string) error {
    // Add your resource validation logic here
    return nil
}
```

### Example 9: Complete Hook Example with Cleanup

```golang
func TestAddonWithAllHooks(t *testing.T) {
    t.Parallel()

    options := setupAddonOptions(t, "all-hooks")

    options.AddonConfig = cloudinfo.NewAddonConfigTerraform(
        options.Prefix,
        "my-addon",
        "standard",
        map[string]interface{}{
            "prefix": options.Prefix,
        },
    )

    // Pre-deployment hook
    options.PreDeployHook = func(options *testaddons.TestAddonOptions) error {
        options.Logger.ShortInfo("Running pre-deployment setup")
        // Setup custom configuration
        return nil
    }

    // Post-deployment hook
    options.PostDeployHook = func(options *testaddons.TestAddonOptions) error {
        options.Logger.ShortInfo("Running post-deployment validation")
        // Validate deployment
        return nil
    }

    // Pre-undeploy hook
    options.PreUndeployHook = func(options *testaddons.TestAddonOptions) error {
        options.Logger.ShortInfo("Running pre-undeploy data preservation")
        // Export important data before cleanup
        if err := exportTestData(options.ProjectID); err != nil {
            return fmt.Errorf("data export failed: %w", err)
        }
        return nil
    }

    // Post-undeploy hook
    options.PostUndeployHook = func(options *testaddons.TestAddonOptions) error {
        options.Logger.ShortInfo("Running post-undeploy cleanup verification")
        // Verify cleanup completed successfully
        return verifyCleanupComplete(options.ResourceGroup)
    }

    err := options.RunAddonTest()
    assert.NoError(t, err, "Addon Test had an unexpected error")
}

func exportTestData(projectID string) error {
    // Add your data export logic here
    return nil
}

func verifyCleanupComplete(resourceGroup string) error {
    // Add your cleanup verification logic here
    return nil
}
```

## Advanced Configuration Examples

### Example 10: Custom Validation Options

```golang
func TestAddonWithCustomValidation(t *testing.T) {
    t.Parallel()

    options := setupAddonOptions(t, "custom-validation")

    // Configure validation options
    options.SkipRefValidation = false // Enable reference validation (default)
    options.SkipDependencyValidation = false // Enable dependency validation (default)
    options.VerboseValidationErrors = true // Show detailed error messages
    options.EnhancedTreeValidationOutput = true // Show dependency tree

    // Configure local change check
    options.SkipLocalChangeCheck = false
    options.LocalChangesIgnorePattern = []string{
        ".*\\.md$",    // ignore markdown files
        "^docs/.*",    // ignore docs directory
        "^temp/.*",    // ignore temporary files
    }

    options.AddonConfig = cloudinfo.NewAddonConfigTerraform(
        options.Prefix,
        "my-addon",
        "standard",
        map[string]interface{}{
            "prefix": options.Prefix,
        },
    )

    err := options.RunAddonTest()
    assert.NoError(t, err, "Addon Test had an unexpected error")
}
```

### Example 11: Skip Infrastructure Deployment

This example shows how to run all validations without actually deploying infrastructure:

```golang
func TestAddonValidationOnly(t *testing.T) {
    t.Parallel()

    options := setupAddonOptions(t, "validation-only")

    // Skip infrastructure deployment but perform all validations
    options.SkipInfrastructureDeployment = true

    options.AddonConfig = cloudinfo.NewAddonConfigTerraform(
        options.Prefix,
        "my-addon",
        "standard",
        map[string]interface{}{
            "prefix": options.Prefix,
        },
    )

    err := options.RunAddonTest()
    assert.NoError(t, err, "Validation test had an unexpected error")
}
```

### Example 12: Custom Project Configuration

```golang
func TestAddonWithCustomProject(t *testing.T) {
    t.Parallel()

    options := testaddons.TestAddonsOptionsDefault(&testaddons.TestAddonOptions{
        Testing:                  t,
        Prefix:                   "custom-project",
        ResourceGroup:            "my-project-rg",
        ProjectName:              "my-custom-project",
        ProjectDescription:       "Custom project for specialized testing",
        ProjectLocation:          "us-south",
        ProjectDestroyOnDelete:   core.BoolPtr(true),
        ProjectMonitoringEnabled: core.BoolPtr(true),
        ProjectAutoDeploy:        core.BoolPtr(false), // Manual deployment
        DeployTimeoutMinutes:     120, // 2 hours instead of default 6
    })

    options.AddonConfig = cloudinfo.NewAddonConfigTerraform(
        options.Prefix,
        "my-addon",
        "standard",
        map[string]interface{}{
            "prefix": options.Prefix,
        },
    )

    err := options.RunAddonTest()
    assert.NoError(t, err, "Addon Test had an unexpected error")
}
```

## Multi-Region Testing Example

### Example 13: Testing Across Multiple Regions

```golang
func TestMultiRegionAddon(t *testing.T) {
    regions := []string{"us-south", "us-east", "eu-gb", "jp-tok"}

    for _, region := range regions {
        region := region // Capture loop variable
        t.Run(fmt.Sprintf("Region_%s", region), func(t *testing.T) {
            t.Parallel()

            options := setupAddonOptions(t, fmt.Sprintf("region-%s", strings.ReplaceAll(region, "-", "")))

            options.AddonConfig = cloudinfo.NewAddonConfigTerraform(
                options.Prefix,
                "my-addon",
                "standard",
                map[string]interface{}{
                    "prefix": options.Prefix,
                    "region": region,
                },
            )

            err := options.RunAddonTest()
            assert.NoError(t, err, "Multi-region test had an unexpected error for region %s", region)
        })
    }
}
```

## Error Handling Example

### Example 14: Test with Error Handling and Retry Logic

```golang
func TestAddonWithErrorHandling(t *testing.T) {
    t.Parallel()

    options := setupAddonOptions(t, "error-handling")

    options.AddonConfig = cloudinfo.NewAddonConfigTerraform(
        options.Prefix,
        "my-addon",
        "standard",
        map[string]interface{}{
            "prefix": options.Prefix,
        },
    )

    // Add error handling in post-deploy hook
    options.PostDeployHook = func(options *testaddons.TestAddonOptions) error {
        maxRetries := 3
        for i := 0; i < maxRetries; i++ {
            if err := validateDeployment(options); err != nil {
                if i == maxRetries-1 {
                    return fmt.Errorf("validation failed after %d attempts: %w", maxRetries, err)
                }
                options.Logger.ShortInfo("Validation failed, retrying in 30 seconds...")
                time.Sleep(30 * time.Second)
                continue
            }
            options.Logger.ShortInfo("Validation successful")
            return nil
        }
        return nil
    }

    err := options.RunAddonTest()
    assert.NoError(t, err, "Addon Test had an unexpected error")
}

func validateDeployment(options *testaddons.TestAddonOptions) error {
    // Add your deployment validation logic with potential for transient failures
    return nil
}
```
