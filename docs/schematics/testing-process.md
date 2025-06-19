# Schematics Testing Process

This guide explains the detailed testing lifecycle and processes used by the Schematics testing framework.

## Standard Testing Process

The `RunSchematicTest()` method follows a comprehensive testing lifecycle:

### 1. Workspace Creation

- Creates a new test workspace in IBM Cloud Schematics
- Names the workspace using your prefix + random suffix
- Applies any configured tags and settings
- Sets the workspace location/region

### 2. Code Upload

- Creates a TAR file containing your Terraform code based on `TarIncludePatterns`
- Uploads the TAR file to the Schematics workspace
- Configures the workspace to use the specified `TemplateFolder`

### 3. Workspace Configuration

- Sets all Terraform variables from `TerraformVars`
- Configures workspace environment variables
- Sets the Terraform version if specified
- Applies any netrc credentials for private repository access

### 4. Planning Phase

- Executes `terraform plan` in the Schematics workspace
- Validates that the plan succeeds without errors
- Checks for any unexpected resources or issues

### 5. Apply Phase

- Executes the `PreApplyHook` if configured
- Runs `terraform apply` to create resources
- Waits for the apply job to complete (up to `WaitJobCompleteMinutes`)
- Executes the `PostApplyHook` if configured
- Captures Terraform outputs in `LastTestTerraformOutputs`

### 6. Consistency Check

- Runs another `terraform plan` after apply
- Verifies that no unexpected changes are detected
- Ensures infrastructure state matches the configuration
- Applies ignore rules for `IgnoreAdds`, `IgnoreUpdates`, `IgnoreDestroys`

### 7. Destruction Phase

- Executes the `PreDestroyHook` if configured
- Runs `terraform destroy` to clean up resources
- Waits for the destroy job to complete
- Executes the `PostDestroyHook` if configured

### 8. Cleanup

- Deletes the Schematics workspace (unless `DeleteWorkspaceOnFail` is false and test failed)
- Removes temporary files

## Upgrade Testing Process

The `RunSchematicUpgradeTest()` method performs upgrade validation:

### 1. Base Branch Setup

- Determines the base repository and branch (usually main/master)
- Creates a workspace configured for the base branch
- Uploads and configures the base branch code

### 2. Base Deployment

- Runs plan and apply on the base branch
- Establishes baseline infrastructure state
- Captures the current state file

### 3. PR Branch Switch

- Updates the workspace to use the current PR branch
- Uploads the updated code (your changes)
- Maintains the existing state file

### 4. Upgrade Analysis

- Runs `terraform plan` with the new code against existing state
- Analyzes the plan for unexpected resource destruction
- Checks for breaking changes that weren't expected

### 5. Optional Apply Validation

- If `CheckApplyResultForUpgrade` is true, applies the changes
- Verifies that the upgrade actually succeeds
- Confirms no unexpected side effects

### 6. Cleanup

- Destroys all resources
- Deletes the workspace

### Upgrade Test Skipping

Upgrade tests are automatically skipped when:

- Commit messages contain "BREAKING CHANGE"
- Commit messages contain "SKIP UPGRADE TEST"
- Override with "UNSKIP UPGRADE TEST" in commit message

## Hook Execution Points

The framework provides several points where you can inject custom code:

### PreApplyHook

**When**: Before `terraform apply` execution
**Purpose**: Setup, preparation, or pre-deployment validation
**Example Use Cases**:

- Configure external dependencies
- Set up monitoring or logging
- Validate prerequisites
- Perform security checks

```golang
PreApplyHook: func(options *testschematic.TestSchematicOptions) error {
    // Setup monitoring before deployment
    return setupMonitoring(options.Prefix)
}
```

### PostApplyHook

**When**: After successful `terraform apply`
**Purpose**: Validation, testing, or post-deployment configuration
**Example Use Cases**:

- Validate deployed resources
- Run integration tests
- Configure post-deployment settings
- Capture metrics or logs

```golang
PostApplyHook: func(options *testschematic.TestSchematicOptions) error {
    outputs := options.LastTestTerraformOutputs
    endpoint := outputs["api_endpoint"].(string)
    return validateEndpoint(endpoint)
}
```

### PreDestroyHook

**When**: Before `terraform destroy` execution
**Purpose**: Cleanup preparation or data preservation
**Example Use Cases**:

- Backup important data
- Capture final metrics
- Notify external systems
- Clean up external resources

```golang
PreDestroyHook: func(options *testschematic.TestSchematicOptions) error {
    // Backup data before destruction
    return backupImportantData(options.LastTestTerraformOutputs)
}
```

### PostDestroyHook

**When**: After successful `terraform destroy`
**Purpose**: Final cleanup or validation
**Example Use Cases**:

- Verify complete cleanup
- Remove external configurations
- Generate test reports
- Clean up temporary resources

```golang
PostDestroyHook: func(options *testschematic.TestSchematicOptions) error {
    // Verify all resources are cleaned up
    return verifyCleanup(options.Prefix)
}
```

## Error Handling and Recovery

### Workspace Preservation

