# Matrix Testing

Matrix testing allows you to run multiple test scenarios in parallel with shared resources for efficiency. The `RunAddonTestMatrix()` method provides a powerful way to test different configurations, input combinations, and dependency scenarios while automatically managing resource sharing and cleanup.

## Overview

Matrix testing solves the problem of creating multiple similar tests by providing:

- **Parallel Execution**: All test cases run in parallel for maximum efficiency
- **Resource Sharing**: Automatically shares catalogs and offerings across test cases
- **Flexible Configuration**: Each test case can have custom inputs, dependencies, and behavior
- **Automatic Cleanup**: Shared resources are cleaned up after all tests complete
- **Progress Feedback**: Clear visual indicators of test progress and results

## Key Features

- **Shared Catalog Management**: Creates one catalog for all test cases instead of N catalogs
- **Configurable Test Cases**: Each test case can have unique inputs, prefixes, and settings
- **Batched Staggered Execution**: Advanced batching system with 87% faster execution for large test suites
- **Quiet Mode Default**: Automatically defaults to enabled for cleaner output (can be overridden)
- **Automatic Resource Cleanup**: Guaranteed cleanup of shared resources

## Basic Usage

### Simple Matrix Test

```golang
package test

import (
    "testing"
    "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/cloudinfo"
    "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testaddons"
    "github.com/IBM/go-sdk-core/v5/core"
)

func TestAddonMatrix(t *testing.T) {
    t.Parallel()

    // Define test cases
    testCases := []testaddons.AddonTestCase{
        {
            Name:   "BasicConfiguration",
            Prefix: "basic",
            Inputs: map[string]interface{}{
                "region": "us-south",
                "plan":   "standard",
            },
        },
        {
            Name:   "CustomConfiguration",
            Prefix: "custom",
            Inputs: map[string]interface{}{
                "region": "eu-gb",
                "plan":   "premium",
            },
        },
        {
            Name:                         "ValidationOnly",
            Prefix:                       "validation",
            SkipInfrastructureDeployment: true,
        },
    }

    // Base options that apply to all test cases
    baseOptions := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
        Testing:       t,
        Prefix:        "matrix-test",
        ResourceGroup: "my-resource-group",
        QuietMode:     true, // Enable quiet mode for clean output
    })

    // Create matrix configuration
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
                    "region": "us-south", // Default, can be overridden by test case inputs
                },
            )
        },
    }

    // Execute the matrix test
    baseOptions.RunAddonTestMatrix(matrix)
}
```

### Advanced Matrix Test with Custom Setup

```golang
func TestAdvancedAddonMatrix(t *testing.T) {
    t.Parallel()

    testCases := []testaddons.AddonTestCase{
        {
            Name:   "WithKMS",
            Prefix: "with-kms",
            Dependencies: []cloudinfo.AddonConfig{
                {
                    OfferingName:   "deploy-arch-ibm-kms",
                    OfferingFlavor: "fully-configurable",
                    Enabled:        core.BoolPtr(true),
                },
            },
            Inputs: map[string]interface{}{
                "enable_encryption": true,
            },
        },
        {
            Name:   "WithoutKMS",
            Prefix: "without-kms",
            Dependencies: []cloudinfo.AddonConfig{
                {
                    OfferingName:   "deploy-arch-ibm-kms",
                    OfferingFlavor: "fully-configurable",
                    Enabled:        core.BoolPtr(false),
                },
            },
            Inputs: map[string]interface{}{
                "enable_encryption": false,
            },
        },
    }

    baseOptions := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
        Testing:       t,
        Prefix:        "advanced-matrix",
        ResourceGroup: "my-resource-group",
        QuietMode:     true,
    })

    matrix := testaddons.AddonTestMatrix{
        TestCases:   testCases,
        BaseOptions: baseOptions,
        BaseSetupFunc: func(baseOptions *testaddons.TestAddonOptions, testCase testaddons.AddonTestCase) *testaddons.TestAddonOptions {
            // Customize options for each test case
            testOptions := baseOptions.copy()
            testOptions.TestCaseName = testCase.Name

            // Add test-case-specific customizations
            if testCase.Name == "WithKMS" {
                testOptions.DeployTimeoutMinutes = 180 // Longer timeout for KMS
            }

            return testOptions
        },
        AddonConfigFunc: func(options *testaddons.TestAddonOptions, testCase testaddons.AddonTestCase) cloudinfo.AddonConfig {
            config := cloudinfo.NewAddonConfigTerraform(
                options.Prefix,
                "my-secure-addon",
                "fully-configurable",
                map[string]interface{}{
                    "prefix": options.Prefix,
                    "region": "us-south",
                },
            )

            // Set dependencies from test case
            if testCase.Dependencies != nil {
                config.Dependencies = testCase.Dependencies
            }

            return config
        },
    }

    baseOptions.RunAddonTestMatrix(matrix)
}
```

