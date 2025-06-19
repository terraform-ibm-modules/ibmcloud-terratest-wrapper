# IBM Cloud Schematics Testing

This section covers testing IBM Cloud Terraform modules using IBM Cloud Schematics Workspaces. The `testschematic` package provides a comprehensive framework for testing your Terraform code in a fully managed IBM Cloud environment without needing to install and configure Terraform locally.

## Documentation Structure

### Getting Started

- [Overview](schematics/overview.md) - Framework introduction and quick start
- [Examples](schematics/examples.md) - Comprehensive examples for common scenarios

### Guides

- [Configuration](schematics/configuration.md) - Complete configuration options and settings
- [Testing Process](schematics/testing-process.md) - Detailed testing lifecycle explanation
- [Troubleshooting](schematics/troubleshooting.md) - Common issues and solutions

## Quick Start

The framework handles the complete testing lifecycle:

1. Creates a test workspace in IBM Cloud Schematics
2. Creates and uploads a TAR file of your Terraform project to the workspace
3. Configures the workspace with your test variables
4. Runs PLAN/APPLY/DESTROY steps on the workspace to provision and destroy resources
5. Checks consistency by running an additional PLAN after APPLY
6. Deletes the test workspace

### Basic Example

```golang
package test

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testschematic"
)

func TestBasicSchematics(t *testing.T) {
    t.Parallel()

    options := testschematic.TestSchematicOptionsDefault(&testschematic.TestSchematicOptions{
        Testing:            t,
        Prefix:             "my-test",
        TarIncludePatterns: []string{"*.tf", "examples/basic/*.tf"},
        TemplateFolder:     "examples/basic",
    })

    options.TerraformVars = []testschematic.TestSchematicTerraformVar{
        {Name: "ibmcloud_api_key", Value: options.RequiredEnvironmentVars["TF_VAR_ibmcloud_api_key"], DataType: "string", Secure: true},
        {Name: "ibm_region", Value: options.Region, DataType: "string"},
        {Name: "prefix", Value: options.Prefix, DataType: "string"},
    }

    err := options.RunSchematicTest()
    assert.NoError(t, err, "Schematics test should complete without errors")
}
```

For detailed examples and advanced usage, see the [Examples](examples.md) guide.
