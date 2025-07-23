# Real-World Testing Scenarios

## For Test Writers

This guide shows distinct real-world scenarios for configuring logging in different testing environments. Each example demonstrates a complete, practical use case.

## Scenario 1: GitHub Actions CI Pipeline

Testing addon deployment in GitHub Actions with clean output and detailed failure diagnostics.

```golang
package test

import (
    "os"
    "testing"
    "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testaddons"
    "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/cloudinfo"
)

func TestWebAppAddonCI(t *testing.T) {
    t.Parallel()

    // Detect GitHub Actions environment
    isCI := os.Getenv("GITHUB_ACTIONS") == "true"

    options := testaddons.TestAddonsOptionsDefault(&testaddons.TestAddonOptions{
        Testing:           t,
        Prefix:            "gh-webapp",
        ResourceGroup:     "webapp-test-rg",
        QuietMode:         &isCI,               // Quiet in CI, verbose locally
        VerboseOnFailure:  true,               // Always show failure details
    })

    // Configure web application addon
    options.AddonConfig = cloudinfo.NewAddonConfigTerraform(
        options.Prefix,
        "webapp-addon",
        "standard",
        map[string]interface{}{
            "instance_count": 2,
            "environment":    "test",
            "region":         "us-south",
        },
    )

    // CI Output (success):
    // 🔄 Setting up project infrastructure
    // 🔄 Deploying configurations
    // ✅ Addon deployment completed

    output, err := options.RunAddonTest()
    if err != nil {
        // CI Output (failure): Full debug logs shown here
        t.Fatalf("Web app addon deployment failed: %v", err)
    }

    // Validate deployment outputs
    if output["webapp_url"] == "" {
        t.Fatal("Expected webapp_url output not found")
    }
}
```

## Scenario 2: Local Development with Multiple Test Types

Developer working locally with different test packages, needs immediate feedback.

```golang
package test

import (
    "testing"
    "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testaddons"
    "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testhelper"
    "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testschematic"
)

// Local development - immediate output for debugging
func TestDatabaseAddonLocal(t *testing.T) {
    // No t.Parallel() for sequential execution during development

    quietMode := false
    options := testaddons.TestAddonsOptionsDefault(&testaddons.TestAddonOptions{
        Testing:           t,
        Prefix:            "local-db",
        ResourceGroup:     "dev-test-rg",
        QuietMode:         &quietMode,          // See all output immediately
        VerboseOnFailure:  true,               // Extra context on failure
    })

    // Verbose output during execution:
    // [local-db] Getting offering details from catalog
    // [local-db] Found offering: database-addon v2.1.3
    // [local-db] Validating configuration inputs
    // [local-db] Creating project with database configuration
    // 🔄 Setting up project infrastructure
    // [local-db] Deploying configuration: postgres-db
    // [local-db] Waiting for deployment status: in_progress
    // [local-db] Deployment status: completed
    // ✅ Addon deployment completed

    output, err := options.RunAddonTest()
    if err != nil {
        t.Fatalf("Database addon test failed: %v", err)
    }
}

// Compare with basic Terraform test - same project, different approach
func TestDatabaseTerraformLocal(t *testing.T) {
    quietMode := false
    options := testhelper.TestOptionsDefault(&testhelper.TestOptions{
        Testing:           t,
        TerraformDir:      "examples/database",
        Prefix:            "local-tf-db",
        QuietMode:         &quietMode,          // Immediate feedback
        VerboseOnFailure:  true,
    })

    // Different output pattern for Terraform tests:
    // [local-tf-db] Running Terraform Init
    // [local-tf-db] Terraform initialized successfully
    // [local-tf-db] Running Terraform Plan
    // 🔄 Planning infrastructure
    // [local-tf-db] Plan shows 5 resources to create
    // [local-tf-db] Running Terraform Apply
    // 🔄 Applying infrastructure
    // ✅ Infrastructure applied successfully

    output, err := options.RunTestConsistency()
    if err != nil {
        t.Fatalf("Terraform consistency test failed: %v", err)
    }
}
```