## Configuration Options

### Test Case Configuration

```golang
type AddonTestCase struct {
    Name                         string                    // Test case name (required)
    Prefix                       string                    // Unique prefix for resources
    Inputs                       map[string]interface{}    // Custom inputs for this test case
    Dependencies                 []cloudinfo.AddonConfig   // Dependency configuration
    SkipTearDown                 bool                      // Skip cleanup for this test case
    SkipInfrastructureDeployment bool                      // Skip actual deployment
}
```

### Matrix Configuration

```golang
type AddonTestMatrix struct {
    TestCases        []AddonTestCase                       // Test cases to run (required)
    BaseOptions      *TestAddonOptions                     // Common options (required)
    BaseSetupFunc    func(*TestAddonOptions, AddonTestCase) *TestAddonOptions  // Optional customization
    AddonConfigFunc  func(*TestAddonOptions, AddonTestCase) cloudinfo.AddonConfig // Config generator (required)
    StaggerDelay     *time.Duration                        // Delay between batches (default: 10s)
    StaggerBatchSize *int                                  // Tests per batch (default: 8)
    WithinBatchDelay *time.Duration                        // Delay within batches (default: 2s)
}
```

### Stagger Configuration

The framework supports advanced batched staggering for efficient rate limiting protection:

```golang
// Default batched staggering (8 tests per batch, 10s between batches, 2s within batches)
matrix := testaddons.AddonTestMatrix{
    TestCases:   testCases,
    BaseOptions: baseOptions,
    // Batched staggering is enabled by default
    // ... other configuration
}

// Custom batched staggering for high-volume tests
matrix := testaddons.AddonTestMatrix{
    TestCases:        testCases,
    BaseOptions:      baseOptions,
    StaggerDelay:     testaddons.StaggerDelay(15 * time.Second),        // Delay between batches
    StaggerBatchSize: testaddons.StaggerBatchSize(20),                  // Tests per batch
    WithinBatchDelay: testaddons.WithinBatchDelay(1 * time.Second),     // Delay within each batch
    // ... other configuration
}

// Legacy linear staggering (not recommended for >20 tests)
matrix := testaddons.AddonTestMatrix{
    TestCases:        testCases,
    BaseOptions:      baseOptions,
    StaggerDelay:     testaddons.StaggerDelay(10 * time.Second),        // Linear delay
    StaggerBatchSize: testaddons.StaggerBatchSize(0),                   // Disable batching
    // ... other configuration
}
```

**Benefits of Batched Staggering:**
- **87% reduction** in setup delays for large test suites (50+ tests)
- **Scalable approach** that works efficiently with any number of tests
- **Maintains rate limiting protection** through batch boundaries

## Quiet Mode Features

### Matrix-Specific Progress Indicators

With quiet mode enabled by default, matrix tests show clean progress for each test case:

```
🔄 Starting test: BasicConfiguration
🔄 Setting up test Catalog and Project
🔄 Deploying Configurations to Project
🔄 Validating dependencies
✅ Infrastructure deployment completed
🔄 Cleaning up resources
  ✓ Passed: BasicConfiguration

🔄 Starting test: CustomConfiguration
🔄 Setting up test Catalog and Project
🔄 Deploying Configurations to Project
🔄 Validating dependencies
✅ Infrastructure deployment completed
🔄 Cleaning up resources
  ✓ Passed: CustomConfiguration
```

### Stagger Delay Behavior

- **Quiet Mode**: Stagger delays are applied silently without log messages
- **Verbose Mode**: Shows stagger delay messages like `[TestName - STAGGER] Delaying test start by 15s to prevent rate limiting (test 2/3)`

### Shared Resource Creation

Matrix tests automatically create shared catalogs and offerings:

```
[TestMatrix] Creating shared catalog for matrix: matrix-test-my-addon-catalog-a1b2
[TestMatrix] Created shared catalog: matrix-test-my-addon-catalog-a1b2 with ID xyz123
[TestMatrix] Importing shared offering: standard as version: v0.0.1-dev-matrix-test-a1b2
[TestMatrix] Imported shared offering: My Addon with ID abc456
```

## Benefits

### Resource Efficiency

- **Single Catalog**: Creates 1 catalog for N test cases instead of N catalogs
- **Shared Offerings**: Imports the offering once and reuses across all test cases
- **Reduced API Calls**: Fewer catalog management operations
- **Faster Execution**: Less time spent on resource creation/deletion

