# Troubleshooting Guide

This guide covers common issues and solutions when using the addon testing framework.

## Known Issues

GIT authentication failures while importing offering:

```text
error: test setup has failed:error preparing offering import: failed to get repository info for offering import: fetch failed: ssh: handshake failed: ssh: unable to authenticate, attempted methods [none publickey], no supported methods remain)
```

**Cause**: Your SSH identity does not exist on your SSH agent.

**Solutions:**

1. **Add your GIT SSH key to your SSH agent** (recommended):

    Verify that your SSH agent is running
    ```
    eval "$(ssh-agent -s)"
    ```

    Add your SSH identity to the SSH agent
    ```
    ssh-add ~/.ssh/<YOUR_SSH_KEY>
    ```

2. **Use https authentication**: Use https authentication using a token. Update your remote to use https in place of ssh.


### API Key Validation Failures

These are intermittent issues with IBM Cloud's reference resolution service that can occur occasionally.

#### API Key Validation Error

**Error message:**

```text
Error resolving references: invalid status code: 500, body: {"errors":[{"state":"Failed to validate api key token.","code":"failed_request","message":"Failed to validate api key token."}],"status_code":500,"trace":"..."}
```

**Cause**: Intermittent issue with IBM Cloud's reference resolution service. The service occasionally has temporary issues validating API key tokens, even when the API key is valid.

**Solutions:**

1. **Automatic Retry** (recommended): The framework automatically retries reference resolution up to 6 times with exponential backoff.

2. **Automatic Skip**: When this specific error occurs after all retries, the framework automatically skips reference validation and continues with the test.

3. **Manual Skip** (for development):

   ```golang
   options.SkipRefValidation = true
   ```

#### Project Not Found Error

**Error message:**

```text
Error resolving references: invalid status code: 404, body: {"errors":[{"state":"Specified provider instance with id 'project-id' could not be found.","code":"not_found","message":"..."}],"status_code":404,"trace":"..."}
```

**Cause**: Timing issue that occurs when checking project details too quickly after project creation. The resolver API needs time to be updated with new project information.

**Solutions:**

1. **Automatic Retry**: The framework automatically handles this with retry logic.
2. **Automatic Skip**: Framework skips validation if the issue persists after retries.

## Common Test Issues

### Local Changes Detected

**Issue**: Test fails immediately with "local changes detected" error.

**Error example:**

```text
Local changes detected. Please commit or stash changes before running tests.
Modified files: main.tf, variables.tf
```

**Solutions:**

```bash
# Option 1: Commit changes
git add .
git commit -m "test changes"

# Option 2: Stash changes
git stash

# Option 3: Skip check in code
```

```golang
options.SkipLocalChangeCheck = true
```

**Prevention**: Always commit or stash changes before running tests, or configure appropriate ignore patterns.

### Timeout Issues

**Issue**: Tests fail with deployment or undeploy timeout errors.

**Error example:**

```text
Deployment timed out after 360 minutes
```

**Solutions:**

```golang
// Increase timeout for complex deployments
options.DeployTimeoutMinutes = 480 // 8 hours

// For debugging, skip infrastructure deployment
options.SkipInfrastructureDeployment = true
```

**Diagnosis:**

- Check IBM Cloud console for deployment status
- Review project logs for specific error details
- Consider resource complexity and regional availability

### Resource Group Access Issues

**Issue**: Test fails with resource group access errors.

**Error example:**

```text
User does not have access to resource group 'my-rg'
```

**Solutions:**

```golang
// Use a resource group you have access to
options.ResourceGroup = "accessible-rg"

// Use default resource group (not recommended for production)
options.ResourceGroup = "Default"
```

**Prevention**: Verify resource group access before running tests.

### Catalog Permission Issues

**Issue**: Test fails when creating or accessing catalogs.

**Error example:**

```text
User does not have permission to create catalog
```

**Solutions:**

```golang
// Use existing catalog instead of creating new one
options.CatalogUseExisting = true
options.CatalogName = "existing-catalog-name"
```

**Requirements**: Ensure your API key has appropriate catalog management permissions.

### Dependency Resolution Failures

**Issue**: Test fails during dependency validation or deployment.

**Error example:**

```text
Dependency 'account-base' not found in catalog
```

**Solutions:**

```golang
// Skip dependency validation
options.SkipDependencyValidation = true

// Use enhanced output for diagnosis
options.EnhancedTreeValidationOutput = true

// Override dependencies manually
options.AddonConfig.Dependencies = []cloudinfo.AddonConfig{
    {
        OfferingName:   "deploy-arch-ibm-account-infra-base",
        OfferingFlavor: "resource-group-only",
        Enabled:        core.BoolPtr(false), // Disable problematic dependency
    },
}
```

## Environment Issues

### Missing Environment Variables

**Issue**: Test fails with missing environment variable error.

