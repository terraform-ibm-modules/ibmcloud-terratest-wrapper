package testprojects

import (
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/IBM/go-sdk-core/v5/core"
	project "github.com/IBM/project-go-sdk/projectv1"
	"github.com/stretchr/testify/assert"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/cloudinfo"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/common"
)

//func (options *TestProjectsOptions) ValidateConfig(configName string) error {
//	// Get all configurations
//	allConfigurations, cfgErr := options.CloudInfoService.GetProjectConfigs(*options.currentProject.ID)
//	if !assert.NoError(options.Testing, cfgErr) {
//		options.Testing.Log("[PROJECTS] Failed to get configurations")
//		return cfgErr
//	}
//	currentConfig, currConfigErr := getConfigFromName(configName, allConfigurations)
//	if currConfigErr != nil {
//		return currConfigErr
//	}
//
//	// set authenticator for current member(configuration)
//	tempConfig, _, tempConfigErr := options.CloudInfoService.GetConfig(*options.currentProject.ID, *currentConfig.ID)
//	if tempConfigErr != nil {
//		return tempConfigErr
//	}
//
//	// We need to do this so the correct type is cast without errors will be present
//	switch def := tempConfig.Definition.(type) {
//	case *project.ProjectConfigDefinitionResponseDAConfigDefinitionPropertiesResponse:
//		if def != nil {
//			var patchConfig *project.ProjectConfigDefinitionPatchResourceConfigDefinitionPropertiesPatch
//			if def.Authorizations.Method == nil ||
//				*def.Authorizations.Method == "" ||
//				(*def.Authorizations.ApiKey == "" && *def.Authorizations.TrustedProfileID == "") {
//				var patchInputs map[string]interface{}
//				if options.StackMemberInputs != nil {
//					patchInputs = options.StackMemberInputs[configName]
//				}
//				patchConfig = &project.ProjectConfigDefinitionPatchResourceConfigDefinitionPropertiesPatch{
//					Authorizations: &project.ProjectConfigAuth{
//						Method: core.StringPtr(project.ProjectConfigAuth_Method_ApiKey),
//						ApiKey: &options.CloudInfoService.(*cloudinfo.CloudInfoService).ApiKey,
//					},
//					Inputs: patchInputs,
//				}
//			} else {
//				var patchInputs map[string]interface{}
//				if options.StackMemberInputs != nil {
//					patchInputs = options.StackMemberInputs[configName]
//				}
//				patchConfig = &project.ProjectConfigDefinitionPatchResourceConfigDefinitionPropertiesPatch{
//					Inputs: patchInputs,
//				}
//			}
//			_, updateResponse, updateErr := options.CloudInfoService.UpdateConfig(*options.currentProject.ID, *currentConfig.ID, patchConfig)
//			if updateErr != nil {
//				return updateErr
//			}
//			if updateResponse.StatusCode != 200 {
//				return fmt.Errorf("error updating configuration %s", configName)
//			}
//		}
//	case *project.ProjectConfigDefinitionResponse:
//		if def != nil {
//			var patchConfig *project.ProjectConfigDefinitionPatchResourceConfigDefinitionPropertiesPatch
//			if def.Authorizations.Method == nil ||
//				*def.Authorizations.Method == "" ||
//				(*def.Authorizations.ApiKey == "" && *def.Authorizations.TrustedProfileID == "") {
//				var patchInputs map[string]interface{}
//				if options.StackMemberInputs != nil {
//					patchInputs = options.StackMemberInputs[configName]
//				}
//				patchConfig = &project.ProjectConfigDefinitionPatchResourceConfigDefinitionPropertiesPatch{
//					Authorizations: &project.ProjectConfigAuth{
//						Method: core.StringPtr(project.ProjectConfigAuth_Method_ApiKey),
//						ApiKey: &options.CloudInfoService.(*cloudinfo.CloudInfoService).ApiKey,
//					},
//					Inputs: patchInputs,
//				}
//			} else {
//				var patchInputs map[string]interface{}
//				if options.StackMemberInputs != nil {
//					patchInputs = options.StackMemberInputs[configName]
//				}
//				patchConfig = &project.ProjectConfigDefinitionPatchResourceConfigDefinitionPropertiesPatch{
//					Inputs: patchInputs,
//				}
//			}
//			_, updateResponse, updateErr := options.CloudInfoService.UpdateConfig(*options.currentProject.ID, *currentConfig.ID, patchConfig)
//			if updateErr != nil {
//				return updateErr
//			}
//			if updateResponse.StatusCode != 200 {
//				return fmt.Errorf("error updating configuration %s", configName)
//			}
//
//		}
//	case *project.ProjectConfigDefinitionResponseResourceConfigDefinitionPropertiesResponse:
//		if def != nil {
//			var patchConfig *project.ProjectConfigDefinitionPatchResourceConfigDefinitionPropertiesPatch
//			if def.Authorizations.Method == nil ||
//				*def.Authorizations.Method == "" ||
//				(*def.Authorizations.ApiKey == "" && *def.Authorizations.TrustedProfileID == "") {
//				var patchInputs map[string]interface{}
//				if options.StackMemberInputs != nil {
//					patchInputs = options.StackMemberInputs[configName]
//				}
//				patchConfig = &project.ProjectConfigDefinitionPatchResourceConfigDefinitionPropertiesPatch{
//					Authorizations: &project.ProjectConfigAuth{
//						Method: core.StringPtr(project.ProjectConfigAuth_Method_ApiKey),
//						ApiKey: &options.CloudInfoService.(*cloudinfo.CloudInfoService).ApiKey,
//					},
//					Inputs: patchInputs,
//				}
//			} else {
//				var patchInputs map[string]interface{}
//				if options.StackMemberInputs != nil {
//					patchInputs = options.StackMemberInputs[configName]
//				}
//				patchConfig = &project.ProjectConfigDefinitionPatchResourceConfigDefinitionPropertiesPatch{
//					Inputs: patchInputs,
//				}
//			}
//			_, updateResponse, updateErr := options.CloudInfoService.UpdateConfig(*options.currentProject.ID, *currentConfig.ID, patchConfig)
//			if updateErr != nil {
//				return updateErr
//			}
//			if updateResponse.StatusCode != 200 {
//				return fmt.Errorf("error updating configuration %s", configName)
//			}
//
//		}
//	default:
//		options.Testing.Log(fmt.Sprintf("[WARNING] Configuration %s is not supported for setting authorization", configName))
//	}
//
//	options.Testing.Log(fmt.Sprintf("[PROJECTS] Validating Configuration %s", configName))
//	_, _, validateErr := options.CloudInfoService.ValidateProjectConfig(*options.currentProject.ID, *currentConfig.ID)
//	if assert.NoError(options.Testing, validateErr) {
//		validateConfig, _, valConfErr := options.CloudInfoService.GetConfig(*options.currentProject.ID, *currentConfig.ID)
//		if assert.NoError(options.Testing, valConfErr) {
//			// Set end time
//			approvalEndTime := time.Now().Add(time.Duration(options.ValidationTimeoutMinutes) * time.Minute)
//
//			if *validateConfig.State == project.ProjectConfig_State_Validating {
//				// Wait for the configuration to finish validating
//				for *validateConfig.State == project.ProjectConfig_State_Validating {
//					// if the time is greater than the timeout
//					// return an error
//					if time.Now().After(approvalEndTime) {
//						return fmt.Errorf("validation timeout for configuration %s", configName)
//					}
//					options.Testing.Log(fmt.Sprintf("[PROJECTS] Configuration %s is still validating", configName))
//					time.Sleep(30 * time.Second)
//					validateConfig, _, validateErr = options.CloudInfoService.GetConfig(*options.currentProject.ID, *currentConfig.ID)
//					if !assert.NoError(options.Testing, validateErr) {
//						return validateErr
//					}
//				}
//				if !assert.Equal(options.Testing, project.ProjectConfig_State_Validated, *validateConfig.State) {
//					schematicsCrn := validateConfig.Schematics.WorkspaceCrn
//					if schematicsCrn != nil {
//						options.Testing.Log(fmt.Sprintf("[PROJECTS] Configuration %s failed validation, schematics workspace: %s", configName, *schematicsCrn))
//						options.Testing.Log(fmt.Sprintf("[PROJECTS] Result: %s", *validateConfig.LastValidated.Result))
//
//						if validateConfig.LastValidated.Job.Summary.PlanMessages != nil && validateConfig.LastValidated.Job.Summary.PlanMessages.ErrorMessages != nil {
//							for _, planErr := range validateConfig.LastValidated.Job.Summary.PlanMessages.ErrorMessages {
//								options.Testing.Log(fmt.Sprintf("[PROJECTS] Plan Error: %s", planErr))
//							}
//						} else {
//							options.Testing.Log(fmt.Sprintf("[PROJECTS] No plan error messages found for configuration %s", configName))
//						}
//					}
//					return fmt.Errorf("validation failed for configuration %s last state: %s", configName, *validateConfig.State)
//				}
//			}
//		} else {
//			return valConfErr
//		}
//	} else {
//		return validateErr
//	}
//	return nil
//}
//
//func (options *TestProjectsOptions) ApproveConfig(configName string) error {
//	// Get all configurations
//	allConfigurations, cfgErr := options.CloudInfoService.GetProjectConfigs(*options.currentProject.ID)
//	if !assert.NoError(options.Testing, cfgErr) {
//		options.Testing.Log("[PROJECTS] Failed to get configurations")
//		return cfgErr
//	}
//	currentConfig, currConfigErr := getConfigFromName(configName, allConfigurations)
//	if currConfigErr != nil {
//		return currConfigErr
//	}
//
//	// Approve the configuration
//	options.Testing.Log(fmt.Sprintf("[PROJECTS] Approving Configuration %s", configName))
//	approveConfig, _, approveErr := options.CloudInfoService.ApproveConfig(*options.currentProject.ID, *currentConfig.ID)
//	if assert.NoError(options.Testing, approveErr) {
//		if !assert.Equal(options.Testing, project.ProjectConfig_State_Approved, *approveConfig.State) {
//			options.Testing.Log(fmt.Sprintf("[PROJECTS] Configuration %s failed to approve", configName))
//			return fmt.Errorf("error approving configuration %s", configName)
//		}
//		options.Testing.Log(fmt.Sprintf("[PROJECTS] Approved Configuration %s", configName))
//	}
//	return nil
//}
//
//func (options *TestProjectsOptions) DeployConfig(configName string) error {
//	// Get all configurations
//	allConfigurations, cfgErr := options.CloudInfoService.GetProjectConfigs(*options.currentProject.ID)
//	if !assert.NoError(options.Testing, cfgErr) {
//		options.Testing.Log("[PROJECTS] Failed to get configurations")
//		return cfgErr
//	}
//	currentConfig, currConfigErr := getConfigFromName(configName, allConfigurations)
//	if currConfigErr != nil {
//		return currConfigErr
//	}
//
//	// Deploy the configuration
//	options.Testing.Log(fmt.Sprintf("[PROJECTS] Deploying Configuration %s", configName))
//	_, _, deployErr := options.CloudInfoService.DeployConfig(*options.currentProject.ID, *currentConfig.ID)
//
//	if assert.NoError(options.Testing, deployErr) {
//		currentCfg, _, curCfgErr := options.CloudInfoService.GetConfig(*options.currentProject.ID, *currentConfig.ID)
//		if assert.NoError(options.Testing, curCfgErr) {
//			// Set end time
//			deployEndTime := time.Now().Add(time.Duration(options.DeployTimeoutMinutes) * time.Minute)
//
//			if *currentCfg.State == project.ProjectConfig_State_Deploying {
//				// Wait for the configuration to finish deploying
//				for *currentCfg.State == project.ProjectConfig_State_Deploying {
//					// if the time is greater than the timeout
//					// return an error
//					if time.Now().After(deployEndTime) {
//						return fmt.Errorf("deploy timeout for configuration %s", configName)
//					}
//					options.Testing.Log(fmt.Sprintf("[PROJECTS] Configuration %s is still deploying", configName))
//					time.Sleep(30 * time.Second)
//					currentCfg, _, curCfgErr = options.CloudInfoService.GetConfig(*options.currentProject.ID, *currentConfig.ID)
//					if !assert.NoError(options.Testing, curCfgErr) {
//						return curCfgErr
//					}
//				}
//				if !assert.Equal(options.Testing, project.ProjectConfig_State_Deployed, *currentCfg.State) {
//					schematicsCrn := currentCfg.Schematics.WorkspaceCrn
//					if schematicsCrn != nil {
//						options.Testing.Log(fmt.Sprintf("[PROJECTS] Configuration %s failed deploy, schematics workspace: %s", configName, *schematicsCrn))
//						options.Testing.Log(fmt.Sprintf("[PROJECTS] Result: %s", *currentCfg.LastDeployed.Result))
//
//						if currentCfg.LastDeployed != nil && currentCfg.LastDeployed.Job != nil && currentCfg.LastDeployed.Job.Summary != nil {
//							if currentCfg.LastDeployed.Job.Summary.PlanMessages != nil && currentCfg.LastDeployed.Job.Summary.PlanMessages.ErrorMessages != nil {
//								for _, planErr := range currentCfg.LastDeployed.Job.Summary.PlanMessages.ErrorMessages {
//									options.Testing.Log(fmt.Sprintf("[PROJECTS] Plan Error: %s", planErr))
//								}
//							} else {
//								options.Testing.Log(fmt.Sprintf("[PROJECTS] No plan error messages found for configuration %s", configName))
//							}
//							if currentCfg.LastDeployed.Job.Summary.ApplyMessages != nil && currentCfg.LastDeployed.Job.Summary.ApplyMessages.ErrorMessages != nil {
//								for _, applyErr := range currentCfg.LastDeployed.Job.Summary.ApplyMessages.ErrorMessages {
//									options.Testing.Log(fmt.Sprintf("[PROJECTS] Apply Error: %s", applyErr))
//								}
//							} else {
//								options.Testing.Log(fmt.Sprintf("[PROJECTS] No apply error messages found for configuration %s", configName))
//							}
//						} else {
//							options.Testing.Log(fmt.Sprintf("[PROJECTS] No messages found for configuration %s", configName))
//						}
//
//					}
//					return fmt.Errorf("deploy failed for configuration %s last state: %s", configName, *currentCfg.State)
//				}
//
//				options.Testing.Log(fmt.Sprintf("[PROJECTS] Deployed Configuration %s", configName))
//			}
//		} else {
//			return curCfgErr
//		}
//	} else {
//		return deployErr
//	}
//	return nil
//}

