# Testing Integration Guide

This guide demonstrates how to integrate the logging framework with various testing scenarios and frameworks commonly used in the IBM Cloud ecosystem.

## Go Testing Integration

### Basic Test Integration

```golang
package test

import (
    "testing"
    "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/common"
)

func TestBasicIntegration(t *testing.T) {
    // Always use test name for logger identification
    logger := common.NewBufferedTestLogger(t.Name(), true)

    logger.ShortInfo("Starting integration test")

    if err := performOperation(); err != nil {
        logger.CriticalError(fmt.Sprintf("Operation failed: %v", err))
        return
    }

    logger.ShortInfo("Test completed successfully")
}
```

### Parallel Test Integration

```golang
func TestParallelIntegration(t *testing.T) {
    t.Parallel() // Critical for parallel execution

    // BufferedTestLogger with quiet mode is essential for parallel tests
    logger := common.NewBufferedTestLogger(t.Name(), true)

    logger.ShortInfo("This won't interfere with other parallel tests")

    // Test logic here

    if failed {
        logger.CriticalError("Test failed")
        return
    }
}
```

### Subtests Integration

```golang
func TestWithSubtests(t *testing.T) {
    scenarios := []struct {
        name string
        config TestConfig
    }{
        {"scenario1", config1},
        {"scenario2", config2},
    }

    for _, scenario := range scenarios {
        t.Run(scenario.name, func(t *testing.T) {
            t.Parallel()

            // Use subtest name for logger
            logger := common.CreateAddonLogger(t.Name(), true)

            logger.ShortInfo("Running scenario: %s", scenario.name)

            if err := runScenario(scenario.config); err != nil {
                logger.CriticalError(fmt.Sprintf("Scenario %s failed: %v", scenario.name, err))
                return
            }
        })
    }
}
```

## Testify Integration

### With Assertions

```golang
import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/common"
)

func TestWithTestifyAssertions(t *testing.T) {
    t.Parallel()

    logger := common.NewBufferedTestLogger(t.Name(), true)

    result := performOperation()
    logger.ShortDebug("Operation result: %+v", result)

    // Use require for critical assertions
    if !require.NotNil(t, result, "Result should not be nil") {
        logger.CriticalError("Assertion failed: Result should not be nil")
        return
    }

    // Use assert for non-critical checks
    if !assert.Equal(t, "expected", result.Value, "Values should match") {
        logger.ErrorWithContext("Assertion failed: Values should match")
        // Continue testing with assert
    }

    logger.ShortInfo("Testify assertions completed")
}
```

### With Test Suites

```golang
import (
    "testing"
    "github.com/stretchr/testify/suite"
    "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/common"
)

type IntegrationTestSuite struct {
    suite.Suite
    logger common.Logger
}

func (suite *IntegrationTestSuite) SetupSuite() {
    suite.logger = common.CreateAddonLogger("IntegrationTestSuite", true)
    suite.logger.ShortInfo("Setting up test suite")
}

func (suite *IntegrationTestSuite) SetupTest() {
    testName := suite.T().Name()
    suite.logger.SetPrefix(testName)
    suite.logger.ShortInfo("Starting test: %s", testName)
}

func (suite *IntegrationTestSuite) TestOperation() {
    if err := performOperation(); err != nil {
        suite.logger.CriticalError(fmt.Sprintf("Operation should succeed: %v", err))
        return
    }
}

func (suite *IntegrationTestSuite) TearDownTest() {
    suite.logger.ShortInfo("Test completed: %s", suite.T().Name())
    suite.logger.SetPrefix("") // Reset prefix
}

func TestIntegrationTestSuite(t *testing.T) {
    suite.Run(t, new(IntegrationTestSuite))
}
```

## Terratest Integration

### Basic Terraform Testing

