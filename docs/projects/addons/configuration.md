# Configuration Guide

This guide covers all configuration options available in the addon testing framework, from basic setup to advanced customization.

## Basic Configuration

### Required Options

```golang
options := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
    Testing:       t,                // Required: testing.T object
    Prefix:        "my-test",        // Required: unique prefix for resources
    ResourceGroup: "my-project-rg",  // Required: resource group for project
})
```

### Optional Basic Settings

```golang
options := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
    Testing:       t,
    Prefix:        "my-test",
    ResourceGroup: "my-project-rg",
    TestCaseName:  "CustomScenario",  // Optional: custom name for log identification
})
```

**TestCaseName**: Sets a custom identifier for log messages. When specified, log output will show:

```text
[TestName - ADDON - CustomScenario] Checking for local changes in the repository
```

This is particularly useful for:

- **Debugging**: Easily identify which test scenario is running
- **Matrix Tests**: Automatically set by the framework using the test case name
- **Custom Tests**: Manually set for clear log identification

### Essential Setup Function

It's recommended to create a setup function for consistency across tests:

```golang
func setupAddonOptions(t *testing.T, prefix string) *testaddons.TestAddonOptions {
    options := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
        Testing:       t,
        Prefix:        prefix,
        ResourceGroup: resourceGroup, // Use a package-level variable
    })
    return options
}
```

## Project Configuration Options

### Basic Project Settings

```golang
options.ProjectName = "my-addon-test"                    // Default: "addon{prefix}"
options.ProjectDescription = "Testing my addon"          // Default: "Testing {prefix}-addon"
options.ProjectLocation = "us-south"                    // Default: "us-south"
```

### Project Behavior Settings

```golang
options.ProjectDestroyOnDelete = core.BoolPtr(true)     // Default: true
options.ProjectMonitoringEnabled = core.BoolPtr(true)   // Default: true
options.ProjectAutoDeploy = core.BoolPtr(true)          // Default: true
```

### Project Environments

```golang
options.ProjectEnvironments = []project.EnvironmentPrototype{
    {
        Definition: &project.EnvironmentDefinitionRequiredProperties{
            Name:        core.StringPtr("development"),
            Description: core.StringPtr("Development environment"),
        },
    },
}
```

## Catalog Configuration

### Using Temporary Catalog (Default)

```golang
// Framework creates and manages temporary catalog automatically
options.CatalogName = "my-test-catalog"  // Optional: customize catalog name
// Default: "dev-addon-test-{prefix}"
```

### Using Existing Catalog

```golang
options.CatalogUseExisting = true
options.CatalogName = "existing-catalog-name"  // Required when using existing
```

### Catalog Sharing Control

The `SharedCatalog` option controls catalog and offering sharing behavior:

```golang
options := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
    Testing:       t,
    Prefix:        "test",
    ResourceGroup: "my-rg",
    SharedCatalog: core.BoolPtr(false),  // Default: false for individual tests
})
```

**SharedCatalog Settings:**

- `false` (default): Each test creates its own catalog and offering for complete isolation and automatic cleanup
- `true`: Catalogs and offerings are shared across tests using the same `TestOptions` object (requires manual cleanup)

**Sharing Behavior:**

```golang
// SharedCatalog = false (default) - isolated tests with automatic cleanup
isolatedOptions := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
    Testing:       t,
    Prefix:        "isolated-test",
    ResourceGroup: "my-rg",
    SharedCatalog: core.BoolPtr(false),  // Can be omitted as it's the default
})

// Each test creates and cleans up its own catalog + offering
err1 := isolatedOptions.RunAddonTest()  // Creates & deletes catalog A
err2 := isolatedOptions.RunAddonTest()  // Creates & deletes catalog B

// SharedCatalog = true - efficient sharing (requires manual cleanup)
baseOptions := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
    Testing:       t,
    Prefix:        "shared-test",
    ResourceGroup: "my-rg",
    SharedCatalog: core.BoolPtr(true),
})

// First test creates catalog + offering, second test reuses them
err3 := baseOptions.RunAddonTest()  // Creates catalog
err4 := baseOptions.RunAddonTest()  // Reuses catalog (manual cleanup needed)
```

### Automatic Catalog Sharing (Matrix Tests)

When using matrix testing with `RunAddonTestMatrix()`, catalogs and offerings are automatically shared across all test cases for improved efficiency:

```golang
// Matrix tests automatically share catalogs - no additional configuration needed
baseOptions := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
    Testing:       t,
    Prefix:        "matrix-test",
    ResourceGroup: "my-resource-group",
})

baseOptions.RunAddonTestMatrix(matrix)  // Catalog automatically shared across all test cases
```

**Benefits:**

- **Resource Efficiency**: Creates only 1 catalog for all test cases instead of N catalogs
- **Time Savings**: Reduced catalog creation time and API calls
- **Automatic Cleanup**: Shared resources cleaned up after all matrix tests complete

**Individual vs Matrix Tests:**

- **Individual Tests**: Respect the `SharedCatalog` setting (default: false - not shared)
- **Matrix Tests**: Always share catalogs regardless of `SharedCatalog` setting

