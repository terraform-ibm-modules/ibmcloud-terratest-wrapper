# Addon Testing

## Overview

The `testaddons` package provides a framework for testing IBM Cloud add-ons. It allows you to run tests against different add-on configurations and validate their behavior.

### Built-in Validations

The framework performs several automated validations to ensure reliable and reproducible tests:

- **Local Change Check**: Verifies that all local changes are committed or pushed before deploying to ensure reproducible builds
- **Reference Validation**: Validates that all configuration references (inputs starting with `ref:/`) can be resolved before deployment
- **Dependency Validation**: Ensures that expected dependencies are deployed and configured correctly
- **Environment Variable Validation**: Checks that required environment variables (like `TF_VAR_ibmcloud_api_key`) are set

Each validation can be individually disabled using skip flags (e.g., `SkipLocalChangeCheck`, `SkipRefValidation`, `SkipDependencyValidation`) if needed for specific testing scenarios.

### Testing Process

The general process for testing an add-on that the framework handles for the user is as follows:

1. **Pre-deployment Validations**: Performs the built-in validations described above
2. **Git Context Discovery**: Automatically determines the repository URL and branch from the current Git context (no need to specify repository URL or branch in your test configuration)
3. **Catalog Setup**: Creates a temporary catalog in the IBM Cloud account
4. **Offering Import**: Imports the offering/addon from the current branch to the temporary catalog
5. **Project Creation**: Creates a test project in IBM Cloud
6. **Configuration Deployment**: Deploys the addon configuration to the project
7. **Dependency Processing**: Deploys dependent configurations that are marked as `on by default` unless explicitly configured otherwise through the `Enabled` flag
8. **Authentication Setup**: Updates the configurations with proper API key authentication using the `TF_VAR_ibmcloud_api_key` environment variable
9. **Input Configuration**: Updates input configurations based on values provided in the test options
10. **Deploy Operation**: Runs a deploy operation and waits for completion
11. **Undeploy Operation**: Runs an undeploy operation and waits for completion
12. **Cleanup**: Cleans up by deleting the project and catalog

During this process, the framework will log detailed information about each step, including the repository URL and branch, catalog and project IDs, and the dependencies that are deployed.

### Hook Points for Custom Code

The framework provides several hook points where you can inject custom code to extend the testing process. These hooks allow you to:

- **Add custom configuration** before deployment starts
- **Perform additional validation** after deployment completes
- **Execute custom tests** against deployed resources
- **Implement custom cleanup** logic
- **Add logging or monitoring** at specific stages

#### Available Hooks

- **`PreDeployHook`**: Executed after project setup but before the deploy operation begins
  - Use for: Custom configuration, pre-deployment checks, environment setup
- **`PostDeployHook`**: Executed immediately after successful deployment
  - Use for: Custom validation, integration tests, resource verification
- **`PreUndeployHook`**: Executed before the undeploy operation begins
  - Use for: Data backup, final state capture, pre-cleanup validation
- **`PostUndeployHook`**: Executed after successful undeploy but before project cleanup
  - Use for: Cleanup verification, final tests, custom cleanup logic

#### Hook Usage Examples

```golang
// Example: Pre-deployment configuration
options.PreDeployHook = func(options *testaddons.TestAddonOptions) error {
    // Configure additional environment variables
    os.Setenv("CUSTOM_CONFIG", "value")

    // Validate custom prerequisites
    if err := validateCustomPrerequisites(); err != nil {
        return fmt.Errorf("custom prerequisites failed: %w", err)
    }

    options.Logger.ShortInfo("Custom pre-deployment configuration completed")
    return nil
}

// Example: Post-deployment testing and validation
options.PostDeployHook = func(options *testaddons.TestAddonOptions) error {
    // Test custom endpoints or services
    if err := testCustomEndpoints(options.AddonConfig); err != nil {
        return fmt.Errorf("custom endpoint tests failed: %w", err)
    }

    // Validate deployed resources meet custom requirements
    if err := validateDeployedResources(options.ProjectID); err != nil {
        return fmt.Errorf("resource validation failed: %w", err)
    }

    options.Logger.ShortInfo("Custom post-deployment validation passed")
    return nil
}

// Example: Pre-undeploy data preservation
options.PreUndeployHook = func(options *testaddons.TestAddonOptions) error {
    // Export important data before cleanup
    if err := exportTestData(options.ProjectID); err != nil {
        return fmt.Errorf("data export failed: %w", err)
    }

    // Verify final state before teardown
    if err := captureFinialState(options.AddonConfig); err != nil {
        return fmt.Errorf("state capture failed: %w", err)
    }

    return nil
}

// Example: Post-undeploy cleanup verification
options.PostUndeployHook = func(options *testaddons.TestAddonOptions) error {
    // Verify all resources were properly cleaned up
    if err := verifyCleanupComplete(options.ResourceGroup); err != nil {
        return fmt.Errorf("cleanup verification failed: %w", err)
    }

    // Perform additional cleanup if needed
    if err := performAdditionalCleanup(); err != nil {
        return fmt.Errorf("additional cleanup failed: %w", err)
    }

    options.Logger.ShortInfo("Custom cleanup verification completed")
    return nil
}
```