When `DeleteWorkspaceOnFail` is set to `false` (default):

- Failed test workspaces are preserved for debugging
- You can manually inspect the workspace in the IBM Cloud console
- Workspace logs and state are available for analysis
- Remember to manually clean up preserved workspaces

### Log Management

The framework provides comprehensive logging:

- **Default**: Logs are shown only on test failure
- **Verbose**: Set `PrintAllSchematicsLogs: true` to see all logs
- **Job Logs**: All Schematics job execution logs are captured
- **Terraform Outputs**: Available in `LastTestTerraformOutputs`

### Timeout Handling

- **Job Timeout**: Controlled by `WaitJobCompleteMinutes` (default: 120)
- **Apply Timeout**: Applies to plan, apply, and destroy operations
- **Retry Logic**: Built-in retry for transient Schematics API issues

### Failure Modes

Different failure scenarios and their handling:

1. **Plan Failure**: Test stops immediately, workspace preserved if configured
2. **Apply Failure**: Test stops, attempts destroy if possible
3. **Consistency Check Failure**: Reported as test failure, continues to destroy
4. **Destroy Failure**: Reported as test failure, workspace preserved
5. **Hook Failure**: Test stops at hook failure point

## State Management

### State File Handling

- **Initial State**: Empty state for new deployments
- **Upgrade Tests**: State carried over from base branch deployment
- **State Backup**: Schematics automatically backs up state
- **State Access**: Not directly accessible in tests (use outputs instead)

### Output Access

Terraform outputs are available after apply:

```golang
PostApplyHook: func(options *testschematic.TestSchematicOptions) error {
    outputs := options.LastTestTerraformOutputs
    if outputs != nil {
        // Access specific outputs
        if endpoint, ok := outputs["api_endpoint"]; ok {
            t.Logf("API endpoint: %s", endpoint.(string))
        }

        // Iterate through all outputs
        for key, value := range outputs {
            t.Logf("Output %s: %v", key, value)
        }
    }
    return nil
}
```

## Testing Best Practices

### Test Organization

1. **Separate Concerns**: Use different test functions for different scenarios
2. **Parallel Execution**: Use `t.Parallel()` for concurrent test execution
3. **Resource Isolation**: Use unique prefixes to avoid conflicts
4. **Cleanup Verification**: Always verify resources are properly destroyed

### Error Handling

1. **Graceful Degradation**: Handle hook failures gracefully
2. **Meaningful Messages**: Provide clear error messages in assertions
3. **Debug Information**: Log relevant information for troubleshooting
4. **Timeout Configuration**: Set appropriate timeouts for long-running deployments

### Performance Optimization

1. **Efficient Patterns**: Use minimal TAR file patterns
2. **Region Selection**: Use `BestRegionYAMLPath` for optimal region selection
3. **Parallel Tests**: Run independent tests in parallel
4. **Resource Cleanup**: Ensure proper cleanup to avoid resource limits

### Example Complete Test Flow

```golang
func TestCompleteFlow(t *testing.T) {
    t.Parallel()

    // Setup phase
    options := testschematic.TestSchematicOptionsDefault(&testschematic.TestSchematicOptions{
        Testing:               t,
        Prefix:                "complete-flow",
        TarIncludePatterns:    []string{"*.tf", "modules/**/*.tf"},
        TemplateFolder:        "examples/complete",
        DeleteWorkspaceOnFail: false, // Keep for debugging

        // Comprehensive hooks
        PreApplyHook: func(opts *testschematic.TestSchematicOptions) error {
            t.Log("Starting deployment validation...")
            return nil
        },

        PostApplyHook: func(opts *testschematic.TestSchematicOptions) error {
            t.Log("Validating deployed infrastructure...")
            outputs := opts.LastTestTerraformOutputs

            // Validate required outputs exist
            requiredOutputs := []string{"vpc_id", "subnet_ids", "security_group_id"}
            for _, output := range requiredOutputs {
                if _, exists := outputs[output]; !exists {
                    return fmt.Errorf("required output %s not found", output)
                }
            }

            // Additional validation logic
            return validateInfrastructure(outputs)
        },

        PreDestroyHook: func(opts *testschematic.TestSchematicOptions) error {
            t.Log("Preparing for resource cleanup...")
            return nil
        },

        PostDestroyHook: func(opts *testschematic.TestSchematicOptions) error {
            t.Log("Verifying complete cleanup...")
            return verifyResourceCleanup(opts.Prefix)
        },
    })

    // Configure variables
    options.TerraformVars = []testschematic.TestSchematicTerraformVar{
        {Name: "ibmcloud_api_key", Value: options.RequiredEnvironmentVars["TF_VAR_ibmcloud_api_key"], DataType: "string", Secure: true},
        {Name: "region", Value: options.Region, DataType: "string"},
        {Name: "prefix", Value: options.Prefix, DataType: "string"},
        {Name: "resource_group", Value: "default", DataType: "string"},
    }

    // Execute test
    err := options.RunSchematicTest()
    assert.NoError(t, err, "Complete flow test should succeed")
}
```