## Scenario 3: Large Integration Test Suite

Performance-conscious testing of multiple microservices with batched operations.

```golang
package test

import (
    "fmt"
    "testing"
    "sync"
    "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testaddons"
)

// Parent test coordinates multiple parallel addon tests
func TestMicroservicesIntegrationSuite(t *testing.T) {
    services := []struct {
        name       string
        addonName  string
        config     map[string]interface{}
    }{
        {"auth-service", "auth-addon", map[string]interface{}{"replicas": 3}},
        {"api-gateway", "gateway-addon", map[string]interface{}{"rate_limit": 1000}},
        {"user-service", "users-addon", map[string]interface{}{"db_tier": "standard"}},
        {"notification-service", "notify-addon", map[string]interface{}{"queue_size": 100}},
        {"analytics-service", "analytics-addon", map[string]interface{}{"storage_gb": 50}},
    }

    var wg sync.WaitGroup
    results := make(chan testResult, len(services))

    for _, service := range services {
        wg.Add(1)
        go func(svc serviceConfig) {
            defer wg.Done()

            t.Run(svc.name, func(t *testing.T) {
                t.Parallel()

                quietMode := true
                options := testaddons.TestAddonsOptionsDefault(&testaddons.TestAddonOptions{
                    Testing:           t,
                    Prefix:            fmt.Sprintf("suite-%s", svc.name),
                    ResourceGroup:     "microservices-rg",
                    QuietMode:         &quietMode,      // Clean parallel output
                    VerboseOnFailure:  true,           // Debug on failure

                    // Performance optimization for large suites
                    DeployTimeoutMinutes: 45,          // Longer timeout for busy environments
                })

                options.AddonConfig = cloudinfo.NewAddonConfigTerraform(
                    options.Prefix,
                    svc.addonName,
                    "production",
                    svc.config,
                )

                // Clean parallel output across all services:
                // 🔄 Setting up project infrastructure (auth-service)
                // 🔄 Setting up project infrastructure (api-gateway)
                // 🔄 Deploying configurations (auth-service)
                // 🔄 Deploying configurations (user-service)
                // ✅ Addon deployment completed (auth-service)
                // ✅ Addon deployment completed (api-gateway)

                output, err := options.RunAddonTest()
                results <- testResult{service: svc.name, output: output, err: err}
            })
        }(service)
    }

    go func() {
        wg.Wait()
        close(results)
    }()

    // Collect and validate results
    var failures []string
    for result := range results {
        if result.err != nil {
            failures = append(failures, fmt.Sprintf("%s: %v", result.service, result.err))
        }
    }

    if len(failures) > 0 {
        t.Fatalf("Integration suite failures:\n%s", strings.Join(failures, "\n"))
    }
}

type serviceConfig struct {
    name      string
    addonName string
    config    map[string]interface{}
}

type testResult struct {
    service string
    output  map[string]interface{}
    err     error
}
```

## Scenario 4: Multi-Environment Deployment Pipeline

Testing the same infrastructure across development, staging, and production configurations.

