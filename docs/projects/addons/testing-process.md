# Testing Process Overview

This document explains the detailed testing lifecycle that the addon testing framework automatically handles for you.

## Built-in Validations

The framework performs several automated validations to ensure reliable and reproducible tests:

- **Local Change Check**: Verifies that all local changes are committed or pushed before deploying to ensure reproducible builds
- **Reference Validation**: Validates that all configuration references (inputs starting with `ref:/`) can be resolved before deployment. **Important**: When `SkipInfrastructureDeployment` is enabled (validation-only mode), reference validation becomes critical and will not automatically skip intermittent service errors, as this is the only opportunity to catch reference issues.
- **Dependency Validation**: Ensures that expected dependencies are deployed and configured correctly
- **Environment Variable Validation**: Checks that required environment variables (like `TF_VAR_ibmcloud_api_key`) are set

Each validation can be individually disabled using skip flags (e.g., `SkipLocalChangeCheck`, `SkipRefValidation`, `SkipDependencyValidation`) if needed for specific testing scenarios.

## Testing Lifecycle

The general process for testing an addon that the framework handles automatically:

### 1. Pre-deployment Validations

- **Environment Variable Check**: Validates that required environment variables are set
- **Local Change Check**: Ensures all local changes are committed (unless skipped)
- **Git Context Discovery**: Automatically determines repository URL and branch from Git context
- **Reference Validation**: Validates that all configuration references can be resolved (unless skipped)
- **Dependency Validation**: Validates dependency configuration and availability (unless skipped)

### 2. Setup Phase

- **Catalog Setup**: Creates a temporary catalog in the IBM Cloud account (or uses existing if configured)
- **Offering Import**: Imports the offering/addon from the current branch to the catalog
- **Project Creation**: Creates a test project in IBM Cloud with specified configuration
- **Environment Setup**: Configures project environments and settings

### 3. Configuration Phase

- **Configuration Creation**: Creates addon configuration in the project
- **Dependency Processing**: Deploys dependent configurations that are marked as `on by default` unless explicitly configured
- **Authentication Setup**: Updates configurations with proper API key authentication using `TF_VAR_ibmcloud_api_key`
- **Input Configuration**: Updates input configurations based on values provided in test options

### 4. Hook Execution

- **PreDeployHook**: Executes custom pre-deployment logic if defined
- **Deployment**: Runs deploy operation and waits for completion (unless `SkipInfrastructureDeployment` is true)
- **PostDeployHook**: Executes custom post-deployment validation if defined

### 5. Teardown Phase

- **PreUndeployHook**: Executes custom pre-undeploy logic if defined
- **Undeploy Operation**: Runs undeploy operation and waits for completion (unless skipped)
- **PostUndeployHook**: Executes custom post-undeploy logic if defined

### 6. Cleanup Phase

- **Project Cleanup**: Deletes the test project (unless `SkipProjectDelete` is true)
- **Catalog Cleanup**: Removes the temporary catalog (unless using existing catalog)
- **Resource Verification**: Verifies that resources were properly cleaned up

## Detailed Process Steps

### Git Context Discovery

The framework automatically determines:

- Repository URL from the current Git remote
- Current branch name
- Commit hash for reproducibility

**No manual configuration required** - the framework reads this information from your local Git environment.

### Catalog Management

**Temporary Catalog Creation:**

- Creates a unique catalog named `dev-addon-test-{prefix}`
- Imports the current branch/commit of your addon
- Configures catalog visibility and access permissions

**Existing Catalog Usage:**

- Set `CatalogUseExisting: true` to use an existing catalog
- Specify catalog name with `CatalogName`
- Framework will import/update the offering in the existing catalog

### Project Configuration

**Automatic Project Setup:**

- Creates project with name `addon{prefix}` (customizable)
- Configures project in specified region (defaults to us-south)
- Sets up monitoring and auto-deploy settings
- Configures destroy-on-delete behavior

**Custom Project Settings:**

```golang
options.ProjectName = "my-custom-project"
options.ProjectDescription = "Custom project description"
options.ProjectLocation = "us-east"
options.ProjectDestroyOnDelete = core.BoolPtr(true)
options.ProjectMonitoringEnabled = core.BoolPtr(true)
options.ProjectAutoDeploy = core.BoolPtr(false)
options.ProjectAutoDeployMode = "manual_approval"
```

### Dependency Processing

**Automatic Discovery:**

- Analyzes addon's component references
- Identifies dependencies marked as "on by default"
- Creates configurations for enabled dependencies

**Manual Override:**