**Error example:**

```text
Required environment variable TF_VAR_ibmcloud_api_key is not set
```

**Solutions:**

```bash
# Set required environment variable
export TF_VAR_ibmcloud_api_key="your-api-key"

# Verify it's set
echo $TF_VAR_ibmcloud_api_key
```

### Invalid API Key

**Issue**: Test fails with API key authentication errors.

**Error example:**

```text
Invalid API key provided
```

**Solutions:**

1. **Verify API key**: Check that your API key is valid and not expired
2. **Regenerate API key**: Create a new API key in IBM Cloud console
3. **Check permissions**: Ensure API key has required permissions

### Network Connectivity Issues

**Issue**: Tests fail due to network connectivity problems.

**Solutions:**

```golang
// Increase timeout for network-related operations
options.DeployTimeoutMinutes = 480

// Add retry logic in hooks
options.PostDeployHook = func(options *testaddons.TestAddonOptions) error {
    maxRetries := 5
    for i := 0; i < maxRetries; i++ {
        if err := testConnectivity(); err != nil {
            if i == maxRetries-1 {
                return err
            }
            time.Sleep(time.Duration(i+1) * 30 * time.Second)
            continue
        }
        return nil
    }
    return nil
}
```

## Resource Issues

### Resource Quota Exceeded

**Issue**: Tests fail due to resource quota limits.

**Error example:**

```text
Quota exceeded for resource type 'vpc' in region 'us-south'
```

**Solutions:**

1. **Clean up existing resources**: Remove unused resources in the region
2. **Use different region**: Test in a region with available quota
3. **Request quota increase**: Contact IBM Cloud support
4. **Skip infrastructure deployment**: For validation-only testing

```golang
// Test in different region
options.AddonConfig.Inputs["region"] = "eu-gb"

// Skip infrastructure deployment
options.SkipInfrastructureDeployment = true
```

### Resource Naming Conflicts

**Issue**: Tests fail due to resource naming conflicts.

**Error example:**

```text
Resource name 'test-vpc-123' already exists
```

**Solutions:**

```golang
// Ensure unique prefixes
func setupAddonOptions(t *testing.T, prefix string) *testaddons.TestAddonOptions {
    // Framework automatically adds random suffix to prefix
    options := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
        Testing:       t,
        Prefix:        prefix, // Will become prefix-random
        ResourceGroup: resourceGroup,
    })
    return options
}

// Or use more specific prefixes
options := setupAddonOptions(t, fmt.Sprintf("test-%d", time.Now().Unix()))
```

### Cleanup Failures

**Issue**: Resources remain after test completion.

**Diagnosis steps:**

1. Check test logs for cleanup errors
2. Verify project deletion in IBM Cloud console
3. Check resource group for orphaned resources

**Solutions:**

```golang
// Add cleanup verification hook
options.PostUndeployHook = func(options *testaddons.TestAddonOptions) error {
    return verifyResourceCleanup(options.ResourceGroup, options.Prefix)
}

func verifyResourceCleanup(resourceGroup, prefix string) error {
    // Check for resources with test prefix
    // Log any remaining resources
    // Optionally perform manual cleanup
    return nil
}
```

**Manual cleanup:**

```bash
# List resources in resource group
ibmcloud resource service-instances --resource-group-name "my-rg"

# Delete specific resources
ibmcloud resource service-instance-delete "resource-name"
```

## Parallel Testing Issues

### Resource Contention

**Issue**: Parallel tests interfere with each other.

**Solutions:**

```golang
// Use unique prefixes per test
testCases := []testaddons.AddonTestCase{
    {Name: "Test1", Prefix: "unique1"},
    {Name: "Test2", Prefix: "unique2"},
}

// Use different resource groups
func setupAddonOptions(t *testing.T, prefix string) *testaddons.TestAddonOptions {
    return testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
        Testing:       t,
        Prefix:        prefix,
        ResourceGroup: fmt.Sprintf("test-rg-%s", prefix),
    })
}
```

### Rate Limiting

**Issue**: Tests fail due to API rate limits.

**Solutions:**

```golang
// Reduce parallelism
// Don't use t.Parallel() for all tests

// Add delays between operations
options.PreDeployHook = func(options *testaddons.TestAddonOptions) error {
    time.Sleep(10 * time.Second) // Stagger API calls
    return nil
}

// Increase timeouts
options.DeployTimeoutMinutes = 480
```

## Debugging Techniques

### Enable Verbose Logging

```golang
// Enable detailed validation output
options.VerboseValidationErrors = true
options.EnhancedTreeValidationOutput = true

// Add custom logging in hooks
options.PreDeployHook = func(options *testaddons.TestAddonOptions) error {
    options.Logger.LongInfo("Detailed debug information: %+v", options.AddonConfig)
    return nil
}
```

