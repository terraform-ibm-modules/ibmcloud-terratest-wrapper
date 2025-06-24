# Parallel Testing Guide

This guide covers how to run multiple addon test configurations in parallel using the **matrix testing approach**. Matrix testing is the **primary and recommended pattern** for parallel addon testing, providing a clean, declarative way to define multiple test scenarios.

## Matrix Testing Pattern (Recommended)

Matrix testing uses the `AddonTestMatrix` structure to define test cases and run them in parallel. This is the primary approach for parallel testing in the framework.

### Basic Matrix Test Structure

```golang
package test

import (
    "testing"
    "github.com/IBM/go-sdk-core/v5/core"
    "github.com/stretchr/testify/assert"
    "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/cloudinfo"
    "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testaddons"
)

// TestRunAddonTests demonstrates the primary matrix testing approach
func TestRunAddonTests(t *testing.T) {
    testCases := []testaddons.AddonTestCase{
        {
            Name:   "FullDeployment",
            Prefix: "deploy",
            // Full deployment with dependencies
            Dependencies: []cloudinfo.AddonConfig{
                {
                    OfferingName:   "deploy-arch-ibm-account-infra-base",
                    OfferingFlavor: "resource-group-only",
                    Enabled:        core.BoolPtr(true),
                },
            },
        },
        {
            Name:   "ValidationOnly",
            Prefix: "validate",
            SkipInfrastructureDeployment: true, // Validation-only test
            Dependencies: []cloudinfo.AddonConfig{
                {
                    OfferingName:   "deploy-arch-ibm-account-infra-base",
                    OfferingFlavor: "resource-group-only",
                    Enabled:        core.BoolPtr(true),
                },
            },
        },
        {
            Name:   "CustomInputsDeployment",
            Prefix: "custom",
            Inputs: map[string]interface{}{
                "region": "eu-gb",
                "plan":   "standard",
            },
        },
    }

    matrix := testaddons.AddonTestMatrix{
        TestCases: testCases,
        BaseOptions: testaddons.TestAddonsOptionsDefault(&testaddons.TestAddonOptions{
            Testing:       t,
            Prefix:        "matrix-test", // Individual test cases will override with their own prefixes
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
                "my-addon",
                "standard",
                map[string]interface{}{
                    "prefix": options.Prefix,
                    "region": "us-south",
                },
            )
        },
    }

    baseOptions.RunAddonTestMatrix(matrix)
}
```

## Matrix Testing Structure and Configuration

The `AddonTestMatrix` provides a declarative way to define multiple test cases and run them in parallel. This is the **primary approach** for parallel testing.

### Key Components

- **BaseOptions**: A `TestAddonOptions` object containing common settings for all test cases
- **TestCases**: An array of `AddonTestCase` structures defining the individual test scenarios
- **BaseSetupFunc** (optional): A function to customize the copied BaseOptions for each test case
  - Signature: `func(baseOptions *TestAddonOptions, testCase AddonTestCase) *TestAddonOptions`
  - The `baseOptions` parameter is a copy of the `BaseOptions` field
- **AddonConfigFunc**: A function that creates the specific addon configuration for each test case
  - Signature: `func(options *TestAddonOptions, testCase AddonTestCase) cloudinfo.AddonConfig`

### Basic Usage Pattern

The recommended pattern reduces boilerplate by allowing you to specify common options that apply to all test cases:

```go
func TestMultipleAddons(t *testing.T) {
    testCases := []testaddons.AddonTestCase{
        {
            Name:   "BasicDeployment",
            Prefix: "basic",
        },
        {
            Name:   "CustomInputsDeployment",
            Prefix: "custom",
            Inputs: map[string]interface{}{
                "region": "eu-gb",
                "plan":   "standard",
            },
        },
    }

    // Define common options that apply to all test cases
    baseOptions := testaddons.TestAddonsOptionsDefault(&testaddons.TestAddonOptions{
        Testing:       t,
        Prefix:        "matrix-test", // Individual test cases will override with their own prefixes
        ResourceGroup: "my-resource-group",
    })

    matrix := testaddons.AddonTestMatrix{
        TestCases:   testCases,
        BaseOptions: baseOptions, // Common options for all test cases
        BaseSetupFunc: func(baseOptions *testaddons.TestAddonOptions, testCase testaddons.AddonTestCase) *testaddons.TestAddonOptions {
            // Optional: customize options per test case
            // The baseOptions parameter is a copy of the BaseOptions field above
            // Most common customizations (like Prefix) are handled automatically
            return baseOptions
        },
        AddonConfigFunc: func(options *testaddons.TestAddonOptions, testCase testaddons.AddonTestCase) cloudinfo.AddonConfig {
            return cloudinfo.NewAddonConfigTerraform(
                options.Prefix,
                "my-addon",
                "standard",
                map[string]interface{}{
                    "prefix": options.Prefix,
                    "region": "us-south",
                },
            )
        },
    }

    baseOptions.RunAddonTestMatrix(matrix)
}
```

### Legacy Pattern (Still Supported)

For backward compatibility, you can still create options from scratch for each test case:

```go
func TestMultipleAddonsLegacy(t *testing.T) {
    testCases := []testaddons.AddonTestCase{
        {
            Name:   "BasicDeployment",
            Prefix: "basic",
        },
        {
            Name:   "CustomInputsDeployment",
            Prefix: "custom",
            Inputs: map[string]interface{}{
                "region": "eu-gb",
                "plan":   "standard",
            },
        },
    }

    matrix := testaddons.AddonTestMatrix{
        TestCases: testCases,
        BaseSetupFunc: func(baseOptions *testaddons.TestAddonOptions, testCase testaddons.AddonTestCase) *testaddons.TestAddonOptions {
            // Note: baseOptions will be nil in legacy mode
            return testaddons.TestAddonsOptionsDefault(&testaddons.TestAddonOptions{
                Testing:       t,
                Prefix:        testCase.Prefix,
                ResourceGroup: "my-resource-group",
            })
        },
        AddonConfigFunc: func(options *testaddons.TestAddonOptions, testCase testaddons.AddonTestCase) cloudinfo.AddonConfig {
            return cloudinfo.NewAddonConfigTerraform(
                options.Prefix,
                "my-addon",
                "standard",
                map[string]interface{}{
                    "prefix": options.Prefix,
                    "region": "us-south",
                },
            )
        },
    }

    // Create a base options object to run the matrix (only used for the test runner, not test cases)
    baseOptions := testaddons.TestAddonsOptionsDefault(&testaddons.TestAddonOptions{
        Testing: t,
        Prefix:  "matrix-runner",
    })

    baseOptions.RunAddonTestMatrix(matrix)
}
```

### Benefits of Using BaseOptions

1. **Reduced Boilerplate**: Common options like `ResourceGroup`, `Testing`, etc. are defined once
2. **Better Maintainability**: Changes to common settings only need to be made in one place
3. **Clearer Intent**: Separates common configuration from test-specific customization
4. **Automatic Handling**: Framework automatically handles common patterns like prefix assignment

### AddonTestCase Configuration Options

Each test case supports several configuration options:

- **Name**: Test case name that appears in test output and log messages
- **Prefix**: Unique prefix for resource naming in this test case (automatically used if provided)
- **Dependencies**: Addon dependencies to configure for this test case
- **Inputs**: Additional inputs to merge with the base addon configuration
- **SkipTearDown**: Skip cleanup for this specific test case (useful for debugging)
- **SkipInfrastructureDeployment**: Skip infrastructure deployment and undeploy operations for this specific test case

## Complete Example: Testing Multiple Configurations

This comprehensive example demonstrates testing an addon across different configurations and regions:

