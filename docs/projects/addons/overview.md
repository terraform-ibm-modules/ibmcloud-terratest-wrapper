# Addon Testing Framework

The `testaddons` package provides a comprehensive framework for testing IBM Cloud add-ons in IBM Cloud Projects. This framework automates the complete lifecycle of addon testing, from catalog setup to deployment validation and cleanup.

## Quick Start

For most addon testing scenarios, you'll want to use the standard Terraform addon approach:

```golang
package test

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/cloudinfo"
    "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testaddons"
)

func TestBasicAddon(t *testing.T) {
    t.Parallel()

    options := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
        Testing:       t,
        Prefix:        "test-addon",
        ResourceGroup: "my-project-rg",
    })

    options.AddonConfig = cloudinfo.NewAddonConfigTerraform(
        options.Prefix,        // prefix for unique resource naming
        "my-addon",           // offering name
        "standard",           // offering flavor
        map[string]interface{}{ // inputs
            "prefix": options.Prefix,
            "region": "us-south",
        },
    )

    err := options.RunAddonTest()
    assert.NoError(t, err, "Addon Test had an unexpected error")
}
```

## Documentation Structure

The addon testing documentation is organized into the following focused guides:

- **[Testing Process Overview](testing-process.md)** - Detailed explanation of the automated testing lifecycle
- **[Configuration Guide](configuration.md)** - Complete configuration options and advanced settings
- **[Parallel Testing Guide](parallel-testing.md)** - Matrix testing and parallel execution patterns
- **[Dependency Permutation Testing](dependency-permutation-testing.md)** - Automated testing of all dependency combinations
- **[Validation and Hooks](validation-hooks.md)** - Built-in validations and custom hook points
- **[Examples](examples.md)** - Comprehensive examples for common scenarios
- **[Troubleshooting](troubleshooting.md)** - Common issues and solutions

## Key Features

- **Automated Lifecycle Management**: Handles catalog creation, offering import, project setup, deployment, and cleanup
- **Built-in Validations**: Reference validation, dependency validation, and local change checks
- **Parallel Testing Support**: Run multiple test configurations simultaneously with matrix testing
- **Dependency Permutation Testing**: Automatically test all possible dependency combinations
- **Flexible Hooks**: Inject custom code at key points in the testing process
- **Comprehensive Logging**: Detailed logging throughout the testing process
- **Dependency Management**: Automatic dependency discovery and validation

## Core Concepts

### Test Options

The `TestAddonOptions` structure is the primary configuration object that controls all aspects of your addon test.

### Addon Configuration

The `AddonConfig` structure defines what addon to test, which flavor to use, and what inputs to provide.

### Dependencies

The framework automatically handles addon dependencies, deploying required dependencies and validating their configuration. Direct dependencies are determined from the catalog version metadata (`SolutionInfo.Dependencies`), and transitive-only relationships are pruned when building expected trees.

### Hooks

Hook functions allow you to inject custom code at specific points in the testing lifecycle for validation, configuration, or cleanup.

## Next Steps

- Start with the [Examples Guide](examples.md) to see common patterns
- Review the [Configuration Guide](configuration.md) for detailed options
- Check out [Parallel Testing Guide](parallel-testing.md) for matrix testing approaches
- Learn about [Dependency Permutation Testing](dependency-permutation-testing.md) for automated dependency validation
- See [Validation and Hooks](validation-hooks.md) for advanced customization
