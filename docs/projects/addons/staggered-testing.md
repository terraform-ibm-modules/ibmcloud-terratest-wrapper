# Staggered Testing Guide

This guide explains how to use staggered testing to prevent rate limiting in parallel addon tests.

## Problem: Rate Limiting in Parallel Tests

When running multiple addon tests in parallel (e.g., using `AddonTestMatrix`), all tests start simultaneously and can hit the same IBM Cloud APIs at the same time. This often results in rate limiting errors (HTTP 429) like:

```text
Rate limited (429) on GetComponentReferences for 7a4d68b4-cf8b-40cd-a3d1-f49aff526eb3.94cb0f29-2d71-4882-9cfa-3f8e9f606105-global
Retrying GetComponentReferences after 5.472483175s (attempt 2/10)
```

## Solution: Staggered Test Starts

The framework now supports **staggered test starts** where parallel tests are delayed by a configurable amount to spread out API calls and prevent rate limiting.

## Basic Usage

### Default Stagger

```go
func TestRunAddonTests(t *testing.T) {
    matrix := testaddons.AddonTestMatrix{
        TestCases: []testaddons.AddonTestCase{
            {Name: "Test1", Prefix: "test1"},
            {Name: "Test2", Prefix: "test2"},
            {Name: "Test3", Prefix: "test3"},
        },
        BaseOptions: testaddons.TestAddonsOptionsDefault(&testaddons.TestAddonOptions{
            Testing: t,
            Prefix:  "matrix-test",
        }),
        // StaggerDelay has a default value if not specified
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

**Result**: Test1 starts immediately, Test2 starts after 5 seconds, Test3 starts after 10 seconds.

### Custom Stagger Delays

```go
// Use 10 second stagger for heavier workloads
matrix.StaggerDelay = testaddons.StaggerDelay(10 * time.Second)

// Use 20 second stagger for very heavy workloads
matrix.StaggerDelay = testaddons.StaggerDelay(20 * time.Second)

// Disable staggering - all tests start simultaneously
matrix.StaggerDelay = testaddons.StaggerDelay(0)
```

### Advanced Usage

For complex test scenarios, you can customize the stagger delay based on your workload:

```go
func TestHeavyWorkloadMatrix(t *testing.T) {
    // Heavy workload with many parallel tests
    testCases := make([]testaddons.AddonTestCase, 20)
    for i := 0; i < 20; i++ {
        testCases[i] = testaddons.AddonTestCase{
            Name:   fmt.Sprintf("Test%d", i+1),
            Prefix: fmt.Sprintf("test%d", i+1),
        }
    }

    matrix := testaddons.AddonTestMatrix{
        TestCases:    testCases,
        BaseOptions:  baseOptions,
        StaggerDelay: testaddons.StaggerDelay(20 * time.Second), // 20 second stagger for heavy workloads
        AddonConfigFunc: func(options *testaddons.TestAddonOptions, testCase testaddons.AddonTestCase) cloudinfo.AddonConfig {
            return cloudinfo.NewAddonConfigTerraform(
                options.Prefix,
                "heavy-addon",
                "enterprise",
                map[string]interface{}{
                    "prefix": options.Prefix,
                },
            )
        },
    }

    baseOptions.RunAddonTestMatrix(matrix)
}
```

## How It Works

1. **Test 1**: Starts immediately (no delay)
2. **Test 2**: Waits `StaggerDelay` duration, then starts
3. **Test 3**: Waits `2 * StaggerDelay` duration, then starts
4. **Test N**: Waits `(N-1) * StaggerDelay` duration, then starts

## Choosing the Right Stagger Delay

| Scenario | Recommended Delay | Usage |
|----------|------------------|-------|
| Light workloads (2-5 tests) | 5 seconds | Default (no StaggerDelay needed) |
| Normal workloads (5-10 tests) | 10 seconds | `StaggerDelay(10 * time.Second)` |
| Heavy workloads (10+ tests) | 20 seconds | `StaggerDelay(20 * time.Second)` |
| No rate limiting concerns | 0 seconds | `StaggerDelay(0)` |

## Impact on Test Runtime

Staggering does increase total test setup time, but tests still run in parallel once started:

**Without staggering (8 tests)**:

- All tests start at T+0
- Risk of rate limiting and retries

**With 5-second staggering (8 tests)**:

- Test 1 starts at T+0
- Test 8 starts at T+35s
- But all tests run in parallel once started
- Reduces rate limiting and improves reliability

## Monitoring Stagger Activity

The framework logs stagger activity to help you monitor the delays:

```text
[Test2 - STAGGER] Delaying test start by 5s to prevent rate limiting (test 2/8)
[Test3 - STAGGER] Delaying test start by 10s to prevent rate limiting (test 3/8)
[Test4 - STAGGER] Delaying test start by 15s to prevent rate limiting (test 4/8)
```

## Best Practices

1. **Start with Default**: Use the default 5-second stagger for most scenarios
2. **Adjust Based on Load**: Increase for heavy workloads, decrease for light ones
3. **Monitor Logs**: Watch for rate limiting messages and adjust accordingly
4. **Balance Speed vs Reliability**: Shorter delays = faster tests but higher rate limit risk

## Alternative Approaches

If staggering doesn't solve rate limiting issues, consider:

1. **Reduce Parallelism**: Don't use `t.Parallel()` for all tests
2. **Split Test Suites**: Break large test matrices into smaller groups
3. **Use Validation-Only Tests**: Set `SkipInfrastructureDeployment: true` for some tests

## Example: Complete Staggered Test

```go
package test

import (
    "testing"
    "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/cloudinfo"
    "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testaddons"
)

func TestStaggeredAddonMatrix(t *testing.T) {
    testCases := []testaddons.AddonTestCase{
        {Name: "BasicConfiguration", Prefix: "basic"},
        {Name: "AdvancedConfiguration", Prefix: "advanced"},
        {Name: "ProductionConfiguration", Prefix: "production"},
        {Name: "ValidationOnlyTest", Prefix: "validate", SkipInfrastructureDeployment: true},
    }

    baseOptions := testaddons.TestAddonsOptionsDefault(&testaddons.TestAddonOptions{
        Testing:       t,
        Prefix:        "staggered-test",
        ResourceGroup: "my-resource-group",
    })

    matrix := testaddons.AddonTestMatrix{
        TestCases:    testCases,
        BaseOptions:  baseOptions,
        StaggerDelay: testaddons.StaggerDelay(10 * time.Second), // 10 second stagger between tests
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

This example will:

1. Start "BasicConfiguration" immediately
2. Start "AdvancedConfiguration" after 10 seconds
3. Start "ProductionConfiguration" after 20 seconds
4. Start "ValidationOnlyTest" after 30 seconds

All tests run in parallel once started, with minimal risk of rate limiting.
