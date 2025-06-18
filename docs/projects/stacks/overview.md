# Stack Testing Framework

The `testprojects` package provides a comprehensive framework for testing IBM Cloud Projects stacks. This framework automates the complete lifecycle of stack testing, from project creation to deployment validation and cleanup.

## Quick Start

For most stack testing scenarios, you'll want to use the standard approach:

```golang
package test

import (
    "os"
    "testing"
    "github.com/IBM/go-sdk-core/v5/core"
    "github.com/stretchr/testify/assert"
    "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testprojects"
)

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

## Documentation Structure

The stack testing documentation is organized into the following focused guides:

- **[Testing Process Overview](testing-process.md)** - Detailed explanation of the automated testing lifecycle
- **[Configuration Guide](configuration.md)** - Complete configuration options and advanced settings
- **[Examples](examples.md)** - Comprehensive examples for common scenarios
- **[Troubleshooting](troubleshooting.md)** - Common issues and solutions

## Key Features

- **Automated Lifecycle Management**: Handles project creation, stack configuration, deployment, and cleanup
- **Built-in Validations**: Environment validation, file validation, and configuration checks
- **Flexible Hooks**: Inject custom code at key points in the testing process
- **Comprehensive Logging**: Detailed logging throughout the testing process
- **Authorization Support**: Support for API keys and trusted profiles
- **Timeout Management**: Configurable timeouts for deployment operations

## Core Concepts

### Test Options

The `TestProjectsOptions` structure is the primary configuration object that controls all aspects of your stack test.

### Stack Configuration

Stack tests require two key files:
- **Stack Definition** (`stack_definition.json`): Defines the stack members and their relationships
- **Catalog File** (`ibm_catalog.json`): Contains offering information for stack components

### Input Management

The framework supports both stack-level inputs and member-specific inputs with clear precedence rules for configuration inheritance.

### Hooks

Hook functions allow you to inject custom code at specific points in the testing lifecycle for validation, configuration, or cleanup.

## Next Steps

- Start with the [Examples Guide](examples.md) to see common patterns
- Review the [Configuration Guide](configuration.md) for detailed options
- Check the [Testing Process](testing-process.md) for lifecycle details
- See [Troubleshooting](troubleshooting.md) for common issues and solutions
