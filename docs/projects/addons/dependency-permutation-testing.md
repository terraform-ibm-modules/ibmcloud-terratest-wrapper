# Dependency Permutation Testing

The `RunAddonPermutationTest()` method provides automated testing of all dependency permutations for an addon without manual configuration. This feature automatically discovers dependencies from the catalog and generates all enabled/disabled combinations for comprehensive validation testing.

## Overview

Dependency permutation testing solves the problem of manually creating test cases for every possible combination of addon dependencies. Instead of writing individual test cases for each dependency combination, you can use a single method call to test all permutations automatically.

## Key Features

- **Automatic Dependency Discovery**: Discovers all direct dependencies from the catalog metadata
- **Algorithmic Permutation Generation**: Generates all 2^n combinations of dependencies (where n = number of dependencies)
- **Validation-Only Testing**: All permutations use validation-only mode for efficiency and cost savings
- **Failure-Only Logging**: Reduces noise by showing only failed permutations
- **Parallel Execution**: Leverages matrix testing infrastructure for efficient parallel execution
- **Zero Maintenance**: No manual test case creation or maintenance required

## When to Use Permutation Testing

### Use `RunAddonPermutationTest()` when:
- You want to test all possible dependency combinations
- You need comprehensive validation without deployment costs
- You want to catch dependency configuration issues early
- You don't want to manually maintain permutation test cases
- You're focused on validation rather than full deployment testing

### Use Manual Matrix Testing when:
- You need specific custom configurations for each test case
- You want to control exactly which scenarios to test
- You need some full deployment tests mixed with validation tests
- You want to customize individual test behavior

## Basic Usage

### Simple Permutation Test

```golang
package test

import (
    "testing"
    "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/cloudinfo"
    "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testaddons"
)

func TestSecretsManagerDependencyPermutations(t *testing.T) {
    t.Parallel()

    options := testaddons.TestAddonsOptionsDefault(&testaddons.TestAddonOptions{
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
    if err != nil {
        t.Fatalf("Dependency permutation test failed: %v", err)
    }
}
```

### Alternative Pattern with Error Handling

```golang
func TestKMSAddonPermutations(t *testing.T) {
    t.Parallel()

    options := testaddons.TestAddonsOptionsDefault(&testaddons.TestAddonOptions{
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
    assert.NoError(t, err, "Dependency permutation test should not fail")
}
```

## Configuration Requirements

### Required Fields

- **Testing**: The test instance (required)
- **Prefix**: Unique prefix for resource naming (required)
- **AddonConfig.OfferingName**: The name of the addon to test (required)
- **AddonConfig.OfferingFlavor**: The flavor of the addon to test (required)
- **AddonConfig.Inputs**: Input variables for the addon configuration (required)

### Automatic Settings

The framework automatically configures the following settings for permutation tests:

- **Logging Mode**: Set to "failure_only" to reduce log noise
- **Infrastructure Deployment**: Set to `SkipInfrastructureDeployment: true` for all permutations
- **Parallel Execution**: Uses `RunAddonTestMatrix` for efficient parallel testing
- **Validation Focus**: All permutations perform validation-only testing

## How It Works

### 1. Dependency Discovery
The method automatically queries the IBM Cloud catalog to discover all direct dependencies of the specified addon using the addon's metadata.

