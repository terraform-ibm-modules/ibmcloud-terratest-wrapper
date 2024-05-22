package testprojects

import (
	"errors"
	"fmt"
	"github.com/IBM/go-sdk-core/v5/core"
	project "github.com/IBM/project-go-sdk/projectv1"
	"github.com/stretchr/testify/assert"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/cloudinfo"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/common"
	"os"
	"strings"
	"sync"
	"time"
)

func (options *TestProjectsOptions) ValidateConfig(configName string) error {
	// Get all configurations
	allConfigurations, cfgErr := options.CloudInfoService.GetProjectConfigs(*options.currentProject.ID)
	if !assert.NoError(options.Testing, cfgErr) {
		options.Testing.Log("[PROJECTS] Failed to get configurations")
		return cfgErr
	}
	currentConfig, currConfigErr := getConfigFromName(configName, allConfigurations)
	if currConfigErr != nil {
		return currConfigErr
	}

	// set authenticator for current member(configuration)
	tempConfig, _, tempConfigErr := options.CloudInfoService.GetConfig(*options.currentProject.ID, *currentConfig.ID)
	if tempConfigErr != nil {
		return tempConfigErr
	}

	// We need to do this so the correct type is cast without errors will be present
	switch def := tempConfig.Definition.(type) {
	case *project.ProjectConfigDefinitionResponseDAConfigDefinitionPropertiesResponse:
		if def != nil {
			var patchConfig *project.ProjectConfigDefinitionPatchResourceConfigDefinitionPropertiesPatch
			if def.Authorizations.Method == nil ||
				*def.Authorizations.Method == "" ||
				(*def.Authorizations.ApiKey == "" && *def.Authorizations.TrustedProfileID == "") {
				var patchInputs map[string]interface{}
				if options.StackMemberInputs != nil {
					patchInputs = options.StackMemberInputs[configName]
				}
				patchConfig = &project.ProjectConfigDefinitionPatchResourceConfigDefinitionPropertiesPatch{
					Authorizations: &project.ProjectConfigAuth{
						Method: core.StringPtr(project.ProjectConfigAuth_Method_ApiKey),
						ApiKey: &options.CloudInfoService.(*cloudinfo.CloudInfoService).ApiKey,
					},
					Inputs: patchInputs,
				}
			} else {
				var patchInputs map[string]interface{}
				if options.StackMemberInputs != nil {
					patchInputs = options.StackMemberInputs[configName]
				}
				patchConfig = &project.ProjectConfigDefinitionPatchResourceConfigDefinitionPropertiesPatch{
					Inputs: patchInputs,
				}
			}
			_, updateResponse, updateErr := options.CloudInfoService.UpdateConfig(*options.currentProject.ID, *currentConfig.ID, patchConfig)
			if updateErr != nil {
				return updateErr
			}
			if updateResponse.StatusCode != 200 {
				return fmt.Errorf("error updating configuration %s", configName)
			}
		}
	case *project.ProjectConfigDefinitionResponse:
		if def != nil {
			var patchConfig *project.ProjectConfigDefinitionPatchResourceConfigDefinitionPropertiesPatch
			if def.Authorizations.Method == nil ||
				*def.Authorizations.Method == "" ||
				(*def.Authorizations.ApiKey == "" && *def.Authorizations.TrustedProfileID == "") {
				var patchInputs map[string]interface{}
				if options.StackMemberInputs != nil {
					patchInputs = options.StackMemberInputs[configName]
				}
				patchConfig = &project.ProjectConfigDefinitionPatchResourceConfigDefinitionPropertiesPatch{
					Authorizations: &project.ProjectConfigAuth{
						Method: core.StringPtr(project.ProjectConfigAuth_Method_ApiKey),
						ApiKey: &options.CloudInfoService.(*cloudinfo.CloudInfoService).ApiKey,
					},
					Inputs: patchInputs,
				}
			} else {
				var patchInputs map[string]interface{}
				if options.StackMemberInputs != nil {
					patchInputs = options.StackMemberInputs[configName]
				}
				patchConfig = &project.ProjectConfigDefinitionPatchResourceConfigDefinitionPropertiesPatch{
					Inputs: patchInputs,
				}
			}
			_, updateResponse, updateErr := options.CloudInfoService.UpdateConfig(*options.currentProject.ID, *currentConfig.ID, patchConfig)
			if updateErr != nil {
				return updateErr
			}
			if updateResponse.StatusCode != 200 {
				return fmt.Errorf("error updating configuration %s", configName)
			}

		}
	case *project.ProjectConfigDefinitionResponseResourceConfigDefinitionPropertiesResponse:
		if def != nil {
			var patchConfig *project.ProjectConfigDefinitionPatchResourceConfigDefinitionPropertiesPatch
			if def.Authorizations.Method == nil ||
				*def.Authorizations.Method == "" ||
				(*def.Authorizations.ApiKey == "" && *def.Authorizations.TrustedProfileID == "") {
				var patchInputs map[string]interface{}
				if options.StackMemberInputs != nil {
					patchInputs = options.StackMemberInputs[configName]
				}
				patchConfig = &project.ProjectConfigDefinitionPatchResourceConfigDefinitionPropertiesPatch{
					Authorizations: &project.ProjectConfigAuth{
						Method: core.StringPtr(project.ProjectConfigAuth_Method_ApiKey),
						ApiKey: &options.CloudInfoService.(*cloudinfo.CloudInfoService).ApiKey,
					},
					Inputs: patchInputs,
				}
			} else {
				var patchInputs map[string]interface{}
				if options.StackMemberInputs != nil {
					patchInputs = options.StackMemberInputs[configName]
				}
				patchConfig = &project.ProjectConfigDefinitionPatchResourceConfigDefinitionPropertiesPatch{
					Inputs: patchInputs,
				}
			}
			_, updateResponse, updateErr := options.CloudInfoService.UpdateConfig(*options.currentProject.ID, *currentConfig.ID, patchConfig)
			if updateErr != nil {
				return updateErr
			}
			if updateResponse.StatusCode != 200 {
				return fmt.Errorf("error updating configuration %s", configName)
			}

		}
	default:
		options.Testing.Log(fmt.Sprintf("[WARNING] Configuration %s is not supported for setting authorization", configName))
	}

	validateConfig, _, validateErr := options.CloudInfoService.ValidateProjectConfig(*options.currentProject.ID, *currentConfig.ID)
	if assert.NoError(options.Testing, validateErr) {
		// Set end time
		approvalEndTime := time.Now().Add(time.Duration(options.ValidationTimeoutMinutes) * time.Minute)

		if *validateConfig.State == cloudinfo.VALIDATING {
			// Wait for the configuration to finish validating
			for *validateConfig.State == cloudinfo.VALIDATING {
				// if the time is greater than the timeout
				// return an error
				if time.Now().After(approvalEndTime) {
					return fmt.Errorf("validation timeout for configuration %s", configName)
				}
				options.Testing.Log(fmt.Sprintf("[PROJECTS] Configuration %s is still validating", configName))
				time.Sleep(30 * time.Second)
				validateConfig, _, validateErr = options.CloudInfoService.GetProjectConfigVersion(*options.currentProject.ID, *currentConfig.ID, *currentConfig.Version)
				if !assert.NoError(options.Testing, validateErr) {
					return validateErr
				}
			}
			if !assert.Equal(options.Testing, cloudinfo.VALIDATED, *validateConfig.State) {
				schematicsCrn := validateConfig.Schematics.WorkspaceCrn
				if schematicsCrn != nil {
					options.Testing.Log(fmt.Sprintf("[PROJECTS] Configuration %s failed validation, schematics workspace: %s", configName, *schematicsCrn))
					options.Testing.Log(fmt.Sprintf("[PROJECTS] Result: %s", *validateConfig.LastValidated.Result))
					if validateConfig.LastValidated.Job.Summary.PlanMessages != nil && validateConfig.LastValidated.Job.Summary.PlanMessages.ErrorMessages != nil {
						for _, planErr := range validateConfig.LastValidated.Job.Summary.PlanMessages.ErrorMessages {
							options.Testing.Log(fmt.Sprintf("[PROJECTS] Plan Error: %s", planErr))
						}
					} else {
						options.Testing.Log(fmt.Sprintf("[PROJECTS] No plan error messages found for configuration %s", configName))
					}
				}
				return fmt.Errorf("validation failed for configuration %s last state: %s", configName, *validateConfig.State)
			}
		}
	}
	return nil
}

