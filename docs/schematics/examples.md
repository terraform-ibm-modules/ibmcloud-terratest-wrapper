# Schematics Testing Examples

This guide provides comprehensive examples for common Schematics testing scenarios.

## Basic Schematics Test

This example shows how to run a basic test using the `testschematic` package. It sets up the test options, including the required file patterns and variables, and then runs the test.

```golang
package test

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testschematic"
)

func TestRunBasicInSchematic(t *testing.T) {
    t.Parallel()

    options := testschematic.TestSchematicOptionsDefault(&testschematic.TestSchematicOptions{
        Testing:            t,                      // the test object for unit test
        Prefix:             "my-test",              // will have 6 char random string appended
        BestRegionYAMLPath: "location/of/yaml.yml", // YAML file to configure dynamic region selection
        // Supply filters in order to build TAR file to upload to schematics
        TarIncludePatterns: []string{"*.tf", "scripts/*.sh", "examples/basic/*.tf"},
        // Directory within the TAR where Terraform will execute
        TemplateFolder:    "examples/basic",
        // Delete the workspace if the test fails (false keeps it for debugging)
        DeleteWorkspaceOnFail: false,
    })

    // Set up the schematic workspace variables
    options.TerraformVars = []testschematic.TestSchematicTerraformVar{
        {Name: "ibmcloud_api_key", Value: options.RequiredEnvironmentVars["TF_VAR_ibmcloud_api_key"], DataType: "string", Secure: true},
        {Name: "ibm_region", Value: options.Region, DataType: "string"},
        {Name: "resource_group_name", Value: "default", DataType: "string"},
        {Name: "prefix", Value: options.Prefix, DataType: "string"},
        {Name: "do_something", Value: true, DataType: "bool"},
        {Name: "tags", Value: []string{"test", "schematic"}, DataType: "list(string)"},
    }

    // Run the test
    err := options.RunSchematicTest()
    assert.NoError(t, err, "Schematics test should complete without errors")
}
```

## Custom Hooks Example

This example demonstrates how to add custom hooks to perform actions before or after resource deployment and destruction:

```golang
func TestRunSchematicWithHooks(t *testing.T) {
    t.Parallel()

    options := testschematic.TestSchematicOptionsDefault(&testschematic.TestSchematicOptions{
        Testing:             t,
        Prefix:              "hook-test",
        TarIncludePatterns:  []string{"*.tf", "modules/**/*.tf"},
        TemplateFolder:      "examples/complex",

        // Define custom hooks
        PreApplyHook: func(options *testschematic.TestSchematicOptions) error {
            // Execute code before the APPLY step
            t.Log("Executing pre-apply setup...")
            return nil
        },

        PostApplyHook: func(options *testschematic.TestSchematicOptions) error {
            // Execute code after successful APPLY
            t.Log("Validating deployed resources...")

            // Access terraform outputs if needed
            outputs := options.LastTestTerraformOutputs
            if outputs != nil {
                t.Logf("Found output: %v", outputs["example_output"])
            }
            return nil
        },

        PreDestroyHook: func(options *testschematic.TestSchematicOptions) error {
            // Execute code before the DESTROY step
            t.Log("Preparing for resource teardown...")
            return nil
        },

        PostDestroyHook: func(options *testschematic.TestSchematicOptions) error {
            // Execute code after successful DESTROY
            t.Log("Performing post-destruction cleanup...")
            return nil
        },
    })

    options.TerraformVars = []testschematic.TestSchematicTerraformVar{
        {Name: "ibmcloud_api_key", Value: options.RequiredEnvironmentVars["TF_VAR_ibmcloud_api_key"], DataType: "string", Secure: true},
        {Name: "ibm_region", Value: options.Region, DataType: "string"},
    }

    err := options.RunSchematicTest()
    assert.NoError(t, err)
}
```

## Upgrade Testing Example

This example shows how to run an upgrade test to verify that changes in a PR branch don't cause unexpected resource destruction:

```golang
func TestRunSchematicUpgrade(t *testing.T) {
    t.Parallel()

    options := testschematic.TestSchematicOptionsDefault(&testschematic.TestSchematicOptions{
        Testing:              t,
        Prefix:               "upgrade-test",
        TarIncludePatterns:   []string{"*.tf", "modules/**/*.tf"},
        TemplateFolder:       "examples/basic",
        // If true, will run 'apply' after upgrade plan consistency check
        CheckApplyResultForUpgrade: true,
    })

    options.TerraformVars = []testschematic.TestSchematicTerraformVar{
        {Name: "ibmcloud_api_key", Value: options.RequiredEnvironmentVars["TF_VAR_ibmcloud_api_key"], DataType: "string", Secure: true},
        {Name: "ibm_region", Value: options.Region, DataType: "string"},
        {Name: "prefix", Value: options.Prefix, DataType: "string"},
    }

    err := options.RunSchematicUpgradeTest()
    if !options.UpgradeTestSkipped {
        assert.NoError(t, err, "Upgrade test should complete without errors")
    }
}
```

## Private Git Repository Access

If your Terraform code references modules in private Git repositories, you can provide netrc credentials for authentication:

```golang
func TestWithPrivateRepoAccess(t *testing.T) {
    t.Parallel()

    options := testschematic.TestSchematicOptionsDefault(&testschematic.TestSchematicOptions{
        Testing:            t,
        Prefix:             "private-repo-test",
        TarIncludePatterns: []string{"*.tf"},
    })

    // Add credentials for private Git repos
    options.AddNetrcCredential("github.com", "github-username", options.RequiredEnvironmentVars["GITHUB_TOKEN"])
    options.AddNetrcCredential("bitbucket.com", "bit-username", options.RequiredEnvironmentVars["BITBUCKET_TOKEN"])

    options.TerraformVars = []testschematic.TestSchematicTerraformVar{
        {Name: "ibmcloud_api_key", Value: options.RequiredEnvironmentVars["TF_VAR_ibmcloud_api_key"], DataType: "string", Secure: true},
        {Name: "ibm_region", Value: options.Region, DataType: "string"},
    }

    err := options.RunSchematicTest()
    assert.NoError(t, err)
}
```

## Working with Different Variable Types

Example of setting different variable types in your Schematics workspace:

```golang
func TestVariableTypes(t *testing.T) {
    t.Parallel()

    options := testschematic.TestSchematicOptionsDefault(&testschematic.TestSchematicOptions{
        Testing:            t,
        Prefix:             "var-test",
        TarIncludePatterns: []string{"*.tf"},
    })

    options.TerraformVars = []testschematic.TestSchematicTerraformVar{
        // String variable
        {Name: "string_var", Value: "hello", DataType: "string", Secure: false},

        // Boolean variable
        {Name: "bool_var", Value: true, DataType: "bool", Secure: false},

        // Number variable
        {Name: "number_var", Value: 42, DataType: "number", Secure: false},

        // List variable
        {Name: "list_var", Value: []string{"item1", "item2"}, DataType: "list(string)", Secure: false},

        // Map variable
        {Name: "map_var", Value: map[string]interface{}{
            "key1": "value1",
            "key2": 42,
            "key3": true,
        }, DataType: "map(any)", Secure: false},

        // Secure variable (hidden in logs)
        {Name: "api_key", Value: "sensitive-value", DataType: "string", Secure: true},
    }

    err := options.RunSchematicTest()
    assert.NoError(t, err)
}
```

## Multiple Test Examples in One Package