```golang
package test

import (
    "strings"
    "testing"
    "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testprojects"
)

func TestMultiEnvironmentDeployment(t *testing.T) {
    environments := []struct {
        name     string
        region   string
        replicas int
        tier     string
    }{
        {"development", "us-south", 1, "lite"},
        {"staging", "us-east", 2, "standard"},
        {"production", "eu-gb", 3, "premium"},
    }

    for _, env := range environments {
        t.Run(env.name, func(t *testing.T) {
            t.Parallel()

            // Environment-specific configuration
            quietMode := env.name == "production" // Extra quiet for prod tests

            options := testprojects.TestProjectsOptionsDefault(&testprojects.TestProjectsOptions{
                Testing:               t,
                Prefix:                fmt.Sprintf("%s-deploy", env.name),
                QuietMode:             &quietMode,
                VerboseOnFailure:      true,

                // Environment-specific settings
                StackDefinitionPath:   fmt.Sprintf("stacks/%s-stack.json", env.name),
                ProjectEnvironments:   createEnvironmentConfig(env),
                DeployTimeoutMinutes:  getTimeoutForEnvironment(env.name),
            })

            // Different output verbosity based on environment:
            // Development: More verbose (quietMode = false)
            // Staging: Standard verbosity
            // Production: Minimal verbosity (quietMode = true)

            output, err := options.RunProjectsTest()
            if err != nil {
                t.Fatalf("%s environment deployment failed: %v", env.name, err)
            }

            // Environment-specific validation
            validateEnvironmentDeployment(t, env.name, output)
        })
    }
}

func createEnvironmentConfig(env environmentConfig) []project.EnvironmentPrototype {
    return []project.EnvironmentPrototype{
        {
            Name:        env.name,
            Region:      env.region,
            Description: fmt.Sprintf("%s environment deployment", strings.Title(env.name)),
        },
    }
}

func getTimeoutForEnvironment(envName string) int {
    timeouts := map[string]int{
        "development": 30,
        "staging":     60,
        "production":  90,
    }
    return timeouts[envName]
}

func validateEnvironmentDeployment(t *testing.T, envName string, output map[string]interface{}) {
    switch envName {
    case "development":
        // Validate dev-specific outputs
        if output["dev_debug_endpoint"] == "" {
            t.Error("Development debug endpoint not found")
        }
    case "staging":
        // Validate staging-specific outputs
        if output["staging_monitoring_url"] == "" {
            t.Error("Staging monitoring URL not found")
        }
    case "production":
        // Validate production-specific outputs
        if output["prod_health_check_url"] == "" {
            t.Error("Production health check URL not found")
        }
        if output["prod_backup_schedule"] == "" {
            t.Error("Production backup schedule not configured")
        }
    }
}

type environmentConfig struct {
    name     string
    region   string
    replicas int
    tier     string
}
```

## Scenario 5: Schematics Workspace Testing with Custom Patterns

Testing Terraform modules through IBM Cloud Schematics with workspace-specific logging.