### Catalog Cleanup Behavior

Understanding when catalogs are cleaned up is important for resource management:

**Matrix Tests (RunAddonTestMatrix):**

- Catalogs are automatically cleaned up after all test cases complete
- Uses Go's `t.Cleanup()` mechanism to ensure cleanup happens

**Individual Tests with SharedCatalog=false (default):**

- Each test creates and deletes its own catalog
- Automatic cleanup with guaranteed isolation
- Use for most individual tests and when isolation is important

**Individual Tests with SharedCatalog=true:**

- Catalogs are shared and persist after test completion
- Efficient for development workflows and sequential test runs
- **Manual cleanup required** - catalogs will persist until manually deleted

**Best Practices:**

```golang
// For most tests - automatic cleanup with isolation (recommended)
options.SharedCatalog = core.BoolPtr(false)  // Default

// For development and sequential tests - efficient sharing
options.SharedCatalog = core.BoolPtr(true)   // Manual cleanup required

// For matrix tests - automatic sharing and cleanup (recommended)
baseOptions.RunAddonTestMatrix(matrix)
```

**When to use each approach:**

- **SharedCatalog=false**: Most individual tests, CI pipelines, when automatic cleanup is needed
- **SharedCatalog=true**: Development workflows, sequential tests with same prefix
- **Matrix tests**: Multiple test cases with variations (automatic sharing + cleanup)

### Manual Cleanup for Shared Catalogs

When using `SharedCatalog=true` with individual tests, you can manually clean up shared resources using `CleanupSharedResources()`:

```golang
func TestMultipleAddonsWithSharedCatalog(t *testing.T) {
    baseOptions := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
        Testing:       t,
        Prefix:        "shared-test",
        ResourceGroup: "my-resource-group",
        SharedCatalog: core.BoolPtr(true), // Enable sharing
    })

    // Ensure cleanup happens at the end
    defer baseOptions.CleanupSharedResources()

    // Run multiple tests that share the catalog
    t.Run("TestScenario1", func(t *testing.T) {
        options1 := baseOptions
        options1.AddonConfig = cloudinfo.NewAddonConfigTerraform(/* config */)
        err := options1.RunAddonTest()
        require.NoError(t, err)
    })

    t.Run("TestScenario2", func(t *testing.T) {
        options2 := baseOptions
        options2.AddonConfig = cloudinfo.NewAddonConfigTerraform(/* different config */)
        err := options2.RunAddonTest()
        require.NoError(t, err)
    })

    // CleanupSharedResources() called automatically via defer
}
```

**Benefits of manual cleanup:**

- Guaranteed resource cleanup regardless of test failures
- Works with any number of individual test variations
- Simple defer pattern ensures cleanup runs even if tests panic

#### Alternative: Cleanup in TestMain

For package-level cleanup across multiple test functions:

```golang
func TestMain(m *testing.M) {
    // Setup shared options if needed
    sharedOptions := setupSharedOptions()

    // Run tests
    code := m.Run()

    // Cleanup shared resources
    if sharedOptions != nil {
        sharedOptions.CleanupSharedResources()
    }

    os.Exit(code)
}
```

## Addon Configuration

### Terraform Addon (Primary Use Case)

```golang
options.AddonConfig = cloudinfo.NewAddonConfigTerraform(
    options.Prefix,        // prefix for unique resource naming
    "offering-name",       // offering name in catalog
    "flavor-name",         // offering flavor
    map[string]interface{}{ // inputs
        "prefix": options.Prefix,
        "region": "us-south",
        "custom_setting": "value",
    },
)
```

### Stack Addon (Advanced/Rare Use Case)

```golang
options.AddonConfig = cloudinfo.NewAddonConfigStack(
    options.Prefix,        // prefix for unique resource naming
    "stack-name",          // stack offering name
    "flavor-name",         // stack flavor
    map[string]interface{}{ // inputs
        "prefix": options.Prefix,
        "region": "us-south",
    },
)
```

### Manual Addon Configuration

```golang
options.AddonConfig = cloudinfo.AddonConfig{
    Prefix:         options.Prefix,
    OfferingName:   "my-addon",
    OfferingFlavor: "standard",
    InstallKind:    "terraform", // or "stack"
    Inputs: map[string]interface{}{
        "prefix": options.Prefix,
        "region": "us-south",
    },
}
```

## Dependency Management

### Automatic Dependencies (Default)

```golang
// Framework automatically discovers and configures dependencies
// No additional configuration needed
```

### Manual Dependency Override

```golang
options.AddonConfig.Dependencies = []cloudinfo.AddonConfig{
    {
        OfferingName:   "dependency-addon",
        OfferingFlavor: "standard",
        Enabled:        core.BoolPtr(true),  // explicitly enable
        Inputs: map[string]interface{}{
            "prefix": options.Prefix,
        },
    },
    {
        OfferingName:   "optional-addon",
        OfferingFlavor: "basic",
        Enabled:        core.BoolPtr(false), // explicitly disable
    },
}
```

### Dependency Configuration Options