func (options *TestProjectsOptions) ConfigureTestStack() error {
	// Configure the test stack
	options.Testing.Log("[PROJECTS] Configuring Test Stack")
	var stackResp *core.DetailedResponse
	var stackErr error
	options.currentStack, stackResp, stackErr = options.CloudInfoService.CreateStackFromConfigFile(cloudinfo.ConfigDetails{ProjectID: *options.currentProject.ID}, options.StackConfigurationPath, options.StackCatalogJsonPath)
	if !assert.NoError(options.Testing, stackErr) {
		options.Testing.Log("[PROJECTS] Failed to configure Test Stack")
		var sdkProblem *core.SDKProblem

		if errors.As(stackErr, &sdkProblem) {
			if strings.Contains(sdkProblem.Summary, "A stack definition member input") &&
				strings.Contains(sdkProblem.Summary, "was not found in the configuration") {
				sdkProblem.Summary = fmt.Sprintf("%s Input name possibly removed or renamed", sdkProblem.Summary)
				// A stack definition member input resource_tag was not found in the configuration primary-da.
				// extract the member config name, get the member config version, get all inputs for this version
				member_name := strings.Split(sdkProblem.Summary, "was not found in the configuration ")[1]
				member_name = strings.Split(member_name, ".")[0]
				versionLocator, vlErr := GetVersionLocatorFromStackDefinitionForMemberName(options.StackConfigurationPath, member_name)
				if !assert.NoError(options.Testing, vlErr) {
					options.Testing.Log("Error getting version locator from stack definition")
					return sdkProblem
				}
				// get inputs for the member config of version
				version, vererr := options.CloudInfoService.GetCatalogVersionByLocator(versionLocator)
				if !assert.NoError(options.Testing, vererr) {
					options.Testing.Log("Error getting offering")
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
			options.Testing.Log("[PROJECTS] Configured Test Stack")
		} else {
			options.Testing.Log("[PROJECTS] Failed to configure Test Stack")
			return fmt.Errorf("error configuring test stack response code: %d\nrespone:%s", stackResp.StatusCode, stackResp.Result)
		}
	}
	return nil
}

//
//func (options *TestProjectsOptions) SerialDeployConfigurations() error {
//
//	// Loop through the StackConfigurationOrder
//	for _, configName := range options.StackConfigurationOrder {
//		err := options.ValidateApproveDeploy(configName)
//		if err != nil {
//			return err
//		}
//	}
//	return nil
//}
//
//func (options *TestProjectsOptions) ValidateApproveDeploy(configName string) error {
//	if err := options.ValidateConfig(configName); err != nil {
//		options.Testing.Log(fmt.Sprintf("Error validating configuration %s: %s", configName, err))
//		options.Testing.Fail()
//		return err
//	}
//	if err := options.ApproveConfig(configName); err != nil {
//		options.Testing.Log(fmt.Sprintf("Error approving configuration %s: %s", configName, err))
//		options.Testing.Fail()
//		return err
//	}
//	if err := options.DeployConfig(configName); err != nil {
//		options.Testing.Log(fmt.Sprintf("Error deploying configuration %s: %s", configName, err))
//		options.Testing.Fail()
//		allConfigurations, cfgErr := options.CloudInfoService.GetProjectConfigs(*options.currentProject.ID)
//		if !assert.NoError(options.Testing, cfgErr) {
//			options.Testing.Log("[PROJECTS] Failed to get configurations")
//			return cfgErr
//		}
//		currentConfig, currConfigErr := getConfigFromName(configName, allConfigurations)
//		if currConfigErr != nil {
//			return currConfigErr
//		}
//		currentCfg, _, curCfgErr := options.CloudInfoService.GetConfig(*options.currentProject.ID, *currentConfig.ID)
//		if assert.NoError(options.Testing, curCfgErr) {
//			if currentCfg.LastDeployed.Job.Summary.ApplyMessages != nil && currentCfg.LastDeployed.Job.Summary.ApplyMessages.ErrorMessages != nil {
//				for _, applyErr := range currentCfg.LastDeployed.Job.Summary.ApplyMessages.ErrorMessages {
//					options.Testing.Log(fmt.Sprintf("[PROJECTS] Apply Error: %s", applyErr))
//				}
//			}
//		}
//
//		return err
//	}
//
//	return nil
//
//}
//func (options *TestProjectsOptions) ParallelDeployConfigurations() []error {
//	// Get all configurations
//	allConfigurations, cfgErr := options.CloudInfoService.GetProjectConfigs(*options.currentProject.ID)
//	if cfgErr != nil {
//		return []error{cfgErr}
//	}
//	setUndeployOrder := false
//	if options.StackUndeployOrder == nil {
//		setUndeployOrder = true
//	}
//	// create a slice of strings to store the configurations that have been deployed
//	deployedConfigurations := make([]string, 0)
//	// create a channel to collect errors from goroutines
//	errChan := make(chan error, len(allConfigurations)-1) // -1 to account for the stack configuration
//
//	hasError := false
//	// while all configurations are not deployed
//	for len(deployedConfigurations) != len(allConfigurations)-1 {
//		if hasError {
//			options.Testing.Log("[PROJECTS] Error deploying configurations, terminating deployment.")
//			break
//		}
//		// Loop through the StackConfigurationOrder identify any configurations that are already deployed
//		currentDeployGroup := make([]string, 0)
//		for _, currentConfig := range allConfigurations {
//			if common.StrArrayContains(deployedConfigurations, *currentConfig.Definition.Name) {
//				continue
//			}
//			cfg, _, _ := options.CloudInfoService.GetConfig(*options.currentProject.ID, *currentConfig.ID)
//
//			// Ignore the stack configuration
//			if *currentConfig.DeploymentModel != "stack" && *currentConfig.State == project.ProjectConfig_State_Draft && (cfg.StateCode == nil || *cfg.StateCode != project.ProjectConfig_StateCode_AwaitingPrerequisite) {
//				currentDeployGroup = append(currentDeployGroup, *currentConfig.Definition.Name)
//				// if setundeploy order and config not in undeploy order, add to undeploy order
//				if setUndeployOrder && !common.StrArrayContains(options.StackUndeployOrder, *currentConfig.Definition.Name) {
//					// Prepend the name of the current configuration to the StackUndeployOrder slice.
//					// This is done by creating a new slice with the current configuration name as the only element,
//					// and then appending the existing StackUndeployOrder slice to it.
//					// The result is a new slice with the current configuration name at the beginning,
//					// followed by the elements of the original StackUndeployOrder slice.
//					options.StackUndeployOrder = append([]string{*currentConfig.Definition.Name}, options.StackUndeployOrder...)
//				}
//			}
//		}
//		//// if there are no configurations to deploy, break the loop
//		//if len(currentDeployGroup) == 0 {
//		//	break
//		//}
//		// Check if there are configurations to deploy
//		// 'currentDeployGroup' is a slice that contains the configurations to be deployed
//		if len(currentDeployGroup) > 0 {
//			// If there are configurations to deploy, we need to add them to the 'stackUndeployGroups'
//			// 'stackUndeployGroups' is a 2D slice where each element is a group of configurations that need to be undeployed
//			// We want to add the current group of configurations to be deployed at the start of 'stackUndeployGroups'
//			// This is because the configurations that are deployed last should be undeployed first
//			// 'append' is a built-in function in Go that concatenates slices
//			// Here, we are creating a new slice with 'currentDeployGroup' as the first element and the existing 'stackUndeployGroups' as the remaining elements
//			// This effectively adds 'currentDeployGroup' to the start of 'stackUndeployGroups'
//			options.stackUndeployGroups = append([][]string{currentDeployGroup}, options.stackUndeployGroups...)
//		}
//		options.Testing.Log(fmt.Sprintf("[Projects] Deploying group %d/X", len(options.stackUndeployGroups)))
//		// deploy all currentDeployGroup configurations in parallel, and wait for all deployments to complete
//		var wg sync.WaitGroup
//		for _, configName := range currentDeployGroup {
//			wg.Add(1)
//			go func(name string) {
//				defer wg.Done()
//				options.Testing.Log(fmt.Sprintf("[PROJECTS] Starting Validate, Approve, Deploy of Configuration %s", name)) // Add configuration name to the log
//				if err := options.ValidateApproveDeploy(name); err != nil {
//					errChan <- err // send error to the error channel
//					hasError = true
//				} else {
//					// If deployment is successful, add the configuration to the deployed configurations list
//					deployedConfigurations = append(deployedConfigurations, name)
//				}
//			}(configName)
//		}
//		wg.Wait()
//	}
//
//	close(errChan) // close the error channel
//
//	// Check if there were any errors during the deployments
//	var errs []error
//	for err := range errChan {
//		errs = append(errs, err)
//	}
//
//	return errs
//}

func (options *TestProjectsOptions) TriggerDeploy() error {

	return nil
}

func (options *TestProjectsOptions) TriggerDeployAndWait() (errors []error) {

	return nil
}

func (options *TestProjectsOptions) TriggerUnDeploy() error {

	return nil
}

func (options *TestProjectsOptions) TriggerUnDeployAndWait() (errors []error) {

	return nil
}

// RunProjectsTest : Run the test for the projects service
// Creates a new project
// Adds a configuration
// Deploys the configuration
// Deletes the project
func (options *TestProjectsOptions) RunProjectsTest() error {
	if !options.SkipTestTearDown {
		defer options.TestTearDown()
	}

	setupErr := options.testSetup()
	if !assert.NoError(options.Testing, setupErr) {
		return fmt.Errorf("test setup has failed:%w", setupErr)
	}

	// Create a new project
	options.Testing.Log("[PROJECTS] Creating Test Project")
	if options.ProjectDestroyOnDelete == nil {
		options.ProjectDestroyOnDelete = core.BoolPtr(true)
	}
	if options.ProjectAutoDeploy == nil {
		options.ProjectAutoDeploy = core.BoolPtr(false)
	}
	if options.ProjectMonitoringEnabled == nil {
		options.ProjectMonitoringEnabled = core.BoolPtr(false)
	}
	newProject := cloudinfo.ProjectsConfig{
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
	prj, resp, err := options.CloudInfoService.CreateProjectFromConfig(newProject)
	if assert.NoError(options.Testing, err) {
		if assert.Equal(options.Testing, 201, resp.StatusCode) {
			options.Testing.Log(fmt.Sprintf("[PROJECTS] Created Test Project - %s", *prj.Definition.Name))
			options.currentProject = prj

			if assert.NoError(options.Testing, options.ConfigureTestStack()) {
				// Deploy the configuration in parallel
				deployErr := options.TriggerDeployAndWait()
				if !assert.Empty(options.Testing, deployErr) {
					// print all errors and return a single error
					for _, derr := range deployErr {
						options.Testing.Error(derr)
					}
					return fmt.Errorf("error deploying configurations")
				}

				options.Testing.Log("[PROJECTS] All configurations deployed successfully")
				return nil
			}
		}
	}
	return nil
}

func (options *TestProjectsOptions) TestTearDown() {
	if options.currentProject == nil {
		options.Testing.Log("[PROJECTS] No project to delete")
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
					options.Testing.Log("[PROJECTS] Failed to get configurations during project delete, attempting blind delete")
					break
				}

				// Check if any configuration is still in VALIDATING, DEPLOYING, or UNDEPLOYING state
				isAnyConfigInProcess := false
				for _, config := range allConfigurations {
					if *config.State == project.ProjectConfig_State_Validating || *config.State == project.ProjectConfig_State_Deploying || *config.State == project.ProjectConfig_State_Undeploying {
						isAnyConfigInProcess = true
						options.Testing.Log(fmt.Sprintf("[PROJECTS] Configuration %s is still %s", *config.Definition.Name, *config.State))
						break
					}
				}

				// If no configuration is in VALIDATING, DEPLOYING, or UNDEPLOYING state, break the loop
				if !isAnyConfigInProcess {
					break
				}

				// If the time is greater than the timeout, return an error
				if time.Now().After(validationEndTime) {
					options.Testing.Log("validation timeout for configurations")
				}

				// Sleep for 30 seconds before the next check
				time.Sleep(30 * time.Second)

			}

			options.Testing.Log("[PROJECTS] Deleting Test Project")
			_, resp, err := options.CloudInfoService.DeleteProject(*options.currentProject.ID)
			if assert.NoError(options.Testing, err) {
				assert.Equal(options.Testing, 202, resp.StatusCode)
				options.Testing.Log("[PROJECTS] Deleted Test Project")
			}
		}
	}
}

func getConfigFromName(configName string, allConfigs []project.ProjectConfigSummary) (*project.ProjectConfigSummary, error) {
	for _, config := range allConfigs {
		if *config.Definition.Name == configName {
			return &config, nil
		}
	}
	return nil, fmt.Errorf("configuration %s not found", configName)
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
		options.Testing.Log("Terratest failed. Debug the Test and delete the project manually.")
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
