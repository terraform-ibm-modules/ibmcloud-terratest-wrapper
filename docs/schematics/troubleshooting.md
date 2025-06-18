# Schematics Testing Troubleshooting

This guide helps you diagnose and resolve common issues when using the Schematics testing framework.

## Common Issues and Solutions

### Authentication Problems

#### Issue: "401 Unauthorized" or "403 Forbidden" errors

**Symptoms**:
- Test fails immediately with authentication errors
- Error mentions invalid API key or insufficient permissions

**Solutions**:

1. **Verify API Key**:
   ```bash
   export TF_VAR_ibmcloud_api_key="your-api-key-here"
   echo $TF_VAR_ibmcloud_api_key  # Verify it's set
   ```

2. **Check API Key Permissions**:
   - Ensure the API key has Schematics service access
   - Verify permissions for target resource group
   - Check for account-level restrictions

3. **Test API Key Manually**:
   ```bash
   ibmcloud login --apikey $TF_VAR_ibmcloud_api_key
   ibmcloud target -g default
   ibmcloud schematics workspace list
   ```

#### Issue: "Invalid credentials for Git repository access"

**Symptoms**:
- Test fails during TAR upload or workspace setup
- Error mentions Git authentication failures

**Solutions**:

1. **Configure Private Repository Access**:
   ```golang
   options.AddNetrcCredential("github.com", "username", os.Getenv("GITHUB_TOKEN"))
   ```

2. **Verify Token Permissions**:
   - Check that the Git token has read access to required repositories
   - Ensure token hasn't expired

### Workspace Issues

#### Issue: "Workspace creation failed"

**Symptoms**:
- Error creating Schematics workspace
- Region or location errors

**Solutions**:

1. **Check Region Availability**:
   ```golang
   options := testschematic.TestSchematicOptionsDefault(&testschematic.TestSchematicOptions{
       WorkspaceLocation: "us-south", // Specify explicit region
       // ... other options
   })
   ```

2. **Use Dynamic Region Selection**:
   ```golang
   options := testschematic.TestSchematicOptionsDefault(&testschematic.TestSchematicOptions{
       BestRegionYAMLPath: "test-config/regions.yaml",
       // ... other options
   })
   ```

3. **Verify Service Availability**:
   - Check IBM Cloud status page for Schematics service issues
   - Try different regions if one is experiencing problems

#### Issue: "TAR file upload failed"

**Symptoms**:
- Error during file upload to workspace
- Large file size errors

**Solutions**:

1. **Optimize TAR Patterns**:
   ```golang
   options := testschematic.TestSchematicOptionsDefault(&testschematic.TestSchematicOptions{
       TarIncludePatterns: []string{
           "*.tf",
           "examples/basic/*.tf",
           // Avoid including large directories like .git, .terraform
       },
       // ... other options
   })
   ```

2. **Check File Size Limits**:
   - Schematics has TAR file size limits
   - Exclude unnecessary files (.git, .terraform, node_modules, etc.)

3. **Verify File Paths**:
   - Ensure all patterns match existing files
   - Check for typos in `TarIncludePatterns`

### Variable and Configuration Issues

#### Issue: "Variable validation failed"

**Symptoms**:
- Test fails during plan phase
- Error messages about invalid variable values or types

**Solutions**:

1. **Check Variable Types**:
   ```golang
   options.TerraformVars = []testschematic.TestSchematicTerraformVar{
       // Correct: Boolean value with bool type
       {Name: "enable_feature", Value: true, DataType: "bool"},

       // Incorrect: String value with bool type
       // {Name: "enable_feature", Value: "true", DataType: "bool"},
   }
   ```

2. **Validate Complex Types**:
   ```golang
   // List variable
   {Name: "zones", Value: []string{"us-south-1", "us-south-2"}, DataType: "list(string)"},

   // Map variable
   {Name: "tags", Value: map[string]interface{}{
       "environment": "test",
       "cost_center": 12345,
   }, DataType: "map(any)"},
   ```

