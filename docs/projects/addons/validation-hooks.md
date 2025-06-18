# Validation and Hooks Guide

This guide covers the built-in validation system and hook points available in the addon testing framework.

## Built-in Validations

The framework performs several automated validations to ensure reliable and reproducible tests. Each validation can be individually controlled.

### Local Change Validation

**Purpose**: Ensures reproducible builds by checking for uncommitted local changes.

**What it checks:**

- Uncommitted changes in tracked files
- Untracked files that might affect the build
- Modified files that haven't been staged

**Configuration:**

```golang
// Skip local change check entirely
options.SkipLocalChangeCheck = true

// Configure ignore patterns (in addition to defaults)
options.LocalChangesIgnorePattern = []string{
    ".*\\.md$",        // ignore markdown files
    "^docs/.*",        // ignore docs directory
    "^temp/.*",        // ignore temporary files
    ".*\\.log$",       // ignore log files
}
```

**Default ignore patterns:**

- `^common-dev-assets/.*` - common development assets directory
- `^tests/.*` - tests directory
- `.*\\.json$` - JSON files (except `ibm_catalog.json` which is always tracked)
- `.*\\.out$` - output files

**When to skip:**

- Development/debugging scenarios
- CI environments where changes are expected
- Testing non-production branches

### Reference Validation

**Purpose**: Validates that all configuration references can be resolved before deployment.

**What it checks:**

- References starting with `ref:/` in addon inputs
- Dependency reference resolution
- Cross-configuration reference validity

**Configuration:**

```golang
// Skip reference validation
options.SkipRefValidation = true
```

**Automatic retry behavior:**

- Retries up to 6 times with exponential backoff
- Handles intermittent API issues automatically
- Automatically skips validation on known transient errors

**When to skip:**

- Known issues with reference resolution service
- Testing scenarios where references aren't critical
- Debugging non-reference-related issues

### Dependency Validation

**Purpose**: Ensures that expected dependencies are properly configured and available.

**What it checks:**

- Dependency availability in catalogs
- Dependency version compatibility
- Required dependency configurations
- Circular dependency detection

**Configuration:**

```golang
// Skip dependency validation
options.SkipDependencyValidation = true

// Control error output format
options.VerboseValidationErrors = true         // Detailed individual errors
options.EnhancedTreeValidationOutput = true    // Visual dependency tree
```

**Validation output formats:**

1. **Enhanced Tree Output** (when `EnhancedTreeValidationOutput = true`):

   ```text
   Dependency Tree with Validation Status:
   ├── main-addon (✓ valid)
   │   ├── account-base [resource-group-only] (✓ deployed)
   │   └── vpc-addon [standard] (⚠ dependency error)
   │       └── subnet-addon [basic] (✗ not available)
   ```

2. **Verbose Mode** (when `VerboseValidationErrors = true`):

   ```text
   Individual validation errors:
   - Dependency 'subnet-addon' version '1.2.3' not found in catalog
   - Reference 'ref:/vpc/subnet_id' cannot be resolved
   ```

3. **Consolidated Summary** (default):

   ```text
   Dependency validation summary:
   - 2 missing dependencies
   - 1 reference resolution error
   - Total configurations checked: 5
   ```

**When to skip:**

- Testing without dependencies
- Debugging main addon functionality
- Scenarios where dependencies are known to be unavailable

### Environment Variable Validation

**Purpose**: Checks that required environment variables are set.

**Default required variables:**

- `TF_VAR_ibmcloud_api_key` - IBM Cloud API key

**Configuration:**

```golang
// Add custom required variables
options.RequiredEnvironmentVars = map[string]string{
    "TF_VAR_ibmcloud_api_key": os.Getenv("TF_VAR_ibmcloud_api_key"),
    "CUSTOM_API_KEY":          os.Getenv("CUSTOM_API_KEY"),
    "EXTERNAL_SERVICE_URL":    os.Getenv("EXTERNAL_SERVICE_URL"),
}
```

**Note**: This validation cannot be skipped as it's essential for framework operation.

## Hook System

The framework provides four hook points where you can inject custom code into the testing lifecycle.

### PreDeployHook

**When it runs**: After project setup but before the deploy operation begins.

**Use cases:**

- Custom configuration setup
- Pre-deployment checks
- Environment preparation
- External service initialization

**Example:**

```golang
options.PreDeployHook = func(options *testaddons.TestAddonOptions) error {
    // Configure additional environment variables
    os.Setenv("CUSTOM_CONFIG", "value")

    // Validate custom prerequisites
    if err := validateCustomPrerequisites(); err != nil {
        return fmt.Errorf("custom prerequisites failed: %w", err)
    }

    // Setup external dependencies
    if err := setupExternalDependencies(); err != nil {
        return fmt.Errorf("external setup failed: %w", err)
    }

    options.Logger.ShortInfo("Custom pre-deployment configuration completed")
    return nil
}

func validateCustomPrerequisites() error {
    // Check external service availability
    // Validate custom configurations
    // Verify resource quotas
    return nil
}

func setupExternalDependencies() error {
    // Initialize external databases
    // Setup monitoring
    // Configure networking
    return nil
}
```

