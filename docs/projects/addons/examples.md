# Addon Testing Examples

This guide provides comprehensive examples for different addon testing scenarios. Each example demonstrates best practices and common patterns.

## Basic Examples

### Simple Terraform Addon Test (Recommended)

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
    options := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
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

### Stack Addon Test (Advanced/Rare Use Case)

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

## Catalog Sharing Examples

### Individual Tests with Shared Catalog Cleanup

When running multiple individual tests with shared catalogs, use manual cleanup:

```golang
package test

import (
    "testing"
    "github.com/IBM/go-sdk-core/v5/core"
    "github.com/stretchr/testify/require"
    "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/cloudinfo"
    "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testaddons"
)

func TestMultipleAddonsWithSharedCatalog(t *testing.T) {
    baseOptions := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
        Testing:       t,
        Prefix:        "shared-test",
        ResourceGroup: "my-resource-group",
        SharedCatalog: core.BoolPtr(true), // Enable catalog sharing
    })

    // Ensure cleanup happens at the end
    defer baseOptions.CleanupSharedResources()

    // Test scenario 1
    t.Run("BasicDeployment", func(t *testing.T) {
        options1 := baseOptions
        options1.AddonConfig = cloudinfo.NewAddonConfigTerraform(
            options1.Prefix,
            "test-addon",
            "basic-flavor",
            map[string]interface{}{
                "prefix": options1.Prefix,
                "region": "us-south",
            },
        )
        err := options1.RunAddonTest()
        require.NoError(t, err)
    })

    // Test scenario 2 - reuses the same catalog/offering
    t.Run("CustomConfiguration", func(t *testing.T) {
        options2 := baseOptions
        options2.AddonConfig = cloudinfo.NewAddonConfigTerraform(
            options2.Prefix,
            "test-addon",
            "basic-flavor", // Same flavor reuses offering
            map[string]interface{}{
                "prefix": options2.Prefix,
                "region": "us-east",
                "custom_setting": "value",
            },
        )
        err := options2.RunAddonTest()
        require.NoError(t, err)
    })

    // CleanupSharedResources() called automatically via defer
    // Catalog and offering are deleted after all tests complete
}
```

**Key Benefits:**

- **Efficiency**: Creates only 1 catalog + offering for multiple test scenarios
- **Speed**: Faster test execution due to fewer IBM Cloud API calls
- **Guaranteed Cleanup**: `defer` ensures resources are cleaned up even if tests fail

## Dependency Management Examples

### Automatic Dependency Discovery

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

### Manual Dependency Configuration

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

### Manual Matrix Testing

```golang
func TestRunAddonTests(t *testing.T) {
    t.Parallel()

    testCases := []testaddons.AddonTestCase{
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
            Name:   "ResourceGroupWithAccountSettings",
            Prefix: "kmsadd",
            Dependencies: []cloudinfo.AddonConfig{
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
        t.Run(tc.Name, func(t *testing.T) {
            t.Parallel()

            options := setupAddonOptions(t, tc.Prefix)

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
            if tc.Dependencies != nil {
                options.AddonConfig.Dependencies = tc.Dependencies
            }

            err := options.RunAddonTest()
            assert.NoError(t, err, "Addon Test had an unexpected error")
        })
    }
}
```

### Framework Matrix Testing

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
        BaseOptions: testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
            Testing:       t,
            Prefix:        "matrix-example", // Test cases will override with their own prefixes
            ResourceGroup: "my-resource-group",
        }),
        BaseSetupFunc: func(baseOptions *testaddons.TestAddonOptions, testCase testaddons.AddonTestCase) *testaddons.TestAddonOptions {
            // Optional: customize options per test case
            // Most common patterns are handled automatically (e.g., prefix assignment)
            return baseOptions
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

    baseOptions.RunAddonTestMatrix(matrix)
}
}
```

## Hook Examples

### Using Pre-Deploy Hook

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
        // Note: import "os" required for os.Setenv
        // os.Setenv("CUSTOM_CONFIG", "value")

        // Validate custom prerequisites
        if err := validateCustomPrerequisites(); err != nil {
            // Note: import "fmt" required for fmt.Errorf
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

### Using Post-Deploy Hook for Validation

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
        if err := validateDeployedResources(options.currentProjectConfig.ProjectID); err != nil {
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

### Complete Hook Example with Cleanup

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
        if err := exportTestData(options.currentProjectConfig.ProjectID); err != nil {
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

### Custom Test Case Naming for Logging

```golang
func TestAddonWithCustomLogging(t *testing.T) {
    t.Parallel()

    options := setupAddonOptions(t, "custom-logging")

    // Set custom test case name for clear log identification
    options.TestCaseName = "ProductionScenarioValidation"

    options.AddonConfig = cloudinfo.NewAddonConfigTerraform(
        options.Prefix,
        "my-addon",
        "production",
        map[string]interface{}{
            "prefix":      options.Prefix,
            "environment": "production",
            "region":      "us-south",
        },
    )

    // Log output will show:
    // [TestAddonWithCustomLogging - ADDON - ProductionScenarioValidation] Starting addon test setup
    // [TestAddonWithCustomLogging - ADDON - ProductionScenarioValidation] Checking for local changes...

    err := options.RunAddonTest()
    assert.NoError(t, err)
}

