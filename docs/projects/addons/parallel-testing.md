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
                "my-addon",
                "standard",
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

## Matrix Testing Structure and Configuration

The `AddonTestMatrix` provides a declarative way to define multiple test cases and run them in parallel. This is the **primary approach** for parallel testing.

### Key Components

- **TestCases**: An array of `AddonTestCase` structures defining the individual test scenarios
- **BaseSetupFunc**: A function that creates the base `TestAddonOptions` for each test case
  - Signature: `func(testCase AddonTestCase) *TestAddonOptions`
- **AddonConfigFunc**: A function that creates the specific addon configuration for each test case
  - Signature: `func(options *TestAddonOptions, testCase AddonTestCase) cloudinfo.AddonConfig`

### AddonTestCase Configuration Options

Each test case supports several configuration options:

- **Name**: Test case name that appears in test output and log messages
- **Prefix**: Unique prefix for resource naming in this test case
- **Dependencies**: Addon dependencies to configure for this test case
- **Inputs**: Additional inputs to merge with the base addon configuration
- **SkipTearDown**: Skip cleanup for this specific test case (useful for debugging)
- **SkipInfrastructureDeployment**: Skip actual infrastructure deployment for validation-only testing

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
- **BaseSetupFunc**: A function that creates the base `TestAddonOptions` for each test case, typically setting common configurations like resource group and testing context
  - Signature: `func(testCase AddonTestCase) *TestAddonOptions`
- **AddonConfigFunc**: A function that creates the specific addon configuration for each test case, allowing customization based on the test case parameters
  - Signature: `func(options *TestAddonOptions, testCase AddonTestCase) cloudinfo.AddonConfig`

This approach separates concerns by letting you define:

1. **What to test** (in `TestCases`)
2. **How to set up the test environment** (in `BaseSetupFunc`)
3. **How to configure the addon** (in `AddonConfigFunc`)

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
                "my-addon",
                "standard",
                map[string]interface{}{
                    "prefix": options.Prefix,
                    "region": "us-south", // Default region, can be overridden by testCase.Inputs
                },
            )
        },
    }

    testaddons.RunAddonTestMatrix(t, matrix)
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
        BaseSetupFunc: func(testCase testaddons.AddonTestCase) *testaddons.TestAddonOptions {
            return testaddons.TestAddonsOptionsDefault(&testaddons.TestAddonOptions{
                Testing:       t,
                Prefix:        testCase.Prefix,
                ResourceGroup: "test-resource-group",
            })
        },
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

    testaddons.RunAddonTestMatrix(t, matrix)
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
        BaseSetupFunc: func(testCase testaddons.AddonTestCase) *testaddons.TestAddonOptions {
            return testaddons.TestAddonsOptionsDefault(&testaddons.TestAddonOptions{
                Testing:       t,
                Prefix:        testCase.Prefix,
                ResourceGroup: "flavor-test-rg",
            })
        },
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

    testaddons.RunAddonTestMatrix(t, matrix)
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
