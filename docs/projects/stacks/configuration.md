# Stack Testing Configuration

This guide covers all configuration options available in the stack testing framework.

## Basic Configuration

### Required Options

```golang
options := testprojects.TestProjectOptionsDefault(&testprojects.TestProjectsOptions{
    Testing:                t,                        // Required: testing.T object
    Prefix:                 "my-test",               // Required: unique prefix for resources
    StackConfigurationPath: "stack_definition.json", // Required: path to stack definition
    StackCatalogJsonPath:   "ibm_catalog.json",     // Required: path to catalog file
})
```

### Project Configuration

```golang
options.ProjectLocation = "us-south"                    // Default: random region selection
options.ProjectDestroyOnDelete = core.BoolPtr(true)     // Default: true
options.ProjectMonitoringEnabled = core.BoolPtr(true)   // Default: true
```

## Stack Configuration

### Stack Definition Path

```golang
options.StackConfigurationPath = "path/to/stack_definition.json"
```

The stack definition file should contain the configuration for your stack members.

### Catalog Configuration

```golang
options.StackCatalogJsonPath = "path/to/ibm_catalog.json"
```

The catalog JSON file contains the offering information for your stack.

## Input Configuration

### Stack-Level Inputs

Apply inputs to the entire stack:

```golang
options.StackInputs = map[string]interface{}{
    "resource_group_name": "my-resource-group",
    "ibmcloud_api_key":    os.Getenv("TF_VAR_ibmcloud_api_key"),
    "region":              "us-south",
}
```

### Member-Specific Inputs

Configure inputs for individual stack members:

```golang
options.StackMemberInputs = map[string]map[string]interface{}{
    "database": {
        "prefix":   fmt.Sprintf("db-%s", options.Prefix),
        "plan":     "standard",
        "region":   "us-south",
    },
    "compute": {
        "prefix":      fmt.Sprintf("comp-%s", options.Prefix),
        "region":      "us-south",
        "machine_type": "bx2.2x8",
    },
    "networking": {
        "prefix": fmt.Sprintf("net-%s", options.Prefix),
        "zones":  []string{"us-south-1", "us-south-2"},
    },
}
```

## Testing Options

### Timeout Configuration

```golang
options.DeployTimeoutMinutes = 120  // 2 hours instead of default 6 hours
```

### Skip Options

```golang
// Skip undeploy operation
options.SkipUndeploy = true

// Skip project deletion
options.SkipProjectDelete = true

// Skip entire teardown process
options.SkipTestTearDown = true
```

## Hook Configuration

### Available Hooks

```golang
// Pre-deployment setup
options.PreDeployHook = func(options *testprojects.TestProjectsOptions) error {
    // Custom setup logic
    return nil
}

// Post-deployment validation
options.PostDeployHook = func(options *testprojects.TestProjectsOptions) error {
    // Custom validation logic
    return nil
}

// Pre-undeploy preparation
options.PreUndeployHook = func(options *testprojects.TestProjectsOptions) error {
    // Data backup, final state capture
    return nil
}

// Post-undeploy cleanup
options.PostUndeployHook = func(options *testprojects.TestProjectsOptions) error {
    // Cleanup verification, additional cleanup
    return nil
}
```

### Hook Best Practices

- **Error Handling**: Return errors to fail the test
- **Logging**: Use the testing object for consistent output
- **State Access**: Access project and stack details via options
- **Resource Validation**: Validate resources in post-deploy hooks

## Advanced Configuration

### Environment Variables

The framework requires the following environment variable:

```bash
export TF_VAR_ibmcloud_api_key="your-api-key"
```

### Custom Project Settings

```golang
options := testprojects.TestProjectOptionsDefault(&testprojects.TestProjectsOptions{
    Testing:                  t,
    Prefix:                   "custom-project",
    StackConfigurationPath:   "stack_definition.json",
    StackCatalogJsonPath:     "ibm_catalog.json",
    ProjectLocation:          "eu-gb",
    DeployTimeoutMinutes:     180,
    ProjectDestroyOnDelete:   core.BoolPtr(true),
    ProjectMonitoringEnabled: core.BoolPtr(false),
})
```

## Configuration Examples

### Minimal Configuration

```golang
func TestMinimalStack(t *testing.T) {
    options := testprojects.TestProjectOptionsDefault(&testprojects.TestProjectsOptions{
        Testing:                t,
        Prefix:                 "minimal",
        StackConfigurationPath: "stack_definition.json",
        StackCatalogJsonPath:   "ibm_catalog.json",
    })

    err := options.RunProjectsTest()
    assert.NoError(t, err)
}
```

### Comprehensive Configuration

```golang
func TestComprehensiveStack(t *testing.T) {
    options := testprojects.TestProjectOptionsDefault(&testprojects.TestProjectsOptions{
        Testing:                  t,
        Prefix:                   "comprehensive",
        StackConfigurationPath:   "stacks/full_stack.json",
        StackCatalogJsonPath:     "catalogs/production_catalog.json",
        ProjectLocation:          "us-east",
        DeployTimeoutMinutes:     240,
        ProjectDestroyOnDelete:   core.BoolPtr(true),
        ProjectMonitoringEnabled: core.BoolPtr(true),
    })

    options.StackInputs = map[string]interface{}{
        "resource_group_name": "production-rg",
        "ibmcloud_api_key":    os.Getenv("TF_VAR_ibmcloud_api_key"),
        "environment":         "production",
    }

    options.StackMemberInputs = map[string]map[string]interface{}{
        "infrastructure": {
            "prefix": options.Prefix,
            "region": "us-east",
            "zones":  []string{"us-east-1", "us-east-2", "us-east-3"},
        },
        "database": {
            "prefix":       options.Prefix,
            "plan":         "enterprise",
            "backup_policy": "daily",
        },
        "application": {
            "prefix":     options.Prefix,
            "instances":  3,
            "scaling":    "auto",
        },
    }

    options.PostDeployHook = func(opts *testprojects.TestProjectsOptions) error {
        // Validate all components are running
        return validateFullStack(opts)
    }

    err := options.RunProjectsTest()
    assert.NoError(t, err)
}

func validateFullStack(options *testprojects.TestProjectsOptions) error {
    // Add comprehensive validation logic
    return nil
}
```