func (options *TestProjectsOptions) ApproveConfig(configName string) error {
	// Get all configurations
	allConfigurations, cfgErr := options.CloudInfoService.GetProjectConfigs(*options.currentProject.ID)
	if !assert.NoError(options.Testing, cfgErr) {
		options.Testing.Log("[PROJECTS] Failed to get configurations")
		return cfgErr
	}
	currentConfig, currConfigErr := getConfigFromName(configName, allConfigurations)
	if currConfigErr != nil {
		return currConfigErr
	}

	// Approve the configuration
	options.Testing.Log(fmt.Sprintf("[PROJECTS] Approving Configuration %s", configName))
	approveConfig, _, approveErr := options.CloudInfoService.ApproveConfig(*options.currentProject.ID, *currentConfig.ID)
	if assert.NoError(options.Testing, approveErr) {
		if !assert.Equal(options.Testing, cloudinfo.APPROVED, *approveConfig.State) {
			options.Testing.Log(fmt.Sprintf("[PROJECTS] Configuration %s failed to approve", configName))
			return fmt.Errorf("error approving configuration %s", configName)
		}
		options.Testing.Log(fmt.Sprintf("[PROJECTS] Approved Configuration %s", configName))
	}
	return nil
}

func (options *TestProjectsOptions) DeployConfig(configName string) error {
	// Get all configurations
	allConfigurations, cfgErr := options.CloudInfoService.GetProjectConfigs(*options.currentProject.ID)
	if !assert.NoError(options.Testing, cfgErr) {
		options.Testing.Log("[PROJECTS] Failed to get configurations")
		return cfgErr
	}
	currentConfig, currConfigErr := getConfigFromName(configName, allConfigurations)
	if currConfigErr != nil {
		return currConfigErr
	}

	// Deploy the configuration
	options.Testing.Log(fmt.Sprintf("[PROJECTS] Deploying Configuration %s", configName))
	deployConfig, _, deployErr := options.CloudInfoService.DeployConfig(*options.currentProject.ID, *currentConfig.ID)
	if assert.NoError(options.Testing, deployErr) {
		// Set end time
		deployEndTime := time.Now().Add(time.Duration(options.DeployTimeoutMinutes) * time.Minute)

		if *deployConfig.State == cloudinfo.DEPLOYING {
			// Wait for the configuration to finish deploying
			for *deployConfig.State == cloudinfo.DEPLOYING {
				// if the time is greater than the timeout
				// return an error
				if time.Now().After(deployEndTime) {
					return fmt.Errorf("deploy timeout for configuration %s", configName)
				}
				options.Testing.Log(fmt.Sprintf("[PROJECTS] Configuration %s is still deploying", configName))
				time.Sleep(30 * time.Second)
				deployConfig, _, deployErr = options.CloudInfoService.GetProjectConfigVersion(*options.currentProject.ID, *currentConfig.ID, *currentConfig.Version)
				if !assert.NoError(options.Testing, deployErr) {
					return deployErr
				}
			}
			if !assert.Equal(options.Testing, cloudinfo.DEPLOYED, *deployConfig.State) {
				schematicsCrn := deployConfig.Schematics.WorkspaceCrn
				if schematicsCrn != nil {
					options.Testing.Log(fmt.Sprintf("[PROJECTS] Configuration %s failed deploy, schematics workspace: %s", configName, *schematicsCrn))
					options.Testing.Log(fmt.Sprintf("[PROJECTS] Result: %s", deployConfig.LastDeployed.Result))
					if deployConfig.LastDeployed != nil && deployConfig.LastDeployed.Job != nil && deployConfig.LastDeployed.Job.Summary != nil {
						if deployConfig.LastDeployed.Job.Summary.PlanMessages != nil && deployConfig.LastDeployed.Job.Summary.PlanMessages.ErrorMessages != nil {
							for _, planErr := range deployConfig.LastDeployed.Job.Summary.PlanMessages.ErrorMessages {
								options.Testing.Log(fmt.Sprintf("[PROJECTS] Plan Error: %s", planErr))
							}
						} else {
							options.Testing.Log(fmt.Sprintf("[PROJECTS] No plan error messages found for configuration %s", configName))
						}
						if deployConfig.LastDeployed.Job.Summary.ApplyMessages != nil && deployConfig.LastDeployed.Job.Summary.ApplyMessages.ErrorMessages != nil {
							for _, applyErr := range deployConfig.LastDeployed.Job.Summary.ApplyMessages.ErrorMessages {
								options.Testing.Log(fmt.Sprintf("[PROJECTS] Apply Error: %s", applyErr))
							}
						} else {
							options.Testing.Log(fmt.Sprintf("[PROJECTS] No apply error messages found for configuration %s", configName))
						}
					} else {
						options.Testing.Log(fmt.Sprintf("[PROJECTS] No messages found for configuration %s", configName))
					}

				}
				return fmt.Errorf("deploy failed for configuration %s last state: %s", configName, *deployConfig.State)
			}

			options.Testing.Log(fmt.Sprintf("[PROJECTS] Deployed Configuration %s", configName))
		}
	}
	return nil
}