### Input Override Behavior Examples

These examples demonstrate how `OverrideInputMappings` controls whether user-provided inputs are used or ignored for reference fields:

```golang
// Example 1: Default Behavior - Reference Values Preserved (RECOMMENDED)
func TestAddonWithPreservedReferences(t *testing.T) {
    t.Parallel()

    options := setupAddonOptions(t, "preserved-refs")

    // Default behavior: OverrideInputMappings = false (preserves references)
    // User inputs for reference fields will be IGNORED
    options.AddonConfig = cloudinfo.NewAddonConfigTerraform(
        options.Prefix,
        "my-addon",
        "standard",
        map[string]interface{}{
            "prefix": options.Prefix,
            "region": "us-south",
            // ⚠️ If "existing_kms_crn" has a reference value like "ref:../kms-config.instance_crn"
            // this user-provided value will be IGNORED and the reference preserved
            "existing_kms_crn": "user-provided-crn-value", // THIS WILL BE IGNORED
            "service_plan": "standard", // ✅ This will be used (no existing reference)
        },
    )

    // The framework will:
    // - Keep existing_kms_crn as "ref:../kms-config.instance_crn" (reference preserved)
    // - Use service_plan as "standard" (no existing reference)
    // - Use prefix and region as provided

    err := options.RunAddonTest()
    assert.NoError(t, err)
}

// Example 2: Override Mode - All Values Replaced (DEVELOPMENT/TESTING ONLY)
func TestAddonWithOverriddenReferences(t *testing.T) {
    t.Parallel()

    options := setupAddonOptions(t, "overridden-refs")

    // Override mode: Replace ALL input values including references
    options.OverrideInputMappings = core.BoolPtr(true) // ⚠️ Use with caution

    options.AddonConfig = cloudinfo.NewAddonConfigTerraform(
        options.Prefix,
        "my-addon",
        "standard",
        map[string]interface{}{
            "prefix": options.Prefix,
            "region": "us-south",
            // ✅ This will override any existing reference value
            "existing_kms_crn": "user-provided-crn-value", // THIS WILL BE USED
            "service_plan": "standard",
        },
    )

    // ⚠️ WARNING: This may break dependency relationships
    // Only use for special testing scenarios where you need to override references

    err := options.RunAddonTest()
    assert.NoError(t, err)
}

// Example 3: Production Pattern - Explicit Default Setting
func TestProductionAddonPattern(t *testing.T) {
    t.Parallel()

    options := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
        Testing:       t,
        Prefix:        "prod-pattern",
        ResourceGroup: "my-project-rg",
        // Explicitly document the behavior (though this is the default)
        OverrideInputMappings: core.BoolPtr(false), // Preserve reference mappings
    })

    options.AddonConfig = cloudinfo.NewAddonConfigTerraform(
        options.Prefix,
        "production-addon",
        "enterprise",
        map[string]interface{}{
            "prefix": options.Prefix,
            "region": "us-south",
            "environment": "production",
            // Reference fields will be preserved if they exist
            // Non-reference fields will use these values
        },
    )

    err := options.RunAddonTest()
    assert.NoError(t, err, "Production addon test should preserve proper references")
}
```

**Key Takeaways from Input Override Examples:**

- **Default (`OverrideInputMappings: false`)**: User inputs for reference fields are **IGNORED** - this is the intended behavior
- **Override (`OverrideInputMappings: true`)**: All user inputs replace existing values, including breaking references
- **Reference fields** start with `"ref:"` and connect configurations together
- **Breaking references** can cause deployment failures or incorrect resource connections
- **Use default behavior** for standard testing to maintain proper dependency relationships