### 2. Permutation Generation
Creates all 2^n combinations of discovered dependencies being enabled/disabled:
- **Root addon**: Always present (doesn't participate in permutation)
- **Dependencies**: Each dependency can be enabled or disabled
- **Combinations**: For n dependencies, generates 2^n total combinations

### 3. Default Filtering
Excludes the "on by default" case since this is typically covered by existing default configuration tests.

### 4. Parallel Execution
Uses the existing matrix test infrastructure to run all permutations in parallel for efficiency.

## Generated Test Cases

### Example: Addon with 3 Dependencies

For an addon with 3 dependencies (KMS, Observability, EventNotifications):

**Structure**: 1 root addon (always present) + 3 dependencies = 2^3 = 8 total combinations
**Generated**: 8 - 1 = 7 test cases (excluding "on by default" case)

1. `sm-perm-01` - All dependencies disabled
2. `sm-perm-02` - Only EventNotifications enabled
3. `sm-perm-03` - Only Observability enabled
4. `sm-perm-04` - Observability + EventNotifications enabled
5. `sm-perm-05` - Only KMS enabled
6. `sm-perm-06` - KMS + EventNotifications enabled
7. `sm-perm-07` - KMS + Observability enabled

The "all dependencies enabled" case is excluded as it represents the default configuration.

## Benefits

### Comprehensive Coverage
- Tests all possible dependency combinations automatically
- Catches dependency configuration issues that manual testing might miss
- Ensures addon works correctly with any dependency configuration

### Zero Maintenance
- No need to manually define test cases for each permutation
- Automatically adapts when dependencies change
- Reduces test code maintenance burden

### Cost Effective
- Validation-only mode avoids infrastructure deployment costs
- Parallel execution reduces total test time
- Failure-only logging reduces noise and focuses on issues

### Scalable
- Works with any number of dependencies
- Automatically adjusts to new dependencies
- No code changes required when dependency structure changes

## Example Output

### Success Case
With failure-only logging enabled (automatic), successful permutations produce minimal output:
```
=== RUN   TestSecretsManagerDependencyPermutations
--- PASS: TestSecretsManagerDependencyPermutations (45.23s)
PASS
```

### Failure Case
When permutations fail validation, you'll see detailed error information:
```
=== RUN   TestSecretsManagerDependencyPermutations
=== RUN   TestSecretsManagerDependencyPermutations/sm-perm-03
    permutation_test.go:42: Dependency validation failed: KMS dependency required but not enabled
--- FAIL: TestSecretsManagerDependencyPermutations (47.82s)
    --- FAIL: TestSecretsManagerDependencyPermutations/sm-perm-03 (12.34s)
```

## Comparing with Manual Matrix Testing

### Dependency Permutation Testing (Automated)
**Best for**: Comprehensive validation of all dependency combinations

```golang
// Single method call tests all permutations
func TestAddonPermutations(t *testing.T) {
    options := testaddons.TestAddonsOptionsDefault(&testaddons.TestAddonOptions{
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

### Manual Matrix Testing (Explicit Control)
**Best for**: Custom scenarios with specific configurations

```golang
// Manual control over each test case
func TestAddonMatrix(t *testing.T) {
    testCases := []testaddons.AddonTestCase{
        {
            Name:   "PrimaryDeployment",
            Prefix: "primary",
            // Full deployment
        },
        {
            Name:   "CustomValidation",
            Prefix: "custom",
            SkipInfrastructureDeployment: true,
            Dependencies: []cloudinfo.AddonConfig{
                {
                    OfferingName: "specific-dependency",
                    OfferingFlavor: "custom-flavor",
                    Enabled: core.BoolPtr(true),
                },
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
```

## Best Practices

### 1. Use Descriptive Prefixes
Use clear, short prefixes that identify your addon:
```golang
options := testaddons.TestAddonsOptionsDefault(&testaddons.TestAddonOptions{
    Testing: t,
    Prefix:  "sm-perm",  // Clear abbreviation for secrets-manager-permutation
    // ...
})
```

### 2. Include Required Configuration
Ensure all required inputs are provided:
```golang
AddonConfig: cloudinfo.AddonConfig{
    OfferingName:   "deploy-arch-ibm-secrets-manager",
    OfferingFlavor: "fully-configurable",
    Inputs: map[string]interface{}{
        "prefix":                       "sm-perm",
        "region":                       "us-south",
        "existing_resource_group_name": "default",
        "service_plan":                 "trial",
        // Include all required inputs for your addon
    },
}
```

### 3. Use Appropriate Service Plans
Choose cost-effective service plans for testing:
```golang
Inputs: map[string]interface{}{
    "service_plan": "trial",          // Use trial/free plans when available
    "enable_platform_metrics": false, // Disable expensive features
}
```

### 4. Run as Parallel Tests
Always mark permutation tests as parallel:
```golang
func TestAddonPermutations(t *testing.T) {
    t.Parallel()  // Enable parallel execution

    // Test implementation
}
```

## Integration with Existing Testing

### Combined Testing Strategy
Use permutation testing alongside other testing approaches:

```golang
package test

import (
    "testing"
    "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testaddons"
    "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/cloudinfo"
)

// Full deployment test for primary scenario
func TestAddonFullDeployment(t *testing.T) {
    t.Parallel()

    options := testaddons.TestAddonsOptionsDefault(&testaddons.TestAddonOptions{
        Testing: t,
        Prefix:  "addon-deploy",
        AddonConfig: cloudinfo.NewAddonConfigTerraform(
            "addon-deploy",
            "my-addon",
            "standard",
            map[string]interface{}{
                "prefix": "addon-deploy",
                "region": "us-south",
            },
        ),
    })

    err := options.RunAddonTest()
    assert.NoError(t, err)
}

// Permutation testing for dependency validation
func TestAddonDependencyPermutations(t *testing.T) {
    t.Parallel()

    options := testaddons.TestAddonsOptionsDefault(&testaddons.TestAddonOptions{
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

## Troubleshooting

### Common Issues

**No Dependencies Found**
```
Error: No dependencies found for addon 'my-addon'
```
- Verify the addon name and flavor are correct
- Check that the addon has dependencies defined in the catalog
- Ensure the addon is properly imported in the catalog

**Validation Failures**
```
Error: Dependency validation failed for permutation 'addon-perm-03'
```
- Check that all required inputs are provided
- Verify dependency configurations are valid
- Review the specific error message for details

**Timeout Issues**
```
Error: Test timed out after 30 minutes
```
- Large numbers of dependencies can create many permutations
- Consider using failure-only logging to reduce overhead
- Verify parallel execution is enabled

### Debug Tips

1. **Check Dependency Discovery**: Use the framework's logging to see which dependencies were discovered
2. **Verify Inputs**: Ensure all required inputs are provided and valid
3. **Test Individual Permutations**: Run specific permutations manually to isolate issues
4. **Review Catalog Metadata**: Verify the addon's catalog metadata is correct

## Summary

Dependency permutation testing provides a powerful way to automatically test all possible dependency combinations for your addon. It's particularly valuable for:

- **Comprehensive validation** of dependency configurations
- **Cost-effective testing** without infrastructure deployment
- **Automated discovery** of dependency-related issues
- **Zero-maintenance** testing that adapts to changes

Use `RunAddonPermutationTest()` when you need comprehensive dependency validation, and combine it with manual matrix testing or full deployment tests for complete coverage of your addon's functionality.
