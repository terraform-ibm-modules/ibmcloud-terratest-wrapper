package testprojects

import (
	"errors"
	"fmt"
	"github.com/gruntwork-io/terratest/modules/logger"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/IBM/go-sdk-core/v5/core"
	project "github.com/IBM/project-go-sdk/projectv1"
	"github.com/stretchr/testify/assert"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/cloudinfo"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/common"
)

func (options *TestProjectsOptions) ConfigureTestStack() error {
	// Configure the test stack
	options.ProjectsLog("Configuring Test Stack")
	var stackResp *core.DetailedResponse
	var stackErr error
	options.currentStackConfig = &cloudinfo.ConfigDetails{ProjectID: *options.currentProject.ID}
	options.currentStack, stackResp, stackErr = options.CloudInfoService.CreateStackFromConfigFile(options.currentStackConfig, options.StackConfigurationPath, options.StackCatalogJsonPath)
	if !assert.NoError(options.Testing, stackErr) {
		options.ProjectsLog("Failed to configure Test Stack")
		var sdkProblem *core.SDKProblem

		if errors.As(stackErr, &sdkProblem) {
			if strings.Contains(sdkProblem.Summary, "A stack definition member input") &&
				strings.Contains(sdkProblem.Summary, "was not found in the configuration") {
				sdkProblem.Summary = fmt.Sprintf("%s Input name possibly removed or renamed", sdkProblem.Summary)
				// A stack definition member input resource_tag was not found in the configuration primary-da.
				// extract the member config name, get the member config version, get all inputs for this version
				memberName := strings.Split(sdkProblem.Summary, "was not found in the configuration ")[1]
				memberName = strings.Split(memberName, ".")[0]
				versionLocator, vlErr := GetVersionLocatorFromStackDefinitionForMemberName(options.StackConfigurationPath, memberName)
				if !assert.NoError(options.Testing, vlErr) {
					options.ProjectsLog("Error getting version locator from stack definition")
					return sdkProblem
				}
				// get inputs for the member config of version
				version, vererr := options.CloudInfoService.GetCatalogVersionByLocator(versionLocator)
				if !assert.NoError(options.Testing, vererr) {
					options.ProjectsLog("Error getting offering")
					return sdkProblem
				}
				// version.configurations[x].name append all configuration names to validInputs
				validInputs := "Valid Inputs:\n"

				for _, configuration := range version.Configuration {
					validInputs += fmt.Sprintf("\t%s\n", *configuration.Key)
				}

				sdkProblem.Summary = fmt.Sprintf("%s Inputs possibly removed or renamed.\n%s", sdkProblem.Summary, validInputs)
				return sdkProblem
			}
		} else if assert.Equal(options.Testing, 201, stackResp.StatusCode) {
			options.ProjectsLog("Configured Test Stack")
		} else {
			options.ProjectsLog("Failed to configure Test Stack")
			return fmt.Errorf("error configuring test stack response code: %d\nrespone:%s", stackResp.StatusCode, stackResp.Result)
		}
	}
	return nil
}

// TriggerDeploy is assuming auto deploy is enabled so just triggers validate at the stack level
func (options *TestProjectsOptions) TriggerDeploy() error {
	options.ProjectsLog("Triggering Deploy")
	_, _, err := options.CloudInfoService.ValidateProjectConfig(options.currentStackConfig)
	if err != nil {
		return err
	}
	options.ProjectsLog("Deploy Triggered Successfully")
	return nil
}