```golang
options.AddonConfig.Dependencies = []cloudinfo.AddonConfig{
    {
        OfferingName:   "dependency-name",
        OfferingFlavor: "flavor",
        Enabled:        core.BoolPtr(true),
    },
}
```

### Authentication Configuration

**Automatic API Key Setup:**

- Reads `TF_VAR_ibmcloud_api_key` environment variable
- Configures authentication for all addon configurations
- Validates API key before deployment

**Custom Authentication:**

- Framework handles IBM Cloud authentication automatically
- Additional service-specific authentication can be configured via hooks

### Deployment Operations

**Deploy Phase:**

- Triggers deployment for all enabled configurations
- Waits for completion with configurable timeout (default: 6 hours)
- Monitors deployment status and logs progress
- Handles deployment failures gracefully

**Undeploy Phase:**

- Triggers undeploy for all deployed configurations
- Waits for completion with same timeout settings
- Ensures clean resource removal
- Handles undeploy failures and provides cleanup options

### Error Handling

**Validation Failures:**

- Stops execution before deployment if validations fail
- Provides detailed error messages and resolution guidance
- Allows individual validation steps to be skipped for debugging

**Deployment Failures:**

- Captures detailed error information from IBM Cloud
- Attempts cleanup of partial deployments
- Provides logs and diagnostic information

**Cleanup Failures:**

- Logs cleanup failures but continues with test completion
- Provides manual cleanup guidance
- Ensures test doesn't hang on cleanup issues

## Timeout Configuration

**Default Timeouts:**

- Deployment: 6 hours (360 minutes)
- Undeploy: 6 hours (360 minutes)

**Custom Timeout:**

```golang
options.DeployTimeoutMinutes = 120 // 2 hours
```

## Skip Options

The framework provides several options to skip parts of the testing process:

```golang
// Skip entire teardown process
options.SkipTestTearDown = true

// Skip undeploy but keep project cleanup
options.SkipUndeploy = true

// Skip project deletion
options.SkipProjectDelete = true

// Skip infrastructure deployment but perform all validations
options.SkipInfrastructureDeployment = true

// Skip individual validations
options.SkipLocalChangeCheck = true
options.SkipRefValidation = true
options.SkipDependencyValidation = true
```

**Important Notes for Validation-Only Mode:**

When `SkipInfrastructureDeployment` is enabled, the framework operates in validation-only mode where:

- **Reference validation becomes stricter**: Intermittent service errors that would normally be automatically skipped are treated as failures, since there won't be a deployment phase to validate references later
- **All pre-deployment validations are critical**: Any validation failures should be addressed since they won't be caught during deployment
- **Dependencies are still processed**: The framework still creates and validates dependency configurations to ensure the complete setup would work

## Logging and Monitoring

**Built-in Logging:**

- Detailed logs for each phase of the testing process
- Progress updates during long-running operations
- Error details with context and resolution guidance

**Log Identification:**

The framework uses a hierarchical approach to identify test cases in log output:

1. **TestCaseName (if set)**: `[TestFunction - ADDON - TestCaseName]`
2. **ProjectName (default)**: `[TestFunction - ADDON - ProjectName]` (e.g., `addonmy-test-xu5oby`)
3. **Fallback**: `[TestFunction - ADDON]`

```golang
// Custom test case identification (recommended for clarity)
options.TestCaseName = "ProductionValidation"
// Logs: [TestMyAddon - ADDON - ProductionValidation] Starting test...

// Default behavior uses ProjectName with prefix + random suffix
// Logs: [TestMyAddon - ADDON - addonmy-test-xu5oby] Starting test...
```

**Matrix Tests**: Automatically use the test case `Name` field for `TestCaseName`, providing clear identification without manual configuration.

**Custom Logging:**

```golang
options.Logger.ShortInfo("Custom log message")
options.Logger.LongInfo("Detailed log message")
options.Logger.Warning("Warning message")
```

**Monitoring Integration:**

- Project monitoring can be enabled/disabled
- Framework logs project and configuration IDs for external monitoring
- Integration with IBM Cloud monitoring services when enabled

## Resource Management

**Resource Naming:**

- All resources use the specified prefix for uniqueness
- Automatic random suffix generation for test isolation
- Consistent naming patterns across all created resources

**Resource Groups:**

- Creates resources in specified resource group
- Validates resource group access before deployment
- Uses "Default" resource group if none specified

**Cleanup Verification:**

- Verifies resource deletion after undeploy
- Reports any resources that weren't properly cleaned up
- Provides guidance for manual cleanup if needed