```golang
import (
    "testing"
    "github.com/gruntwork-io/terratest/modules/terraform"
    "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/common"
)

func TestTerraformModule(t *testing.T) {
    t.Parallel()

    // Use helper logger for terraform operations
    logger := common.CreateHelperLogger(t.Name(), true)

    terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
        TerraformDir: "../terraform-modules/example",
        Vars: map[string]interface{}{
            "region": "us-south",
            "name":   "test-" + t.Name(),
        },
    })

    defer func() {
        logger.ProgressStage("Cleaning up Terraform resources")
        terraform.Destroy(t, terraformOptions)
        logger.ProgressSuccess("Cleanup completed")
    }()

    // These will automatically show progress with helper patterns
    logger.ShortInfo("Running Terraform Init")
    terraform.Init(t, terraformOptions)

    logger.ShortInfo("Running Terraform Plan")
    terraform.Plan(t, terraformOptions)

    logger.ShortInfo("Running Terraform Apply")
    terraform.Apply(t, terraformOptions)

    // Validate outputs
    resourceId := terraform.Output(t, terraformOptions, "resource_id")
    logger.ShortDebug("Resource ID: %s", resourceId)

    if resourceId == "" {
        logger.CriticalError("Expected non-empty resource_id output")
        return
    }

    logger.ShortInfo("Terraform Apply Complete")
}
```

### With Retry Logic

```golang
import (
    "github.com/gruntwork-io/terratest/modules/retry"
    "time"
)

func TestTerraformWithRetry(t *testing.T) {
    t.Parallel()

    logger := common.CreateHelperLogger(t.Name(), true)

    terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
        TerraformDir: "../terraform",
    })

    defer terraform.Destroy(t, terraformOptions)

    terraform.InitAndApply(t, terraformOptions)

    // Retry validation with logging
    retry.DoWithRetry(t, "Validate resource availability", 10, 30*time.Second, func() (string, error) {
        logger.ShortInfo("Checking resource availability")

        resourceUrl := terraform.Output(t, terraformOptions, "resource_url")
        if err := validateResource(resourceUrl); err != nil {
            logger.ShortWarn("Resource not ready yet, retrying...")
            return "", err
        }

        logger.ShortInfo("Resource validation successful")
        return "Success", nil
    })
}
```

## IBM Cloud Terratest Wrapper Integration

### With testaddons Package

```golang
import (
    "testing"
    "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/cloudinfo"
    "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testaddons"
)

func setupAddonOptions(t *testing.T) *testaddons.TestAddonOptions {
    options := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
        Testing:       t,
        Prefix:        "logging-test",
        ResourceGroup: "test-rg",
    })

    // The framework automatically provides an appropriate logger
    // But you can override it if needed
    options.Logger = common.CreateAddonLogger(t.Name(), true)

    return options
}

func TestAddonIntegration(t *testing.T) {
    t.Parallel()

    options := setupAddonOptions(t)

    options.AddonConfig = cloudinfo.NewAddonConfigTerraform(
        options.Prefix,
        "test-addon",
        "basic",
        map[string]interface{}{
            "region": "us-south",
        },
    )

    // The framework will automatically use smart logging
    output, err := options.RunAddonTest()
    if err != nil {
        t.Fatalf("Addon test failed: %v", err)
    }

    // Logger is available for additional operations
    options.Logger.ShortInfo("Addon test completed, validating outputs")
}
```

### With testhelper Package

```golang
import (
    "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testhelper"
)

func TestHelperIntegration(t *testing.T) {
    t.Parallel()

    options := testhelper.TestOptionsDefault(&testhelper.TestOptions{
        Testing:      t,
        TerraformDir: "examples/basic",
        Prefix:       "helper-test",
    })

    // Override default logger with smart logging
    options.Logger = common.CreateHelperLogger(t.Name(), true)

    output, err := options.RunTestConsistency()
    if err != nil {
        options.Logger.CriticalError(fmt.Sprintf("Consistency test failed: %v", err))
        return
    }

    options.Logger.ShortInfo("Consistency test passed")
}
```

## CI/CD Integration

### GitHub Actions Integration

