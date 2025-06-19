# Schematics Testing Overview

The `testschematic` package provides a framework for testing IBM Cloud Terraform modules using IBM Cloud Schematics Workspaces. This allows you to test your Terraform code in a fully managed IBM Cloud environment without needing to install and configure Terraform locally.

## How It Works

The general process for testing a Terraform module in Schematics that the framework handles for the user is as follows:

1. **Workspace Creation**: Creates a test workspace in IBM Cloud Schematics
2. **Code Upload**: Creates and uploads a TAR file of your Terraform project to the workspace
3. **Configuration**: Configures the workspace with your test variables
4. **Execution**: Runs PLAN/APPLY/DESTROY steps on the workspace to provision and destroy resources
5. **Consistency Check**: Checks consistency by running an additional PLAN after APPLY and checking for unexpected resource changes
6. **Cleanup**: Deletes the test workspace

## Key Features

### Standard Testing

- Full lifecycle testing (plan, apply, destroy)
- Consistency validation after deployment
- Configurable file inclusion patterns
- Support for custom hooks at each lifecycle stage

### Upgrade Testing

The framework also supports upgrade testing, which allows you to verify that changes in a PR branch do not cause unexpected resource destruction when applied to existing infrastructure.

### Private Repository Support

Built-in support for accessing private Git repositories with netrc credential management.

### Flexible Configuration

Extensive configuration options for workspace settings, Terraform versions, regions, and testing behavior.

## Getting Started

### Prerequisites

1. **IBM Cloud API Key**: Set the `TF_VAR_ibmcloud_api_key` environment variable
2. **Go Module**: Ensure your project is a Go module with terratest dependencies

### Basic Setup

```golang
package test

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testschematic"
)

func TestBasicSchematics(t *testing.T) {
    t.Parallel()

    // Initialize test options with defaults
    options := testschematic.TestSchematicOptionsDefault(&testschematic.TestSchematicOptions{
        Testing:            t,                      // Required: the test object
        Prefix:             "my-test",              // Unique prefix (random string appended)
        TarIncludePatterns: []string{"*.tf", "examples/basic/*.tf"}, // Files to include
        TemplateFolder:     "examples/basic",       // Directory where Terraform executes
    })

    // Configure Terraform variables for the workspace
    options.TerraformVars = []testschematic.TestSchematicTerraformVar{
        {Name: "ibmcloud_api_key", Value: options.RequiredEnvironmentVars["TF_VAR_ibmcloud_api_key"], DataType: "string", Secure: true},
        {Name: "ibm_region", Value: options.Region, DataType: "string"},
        {Name: "prefix", Value: options.Prefix, DataType: "string"},
    }

    // Run the test
    err := options.RunSchematicTest()
    assert.NoError(t, err, "Schematics test should complete without errors")
}
```

### Test Organization

For larger projects, organize your tests with clear naming and separation:

```golang
func TestCompleteExample(t *testing.T) {
    t.Parallel()
    // Test the complete example
}

func TestMinimalExample(t *testing.T) {
    t.Parallel()
    // Test the minimal configuration
}

func TestUpgrade(t *testing.T) {
    t.Parallel()
    // Test upgrade scenarios
}
```

## Next Steps

- **[Examples](examples.md)**: See comprehensive examples for common testing scenarios
- **[Configuration](configuration.md)**: Learn about all available configuration options
- **[Testing Process](testing-process.md)**: Understand the detailed testing lifecycle
- **[Troubleshooting](troubleshooting.md)**: Solutions for common issues
