# Configuration Guide

This guide covers all configuration options available in the addon testing framework, from basic setup to advanced customization.

## Basic Configuration

### Required Options

```golang
options := testaddons.TestAddonsOptionsDefault(&testaddons.TestAddonOptions{
    Testing:       t,                // Required: testing.T object
    Prefix:        "my-test",        // Required: unique prefix for resources
    ResourceGroup: "my-project-rg",  // Required: resource group for project
})
```

### Optional Basic Settings

```golang
options := testaddons.TestAddonsOptionsDefault(&testaddons.TestAddonOptions{
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
    options := testaddons.TestAddonsOptionsDefault(&testaddons.TestAddonOptions{
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
options := testaddons.TestAddonsOptionsDefault(&testaddons.TestAddonOptions{
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
isolatedOptions := testaddons.TestAddonsOptionsDefault(&testaddons.TestAddonOptions{
    Testing:       t,
    Prefix:        "isolated-test",
    ResourceGroup: "my-rg",
    SharedCatalog: core.BoolPtr(false),  // Can be omitted as it's the default
})

// Each test creates and cleans up its own catalog + offering
err1 := isolatedOptions.RunAddonTest()  // Creates & deletes catalog A
err2 := isolatedOptions.RunAddonTest()  // Creates & deletes catalog B

// SharedCatalog = true - efficient sharing (requires manual cleanup)
baseOptions := testaddons.TestAddonsOptionsDefault(&testaddons.TestAddonOptions{
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
baseOptions := testaddons.TestAddonsOptionsDefault(&testaddons.TestAddonOptions{
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
    baseOptions := testaddons.TestAddonsOptionsDefault(&testaddons.TestAddonOptions{
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
    options := testaddons.TestAddonsOptionsDefault(&testaddons.TestAddonOptions{
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
    options := testaddons.TestAddonsOptionsDefault(&testaddons.TestAddonOptions{
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
options := testaddons.TestAddonsOptionsDefault(&testaddons.TestAddonOptions{
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
options := testaddons.TestAddonsOptionsDefault(&testaddons.TestAddonOptions{
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
