# IBM Cloud Projects Testing

This section covers testing frameworks for IBM Cloud Projects, including both stack testing and addon testing.

## Testing Frameworks

### Addon Testing

The `testaddons` package provides a comprehensive framework for testing IBM Cloud add-ons in IBM Cloud Projects.

**Getting Started:**

- [Overview](addons/overview.md) - Framework introduction and quick start
- [Examples](addons/examples.md) - Comprehensive examples for common scenarios

**Guides:**

- [Configuration](addons/configuration.md) - Complete configuration options and settings
- [Parallel Testing](addons/parallel-testing.md) - Matrix testing and parallel execution patterns
- [Validation & Hooks](addons/validation-hooks.md) - Built-in validations and custom hook points
- [Testing Process](addons/testing-process.md) - Detailed testing lifecycle explanation
- [Troubleshooting](addons/troubleshooting.md) - Common issues and solutions

### Stack Testing

The `testprojects` package provides a framework for testing IBM Cloud Projects stacks.

**Getting Started:**

- [Overview](stacks/overview.md) - Framework introduction and quick start
- [Examples](stacks/examples.md) - Comprehensive examples for common scenarios

**Guides:**

- [Configuration](stacks/configuration.md) - Complete configuration options and settings
- [Testing Process](stacks/testing-process.md) - Detailed testing lifecycle explanation
- [Troubleshooting](stacks/troubleshooting.md) - Common issues and solutions

## Quick Start Examples

### Basic Addon Test

```golang
func TestBasicAddon(t *testing.T) {
    t.Parallel()

    options := testaddons.TestAddonsOptionsDefault(&testaddons.TestAddonOptions{
        Testing:       t,
        Prefix:        "test-addon",
        ResourceGroup: "my-project-rg",
    })

    options.AddonConfig = cloudinfo.NewAddonConfigTerraform(
        options.Prefix,
        "my-addon",
        "standard",
        map[string]interface{}{
            "prefix": options.Prefix,
            "region": "us-south",
        },
    )

    err := options.RunAddonTest()
    assert.NoError(t, err, "Addon Test had an unexpected error")
}
```

### Basic Stack Test

```golang
func TestBasicStack(t *testing.T) {
    t.Parallel()

    options := testprojects.TestProjectOptionsDefault(&testprojects.TestProjectsOptions{
        Testing:                t,
        Prefix:                 "test-stack",
        StackConfigurationPath: "stack_definition.json",
        StackCatalogJsonPath:   "ibm_catalog.json",
    })

    options.StackInputs = map[string]interface{}{
        "resource_group_name": "default",
        "ibmcloud_api_key":    os.Getenv("TF_VAR_ibmcloud_api_key"),
    }

    err := options.RunProjectsTest()
    assert.NoError(t, err, "Stack deployment should succeed")
}
```

## Framework Comparison

| Feature | Addon Testing | Stack Testing |
|---------|---------------|---------------|
| **Use Case** | Single addon/module testing | Multi-component stack testing |
| **Configuration** | Single addon configuration | Stack definition with multiple members |
| **Dependencies** | Automatic discovery and management | Defined in stack configuration |
| **Parallel Testing** | Built-in matrix testing support | Manual test case organization |
| **Validation** | Reference, dependency, local change validation | Stack-level validation |
| **Hooks** | Pre/post deploy and undeploy hooks | Pre/post deploy and undeploy hooks |

## Common Patterns

Both frameworks support common testing patterns:

- **Parallel Testing**: Run multiple test configurations simultaneously
- **Hook System**: Inject custom code at key points in the testing lifecycle
- **Resource Management**: Automatic cleanup and resource naming
- **Environment Validation**: Check required environment variables and prerequisites
- **Comprehensive Logging**: Detailed logging throughout the testing process