```golang
dependency := cloudinfo.AddonConfig{
    OfferingName:     "dependency-name",        // Required
    OfferingFlavor:   "flavor-name",           // Required
    Enabled:          core.BoolPtr(true),      // Optional: explicit enable/disable
    OnByDefault:      core.BoolPtr(true),      // Optional: default behavior
    ExistingConfigID: "existing-config-id",    // Optional: use existing config
    Inputs: map[string]interface{}{            // Optional: dependency inputs
        "setting": "value",
    },
}
```

## Timeout Configuration

### Deployment Timeouts

```golang
options.DeployTimeoutMinutes = 120  // 2 hours instead of default 6 hours
```

### Background Process Considerations

- Default timeout: 6 hours (360 minutes)
- Applies to both deploy and undeploy operations
- Consider resource complexity when setting timeouts
- Parallel tests may need longer timeouts due to resource contention

## Logging and Output Configuration

### Quiet Mode

Control log verbosity during test execution to reduce noise while maintaining essential feedback:

```golang
options.QuietMode = true  // Enable quiet mode (default: false)
```

**Quiet Mode Features:**

- **Suppresses verbose operational logs**: Eliminates detailed API calls, configuration details, and internal operations
- **Shows essential progress feedback**: Displays high-level stages with visual indicators
- **Maintains test result visibility**: Always shows final test results and error messages
- **Reduces noise during parallel execution**: Particularly useful for matrix and permutation testing

**Progress Feedback in Quiet Mode:**

When quiet mode is enabled, you'll see clean progress indicators instead of verbose logs:

```
🔄 Setting up test Catalog and Project
🔄 Deploying Configurations to Project
🔄 Validating dependencies
✅ Infrastructure deployment completed
🔄 Cleaning up resources
```

**Progress Indicator Types:**

- `🔄` **Stage indicators**: Setup, deployment, validation, cleanup phases
- `✅` **Success confirmations**: Successful completion of major operations
- `ℹ️` **Essential status updates**: Critical information that bypasses quiet mode
- `✓ Passed` / `✗ Failed`: Final test results

### Verbose Mode (Default)

For detailed debugging and development, use verbose mode:

```golang
options.QuietMode = false  // Show all logs (default behavior)
// Or simply omit QuietMode to use default verbose behavior
```

Verbose mode shows:
- All API calls and responses
- Detailed configuration information
- Step-by-step operation logs
- Full dependency validation details
- Complete reference resolution logs

### Verbose Error Output Control

Control whether detailed error information is shown when tests fail, particularly useful with QuietMode:

```golang
options.QuietMode = true          // Enable quiet mode (default: false)
options.VerboseOnFailure = true   // Show detailed logs on failure (default: true)
```

**VerboseOnFailure Behavior:**

When `VerboseOnFailure` is `true` (default):
- **Success**: Only essential progress indicators shown (when QuietMode=true)
- **Failure**: Detailed logs and error information displayed for debugging
- **Always**: Final test results and critical error messages shown

When `VerboseOnFailure` is `false`:
- **Success**: Only essential progress indicators shown (when QuietMode=true)
- **Failure**: Only basic error messages shown, detailed logs suppressed
- **Minimal**: Very minimal output even on failures

**Common Usage Patterns:**

```golang
// Recommended: Quiet during execution, verbose on failure (default)
options.QuietMode = true          // Clean progress indicators
options.VerboseOnFailure = true   // Full debug info on failure

// Development testing: Always show everything
options.QuietMode = false         // Show all logs always

// CI/Pipeline: Minimal output even on failures
options.QuietMode = true          // Clean progress indicators
options.VerboseOnFailure = false  // Minimal output on failure
```

### Automatic Quiet Mode (Permutation and Matrix Testing)

Some test types automatically default to quiet mode for better user experience:

```golang
func TestAddonPermutations(t *testing.T) {
    options := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
        Testing: t,
        Prefix:  "addon-perm",
        AddonConfig: cloudinfo.AddonConfig{
            OfferingName: "my-addon",
            // QuietMode automatically defaults to true for permutation tests
        },
    })

    // Permutation tests default to quiet mode to reduce log noise
    err := options.RunAddonPermutationTest()
    assert.NoError(t, err)
}

func TestAddonMatrix(t *testing.T) {
    // Matrix tests also default to quiet mode
    baseOptions := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
        Testing: t,
        Prefix:  "matrix-test",
        // QuietMode automatically defaults to true for matrix tests
    })

    baseOptions.RunAddonTestMatrix(matrix)
}
```

**Override automatic behavior:**
```golang
// Force verbose mode even for permutation tests
options.QuietMode = false
err := options.RunAddonPermutationTest()

// Force verbose mode for matrix tests
baseOptions.QuietMode = false
baseOptions.RunAddonTestMatrix(matrix)
```

### Matrix Test Quiet Mode

Matrix tests inherit quiet mode settings from `BaseOptions`:

```golang
baseOptions := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
    Testing:   t,
    Prefix:    "matrix-test",
    QuietMode: true, // Applies to all test cases in the matrix
})

matrix := testaddons.AddonTestMatrix{
    TestCases:   testCases,
    BaseOptions: baseOptions, // QuietMode inherited by all test cases
}

baseOptions.RunAddonTestMatrix(matrix)
```

