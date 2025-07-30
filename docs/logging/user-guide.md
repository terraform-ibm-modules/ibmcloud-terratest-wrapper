# User Guide for Test Writers

## For Test Writers

This guide explains how to configure logging when writing tests with the IBM Cloud Terratest Wrapper. You control logging behavior through test options in the wrapper packages - no need to create or manage loggers directly.

## Core Concept: Test Options Control Logging

All test wrapper packages (`testaddons`, `testhelper`, `testprojects`, `testschematic`) provide logging configuration through their `TestOptions` structs. The framework automatically handles the logging implementation.

```golang
// You configure this
options := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
    QuietMode:         &quietMode,
    VerboseOnFailure:  true,
})

// Framework handles this automatically
// - Creates appropriate logger
// - Buffers output based on QuietMode
// - Shows debug info on failure based on VerboseOnFailure
// - Provides progress tracking
```

## Key Configuration Options

### QuietMode (`*bool`)

Controls when logs are displayed during test execution.

**`true` (Recommended for most tests)**
- Suppresses detailed output during test execution
- Only shows high-level progress stages like "🔄 Deploying infrastructure"
- Reveals all buffered debug logs if test fails
- Perfect for parallel test execution

**`false` (Good for debugging)**
- Shows all output immediately as it happens
- Useful for local development and troubleshooting
- Can be noisy in parallel test environments

### VerboseOnFailure (`bool`)

Controls whether additional debug information is shown when tests fail.

**`true` (Recommended)**
- Shows detailed logs and context when test fails
- Provides diagnostic information for troubleshooting
- No impact on successful test output

**`false`**
- Shows only basic error messages on failure
- Less helpful for debugging test issues

## Configuration Patterns by Scenario

### Standard Parallel Test (Recommended)

For most integration tests running in parallel:

```golang
func TestStandardAddon(t *testing.T) {
    t.Parallel()

    quietMode := true
    options := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
        Testing:           t,
        Prefix:            "test-addon",
        ResourceGroup:     "test-rg",
        QuietMode:         &quietMode,          // Clean parallel execution
        VerboseOnFailure:  true,               // Debug info on failure
    })

    // Clean output during execution:
    // 🔄 Setting up project infrastructure
    // 🔄 Deploying configurations
    // ✅ Test completed successfully

    output, err := options.RunAddonTest()
    if err != nil {
        // Now you'll see all the buffered debug logs
        t.Fatalf("Test failed: %v", err)
    }
}
```

### Local Development/Debugging

When developing tests locally and need immediate feedback:

```golang
func TestWithDebugging(t *testing.T) {
    // Omit t.Parallel() for sequential execution

    quietMode := false
    options := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
        Testing:           t,
        Prefix:            "debug-test",
        ResourceGroup:     "test-rg",
        QuietMode:         &quietMode,          // See all output immediately
        VerboseOnFailure:  true,               // Extra context on failure
    })

    // Verbose output during execution:
    // [test] Getting offering details from catalog
    // [test] Validating configuration inputs
    // [test] Creating project with 3 configurations
    // 🔄 Setting up project infrastructure
    // [test] Deploying configuration: web-app
    // [test] Waiting for deployment status...

    output, err := options.RunAddonTest()
}
```

### CI/CD Environment

Automatically adapt to CI/CD vs local environments:

```golang
func TestCIAdaptive(t *testing.T) {
    t.Parallel()

    // Detect CI environment
    isCI := os.Getenv("CI") == "true" ||
            os.Getenv("GITHUB_ACTIONS") == "true" ||
            os.Getenv("JENKINS_URL") != ""

    options := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
        Testing:           t,
        Prefix:            "ci-test",
        ResourceGroup:     "test-rg",
        QuietMode:         &isCI,               // Quiet in CI, verbose locally
        VerboseOnFailure:  true,               // Always show failure details
    })

    output, err := options.RunAddonTest()
}
```

## Package-Specific Usage

### testaddons Package

For IBM Cloud Projects addon testing:

```golang
import "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testaddons"

quietMode := true
options := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
    Testing:           t,
    Prefix:            "addon-test",
    ResourceGroup:     "project-rg",
    QuietMode:         &quietMode,
    VerboseOnFailure:  true,
    // Addon-specific configuration
    AddonConfig:       addonConfig,
})

output, err := options.RunAddonTest()
```

### testhelper Package

For basic Terraform testing:

```golang
import "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testhelper"

quietMode := true
options := testhelper.TestOptionsDefault(&testhelper.TestOptions{
    Testing:           t,
    TerraformDir:      "examples/basic",
    Prefix:            "helper-test",
    QuietMode:         &quietMode,
    VerboseOnFailure:  true,
})

output, err := options.RunTestConsistency()
```

### testprojects Package

For IBM Cloud Projects stack testing:

```golang
import "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testprojects"

quietMode := true
options := testprojects.TestProjectsOptionsDefault(&testprojects.TestProjectsOptions{
    Testing:           t,
    Prefix:            "project-test",
    QuietMode:         &quietMode,
    VerboseOnFailure:  true,
    // Project-specific configuration
    StackDefinitionPath: "stack-definition.json",
})

output, err := options.RunProjectsTest()
```

### testschematic Package

For IBM Cloud Schematics testing:

```golang
import "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testschematic"

quietMode := true
options := testschematic.TestSchematicOptionsDefault(&testschematic.TestSchematicOptions{
    Testing:           t,
    TerraformDir:      "terraform/",
    Prefix:            "schematic-test",
    QuietMode:         &quietMode,
    VerboseOnFailure:  true,
})

output, err := options.RunSchematicTest()
```

## What Happens Behind the Scenes

When you configure logging options, the framework automatically:

1. **Creates appropriate loggers** based on your settings
2. **Buffers output** when QuietMode is enabled
3. **Detects test phases** and shows progress like "🔄 Deploying infrastructure"
4. **Provides colored output** for easy scanning (green success, yellow warnings, red errors)
5. **Flushes debug logs** on test failure when VerboseOnFailure is true
6. **Handles parallel execution** cleanly with proper output isolation

## Output Examples

### Successful Parallel Test (QuietMode: true)
```
=== RUN   TestStandardAddon
=== PAUSE TestStandardAddon
=== CONT  TestStandardAddon
🔄 Setting up project infrastructure
🔄 Deploying configurations
🔄 Validating deployment
✅ Test completed successfully
--- PASS: TestStandardAddon (45.67s)
```

### Failed Test with Debug Output (VerboseOnFailure: true)
```
=== RUN   TestFailingAddon
🔄 Setting up project infrastructure
🔄 Deploying configurations
=== BUFFERED LOG OUTPUT ===
[addon-test] Getting offering details from catalog
[addon-test] Found offering: web-app-addon v1.2.0
[addon-test] Validating configuration inputs
[addon-test] Creating project with 2 configurations
[addon-test] Deploying configuration: web-app
[addon-test] Error: deployment failed - insufficient permissions
=== END BUFFERED LOG OUTPUT ===
--- FAIL: TestFailingAddon (30.24s)
    test.go:67: Test failed: deployment timeout
```

## Error Handling

The framework provides enhanced error handling methods that automatically manage buffer display and error formatting:

### CriticalError - For Severe Test Failures

```golang
func TestWithCriticalError(t *testing.T) {
    t.Parallel()

    options := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
        Testing:           t,
        QuietMode:         &[]bool{true}[0],
        VerboseOnFailure:  true,
        // ... other options
    })

    output, err := options.RunAddonTest()
    if err != nil {
        // This automatically:
        // 1. Shows all buffered debug logs first
        // 2. Displays prominent red-bordered error
        // 3. Marks test as failed
        options.Logger.CriticalError(fmt.Sprintf("Addon deployment failed: %v", err))
        return
    }
}
```

### ErrorWithContext - For Expected Errors

```golang
func TestWithExpectedError(t *testing.T) {
    t.Parallel()

    options := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
        Testing:           t,
        QuietMode:         &[]bool{true}[0],
        VerboseOnFailure:  true,
        // ... other options
    })

    output, err := options.RunAddonTest()
    if err != nil && strings.Contains(err.Error(), "expected validation failure") {
        // Less prominent formatting for expected errors
        options.Logger.ErrorWithContext("Validation failed as expected - continuing with fallback")
    }
}
```

### FatalError - For Immediate Display

```golang
func TestWithFatalError(t *testing.T) {
    t.Parallel()

    options := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
        Testing:           t,
        QuietMode:         &[]bool{true}[0],
        VerboseOnFailure:  true,
        // ... other options
    })

    if os.Getenv("TF_VAR_ibmcloud_api_key") == "" {
        // Immediate error display, bypasses buffering
        options.Logger.FatalError("Required environment variable TF_VAR_ibmcloud_api_key not set")
        t.FailNow()
    }
}
```

## Best Practices

1. **Use QuietMode for parallel tests** - Keeps output clean and readable
2. **Always enable VerboseOnFailure** - Essential for debugging test failures
3. **Use enhanced error methods** - CriticalError, ErrorWithContext, or FatalError instead of manual error handling
4. **Omit t.Parallel() when debugging** - Easier to follow sequential output
5. **Use environment detection** - Adapt behavior for CI vs local development
6. **Test both quiet and verbose modes** - Ensure your tests work in both scenarios

## Common Gotchas

- **Forgetting the pointer**: `QuietMode` expects `*bool`, not `bool`
- **Not calling t.Parallel()**: Quiet mode is most beneficial with parallel execution
- **Disabling VerboseOnFailure**: Makes debugging failed tests much harder

## Next Steps

- See [Examples](examples.md) for complete test scenarios
- Check [Troubleshooting](troubleshooting.md) for common configuration issues
- Review [Configuration Reference](configuration.md) for advanced options

For framework developers who need to understand or modify the logging implementation, see the [Developer Guide](developer-guide.md).
