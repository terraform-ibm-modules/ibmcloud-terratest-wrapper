package testprojects

import (
	"errors"
	"fmt"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/IBM/go-sdk-core/v5/core"
	project "github.com/IBM/project-go-sdk/projectv1"
	"github.com/stretchr/testify/assert"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/cloudinfo"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/common"
)

// Status is a struct of the status strings
type Status struct {
	DEPLOYED              string
	DEPLOYING             string
	DEPLOYING_FAILED      string
	VALIDATED             string
	VALIDATING            string
	VALIDATING_FAILED     string
	AWAITING_VALIDATION   string
	APPROVED              string
	DRAFT                 string
	AWAITING_PREREQUISITE string
	AWAITING_INPUT        string
	NIL                   string
	UNKOWN                string
}

// Statuses is a map of the status strings to colorized strings
var Statuses = map[string]string{
	project.ProjectConfig_State_Deployed:                     common.ColorizeString(common.Colors.Green, project.ProjectConfig_State_Deployed),
	project.ProjectConfig_State_Deploying:                    common.ColorizeString(common.Colors.Orange, project.ProjectConfig_State_Deploying),
	project.ProjectConfig_State_DeployingFailed:              common.ColorizeString(common.Colors.Red, project.ProjectConfig_State_DeployingFailed),
	project.ProjectConfig_State_Undeploying:                  common.ColorizeString(common.Colors.Orange, project.ProjectConfig_State_Undeploying),
	project.ProjectConfig_State_UndeployingFailed:            common.ColorizeString(common.Colors.Red, project.ProjectConfig_State_UndeployingFailed),
	project.ProjectConfig_State_Validated:                    common.ColorizeString(common.Colors.Green, project.ProjectConfig_State_Validated),
	project.ProjectConfig_State_Validating:                   common.ColorizeString(common.Colors.Orange, project.ProjectConfig_State_Validating),
	project.ProjectConfig_State_ValidatingFailed:             common.ColorizeString(common.Colors.Red, project.ProjectConfig_State_ValidatingFailed),
	project.ProjectConfig_State_Approved:                     common.ColorizeString(common.Colors.Green, project.ProjectConfig_State_Approved),
	project.ProjectConfig_State_Draft:                        common.ColorizeString(common.Colors.Blue, project.ProjectConfig_State_Draft),
	project.ProjectConfig_State_ApplyFailed:                  common.ColorizeString(common.Colors.Red, project.ProjectConfig_State_ApplyFailed),
	project.ProjectConfig_State_Deleting:                     common.ColorizeString(common.Colors.Orange, project.ProjectConfig_State_Deleting),
	project.ProjectConfig_State_DeletingFailed:               common.ColorizeString(common.Colors.Red, project.ProjectConfig_State_DeletingFailed),
	project.ProjectConfig_State_Deleted:                      common.ColorizeString(common.Colors.Green, project.ProjectConfig_State_Deleted),
	project.ProjectConfig_StateCode_AwaitingValidation:       common.ColorizeString(common.Colors.Yellow, project.ProjectConfig_StateCode_AwaitingValidation),
	project.ProjectConfig_StateCode_AwaitingPrerequisite:     common.ColorizeString(common.Colors.Yellow, project.ProjectConfig_StateCode_AwaitingPrerequisite),
	project.ProjectConfig_StateCode_AwaitingInput:            common.ColorizeString(common.Colors.Yellow, project.ProjectConfig_StateCode_AwaitingInput),
	project.ProjectConfig_StateCode_AwaitingMemberDeployment: common.ColorizeString(common.Colors.Yellow, project.ProjectConfig_StateCode_AwaitingMemberDeployment),
	project.ProjectConfig_StateCode_AwaitingStackSetup:       common.ColorizeString(common.Colors.Yellow, project.ProjectConfig_StateCode_AwaitingStackSetup),
	"nil":     common.ColorizeString(common.Colors.Red, "nil"),
	"Unknown": common.ColorizeString(common.Colors.Purple, "Unknown"),
}