func (options *TestProjectsOptions) TriggerDeployAndWait() (errors []error) {
	err := options.TriggerDeploy()
	if err != nil {
		return []error{err}
	}

	// Initialize the map to store the start time for each member's state
	memberStateStartTime := make(map[string]time.Time)
	// Mutex to ensure safe concurrent access to the map
	var mu sync.Mutex

	// setup timeout
	deployEndTime := time.Now().Add(time.Duration(options.DeployTimeoutMinutes) * time.Minute)
	deployComplete := false
	failed := false
	for !deployComplete && time.Now().Before(deployEndTime) && !failed {
		options.ProjectsLog("Checking Stack Deploy Status")
		stackDetails, _, err := options.CloudInfoService.GetConfig(options.currentStackConfig)
		if err != nil {
			return []error{err}
		}
		if stackDetails == nil {
			return []error{fmt.Errorf("stackDetails is nil")}
		}
		// Get the current state of all members in the stack
		stackMembers, err := options.CloudInfoService.GetStackMembers(options.currentStackConfig)
		if err != nil {
			return []error{err}
		}
		// If the stack is not fully deployed and no members are in a state that can be deployed then we have an error
		deployableState := false
		memberStates := []string{}
		currentDeployStatus := fmt.Sprintf("[STACK - %s] Current Deploy Status:\n", options.currentStackConfig.Name)
		// loop each member and check state
		for _, member := range stackMembers {
			if member.ID == nil {
				return []error{fmt.Errorf("member ID is nil")}
			}
			memberName, err := options.CloudInfoService.LookupMemberNameByID(stackDetails, *member.ID)
			if err != nil {
				memberName = fmt.Sprintf("Unknown name, ID: %s", *member.ID)
			}
			// Lock the mutex before accessing the map
			mu.Lock()
			// Check if the member's ID is in the map
			// If it is, check if the member has been in the same state for more than 30 minutes
			// If it has, trigger SyncConfig and reset the start time
			// If it is not, store the start time for the member's current state
			if startTime, exists := memberStateStartTime[*member.ID]; exists {
				// Check if the member has been in the same state for more than 30 minutes
				if time.Since(startTime) > 30*time.Minute {
					if member.State == nil {
						mu.Unlock()
						return []error{fmt.Errorf("member state is nil")}
					}
					options.ProjectsLog(fmt.Sprintf("Member %s stuck in state %s for more than 30 minutes, triggering SyncConfig", memberName, *member.State))
					_, syncErr := options.CloudInfoService.SyncConfig(options.currentStackConfig.ProjectID, *member.ID)
					if syncErr != nil {
						errors = append(errors, syncErr)
					}
					// Reset the start time after triggering SyncConfig
					memberStateStartTime[*member.ID] = time.Now()
				}
			} else {
				// Store the start time for the member's current state
				memberStateStartTime[*member.ID] = time.Now()
			}
			mu.Unlock()

			if member.State == nil {
				memberStates = append(memberStates, fmt.Sprintf(" - member %s state is nil, skipping this time", memberName))
				// assume deployable state
				deployableState = true
				continue
			}
			// Lookup member name using member ID from stackDetails Definition Member list
			stateCode := "Unknown"
			if member.StateCode != nil {
				stateCode = *member.StateCode
			}
			if *member.State == project.ProjectConfig_State_Deployed {
				memberStates = append(memberStates, fmt.Sprintf(" - member %s current state: %s", memberName, *member.State))
			} else {
				memberStates = append(memberStates, fmt.Sprintf(" - member %s current state: %s and state code: %s", memberName, *member.State, stateCode))
			}

			if member.StateCode != nil && (*member.StateCode == project.ProjectConfig_StateCode_AwaitingMemberDeployment ||
				*member.StateCode == project.ProjectConfig_StateCode_AwaitingValidation) {
				deployableState = true
			}
			if *member.State == project.ProjectConfig_State_Validating {
				currentDeployStatus = fmt.Sprintf("%s - member %s is still validating\n", currentDeployStatus, memberName)
				deployableState = true
			} else if *member.State == project.ProjectConfig_State_Deploying {
				currentDeployStatus = fmt.Sprintf("%s - member %s is still deploying\n", currentDeployStatus, memberName)
				deployableState = true
			} else if *member.State == project.ProjectConfig_State_Deployed {
				currentDeployStatus = fmt.Sprintf("%s - member %s is deployed\n", currentDeployStatus, memberName)
			} else if member.StateCode != nil && *member.StateCode == project.ProjectConfig_StateCode_AwaitingPrerequisite {
				currentDeployStatus = fmt.Sprintf("%s - member %s is awaiting prerequisite\n", currentDeployStatus, memberName)
			} else {
				currentDeployStatus = fmt.Sprintf("%s - member %s is in an unknown state\n", currentDeployStatus, memberName)
			}
		}

		if !deployableState {
			errorMessage := "Stack stuck in undeployable state:"
			for _, memberState := range memberStates {
				errorMessage = fmt.Sprintf("%s\n%s", errorMessage, memberState)
			}
			errors = append(errors, fmt.Errorf(errorMessage))
			// TODO: Create function to resolve all the refs for the current stack and print an error message with the unresolved refs
			failed = true
		}
		// fail fast if any error states
		if stackDetails.State == nil {
			errors = append(errors, fmt.Errorf("stackDetails state is nil"))
			return errors
		}
		if *stackDetails.State == project.ProjectConfig_State_ApplyFailed {
			// get the error message and add to errors
			errors = append(errors, fmt.Errorf("Failed with state %s", *stackDetails.State))
			failed = true
		}
		if *stackDetails.State == project.ProjectConfig_State_DeployingFailed {
			// get the error message and add to errors
			errors = append(errors, fmt.Errorf("Failed with state %s", *stackDetails.State))
			failed = true
		}
		if *stackDetails.State == project.ProjectConfig_State_ValidatingFailed {
			// get the error message and add to errors
			errors = append(errors, fmt.Errorf("Failed with state %s", *stackDetails.State))
			failed = true
		}

		if failed {
			options.ProjectsLog(fmt.Sprintf("Stack Deploy Failed, current state: %s\n%s", *stackDetails.State, currentDeployStatus))
			return errors
		}
		if stackDetails.StateCode == nil {
			options.ProjectsLog("Stack state code is nil, skipping this time")
			time.Sleep(time.Duration(30) * time.Second)
			continue
		}
		if *stackDetails.State == project.ProjectConfig_State_Deployed &&
			*stackDetails.StateCode != project.ProjectConfig_StateCode_AwaitingMemberDeployment &&
			*stackDetails.StateCode != project.ProjectConfig_StateCode_AwaitingPrerequisite &&
			*stackDetails.StateCode != project.ProjectConfig_StateCode_AwaitingStackSetup &&
			*stackDetails.StateCode != project.ProjectConfig_StateCode_AwaitingValidation {
			deployComplete = true
			options.ProjectsLog(fmt.Sprintf("Stack Deployed Successfully, current state: %s and state code: %s", *stackDetails.State, *stackDetails.StateCode))
		} else {
			options.ProjectsLog(fmt.Sprintf("Stack is still deploying, current state: %s and state code: %s\n%s", *stackDetails.State, *stackDetails.StateCode, currentDeployStatus))
			time.Sleep(time.Duration(30) * time.Second)
		}
	}

	if !deployComplete {
		name := "Unknown"
		if options.currentStackConfig.Name != "" {
			name = options.currentStackConfig.Name
		}
		return []error{fmt.Errorf("deploy timeout for stack configuration %s", name)}
	}

	return nil
}