### PostDeployHook

**When it runs**: Immediately after successful deployment.

**Use cases:**

- Custom validation of deployed resources
- Integration testing
- Resource verification
- Performance testing

**Example:**

```golang
options.PostDeployHook = func(options *testaddons.TestAddonOptions) error {
    // Test custom endpoints
    if err := testCustomEndpoints(options.AddonConfig); err != nil {
        return fmt.Errorf("endpoint tests failed: %w", err)
    }

    // Validate deployed resources
    if err := validateDeployedResources(options.ProjectID); err != nil {
        return fmt.Errorf("resource validation failed: %w", err)
    }

    // Run integration tests
    if err := runIntegrationTests(options); err != nil {
        return fmt.Errorf("integration tests failed: %w", err)
    }

    options.Logger.ShortInfo("Custom post-deployment validation passed")
    return nil
}

func testCustomEndpoints(config cloudinfo.AddonConfig) error {
    // Test API endpoints
    // Validate service responses
    // Check authentication
    return nil
}

func validateDeployedResources(projectID string) error {
    // Check resource states
    // Validate configurations
    // Verify connectivity
    return nil
}

func runIntegrationTests(options *testaddons.TestAddonOptions) error {
    // Test cross-service communication
    // Validate data flows
    // Check monitoring setup
    return nil
}
```

### PreUndeployHook

**When it runs**: Before the undeploy operation begins.

**Use cases:**

- Data backup before cleanup
- Final state capture
- Pre-cleanup validation
- Resource state documentation

**Example:**

```golang
options.PreUndeployHook = func(options *testaddons.TestAddonOptions) error {
    // Export important data before cleanup
    if err := exportTestData(options.ProjectID); err != nil {
        return fmt.Errorf("data export failed: %w", err)
    }

    // Capture final state for analysis
    if err := captureFinialState(options.AddonConfig); err != nil {
        return fmt.Errorf("state capture failed: %w", err)
    }

    // Document resource states
    if err := documentResourceStates(options); err != nil {
        return fmt.Errorf("documentation failed: %w", err)
    }

    options.Logger.ShortInfo("Pre-undeploy preparation completed")
    return nil
}

func exportTestData(projectID string) error {
    // Export databases
    // Save configuration files
    // Backup logs and metrics
    return nil
}

func captureFinialState(config cloudinfo.AddonConfig) error {
    // Take snapshots
    // Export monitoring data
    // Save test results
    return nil
}

func documentResourceStates(options *testaddons.TestAddonOptions) error {
    // Generate resource inventory
    // Document configurations
    // Save deployment artifacts
    return nil
}
```

### PostUndeployHook

**When it runs**: After successful undeploy but before project cleanup.

**Use cases:**

- Cleanup verification
- Final testing
- Custom cleanup logic
- Resource leak detection

**Example:**

```golang
options.PostUndeployHook = func(options *testaddons.TestAddonOptions) error {
    // Verify all resources were properly cleaned up
    if err := verifyCleanupComplete(options.ResourceGroup); err != nil {
        return fmt.Errorf("cleanup verification failed: %w", err)
    }

    // Perform additional cleanup if needed
    if err := performAdditionalCleanup(options); err != nil {
        return fmt.Errorf("additional cleanup failed: %w", err)
    }

    // Check for resource leaks
    if err := checkForResourceLeaks(options.Prefix); err != nil {
        return fmt.Errorf("resource leak check failed: %w", err)
    }

    options.Logger.ShortInfo("Custom cleanup verification completed")
    return nil
}

func verifyCleanupComplete(resourceGroup string) error {
    // Check resource group is empty
    // Verify no orphaned resources
    // Validate billing stops
    return nil
}

func performAdditionalCleanup(options *testaddons.TestAddonOptions) error {
    // Clean external dependencies
    // Remove temporary files
    // Reset configurations
    return nil
}

func checkForResourceLeaks(prefix string) error {
    // Search for resources with test prefix
    // Check across multiple resource groups
    // Validate no unexpected charges
    return nil
}
```

## Hook Best Practices

### Error Handling

```golang
options.PostDeployHook = func(options *testaddons.TestAddonOptions) error {
    // Always provide context in error messages
    if err := validateDeployment(); err != nil {
        return fmt.Errorf("deployment validation failed: %w", err)
    }

    // Use multiple validation steps
    validations := []func() error{
        validateEndpoints,
        validateConnectivity,
        validatePermissions,
    }

    for i, validate := range validations {
        if err := validate(); err != nil {
            return fmt.Errorf("validation step %d failed: %w", i+1, err)
        }
    }

    return nil
}
```

