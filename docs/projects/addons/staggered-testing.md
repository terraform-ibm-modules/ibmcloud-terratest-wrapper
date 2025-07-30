# Staggered Testing Guide

This guide explains how to use staggered testing to prevent rate limiting in parallel addon tests.

## Problem: Rate Limiting in Parallel Tests

When running multiple addon tests in parallel (e.g., using `AddonTestMatrix`), all tests start simultaneously and can hit the same IBM Cloud APIs at the same time. This often results in rate limiting errors (HTTP 429) like:

```text
Rate limited (429) on GetComponentReferences for 7a4d68b4-cf8b-40cd-a3d1-f49aff526eb3.94cb0f29-2d71-4882-9cfa-3f8e9f606105-global
Retrying GetComponentReferences after 5.472483175s (attempt 2/10)
```

## Solution: Batched Staggered Test Starts

The framework supports **batched staggered test starts** where parallel tests are organized into batches with configurable delays both between batches and within batches. This approach provides efficient API rate limiting protection while minimizing excessive delays for large test suites.

### Why Batched Staggering?

**Previous Linear Approach**: Tests were staggered linearly (test N waits N × stagger_delay), which caused excessive delays for large test suites:
- Test 50: 8+ minutes delay
- Test 100: 16+ minutes delay

**New Batched Approach**: Tests are grouped into batches with larger delays between batches and smaller delays within batches:
- Test 50: ~1 minute delay (87% improvement)
- Test 100: ~2 minutes delay (87% improvement)

## Basic Usage

### Default Batched Stagger

```go
func TestRunAddonTests(t *testing.T) {
    matrix := testaddons.AddonTestMatrix{
        TestCases: []testaddons.AddonTestCase{
            {Name: "Test1", Prefix: "test1"},
            {Name: "Test2", Prefix: "test2"},
            {Name: "Test3", Prefix: "test3"},
            {Name: "Test4", Prefix: "test4"},
            {Name: "Test5", Prefix: "test5"},
            {Name: "Test6", Prefix: "test6"},
            {Name: "Test7", Prefix: "test7"},
            {Name: "Test8", Prefix: "test8"},
            {Name: "Test9", Prefix: "test9"},
        },
        BaseOptions: testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
            Testing: t,
            Prefix:  "matrix-test",
        }),
        // Batched staggering enabled by default (8 tests per batch, 10s between batches, 2s within batches)
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

**Default Batched Result**:
- **Batch 1 (Tests 1-8)**: Test1 starts immediately, Test2 after 2s, Test3 after 4s, ..., Test8 after 14s
- **Batch 2 (Test 9)**: Test9 starts after 10s (batch delay) = 10s total

### Custom Batched Stagger Configuration

```go
// Custom batch configuration for high-volume tests (20 tests per batch, 15s between batches, 1s within batches)
matrix := testaddons.AddonTestMatrix{
    TestCases:        testCases,
    BaseOptions:      baseOptions,
    StaggerDelay:     testaddons.StaggerDelay(15 * time.Second),        // Delay between batches
    StaggerBatchSize: testaddons.StaggerBatchSize(20),                  // Tests per batch
    WithinBatchDelay: testaddons.WithinBatchDelay(1 * time.Second),     // Delay within each batch
    AddonConfigFunc:  configFunc,
}

// Smaller batches for API-sensitive environments (4 tests per batch, 20s between batches, 5s within batches)
matrix := testaddons.AddonTestMatrix{
    TestCases:        testCases,
    BaseOptions:      baseOptions,
    StaggerDelay:     testaddons.StaggerDelay(20 * time.Second),        // Longer delay between batches
    StaggerBatchSize: testaddons.StaggerBatchSize(4),                   // Smaller batches
    WithinBatchDelay: testaddons.WithinBatchDelay(5 * time.Second),     // Longer delay within batches
    AddonConfigFunc:  configFunc,
}

// Disable batching to use linear staggering (original behavior, not recommended for >20 tests)
matrix := testaddons.AddonTestMatrix{
    TestCases:        testCases,
    BaseOptions:      baseOptions,
    StaggerDelay:     testaddons.StaggerDelay(10 * time.Second),        // Linear delay
    StaggerBatchSize: testaddons.StaggerBatchSize(0),                   // Disable batching
    AddonConfigFunc:  configFunc,
}

// Disable all staggering - all tests start simultaneously
matrix := testaddons.AddonTestMatrix{
    TestCases:       testCases,
    BaseOptions:     baseOptions,
    StaggerDelay:    testaddons.StaggerDelay(0),                        // No delays
    AddonConfigFunc: configFunc,
}
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

## How Batched Staggering Works

### Batching Algorithm

Tests are organized into batches using the following calculation:

```
batchNumber = testIndex / batchSize
inBatchIndex = testIndex % batchSize
staggerWait = (batchNumber * staggerDelay) + (inBatchIndex * withinBatchDelay)
```