**Matrix-specific quiet mode features:**
- Individual test progress: `🔄 Starting test: test-case-name`
- Stagger delay messages only shown in verbose mode
- Clean test results: `✓ Passed: test-case-name`
- Shared catalog creation progress indicators

## StrictMode Configuration

StrictMode controls how the framework handles validation errors and dependency conflicts during testing.

### Default Behavior (StrictMode=true)

```golang
options := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
    Testing: t,
    Prefix:  "test",
    // StrictMode defaults to true
})

// Or explicitly set
options.StrictMode = core.BoolPtr(true)
```

**In strict mode (default):**

- **Circular Dependencies**: Logged as errors and cause test failure
- **Required Dependencies**: Warns when disabled required dependencies are force-enabled
- **Validation Failures**: All validation issues cause test failure

**Example strict mode output:**
```
ERROR: Circular dependency detected - configs are waiting on each other:
  🔍 CIRCULAR DEPENDENCY DETECTED: deploy-arch-ibm-event-notifications → deploy-arch-ibm-cloud-logs → deploy-arch-ibm-activity-tracker → deploy-arch-ibm-cloud-logs
WARN: Required dependency deploy-arch-ibm-kms was force-enabled despite being disabled
WARN:   Required by: deploy-arch-ibm-event-notifications
WARN:   Use StrictMode=false to suppress this warning
```

### Non-Strict Mode (StrictMode=false)

```golang
options := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
    Testing:    t,
    Prefix:     "test",
    StrictMode: core.BoolPtr(false), // Disable strict mode
})
```

**In non-strict mode:**

- **Circular Dependencies**: Logged as warnings only, test continues
- **Required Dependencies**: Shows informational messages and captures warnings for final report
- **Final Report Integration**: Warnings displayed in final permutation test report showing what would have failed in strict mode

**Example non-strict mode output:**
```
WARN: Circular dependency detected (StrictMode=false - test will continue):
  🔍 CIRCULAR DEPENDENCY DETECTED: deploy-arch-ibm-event-notifications → deploy-arch-ibm-cloud-logs → deploy-arch-ibm-activity-tracker → deploy-arch-ibm-cloud-logs
INFO: Required dependency deploy-arch-ibm-kms was force-enabled (required by deploy-arch-ibm-event-notifications)
```

**Final report includes strict mode warnings:**

```text
================================================================================
🧪 PERMUTATION TEST REPORT - Complete
================================================================================
📊 Summary: 63 total tests | ✅ 63 passed (100.0%) | ❌ 0 failed (0.0%)

✅ PASSED: 63 tests completed successfully

⚠️ STRICT MODE DISABLED - The following would have failed in strict mode:
• Circular Dependencies Detected (2 tests):
  - Test "pp0kwd-dai-e-n-35": Circular dependency: deploy-arch-ibm-activity-tracker → deploy-arch-ibm-cloud-logs → deploy-arch-ibm-activity-tracker
• Required Dependencies Force-Enabled (5 tests):
  - Test "xyz-123": Required dependency deploy-arch-ibm-kms was force-enabled despite being disabled (required by deploy-arch-ibm-event-notifications)

📁 Full test logs available if additional context needed
================================================================================
```

### When to Use Each Mode

**Use StrictMode=true (default) when:**
- Running production validation tests
- Need to catch all potential issues
- Want strict validation for CI/CD pipelines
- Testing critical dependency configurations

**Use StrictMode=false when:**
- Running permutation tests with known circular dependencies
- Testing legacy configurations with complex dependency trees
- Need tests to continue despite validation warnings
- Developing and debugging dependency configurations

### StrictMode with Different Test Types

```golang
// Individual tests - explicit control
options := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
    Testing:    t,
    Prefix:     "individual",
    StrictMode: core.BoolPtr(false), // Override default
})

// Matrix tests - applied to all test cases
baseOptions := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
    Testing:    t,
    Prefix:     "matrix",
    StrictMode: core.BoolPtr(false), // All test cases use non-strict mode
})

// Permutation tests - often use non-strict mode for comprehensive testing
options := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
    Testing:    t,
    Prefix:     "permutation",
    StrictMode: core.BoolPtr(false), // Allow tests to continue with warnings
})
```

## Skip Options

### Infrastructure Operations

```golang
// Skip actual deployment/undeploy but run all validations
options.SkipInfrastructureDeployment = true

// Skip undeploy operation
options.SkipUndeploy = true
```

### Cleanup Operations

```golang
// Skip entire teardown process
options.SkipTestTearDown = true

// Skip project deletion but allow other cleanup
options.SkipProjectDelete = true
```

### Validation Operations

```golang
// Skip local change validation
options.SkipLocalChangeCheck = true

// Skip reference validation
options.SkipRefValidation = true

// Skip dependency validation
options.SkipDependencyValidation = true
```

## Validation Configuration

### Local Change Check

```golang
// Configure files/patterns to ignore during local change check
options.LocalChangesIgnorePattern = []string{
    ".*\\.md$",        // ignore markdown files
    "^docs/.*",        // ignore docs directory
    "^temp/.*",        // ignore temporary files
    ".*\\.log$",       // ignore log files
}
```