### Logging Best Practices

```golang
options.PreDeployHook = func(options *testaddons.TestAddonOptions) error {
    // Use appropriate log levels
    options.Logger.ShortInfo("Starting custom setup")
    options.Logger.LongInfo("Detailed setup information for debugging")

    // Log important milestones
    options.Logger.ShortInfo("Custom configuration applied")
    options.Logger.ShortInfo("External dependencies verified")

    // Log warnings for non-fatal issues
    if warning := checkOptionalService(); warning != nil {
        options.Logger.Warning("Optional service unavailable: %s", warning)
    }

    return nil
}
```

### State Access Patterns

```golang
options.PostDeployHook = func(options *testaddons.TestAddonOptions) error {
    // Access project information
    projectID := options.ProjectID
    projectName := options.ProjectName

    // Access addon configuration
    addonName := options.AddonConfig.OfferingName
    addonFlavor := options.AddonConfig.OfferingFlavor
    addonInputs := options.AddonConfig.Inputs

    // Access test metadata
    testPrefix := options.Prefix
    resourceGroup := options.ResourceGroup

    // Use information for validation
    return validateProjectState(projectID, addonName, testPrefix)
}
```

### Retry Logic in Hooks

```golang
options.PostDeployHook = func(options *testaddons.TestAddonOptions) error {
    // Implement retry for potentially flaky operations
    maxRetries := 3
    retryDelay := 10 * time.Second

    for attempt := 1; attempt <= maxRetries; attempt++ {
        if err := validateExternalService(); err != nil {
            if attempt == maxRetries {
                return fmt.Errorf("validation failed after %d attempts: %w", maxRetries, err)
            }

            options.Logger.Warning("Validation attempt %d failed, retrying in %v: %s",
                attempt, retryDelay, err)
            time.Sleep(retryDelay)
            continue
        }

        options.Logger.ShortInfo("Validation successful on attempt %d", attempt)
        break
    }

    return nil
}
```

## Validation Troubleshooting

### Common Local Change Issues

**Issue**: Test fails with local changes detected.

**Solutions:**

```golang
// Option 1: Commit or stash changes
git add . && git commit -m "test changes"

// Option 2: Skip validation for development
options.SkipLocalChangeCheck = true

// Option 3: Add ignore patterns
options.LocalChangesIgnorePattern = []string{
    ".*\\.tmp$",
    "^build/.*",
}
```

### Common Reference Resolution Issues

**Issue**: Reference validation fails with API errors.

**Solutions:**

```golang
// Option 1: Let automatic retry handle it (recommended)
// Framework automatically retries and skips on known issues

// Option 2: Skip validation for testing
options.SkipRefValidation = true

// Option 3: Enable verbose output for debugging
options.VerboseValidationErrors = true
```

### Common Dependency Issues

**Issue**: Dependency validation fails.

**Solutions:**

```golang
// Option 1: Use enhanced tree output for diagnosis
options.EnhancedTreeValidationOutput = true

// Option 2: Skip dependency validation
options.SkipDependencyValidation = true

// Option 3: Override dependencies manually
options.AddonConfig.Dependencies = []cloudinfo.AddonConfig{
    // Explicit dependency configuration
}
```

## Advanced Validation Scenarios

### Conditional Validation

```golang
options.PostDeployHook = func(options *testaddons.TestAddonOptions) error {
    // Skip certain validations based on addon flavor
    if options.AddonConfig.OfferingFlavor == "basic" {
        options.Logger.ShortInfo("Skipping advanced validation for basic flavor")
        return nil
    }

    // Run flavor-specific validation
    return validateAdvancedFeatures(options)
}
```

### Environment-Specific Validation

```golang
options.PreDeployHook = func(options *testaddons.TestAddonOptions) error {
    environment := os.Getenv("TEST_ENVIRONMENT")

    switch environment {
    case "ci":
        return validateCIEnvironment(options)
    case "dev":
        return validateDevEnvironment(options)
    case "staging":
        return validateStagingEnvironment(options)
    default:
        return validateDefaultEnvironment(options)
    }
}
```

### Resource-Specific Validation

```golang
options.PostDeployHook = func(options *testaddons.TestAddonOptions) error {
    // Validate different resource types
    validations := map[string]func() error{
        "compute":  validateComputeResources,
        "storage":  validateStorageResources,
        "network":  validateNetworkResources,
        "security": validateSecurityResources,
    }

    for resourceType, validate := range validations {
        if err := validate(); err != nil {
            return fmt.Errorf("%s validation failed: %w", resourceType, err)
        }
        options.Logger.ShortInfo("%s validation passed", resourceType)
    }

    return nil
}
```