```go
func TestMyAddonMatrix(t *testing.T) {
    testCases := []testaddons.AddonTestCase{
        {
            Name:   "BasicDeployment",
            Prefix: "basic",
        },
        {
            Name:   "PremiumPlan",
            Prefix: "premium",
            Inputs: map[string]interface{}{
                "plan": "premium",
            },
        },
        {
            Name:   "EuropeRegion",
            Prefix: "eu",
            Inputs: map[string]interface{}{
                "region": "eu-gb",
                "datacenter": "lon06",
            },
        },
        {
            Name:   "WithDependencies",
            Prefix: "deps",
            Dependencies: []cloudinfo.AddonConfig{
                cloudinfo.NewAddonConfigTerraform("dep", "prereq-addon", "1.0.0", nil),
            },
        },
        {
            Name:                            "ValidationOnly",
            Prefix:                          "valid",
            SkipInfrastructureDeployment:    true, // Skip actual deployment for this test
        },
    }

    // Define common options that apply to all test cases
    baseOptions := testaddons.TestAddonsOptionsDefault(&testaddons.TestAddonOptions{
        Testing:                     t,
        Prefix:                      "addon-matrix",
        ResourceGroup:               "default",
        DeployTimeoutMinutes:        60,
        SkipLocalChangeCheck:        true,
        VerboseValidationErrors:     true,
    })

    matrix := testaddons.AddonTestMatrix{
        TestCases:   testCases,
        BaseOptions: baseOptions,
        BaseSetupFunc: func(baseOptions *testaddons.TestAddonOptions, testCase testaddons.AddonTestCase) *testaddons.TestAddonOptions {
            // Optional customization per test case
            // For most cases, the default behavior (using BaseOptions as-is) is sufficient

            if testCase.Name == "PremiumPlan" {
                // Example: increase timeout for premium deployments
                baseOptions.DeployTimeoutMinutes = 120
            }

            return baseOptions
        },
        AddonConfigFunc: func(options *testaddons.TestAddonOptions, testCase testaddons.AddonTestCase) cloudinfo.AddonConfig {
            // Create base configuration
            config := cloudinfo.NewAddonConfigTerraform(
                options.Prefix,
                "my-addon",
                "1.2.3",
                map[string]interface{}{
                    "prefix":         options.Prefix,
                    "resource_group": options.ResourceGroup,
                    "region":         "us-south", // Default region
                },
            )

            return config
        },
    }

    baseOptions.RunAddonTestMatrix(matrix)
}

## Automatic Catalog Sharing in Matrix Tests

When using matrix testing with `RunAddonTestMatrix()`, the framework automatically shares catalogs and offerings across all test cases for improved efficiency and reduced resource usage.

### How Catalog Sharing Works

- **Matrix Tests**: All test cases in a matrix automatically share a single catalog and offering
- **Individual Tests**: Each individual test still gets its own catalog and offering
- **Resource Lifecycle**: The first test case creates the catalog/offering, subsequent tests reuse them
- **Cleanup**: Shared resources are cleaned up automatically after all matrix tests complete

### Benefits

**Resource Efficiency**: Instead of creating 20 catalogs for 20 test cases, only 1 catalog is created and shared.

**Time Savings**: Significant reduction in catalog/offering creation time and IBM Cloud API calls.

**Cost Optimization**: Fewer temporary resources created, reduced chance of resource conflicts.

### Controlling Catalog Sharing

The framework provides a `SharedCatalog` option to control sharing behavior:

```go
// Default: SharedCatalog = true (efficient sharing)
baseOptions := testaddons.TestAddonsOptionsDefault(&testaddons.TestAddonOptions{
    Testing:       t,
    Prefix:        "test",
    ResourceGroup: "my-rg",
    // SharedCatalog defaults to true - can omit or explicitly set
    SharedCatalog: core.BoolPtr(true),
})