**Default ignore patterns:**

- `^common-dev-assets$` - git submodule pointer changes for common-dev-assets
- `^common-dev-assets/.*` - common development assets
- `^tests/.*` - tests directory
- `.*\\.json$` - JSON files (except `ibm_catalog.json`)
- `.*\\.out$` - output files

### Validation Error Output

```golang
// Enhanced dependency tree visualization with validation status
options.EnhancedTreeValidationOutput = true

// Detailed error messages for validation failures
options.VerboseValidationErrors = true
```

### Input Validation Retry Configuration

```golang
// Configure retry behavior for input validation (handles database timing issues)
options.InputValidationRetries = 5        // Default: 3 retries
options.InputValidationRetryDelay = 3 * time.Second  // Default: 2 seconds
```

**Note:** Input validation includes automatic retry logic to handle cases where the backend database hasn't been updated yet after configuration changes. This prevents false failures due to timing issues between configuration updates and validation checks.

### Operation Retry Configuration

Configure retry behavior for different types of operations to handle transient failures and adapt to different environment reliability characteristics:

```golang
// Configure project operation retries (creation, deletion)
projectRetry := common.ProjectOperationRetryConfig() // Get default config
projectRetry.MaxRetries = 8                          // Increase retries for unreliable environments
projectRetry.InitialDelay = 5 * time.Second         // Longer initial delay
projectRetry.MaxDelay = 60 * time.Second            // Higher maximum delay
options.ProjectRetryConfig = &projectRetry

// Configure catalog operation retries (offering fetches, catalog operations)
catalogRetry := common.CatalogOperationRetryConfig() // Get default config
catalogRetry.MaxRetries = 3                          // Fewer retries for fast environments
catalogRetry.InitialDelay = 1 * time.Second         // Shorter delays
options.CatalogRetryConfig = &catalogRetry

// Configure deployment operation retries
deployRetry := common.DefaultRetryConfig()           // Get default config
deployRetry.Strategy = common.LinearBackoff          // Use linear instead of exponential
deployRetry.MaxRetries = 2                          // Minimal retries for fast execution
options.DeployRetryConfig = &deployRetry
```

**Retry Configuration Types:**

- **`ProjectRetryConfig`**: Controls project creation and deletion retry behavior
  - Default: 5 retries, 3s initial delay, 45s max delay, exponential backoff
  - Handles transient database errors during project operations

- **`CatalogRetryConfig`**: Controls catalog and offering operation retry behavior
  - Default: 5 retries, 3s initial delay, 30s max delay, linear backoff
  - Handles API rate limiting and temporary service unavailability

- **`DeployRetryConfig`**: Controls deployment operation retry behavior
  - Default: 3 retries, 2s initial delay, 30s max delay, exponential backoff
  - Handles deployment timeouts and infrastructure failures

**Default Values:**

When retry configurations are `nil` (default), the framework uses sensible defaults optimized for typical testing environments. Only specify custom retry configurations when you need to adapt to specific environment characteristics.

**Environment-Specific Examples:**

```golang
// High-reliability environment (unstable network, frequent transients)
projectRetry := common.ProjectOperationRetryConfig()
projectRetry.MaxRetries = 8
projectRetry.InitialDelay = 5 * time.Second
projectRetry.MaxDelay = 60 * time.Second

options := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
    Testing: t,
    Prefix: "high-reliability-test",
    ProjectRetryConfig: &projectRetry,
})

// Fast execution environment (stable network, minimal retries needed)
fastRetry := common.DefaultRetryConfig()
fastRetry.MaxRetries = 2
fastRetry.InitialDelay = 1 * time.Second

options := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
    Testing: t,
    Prefix: "fast-test",
    CatalogRetryConfig: &fastRetry,
    DeployRetryConfig: &fastRetry,
})

// Development environment (balanced approach)
// Use defaults by not specifying retry configs (recommended for most cases)
options := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
    Testing: t,
    Prefix: "dev-test",
    // ProjectRetryConfig, CatalogRetryConfig, DeployRetryConfig all nil = use defaults
})
```

**Retry Strategy Options:**

Available retry strategies from `common.RetryStrategy`:
- `common.ExponentialBackoff`: Delay doubles each retry (2s, 4s, 8s, 16s...)
- `common.LinearBackoff`: Delay increases linearly (2s, 4s, 6s, 8s...)
- `common.FixedDelay`: Same delay every retry (2s, 2s, 2s, 2s...)

**Best Practices:**

- **Use defaults** for most scenarios - they're optimized for typical IBM Cloud environments
- **Increase retries** in unreliable network environments or during high-load periods
- **Decrease retries** in stable environments or when fast failure feedback is preferred
- **Match strategy to failure type**: exponential for transient issues, linear for rate limiting
- **Test retry configurations** in your specific environment before using in CI/CD pipelines

### Enhanced Debug Output

When input validation fails, the framework automatically provides detailed debug information to help diagnose issues:

**Debug Information Includes:**

- Current configuration state and inputs (with sensitive values redacted)
- All configurations in the project and their current state
- Expected addon configuration details
- Required input validation attempts and results
- Clear identification of configurations in "waiting on inputs" state