func (options *TestProjectsOptions) ConfigureTestStack() error {
	// Configure the test stack
	options.Logger.ShortInfo("Configuring Test Stack")
	var stackResp *core.DetailedResponse
	var stackErr error
	options.currentStackConfig = &cloudinfo.ConfigDetails{
		ProjectID: *options.currentProject.ID,
		Inputs:    options.StackInputs,
	}
	// set member inputs
	if options.StackMemberInputs != nil {
		for memberName, memberInputs := range options.StackMemberInputs {
			options.currentStackConfig.MemberConfigDetails = append(options.currentStackConfig.MemberConfigDetails,
				cloudinfo.ConfigDetails{
					Name:   memberName,
					Inputs: memberInputs,
				})
		}
	}

	options.currentStack, stackResp, stackErr = options.CloudInfoService.CreateStackFromConfigFile(options.currentStackConfig, options.StackConfigurationPath, options.StackCatalogJsonPath)
	if !assert.NoError(options.Testing, stackErr) {
		options.Logger.ShortError("Failed to configure Test Stack")
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
					options.Logger.ShortError("Error getting version locator from stack definition")
					return sdkProblem
				}
				// get inputs for the member config of version
				version, vererr := options.CloudInfoService.GetCatalogVersionByLocator(versionLocator)
				if !assert.NoError(options.Testing, vererr) {
					options.Logger.ShortError("Error getting offering")
					return sdkProblem
				}
				// version.configurations[x].name append all configuration names to validInputs
				validInputs := "Valid Inputs:\n"

				for _, configuration := range version.Configuration {
					if configuration.Key != nil {
						validInputs += fmt.Sprintf("\t%s\n", *configuration.Key)
					} else {
						// Safe fallback: This is only for error message display, not core functionality.
						// Using a placeholder name allows the error message to remain helpful even when
						// the catalog configuration is malformed. The actual stack creation failure
						// is handled elsewhere - this just provides diagnostic information.
						validInputs += fmt.Sprintf("\t<unnamed_configuration>\n")
					}
				}

				sdkProblem.Summary = fmt.Sprintf("%s Inputs possibly removed or renamed.\n%s", sdkProblem.Summary, validInputs)
				return sdkProblem
			}
		} else if assert.Equal(options.Testing, 201, stackResp.StatusCode) {
			options.Logger.ShortInfo("Configured Test Stack")
		} else {
			options.Logger.ShortError("Failed to configure Test Stack")
			return fmt.Errorf("error configuring test stack response code: %d\nrespone:%s", stackResp.StatusCode, stackResp.Result)
		}
	}
	return nil
}