```golang
package test

import (
    "fmt"
    "testing"
    "time"
    "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testschematic"
)

func TestVPCSchematicsDeployment(t *testing.T) {
    t.Parallel()

    quietMode := true
    options := testschematic.TestSchematicOptionsDefault(&testschematic.TestSchematicOptions{
        Testing:      t,
        TerraformDir: "terraform/vpc-module",
        Prefix:       "schematics-vpc",
        Region:       "us-south",
        QuietMode:    &quietMode,
        VerboseOnFailure: true,

        // Schematics-specific configuration
        ResourceGroup:    "schematics-test-rg",
        WorkspaceRegion:  "us-south",

        // Custom variables for VPC deployment
        TerraformVars: map[string]interface{}{
            "vpc_name":           "test-vpc-schematics",
            "resource_group":     "schematics-test-rg",
            "enable_public_gateway": true,
            "subnet_count":       3,
        },
    })

    // Schematics-specific progress tracking:
    // 🔄 Creating workspace
    // 🔄 Uploading template
    // 🔄 Generating plan
    // 🔄 Applying plan
    // ✅ Plan applied successfully

    output, err := options.RunSchematicTest()
    if err != nil {
        t.Fatalf("Schematics VPC deployment failed: %v", err)
    }

    // Validate VPC-specific outputs
    validateVPCDeployment(t, output)

    // Test workspace cleanup
    validateWorkspaceCleanup(t, options)
}

func TestSchematicsUpgradeScenario(t *testing.T) {
    t.Parallel()

    // Test infrastructure upgrade through Schematics
    quietMode := true
    options := testschematic.TestSchematicOptionsDefault(&testschematic.TestSchematicOptions{
        Testing:      t,
        TerraformDir: "terraform/app-infrastructure",
        Prefix:       "upgrade-test",
        QuietMode:    &quietMode,
        VerboseOnFailure: true,

        // Start with v1 configuration
        TerraformVars: map[string]interface{}{
            "app_version":     "1.0.0",
            "instance_type":   "cx2.2x4",
            "min_replicas":    2,
        },
    })

    // Deploy initial version
    output, err := options.RunSchematicTest()
    if err != nil {
        t.Fatalf("Initial deployment failed: %v", err)
    }

    initialVersion := output["app_version"].(string)
    if initialVersion != "1.0.0" {
        t.Fatalf("Expected app version 1.0.0, got %s", initialVersion)
    }

    // Update to v2 configuration
    options.TerraformVars["app_version"] = "2.0.0"
    options.TerraformVars["instance_type"] = "cx2.4x8"  // Upgrade instance
    options.TerraformVars["min_replicas"] = 3           // Scale up

    // Apply upgrade
    upgradeOutput, err := options.RunSchematicTest()
    if err != nil {
        t.Fatalf("Upgrade deployment failed: %v", err)
    }

    upgradedVersion := upgradeOutput["app_version"].(string)
    if upgradedVersion != "2.0.0" {
        t.Fatalf("Expected upgraded version 2.0.0, got %s", upgradedVersion)
    }

    // Validate zero-downtime upgrade
    validateUpgradeMetrics(t, upgradeOutput)
}

func validateVPCDeployment(t *testing.T, output map[string]interface{}) {
    requiredOutputs := []string{"vpc_id", "subnet_ids", "public_gateway_id"}
    for _, outputKey := range requiredOutputs {
        if output[outputKey] == nil || output[outputKey] == "" {
            t.Errorf("Required VPC output '%s' is missing or empty", outputKey)
        }
    }

    // Validate subnet count matches configuration
    if subnets, ok := output["subnet_ids"].([]interface{}); ok {
        if len(subnets) != 3 {
            t.Errorf("Expected 3 subnets, got %d", len(subnets))
        }
    }
}

func validateWorkspaceCleanup(t *testing.T, options *testschematic.TestSchematicOptions) {
    // Verify workspace was properly destroyed
    time.Sleep(5 * time.Second) // Allow cleanup to complete

    // Additional validation would go here
    // This is a placeholder for workspace verification logic
}

func validateUpgradeMetrics(t *testing.T, output map[string]interface{}) {
    // Validate that upgrade completed without downtime
    if downtime, ok := output["upgrade_downtime_seconds"].(float64); ok {
        if downtime > 30 { // Allow max 30 seconds
            t.Errorf("Upgrade downtime too high: %v seconds", downtime)
        }
    }
}
```

## Key Differences Between Scenarios

1. **GitHub Actions CI**: Environment detection, clean parallel output
2. **Local Development**: Immediate verbose feedback, sequential execution
3. **Large Integration Suite**: Performance optimization, coordinated parallel tests
4. **Multi-Environment**: Environment-specific configuration and validation
5. **Schematics Testing**: Workspace lifecycle management, upgrade scenarios

Each scenario demonstrates distinct configuration patterns and testing approaches, not just parameter variations.

See [User Guide](user-guide.md) for detailed configuration options and [Troubleshooting](troubleshooting.md) for common issues in these scenarios.

## Advanced Examples

### Smart Logger with Phase Detection

For complex tests with automatic progress tracking:

```golang
func TestSmartLogging(t *testing.T) {
    t.Parallel()

    // Create base logger with buffering
    baseLogger := common.NewBufferedTestLogger(t.Name(), true)

    // Add smart phase detection for addon testing
    config := common.SmartLoggerConfig{
        PhasePatterns: common.AddonPhasePatterns,
    }
    logger := common.NewSmartLogger(baseLogger, config)

    // These messages trigger automatic progress tracking
    logger.ShortInfo("Getting offering details")
    // Shows: "🔄 Retrieving catalog information"

    logger.ShortInfo("Validating configuration")
    // Shows: "🔄 Validating inputs"

    logger.ShortInfo("Request completed")
    // Shows: "✅ Operation completed"

    // Regular logging continues to work
    logger.ShortDebug("This is buffered debug info")

    if testFailed {
        // Use enhanced error handling methods
        logger.CriticalError("Test failed with critical error")
        // This automatically:
        // 1. Marks test as failed
        // 2. Flushes buffered logs with context
        // 3. Shows prominent red-bordered error message
    }
}
```