func (options *TestProjectsOptions) TriggerUnDeploy() error {
	if !options.SkipUndeploy {
		_, _, err := options.CloudInfoService.UndeployConfig(options.currentStackConfig)
		if err != nil {
			return err
		}
	}
	return nil
}

func (options *TestProjectsOptions) TriggerUnDeployAndWait() (errors []error) {

	if !options.SkipUndeploy {
		err := options.TriggerUnDeploy()
		if err != nil {
			return []error{err}
		}

		// check stack undeploy status
		// setup timeout
		undeployEndTime := time.Now().Add(time.Duration(options.DeployTimeoutMinutes) * time.Minute)
		undeployComplete := false
		failed := false
		for !undeployComplete && time.Now().Before(undeployEndTime) && !failed {

		}
	}
	return nil
}

// RunProjectsTest : Run the test for the projects service
// Creates a new project
// Adds a configuration
// Deploys the configuration
// Deletes the project
func (options *TestProjectsOptions) RunProjectsTest() error {
	if !options.SkipTestTearDown {
		// TODO: REMOVE AFTER TESTING
		defer func() {
			if r := recover(); r != nil {
				options.ProjectsLog(fmt.Sprintf("Recovered from panic: %v", r))
			}
			options.TestTearDown()
		}()
	}

	setupErr := options.testSetup()
	if !assert.NoError(options.Testing, setupErr) {
		return fmt.Errorf("test setup has failed:%w", setupErr)
	}

	// Create a new project
	options.ProjectsLog("Creating Test Project")
	if options.ProjectDestroyOnDelete == nil {
		options.ProjectDestroyOnDelete = core.BoolPtr(true)
	}
	if options.ProjectAutoDeploy == nil {
		options.ProjectAutoDeploy = core.BoolPtr(false)
	}
	if options.ProjectMonitoringEnabled == nil {
		options.ProjectMonitoringEnabled = core.BoolPtr(false)
	}
	options.currentProjectConfig = &cloudinfo.ProjectsConfig{
		Location:           options.ProjectLocation,
		ProjectName:        options.ProjectName,
		ProjectDescription: options.ProjectDescription,
		ResourceGroup:      options.ResourceGroup,
		DestroyOnDelete:    *options.ProjectDestroyOnDelete,
		MonitoringEnabled:  *options.ProjectMonitoringEnabled,
		AutoDeploy:         *options.ProjectAutoDeploy,
		Environments:       options.ProjectEnvironments,
		ComplianceProfile:  options.ProjectComplianceProfile,
	}
	prj, resp, err := options.CloudInfoService.CreateProjectFromConfig(options.currentProjectConfig)
	if assert.NoError(options.Testing, err) {
		if assert.Equal(options.Testing, 201, resp.StatusCode) {
			options.ProjectsLog(fmt.Sprintf("Created Test Project - %s", *prj.Definition.Name))
			options.currentProject = prj
			options.currentProjectConfig.ProjectID = *prj.ID
			if assert.NoError(options.Testing, options.ConfigureTestStack()) {
				if options.PreDeployHook != nil {
					options.ProjectsLog("Running PreDeployHook")
					hook_err := options.PreDeployHook(options)
					if hook_err != nil {
						return hook_err
					}
					options.ProjectsLog("Finished PreDeployHook")
				}
				// Deploy the configuration in parallel
				deployErrs := options.TriggerDeployAndWait()

				var finalError error

				if !assert.Empty(options.Testing, deployErrs) {
					// print all errors and return a single error
					for _, derr := range deployErrs {
						options.ProjectsLog(fmt.Sprintf("Error: %s", derr.Error()))
						finalError = fmt.Errorf("%w\n%s", finalError, derr)
					}
				} else {
					options.ProjectsLog("All configurations deployed successfully")
				}

				if options.PostDeployHook != nil {
					options.ProjectsLog("Running PostDeployHook")
					hook_err := options.PostDeployHook(options)
					if hook_err != nil {
						return hook_err
					}
					options.ProjectsLog("Finished PostDeployHook")
				}
				if finalError != nil {
					return finalError
				} else {
					return nil
				}
			}
		}
	}
	return nil
}