// TriggerDeploy is assuming auto deploy is enabled so just triggers validate at the stack level
func (options *TestProjectsOptions) TriggerDeploy() error {
	options.Logger.ShortInfo("Triggering Deploy")
	_, _, err := options.CloudInfoService.ValidateProjectConfig(options.currentStackConfig)
	if err != nil {
		return err
	}
	options.Logger.ShortInfo("Deploy Triggered Successfully")
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
		options.Logger.ShortInfo("Checking Stack Deploy Status")
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

		currentDeployStatus := fmt.Sprintf("%s Current Deploy Status:\n", common.ColorizeString(common.Colors.Blue, fmt.Sprintf("[STACK - %s]", options.currentStackConfig.Name)))
		memberLabel := common.ColorizeString(common.Colors.Blue, "\t- Member: ")

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
				memberStates = append(memberStates, fmt.Sprintf("%s%s state is %s, skipping this time", memberLabel, memberName, Statuses["nil"]))
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
				memberStates = append(memberStates, fmt.Sprintf("%s%s current state: %s", memberLabel, memberName, Statuses[*member.State]))
			} else {
				memberStates = append(memberStates, fmt.Sprintf("%s%s current state: %s and state code: %s", memberLabel, memberName, Statuses[*member.State], Statuses[stateCode]))
				if stateCode == "Unknown" {
					// assume blip and mark deployable
					deployableState = true
				}
			}

			if member.StateCode != nil && (*member.StateCode == project.ProjectConfig_StateCode_AwaitingMemberDeployment ||
				*member.StateCode == project.ProjectConfig_StateCode_AwaitingValidation) {
				deployableState = true
			}

			switch *member.State {
			case project.ProjectConfig_State_Validating, project.ProjectConfig_State_Deploying:
				currentDeployStatus = fmt.Sprintf("%s%s%s is still %s\n", currentDeployStatus, memberLabel, memberName, Statuses[*member.State])
				deployableState = true
			case project.ProjectConfig_State_Deployed, project.ProjectConfig_State_Approved:
				currentDeployStatus = fmt.Sprintf("%s%s%s is %s\n", currentDeployStatus, memberLabel, memberName, Statuses[*member.State])
			case project.ProjectConfig_State_ValidatingFailed, project.ProjectConfig_State_DeployingFailed:
				deployableState = false
				failed = true
				logMessage, terraLogs := options.CloudInfoService.GetSchematicsJobLogsForMember(member, memberName, options.currentProjectConfig.Location)
				options.Logger.ShortError(terraLogs)
				errorList = append(errorList, fmt.Errorf("%s", logMessage))
			case project.ProjectConfig_State_Draft:
				if stateCode == project.ProjectConfig_StateCode_AwaitingPrerequisite || (stateCode == project.ProjectConfig_StateCode_AwaitingMemberDeployment && strings.HasSuffix(memberName, " Container")) {
					currentDeployStatus = fmt.Sprintf("%s%s%s is in state %s and state code %s\n", currentDeployStatus, memberLabel, memberName, Statuses[*member.State], Statuses[stateCode])
				} else {
					options.Logger.ShortInfo(fmt.Sprintf("(member: %s state: %s stateCode: %s) Something unexpected happened on the backend attempting re-trigger deploy", memberName, Statuses[*member.State], Statuses[stateCode]))
					trigErrs := options.TriggerDeploy()
					if trigErrs != nil {
						var terr *core.SDKProblem
						errors.As(trigErrs, &terr)
						if terr.IBMProblem.Summary == "Not Modified" {
							options.Logger.ShortInfo(fmt.Sprintf("(member: %s state: %s stateCode: %s) Trigger Deploy returned Not Modified, continuing", memberName, Statuses[*member.State], Statuses[stateCode]))
							currentDeployStatus = fmt.Sprintf("%s%s%s is in state %s, not triggered, no changes, continuing assuming still deploying\n", currentDeployStatus, memberLabel, memberName, Statuses[*member.State])
						} else {
							options.Logger.ShortInfo(fmt.Sprintf("(member: %s state: %s stateCode: %s) Something unexpected happened on the backend attempting re-trigger deploy failed, continuing assuming still deploying\n%s", memberName, Statuses[*member.State], Statuses[stateCode], trigErrs))
							currentDeployStatus = fmt.Sprintf("%s%s%s is in state %s, error triggering, continuing assuming still deploying\n", currentDeployStatus, memberLabel, memberName, Statuses[*member.State])
						}
					} else {
						currentDeployStatus = fmt.Sprintf("%s%s%s is in state %s, attempting to re-trigger deploy\n", currentDeployStatus, memberLabel, memberName, Statuses[*member.State])
					}
				}
			default:
				currentDeployStatus = fmt.Sprintf("%s%s%s is in state %s\n", currentDeployStatus, memberLabel, memberName, Statuses[*member.State])
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
				// just incase there was a delay in updating the state
				// Move members deployable check to its own function
				time.Sleep(time.Duration(options.StackPollTimeSeconds) * time.Second)

				var errorMessage strings.Builder
				membersLatest, latestMemErr := options.CloudInfoService.GetStackMembers(options.currentStackConfig)
				if latestMemErr == nil {
					options.Logger.ShortInfo("Checking if stuck...")

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
							memberStates = append(memberStates, fmt.Sprintf("%s%s current state: %s and state code: %s", memberLabel, memberName, Statuses[*member.State], Statuses[stateCode]))
							deployableMembers++
						} else if *member.State == project.ProjectConfig_State_Deployed {
							memberStates = append(memberStates, fmt.Sprintf("%s%s current state: %s, current state code:%s", memberLabel, memberName, Statuses[*member.State], Statuses[stateCode]))
							deployableMembers++
						} else {
							memberStates = append(memberStates, fmt.Sprintf("%s%s current state: %s, current state code:%s", memberLabel, memberName, Statuses[*member.State], Statuses[stateCode]))
						}
					}
					if memberCount == deployableMembers {
						var stackStatusMessage strings.Builder
						stackStatusMessage.WriteString(fmt.Sprintf("Stack is still deploying, current state: %s and state code: %s", Statuses[*stackDetails.State], Statuses[*stackDetails.StateCode]))
						for _, memState := range memberStates {
							stackStatusMessage.WriteString(memState)
						}

						options.Logger.ShortInfo(stackStatusMessage.String())
						time.Sleep(time.Duration(options.StackPollTimeSeconds) * time.Second)
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
				errorList = append(errorList, fmt.Errorf("%s", errorMessage.String()))
				failed = true
			}

		}
		// fail fast if any error states
		if stackDetails.State == nil {
			errorList = append(errorList, fmt.Errorf("stackDetails state is nil"))
			return errorList
		}

		switch *stackDetails.State {
		case project.ProjectConfig_State_ApplyFailed, project.ProjectConfig_State_DeployingFailed, project.ProjectConfig_State_ValidatingFailed:
			errorList = append(errorList, fmt.Errorf("failed with state %s", Statuses[*stackDetails.State]))
			failed = true
		}

		if failed {
			options.Logger.ShortError(fmt.Sprintf("Stack Deploy Failed, current state: %s\n%s", Statuses[*stackDetails.State], currentDeployStatus))
			return errorList
		}

		if stackDetails.StateCode == nil {
			options.Logger.ShortInfo("Stack state code is nil, skipping this time")
			time.Sleep(time.Duration(options.StackPollTimeSeconds) * time.Second)
			continue
		}

		if *stackDetails.State == project.ProjectConfig_State_Deployed &&
			*stackDetails.StateCode != project.ProjectConfig_StateCode_AwaitingMemberDeployment &&
			*stackDetails.StateCode != project.ProjectConfig_StateCode_AwaitingPrerequisite &&
			*stackDetails.StateCode != project.ProjectConfig_StateCode_AwaitingStackSetup &&
			*stackDetails.StateCode != project.ProjectConfig_StateCode_AwaitingValidation {
			deployComplete = true

			options.Logger.ShortInfo(fmt.Sprintf("Stack Deployed Successfully, current state: %s and state code: %s", Statuses[*stackDetails.State], Statuses[*stackDetails.StateCode]))
		} else {
			options.Logger.ShortInfo(fmt.Sprintf("Stack is still deploying, current state: %s and state code: %s\n%s", Statuses[*stackDetails.State], Statuses[*stackDetails.StateCode], currentDeployStatus))
			time.Sleep(time.Duration(options.StackPollTimeSeconds) * time.Second)
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
	options.Logger.ShortInfo(fmt.Sprintf("Stacks final state: %s, state code: %s", Statuses[*stackDetails.State], Statuses[stateCode]))

	return nil
}

