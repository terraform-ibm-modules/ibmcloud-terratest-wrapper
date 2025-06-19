# Schematics Testing Configuration

This guide covers all available configuration options for the Schematics testing framework.

## TestSchematicOptions Structure

The `TestSchematicOptions` structure provides comprehensive configuration options for customizing your Schematics tests.

## Basic Configuration

### Required Settings

- **`Testing`** - Required testing.T object from your Go test function
- **`Prefix`** - A unique prefix for all resources created during testing (a random string will be appended)

### File and Path Configuration

- **`TarIncludePatterns`** - List of file patterns to include in the TAR file uploaded to Schematics
  - Defaults to `["*.tf"]` in project root
  - Example: `[]string{"*.tf", "scripts/*.sh", "examples/basic/*.tf"}`

- **`TemplateFolder`** - Directory within the TAR file where Terraform should execute
  - Defaults to `"."`
  - Example: `"examples/basic"`

### Environment Variables

- **`RequiredEnvironmentVars`** - Environment variables required for testing
  - `TF_VAR_ibmcloud_api_key` is required by default
  - Additional variables can be added as needed

## Workspace Configuration

### Location and Region

- **`WorkspaceLocation`** - Region for the Schematics workspace
  - If not set, a random location will be selected
  - Example: `"us-south"`

- **`Region`** - Specific region to use for resources
  - If set, dynamic region selection will be skipped
  - Works with `BestRegionYAMLPath` for dynamic selection

- **`BestRegionYAMLPath`** - Path to YAML file configuring dynamic region selection
  - Enables intelligent region selection based on availability

### Terraform Configuration

- **`TerraformVersion`** - Specific Terraform version to use in the workspace
  - Format: `"terraform_v1.x"`
  - Example: `"terraform_v1.5"`