**Important Notes:**

- All hooks receive the current `TestAddonOptions` object, giving access to configuration, logger, and test context
- If any hook returns an error, the test will fail and cleanup will be triggered
- Hooks are optional - only implement the ones you need for your specific use case
- Use the built-in logger (`options.Logger`) for consistent logging output

## Examples

### Example 1: Basic Test with Terraform Add-on (Recommended)

This example shows how to run a basic test using the `testaddons` package with a Terraform add-on. **This is the standard and recommended approach for most addon testing scenarios.** It sets up the test options, including the prefix and resource group, and then runs the addon test.

```golang
package test

import (
 "testing"

 "github.com/IBM/go-sdk-core/v5/core"
 "github.com/stretchr/testify/assert"
 "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/cloudinfo"
 "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testaddons"
)

func setupAddonOptions(t *testing.T, prefix string) *testaddons.TestAddonOptions {
 options := testaddons.TestAddonsOptionsDefault(&testaddons.TestAddonOptions{
  Testing:              t,
  Prefix:               prefix,
  ResourceGroup:        "my-project-rg",
 })

 return options
}

func TestRunTerraformAddon(t *testing.T) {
 t.Parallel()

 options := setupAddonOptions(t, "test-terraform-addon")

 // Using the standard Terraform helper function (recommended for most use cases)
 options.AddonConfig = cloudinfo.NewAddonConfigTerraform(
  options.Prefix,        // prefix for unique resource naming
  "test-addon",           // offering name
  "test-flavor",          // offering flavor
  map[string]interface{}{ // inputs
   "prefix": options.Prefix,
  },
 )

 err := options.RunAddonTest()
 assert.Nil(t, err, "This should not have errored")
}
```

### Example 2: Using Stack Add-on (Advanced/Rare Use Case)

This example demonstrates how to configure a test with a Stack add-on. **Note: This is an advanced use case that most users won't need.** The Terraform example above is the recommended approach for most scenarios.

```golang
func TestRunStackAddon(t *testing.T) {
 t.Parallel()

 options := setupAddonOptions(t, "test-stack-addon")

 // Using the specialized Stack helper function (rarely needed)
 options.AddonConfig = cloudinfo.NewAddonConfigStack(
  options.Prefix,        // prefix for unique resource naming
  "test-addon",          // offering name
  "test-flavor",         // offering flavor
  map[string]interface{}{ // inputs
   "prefix": options.Prefix,
   "region": "us-south",
  },
 )

 err := options.RunAddonTest()
 assert.Nil(t, err, "This should not have errored")
}
```

### Example 3: Managing Dependencies

The framework provides two ways to configure addon dependencies:

#### Option 1: Automatic Dependency Discovery

By default, the framework automatically discovers and processes dependencies from the addon's component references. Dependencies marked as "on by default" will be automatically enabled and deployed.

```golang
func TestRunAddonWithAutoDependencies(t *testing.T) {
 t.Parallel()

 options := setupAddonOptions(t, "auto-dependencies")

 // Create the base addon config - dependencies will be auto-discovered
 options.AddonConfig = cloudinfo.NewAddonConfigTerraform(
  options.Prefix,
  "example-app",
  "standard",
  map[string]interface{}{
   "prefix": options.Prefix,
   "region": "us-south",
  },
 )

 err := options.RunAddonTest()
 assert.Nil(t, err, "This should not have errored")
}
```

#### Option 2: Manual Dependency Configuration

You can override automatic dependency discovery by explicitly setting the `Dependencies` array:

```golang
func TestRunAddonWithCustomDependencyConfig(t *testing.T) {
 t.Parallel()

 options := setupAddonOptions(t, "custom-dependency-config")

 // Create the base addon config
 options.AddonConfig = cloudinfo.NewAddonConfigStack(
  options.Prefix,
  "example-app",
  "standard",
  map[string]interface{}{
   "prefix": options.Prefix,
   "region": "us-south",
  },
 )

 // Override dependencies by directly assigning to the Dependencies array
 options.AddonConfig.Dependencies = []cloudinfo.AddonConfig{
  {
   // First dependency - explicitly enable
   OfferingName:   "database",
   OfferingFlavor: "postgresql",
   Inputs: map[string]interface{}{
    "prefix": options.Prefix,
    "plan":   "standard",
   },
   Enabled: core.BoolPtr(true), // explicitly enable this dependency
  },
  {
   // Second dependency - explicitly disable
   OfferingName:   "monitoring",
   OfferingFlavor: "basic",
   Enabled: core.BoolPtr(false), // explicitly disable this dependency
  },
 }

 err := options.RunAddonTest()
 assert.Nil(t, err, "This should not have errored")
}
```

#### Dependency Configuration Options

When manually configuring dependencies, you can set the following options:

- `OfferingName` - The name of the dependency offering
- `OfferingFlavor` - The flavor of the dependency offering
- `Enabled` - Pointer to bool to explicitly enable/disable the dependency
- `OnByDefault` - Pointer to bool indicating if this dependency is on by default
- `Inputs` - Map of input variables for the dependency
- `ExistingConfigID` - Use an existing configuration instead of deploying a new one

**Note:** When you manually set the `Dependencies` array, it overrides the automatic dependency discovery. The framework will still populate metadata fields (like `VersionLocator`, `CatalogID`, etc.) from component references, but user-defined settings take precedence.

## Helper Functions

The framework provides convenient helper functions to create addon configurations for different install types:

### `cloudinfo.NewAddonConfigTerraform()` **Primary Use Case**

Creates a new AddonConfig with Terraform install kind. **This is the main function you should use** for Terraform-based modules and is the expected approach for most addon testing scenarios.

```golang
options.AddonConfig = cloudinfo.NewAddonConfigTerraform(
    options.Prefix,        // prefix for unique resource naming
    "offering-name",       // offering name in catalog
    "flavor-name",         // offering flavor (e.g., "standard", "basic")
    map[string]interface{}{ // inputs
        "prefix": options.Prefix,
        "region": "us-south",
    },
)
```

### `cloudinfo.NewAddonConfigStack()` **Advanced/Rare Use Case**

Creates a new AddonConfig with Stack install kind. This function is provided for API completeness but is **rarely used in practice**. Most users should use the Terraform version above. Only use this if you specifically need to test stack-based deployable architectures.

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

## Advanced Configuration Options

The `TestAddonOptions` structure provides several configuration options that you can use to customize your addon tests:

### Basic Configuration

- `Testing` - Required testing.T object
- `Prefix` - A unique prefix for all resources created during testing
- `ResourceGroup` - The resource group where resources will be created
- `RequiredEnvironmentVars` - Environment variables required for testing (TF_VAR_ibmcloud_api_key is required by default)

### Project Configuration

- `ProjectName` - The name of the test project (defaults to "addon" + prefix)
- `ProjectDescription` - Description of the test project
- `ProjectLocation` - The location for the project
- `ProjectDestroyOnDelete` - Whether to destroy resources when deleting the project
- `ProjectMonitoringEnabled` - Whether project monitoring is enabled
- `ProjectAutoDeploy` - Whether to automatically deploy configurations
- `ProjectEnvironments` - Define custom environments

### Catalog Configuration

- `CatalogUseExisting` - Whether to use an existing catalog
- `CatalogName` - The name of the catalog to create/use

### Testing Options

- `DeployTimeoutMinutes` - Max time to wait for deployment (defaults to 6 hours)
- `SkipTestTearDown` - Skip cleanup after tests
- `SkipUndeploy` - Skip the undeploy step
- `SkipProjectDelete` - Don't delete the project after testing
- `SkipInfrastructureDeployment` - Skip infrastructure deployment and undeploy operations while performing all other validations
- `SkipLocalChangeCheck` - Skip checking for uncommitted local changes
- `SkipRefValidation` - Skip reference validation before deploying
- `SkipDependencyValidation` - Skip dependency validation before deploying
- `VerboseValidationErrors` - Show detailed individual error messages instead of consolidated summary
- `EnhancedTreeValidationOutput` - Show dependency tree with validation status annotations
- `LocalChangesIgnorePattern` - Regex patterns for files to ignore in local change check

#### Infrastructure Deployment

The framework performs both deploy and undeploy operations by default. You can skip the infrastructure deployment and undeploy while still performing all other validations:

```golang
options.SkipInfrastructureDeployment = true
```

**Note**: This option skips both the deployment (`TriggerDeployAndWait()`) and undeploy (`TriggerUnDeployAndWait()`) operations. All other validations including reference validation, dependency validation, and hook execution will still be performed.