**Debug Output Triggers:**

- Missing required inputs detected during validation
- Configurations found in "awaiting_input" state (timing issues)
- Configuration matching failures during validation

**State Detection Improvements:**

The framework now correctly identifies only configurations that are truly in the `awaiting_input` state, avoiding false positives for configurations in other valid states like `awaiting_member_deployment` or `awaiting_validation`.

**Example Debug Output:**

```text
=== INPUT VALIDATION FAILURE DEBUG INFO ===
Found 2 configurations in project:
  Config: my-kms-config (ID: abc123) [IN WAITING LIST]
    State: awaiting_input
    StateCode: awaiting_input
    LocatorID: catalog.def456.version.ghi789
    Current Inputs:
      region: us-south
      prefix: test-prefix
      ibmcloud_api_key: [REDACTED]
      key_protect_plan: (missing)

  Config: my-base-config (ID: def456)
    State: awaiting_member_deployment
    StateCode: awaiting_member_deployment
    LocatorID: catalog.abc123.version.xyz789
    Current Inputs:
      region: us-south
      prefix: test-prefix

Expected addon configuration details:
  Main Addon Name: deploy-arch-ibm-kms
  Main Addon Version: v1.2.3
  Main Addon Config Name: my-kms-config
  Prefix: test-prefix
=== END DEBUG INFO ===
```

This debug output helps identify exactly which inputs are missing, configuration state issues, and timing problems with the backend systems. Configurations marked with `[IN WAITING LIST]` are those actually flagged as requiring input attention.

## Environment Variables

### Required Variables

```golang
// Automatically checked by framework
TF_VAR_ibmcloud_api_key="your-api-key"
```

### Custom Required Variables

```golang
options.RequiredEnvironmentVars = map[string]string{
    "CUSTOM_API_KEY":    os.Getenv("CUSTOM_API_KEY"),
    "EXTERNAL_SERVICE":  os.Getenv("EXTERNAL_SERVICE"),
}
```

## Hook Configuration

### Available Hooks

```golang
// Pre-deployment setup
options.PreDeployHook = func(options *testaddons.TestAddonOptions) error {
    // Custom setup logic
    return nil
}

// Post-deployment validation
options.PostDeployHook = func(options *testaddons.TestAddonOptions) error {
    // Custom validation logic
    return nil
}

// Pre-undeploy preparation
options.PreUndeployHook = func(options *testaddons.TestAddonOptions) error {
    // Data backup, final state capture
    return nil
}

// Post-undeploy cleanup
options.PostUndeployHook = func(options *testaddons.TestAddonOptions) error {
    // Cleanup verification, additional cleanup
    return nil
}
```

### Hook Best Practices

- **Error Handling**: Return errors to fail the test
- **Logging**: Use `options.Logger` for consistent output
- **State Access**: Access project ID, config details via options
- **Cleanup**: Post hooks run even if deploy/undeploy fails

## Advanced Configuration

### Custom CloudInfo Service

```golang
// Share CloudInfo service across multiple tests
options.CloudInfoService = myCloudInfoService
```

### Custom Logger

```golang
// Use custom logger implementation
options.Logger = myCustomLogger
```

### Complex Input Configuration

```golang
options.AddonConfig.Inputs = map[string]interface{}{
    "prefix": options.Prefix,
    "region": "us-south",
    "complex_object": map[string]interface{}{
        "setting1": "value1",
        "setting2": []string{"item1", "item2"},
        "setting3": map[string]string{
            "key1": "value1",
            "key2": "value2",
        },
    },
    "boolean_setting": true,
    "numeric_setting": 42,
}
```

### Input Override Behavior (OverrideInputMappings)

**⚠️ IMPORTANT**: By default, user-provided inputs will be **IGNORED** for fields that contain reference values. This is the intended behavior to preserve proper configuration mappings.

```golang
options := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
    Testing: t,
    Prefix:  "test",
    // OverrideInputMappings defaults to false - preserves reference values
})

// Default behavior (OverrideInputMappings: false) - RECOMMENDED
options.OverrideInputMappings = core.BoolPtr(false) // Can be omitted as it's the default
```

**Default Behavior (OverrideInputMappings=false - RECOMMENDED):**

When `OverrideInputMappings` is `false` (default), the framework preserves existing reference values (those starting with "ref:") and **ignores user-provided replacement values** for those fields.

```golang
// Example: Configuration has existing reference mapping
// Existing config input: "existing_kms_instance_crn": "ref:../kms-config.instance_crn"

options.AddonConfig.Inputs = map[string]interface{}{
    "prefix": "my-test",
    "existing_kms_instance_crn": "user-provided-value", // ⚠️ THIS WILL BE IGNORED
    "region": "us-south", // ✅ This will be used (no existing reference)
}

// Result after merging:
// - existing_kms_instance_crn: "ref:../kms-config.instance_crn" (preserved)
// - region: "us-south" (user value used)
// - prefix: "my-test" (user value used)
```

**Why User Inputs Are Ignored for Reference Fields:**

- **Input mappings** (references starting with "ref:") connect configuration outputs between different components
- **End users** are not expected to modify these reference mappings
- **Preserving references** maintains proper dependency relationships between configurations
- **Breaking references** can cause deployment failures or incorrect resource connections

