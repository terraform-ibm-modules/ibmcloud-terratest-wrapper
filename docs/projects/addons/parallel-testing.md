# Parallel Testing Guide

This guide covers how to run multiple addon test configurations in parallel using both **matrix testing** and **dependency permutation testing**. These approaches provide different ways to test multiple scenarios efficiently.

## Testing Approaches Overview

### Matrix Testing (Manual Control)
Matrix testing uses the `AddonTestMatrix` structure to define specific test cases and run them in parallel. This approach gives you full control over which scenarios to test and how to configure them.

### Dependency Permutation Testing (Automated)
Dependency permutation testing uses `RunAddonPermutationTest()` to automatically test all possible dependency combinations. This approach provides comprehensive validation with minimal configuration.

## Matrix Testing Pattern

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

    baseOptions := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
        Testing:       t,
        Prefix:        "matrix-test", // Individual test cases will override with their own prefixes
        ResourceGroup: "my-resource-group",
        QuietMode:     true, // Enable quiet mode for clean parallel test output
    })

    matrix := testaddons.AddonTestMatrix{
        TestCases:   testCases,
        BaseOptions: baseOptions,
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
    baseOptions := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
        Testing:       t,
        Prefix:        "matrix-test", // Individual test cases will override with their own prefixes
        ResourceGroup: "my-resource-group",
        QuietMode:     true, // Enable quiet mode for clean parallel test output
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

### Alternative Pattern (Create Options Per Test Case)

You can also create options from scratch for each test case by not providing BaseOptions:

```go
func TestMultipleAddonsAlternative(t *testing.T) {
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
        // BaseOptions: nil, // Don't provide BaseOptions for this pattern
        BaseSetupFunc: func(baseOptions *testaddons.TestAddonOptions, testCase testaddons.AddonTestCase) *testaddons.TestAddonOptions {
            // Note: baseOptions will be nil when BaseOptions is not provided
            return testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
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
    baseOptions := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
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

### When to Use Each Pattern

**Use BaseOptions Pattern (Recommended):**

- When test cases share common configuration
- For cleaner, more maintainable code
- When you want automatic prefix handling

**Use Alternative Pattern:**

- When each test case needs completely different base configuration
- When you need maximum flexibility per test case
- When test cases are fundamentally different in nature

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
    baseOptions := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
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
baseOptions := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
    Testing:       t,
    Prefix:        "test",
    ResourceGroup: "my-rg",
    // SharedCatalog defaults to true - can omit or explicitly set
    SharedCatalog: core.BoolPtr(true),
})

// For complete isolation: SharedCatalog = false
isolatedOptions := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
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
    options := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
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
        BaseOptions: testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
            Testing:       t,
            Prefix:        "comprehensive-test",
            ResourceGroup: "my-resource-group",
            QuietMode:     true, // Enable quiet mode for clean output
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
        BaseOptions: testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
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
        BaseOptions: testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
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

### Project Isolation in Matrix Tests

The framework provides intelligent project management optimized for different testing scenarios:

```go
// Matrix tests: Each test case gets its own project for complete isolation
// The framework automatically creates separate projects for each test case
baseOptions := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
    Testing:       t,
    Prefix:        "test",
    ResourceGroup: "my-rg",
})

// Individual tests: Each test gets its own project for isolation
individualOptions := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
    Testing:       t,
    Prefix:        "individual",
    ResourceGroup: "my-rg",
})
```

**Matrix Test Behavior (Automatic):**

- **All matrix test cases**: Get their own dedicated projects for complete isolation
- **Configuration isolation**: Each test case gets its own project and uniquely named configurations
- **Clean separation**: Framework handles project lifecycle automatically

**Individual Test Behavior:**

- **Default**: Each individual test gets its own project for complete isolation
- **Reliable**: Consistent behavior across all test types

**Default behavior (automatic and recommended):**

- **Matrix Tests**: Each test case gets its own project for complete isolation
- **Individual Tests**: Each gets its own project for complete isolation
- **Configuration Isolation**: Each test case gets its own uniquely named configuration in its own project
- **No configuration needed**: Framework automatically manages project lifecycle

**Benefits of project isolation:**

- Each test case has its own dedicated project
- Complete isolation prevents test interference
- Easier debugging when tests fail
- Reliable cleanup of resources per test case
- Consistent behavior across test types

**When project isolation helps:**

- When testing project-specific functionality
- When debugging project-related issues
- Preventing test cases from interfering with each other
- Ensuring clean resource management and cleanup

### Example: Efficient Matrix Testing

```golang
func TestEfficientAddonMatrix(t *testing.T) {
    testCases := []testaddons.AddonTestCase{
        {Name: "BasicConfiguration", Prefix: "basic"},
        {Name: "AdvancedConfiguration", Prefix: "advanced"},
        {Name: "ProductionConfiguration", Prefix: "prod"},
    }

    baseOptions := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
        Testing:       t,
        Prefix:        "efficient-test",
        ResourceGroup: "my-resource-group",
        SharedCatalog: core.BoolPtr(true),  // Share catalog for efficiency
    })

    matrix := testaddons.AddonTestMatrix{
        TestCases:   testCases,
        BaseOptions: baseOptions,
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
    // Result: 1 shared catalog, but each test case gets its own project
    // Each test case gets its own uniquely named configuration in its own project
}
```

### Resource Sharing Summary

| Resource Type | Matrix Default | Individual Default | Isolation Level |
|---------------|----------------|-------------------|-----------------|
| **Catalog** | Shared (automatic) | Private | Offering level |
| **Project** | Private (per test case) | Private | Configuration level |

**Matrix project behavior:**

- **All matrix tests**: Each test case gets its own dedicated project
- **Complete isolation**: Each test case has its own project and configurations
- **Reliable cleanup**: Each test case manages its own project lifecycle
- **Consistent**: Same isolation model across all test types

**Maximum Efficiency Options**: Matrix tests can share catalogs while maintaining project isolation
**Individual Test Isolation**: Each individual test gets its own catalog and project
**Recommended**: Use matrix tests for parallel scenarios with catalog sharing for efficiency

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
        Name:   "PrimaryFullDeploy",
        Prefix: "primary",
        // Deploy one main scenario fully
    },
    {
        Name:   "AlternativeValidate",
        Prefix: "alt-validate",
        SkipInfrastructureDeployment: true, // Validate other scenarios quickly
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
}
```