### Using Predefined Factory Functions

Simplified creation for common scenarios:

```golang
// Addon Testing
func TestAddonWithSmartLogger(t *testing.T) {
    t.Parallel()

    logger := common.CreateAddonLogger(t.Name(), true) // auto-configured

    logger.ShortInfo("Creating catalog")
    logger.ShortInfo("Importing offering")
    logger.ShortInfo("Building dependency graph")
    // All automatically show progress stages
}

// Project Testing
func TestProjectWithSmartLogger(t *testing.T) {
    t.Parallel()

    logger := common.CreateProjectLogger(t.Name(), true)

    logger.ShortInfo("Configuring Test Stack")
    logger.ShortInfo("Triggering Deploy")
    logger.ShortInfo("Checking Stack Deploy Status")
    // Configured for project-specific patterns
}

// Terraform Helper Testing
func TestTerraformWithSmartLogger(t *testing.T) {
    t.Parallel()

    logger := common.CreateHelperLogger(t.Name(), true)

    logger.ShortInfo("Running Terraform Init")
    logger.ShortInfo("Running Terraform Plan")
    logger.ShortInfo("Running Terraform Apply")
    // Configured for terraform-specific patterns
}
```

## Color and Custom Logging

### Using Colors Effectively

```golang
func TestWithColors(t *testing.T) {
    logger := common.NewTestLogger(t.Name())

    // Using predefined colors
    logger.ShortCustom("Custom cyan message", common.Colors.Cyan)
    logger.ShortCustom("Custom orange message", common.Colors.Orange)

    // Creating colored strings
    coloredMessage := common.ColorizeString(common.Colors.Purple, "This is purple")
    logger.ShortInfo(coloredMessage)

    // Custom logging with level
    logger.Custom("CUSTOM", "Custom level message", common.Colors.Purple)

    // Enhanced error handling examples
    logger.CriticalError("Critical system failure")      // Red-bordered, shows buffer context
    logger.FatalError("Immediate failure - bypasses buffering") // Immediate display
    logger.ErrorWithContext("Error with moderate formatting")   // Yellow-bordered, shows context
}
```

### Progress Methods

Progress methods bypass quiet mode for essential user feedback:

```golang
func TestWithProgressMessages(t *testing.T) {
    t.Parallel()

    logger := common.NewBufferedTestLogger(t.Name(), true) // quiet mode

    // These always show, even in quiet mode
    logger.ProgressStage("Setting up test environment")    // 🔄 Setting up test environment
    logger.ProgressInfo("Found 3 existing resources")      // ℹ️  Found 3 existing resources
    logger.ProgressSuccess("Environment ready")            // ✅ Environment ready

    // These are buffered (quiet mode)
    logger.ShortInfo("Detailed setup information")
    logger.ShortDebug("Debug configuration details")

    // More progress updates
    logger.ProgressStage("Running validation tests")
    logger.ProgressStage("Cleaning up resources")
    logger.ProgressSuccess("Test completed successfully")
}
```

## Integration with Testing Frameworks

### With Testify

```golang
import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/common"
)

func TestWithTestify(t *testing.T) {
    t.Parallel()

    logger := common.NewBufferedTestLogger(t.Name(), true)

    logger.ShortInfo("Starting testify integration test")

    result := performOperation()

    if !assert.NotNil(t, result, "Result should not be nil") {
        logger.ErrorWithContext("Assertion failed: Result should not be nil")
        return
    }

    if !assert.Equal(t, "expected", result.Value, "Values should match") {
        logger.ErrorWithContext("Assertion failed: Values should match")
        return
    }

    logger.ShortInfo("All assertions passed")
}
```