func (options *TestProjectsOptions) TestTearDown() {
	if options.currentProject == nil {
		options.ProjectsLog("No project to delete")
		return
	}
	if !options.SkipTestTearDown {
		if options.executeResourceTearDown() {
			//		trigger undeploy
		}

		if options.executeProjectTearDown() {
			// Delete the project

			// Check if any configurations are still validating, or deploying or undeploying, if so wait, timeout after 10 minutes
			// Set end time
			validationEndTime := time.Now().Add(time.Duration(options.DeployTimeoutMinutes) * time.Minute)

			for {
				// Get all configurations
				allConfigurations, cfgErr := options.CloudInfoService.GetProjectConfigs(*options.currentProject.ID)
				if cfgErr != nil {
					options.ProjectsLog("Failed to get configurations during project delete, attempting blind delete")
					break
				}

				// Check if any configuration is still in VALIDATING, DEPLOYING, or UNDEPLOYING state
				isAnyConfigInProcess := false
				for _, config := range allConfigurations {
					if *config.State == project.ProjectConfig_State_Validating || *config.State == project.ProjectConfig_State_Deploying || *config.State == project.ProjectConfig_State_Undeploying {
						isAnyConfigInProcess = true
						options.ProjectsLog(fmt.Sprintf("Configuration %s is still %s", *config.Definition.Name, *config.State))
						break
					}
				}

				// If no configuration is in VALIDATING, DEPLOYING, or UNDEPLOYING state, break the loop
				if !isAnyConfigInProcess {
					break
				}

				// If the time is greater than the timeout, return an error
				if time.Now().After(validationEndTime) {
					options.ProjectsLog("validation timeout for configurations")
				}

				// Sleep for 30 seconds before the next check
				time.Sleep(30 * time.Second)

			}

			options.Testing.Log("[PROJECTS] Deleting Test Project")
			if options.currentProject.ID != nil {
				_, resp, err := options.CloudInfoService.DeleteProject(*options.currentProject.ID)
				if assert.NoError(options.Testing, err) {
					assert.Equal(options.Testing, 202, resp.StatusCode)
					options.ProjectsLog("Deleted Test Project")
				}
			} else {
				options.ProjectsLog("No project to delete")
			}
		}
	}
}