- **`TerraformVars`** - List of Terraform variables to set in the Schematics workspace
  - Type: `[]TestSchematicTerraformVar`
  - See [Variable Configuration](#variable-configuration) section

### Workspace Settings

- **`Tags`** - List of tags to apply to the workspace
  - Example: `[]string{"test", "automation", "terratest"}`

- **`WorkspaceEnvVars`** - Additional environment variables to set in the workspace
  - Type: `map[string]string`

### API Configuration

- **`SchematicsApiURL`** - Base URL of the Schematics REST API
  - Defaults to appropriate endpoint for the chosen region
  - Override for custom or test environments

## Variable Configuration

The `TerraformVars` field accepts a list of `TestSchematicTerraformVar` objects:

```golang
type TestSchematicTerraformVar struct {
    Name     string      // The name of the Terraform variable
    Value    interface{} // The value to set for the variable
    DataType string      // The Terraform data type
    Secure   bool        // Whether the variable should be hidden in logs
}
```

### Supported Data Types

- `"string"` - String values
- `"bool"` - Boolean values
- `"number"` - Numeric values
- `"list(string)"` - List of strings
- `"list(number)"` - List of numbers
- `"map(any)"` - Map with mixed value types
- `"map(string)"` - Map with string values

### Variable Examples

```golang
options.TerraformVars = []testschematic.TestSchematicTerraformVar{
    // String variable
    {Name: "resource_group", Value: "default", DataType: "string", Secure: false},

    // Secure string (hidden in logs)
    {Name: "api_key", Value: "secret-key", DataType: "string", Secure: true},

    // Boolean variable
    {Name: "enable_feature", Value: true, DataType: "bool", Secure: false},

    // Number variable
    {Name: "instance_count", Value: 3, DataType: "number", Secure: false},

    // List of strings
    {Name: "zones", Value: []string{"us-south-1", "us-south-2"}, DataType: "list(string)", Secure: false},

    // Map variable
    {Name: "tags", Value: map[string]interface{}{
        "environment": "test",
        "project": "terratest",
        "cost_center": 12345,
    }, DataType: "map(any)", Secure: false},
}
```

## Git Repository Access

### Private Repository Configuration

For accessing private Git repositories, configure netrc credentials:

- **`NetrcSettings`** - List of credentials for accessing private Git repositories
  - Use `AddNetrcCredential()` method to add credentials
  - Example: `options.AddNetrcCredential("github.com", "username", "token")`

### Base Repository Settings (for Upgrade Tests)

- **`BaseTerraformRepo`** - The URL of the origin git repository for upgrade tests
  - Usually auto-detected from git configuration
  - Override with environment variable `BASE_TERRAFORM_REPO`

- **`BaseTerraformBranch`** - The branch name of the main origin branch for upgrade tests
  - Usually auto-detected (typically "main" or "master")
  - Override with environment variable `BASE_TERRAFORM_BRANCH`

## Testing Control Options

### Timing Configuration

- **`WaitJobCompleteMinutes`** - Minutes to wait for Schematics jobs to complete
  - Defaults to 120 minutes
  - Increase for long-running deployments

### Cleanup Behavior

- **`DeleteWorkspaceOnFail`** - Whether to delete the workspace if the test fails
  - Defaults to `false` (keeps workspace for debugging)
  - Set to `true` for CI/CD environments

- **`SkipTestTearDown`** - Skip test teardown completely
  - Skips both resource destroy and workspace deletion
  - Useful for debugging or manual inspection

### Logging Configuration

- **`PrintAllSchematicsLogs`** - Whether to print all Schematics job logs
  - Defaults to `false` (only prints logs on failure)
  - Set to `true` for verbose debugging

## Upgrade Testing Configuration

### Upgrade Test Control

- **`CheckApplyResultForUpgrade`** - For upgrade tests, whether to perform a final apply after consistency check
  - Defaults to `false`
  - Set to `true` to validate that the upgrade actually applies successfully

### Consistency Checking

Control which resources to ignore during plan consistency checks:

- **`IgnoreAdds`** - List of resource names to ignore when checking for added resources
  - Example: `[]string{"random_id.suffix"}`

- **`IgnoreUpdates`** - List of resource names to ignore when checking for updated resources
  - Example: `[]string{"ibm_resource_instance.example"}`

- **`IgnoreDestroys`** - List of resource names to ignore when checking for destroyed resources
  - Example: `[]string{"null_resource.temp"}`

## Hook Configuration

The framework provides several hook points for custom code injection:

### Available Hooks

- **`PreApplyHook`** - Executed before the APPLY step
- **`PostApplyHook`** - Executed after successful APPLY
- **`PreDestroyHook`** - Executed before the DESTROY step
- **`PostDestroyHook`** - Executed after successful DESTROY

### Hook Function Signature

```golang
type HookFunction func(options *TestSchematicOptions) error
```

### Hook Examples

```golang
options := testschematic.TestSchematicOptionsDefault(&testschematic.TestSchematicOptions{
    Testing: t,
    Prefix:  "hook-test",

    PreApplyHook: func(options *testschematic.TestSchematicOptions) error {
        // Setup code before apply
        t.Log("Preparing for deployment...")
        return nil
    },

    PostApplyHook: func(options *testschematic.TestSchematicOptions) error {
        // Validation code after apply
        outputs := options.LastTestTerraformOutputs
        if outputs != nil {
            t.Logf("Deployment output: %v", outputs["endpoint"])
        }
        return nil
    },
})
```

## Environment Variable Overrides

Several configuration options can be overridden with environment variables:

- `BASE_TERRAFORM_REPO` - Override base repository URL
- `BASE_TERRAFORM_BRANCH` - Override base branch name
- `DO_NOT_DESTROY_ON_FAILURE=true` - Keep resources after test failure
- `TF_VAR_ibmcloud_api_key` - IBM Cloud API key (required)

## Advanced Configuration Example

```golang
func TestAdvancedConfiguration(t *testing.T) {
    t.Parallel()

    options := testschematic.TestSchematicOptionsDefault(&testschematic.TestSchematicOptions{
        Testing:                    t,
        Prefix:                     "advanced-test",
        BestRegionYAMLPath:        "test-config/regions.yaml",
        TarIncludePatterns:        []string{"*.tf", "modules/**/*.tf", "scripts/*.sh"},
        TemplateFolder:            "examples/complete",
        TerraformVersion:          "terraform_v1.5",
        Tags:                      []string{"terratest", "advanced", "ci"},
        DeleteWorkspaceOnFail:     true,
        PrintAllSchematicsLogs:    true,
        WaitJobCompleteMinutes:    180,
        CheckApplyResultForUpgrade: true,

        // Ignore specific resources in consistency checks
        IgnoreAdds:    []string{"random_id.unique_suffix"},
        IgnoreUpdates: []string{"time_rotating.certificate"},

        // Custom hooks
        PostApplyHook: func(options *testschematic.TestSchematicOptions) error {
            // Custom validation logic
            return validateDeployment(options.LastTestTerraformOutputs)
        },
    })

    // Add private repository access
    options.AddNetrcCredential("github.com", "myuser", os.Getenv("GITHUB_TOKEN"))

    // Configure workspace environment variables
    options.WorkspaceEnvVars = map[string]string{
        "CUSTOM_ENV_VAR": "custom_value",
        "DEBUG_MODE":     "true",
    }

    // Configure comprehensive variables
    options.TerraformVars = []testschematic.TestSchematicTerraformVar{
        {Name: "ibmcloud_api_key", Value: options.RequiredEnvironmentVars["TF_VAR_ibmcloud_api_key"], DataType: "string", Secure: true},
        {Name: "region", Value: options.Region, DataType: "string"},
        {Name: "prefix", Value: options.Prefix, DataType: "string"},
        {Name: "resource_group", Value: "default", DataType: "string"},
        {Name: "enable_monitoring", Value: true, DataType: "bool"},
        {Name: "instance_count", Value: 3, DataType: "number"},
        {Name: "allowed_zones", Value: []string{"us-south-1", "us-south-2"}, DataType: "list(string)"},
        {Name: "resource_tags", Value: map[string]string{
            "environment": "test",
            "project":     "advanced-terratest",
        }, DataType: "map(string)"},
    }

    err := options.RunSchematicTest()
    assert.NoError(t, err)
}
```
