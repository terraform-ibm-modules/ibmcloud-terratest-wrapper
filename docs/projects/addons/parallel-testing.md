# Parallel Testing Guide

This guide covers how to run multiple addon test configurations in parallel using matrix testing approaches. Parallel testing is particularly useful when you want to test different configurations, dependencies, or inputs for the same addon.

## Matrix Testing Pattern

The framework provides a reusable pattern for defining and running multiple test cases in parallel. This approach is ideal for testing different addon configurations efficiently.

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

func setupAddonOptions(t *testing.T, prefix string) *testaddons.TestAddonOptions {
    options := testaddons.TestAddonsOptionsDefault(&testaddons.TestAddonOptions{
        Testing:       t,
        Prefix:        prefix,
        ResourceGroup: resourceGroup,
    })
    return options
}

// TestRunAddonTests runs addon tests in parallel using a matrix approach
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

## Framework-Provided Matrix Testing

The framework provides built-in support for matrix testing to reduce boilerplate code. Here's how to use the framework's matrix testing utilities:

### Using AddonTestCase and RunAddonTestMatrix

```golang
package test

import (
    "testing"
    "github.com/IBM/go-sdk-core/v5/core"
    "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/cloudinfo"
    "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testaddons"
)

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

## Common Matrix Testing Patterns

### Testing Different Dependency Configurations

```golang
func TestDifferentDependencies(t *testing.T) {
    testCases := []testaddons.AddonTestCase{
        {
            Name:   "NoDependencies",
            Prefix: "nodeps",
        },
        {
            Name:   "WithResourceGroup",
            Prefix: "rg",
            Dependencies: []cloudinfo.AddonConfig{
                {
                    OfferingName:   "deploy-arch-ibm-account-infra-base",
                    OfferingFlavor: "resource-group-only",
                    Enabled:        core.BoolPtr(true),
                },
            },
        },
        {
            Name:   "WithMultipleDependencies",
            Prefix: "multi",
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
        TestCases:       testCases,
        BaseSetupFunc:   setupBasicOptions,
        AddonConfigFunc: createStandardAddonConfig,
    }

    testaddons.RunAddonTestMatrix(t, matrix)
}
```

### Testing Different Input Configurations

```golang
func TestDifferentInputs(t *testing.T) {
    testCases := []testaddons.AddonTestCase{
        {
            Name:   "USEast",
            Prefix: "useast",
            Inputs: map[string]interface{}{
                "region": "us-east",
                "zone":   "us-east-1a",
            },
        },
        {
            Name:   "EuropeGB",
            Prefix: "eugb",
            Inputs: map[string]interface{}{
                "region": "eu-gb",
                "zone":   "eu-gb-1a",
            },
        },
        {
            Name:   "AsiaPacific",
            Prefix: "ap",
            Inputs: map[string]interface{}{
                "region": "jp-tok",
                "zone":   "jp-tok-1a",
            },
        },
    }

    matrix := testaddons.AddonTestMatrix{
        TestCases:       testCases,
        BaseSetupFunc:   setupBasicOptions,
        AddonConfigFunc: createRegionalAddonConfig,
    }

    testaddons.RunAddonTestMatrix(t, matrix)
}
```

### Testing Different Flavors

```golang
func TestDifferentFlavors(t *testing.T) {
    testCases := []testaddons.AddonTestCase{
        {
            Name:   "BasicFlavor",
            Prefix: "basic",
        },
        {
            Name:   "StandardFlavor",
            Prefix: "std",
        },
        {
            Name:   "AdvancedFlavor",
            Prefix: "adv",
        },
    }

    matrix := testaddons.AddonTestMatrix{
        TestCases: testCases,
        BaseSetupFunc: func(testCase testaddons.AddonTestCase) *testaddons.TestAddonOptions {
            return testaddons.TestAddonsOptionsDefault(&testaddons.TestAddonOptions{
                Testing:       t,
                Prefix:        testCase.Prefix,
                ResourceGroup: "test-rg",
            })
        },
        AddonConfigFunc: func(options *testaddons.TestAddonOptions, testCase testaddons.AddonTestCase) cloudinfo.AddonConfig {
            flavorMap := map[string]string{
                "BasicFlavor":    "basic",
                "StandardFlavor": "standard",
                "AdvancedFlavor": "advanced",
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

## Best Practices for Parallel Testing

### 1. Resource Naming

- Always use unique prefixes for each test case to avoid resource conflicts
- Use the test case name or a variant in the prefix when possible

### 2. Cost Considerations

- Parallel testing can increase cloud costs due to simultaneous resource provisioning
- Consider the cost of the resources being deployed when deciding on parallelism
- Use `t.Parallel()` selectively based on the addon's resource cost and deployment time

### 3. Test Isolation

- Ensure test cases are completely independent
- Don't share resources between test cases
- Each test case should be able to run successfully in isolation

### 4. Timeout Considerations

- Parallel tests may take longer due to resource contention
- Consider adjusting `DeployTimeoutMinutes` for parallel execution
- Monitor for cloud service rate limits

### 5. Cleanup

- Each test case should clean up its own resources
- Use `SkipTestTearDown: false` (default) to ensure cleanup
- Consider using different resource groups for better isolation

## When to Use Parallel Testing

**Good Use Cases:**

- Testing different dependency combinations
- Testing different input parameters
- Testing different flavors of the same addon
- Testing against different regions
- Quick-deploying, low-cost resources

**Consider Serial Testing When:**

- Resources are expensive or have strict quotas
- Deployment times are very long
- Testing upgrade scenarios
- Cloud service has strict rate limits
- Debugging specific issues

## Example: Cost-Effective Parallel Testing

This example shows how to run parallel tests for a low-cost addon like KMS:

```golang
// No cost for the KMS instance and its quick to run, so we can run these in parallel
// and fully deploy each time. This can be used as an example of how to run
// multiple addon tests in parallel
func TestRunKMSAddonTests(t *testing.T) {
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
        },
        BaseSetupFunc: func(testCase testaddons.AddonTestCase) *testaddons.TestAddonOptions {
            return testaddons.TestAddonsOptionsDefault(&testaddons.TestAddonOptions{
                Testing:       t,
                Prefix:        testCase.Prefix,
                ResourceGroup: resourceGroup,
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