```golang
func TestCompleteExample(t *testing.T) {
    t.Parallel()

    options := testschematic.TestSchematicOptionsDefault(&testschematic.TestSchematicOptions{
        Testing:            t,
        Prefix:             "complete",
        TarIncludePatterns: []string{"*.tf", "examples/complete/*.tf"},
        TemplateFolder:     "examples/complete",
    })

    // Configure all required variables for complete example
    options.TerraformVars = []testschematic.TestSchematicTerraformVar{
        {Name: "ibmcloud_api_key", Value: options.RequiredEnvironmentVars["TF_VAR_ibmcloud_api_key"], DataType: "string", Secure: true},
        {Name: "ibm_region", Value: options.Region, DataType: "string"},
        {Name: "prefix", Value: options.Prefix, DataType: "string"},
        {Name: "resource_group_name", Value: "default", DataType: "string"},
    }

    err := options.RunSchematicTest()
    assert.NoError(t, err)
}

func TestMinimalExample(t *testing.T) {
    t.Parallel()

    options := testschematic.TestSchematicOptionsDefault(&testschematic.TestSchematicOptions{
        Testing:            t,
        Prefix:             "minimal",
        TarIncludePatterns: []string{"*.tf", "examples/minimal/*.tf"},
        TemplateFolder:     "examples/minimal",
    })

    // Minimal configuration with only required variables
    options.TerraformVars = []testschematic.TestSchematicTerraformVar{
        {Name: "ibmcloud_api_key", Value: options.RequiredEnvironmentVars["TF_VAR_ibmcloud_api_key"], DataType: "string", Secure: true},
        {Name: "prefix", Value: options.Prefix, DataType: "string"},
    }

    err := options.RunSchematicTest()
    assert.NoError(t, err)
}
```

## Test Organization Best Practices

### Separate Test Files

For larger projects, organize tests into separate files:

```text
tests/
├── basic_test.go          # Basic functionality tests
├── complete_test.go       # Complete example tests
├── upgrade_test.go        # Upgrade scenario tests
└── integration_test.go    # Integration tests
```

### Common Test Helper

Create a helper function for common test setup:

```golang
func setupBasicTest(t *testing.T, prefix string) *testschematic.TestSchematicOptions {
    return testschematic.TestSchematicOptionsDefault(&testschematic.TestSchematicOptions{
        Testing:               t,
        Prefix:                prefix,
        TarIncludePatterns:    []string{"*.tf", "examples/basic/*.tf"},
        TemplateFolder:        "examples/basic",
        DeleteWorkspaceOnFail: false,
    })
}

func TestBasicScenario1(t *testing.T) {
    t.Parallel()

    options := setupBasicTest(t, "scenario1")
    // Add scenario-specific configuration

    err := options.RunSchematicTest()
    assert.NoError(t, err)
}
```

### Recursively collects file paths

This function recursively collects file paths from a given root directory, applying include file type filters and exclude directory rules. It returns a list of tar include patterns for use in packaging or deployment:

```golang
excludeDirs := [
    "tests/important",
    "tests/do-not-delete",
]
includeFiletypes = [".py", ".txt", ".json"]

tarIncludePatterns, recurseErr := testhelper.GetTarIncludeDirsWithDefaults(".", excludeDirs, includeFiletypes)
```

Example running schematics test with `GetTarIncludeDirsWithDefaults`

```golang
func TestRunRegionalFullyConfigurableSchematics(t *testing.T) {
	t.Parallel()

	tarIncludePatterns, recurseErr := testhelper.GetTarIncludeDirsWithDefaults("..", excludeDirs, includeFiletypes)

	// if error producing tar patterns (very unexpected) fail test immediately
	require.NoError(t, recurseErr, "Schematic Test had unexpected error traversing directory tree")

	options := testschematic.TestSchematicOptionsDefault(&testschematic.TestSchematicOptions{
		Testing:                t,
		Prefix:                 "example",
		Region:                 region,
		TarIncludePatterns:     tarIncludePatterns,
		ResourceGroup:          resourceGroup,
		TemplateFolder:         RegionalfullyConfigurableDir,
		Tags:                   []string{"ex-1"},
		DeleteWorkspaceOnFail:  false,
		WaitJobCompleteMinutes: 80,
		TerraformVersion:       terraformVersion,
	})

	err := options.RunSchematicTest()
	assert.Nil(t, err, "This should not have errored")
}
```