// Function to determine if test resources should be destroyed
//
// Conditions for teardown:
// - The `SkipUndeploy` option is false (if true will override everything else)
// - Test failed and DO_NOT_DESTROY_ON_FAILURE was not set or false
// - Test completed with success (and `SkipUndeploy` was false)
func (options *TestProjectsOptions) executeResourceTearDown() bool {

	// assume we will execute
	execute := true

	// if skipundeploy is true, short circuit we are done
	if options.SkipUndeploy {
		execute = false
	}

	envVal, _ := os.LookupEnv("DO_NOT_DESTROY_ON_FAILURE")

	if options.Testing.Failed() && strings.ToLower(envVal) == "true" {
		execute = false
	}

	// if test failed and we are not executing, add a log line stating this
	if options.Testing.Failed() && !execute {
		options.Testing.Log("Terratest failed. Debug the Test and delete resources manually.")
	}

	return execute
}

// Function to determine if the project or stack steps (and their schematics workspaces) should be destroyed
//
// Conditions for teardown:
// - Test completed with success and `SkipProjectDelete` is false
func (options *TestProjectsOptions) executeProjectTearDown() bool {

	// assume we will execute
	execute := true

	// if SkipProjectDelete then short circuit we are done
	if options.SkipProjectDelete {
		execute = false
	}

	if options.Testing.Failed() {
		execute = false
	}

	// if test failed and we are not executing, add a log line stating this
	if options.Testing.Failed() && !execute {
		logger.Log(options.Testing, "Terratest failed. Debug the Test and delete the project manually.")
	}

	return execute
}

// Perform required steps for new test
func (options *TestProjectsOptions) testSetup() error {

	// change relative paths of configuration files to full path based on git root
	repoRoot, repoErr := common.GitRootPath(".")
	if repoErr != nil {
		repoRoot = "."
	}
	options.ConfigrationPath = path.Join(repoRoot, options.ConfigrationPath)
	options.StackConfigurationPath = path.Join(repoRoot, options.StackConfigurationPath)
	options.StackCatalogJsonPath = path.Join(repoRoot, options.StackCatalogJsonPath)

	// create new CloudInfoService if not supplied
	if options.CloudInfoService == nil {
		cloudInfoSvc, err := cloudinfo.NewCloudInfoServiceFromEnv("TF_VAR_ibmcloud_api_key", cloudinfo.CloudInfoServiceOptions{})
		if err != nil {
			return err
		}
		options.CloudInfoService = cloudInfoSvc
	}

	return nil
}

func (options *TestProjectsOptions) ProjectsLog(message string) {
	if options.ProjectName != "" {
		logPrefix := fmt.Sprintf("[PROJECTS - %s] ", options.ProjectName)
		logger.Log(options.Testing, fmt.Sprintf("%s %s", logPrefix, message))
	} else {
		logger.Log(options.Testing, fmt.Sprintf("[PROJECTS] %s", message))
	}
}