// Example comparing different scenarios with clear naming
func TestMultipleScenarios(t *testing.T) {
    scenarios := []struct {
        name        string
        environment string
        flavor      string
    }{
        {"DevelopmentTest", "development", "minimal"},
        {"StagingTest", "staging", "standard"},
        {"ProductionTest", "production", "enterprise"},
    }

    for _, scenario := range scenarios {
        scenario := scenario // capture loop variable
        t.Run(scenario.name, func(t *testing.T) {
            t.Parallel()

            options := setupAddonOptions(t, fmt.Sprintf("multi-%s", scenario.environment))
            // Note: import "fmt" required for fmt.Sprintf
            options.TestCaseName = scenario.name // Clear identification in logs

            options.AddonConfig = cloudinfo.NewAddonConfigTerraform(
                options.Prefix,
                "my-addon",
                scenario.flavor,
                map[string]interface{}{
                    "prefix":      options.Prefix,
                    "environment": scenario.environment,
                },
            )

            err := options.RunAddonTest()
            assert.NoError(t, err)
        })
    }
}
```

### Custom Validation Options

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

### Skip Infrastructure Deployment

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

### Custom Project Configuration

```golang
func TestAddonWithCustomProject(t *testing.T) {
    t.Parallel()

    options := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
        Testing:                        t,
        Prefix:                         "custom-project",
        ResourceGroup:                  "my-project-rg",
        ProjectName:                    "my-custom-project",
        ProjectDescription:             "Custom project for specialized testing",
        ProjectLocation:                "us-south",
        ProjectDestroyOnDelete:         core.BoolPtr(true),
        ProjectMonitoringEnabled:       core.BoolPtr(true),
        ProjectAutoDeploy:              core.BoolPtr(false), // Manual deployment
        ProjectAutoDeployMode:          "manual_approve",    // Require manual approval
        DeployTimeoutMinutes:           120, // 2 hours instead of default 6
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

### Testing Across Multiple Regions

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

## Dependency Permutation Testing Examples

### Basic Permutation Test

The `RunAddonPermutationTest()` method automatically tests all possible dependency combinations for your addon:

```golang
package test

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/cloudinfo"
    "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testaddons"
)

// TestSecretsManagerPermutations tests all dependency combinations automatically
func TestSecretsManagerPermutations(t *testing.T) {
    t.Parallel()

    options := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
        Testing: t,
        Prefix:  "sm-perm",
        AddonConfig: cloudinfo.AddonConfig{
            OfferingName:   "deploy-arch-ibm-secrets-manager",
            OfferingFlavor: "fully-configurable",
            Inputs: map[string]interface{}{
                "prefix":                       "sm-perm",
                "region":                       "us-south",
                "existing_resource_group_name": "default",
                "service_plan":                 "trial",
                "enable_platform_metrics":     false,
            },
        },
    })

    err := options.RunAddonPermutationTest()
    assert.NoError(t, err, "Dependency permutation test should not fail")
}
```

### Multiple Addon Permutation Tests

```golang
func TestMultipleAddonPermutations(t *testing.T) {
    // Test Secrets Manager permutations
    t.Run("SecretsManager", func(t *testing.T) {
        t.Parallel()

        options := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
            Testing: t,
            Prefix:  "sm-perm",
            AddonConfig: cloudinfo.AddonConfig{
                OfferingName:   "deploy-arch-ibm-secrets-manager",
                OfferingFlavor: "fully-configurable",
                Inputs: map[string]interface{}{
                    "prefix":                       "sm-perm",
                    "region":                       "us-south",
                    "existing_resource_group_name": "default",
                    "service_plan":                 "trial",
                },
            },
        })

        err := options.RunAddonPermutationTest()
        assert.NoError(t, err)
    })

    // Test KMS permutations
    t.Run("KMS", func(t *testing.T) {
        t.Parallel()

        options := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
            Testing: t,
            Prefix:  "kms-perm",
            AddonConfig: cloudinfo.AddonConfig{
                OfferingName:   "deploy-arch-ibm-kms",
                OfferingFlavor: "fully-configurable",
                Inputs: map[string]interface{}{
                    "prefix":                       "kms-perm",
                    "region":                       "us-south",
                    "existing_resource_group_name": "default",
                    "service_plan":                 "tiered-pricing",
                },
            },
        })

        err := options.RunAddonPermutationTest()
        assert.NoError(t, err)
    })
}
```

### Combined Testing Strategy

Use permutation testing alongside other testing approaches:

```golang
package test

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/cloudinfo"
    "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testaddons"
)

// Full deployment test for primary scenario
func TestAddonFullDeployment(t *testing.T) {
    t.Parallel()

    options := setupAddonOptions(t, "addon-deploy")

    options.AddonConfig = cloudinfo.NewAddonConfigTerraform(
        options.Prefix,
        "my-addon",
        "standard",
        map[string]interface{}{
            "prefix": options.Prefix,
            "region": "us-south",
        },
    )

    err := options.RunAddonTest()
    assert.NoError(t, err, "Full deployment test should not fail")
}

// Permutation testing for comprehensive dependency validation
func TestAddonDependencyPermutations(t *testing.T) {
    t.Parallel()

    options := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
        Testing: t,
        Prefix:  "addon-perm",
        AddonConfig: cloudinfo.AddonConfig{
            OfferingName:   "my-addon",
            OfferingFlavor: "standard",
            Inputs: map[string]interface{}{
                "prefix": "addon-perm",
                "region": "us-south",
            },
        },
    })

    err := options.RunAddonPermutationTest()
    assert.NoError(t, err, "Dependency permutation test should not fail")
}
```

## Error Handling Example

### Test with Error Handling and Retry Logic

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
