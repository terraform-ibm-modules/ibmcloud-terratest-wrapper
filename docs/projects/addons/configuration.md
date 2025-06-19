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
// Show detailed individual error messages
options.VerboseValidationErrors = true

// Show dependency tree with validation status
options.EnhancedTreeValidationOutput = true
```

**Validation output priority:**

1. **Enhanced Tree Output**: Visual dependency tree with status indicators
2. **Verbose Mode**: Detailed individual error messages
3. **Consolidated Summary** (default): Clean grouped error summary

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