### With Terratest

```golang
import (
    "testing"
    "github.com/gruntwork-io/terratest/modules/terraform"
    "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/common"
)

func TestTerraformIntegration(t *testing.T) {
    t.Parallel()

    logger := common.CreateHelperLogger(t.Name(), true)

    terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
        TerraformDir: "../terraform-modules/example",
        Vars: map[string]interface{}{
            "region": "us-south",
        },
    })

    defer func() {
        logger.ProgressStage("Cleaning up Terraform resources")
        terraform.Destroy(t, terraformOptions)
        logger.ProgressSuccess("Cleanup completed")
    }()

    logger.ProgressStage("Initializing Terraform")
    terraform.InitAndPlan(t, terraformOptions)

    logger.ProgressStage("Applying Terraform configuration")
    terraform.ApplyAndIdempotent(t, terraformOptions)

    logger.ProgressStage("Validating outputs")
    output := terraform.Output(t, terraformOptions, "resource_id")

    if output == "" {
        logger.CriticalError("Expected non-empty resource_id output")
        return
    }

    logger.ProgressSuccess("Terraform test completed successfully")
}
```

## Real-World Scenarios

### Complex Integration Test

```golang
func TestComplexIntegration(t *testing.T) {
    t.Parallel()

    // Setup with smart phase detection
    logger := common.CreateAddonLogger(t.Name(), true)

    // Test setup phase
    logger.ProgressStage("Initializing test environment")

    testConfig := setupTestConfig()
    logger.ShortDebug("Test config: %+v", testConfig)

    // Phase 1: Catalog operations (automatic phase detection)
    logger.ShortInfo("Creating catalog")
    catalog, err := createCatalog(testConfig)
    if err != nil {
        logger.CriticalError(fmt.Sprintf("Failed to create catalog: %v", err))
        return
    }

    // Phase 2: Offering operations
    logger.ShortInfo("Importing offering")
    offering, err := importOffering(catalog, testConfig.OfferingPath)
    if err != nil {
        logger.CriticalError(fmt.Sprintf("Failed to import offering: %v", err))
        return
    }

    // Phase 3: Validation
    logger.ShortInfo("Validating configuration")
    if err := validateOffering(offering); err != nil {
        logger.CriticalError(fmt.Sprintf("Validation failed: %v", err))
        return
    }

    // Cleanup phase
    logger.ProgressStage("Cleaning up test resources")
    defer func() {
        if err := cleanup(catalog); err != nil {
            logger.ShortError("Cleanup failed: %v", err)
        } else {
            logger.ProgressSuccess("Cleanup completed successfully")
        }
    }()

    logger.ProgressSuccess("Integration test completed")
}
```

### Batch Processing with Smart Logger

```golang
func TestBatchProcessing(t *testing.T) {
    t.Parallel()

    logger := common.CreateAddonLogger(t.Name(), true)
    smartLogger := logger.(*common.SmartLogger)

    // Enable batch mode to reduce repetitive progress messages
    smartLogger.EnableBatchMode()
    defer smartLogger.DisableBatchMode()

    items := []string{"item1", "item2", "item3", "item4", "item5"}

    logger.ProgressStage("Processing batch of items")

    for i, item := range items {
        logger.ShortInfo("Getting offering details") // Only shows once in batch mode

        if err := processItem(item); err != nil {
            logger.CriticalError(fmt.Sprintf("Failed to process item %s: %v", item, err))
            return
        }

        logger.ShortInfo("Request completed") // Shows completion for each
        logger.ProgressInfo("Completed %d/%d items", i+1, len(items))
    }

    logger.ProgressSuccess("Batch processing completed")
}
```

### Custom Phase Patterns