### Time Savings

- **Parallel Execution**: All test cases run simultaneously
- **Shared Setup**: Catalog and offering creation happens once
- **Automatic Cleanup**: No manual resource management required

### Clean Output

With quiet mode enabled:
- Essential progress indicators only
- Clear test results
- No verbose API logs
- Easy identification of failed tests

## Best Practices

### 1. Use Descriptive Test Case Names

```golang
testCases := []testaddons.AddonTestCase{
    {
        Name: "StandardPlanUSRegion",      // Clear and descriptive
        // ...
    },
    {
        Name: "PremiumPlanEuropeRegion",   // Clear and descriptive
        // ...
    },
}
```

### 2. Enable Quiet Mode for Matrix Tests

```golang
baseOptions := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
    Testing:   t,
    Prefix:    "matrix-test",
    QuietMode: true, // Recommended for matrix tests
})
```

### 3. Use Appropriate Stagger Delays

```golang
matrix := testaddons.AddonTestMatrix{
    TestCases:    testCases,
    BaseOptions:  baseOptions,
    StaggerDelay: core.DurationPtr(10 * time.Second), // Prevent rate limiting
}
```

### 4. Organize Test Cases Logically

```golang
// Group related test cases together
testCases := []testaddons.AddonTestCase{
    // Basic functionality tests
    {Name: "BasicStandardPlan", /* ... */},
    {Name: "BasicPremiumPlan", /* ... */},

    // Dependency tests
    {Name: "WithKMS", /* ... */},
    {Name: "WithoutKMS", /* ... */},

    // Validation-only tests
    {Name: "ValidationOnly", SkipInfrastructureDeployment: true},
}
```

### 5. Handle Test Case Customization Properly

```golang
BaseSetupFunc: func(baseOptions *testaddons.TestAddonOptions, testCase testaddons.AddonTestCase) *testaddons.TestAddonOptions {
    testOptions := baseOptions.copy() // Always copy base options
    testOptions.TestCaseName = testCase.Name

    // Apply test-case-specific customizations
    switch testCase.Name {
    case "LongRunningTest":
        testOptions.DeployTimeoutMinutes = 240
    case "ValidationOnly":
        testOptions.SkipInfrastructureDeployment = true
    }

    return testOptions
}
```

## Comparing Matrix Testing with Other Approaches

### Matrix Testing vs Individual Tests

**Matrix Testing:**
- Single shared catalog for all test cases
- Parallel execution with automatic coordination
- Efficient resource usage
- Ideal for testing multiple configurations of the same addon

**Individual Tests:**
- Separate catalogs for each test
- Complete isolation between tests
- More resource overhead
- Better for completely different addon tests

### Matrix Testing vs Permutation Testing

**Matrix Testing:**
- Manual definition of specific test scenarios
- Full control over each test case configuration
- Can mix deployment and validation-only tests
- Best for targeted scenario testing

**Permutation Testing:**
- Automatic generation of all dependency combinations
- Focused on dependency validation
- All tests are validation-only
- Best for comprehensive dependency coverage

## Integration with Other Testing Patterns

```golang
package test

func TestComprehensiveAddonTesting(t *testing.T) {
    t.Parallel()

    // Run basic deployment test first
    t.Run("BasicDeployment", func(t *testing.T) {
        t.Parallel()
        options := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
            Testing: t,
            Prefix:  "basic-deploy",
        })
        err := options.RunAddonTest()
        assert.NoError(t, err)
    })

    // Run matrix tests for multiple scenarios
    t.Run("MatrixTests", func(t *testing.T) {
        t.Parallel()
        // Matrix test implementation here
    })

    // Run permutation tests for dependency validation
    t.Run("DependencyPermutations", func(t *testing.T) {
        t.Parallel()
        options := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
            Testing: t,
            Prefix:  "dep-perm",
            AddonConfig: cloudinfo.AddonConfig{
                OfferingName:   "my-addon",
                OfferingFlavor: "standard",
            },
        })
        err := options.RunAddonPermutationTest()
        assert.NoError(t, err)
    })
}
```

## Summary

Matrix testing provides an efficient way to test multiple addon configurations in parallel with shared resources. Key advantages include:

- **Efficient Resource Usage**: Single catalog shared across all test cases
- **Parallel Execution**: All tests run simultaneously for speed
- **Flexible Configuration**: Each test case can be customized as needed
- **Clean Output**: Quiet mode (enabled by default) provides clear progress indicators
- **Automatic Cleanup**: No manual resource management required

Use matrix testing when you need to test multiple specific scenarios for the same addon, and combine it with permutation testing for comprehensive dependency validation.