### Skip Components for Isolation

```golang
// Skip validations to isolate issues
options.SkipLocalChangeCheck = true
options.SkipRefValidation = true
options.SkipDependencyValidation = true

// Skip infrastructure for validation testing
options.SkipInfrastructureDeployment = true

// Skip cleanup for investigation
options.SkipTestTearDown = true
```

### Use Hooks for Debugging

```golang
options.PostDeployHook = func(options *testaddons.TestAddonOptions) error {
    // Capture state for debugging
    options.Logger.LongInfo("Project ID: %s", options.ProjectID)
    options.Logger.LongInfo("Catalog ID: %s", options.AddonConfig.CatalogID)

    // Wait for manual investigation
    if os.Getenv("DEBUG_PAUSE") == "true" {
        fmt.Println("Press Enter to continue...")
        fmt.Scanln()
    }

    return nil
}
```

### Check IBM Cloud Console

1. **Projects**: View project status and configurations
2. **Catalogs**: Check offering import status
3. **Resource Groups**: Verify resource creation and cleanup
4. **Activity Tracker**: Review API calls and errors
5. **Logs**: Check detailed deployment logs

## Performance Issues

### Slow Test Execution

**Issue**: Tests take much longer than expected.

**Diagnosis:**

```golang
// Add timing to hooks
options.PreDeployHook = func(options *testaddons.TestAddonOptions) error {
    start := time.Now()
    defer func() {
        options.Logger.ShortInfo("Pre-deploy hook took %v", time.Since(start))
    }()
    // Hook logic
    return nil
}
```

**Solutions:**

- Review addon complexity and deployment time
- Consider parallel testing for independent tests
- Use `SkipInfrastructureDeployment` for validation-only tests
- Check network connectivity and region selection

### Memory Issues

**Issue**: Tests consume excessive memory.

**Solutions:**

```golang
// Limit parallel test count
// Use smaller test datasets
// Clean up resources in hooks

options.PostDeployHook = func(options *testaddons.TestAddonOptions) error {
    // Force garbage collection if needed
    runtime.GC()
    return nil
}
```

## Getting Help

### Diagnostic Information to Collect

When seeking help, include:

1. **Test configuration**: Your `TestAddonOptions` setup
2. **Error messages**: Complete error output with stack traces
3. **Environment**: Go version, framework version, OS
4. **IBM Cloud details**: Region, account type, permissions
5. **Test logs**: Complete test output with timestamps

### Log Analysis

```golang
// Enable comprehensive logging
options.VerboseValidationErrors = true
options.EnhancedTreeValidationOutput = true

// Add context to custom logs
options.Logger.LongInfo("Test context: prefix=%s, rg=%s, project=%s",
    options.Prefix, options.ResourceGroup, options.ProjectName)
```

### Minimal Reproduction Case

Create a minimal test that reproduces the issue:

```golang
func TestMinimalReproduction(t *testing.T) {
    options := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
        Testing:       t,
        Prefix:        "minimal",
        ResourceGroup: "test-rg",
    })

    options.AddonConfig = cloudinfo.NewAddonConfigTerraform(
        options.Prefix,
        "simple-addon",
        "basic",
        map[string]interface{}{
            "prefix": options.Prefix,
        },
    )

    err := options.RunAddonTest()
    assert.NoError(t, err)
}
```

### Framework Issues vs. IBM Cloud Issues

**Framework Issues** (report to framework maintainers):

- Configuration validation problems
- Hook execution issues
- Framework crashes or panics
- Documentation gaps

**IBM Cloud Issues** (report to IBM Cloud support):

- Service API failures
- Resource provisioning problems
- Quota and permission issues
- Service outages

### Community Resources

- Framework documentation and examples
- IBM Cloud documentation
- Developer forums and communities
- IBM Cloud support channels

#### Member Configuration Deployment Reference Warning

**Warning message:**

```text
⚠   ref://project.example/configs/member-config/inputs/resource_group_name - Warning: unresolved
      Message: The project reference requires the specified member configuration deploy-arch-ibm-account-infra-base-abc123 to be deployed. Please deploy the referring configuration.
      Code: 400
      This is a valid reference that cannot be resolved until the member configuration is deployed.
```

**Cause**: This occurs when a reference points to a resource that will be created by a member configuration that hasn't been deployed yet. This is a normal part of the deployment process in multi-tier architectures.

**Behavior**: The framework treats this as a warning, not an error. The test will continue and the reference will be resolved once the member configuration is deployed.

**Solutions:**

1. **Normal Operation**: This is expected behavior. The framework will proceed with deployment and the reference will resolve automatically.

2. **If the reference fails during actual deployment**: This indicates a real issue with the reference configuration that needs to be addressed.

3. **Skip reference validation** (for development/debugging):

   ```golang
   options.SkipRefValidation = true
   ```