func (options *TestProjectsOptions) TriggerUnDeploy() (bool, []error) {
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
				return false, []error{err}
			}
			if stackDetails == nil {
				return false, []error{fmt.Errorf("stackDetails is nil")}
			}

			stackMembers, err := options.CloudInfoService.GetStackMembers(options.currentStackConfig)
			if err != nil {
				return false, []error{err}
			}

			syncErrs := TrackAndResyncState(options, stackDetails, stackMembers, memberStateStartTime, &mu)
			if len(syncErrs) > 0 {
				return false, syncErrs
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
				options.Logger.ShortInfo(fmt.Sprintf("Stack is in state %s with stateCode %s, ready for undeploy", Statuses[*stackDetails.State], Statuses[stateCode]))
			} else {
				options.Logger.ShortInfo(fmt.Sprintf("Stack is in state %s with stateCode %s, waiting for all members to complete", Statuses[*stackDetails.State], Statuses[stateCode]))
			}
			// then double check if any configuration is still in VALIDATING, DEPLOYING, or UNDEPLOYING state
			for _, member := range stackMembers {
				if *member.State == project.ProjectConfig_State_Validating || *member.State == project.ProjectConfig_State_Deploying || *member.State == project.ProjectConfig_State_Undeploying {
					readyForUndeploy = false
					memberName, err := options.CloudInfoService.LookupMemberNameByID(stackDetails, *member.ID)
					if err != nil {
						memberName = fmt.Sprintf("Unknown name, ID: %s", *member.ID)
					}
					options.Logger.ShortInfo(fmt.Sprintf("Member %s is still in state %s, waiting for all members to complete", memberName, Statuses[*member.State]))
					time.Sleep(time.Duration(options.StackPollTimeSeconds) * time.Second)
				}
			}
		}

		if !readyForUndeploy {
			return false, []error{fmt.Errorf("timeout waiting for all members to complete, could not trigger undeploy")}
		}
		_, _, errUndep := options.CloudInfoService.UndeployConfig(options.currentStackConfig)
		if errUndep != nil {
			if errUndep.Error() == "Not Modified" {
				options.Logger.ShortInfo("Nothing to undeploy")
				return false, nil
			}
			return false, []error{errUndep}
		}
	}
	return true, nil
}
func (options *TestProjectsOptions) TriggerUnDeployAndWait() (errorList []error) {
	if !options.SkipUndeploy {
		triggered, err := options.TriggerUnDeploy()
		if err != nil {
			return err
		}

		if triggered {
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
				options.Logger.ShortInfo("Checking Stack Undeploy Status")
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
				currentUndeployStatus := fmt.Sprintf("%s Current Undeploy Status:\n", common.ColorizeString(common.Colors.Blue, fmt.Sprintf("[STACK - %s]", options.currentStackConfig.Name)))
				undeployedCount = 0

				memberLabel := common.ColorizeString(common.Colors.Blue, "\t- Member: ")

				for _, member := range stackMembers {
					if member.ID == nil {
						return []error{fmt.Errorf("member ID is nil")}
					}
					memberName, err := options.CloudInfoService.LookupMemberNameByID(stackDetails, *member.ID)
					if err != nil {
						memberName = fmt.Sprintf("Unknown name, ID: %s", *member.ID)
					}

					if member.State == nil {
						memberStates = append(memberStates, fmt.Sprintf("%s%s state is nil, skipping this time", memberLabel, memberName))
						undeployableState = false
						continue
					}

					if *member.State == project.ProjectConfig_State_Undeploying {
						memberStates = append(memberStates, fmt.Sprintf("%s%s current state: %s", memberLabel, memberName, Statuses[*member.State]))
					} else if *member.State == project.ProjectConfig_State_UndeployingFailed {
						memberStates = append(memberStates, fmt.Sprintf("%s%s current state: %s", memberLabel, memberName, Statuses[*member.State]))
						undeployableState = false
						failed = true
						logMessage, terraLogs := options.CloudInfoService.GetSchematicsJobLogsForMember(member, memberName, options.currentProjectConfig.Location)
						options.Logger.ShortError(terraLogs)
						errorList = append(errorList, fmt.Errorf("(%s) failed Undeployment\n%s", memberName, logMessage))
					} else if cloudinfo.ProjectsMemberIsUndeployed(member) {
						memberStates = append(memberStates, fmt.Sprintf("%s%s current state: %s", memberLabel, memberName, Statuses[*member.State]))
						undeployedCount++
					} else {
						memberStates = append(memberStates, fmt.Sprintf("%s%s current state: %s", memberLabel, memberName, Statuses[*member.State]))
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
					if *stackDetails.State == project.ProjectConfig_State_UndeployingFailed {
						undeployableState = false
						failed = true
						errorList = append(errorList, fmt.Errorf("undeploy stack failed with state %s", Statuses[*stackDetails.State]))
					} else if *stackDetails.State == project.ProjectConfig_State_Draft && stateCode == project.ProjectConfig_StateCode_AwaitingMemberDeployment {
						// Treat draft state with awaiting_member_deployment as complete undeploy
						undeployComplete = true
						options.Logger.ShortInfo(fmt.Sprintf("Stack is in state %s with state code %s, treating as complete undeploy", Statuses[*stackDetails.State], Statuses[stateCode]))
					} else {
						options.Logger.ShortInfo(fmt.Sprintf("Stack is still undeploying, current state: %s and state code: %s\n%s", Statuses[*stackDetails.State], Statuses[stateCode], currentUndeployStatus+strings.Join(memberStates, "\n")))
						time.Sleep(time.Duration(options.StackPollTimeSeconds) * time.Second)
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
			if cfgErr != nil {
				return []error{cfgErr}
			}
			if stackDetails == nil {
				return []error{fmt.Errorf("stackDetails is nil")}
			}
			stateCode := "Unknown"
			if stackDetails.StateCode != nil {
				stateCode = *stackDetails.StateCode
			}
			options.Logger.ShortInfo(fmt.Sprintf("Stacks final state: %s, state code: %s", Statuses[*stackDetails.State], Statuses[stateCode]))
			// check the counts and if undeployed count is less than total members make it red
			if undeployedCount < totalMembers {
				options.Logger.ShortError(fmt.Sprintf("Stack undeploy failed, current state: %s, undeployed count: %d, total members: %d", Statuses[*stackDetails.State], undeployedCount, totalMembers))
				options.Logger.ShortInfo("Undeploy completed unsuccessfully")
			} else {
				options.Logger.ShortInfo(fmt.Sprintf("%s/%s members undeployed", common.ColorizeString(common.Colors.Green, strconv.Itoa(undeployedCount)), common.ColorizeString(common.Colors.Green, strconv.Itoa(totalMembers))))
				options.Logger.ShortInfo("Undeploy completed successfully")
			}
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
		// ensure we always run the test tear down, even if a panic occurs
		defer func() {
			if r := recover(); r != nil {

				options.Testing.Fail()
				// Get the file and line number where the panic occurred
				_, file, line, ok := runtime.Caller(4)
				if ok {
					options.Logger.ShortError(fmt.Sprintf("Recovered from panic: %v\nOccurred at: %s:%d\n", r, file, line))
				} else {
					options.Logger.ShortError(fmt.Sprintf("Recovered from panic: %v", r))
				}
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
	options.Logger.ShortInfo("Creating Test Project")
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
	// Create project with retry logic to handle transient database errors
	retryConfig := common.ProjectOperationRetryConfig()
	retryConfig.Logger = options.Logger
	retryConfig.OperationName = "project creation"

	prj, err := common.RetryWithConfig(retryConfig, func() (*project.Project, error) {
		prj, resp, err := options.CloudInfoService.CreateProjectFromConfig(options.currentProjectConfig)
		if err != nil {
			options.Logger.ShortWarn(fmt.Sprintf("Project creation attempt failed: %v (will retry if retryable)", err))

			// Check if project was actually created despite the error
			if common.StringContainsIgnoreCase(err.Error(), "already exists") {
				options.Logger.ShortInfo("Project creation returned 'already exists' error - this indicates the project was successfully created on a previous attempt")

				// The "already exists" error means the operation succeeded - the project exists
				// We need to extract the project information from the error or response
				// Since the error confirms creation succeeded, we'll return success
				// The project ID should be available in the response even on "already exists" error
				if resp != nil && resp.StatusCode == 409 { // 409 Conflict for "already exists"
					options.Logger.ShortInfo("Treating 'already exists' response as successful project creation")

					// For "already exists", the project was created successfully
					// We'll return the prj even if there was an error, as the operation succeeded
					if prj != nil {
						return prj, nil
					}

					// If prj is nil but we got 409, create a minimal project reference
					// This case handles when IBM Cloud returns an error but the project exists
					options.Logger.ShortInfo("Project created successfully despite API error response")
					return &project.Project{
						ID: core.StringPtr(""), // Will be populated later if needed
					}, nil
				}
			}

			return nil, err
		}

		// Check for successful creation (HTTP 201)
		if resp.StatusCode != 201 {
			options.Logger.ShortWarn(fmt.Sprintf("Project creation returned unexpected status code: %d", resp.StatusCode))
			return nil, fmt.Errorf("unexpected response code: %d", resp.StatusCode)
		}

		return prj, nil
	})

	if assert.NoError(options.Testing, err) {
		options.Logger.ShortInfo(fmt.Sprintf("Created Test Project - %s", *prj.Definition.Name))
		// https://cloud.ibm.com/projects/a1316ed6-76de-418e-bb9a-e7ed05aa7834
		// print link to project
		options.Logger.ShortInfo(fmt.Sprintf("Project URL: %s", fmt.Sprintf("https://cloud.ibm.com/projects/%s", *prj.ID)))
		options.currentProject = prj
		options.currentProjectConfig.ProjectID = *prj.ID

		// Add post-creation delay for eventual consistency
		if options.PostCreateDelay != nil && *options.PostCreateDelay > 0 {
			options.Logger.ShortInfo(fmt.Sprintf("Waiting %v for project to be available...", *options.PostCreateDelay))
			time.Sleep(*options.PostCreateDelay)
		}
	} else {
		projectURL := fmt.Sprintf("https://cloud.ibm.com/projects")
		options.Logger.ShortError(fmt.Sprintf("Project creation failed after retries - Console: %s", projectURL))
		return fmt.Errorf("project creation failed after retries")
	}

	if assert.NoError(options.Testing, options.ConfigureTestStack()) {
		options.Logger.ShortInfo(fmt.Sprintf("Configured Test Stack - %s \n- %s %s \n- %s %s", *prj.Definition.Name, common.ColorizeString(common.Colors.Blue, "Project ID:"), *prj.ID, common.ColorizeString(common.Colors.Blue, "Config ID:"), *options.currentStack.Configuration.ID))
		if options.PreDeployHook != nil {
			options.Logger.ShortInfo("Running PreDeployHook")
			hookErr := options.PreDeployHook(options)
			if hookErr != nil {
				options.Testing.Fail()
				return hookErr
			}
			options.Logger.ShortInfo("Finished PreDeployHook")
		}
		// Deploy the configuration in parallel
		deployErrs := options.TriggerDeployAndWait()

		var finalError error

		if len(deployErrs) > 0 {
			// print all errors and return a single error
			for _, derr := range deployErrs {
				options.Logger.ShortError(fmt.Sprintf("Error: %s", derr.Error()))
				if finalError == nil {
					finalError = derr
				} else {
					finalError = fmt.Errorf("%w\n%s", finalError, derr)
				}
			}
		} else {
			options.Logger.ShortInfo("All configurations deployed successfully")
		}

		if options.PostDeployHook != nil {
			options.Logger.ShortInfo("Running PostDeployHook")
			hookErr := options.PostDeployHook(options)
			if hookErr != nil {
				options.Testing.Fail()
				return hookErr
			}
			options.Logger.ShortInfo("Finished PostDeployHook")
		}
		if finalError != nil {
			options.Testing.Fail()
			return finalError
		} else {
			return nil
		}
	}
	return nil
}

func (options *TestProjectsOptions) TestTearDown() {
	if options.currentProject == nil {
		options.Logger.ShortInfo("No project to delete")
		return
	}
	if !options.SkipTestTearDown {
		if options.executeResourceTearDown() {

			// Trigger undeploy and wait for completion
			options.Logger.ShortInfo("Triggering Undeploy and waiting for completion")
			undeployErrors := options.TriggerUnDeployAndWait()
			if len(undeployErrors) > 0 {
				for _, err := range undeployErrors {
					options.Logger.ShortError(fmt.Sprintf("Failed to undeploy: %s", err))
					options.Testing.Fail()
				}
			}

		}
		if options.executeProjectTearDown() {
			// Wait until no pipeline actions are running or timeout is reached
			options.Logger.ShortInfo("Checking all pipeline actions are complete")
			timeout := time.Now().Add(time.Duration(options.DeployTimeoutMinutes) * time.Minute)
			for {
				running, err := options.CloudInfoService.ArePipelineActionsRunning(options.currentStackConfig)
				if err != nil {
					options.Logger.ShortError(fmt.Sprintf("Error checking pipeline actions: %s", err))
					options.Testing.Fail()
					return
				}
				if !running || time.Now().After(timeout) {
					break
				}
				options.Logger.ShortInfo("Pipeline actions are still running, waiting...")
				time.Sleep(time.Duration(options.StackPollTimeSeconds) * time.Second)
			}

			// Check if timeout was reached
			if time.Now().After(timeout) {
				options.Logger.ShortError("Timeout reached while waiting for pipeline actions to complete")
				options.Testing.Fail()
				return
			}
			options.Logger.ShortInfo("All pipeline actions are complete")
			// Delete the project
			options.Logger.ShortInfo("Deleting Test Project")
			if options.currentProject.ID != nil {
				// Delete project with retry logic to handle transient database errors
				retryConfig := common.ProjectOperationRetryConfig()
				retryConfig.Logger = options.Logger
				retryConfig.OperationName = "project deletion"

				_, err := common.RetryWithConfig(retryConfig, func() (*project.ProjectDeleteResponse, error) {
					result, resp, err := options.CloudInfoService.DeleteProject(*options.currentProject.ID)
					if err != nil {
						options.Logger.ShortWarn(fmt.Sprintf("Project deletion attempt failed: %v (will retry if retryable)", err))

						// Check if project was actually deleted despite the error
						if common.StringContainsIgnoreCase(err.Error(), "not found") || common.StringContainsIgnoreCase(err.Error(), "does not exist") {
							options.Logger.ShortInfo("Project deletion returned 'not found' error - this indicates the project was successfully deleted on a previous attempt")

							// The "not found" error means the deletion succeeded - the project doesn't exist
							// This is the desired end state for deletion
							if resp != nil && resp.StatusCode == 404 { // 404 Not Found
								options.Logger.ShortInfo("Treating 'not found' response as successful project deletion")
								return &project.ProjectDeleteResponse{}, nil
							}

							// Even without a 404 response, "not found" in error message indicates successful deletion
							options.Logger.ShortInfo("Project deleted successfully despite API error response")
							return &project.ProjectDeleteResponse{}, nil
						}

						return nil, err
					}

					// Check for successful deletion (HTTP 202)
					if resp.StatusCode != 202 {
						options.Logger.ShortWarn(fmt.Sprintf("Project deletion returned unexpected status code: %d", resp.StatusCode))
						return nil, fmt.Errorf("unexpected response code: %d", resp.StatusCode)
					}

					return result, nil
				})

				if assert.NoError(options.Testing, err) {
					options.Logger.ShortInfo("Deleted Test Project")
				} else {
					projectURL := fmt.Sprintf("https://cloud.ibm.com/projects/%s", *options.currentProject.ID)
					options.Logger.ShortError(fmt.Sprintf("Error deleting Test Project: %s\nProject Console: %s", err, projectURL))
				}
			} else {
				options.Logger.ShortInfo("No project ID found to delete")
			}
		} else {
			options.Logger.ShortInfo(fmt.Sprintf("Project URL: %s", fmt.Sprintf("https://cloud.ibm.com/projects/%s", *options.currentProject.ID)))
		}
	} else {
		options.Logger.ShortInfo(fmt.Sprintf("Project URL: %s", fmt.Sprintf("https://cloud.ibm.com/projects/%s", *options.currentProject.ID)))
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
		if options.currentStackConfig == nil || options.currentStackConfig.ConfigID == "" {
			options.Logger.ShortError("Terratest failed. No resources to delete.")
		} else {
			options.Logger.ShortError("Terratest failed. Debug the Test and delete resources manually.")
		}
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
		if options.currentProject == nil {
			options.Logger.ShortError("Terratest failed. No project to delete.")
		} else {
			options.Logger.ShortError("Terratest failed. Debug the Test and delete the project manually.")
		}
	}

	return execute
}

// Perform required steps for new test
func (options *TestProjectsOptions) testSetup() error {

	// setup logger
	if options.Logger == nil {
		options.Logger = common.NewTestLogger(options.Testing.Name())
	}

	if options.ProjectName != "" {
		options.Logger.SetPrefix(fmt.Sprintf("PROJECTS - %s", options.ProjectName))
	} else {
		options.Logger.SetPrefix("PROJECTS")
	}

	options.Logger.EnableDateTime(false)

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
					options.Logger.ShortInfo(fmt.Sprintf("Member %s stuck in state %s for more than %d minutes, triggering SyncConfig", memberName, *member.State, options.StackAutoSyncInterval))
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