```golang
// Detect CI environment and configure logger appropriately
func createCILogger(testName string) common.Logger {
    // GitHub Actions sets GITHUB_ACTIONS=true
    isCI := os.Getenv("GITHUB_ACTIONS") == "true"

    if isCI {
        // CI: quiet mode with progress tracking
        return common.CreateAddonLogger(testName, true)
    } else {
        // Local development: verbose mode
        return common.CreateAddonLogger(testName, false)
    }
}

func TestForCI(t *testing.T) {
    t.Parallel()

    logger := createCILogger(t.Name())

    // CI will only show progress stages and failures
    logger.ShortInfo("Starting CI test")
    logger.ShortDebug("Debug info only shown on failure")

    if testFailed {
        logger.CriticalError("Test failed") // Critical in CI to see failure details
        return
    }
}
```

### Jenkins Integration

```golang
func createJenkinsLogger(testName string) common.Logger {
    // Jenkins sets JENKINS_URL
    isJenkins := os.Getenv("JENKINS_URL") != ""
    buildNumber := os.Getenv("BUILD_NUMBER")

    logger := common.CreateAddonLogger(testName, isJenkins)

    if buildNumber != "" {
        logger.SetPrefix("build-" + buildNumber)
    }

    return logger
}
```

## Performance Considerations

### For Large Test Suites

```golang
func TestLargeTestSuite(t *testing.T) {
    // Use batch mode for repetitive operations
    logger := common.CreateAddonLogger(t.Name(), true)
    smartLogger := logger.(*common.SmartLogger)
    smartLogger.EnableBatchMode()

    defer smartLogger.DisableBatchMode()

    // Process many similar items efficiently
    for i := 0; i < 1000; i++ {
        logger.ShortInfo("Getting offering details") // Only shows once
        processItem(i)
        logger.ShortInfo("Request completed") // Shows progress
    }
}
```

### Memory Management

```golang
func TestMemoryEfficient(t *testing.T) {
    t.Parallel()

    logger := common.NewBufferedTestLogger(t.Name(), true)

    // Periodically clear buffer for long-running tests
    for i := 0; i < 10000; i++ {
        logger.ShortDebug("Processing item %d", i)

        if i%1000 == 0 {
            // Clear buffer periodically to prevent memory growth
            logger.ClearBuffer()
            logger.ProgressInfo("Processed %d items", i)
        }
    }

    if testFailed {
        logger.CriticalError("Test failed")
        return
    }
}
```

## Best Practices for Integration

### Universal Patterns

1. **Always use test names** for logger identification
2. **Enable parallel execution** with `t.Parallel()`
3. **Use quiet mode** for parallel tests
4. **Use enhanced error methods** for test failures
5. **Choose appropriate logger type** for test complexity

### Framework-Specific Patterns

#### For Go Testing
- Use `NewBufferedTestLogger` with quiet mode for parallel tests
- Use `NewTestLogger` for debugging and development
- Always pass `t.Name()` as test name

#### For Testify
- Use `ErrorWithContext()` for assertion failures, `CriticalError()` for critical failures
- Use `require` for critical assertions that should stop execution
- Configure logger in `SetupTest()` for test suites

#### For Terratest
- Use `CreateHelperLogger` for automatic Terraform phase detection
- Set up cleanup in `defer` statements with progress logging
- Log Terraform outputs for debugging

#### For IBM Cloud Wrapper
- Use predefined factory functions (`CreateAddonLogger`, etc.)
- Let the framework handle logger configuration when possible
- Override only when you need specific logging behavior

### Environment-Based Configuration

```golang
func createEnvironmentLogger(testName string) common.Logger {
    switch {
    case os.Getenv("CI") == "true":
        // CI environment: quiet with progress
        return common.CreateAddonLogger(testName, true)
    case os.Getenv("DEBUG") == "true":
        // Debug mode: verbose with timestamps
        logger := common.CreateAddonLogger(testName, false)
        logger.EnableDateTime(true)
        return logger
    case os.Getenv("PARALLEL_TESTS") == "true":
        // Parallel execution: quiet buffered
        return common.NewBufferedTestLogger(testName, true)
    default:
        // Local development: auto-detect based on parallel execution
        return common.CreateAddonLogger(testName, true)
    }
}
```

This integration guide provides the foundation for using the logging framework effectively across different testing scenarios. See [Examples](examples.md) for more specific code samples and [Troubleshooting](troubleshooting.md) for common issues and solutions.