// For complete isolation: SharedCatalog = false
isolatedOptions := testaddons.TestAddonsOptionsDefault(&testaddons.TestAddonOptions{
    Testing:       t,
    Prefix:        "isolated",
    ResourceGroup: "my-rg",
    SharedCatalog: core.BoolPtr(false), // Each test gets own catalog
})
```

**When to use SharedCatalog = false:**

- When you need complete test isolation
- When testing catalog-specific functionality
- When debugging catalog-related issues

**Default behavior (SharedCatalog = true):**

- Efficient resource usage
- Faster test execution
- Recommended for most scenarios

### Thread Safety

The framework uses mutex synchronization during the setup phase to ensure catalog/offering creation is protected against race conditions in parallel test execution.

## Test Output and Logging

When running addon tests, each test generates log output with clear identification:

### Log Format

Addon tests use descriptive names in log messages for easy identification:

```text
[TestRunAddonTests - ADDON - PrimaryScenarioDeploy] Checking for local changes in the repository
[TestRunAddonTests - ADDON - AlternativeScenarioValidate] Checking for local changes in the repository
[TestRunAddonTests - ADDON - EdgeCaseValidate] Checking for local changes in the repository
```

**For Matrix Tests**: The test case `Name` field is automatically used for logging.

**For Individual Tests**: You can set `TestCaseName` manually for custom log identification:

```golang
func TestCustomAddon(t *testing.T) {
    options := testaddons.TestAddonsOptionsDefault(&testaddons.TestAddonOptions{
        Testing:      t,
        Prefix:       "custom-test",
        TestCaseName: "CustomScenarioValidation", // Custom log identifier
    })

    // This will show logs like:
    // [TestCustomAddon - ADDON - CustomScenarioValidation] Checking for local changes...

    err := options.RunAddonTest()
    require.NoError(t, err)
}
```

### Log Prefix Behavior

The framework automatically sets appropriate log prefixes based on context:

- **Matrix Tests**: Uses the test case `Name` field: `"ADDON - [TestCaseName]"`
- **Regular Tests**: Uses the project name: `"ADDON - [ProjectName]"`
- **Fallback**: Uses generic prefix: `"ADDON"`

This makes it easy to:

- **Track Progress**: See which specific test case is running
- **Debug Issues**: Identify which test case encountered problems
- **Correlate Logs**: Match log entries to specific test scenarios

### Example Output

```text
[TestRunAddonTests - ADDON - FullDeploymentDefaults] Starting addon test setup
[TestRunAddonTests - ADDON - ValidationOnlyDefaults] Starting addon test setup
[TestRunAddonTests - ADDON - FullDeploymentWithDependencies] Checking for local changes in the repository
[TestRunAddonTests - ADDON - ValidationOnlyWithDependencies] Checking for local changes in the repository
```

## Mixed Deployment and Validation Testing

The framework supports both full deployment tests and validation-only tests within the same matrix, allowing you to optimize testing costs and time.

### AddonTestMatrix Structure

The `AddonTestMatrix` type has three key components:

- **TestCases**: An array of `AddonTestCase` structures defining the individual test scenarios
- **BaseOptions**: Common configuration options that apply to all test cases
- **BaseSetupFunc**: A function that can customize the base options for each test case
- **AddonConfigFunc**: A function that creates the specific addon configuration for each test case

This approach separates concerns by letting you define:

1. **What to test** (in `TestCases`)
2. **Common configuration** (in `BaseOptions`)
3. **How to customize per test** (in `BaseSetupFunc`)
4. **How to configure the addon** (in `AddonConfigFunc`)

### Using AddonTestCase and RunAddonTestMatrix

```golang
package test

import (
    "testing"
    "github.com/IBM/go-sdk-core/v5/core"
    "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/cloudinfo"
    "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testaddons"
)

