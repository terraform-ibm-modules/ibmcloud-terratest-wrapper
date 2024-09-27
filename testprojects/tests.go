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
	options.currentStackConfig = &cloudinfo.ConfigDetails{
		ProjectID: *options.currentProject.ID,
		Inputs:    options.StackInputs,
	}
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

func (options *TestProjectsOptions) TriggerDeployAndWait() (errorList []error) {
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

	stackMembers, memErr := options.CloudInfoService.GetStackMembers(options.currentStackConfig)
	if memErr != nil {
		return []error{memErr}
	}
	totalMembers := len(stackMembers)

	for !deployComplete && time.Now().Before(deployEndTime) && !failed {
		options.ProjectsLog("Checking Stack Deploy Status")
		stackDetails, _, err := options.CloudInfoService.GetConfig(options.currentStackConfig)
		if err != nil {
			return []error{err}
		}
		if stackDetails == nil {
			return []error{fmt.Errorf("stackDetails is nil")}
		}

		currentMemberCount := 0
		attempt := 0
		// Sometimes not all members are returned by the api, so we need to retry. This is intermittent and infrequent
		for currentMemberCount != totalMembers && attempt < 5 {
			attempt++
			stackMembers, memErr = options.CloudInfoService.GetStackMembers(options.currentStackConfig)
			if memErr != nil {
				return []error{memErr}
			}
			currentMemberCount = len(stackMembers)
		}
		// If the stack is not fully deployed and no members are in a state that can be deployed then we have an error
		deployableState := false
		var memberStates []string

		// Track states and resync if needed
		syncErrs := TrackAndResyncState(options, stackDetails, stackMembers, memberStateStartTime, &mu)
		if len(syncErrs) > 0 {
			return syncErrs
		}

		currentDeployStatus := fmt.Sprintf("[STACK - %s] Current Deploy Status:\n", options.currentStackConfig.Name)
		// loop each member and check state
		for _, member := range stackMembers {
			if member.ID == nil {
				return []error{fmt.Errorf("member ID is nil")}
			}
			memberName, nameErr := options.CloudInfoService.LookupMemberNameByID(stackDetails, *member.ID)
			if nameErr != nil {
				memberName = fmt.Sprintf("Unknown name, ID: %s", *member.ID)
			}

			if member.State == nil {
				memberStates = append(memberStates, fmt.Sprintf(" - member: %s state is nil, skipping this time", memberName))
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
				memberStates = append(memberStates, fmt.Sprintf(" - member: %s current state: %s", memberName, *member.State))
			} else {
				memberStates = append(memberStates, fmt.Sprintf(" - member: %s current state: %s and state code: %s", memberName, *member.State, stateCode))
				if stateCode == "Unknown" {
					// assume blip and mark deployable
					deployableState = true
				}
			}

			if member.StateCode != nil && (*member.StateCode == project.ProjectConfig_StateCode_AwaitingMemberDeployment ||
				*member.StateCode == project.ProjectConfig_StateCode_AwaitingValidation) {
				deployableState = true
			}
			if *member.State == project.ProjectConfig_State_Validating {
				currentDeployStatus = fmt.Sprintf("%s - member: %s is still validating\n", currentDeployStatus, memberName)
				deployableState = true
			} else if *member.State == project.ProjectConfig_State_Deploying {
				currentDeployStatus = fmt.Sprintf("%s - member: %s is still deploying\n", currentDeployStatus, memberName)
				deployableState = true
			} else if *member.State == project.ProjectConfig_State_Deployed {
				currentDeployStatus = fmt.Sprintf("%s - member: %s is deployed\n", currentDeployStatus, memberName)
			} else if *member.State == project.ProjectConfig_State_Approved {
				currentDeployStatus = fmt.Sprintf("%s - member: %s is approved\n", currentDeployStatus, memberName)
			} else if member.StateCode != nil && *member.StateCode == project.ProjectConfig_StateCode_AwaitingPrerequisite {
				currentDeployStatus = fmt.Sprintf("%s - member: %s is awaiting prerequisite\n", currentDeployStatus, memberName)
			} else if *member.State == project.ProjectConfig_State_ValidatingFailed {
				// fail deployment and get the error
				deployableState = false
				failed = true
				logMessage, terraLogs := options.CloudInfoService.GetSchematicsJobLogsForMember(member, memberName)
				options.ProjectsLog(terraLogs)
				errorList = append(errorList, fmt.Errorf(logMessage))
			} else if *member.State == project.ProjectConfig_State_DeployingFailed {
				deployableState = false
				failed = true
				logMessage, terraLogs := options.CloudInfoService.GetSchematicsJobLogsForMember(member, memberName)
				options.ProjectsLog(terraLogs)
				errorList = append(errorList, fmt.Errorf(logMessage))
			} else if *member.State == project.ProjectConfig_State_Draft {
				options.ProjectsLog(fmt.Sprintf("(member: %s state: %s stateCode: %s) Something unexpected happened on the backend attempting re-trigger deploy", memberName, *member.State, stateCode))
				// Something happened re-trigger deploy
				trigErrs := options.TriggerDeploy()
				if trigErrs != nil {
					var terr *core.SDKProblem
					errors.As(trigErrs, &terr)
					if terr.IBMProblem.Summary == "Not Modified" {
						// continue assume still deploying
						options.ProjectsLog(fmt.Sprintf("(member: %s state: %s stateCode: %s) Trigger Deploy returned Not Modified, continuing", memberName, *member.State, stateCode))
						currentDeployStatus = fmt.Sprintf("%s - member: %s is in state %s, not triggered, no changes, continuing assuming still deploying\n", currentDeployStatus, memberName, *member.State)
					} else {
						options.ProjectsLog(fmt.Sprintf("(member: %s state: %s stateCode: %s) Something unexpected happened on the backend attempting re-trigger deploy failed, continuing assuming still deploying\n%s", memberName, *member.State, stateCode, trigErrs))
						currentDeployStatus = fmt.Sprintf("%s - member: %s is in state %s, error triggering, continuing assuming still deploying\n", currentDeployStatus, memberName, *member.State)
					}
				} else {
					currentDeployStatus = fmt.Sprintf("%s - member: %s is in state %s, attempting to re-trigger deploy\n", currentDeployStatus, memberName, *member.State)

				}
			} else {
				if member.State == nil {
					currentDeployStatus = fmt.Sprintf("%s - member: %s is in an unknown state\n", currentDeployStatus, memberName)
				} else {
					currentDeployStatus = fmt.Sprintf("%s - member: %s is in state %s\n", currentDeployStatus, memberName, *member.State)
				}
			}
		}

		if !deployableState {
			// check all members are deployed
			allMembersDeployed := true
			for _, member := range stackMembers {
				if *member.State != project.ProjectConfig_State_Deployed {
					allMembersDeployed = false
					break
				}
			}
			if allMembersDeployed {
				deployComplete = true
			} else if !failed {
				// TODO: pause 30 second then check the member states one last time
				// just incase there was a delay in updating the state
				// Move members deployable check to its own function
				time.Sleep(30 * time.Second)

				var errorMessage strings.Builder
				membersLatest, latestMemErr := options.CloudInfoService.GetStackMembers(options.currentStackConfig)
				if latestMemErr == nil {
					options.ProjectsLog("Checking if stuck...")

					memberStates = []string{}
					memberCount := len(membersLatest)
					deployableMembers := 0
					for _, member := range membersLatest {
						memberName, nameErr := options.CloudInfoService.LookupMemberNameByID(stackDetails, *member.ID)
						if nameErr != nil {
							memberName = fmt.Sprintf("Unknown name, ID: %s", *member.ID)
						}
						stateCode := "Unknown"
						if member.StateCode != nil {
							stateCode = *member.StateCode
						}
						if cloudinfo.ProjectsMemberIsDeploying(member) || (*member.State == project.ProjectConfig_State_Draft && stateCode != project.ProjectConfig_StateCode_AwaitingPrerequisite) {
							memberStates = append(memberStates, fmt.Sprintf(" - member: %s current state: %s and state code: %s", memberName, *member.State, stateCode))
							deployableMembers++
						} else if *member.State == project.ProjectConfig_State_Deployed {
							memberStates = append(memberStates, fmt.Sprintf(" - member: %s current state: %s, current state code:%s", memberName, *member.State, stateCode))
							deployableMembers++
						} else {
							memberStates = append(memberStates, fmt.Sprintf(" - member: %s current state: %s, current state code:%s", memberName, *member.State, stateCode))
						}
					}
					if memberCount == deployableMembers {
						var stackStatusMessage strings.Builder
						stackStatusMessage.WriteString(fmt.Sprintf("Stack is still deploying, current state: %s and state code: %s", *stackDetails.State, *stackDetails.StateCode))
						for _, memState := range memberStates {
							stackStatusMessage.WriteString(memState)
						}

						options.ProjectsLog(stackStatusMessage.String())
						time.Sleep(time.Duration(30) * time.Second)
						continue
					}
				}
				errorMessage.WriteString("Stack stuck in undeployable state:")
				for _, memberState := range memberStates {
					errorMessage.WriteString(fmt.Sprintf("\n%s", memberState))
				}

				stackRefStruc, refErr := common.CreateStackRefStruct(options.currentStack, stackMembers)
				if refErr == nil {
					common.ResolveReferences(stackRefStruc)
					unResolvedRefs := common.GetAllUnresolvedRefsAsString(stackRefStruc)
					if unResolvedRefs != "" {
						errorMessage.WriteString(fmt.Sprintf("\nUnresolved References:\n%s", unResolvedRefs))
					}
				}
				errorList = append(errorList, fmt.Errorf(errorMessage.String()))
				failed = true
			}

		}
		// fail fast if any error states
		if stackDetails.State == nil {
			errorList = append(errorList, fmt.Errorf("stackDetails state is nil"))
			return errorList
		}
		if *stackDetails.State == project.ProjectConfig_State_ApplyFailed {
			// get the error message and add to errorList
			errorList = append(errorList, fmt.Errorf("failed with state %s", *stackDetails.State))

			failed = true
		}
		if *stackDetails.State == project.ProjectConfig_State_DeployingFailed {
			// get the error message and add to errorList
			errorList = append(errorList, fmt.Errorf("failed with state %s", *stackDetails.State))
			failed = true
		}
		if *stackDetails.State == project.ProjectConfig_State_ValidatingFailed {
			// get the error message and add to errorList
			errorList = append(errorList, fmt.Errorf("failed with state %s", *stackDetails.State))
			failed = true
		}

		if failed {
			options.ProjectsLog(fmt.Sprintf("Stack Deploy Failed, current state: %s\n%s", *stackDetails.State, currentDeployStatus))
			return errorList
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

	// print final state of the stack
	stackDetails, _, err := options.CloudInfoService.GetConfig(options.currentStackConfig)
	if err != nil {
		return []error{err}
	}
	if stackDetails == nil {
		return []error{fmt.Errorf("stackDetails is nil")}
	}
	stateCode := "Unknown"
	if stackDetails.StateCode != nil {
		stateCode = *stackDetails.StateCode
	}
	options.ProjectsLog(fmt.Sprintf("Stacks final state: %s, state code: %s", *stackDetails.State, stateCode))

	return nil
}

func (options *TestProjectsOptions) TriggerUnDeploy() []error {
	if !options.SkipUndeploy {
		// Initialize the map to store the start time for each member's state
		memberStateStartTime := make(map[string]time.Time)
		// Mutex to ensure safe concurrent access to the map
		var mu sync.Mutex

		readyForUndeploy := false
		timeoutEndTime := time.Now().Add(time.Duration(options.DeployTimeoutMinutes) * time.Minute)

		// while not ready for undeploy and timeout not reached, keep checking
		for !readyForUndeploy && time.Now().Before(timeoutEndTime) {
			readyForUndeploy = true

			// Fetch the latest state of the members
			stackDetails, _, err := options.CloudInfoService.GetConfig(options.currentStackConfig)
			if err != nil {
				return []error{err}
			}
			if stackDetails == nil {
				return []error{fmt.Errorf("stackDetails is nil")}
			}

			stackMembers, err := options.CloudInfoService.GetStackMembers(options.currentStackConfig)
			if err != nil {
				return []error{err}
			}

			syncErrs := TrackAndResyncState(options, stackDetails, stackMembers, memberStateStartTime, &mu)
			if len(syncErrs) > 0 {
				return syncErrs
			}
			stateCode := "Unknown"
			if stackDetails.StateCode != nil {
				stateCode = *stackDetails.StateCode
			}
			// first check the stack state if it is not in a deploying or undeploying state then we can trigger undeploy
			if stateCode != project.ProjectConfig_StateCode_AwaitingMemberDeployment &&
				*stackDetails.State != project.ProjectConfig_State_Deploying &&
				*stackDetails.State != project.ProjectConfig_State_Undeploying {
				readyForUndeploy = true
				options.ProjectsLog(fmt.Sprintf("Stack is in state %s with stateCode %s, ready for undeploy", *stackDetails.State, stateCode))
			} else {
				options.ProjectsLog(fmt.Sprintf("Stack is in state %s with stateCode %s, waiting for all members to complete", *stackDetails.State, stateCode))
			}
			// then double check if any configuration is still in VALIDATING, DEPLOYING, or UNDEPLOYING state
			for _, member := range stackMembers {
				if *member.State == project.ProjectConfig_State_Validating || *member.State == project.ProjectConfig_State_Deploying || *member.State == project.ProjectConfig_State_Undeploying {
					readyForUndeploy = false
					memberName, err := options.CloudInfoService.LookupMemberNameByID(stackDetails, *member.ID)
					if err != nil {
						memberName = fmt.Sprintf("Unknown name, ID: %s", *member.ID)
					}
					options.ProjectsLog(fmt.Sprintf("Member %s is still in state %s, waiting for all members to complete", memberName, *member.State))
					time.Sleep(time.Duration(30) * time.Second)
				}
			}
		}

		if !readyForUndeploy {
			return []error{fmt.Errorf("timeout waiting for all members to complete, could not trigger undeploy")}
		}
		_, _, errUndep := options.CloudInfoService.UndeployConfig(options.currentStackConfig)
		if errUndep != nil {
			if errUndep.Error() == "Not Modified" {
				options.ProjectsLog("Nothing to undeploy")
				return nil
			}
			return []error{errUndep}
		}
	}
	return nil
}
func (options *TestProjectsOptions) TriggerUnDeployAndWait() (errorList []error) {
	if !options.SkipUndeploy {
		err := options.TriggerUnDeploy()
		if err != nil {
			return err
		}

		undeployEndTime := time.Now().Add(time.Duration(options.DeployTimeoutMinutes) * time.Minute)
		undeployComplete := false
		failed := false

		// Initialize the map to store the start time for each member's state
		memberStateStartTime := make(map[string]time.Time)
		// Mutex to ensure safe concurrent access to the map
		var mu sync.Mutex
		stackMembers, memErr := options.CloudInfoService.GetStackMembers(options.currentStackConfig)
		if memErr != nil {
			return []error{memErr}
		}
		totalMembers := len(stackMembers)
		var undeployedCount int
		for !undeployComplete && time.Now().Before(undeployEndTime) && !failed {
			options.ProjectsLog("Checking Stack Undeploy Status")
			stackDetails, _, err := options.CloudInfoService.GetConfig(options.currentStackConfig)
			if err != nil {
				return []error{err}
			}
			if stackDetails == nil {
				return []error{fmt.Errorf("stackDetails is nil")}
			}

			currentMemberCount := 0
			attempt := 0
			// Sometimes not all members are returned by the API, so we need to retry. This is intermittent and infrequent
			for currentMemberCount != totalMembers && attempt < 5 {
				attempt++
				stackMembers, memErr = options.CloudInfoService.GetStackMembers(options.currentStackConfig)
				if memErr != nil {
					return []error{memErr}
				}
				currentMemberCount = len(stackMembers)
			}

			if currentMemberCount != totalMembers {
				return []error{fmt.Errorf("expected %d members, but got %d", totalMembers, currentMemberCount)}
			}

			syncErrs := TrackAndResyncState(options, stackDetails, stackMembers, memberStateStartTime, &mu)
			if len(syncErrs) > 0 {
				return syncErrs
			}

			undeployableState := true
			var memberStates []string
			currentUndeployStatus := fmt.Sprintf("[STACK - %s] Current Undeploy Status:\n", options.currentStackConfig.Name)
			undeployedCount = 0

			for _, member := range stackMembers {
				if member.ID == nil {
					return []error{fmt.Errorf("member ID is nil")}
				}
				memberName, err := options.CloudInfoService.LookupMemberNameByID(stackDetails, *member.ID)
				if err != nil {
					memberName = fmt.Sprintf("Unknown name, ID: %s", *member.ID)
				}

				if member.State == nil {
					memberStates = append(memberStates, fmt.Sprintf(" - member: %s state is nil, skipping this time", memberName))
					undeployableState = false
					continue
				}

				if *member.State == project.ProjectConfig_State_Undeploying {
					memberStates = append(memberStates, fmt.Sprintf(" - member: %s current state: %s", memberName, *member.State))
				} else if *member.State == project.ProjectConfig_State_UndeployingFailed {
					memberStates = append(memberStates, fmt.Sprintf(" - member: %s current state: %s", memberName, *member.State))
					undeployableState = false
					failed = true
					logMessage, terraLogs := options.CloudInfoService.GetSchematicsJobLogsForMember(member, memberName)
					options.ProjectsLog(terraLogs)
					errorList = append(errorList, fmt.Errorf("(%s) failed Undeployment\n%s", memberName, logMessage))
				} else if cloudinfo.ProjectsMemberIsUndeployed(member) {
					memberStates = append(memberStates, fmt.Sprintf(" - member: %s current state: %s", memberName, *member.State))
					undeployedCount++
				} else {
					memberStates = append(memberStates, fmt.Sprintf(" - member: %s current state: %s", memberName, *member.State))
					undeployableState = false
				}
			}

			if undeployableState && undeployedCount == totalMembers {
				undeployComplete = true
			} else {
				var stateCode string
				if stackDetails.StateCode == nil {
					stateCode = "Unknown"
				} else {
					stateCode = *stackDetails.StateCode
				}
				if stateCode == project.ProjectConfig_State_UndeployingFailed {
					undeployableState = false
					failed = true
					errorList = append(errorList, fmt.Errorf("undeploy stack failed with state code %s", stateCode))
				} else {
					options.ProjectsLog(fmt.Sprintf("Stack is still undeploying, current state: %s and state code: %s\n%s", *stackDetails.State, stateCode, currentUndeployStatus+strings.Join(memberStates, "\n")))
					time.Sleep(30 * time.Second)
				}
			}
		}

		if !undeployComplete {
			if time.Now().After(undeployEndTime) {
				return []error{fmt.Errorf("undeploy timeout for stack configuration %s", options.currentStackConfig.Name)}
			} else if failed {
				return errorList
			} else {
				return []error{fmt.Errorf("undeploy incomplete for unknown reasons for stack configuration %s", options.currentStackConfig.Name)}
			}
		}
		// Log the final undeploy status
		// print final state of the stack
		stackDetails, _, cfgErr := options.CloudInfoService.GetConfig(options.currentStackConfig)
		if err != nil {
			return []error{cfgErr}
		}
		if stackDetails == nil {
			return []error{fmt.Errorf("stackDetails is nil")}
		}
		stateCode := "Unknown"
		if stackDetails.StateCode != nil {
			stateCode = *stackDetails.StateCode
		}
		options.ProjectsLog(fmt.Sprintf("Stacks final state: %s, state code: %s", *stackDetails.State, stateCode))
		options.ProjectsLog(fmt.Sprintf("%d/%d members undeployed", totalMembers, undeployedCount))
		options.ProjectsLog("Undeploy completed successfully")
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
		// ensure we always run the test tear down, even if a panic occurs
		defer func() {
			if r := recover(); r != nil {
				options.ProjectsLog(fmt.Sprintf("Recovered from panic: %v", r))
			}
			options.TestTearDown()
		}()
	}

	setupErr := options.testSetup()
	if !assert.NoError(options.Testing, setupErr) {
		options.Testing.Fail()
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
	}
	prj, resp, err := options.CloudInfoService.CreateProjectFromConfig(options.currentProjectConfig)
	if assert.NoError(options.Testing, err) {
		if assert.Equal(options.Testing, 201, resp.StatusCode) {
			options.ProjectsLog(fmt.Sprintf("Created Test Project - %s", *prj.Definition.Name))
			// https://cloud.ibm.com/projects/a1316ed6-76de-418e-bb9a-e7ed05aa7834
			// print link to project
			options.ProjectsLog(fmt.Sprintf("Project URL: %s", fmt.Sprintf("https://cloud.ibm.com/projects/%s", *prj.ID)))
			options.currentProject = prj
			options.currentProjectConfig.ProjectID = *prj.ID
			if assert.NoError(options.Testing, options.ConfigureTestStack()) {
				options.ProjectsLog(fmt.Sprintf("Configured Test Stack - %s \n- Project ID: %s \n- Config ID: %s", *prj.Definition.Name, *prj.ID, *options.currentStack.Configuration.ID))
				if options.PreDeployHook != nil {
					options.ProjectsLog("Running PreDeployHook")
					hookErr := options.PreDeployHook(options)
					if hookErr != nil {
						options.Testing.Fail()
						return hookErr
					}
					options.ProjectsLog("Finished PreDeployHook")
				}
				// Deploy the configuration in parallel
				deployErrs := options.TriggerDeployAndWait()

				var finalError error

				if len(deployErrs) > 0 {
					// print all errors and return a single error
					for _, derr := range deployErrs {
						options.ProjectsLog(fmt.Sprintf("Error: %s", derr.Error()))
						if finalError == nil {
							finalError = derr
						} else {
							finalError = fmt.Errorf("%w\n%s", finalError, derr)
						}
					}
				} else {
					options.ProjectsLog("All configurations deployed successfully")
				}

				if options.PostDeployHook != nil {
					options.ProjectsLog("Running PostDeployHook")
					hookErr := options.PostDeployHook(options)
					if hookErr != nil {
						options.Testing.Fail()
						return hookErr
					}
					options.ProjectsLog("Finished PostDeployHook")
				}
				if finalError != nil {
					options.Testing.Fail()
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

			// Trigger undeploy and wait for completion
			options.ProjectsLog("Triggering Undeploy and waiting for completion")
			undeployErrors := options.TriggerUnDeployAndWait()
			if len(undeployErrors) > 0 {
				for _, err := range undeployErrors {
					options.ProjectsLog(fmt.Sprintf("Failed to undeploy: %s", err))
					options.Testing.Fail()
				}
			}

		}
		if options.executeProjectTearDown() {
			// Wait until no pipeline actions are running or timeout is reached
			options.ProjectsLog("Checking all pipeline actions are complete")
			timeout := time.Now().Add(time.Duration(options.DeployTimeoutMinutes) * time.Minute)
			for {
				running, err := options.CloudInfoService.ArePipelineActionsRunning(options.currentStackConfig)
				if err != nil {
					options.ProjectsLog(fmt.Sprintf("Error checking pipeline actions: %s", err))
					options.Testing.Fail()
					return
				}
				if !running || time.Now().After(timeout) {
					break
				}
				options.ProjectsLog("Pipeline actions are still running, waiting...")
				time.Sleep(30 * time.Second)
			}

			// Check if timeout was reached
			if time.Now().After(timeout) {
				options.ProjectsLog("Timeout reached while waiting for pipeline actions to complete")
				options.Testing.Fail()
				return
			}
			options.ProjectsLog("All pipeline actions are complete")
			// Delete the project
			options.ProjectsLog("Deleting Test Project")
			if options.currentProject.ID != nil {
				_, resp, err := options.CloudInfoService.DeleteProject(*options.currentProject.ID)
				if assert.NoError(options.Testing, err) {
					if assert.Equal(options.Testing, 202, resp.StatusCode) {
						options.ProjectsLog("Deleted Test Project")
					} else {
						options.ProjectsLog(fmt.Sprintf("Failed to delete Test Project, response code: %d", resp.StatusCode))
					}
				} else {
					options.ProjectsLog(fmt.Sprintf("Error deleting Test Project: %s", err))
				}
			} else {
				options.ProjectsLog("No project ID found to delete")
			}
		} else {
			options.ProjectsLog(fmt.Sprintf("Project URL: %s", fmt.Sprintf("https://cloud.ibm.com/projects/%s", *options.currentProject.ID)))
		}
	} else {
		options.ProjectsLog(fmt.Sprintf("Project URL: %s", fmt.Sprintf("https://cloud.ibm.com/projects/%s", *options.currentProject.ID)))
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

	// dont teardown if there is nothing to teardown
	if options.currentStackConfig == nil || options.currentStackConfig.ConfigID == "" {
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
	// skip teardown if no project was created
	if options.currentProject == nil {
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
		options.Testing.Log(fmt.Sprintf("%s %s", logPrefix, message))
	} else {
		options.Testing.Log(fmt.Sprintf("[PROJECTS] %s", message))
	}
}

// TrackAndResyncState tracks the state of members and triggers a resync if a member is stuck in a state for more than the options.StackAutoSyncInterval.
func TrackAndResyncState(
	options *TestProjectsOptions,
	stackDetails *project.ProjectConfig,
	stackMembers []*project.ProjectConfig,
	memberStateStartTime map[string]time.Time,
	mu *sync.Mutex,
) (errors []error) {

	if options.StackAutoSync {
		for _, member := range stackMembers {
			if member.ID == nil {
				return []error{fmt.Errorf("member ID is nil")}
			}
			if member.State != nil && *member.State == project.ProjectConfig_State_Deployed {
				//options.ProjectsLog(fmt.Sprintf("Member %s is already deployed, skipping sync tracking", *member.ID))
				continue
			}
			stateCode := "Unknown"
			if member.StateCode != nil {
				stateCode = *member.StateCode
			}

			// if state draft and awaiting prerequisite, do not trigger sync
			if *member.State == project.ProjectConfig_State_Draft && stateCode == project.ProjectConfig_StateCode_AwaitingPrerequisite {
				continue
			}

			memberName, err := options.CloudInfoService.LookupMemberNameByID(stackDetails, *member.ID)
			if err != nil {
				memberName = fmt.Sprintf("Unknown name, ID: %s", *member.ID)
			}

			// Lock the mutex before accessing the map
			mu.Lock()
			// Check if the member's ID is in the map
			// If it is, check if the member has been in the same state for more than options.StackAutoSyncInterval minutes
			// If it has, trigger SyncConfig and reset the start time
			// If it is not, store the start time for the member's current state
			if startTime, exists := memberStateStartTime[*member.ID]; exists {
				// Check if the member has been in the same state for more than options.StackAutoSyncInterval minutes
				if time.Since(startTime) > time.Duration(options.StackAutoSyncInterval)*time.Minute {
					if member.State == nil {
						mu.Unlock()
						return []error{fmt.Errorf("member state is nil")}
					}
					options.ProjectsLog(fmt.Sprintf("Member %s stuck in state %s for more than %d minutes, triggering SyncConfig", memberName, *member.State, options.StackAutoSyncInterval))
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
		}
		return errors
	}
	return errors
}