3. **Check Required Variables**:
   - Ensure all required Terraform variables are provided
   - Verify variable names match exactly (case-sensitive)

#### Issue: "Template folder not found"

**Symptoms**:
- Error about missing template folder
- Terraform execution fails to find main.tf

**Solutions**:

1. **Verify Template Folder**:
   ```golang
   options := testschematic.TestSchematicOptionsDefault(&testschematic.TestSchematicOptions{
       TemplateFolder: "examples/basic", // Must exist in TAR file
       TarIncludePatterns: []string{
           "*.tf",
           "examples/basic/*.tf", // Include the template folder
       },
   })
   ```

2. **Check Directory Structure**:
   - Ensure the `TemplateFolder` path exists in your project
   - Verify `TarIncludePatterns` includes files from that folder

### Execution Timeouts

#### Issue: "Job timeout exceeded"

**Symptoms**:
- Test fails with timeout errors
- Long-running apply or destroy operations

**Solutions**:

1. **Increase Timeout**:
   ```golang
   options := testschematic.TestSchematicOptionsDefault(&testschematic.TestSchematicOptions{
       WaitJobCompleteMinutes: 180, // Increase from default 120
       // ... other options
   })
   ```

2. **Optimize Resource Creation**:
   - Review Terraform code for efficiency
   - Consider using smaller test configurations
   - Check for unnecessary dependencies

3. **Monitor Job Progress**:
   ```golang
   options := testschematic.TestSchematicOptionsDefault(&testschematic.TestSchematicOptions{
       PrintAllSchematicsLogs: true, // See detailed progress
       // ... other options
   })
   ```

### Resource Management Issues

#### Issue: "Resources not properly destroyed"

**Symptoms**:
- Test passes but resources remain in account
- Billing charges for test resources

**Solutions**:

1. **Enable Debugging**:
   ```bash
   export DO_NOT_DESTROY_ON_FAILURE=false
   ```

2. **Check Destroy Logs**:
   ```golang
   options := testschematic.TestSchematicOptionsDefault(&testschematic.TestSchematicOptions{
       PrintAllSchematicsLogs: true,
       DeleteWorkspaceOnFail: false, // Keep workspace for inspection
       // ... other options
   })
   ```

3. **Manual Cleanup**:
   - Check IBM Cloud console for remaining resources
   - Use workspace ID from test logs to inspect Schematics workspace
   - Manually destroy resources if needed

#### Issue: "Consistency check failures"

**Symptoms**:
- Test fails after successful apply
- Unexpected plan changes detected

**Solutions**:

1. **Add Ignore Rules**:
   ```golang
   options := testschematic.TestSchematicOptionsDefault(&testschematic.TestSchematicOptions{
       IgnoreAdds:    []string{"random_id.suffix"},
       IgnoreUpdates: []string{"time_rotating.cert"},
       // ... other options
   })
   ```

2. **Review Terraform Code**:
   - Check for resources that change on every plan
   - Look for timestamp-based resources
   - Verify provider configurations

### Network and Connectivity Issues

#### Issue: "Network connectivity errors"

**Symptoms**:
- Intermittent failures
- API timeout errors
- Connection refused errors

**Solutions**:

1. **Implement Retry Logic in Hooks**:
   ```golang
   PostApplyHook: func(options *testschematic.TestSchematicOptions) error {
       return retry.DoWithRetry(t, "validate-endpoint", 3, 10*time.Second, func() (string, error) {
           return validateEndpoint(options.LastTestTerraformOutputs["endpoint"].(string))
       })
   }
   ```

2. **Check Service Status**:
   - Verify IBM Cloud service status
   - Check for planned maintenance windows

## Debugging Techniques

### Enable Verbose Logging

```golang
options := testschematic.TestSchematicOptionsDefault(&testschematic.TestSchematicOptions{
    Testing:                t,
    Prefix:                 "debug-test",
    PrintAllSchematicsLogs: true,  // Show all Schematics logs
    DeleteWorkspaceOnFail:  false, // Keep workspace for inspection
    // ... other options
})
```

