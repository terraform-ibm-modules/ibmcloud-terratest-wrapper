# Dependency Permutation Testing

The `RunAddonPermutationTest()` method provides automated testing of all dependency permutations for an addon without manual configuration. This feature automatically discovers dependencies from the catalog and generates all enabled/disabled combinations for comprehensive validation testing.

## Overview

Dependency permutation testing solves the problem of manually creating test cases for every possible combination of addon dependencies. Instead of writing individual test cases for each dependency combination, you can use a single method call to test all permutations automatically.

## Key Features

- **Automatic Dependency Discovery**: Discovers all direct dependencies from the catalog metadata
- **Algorithmic Permutation Generation**: Generates all 2^n combinations of dependencies (where n = number of dependencies)
- **Validation-Only Testing**: All permutations use validation-only mode for efficiency and cost savings
- **Comprehensive Final Report**: Automatically generates a detailed summary report with error details and failure patterns
- **Reliable Execution**: Runs all permutations in parallel while guaranteeing final report generation even when individual tests fail
- **Zero Maintenance**: No manual test case creation or maintenance required

## When to Use Permutation Testing

### Use `RunAddonPermutationTest()` when

- You want to test all possible dependency combinations
- You need comprehensive validation without deployment costs
- You want to catch dependency configuration issues early
- You don't want to manually maintain permutation test cases
- You're focused on validation rather than full deployment testing

### Use Manual Matrix Testing when

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

- **Quiet Mode**: Automatically enabled (`QuietMode: true`) to reduce log noise and show clean progress indicators
- **Infrastructure Deployment**: Set to `SkipInfrastructureDeployment: true` for all permutations
- **Parallel Execution**: Uses matrix testing infrastructure for efficient parallel execution with reliable final reporting
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

Runs all permutations in parallel using the matrix testing infrastructure to ensure efficient execution while guaranteeing comprehensive result collection and reliable final reporting regardless of individual test failures.

### 5. Final Report Generation

After all tests complete, automatically generates a comprehensive final report that includes:

- **Executive Summary**: Pass/fail counts and success rates
- **Passing Tests**: Collapsed list of successful permutations
- **Failed Tests**: Detailed error information for each failure, including:
  - Test configuration (which addons were enabled/disabled)
  - Complete error messages for debugging
  - Categorized error types (validation, deployment, configuration, runtime)
- **Failure Pattern Analysis**: Groups failures by common causes for quick scanning
- **Resource Prefix Information**: For correlating with logs if needed

This eliminates the need to dig through individual test logs when debugging failures across many permutations.

### 5. Final Report Generation

After all tests complete, automatically generates a comprehensive final report that includes:

- **Executive Summary**: Pass/fail counts and success rates
- **Passing Tests**: Collapsed list of successful permutations
- **Failed Tests**: Detailed error information for each failure, including:
  - Test configuration (which addons were enabled/disabled)
  - Complete error messages for debugging
  - Categorized error types (validation, deployment, configuration, runtime)
- **Failure Pattern Analysis**: Groups failures by common causes for quick scanning
- **Resource Prefix Information**: For correlating with logs if needed

This eliminates the need to dig through individual test logs when debugging failures across many permutations.

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
- Comprehensive final report eliminates need to dig through individual test logs

### Scalable

- Works with any number of dependencies
- Automatically adjusts to new dependencies
- No code changes required when dependency structure changes

## Example Output

### Quiet Mode with Final Report (Default)

With quiet mode enabled automatically, you'll see clean progress indicators during execution followed by a comprehensive final report:

```
[CloudInfoService] Importing offering: fully-configurable from branch URL...
[CloudInfoService] Imported offering: Cloud automation for Event Notifications...
Running 15 dependency permutation tests for deploy-arch-ibm-event-notifications (quiet mode - minimal output)...
🔄 Starting test: event-notifications-0-disable-kms-cos-account-infra-base-observability
🔄 Setting up test Catalog and Project
🔄 Deploying Configurations to Project
🔄 Validating dependencies
✅ Infrastructure deployment completed
🔄 Cleaning up resources
  ✓ Passed: event-notifications-0-disable-kms-cos-account-infra-base-observability
🔄 Starting test: event-notifications-4-disable-kms-cos-observability
🔄 Setting up test Catalog and Project
🔄 Deploying Configurations to Project
🔄 Validating dependencies
✅ Infrastructure deployment completed
🔄 Cleaning up resources
  ✓ Passed: event-notifications-4-disable-kms-cos-observability
...
  ✓ Passed: event-notifications-14

================================================================================
🧪 PERMUTATION TEST REPORT - Complete
================================================================================
📊 Summary: 15 total tests | ✅ 13 passed (86.7%) | ❌ 2 failed (13.3%)

🎯 PASSING TESTS (13) - Collapsed for brevity
├─ ✅ event-notifications-0-disable-kms-cos-account-infra-base-observability
├─ ✅ event-notifications-4-disable-kms-cos-observability
└─ ... 11 more passing tests (expand with --verbose)

❌ FAILED TESTS (2) - Complete Error Details
┌─────────────────────────────────────────────────────────────────────────────┐
│ 1/2 ❌ event-notifications-7-disable-kms-cos                               │
│     📁 Prefix: en-perm-kms-cos-8472                                         │
│     🔧 Addons: event-notifications=enabled, kms=disabled, cos=disabled      │
│                                                                             │
│     🔴 VALIDATION ERRORS:                                                   │
│     • event-notifications addon requires 'kms' dependency but it's disabled │
│     • event-notifications addon requires 'cos' dependency but it's disabled │
│                                                                             │
│     🔴 CONFIGURATION ERRORS:                                                │
│     • Missing configs: ['deploy-arch-ibm-kms', 'deploy-arch-ibm-cos']      │
│     • Project validation failed: 2 errors, 0 warnings                     │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│ 2/2 ❌ event-notifications-12-disable-observability                        │
│     📁 Prefix: en-perm-obs-9384                                            │
│     🔧 Addons: event-notifications=enabled, observability=disabled,        │
│            [3 others enabled]                                              │
│                                                                             │
│     🔴 DEPLOYMENT ERRORS:                                                   │
│     • TriggerDeployAndWait failed: deployment timeout after 15 minutes     │
│     • Configuration state stuck in 'ApplyingFailed'                        │
│                                                                             │
│     🔴 RUNTIME ERRORS:                                                      │
│     • TestRunAddonTest failed: deployment validation failed                 │
└─────────────────────────────────────────────────────────────────────────────┘

🔍 FAILURE PATTERNS (for quick scanning)
├─ Dependency Issues: 1 test (missing required dependencies)
└─ Deployment Errors: 1 test (TriggerDeployAndWait failures)

📁 Full test logs available if additional context needed
================================================================================

PASS
```

### Verbose Mode

For detailed debugging, override the automatic quiet mode:

```golang
options.QuietMode = false  // Enable verbose output
err := options.RunAddonPermutationTest()
```

### Failure Case

When permutations fail validation, you'll see detailed error information:

```
=== RUN   TestSecretsManagerDependencyPermutations
🔄 Starting test: sm-perm-03-disable-kms
🔄 Setting up test Catalog and Project
🔄 Deploying Configurations to Project
🔄 Validating dependencies
  ✗ Failed: sm-perm-03-disable-kms (error: dependency validation failed: 1 missing configs: [deploy-arch-ibm-kms (v5.1.4, fully-configurable)])
--- FAIL: TestSecretsManagerDependencyPermutations (47.82s)
    --- FAIL: TestSecretsManagerDependencyPermutations/sm-perm-03-disable-kms (12.34s)
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