**Override Behavior (OverrideInputMappings=true - DEVELOPMENT/TESTING ONLY):**

```golang
// Override mode - replaces ALL input values including references
options.OverrideInputMappings = core.BoolPtr(true) // Use with caution

options.AddonConfig.Inputs = map[string]interface{}{
    "prefix": "my-test",
    "existing_kms_instance_crn": "user-provided-value", // ✅ This will override the reference
}

// Result: ALL user-provided values replace existing values
// - existing_kms_instance_crn: "user-provided-value" (reference overridden)
// - prefix: "my-test" (user value used)
```

**When to Use Each Setting:**

**Use `OverrideInputMappings: false` (default) when:**
- Running production or standard tests (RECOMMENDED)
- Want to preserve proper configuration mappings
- Testing with real dependency relationships
- Following standard addon testing patterns

**Use `OverrideInputMappings: true` when:**
- Development and debugging scenarios
- Need to override reference mappings for testing
- Testing edge cases or configuration variations
- **⚠️ WARNING**: May break dependency relationships

**Identifying Reference Values:**

Reference values that will be preserved (when `OverrideInputMappings: false`) are those that:
- Start with `"ref:"` (e.g., `"ref:../other-config.output_name"`)
- Connect to outputs from other configurations in the same project
- Are automatically generated by the IBM Cloud Projects system

**Best Practices:**

```golang
// ✅ RECOMMENDED: Use default behavior
options := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
    Testing: t,
    Prefix:  "test",
    // OverrideInputMappings defaults to false - no need to specify
})

// ❌ AVOID: Only use override mode when absolutely necessary
options.OverrideInputMappings = core.BoolPtr(true) // Only for special testing scenarios
```

### Resource Group Configuration

```golang
// Use specific resource group
options.ResourceGroup = "my-specific-rg"

// Use default resource group (not recommended for production tests)
options.ResourceGroup = "Default"
```

## Configuration Validation

### Pre-flight Checks

The framework validates configuration before starting tests:

- Required environment variables are set
- Resource group exists and is accessible
- Catalog permissions are sufficient
- Project location is valid

### Runtime Validation

- Input parameter validation against offering schema
- Dependency compatibility checks
- Resource naming collision detection
- Timeout reasonableness checks

## Configuration Examples

### Minimal Configuration

```golang
func TestMinimalAddon(t *testing.T) {
    options := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
        Testing:       t,
        Prefix:        "minimal",
        ResourceGroup: "test-rg",
    })

    options.AddonConfig = cloudinfo.NewAddonConfigTerraform(
        options.Prefix,
        "my-addon",
        "basic",
        map[string]interface{}{
            "prefix": options.Prefix,
        },
    )

    err := options.RunAddonTest()
    assert.NoError(t, err)
}
```

### Comprehensive Configuration

```golang
func TestComprehensiveAddon(t *testing.T) {
    options := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
        Testing:                     t,
        Prefix:                      "comprehensive",
        ResourceGroup:               "test-rg",
        ProjectName:                 "comprehensive-test",
        ProjectDescription:          "Comprehensive addon test",
        ProjectLocation:             "us-east",
        DeployTimeoutMinutes:        180,
        SkipLocalChangeCheck:        false,
        SkipRefValidation:          false,
        SkipDependencyValidation:   false,
        VerboseValidationErrors:    true,
        EnhancedTreeValidationOutput: true,
    })

    options.LocalChangesIgnorePattern = []string{
        ".*\\.md$",
        "^temp/.*",
    }

    options.AddonConfig = cloudinfo.NewAddonConfigTerraform(
        options.Prefix,
        "my-addon",
        "comprehensive",
        map[string]interface{}{
            "prefix":   options.Prefix,
            "region":   "us-east",
            "plan":     "standard",
            "settings": map[string]interface{}{
                "feature1": true,
                "feature2": "enabled",
            },
        },
    )

    options.PreDeployHook = func(opts *testaddons.TestAddonOptions) error {
        opts.Logger.ShortInfo("Starting comprehensive test")
        return nil
    }

    options.PostDeployHook = func(opts *testaddons.TestAddonOptions) error {
        opts.Logger.ShortInfo("Validating comprehensive deployment")
        return nil
    }

    err := options.RunAddonTest()
    assert.NoError(t, err)
}
```

## Project Management

### Project Creation and Isolation

The framework always creates temporary projects for each test to ensure complete isolation. Each test gets its own dedicated project for maximum safety and ease of debugging:

```golang
options := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
    Testing:       t,
    Prefix:        "test",
    ResourceGroup: "my-rg",
})
```

**Project Behavior:**

Each test automatically:

1. Creates a new temporary project with a unique name
2. Deploys the addon configuration within that project
3. Runs validation tests
4. Cleans up the project and all resources

```golang
options := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
    Testing:       t,
    Prefix:        "isolated-test",
    ResourceGroup: "my-rg",
})

// Each test creates and cleans up its own temporary project
err1 := options.RunAddonTest()  // Creates project A, runs test, deletes project A
err2 := options.RunAddonTest()  // Creates project B, runs test, deletes project B
```