### Example: 12 Tests with Default Settings

**Configuration**: 8 tests per batch, 10s between batches, 2s within batches

| Test | Batch | In-Batch Index | Calculation | Total Wait |
|------|-------|----------------|-------------|------------|
| 1    | 0     | 0              | (0×10) + (0×2) | 0s         |
| 2    | 0     | 1              | (0×10) + (1×2) | 2s         |
| 8    | 0     | 7              | (0×10) + (7×2) | 14s        |
| 9    | 1     | 0              | (1×10) + (0×2) | 10s        |
| 12   | 1     | 3              | (1×10) + (3×2) | 16s        |

### Linear vs Batched Comparison

**Linear Staggering (Old)**: `testIndex × staggerDelay`
- Test 1: 0s, Test 2: 10s, Test 3: 20s, Test 12: 110s

**Batched Staggering (New)**: Batch-based calculation
- Test 1: 0s, Test 2: 2s, Test 3: 4s, Test 12: 16s (85% improvement!)

## Choosing the Right Batch Configuration

### Recommended Configurations

| Scenario | Batch Size | Between Batches | Within Batch | Usage |
|----------|------------|----------------|--------------|-------|
| **Light workloads (2-10 tests)** | 8 (default) | 10s | 2s | Default configuration |
| **Normal workloads (10-25 tests)** | 8-12 | 10-15s | 1-2s | `StaggerBatchSize(10)` + `StaggerDelay(15s)` |
| **Heavy workloads (25+ tests)** | 15-25 | 15-20s | 1s | `StaggerBatchSize(20)` + `WithinBatchDelay(1s)` |
| **API-sensitive environments** | 4-6 | 20-30s | 5s | `StaggerBatchSize(4)` + longer delays |
| **No rate limiting concerns** | N/A | 0s | N/A | `StaggerDelay(0)` |
| **Legacy linear staggering** | 0 (disable) | 10-20s | N/A | `StaggerBatchSize(0)` + `StaggerDelay(15s)` |

### Configuration Guidelines

**Batch Size Guidelines:**
- **8-12**: Default range, good for most scenarios
- **4-6**: High API sensitivity environments
- **15-25**: Low API sensitivity, faster execution
- **0**: Disable batching (use linear staggering)

**Between Batch Delays:**
- **5-15 seconds**: Most scenarios
- **20-30 seconds**: High API sensitivity environments
- **0 seconds**: Disable all staggering

**Within Batch Delays:**
- **1-3 seconds**: Most scenarios
- **5+ seconds**: High API sensitivity environments
- **0.5-1 second**: Low API sensitivity, faster execution

## Impact on Test Runtime

Batched staggering significantly reduces delays compared to linear staggering while maintaining rate limiting protection:

### Runtime Comparison Examples

**50 Tests - Linear vs Batched Staggering**:

| Approach | Test 1 | Test 25 | Test 50 | Total Setup |
|----------|--------|---------|---------|-------------|
| **No staggering** | 0s | 0s | 0s | 0s (high rate limit risk) |
| **Linear (old)** | 0s | 4m | 8m10s | 8m10s |
| **Batched (new)** | 0s | 32s | 1m2s | 1m2s (87% improvement) |

**100 Tests - Scaling Comparison**:

| Approach | Test 50 | Test 100 | Total Setup |
|----------|---------|----------|-------------|
| **Linear (old)** | 8m10s | 16m30s | 16m30s |
| **Batched (new)** | 1m2s | 2m6s | 2m6s (87% improvement) |

### Key Benefits

- **Tests still run in parallel** once started
- **Dramatic reduction in setup delays** for large test suites
- **Maintains rate limiting protection** through batch boundaries
- **Scalable approach** that works efficiently with any number of tests

## Monitoring Stagger Activity

The framework logs detailed stagger activity to help you monitor batch progress:

### Batched Stagger Logs

```text
[Test1 - STAGGER] Starting immediately (batch 1/3, position 1/8, delay: 0s)
[Test2 - STAGGER] Delaying test start by 2s (batch 1/3, position 2/8, within-batch delay)
[Test8 - STAGGER] Delaying test start by 14s (batch 1/3, position 8/8, within-batch delay)
[Test9 - STAGGER] Delaying test start by 10s (batch 2/3, position 1/8, batch boundary delay)
[Test10 - STAGGER] Delaying test start by 12s (batch 2/3, position 2/8, batch + within-batch delay)
```

### Linear Stagger Logs (when batching disabled)

```text
[Test2 - STAGGER] Delaying test start by 10s to prevent rate limiting (test 2/20, linear mode)
[Test3 - STAGGER] Delaying test start by 20s to prevent rate limiting (test 3/20, linear mode)
[Test4 - STAGGER] Delaying test start by 30s to prevent rate limiting (test 4/20, linear mode)
```

### Log Information Provided

- **Current test position**: Which test is starting
- **Batch information**: Which batch and position within batch
- **Delay reason**: Whether it's a batch boundary or within-batch delay
- **Total tests**: Overall progress context