### Validation Options

The framework performs several validation steps by default. You can skip specific validations if needed:

#### Reference Validation

By default, the framework validates that all configuration references (inputs starting with `ref:/`) can be resolved. You can skip this validation:

```golang
options.SkipRefValidation = true
```

#### Dependency Validation

The framework validates that expected dependencies are deployed and configured correctly. You can skip this validation:

```golang
options.SkipDependencyValidation = true
```

#### Validation Error Output Format

By default, the framework provides a clean, consolidated summary of dependency validation errors. If you prefer to see detailed individual error messages (the original verbose behavior), you can enable verbose mode:

```golang
options.VerboseValidationErrors = true
```

For an enhanced view that shows the dependency tree with validation status annotations, you can enable enhanced tree output:

```golang
options.EnhancedTreeValidationOutput = true
```

The validation output options work in priority order:

1. **Enhanced Tree Output** (`EnhancedTreeValidationOutput = true`): Shows a complete dependency tree with visual indicators for deployment status, dependency errors, and available alternatives
2. **Verbose Mode** (`VerboseValidationErrors = true`): Shows detailed individual error messages (the original verbose behavior)
3. **Consolidated Summary** (default): Provides a clean, consolidated summary that groups errors by type and provides counts

The enhanced tree output is particularly useful for complex dependency scenarios as it provides visual context showing the relationship between components and where issues occur within the dependency hierarchy.

#### Local Change Check

The framework checks for uncommitted local changes before deploying to ensure reproducible builds. You can skip this check:

```golang
options.SkipLocalChangeCheck = true
```

You can also specify additional patterns to ignore certain files during the local change check. Note that the following patterns are ignored by default:

- `^common-dev-assets/.*` - common development assets directory
- `^tests/.*` - tests directory
- `.*\\.json$` - JSON files
- `.*\\.out$` - output files

To add additional ignore patterns:

```golang
options.LocalChangesIgnorePattern = []string{
    ".*\\.md$",         // ignore all markdown files
    "^docs/.*",         // ignore all files in docs directory
    "^temp/.*",         // ignore temporary files directory
}
```

### Hooks

The framework provides several hook points where you can inject custom code during the testing process. For detailed information about available hooks and comprehensive examples, see the [Hook Points for Custom Code](#hook-points-for-custom-code) section in the Overview.

**Available hooks:**

- `PreDeployHook` - Executed before deployment
- `PostDeployHook` - Executed after successful deployment
- `PreUndeployHook` - Executed before undeployment
- `PostUndeployHook` - Executed after successful undeployment

**Simple example:**

```golang
options.PreDeployHook = func(options *testaddons.TestAddonOptions) error {
    // Do some extra pre configuration before deploying
    return nil
}

options.PostDeployHook = func(options *testaddons.TestAddonOptions) error {
    // Validate deployed resources or perform tests against them
    return nil
}
```

## Troubleshooting

### Known Issues

#### API Key Validation Failures

**Issue**: Tests may occasionally fail with one of the following errors:

#### API Key Validation Error

```text
Error resolving references: invalid status code: 500, body: {"errors":[{"state":"Failed to validate api key token.","code":"failed_request","message":"Failed to validate api key token."}],"status_code":500,"trace":"..."}
```

**Cause**: This is a known intermittent issue with IBM Cloud's reference resolution service that can occur during the reference validation phase. The service occasionally has temporary issues validating API key tokens, even when the API key is valid.

#### Project Not Found Error

```text
Error resolving references: invalid status code: 404, body: {"errors":[{"state":"Specified provider instance with id 'project-id' could not be found.","code":"not_found","message":"..."}],"status_code":404,"trace":"..."}
```

**Cause**: This is a timing issue that occurs when checking project details too quickly after project creation. The resolver API needs time to be updated with new project information, and querying too soon results in a temporary "not found" error.

**Solution**:

1. **Automatic Retry**: The framework automatically retries reference resolution up to 6 times (initial attempt + 5 retries) with exponential backoff (starting at 2 seconds between attempts) to handle these temporary failures.

2. **Automatic Skip**: When these specific errors occur after exhausting all retries, the framework will automatically skip reference validation for that configuration and continue with the test. The test logs will show warnings indicating that reference validation was skipped due to the intermittent service issue. The test will still fail later if the references are actually invalid during deployment.

3. **Manual Skip**: For development/testing scenarios where you want to completely disable reference validation, you can use:

   ```golang
   options.SkipRefValidation = true
   ```

   **Note**: This disables reference validation entirely for all configurations, whereas the automatic skip (option 2) only skips validation when the specific intermittent error occurs.