func (options *TestProjectsOptions) ConfigureTestStack() error {
	// Configure the test stack
	options.Testing.Log("[PROJECTS] Configuring Test Stack")
	var stackResp *core.DetailedResponse
	var stackErr error
	options.currentStack, stackResp, stackErr = options.CloudInfoService.CreateStackFromConfigFileWithInputs(*options.currentProject.ID, options.StackConfigurationPath, options.StackCatalogJsonPath, options.StackInputs)
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

func (options *TestProjectsOptions) SerialDeployConfigurations() error {

	// Loop through the StackConfigurationOrder
	for _, configName := range options.StackConfigurationOrder {
		err := options.ValidateApproveDeploy(configName)
		if err != nil {
			return err
		}
	}
	return nil
}

func (options *TestProjectsOptions) ValidateApproveDeploy(configName string) error {
	if err := options.ValidateConfig(configName); err != nil {
		return err
	}
	if err := options.ApproveConfig(configName); err != nil {
		return err
	}
	if err := options.DeployConfig(configName); err != nil {
		return err
	}

	return nil

}
func (options *TestProjectsOptions) ParallelDeployConfigurations() []error {
	// Get all configurations
	allConfigurations, cfgErr := options.CloudInfoService.GetProjectConfigs(*options.currentProject.ID)
	if cfgErr != nil {
		return []error{cfgErr}
	}
	setUndeployOrder := false
	if options.StackUndeployOrder == nil {
		setUndeployOrder = true
	}
	// create a slice of strings to store the configurations that have been deployed
	deployedConfigurations := make([]string, 0)
	// create a channel to collect errors from goroutines
	errChan := make(chan error, len(allConfigurations)-1) // -1 to account for the stack configuration

	// while all configurations are not deployed
	for len(deployedConfigurations) != len(allConfigurations)-1 {
		// Loop through the StackConfigurationOrder identify any configurations that are already deployed
		currentDeployGroup := make([]string, 0)
		for _, currentConfig := range allConfigurations {
			if common.StrArrayContains(deployedConfigurations, *currentConfig.Definition.Name) {
				continue
			}
			cfg, _, _ := options.CloudInfoService.GetConfig(*options.currentProject.ID, *currentConfig.ID)

			// Ignore the stack configuration
			if *currentConfig.DeploymentModel != "stack" && *currentConfig.State == cloudinfo.DRAFT && *cfg.StateCode != cloudinfo.AWAITING_PREREQUISITE {
				currentDeployGroup = append(currentDeployGroup, *currentConfig.Definition.Name)
				// if setundeploy order and config not in undeploy order, add to undeploy order
				if setUndeployOrder && !common.StrArrayContains(options.StackUndeployOrder, *currentConfig.Definition.Name) {
					// Prepend the name of the current configuration to the StackUndeployOrder slice.
					// This is done by creating a new slice with the current configuration name as the only element,
					// and then appending the existing StackUndeployOrder slice to it.
					// The result is a new slice with the current configuration name at the beginning,
					// followed by the elements of the original StackUndeployOrder slice.
					options.StackUndeployOrder = append([]string{*currentConfig.Definition.Name}, options.StackUndeployOrder...)
				}
			}
		}
		//// if there are no configurations to deploy, break the loop
		//if len(currentDeployGroup) == 0 {
		//	break
		//}
		// Check if there are configurations to deploy
		// 'currentDeployGroup' is a slice that contains the configurations to be deployed
		if len(currentDeployGroup) > 0 {
			// If there are configurations to deploy, we need to add them to the 'stackUndeployGroups'
			// 'stackUndeployGroups' is a 2D slice where each element is a group of configurations that need to be undeployed
			// We want to add the current group of configurations to be deployed at the start of 'stackUndeployGroups'
			// This is because the configurations that are deployed last should be undeployed first
			// 'append' is a built-in function in Go that concatenates slices
			// Here, we are creating a new slice with 'currentDeployGroup' as the first element and the existing 'stackUndeployGroups' as the remaining elements
			// This effectively adds 'currentDeployGroup' to the start of 'stackUndeployGroups'
			options.stackUndeployGroups = append([][]string{currentDeployGroup}, options.stackUndeployGroups...)
		}
		options.Testing.Log(fmt.Sprintf("[Projects] Deploying group %d/X", len(options.stackUndeployGroups)))
		// deploy all currentDeployGroup configurations in parallel, and wait for all deployments to complete
		var wg sync.WaitGroup
		for _, configName := range currentDeployGroup {
			wg.Add(1)
			go func(name string) {
				defer wg.Done()
				options.Testing.Log(fmt.Sprintf("[PROJECTS] Deploying Configuration %s", name)) // Add configuration name to the log
				if err := options.ValidateApproveDeploy(name); err != nil {
					options.Testing.Log("Error deploying configuration %s: %s", name, err)
					errChan <- err // send error to the error channel
				} else {
					// If deployment is successful, add the configuration to the deployed configurations list
					deployedConfigurations = append(deployedConfigurations, name)
				}
			}(configName)
		}
		wg.Wait()
	}

	close(errChan) // close the error channel

	// Check if there were any errors during the deployments
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	return errs
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

	if options.CloudInfoService == nil {
		cloudInfoSvc, err := cloudinfo.NewCloudInfoServiceFromEnv("TF_VAR_ibmcloud_api_key", cloudinfo.CloudInfoServiceOptions{})
		if err != nil {
			return err
		}
		options.CloudInfoService = cloudInfoSvc
	}
	// Create a new project
	options.Testing.Log("[PROJECTS] Creating Test Project")
	prj, resp, err := options.CloudInfoService.CreateDefaultProject(options.ProjectName, options.ProjectDescription, options.ResourceGroup)
	if assert.NoError(options.Testing, err) {
		if assert.Equal(options.Testing, 201, resp.StatusCode) {
			options.Testing.Log(fmt.Sprintf("[PROJECTS] Created Test Project - %s", *prj.Definition.Name))
			options.currentProject = prj

			if assert.NoError(options.Testing, options.ConfigureTestStack()) {
				// ensure all stack members in the current stack are in the stack configuration order failing if not
				for _, stackMember := range options.currentStack.StackDefinition.Members {
					// check if the stack member is in the configuration order if not nil
					if options.StackConfigurationOrder != nil {
						if !assert.Contains(options.Testing, options.StackConfigurationOrder, *stackMember.Name) {
							return fmt.Errorf("stack member %s not in configuration order", *stackMember.Name)
						}
					}
				}

				if !options.ParallelDeploy {
					// Deploy the configuration in stack order
					deployErr := options.SerialDeployConfigurations()
					if !assert.NoError(options.Testing, deployErr) {
						return err
					}
				} else {
					// Deploy the configuration in parallel
					deployErr := options.ParallelDeployConfigurations()
					if !assert.Empty(options.Testing, deployErr) {
						// print all errors and return a single error
						for _, derr := range deployErr {
							options.Testing.Error(derr)
						}
						return fmt.Errorf("error deploying configurations")
					}
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
		// Check if "DO_NOT_DESTROY_ON_FAILURE" is set
		envVal, _ := os.LookupEnv("DO_NOT_DESTROY_ON_FAILURE")

		// Do not destroy if tests failed and "DO_NOT_DESTROY_ON_FAILURE" is true
		if options.Testing.Failed() && strings.ToLower(envVal) == "true" {
			fmt.Println("Terratest failed. Debug the Test and delete resources manually.")
		} else {
			if !options.SkipUndeploy {
				// Undeploy the configuration in stack undeploy order
				if options.StackUndeployOrder != nil {
					if !options.ParallelDeploy {
						// Serial undeploy
						// Get all configurations
						allConfigurations, cfgErr := options.CloudInfoService.GetProjectConfigs(*options.currentProject.ID)
						if !assert.NoError(options.Testing, cfgErr) {
							options.Testing.Log("[PROJECTS] Failed to get configurations during undeploy, skipping undeploy")
						} else {
							options.Testing.Log("[PROJECTS] Undeploying Configurations in Stack Undeploy Order")
							for _, configName := range options.StackUndeployOrder {

								// Get the configuration
								config, configErr := getConfigFromName(configName, allConfigurations)
								if !assert.NoError(options.Testing, configErr) {
									options.Testing.Errorf("Error getting configuration %s: %s", configName, configErr)
									continue
								}
								// Undeploy the configuration
								options.Testing.Log(fmt.Sprintf("[PROJECTS] Undeploying Configuration %s", configName))
								undeployConfig, _, undeployErr := options.CloudInfoService.UndeployConfig(*options.currentProject.ID, *config.ID)
								if assert.NoError(options.Testing, undeployErr) {
									if !assert.Equal(options.Testing, cloudinfo.UNDEPLOYING, *undeployConfig.State) {
										options.Testing.Errorf("Error undeploying configuration %s", configName)
									}
									options.Testing.Log(fmt.Sprintf("[PROJECTS] Undeployed Configuration %s", configName))
								}
							}
						}
					} else {
						// parallel undeploy
						//	undeploy the each group in undeploygroups in parallel one set of groups at a time until all groups are undeployed
						for index, currentUndeployGroup := range options.stackUndeployGroups {
							options.Testing.Log(fmt.Sprintf("[PROJECTS] Undeploying In Parallel group %d/%d", index+1, len(options.stackUndeployGroups)))
							// create a channel to collect errors from goroutines
							errChan := make(chan error, len(currentUndeployGroup))

							// undeploy all currentUndeployGroup configurations in parallel, and wait for all undeployments to complete
							var wg sync.WaitGroup
							for _, configName := range currentUndeployGroup {
								wg.Add(1)
								go func(name string) {
									defer wg.Done()
									options.Testing.Log(fmt.Sprintf("[PROJECTS] Undeploying Configuration %s", name)) // Add configuration name to the log
									// Get all configurations
									allConfigurations, cfgErr := options.CloudInfoService.GetProjectConfigs(*options.currentProject.ID)
									if !assert.NoError(options.Testing, cfgErr) {
										options.Testing.Log("Error getting configurations: %s", cfgErr)
										errChan <- cfgErr
										return
									}
									// Get the configuration
									config, configErr := getConfigFromName(name, allConfigurations)
									if !assert.NoError(options.Testing, configErr) {
										options.Testing.Log("Error getting configuration %s: %s", name, configErr)
										errChan <- configErr
										return
									}
									// Undeploy the configuration
									undeployConfig, _, undeployErr := options.CloudInfoService.UndeployConfig(*options.currentProject.ID, *config.ID)
									if assert.NoError(options.Testing, undeployErr) {
										if !assert.Equal(options.Testing, cloudinfo.UNDEPLOYING, *undeployConfig.State) {
											options.Testing.Log("Error undeploying configuration %s", name)
											errChan <- fmt.Errorf("error undeploying configuration %s", name)
										} else {
											options.Testing.Log(fmt.Sprintf("[PROJECTS] Undeploy Configuration %s started", name))
										}
									} else {
										errChan <- undeployErr
									}

									// wait for undeployment to complete or timeout
									// Set end time
									undeployEndTime := time.Now().Add(time.Duration(options.DeployTimeoutMinutes) * time.Minute)
									for {
										_, isUndeploying := options.CloudInfoService.IsUndeploying(*options.currentProject.ID, *config.ID)

										if !isUndeploying {
											break
										}

										// if the time is greater than the timeout
										// return an error
										if time.Now().After(undeployEndTime) {
											errChan <- fmt.Errorf("undeploy timeout for configuration %s", name)
											return
										}
										options.Testing.Log(fmt.Sprintf("[PROJECTS] Configuration %s is still undeploying", name))
										time.Sleep(30 * time.Second)
									}
									if *config.State != cloudinfo.UNDEPLOYING && *config.State != cloudinfo.UNDEPLOYING_FAILED {
										options.Testing.Log(fmt.Sprintf("[PROJECTS] Undeploying Configuration %s complete", name))
									}
								}(configName)
							}
							wg.Wait()
							close(errChan) // close the error channel

							// Check if there were any errors during the undeployments
							var errs []error
							for err := range errChan {
								errs = append(errs, err)
							}
							if len(errs) > 0 {
								// print all errors and return a single error
								for _, uerr := range errs {
									options.Testing.Log(uerr)
								}
							}
						}
					}
				}
			}
			if !options.SkipProjectDelete {
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
						if *config.State == cloudinfo.VALIDATING || *config.State == cloudinfo.DEPLOYING || *config.State == cloudinfo.UNDEPLOYING {
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
}

func getConfigFromName(configName string, allConfigs []project.ProjectConfigSummary) (*project.ProjectConfigSummary, error) {
	for _, config := range allConfigs {
		if *config.Definition.Name == configName {
			return &config, nil
		}
	}
	return nil, fmt.Errorf("configuration %s not found", configName)
}