### Preserve Workspaces for Inspection

```golang
options := testschematic.TestSchematicOptionsDefault(&testschematic.TestSchematicOptions{
    DeleteWorkspaceOnFail: false, // Keep failed workspaces
    SkipTestTearDown:      false, // Set to true to skip all cleanup
    // ... other options
})
```

### Add Debug Hooks

```golang
options := testschematic.TestSchematicOptionsDefault(&testschematic.TestSchematicOptions{
    Testing: t,
    Prefix:  "debug",

    PreApplyHook: func(opts *testschematic.TestSchematicOptions) error {
        t.Logf("Starting apply with variables: %+v", opts.TerraformVars)
        return nil
    },

    PostApplyHook: func(opts *testschematic.TestSchematicOptions) error {
        t.Logf("Apply completed, outputs: %+v", opts.LastTestTerraformOutputs)
        return nil
    },
})
```

### Environment Variable Debugging

```bash
# Enable detailed logging for IBM Cloud CLI
export IBMCLOUD_TRACE=true

# Keep resources after test failure
export DO_NOT_DESTROY_ON_FAILURE=true

# Set debug mode for additional logging
export DEBUG=true
```

## Performance Optimization

### Reduce TAR File Size

```golang
options := testschematic.TestSchematicOptionsDefault(&testschematic.TestSchematicOptions{
    TarIncludePatterns: []string{
        "*.tf",
        "modules/**/*.tf",
        "examples/basic/*.tf",
        // Exclude: .git, .terraform, *.zip, node_modules, etc.
    },
})
```

### Optimize Test Execution

```golang
func TestParallelExecution(t *testing.T) {
    t.Parallel() // Enable parallel execution

    // Use minimal resource configurations for faster testing
    options := testschematic.TestSchematicOptionsDefault(&testschematic.TestSchematicOptions{
        Testing:               t,
        Prefix:                "fast-test",
        TarIncludePatterns:    []string{"*.tf", "examples/minimal/*.tf"},
        TemplateFolder:        "examples/minimal", // Use minimal example
        WaitJobCompleteMinutes: 60, // Reduce timeout for simple tests
    })
}
```

### Use Region Selection

```golang
// Create regions.yaml for optimal region selection
options := testschematic.TestSchematicOptionsDefault(&testschematic.TestSchematicOptions{
    BestRegionYAMLPath: "test-config/regions.yaml",
    // Framework will select the best available region
})
```

## Getting Help

### Collect Diagnostic Information

When reporting issues, include:

1. **Test Configuration**:
   ```golang
   t.Logf("Test options: %+v", options)
   ```

2. **Environment Information**:
   ```bash
   go version
   echo $TF_VAR_ibmcloud_api_key | cut -c1-10  # First 10 chars only
   ibmcloud --version
   ```

3. **Error Messages**:
   - Complete error output
   - Schematics job logs
   - Workspace ID for manual inspection

4. **Timing Information**:
   - Test duration
   - Timeout settings
   - Job completion times

### Manual Workspace Inspection

If a test fails and workspace is preserved:

1. **Find Workspace**:
   ```bash
   ibmcloud schematics workspace list | grep your-prefix
   ```

2. **Inspect Workspace**:
   ```bash
   ibmcloud schematics workspace get --id WORKSPACE_ID
   ```

3. **Check Job Logs**:
   ```bash
   ibmcloud schematics job list --id WORKSPACE_ID
   ibmcloud schematics job get --id JOB_ID
   ```

4. **Manual Cleanup**:
   ```bash
   ibmcloud schematics destroy --id WORKSPACE_ID
   ibmcloud schematics workspace delete --id WORKSPACE_ID
   ```

### Community Resources

- **GitHub Issues**: Report bugs and feature requests
- **IBM Cloud Docs**: Official Schematics documentation
- **Terraform Registry**: Module documentation and examples
- **Stack Overflow**: Community Q&A with `ibm-cloud-schematics` tag