## Dependency Permutation Testing

### Overview

Dependency permutation testing automatically tests all possible combinations of addon dependencies without manual configuration. Use `RunAddonPermutationTest()` for comprehensive dependency validation.

### Basic Permutation Test

```golang
package test

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/cloudinfo"
    "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testaddons"
)

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

    // Automatically tests all dependency combinations
    err := options.RunAddonPermutationTest()
    assert.NoError(t, err)
}
```

### How Permutation Testing Works

1. **Automatic Discovery**: Discovers all dependencies from the catalog
2. **Permutation Generation**: Creates all 2^n combinations of dependencies (enabled/disabled)
3. **Validation-Only**: All permutations skip infrastructure deployment
4. **Parallel Execution**: Runs all permutations in parallel for efficiency

### Example: Addon with 3 Dependencies

For an addon with 3 dependencies (KMS, Observability, EventNotifications):

- **Total combinations**: 2^3 = 8
- **Generated test cases**: 7 (excluding "all dependencies enabled" default case)
- **Test time**: All run in parallel, typically 2-5 minutes total

## Choosing Your Testing Approach

### Use Matrix Testing When:
- You need specific custom configurations
- You want to control exactly which scenarios to test
- You need some full deployment tests mixed with validation tests
- You want to customize individual test behavior
- You're testing multiple flavors or regions

### Use Dependency Permutation Testing When:
- You want to test all possible dependency combinations
- You need comprehensive validation without deployment costs
- You want to catch dependency configuration issues early
- You don't want to manually maintain permutation test cases
- You're focused on dependency validation rather than full deployment

### Combined Approach (Recommended)

Use both approaches together for comprehensive coverage:

```golang
package test

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/cloudinfo"
    "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testaddons"
)

// Full deployment test for primary scenario
func TestAddonPrimaryDeployment(t *testing.T) {
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
    assert.NoError(t, err)
}

// Matrix testing for specific custom scenarios
func TestAddonCustomScenarios(t *testing.T) {
    testCases := []testaddons.AddonTestCase{
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
            },
        },
    }

    matrix := testaddons.AddonTestMatrix{
        TestCases: testCases,
        BaseOptions: baseOptions,
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
    assert.NoError(t, err)
}
```

## Performance Comparison

### Matrix Testing
- **Setup Time**: Manual configuration required
- **Flexibility**: Full control over test scenarios
- **Maintenance**: Manual updates when adding scenarios
- **Coverage**: Tests specific scenarios you define

### Dependency Permutation Testing
- **Setup Time**: Minimal configuration required
- **Flexibility**: Automated, no manual control
- **Maintenance**: Zero maintenance, adapts automatically
- **Coverage**: Tests all possible dependency combinations

## Best Practices

### 1. Use Both Approaches Together
Combine matrix testing for specific scenarios with permutation testing for comprehensive dependency validation.

### 2. Optimize for Cost and Time
- Use permutation testing for validation-only dependency testing
- Use matrix testing for specific deployment scenarios
- Mix validation-only and deployment tests in matrices

### 3. Organize Your Tests
```golang
// tests/addon_test.go - Primary deployment test
func TestAddonPrimaryDeployment(t *testing.T) { ... }

// tests/addon_scenarios_test.go - Custom scenarios with matrix testing
func TestAddonCustomScenarios(t *testing.T) { ... }

// tests/addon_permutations_test.go - Dependency permutations
func TestAddonDependencyPermutations(t *testing.T) { ... }
```

### 4. Use Clear Naming
- Use descriptive test names that indicate the testing approach
- Use clear prefixes that identify the test type
- Document which approach is used in each test function

## Summary

### Matrix Testing
✅ **Full Control**: Define exactly which scenarios to test
✅ **Mixed Testing**: Combine deployment and validation tests
✅ **Custom Configuration**: Tailor each test case individually
✅ **Explicit Scenarios**: Clear visibility into what's being tested

### Dependency Permutation Testing
✅ **Comprehensive Coverage**: Tests all possible dependency combinations
✅ **Zero Maintenance**: Automatically adapts to dependency changes
✅ **Cost Effective**: Validation-only testing reduces costs
✅ **Automated Discovery**: No manual dependency configuration

### Combined Approach (Recommended)
Use both approaches together:
- **Matrix testing** for specific custom scenarios and deployment testing
- **Dependency permutation testing** for comprehensive dependency validation
- **Primary deployment test** for full end-to-end validation

This provides the best balance of comprehensive coverage, cost efficiency, and maintainability.