**Benefits of Project Isolation:**

- **Complete isolation**: Tests cannot interfere with each other
- **Reliable cleanup**: Each test cleans up its own resources
- **Easier debugging**: Failed tests don't affect other tests
- **CI/CD friendly**: Safe for parallel execution in pipelines

**Resource Sharing Options:**

While projects are always isolated, catalogs can optionally be shared for efficiency:

```golang
// Default: each test creates its own catalog and project
options.SharedCatalog = core.BoolPtr(false)  // Each test creates own catalog

// Efficient: share catalogs between tests (projects still isolated)
options.SharedCatalog = core.BoolPtr(true)   // Share catalogs between tests
```

## Testing Methods

The addon testing framework provides several methods for running tests, each optimized for different use cases:

### RunAddonTest() - Single Test Execution

The primary method for running a single addon test with full lifecycle management:

```golang
func TestBasicAddon(t *testing.T) {
    t.Parallel()

    options := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
        Testing: t,
        Prefix:  "basic-addon",
        AddonConfig: cloudinfo.NewAddonConfigTerraform(
            "basic-addon",
            "my-addon",
            "standard",
            map[string]interface{}{
                "prefix": "basic-addon",
                "region": "us-south",
            },
        ),
    })

    err := options.RunAddonTest()
    assert.NoError(t, err)
}
```

### RunAddonTestMatrix() - Matrix Testing

For running multiple test scenarios with custom configurations:

```golang
func TestAddonMatrix(t *testing.T) {
    testCases := []testaddons.AddonTestCase{
        {
            Name:   "BasicScenario",
            Prefix: "basic",
        },
        {
            Name:   "CustomScenario",
            Prefix: "custom",
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
```

### RunAddonPermutationTest() - Dependency Permutation Testing

For automatically testing all possible dependency combinations:

```golang
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

#### RunAddonPermutationTest() Configuration

The `RunAddonPermutationTest()` method automatically configures several settings:

- **Logging Mode**: Per-test quiet mode with suite-level progress visible
- **Infrastructure Deployment**: Set to `SkipInfrastructureDeployment: true` for all permutations
- **Parallel Execution**: Uses matrix testing infrastructure for efficient parallel execution
- **Dependency Discovery**: Parses the local `ibm_catalog.json` to discover direct dependencies and their flavors
- **Permutation Generation**: For each enabled/disabled subset, enumerates all flavor combinations for enabled dependencies (excludes the default “all enabled” case)

#### Required Configuration for Permutation Testing

```golang
options := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
    Testing: t,                        // Required: testing.T object
    Prefix:  "addon-perm",             // Required: unique prefix for resources
    AddonConfig: cloudinfo.AddonConfig{
        OfferingName:   "my-addon",    // Required: addon name
        OfferingFlavor: "standard",    // Required: addon flavor
        Inputs: map[string]interface{}{ // Required: addon inputs
            "prefix": "addon-perm",
            "region": "us-south",
            // Include all required inputs for your addon
        },
    },
})
```

#### Permutation Test Behavior

- **Automatic Discovery**: Discovers direct dependencies from local `ibm_catalog.json`
- **Validation-Only**: All permutations skip infrastructure deployment for efficiency
- **Parallel Execution**: All permutations run in parallel
- **Per-test Quiet Mode**: Shows detailed logs only on failure; high-level progress remains visible
- **Excludes Default**: Excludes the "all enabled" case; other subsets expand to all flavor combinations

### Method Comparison

| Method | Use Case | Manual Configuration | Dependency Testing | Full Deployment |
|--------|----------|---------------------|-------------------|-----------------|
| `RunAddonTest()` | Single test scenario | Manual | Manual | Yes |
| `RunAddonTestMatrix()` | Multiple custom scenarios | Manual | Manual | Configurable |
| `RunAddonPermutationTest()` | All dependency combinations | Automatic | Automatic | No (validation-only) |

### Choosing the Right Method

**Use `RunAddonTest()` when:**
- Testing a single, specific scenario
- Need full deployment testing
- Want maximum control over configuration

**Use `RunAddonTestMatrix()` when:**
- Testing multiple specific scenarios
- Need custom configuration for each test case
- Want to mix deployment and validation tests
- Need explicit control over dependencies

**Use `RunAddonPermutationTest()` when:**
- Want to test all possible dependency combinations
- Need comprehensive dependency validation
- Want zero-maintenance permutation testing
- Focused on validation rather than deployment
### Dependency Processing Behavior

- User configuration is the source of truth for which direct dependencies are enabled or disabled.
- Catalog metadata is used to fill in version locators and flavors and to determine true direct dependencies (from the catalog version’s `SolutionInfo.Dependencies`).
- Only enabled branches are traversed when building the dependency tree; disabled branches are pruned.
- Direct dependencies marked `on_by_default` in the catalog are automatically included if not specified by the user, provided they are true direct dependencies of the current addon.

### Required Dependencies and StrictMode

If a required dependency is explicitly disabled, the framework force-enables it and marks it as required.

- StrictMode=true (default): logs warnings and continues.
- StrictMode=false: logs informational messages and captures warnings for the final report.