// TestComprehensiveAddonMatrix demonstrates mixed deployment and validation testing
func TestComprehensiveAddonMatrix(t *testing.T) {
    testCases := []testaddons.AddonTestCase{
        {
            Name:   "FullDeploymentDefaults",
            Prefix: "deploy-default",
            // Full deployment with default configuration
        },
        {
            Name:   "ValidationOnlyDefaults",
            Prefix: "validate-default",
            SkipInfrastructureDeployment: true, // Validation-only test
        },
        {
            Name:   "FullDeploymentWithDependencies",
            Prefix: "deploy-deps",
            Dependencies: []cloudinfo.AddonConfig{
                {
                    OfferingName:   "deploy-arch-ibm-account-infra-base",
                    OfferingFlavor: "resource-group-only",
                    Enabled:        core.BoolPtr(true),
                },
            },
        },
        {
            Name:   "ValidationOnlyWithDependencies",
            Prefix: "validate-deps",
            SkipInfrastructureDeployment: true, // Validation-only test with dependencies
            Dependencies: []cloudinfo.AddonConfig{
                {
                    OfferingName:   "deploy-arch-ibm-account-infra-base",
                    OfferingFlavor: "resource-groups-with-account-settings",
                    Enabled:        core.BoolPtr(true),
                },
            },
        },
        {
            Name:   "FullDeploymentCustomInputs",
            Prefix: "deploy-custom",
            Inputs: map[string]interface{}{
                "region": "eu-gb",
                "plan":   "standard",
            },
        },
        {
            Name:   "ValidationOnlyCustomInputs",
            Prefix: "validate-custom",
            SkipInfrastructureDeployment: true, // Validation-only with custom inputs
            Inputs: map[string]interface{}{
                "region": "ap-south",
                "plan":   "enterprise",
            },
        },
    }

    matrix := testaddons.AddonTestMatrix{
        TestCases: testCases,
        BaseOptions: testaddons.TestAddonsOptionsDefault(&testaddons.TestAddonOptions{
            Testing:       t,
            Prefix:        "comprehensive-test",
            ResourceGroup: "my-resource-group",
        }),
        BaseSetupFunc: func(baseOptions *testaddons.TestAddonOptions, testCase testaddons.AddonTestCase) *testaddons.TestAddonOptions {
            // Optional: customize per test case if needed
            return baseOptions
        },
        AddonConfigFunc: func(options *testaddons.TestAddonOptions, testCase testaddons.AddonTestCase) cloudinfo.AddonConfig {
            return cloudinfo.NewAddonConfigTerraform(
                options.Prefix,
                "my-addon",
                "standard",
                map[string]interface{}{
                    "prefix": options.Prefix,
                    "region": "us-south", // Default region, can be overridden by testCase.Inputs
                },
            )
        },
    }

    baseOptions.RunAddonTestMatrix(matrix)
}
```

## Cost-Effective Testing Strategies

Matrix testing allows you to optimize testing costs by mixing deployment and validation tests:

### Example: Balanced Cost and Coverage Testing

```golang
func TestCostEffectiveAddonMatrix(t *testing.T) {
    testCases := []testaddons.AddonTestCase{
        {
            Name:   "PrimaryDeployment",
            Prefix: "deploy-main",
            // Full deployment for primary scenario
        },
        {
            Name:   "ValidationAlternativeInputs",
            Prefix: "validate-alt",
            SkipInfrastructureDeployment: true, // Fast validation for alternative scenarios
            Inputs: map[string]interface{}{
                "region": "eu-gb",
                "plan":   "enterprise",
            },
        },
        {
            Name:   "ValidationMultipleDependencies",
            Prefix: "validate-multi",
            SkipInfrastructureDeployment: true, // Validate complex dependencies without deployment cost
            Dependencies: []cloudinfo.AddonConfig{
                {
                    OfferingName:   "deploy-arch-ibm-account-infra-base",
                    OfferingFlavor: "resource-groups-with-account-settings",
                    Enabled:        core.BoolPtr(true),
                },
                {
                    OfferingName:   "deploy-arch-ibm-vpc",
                    OfferingFlavor: "standard",
                    Enabled:        core.BoolPtr(true),
                },
            },
        },
    }

    matrix := testaddons.AddonTestMatrix{
        TestCases: testCases,
        BaseOptions: testaddons.TestAddonsOptionsDefault(&testaddons.TestAddonOptions{
            Testing:       t,
            Prefix:        "cost-effective-test",
            ResourceGroup: "test-resource-group",
        }),
        AddonConfigFunc: func(options *testaddons.TestAddonOptions, testCase testaddons.AddonTestCase) cloudinfo.AddonConfig {
            return cloudinfo.NewAddonConfigTerraform(
                options.Prefix,
                "my-addon",
                "standard",
                map[string]interface{}{
                    "prefix": options.Prefix,
                },
            )
        },
    }

    baseOptions.RunAddonTestMatrix(matrix)
}
```

## Advanced Matrix Testing Patterns

### Testing Multiple Flavors with Validation

```golang
func TestFlavorMatrix(t *testing.T) {
    testCases := []testaddons.AddonTestCase{
        {
            Name:   "BasicFlavorDeploy",
            Prefix: "basic-deploy",
            // Full deployment of basic flavor
        },
        {
            Name:   "StandardFlavorValidate",
            Prefix: "std-validate",
            SkipInfrastructureDeployment: true, // Validate standard flavor without deployment
        },
        {
            Name:   "EnterpriseFlavorValidate",
            Prefix: "ent-validate",
            SkipInfrastructureDeployment: true, // Validate enterprise flavor without deployment
        },
    }

    matrix := testaddons.AddonTestMatrix{
        TestCases: testCases,
        BaseOptions: testaddons.TestAddonsOptionsDefault(&testaddons.TestAddonOptions{
            Testing:       t,
            Prefix:        "multi-flavor-test",
            ResourceGroup: "flavor-test-rg",
        }),
        AddonConfigFunc: func(options *testaddons.TestAddonOptions, testCase testaddons.AddonTestCase) cloudinfo.AddonConfig {
            // Map test case names to flavors
            flavorMap := map[string]string{
                "BasicFlavorDeploy":       "basic",
                "StandardFlavorValidate":  "standard",
                "EnterpriseFlavorValidate": "enterprise",
            }

            return cloudinfo.NewAddonConfigTerraform(
                options.Prefix,
                "my-addon",
                flavorMap[testCase.Name],
                map[string]interface{}{
                    "prefix": options.Prefix,
                },
            )
        },
    }

    baseOptions.RunAddonTestMatrix(matrix)
}
```

## Best Practices for Matrix Testing

### 1. **Use Validation-Only Tests for Expensive Resources**

Skip infrastructure deployment for scenarios that only need configuration validation:

```golang
{
    Name:   "ExpensiveResourceValidation",
    Prefix: "expensive-validate",
    SkipInfrastructureDeployment: true, // Skip deployment for expensive resources
}
```

### 2. **Mix Deployment and Validation Tests**

Deploy one representative scenario fully, validate others quickly:

```golang
testCases := []testaddons.AddonTestCase{
    {
        Name:   "PrimaryScenarioDeploy",
        Prefix: "primary",
        // Full deployment
    },
    {
        Name:   "AlternativeScenarioValidate",
        Prefix: "alt",
        SkipInfrastructureDeployment: true, // Quick validation
    },
}
```

### 3. **Use Clear Naming Conventions**

Name test cases to clearly indicate deployment vs validation:

- `*Deploy*` - Full deployment tests
- `*Validate*` - Validation-only tests

### 4. **Optimize for Cost and Time**

Use the matrix pattern to balance thorough testing with resource costs:

```golang
testCases := []testaddons.AddonTestCase{
    {
        Name:   "PrimaryFullDeploy",
        Prefix: "primary",
        // Deploy one main scenario fully
    },
    {
        Name:   "AlternativeValidate",
        Prefix: "alt-validate",
        SkipInfrastructureDeployment: true, // Validate other scenarios quickly
    },
    {
        Name:   "EdgeCaseValidate",
        Prefix: "edge-validate",
        SkipInfrastructureDeployment: true, // Validate edge cases without cost
    },
}
```

## Summary

Matrix testing with `AddonTestMatrix` is the **primary and recommended approach** for parallel addon testing. Key benefits:

✅ **Declarative Configuration**: Define test scenarios clearly in `TestCases`
✅ **Mixed Testing Types**: Combine full deployment and validation-only tests
✅ **Cost Optimization**: Use `SkipInfrastructureDeployment` for expensive scenarios
✅ **Maintainable**: Clean separation of setup, configuration, and test cases
✅ **Scalable**: Easy to add new test scenarios

The matrix pattern allows you to:

- Deploy critical scenarios fully
- Validate alternative configurations quickly
- Test complex dependencies without deployment costs
- Scale testing coverage efficiently

**Use this pattern as your primary approach for parallel addon testing.**
