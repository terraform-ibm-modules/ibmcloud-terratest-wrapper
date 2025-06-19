# Stack Testing Process

This document explains the detailed testing lifecycle that the stack testing framework automatically handles.

## Overview

The stack testing framework manages the complete lifecycle of testing IBM Cloud Projects stacks, from project setup to resource cleanup.

## Testing Lifecycle

### 1. Pre-test Setup

- **Environment Validation**: Checks that required environment variables are set
- **File Validation**: Verifies that stack definition and catalog files exist
- **Configuration Validation**: Validates the test configuration options

### 2. Project Creation

- **Project Setup**: Creates a new IBM Cloud Project with specified configuration
- **Location Selection**: Uses specified region or selects one automatically
- **Monitoring Setup**: Configures project monitoring if enabled
- **Permissions**: Sets up appropriate access permissions

### 3. Stack Configuration

- **Stack Definition**: Loads and parses the stack definition file
- **Catalog Import**: Imports the catalog configuration
- **Member Configuration**: Sets up individual stack member configurations
- **Input Application**: Applies stack-level and member-specific inputs

### 4. Hook Execution

- **PreDeployHook**: Executes custom pre-deployment logic if defined
- **Custom Setup**: Allows for environment preparation and validation

### 5. Deployment Phase

- **Stack Deployment**: Triggers deployment of the stack and all members
- **Progress Monitoring**: Monitors deployment status and logs progress
- **Error Handling**: Captures and reports deployment failures
- **Timeout Management**: Enforces deployment timeout limits

### 6. Post-Deployment

- **PostDeployHook**: Executes custom post-deployment validation if defined
- **Resource Validation**: Validates that resources were created successfully
- **Integration Testing**: Allows for testing of deployed services

### 7. Undeploy Phase

- **PreUndeployHook**: Executes custom pre-undeploy logic if defined
- **Stack Undeploy**: Triggers undeploy of all stack members
- **Resource Cleanup**: Ensures all resources are properly removed
- **Progress Monitoring**: Monitors undeploy status

### 8. Final Cleanup

- **PostUndeployHook**: Executes custom post-undeploy logic if defined
- **Project Cleanup**: Deletes the test project (unless skipped)
- **Verification**: Verifies that cleanup completed successfully

## Detailed Process Steps

### Project Management

**Project Creation:**
- Creates project with unique name based on prefix
- Configures project location (specified or auto-selected)
- Sets up project-level settings (monitoring, destroy-on-delete)
- Establishes project permissions and access

**Project Configuration:**
```golang
options.ProjectLocation = "us-south"
options.ProjectDestroyOnDelete = core.BoolPtr(true)
options.ProjectMonitoringEnabled = core.BoolPtr(true)
```

### Stack Definition Processing

**Stack Configuration Loading:**
- Reads stack definition from specified JSON file
- Parses member configurations and dependencies
- Validates stack structure and requirements

**Catalog Integration:**
- Loads catalog information from JSON file
- Maps stack members to catalog offerings
- Validates version compatibility

### Input Management

**Stack-Level Inputs:**
- Applied to the entire stack configuration
- Override default values in stack definition
- Provide environment-specific settings

**Member-Specific Inputs:**
- Applied to individual stack members
- Allow customization per component
- Override both stack and default values

### Deployment Operations

**Deployment Process:**
- Triggers deployment through IBM Cloud Projects API
- Monitors deployment status for all stack members
- Handles dependencies between stack members
- Provides detailed logging of deployment progress

**Status Monitoring:**
- Polls deployment status at regular intervals
- Reports progress updates during deployment
- Captures error details for failed deployments
- Enforces timeout limits to prevent hanging tests

### Error Handling

**Validation Failures:**
- Stops execution before deployment if validations fail
- Provides detailed error messages and resolution guidance
- Allows configuration validation without deployment

**Deployment Failures:**
- Captures detailed error information from IBM Cloud
- Attempts cleanup of partial deployments
- Provides logs and diagnostic information
- Ensures test doesn't leave orphaned resources

**Cleanup Failures:**
- Logs cleanup failures but continues with test completion
- Provides manual cleanup guidance
- Reports resources that may need manual intervention

## Timeout Configuration

**Default Timeouts:**
- Deployment: 6 hours (360 minutes)
- Undeploy: 6 hours (360 minutes)

**Custom Timeout:**
```golang
options.DeployTimeoutMinutes = 120 // 2 hours
```

**Timeout Behavior:**
- Applies to both deploy and undeploy operations
- Framework monitors progress and enforces limits
- Provides graceful handling when timeouts occur

## Skip Options

The framework provides several options to skip parts of the testing process:

```golang
// Skip undeploy but keep project cleanup
options.SkipUndeploy = true

// Skip project deletion
options.SkipProjectDelete = true

// Skip entire teardown process
options.SkipTestTearDown = true
```

## Logging and Monitoring

**Built-in Logging:**
- Detailed logs for each phase of the testing process
- Progress updates during long-running operations
- Error details with context and resolution guidance

**Custom Logging:**
```golang
options.PostDeployHook = func(options *testprojects.TestProjectsOptions) error {
    t.Log("Custom validation step completed")
    return nil
}
```

**Monitoring Integration:**
- Project monitoring can be enabled/disabled
- Framework logs project and configuration IDs for external monitoring
- Integration with IBM Cloud monitoring services when enabled

## Resource Management

**Resource Naming:**
- All resources use the specified prefix for uniqueness
- Automatic random suffix generation for test isolation
- Consistent naming patterns across all created resources

**Cleanup Verification:**
- Verifies resource deletion after undeploy
- Reports any resources that weren't properly cleaned up
- Provides guidance for manual cleanup if needed

**State Management:**
- Maintains state information throughout the test lifecycle
- Provides access to project and stack details in hooks
- Enables custom validation and resource verification