## Best Practices

### 1. **Start with Defaults**
Use the default batched configuration for most scenarios (8 tests per batch, 10s between batches, 2s within batches):

```go
matrix := testaddons.AddonTestMatrix{
    TestCases:   testCases,
    BaseOptions: baseOptions,
    // Default batching is enabled automatically
    AddonConfigFunc: configFunc,
}
```

### 2. **Adjust Based on Test Suite Size**
- **Small suites (< 10 tests)**: Use defaults
- **Medium suites (10-25 tests)**: Consider `StaggerBatchSize(10)` or `StaggerBatchSize(12)`
- **Large suites (25+ tests)**: Use `StaggerBatchSize(15-25)` with `WithinBatchDelay(1s)`

### 3. **Tune for API Sensitivity**
- **High API sensitivity**: Smaller batches (4-6) with longer delays (20-30s between batches)
- **Low API sensitivity**: Larger batches (15-25) with shorter delays (1s within batches)

### 4. **Monitor and Adjust**
- **Watch stagger logs**: Monitor batch progress and delay patterns
- **Check for rate limiting**: Look for HTTP 429 errors and retry messages
- **Measure total runtime**: Balance speed vs reliability based on your needs

### 5. **Consider Legacy Linear Mode**
Only use linear staggering (`StaggerBatchSize(0)`) for:
- **Very small test suites** (< 5 tests) where batching overhead isn't beneficial
- **Debugging purposes** when you need predictable linear delays
- **Gradual migration** from existing linear configurations

## Alternative Approaches

If staggering doesn't solve rate limiting issues, consider:

1. **Reduce Parallelism**: Don't use `t.Parallel()` for all tests
2. **Split Test Suites**: Break large test matrices into smaller groups
3. **Use Validation-Only Tests**: Set `SkipInfrastructureDeployment: true` for some tests

## Complete Examples

### Example 1: Default Batched Staggering

```go
package test

import (
    "testing"
    "time"
    "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/cloudinfo"
    "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testaddons"
)

func TestDefaultBatchedStaggering(t *testing.T) {
    // Create 12 test cases to demonstrate batching
    testCases := make([]testaddons.AddonTestCase, 12)
    for i := 0; i < 12; i++ {
        testCases[i] = testaddons.AddonTestCase{
            Name:   fmt.Sprintf("Test%d", i+1),
            Prefix: fmt.Sprintf("test%d", i+1),
        }
    }

    baseOptions := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
        Testing:       t,
        Prefix:        "batched-test",
        ResourceGroup: "my-resource-group",
    })

    matrix := testaddons.AddonTestMatrix{
        TestCases:   testCases,
        BaseOptions: baseOptions,
        // Default batching: 8 tests per batch, 10s between batches, 2s within batches
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

**This example executes as**:
- **Batch 1 (Tests 1-8)**: Test1 starts immediately, Test2 after 2s, ..., Test8 after 14s
- **Batch 2 (Tests 9-12)**: Test9 starts after 10s, Test10 after 12s, Test11 after 14s, Test12 after 16s

### Example 2: Custom High-Volume Configuration

```go
func TestHighVolumeCustomBatching(t *testing.T) {
    // Create 50 test cases
    testCases := make([]testaddons.AddonTestCase, 50)
    for i := 0; i < 50; i++ {
        testCases[i] = testaddons.AddonTestCase{
            Name:   fmt.Sprintf("HighVolumeTest%d", i+1),
            Prefix: fmt.Sprintf("hv-test%d", i+1),
        }
    }

    baseOptions := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
        Testing:       t,
        Prefix:        "high-volume-test",
        ResourceGroup: "my-resource-group",
    })

    matrix := testaddons.AddonTestMatrix{
        TestCases:        testCases,
        BaseOptions:      baseOptions,
        StaggerDelay:     testaddons.StaggerDelay(15 * time.Second),        // 15s between batches
        StaggerBatchSize: testaddons.StaggerBatchSize(20),                  // 20 tests per batch
        WithinBatchDelay: testaddons.WithinBatchDelay(1 * time.Second),     // 1s within batches
        AddonConfigFunc: func(options *testaddons.TestAddonOptions, testCase testaddons.AddonTestCase) cloudinfo.AddonConfig {
            return cloudinfo.NewAddonConfigTerraform(
                options.Prefix,
                "high-volume-addon",
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

**This high-volume example**:
- **Batch 1 (Tests 1-20)**: Last test starts after 19s
- **Batch 2 (Tests 21-40)**: First test starts after 15s (batch delay), last test starts after 34s
- **Batch 3 (Tests 41-50)**: First test starts after 30s (batch delay), last test starts after 39s
- **Total time to start all 50 tests**: ~39s (vs 8+ minutes with linear staggering!)

All tests run in parallel once started, with excellent rate limiting protection and minimal delays.
