# Stack Testing Troubleshooting

This guide covers common issues and solutions when using the stack testing framework.

## Common Issues

### Stack Definition File Not Found

**Error:** `stack definition file not found`

**Solutions:**


- Verify the file path in `StackConfigurationPath`
- Ensure the file exists in the specified location
- Check file permissions

### Catalog File Issues

**Error:** `catalog file not found` or `invalid catalog format`


**Solutions:**

- Verify the file path in `StackCatalogJsonPath`
- Validate JSON format of the catalog file
- Ensure catalog contains required offering information

### Deployment Timeout

**Error:** `deployment timed out after X minutes`


**Solutions:**

```golang
// Increase timeout for complex stacks
options.DeployTimeoutMinutes = 240 // 4 hours
```

### Environment Variable Missing


**Error:** `required environment variable not set`

**Solutions:**

```bash
export TF_VAR_ibmcloud_api_key="your-api-key"
```

### Project Creation Failures


**Error:** `failed to create project`

**Solutions:**

- Verify IBM Cloud API key permissions
- Check project quotas and limits
- Ensure region is valid and accessible

## Debug Techniques

### Enable Detailed Logging

Use the testing object for detailed output:

```golang
options.PostDeployHook = func(options *testprojects.TestProjectsOptions) error {
    t.Logf("Project ID: %s", options.ProjectID)
    t.Logf("Stack ID: %s", options.StackID)
    return nil
}
```

### Skip Cleanup for Investigation

```golang
options.SkipTestTearDown = true  // Keep everything for manual inspection
options.SkipUndeploy = true      // Skip undeploy only
options.SkipProjectDelete = true // Keep project but undeploy resources
```

### Validate Configuration Before Deployment

Use hooks to validate your configuration:

```golang
options.PreDeployHook = func(options *testprojects.TestProjectsOptions) error {
    // Validate inputs
    if options.StackInputs["ibmcloud_api_key"] == "" {
        return fmt.Errorf("API key not provided")
    }
    return nil
}
```

## Getting Help

When seeking help, include:

1. Complete error messages
2. Test configuration (`TestProjectsOptions`)
3. Stack definition and catalog files (sanitized)
4. Environment details (Go version, OS)
5. IBM Cloud account type and region

## Resource Cleanup

If tests fail and leave resources behind:

1. Check the IBM Cloud console for the test project
2. Delete the project manually if it still exists
3. Verify no orphaned resources remain
4. Check for any charges related to test resources