```golang
func TestCustomPatterns(t *testing.T) {
    t.Parallel()

    // Define custom patterns for your specific use case
    customPatterns := common.PhasePatterns{
        "Starting data migration":     "🔄 Migrating data",
        "Validating data integrity":   "🔄 Validating integrity",
        "Updating database schema":    "🔄 Updating schema",
        "Migration completed":         "✅ Migration successful",
        "Schema update failed":        "❌ Schema update failed",
    }

    baseLogger := common.NewBufferedTestLogger(t.Name(), true)
    config := common.SmartLoggerConfig{PhasePatterns: customPatterns}
    logger := common.NewSmartLogger(baseLogger, config)

    logger.ShortInfo("Starting data migration")    // Auto-detected
    logger.ShortInfo("Validating data integrity")  // Auto-detected
    logger.ShortInfo("Updating database schema")   // Auto-detected
    logger.ShortInfo("Migration completed")        // Auto-detected
}
```

## Enhanced Error Handling Methods

The logger provides three specialized error handling methods for different scenarios:

### CriticalError - For Severe Test Failures

Use when test failure requires immediate attention and full context:

```golang
func TestCriticalErrorExample(t *testing.T) {
    t.Parallel()

    logger := common.CreateAddonLogger(t.Name(), true)

    logger.ShortInfo("Starting deployment process")
    logger.ShortDebug("Configuring security settings")
    logger.ShortDebug("Setting up network policies")

    if deploymentFailed {
        // Shows buffer context first, then prominent error
        logger.CriticalError("Deployment failed - security policy violation detected")
        // Output:
        // === BUFFERED LOG OUTPUT ===
        // [test] Starting deployment process
        // [test] Configuring security settings
        // [test] Setting up network policies
        // === END BUFFERED LOG OUTPUT ===
        // ================================================================================
        // CRITICAL ERROR: Deployment failed - security policy violation detected
        // ================================================================================
        return
    }
}
```

### FatalError - For Immediate Failures

Use when you need immediate error display without buffering:

```golang
func TestFatalErrorExample(t *testing.T) {
    logger := common.CreateAddonLogger(t.Name(), true)

    logger.ShortInfo("Checking prerequisites")

    if !hasRequiredPermissions() {
        // Immediate display, bypasses all buffering
        logger.FatalError("Insufficient permissions - cannot proceed with test")
        // Output: FATAL ERROR: Insufficient permissions - cannot proceed with test
        t.FailNow()
    }
}
```

### ErrorWithContext - For Moderate Errors

Use for errors that need context but less visual prominence:

```golang
func TestErrorWithContextExample(t *testing.T) {
    t.Parallel()

    logger := common.CreateAddonLogger(t.Name(), true)

    logger.ShortInfo("Processing configurations")
    logger.ShortDebug("Loading config file: app-config.json")
    logger.ShortDebug("Validating configuration schema")

    if configValidationFailed {
        // Shows buffer context with moderate formatting
        logger.ErrorWithContext("Configuration validation failed - using default values")
        // Output:
        // === BUFFERED LOG OUTPUT ===
        // [test] Processing configurations
        // [test] Loading config file: app-config.json
        // [test] Validating configuration schema
        // === END BUFFERED LOG OUTPUT ===
        // ------------------------------------------------------------
        // ERROR: Configuration validation failed - using default values
        // ------------------------------------------------------------
    }
}
```

## Best Practices from Examples

1. **Always use `t.Parallel()` with BufferedTestLogger**
2. **Use enhanced error methods for automatic buffer management and error handling**
3. **Choose error method based on severity**: CriticalError > ErrorWithContext > FatalError
4. **Use progress methods for user-facing status updates**
5. **Leverage predefined factory functions when possible**
6. **Enable batch mode for repetitive operations**
7. **Use smart loggers for complex multi-phase operations**
8. **Combine different logger features as needed**

See [Configuration](configuration.md) for detailed customization options and [Testing Integration](testing-integration.md) for framework-specific guidance.
