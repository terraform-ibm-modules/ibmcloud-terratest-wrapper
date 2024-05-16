package testprojects

import (
	"errors"
	"fmt"
	"github.com/IBM/go-sdk-core/v5/core"
	project "github.com/IBM/project-go-sdk/projectv1"
	"github.com/stretchr/testify/assert"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/cloudinfo"
	"os"
	"strings"
	"time"
)

// Configurations States
const APPROVED = "approved"
const DELETED = "deleted"
const DELETING = "deleting"
const DELETING_FAILED = "deleting_failed"
const DISCARDED = "discarded"
const DRAFT = "draft"
const DEPLOYED = "deployed"
const DEPLOYING_FAILED = "deploying_failed"
const DEPLOYING = "deploying"
const SUPERSEDED = "superseded"
const UNDEPLOYING = "undeploying"
const UNDEPLOYING_FAILED = "undeploying_failed"
const VALIDATED = "validated"
const VALIDATING = "validating"
const VALIDATING_FAILED = "validating_failed"
const APPLIED = "applied"
const APPLY_FAILED = "apply_failed"

// RunProjectsTest : Run the test for the projects service
// Creates a new project
// Adds a configuration
// Deploys the configuration
// Deletes the project
func (options *TestProjectsOptions) RunProjectsTest() error {

	if !options.SkipTestTearDown {
		defer options.TestTearDown()
	}

	cloudInfoSvc, err := cloudinfo.NewCloudInfoServiceFromEnv("TF_VAR_ibmcloud_api_key", cloudinfo.CloudInfoServiceOptions{})
	if err != nil {
		return err
	}
	// Create a new project
	options.Testing.Log("[PROJECTS] Creating Test Project")
	prj, resp, err := cloudInfoSvc.CreateDefaultProject(options.ProjectName, options.ProjectDescription, options.ResourceGroup)
	if assert.NoError(options.Testing, err) {
		if assert.Equal(options.Testing, resp.StatusCode, 201) {
			options.Testing.Log(fmt.Sprintf("[PROJECTS] Created Test Project - %s", *prj.Definition.Name))
			options.currentProject = prj
			// Deploy the configuration
			options.Testing.Log("[PROJECTS] Deploying Test Stack")
			var stackResp *core.DetailedResponse
			var stackErr error
			options.currentStack, stackResp, stackErr = cloudInfoSvc.CreateStackFromConfigFileWithInputs(*options.currentProject.ID, options.StackConfigurationPath, options.StackCatalogJsonPath, options.StackInputs)

			if assert.NoError(options.Testing, stackErr) {
				if assert.Equal(options.Testing, stackResp.StatusCode, 201) {
					options.Testing.Log("[PROJECTS] Deployed Test Stack")
					allConfigurations, configErr := cloudInfoSvc.GetProjectConfigs(*options.currentProject.ID)
					if !assert.NoError(options.Testing, configErr) {
						return configErr
					}
					// ensure all stack members in the current stack are in the stack configuration order failing if not
					for _, stackMember := range options.currentStack.StackDefinition.Members {
						// check if the stack member is in the configuration order
						if !assert.Contains(options.Testing, options.StackConfigurationOrder, *stackMember.Name) {
							return fmt.Errorf("stack member %s not in configuration order", *stackMember.Name)
						}
					}
					// Validate each configuration in the stack loop through the stack configuration order options.StackConfigurationOrder
					for _, configName := range options.StackConfigurationOrder {

						// Validate the configuration
						currentConfig, currConfigErr := getConfigFromName(configName, allConfigurations)
						if !assert.NoError(options.Testing, currConfigErr) {
							return currConfigErr
						}

						// set authenticator for current member(configuration)
						tempConfig, _, tempConfigErr := cloudInfoSvc.GetConfig(*options.currentProject.ID, *currentConfig.ID)
						if !assert.NoError(options.Testing, tempConfigErr) {
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
											ApiKey: &cloudInfoSvc.ApiKey,
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
								_, updateResponse, updateErr := cloudInfoSvc.UpdateConfig(*options.currentProject.ID, *currentConfig.ID, patchConfig)
								if !assert.NoError(options.Testing, updateErr) {
									return updateErr
								}
								if !assert.Equal(options.Testing, updateResponse.StatusCode, 200) {
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
											ApiKey: &cloudInfoSvc.ApiKey,
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
								_, updateResponse, updateErr := cloudInfoSvc.UpdateConfig(*options.currentProject.ID, *currentConfig.ID, patchConfig)
								if !assert.NoError(options.Testing, updateErr) {
									return updateErr
								}
								if !assert.Equal(options.Testing, updateResponse.StatusCode, 200) {
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
											ApiKey: &cloudInfoSvc.ApiKey,
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
								_, updateResponse, updateErr := cloudInfoSvc.UpdateConfig(*options.currentProject.ID, *currentConfig.ID, patchConfig)
								if !assert.NoError(options.Testing, updateErr) {
									return updateErr
								}
								if !assert.Equal(options.Testing, updateResponse.StatusCode, 200) {
									return fmt.Errorf("error updating configuration %s", configName)
								}

							}
						default:
							options.Testing.Log(fmt.Sprintf("[WARNING] Configuration %s is not supported for setting authorization", configName))
						}

						validateConfig, _, validateErr := cloudInfoSvc.ValidateConfig(*options.currentProject.ID, *currentConfig.ID)
						if assert.NoError(options.Testing, validateErr) {
							// Set end time
							approvalEndTime := time.Now().Add(time.Duration(options.ValidationTimeoutMinutes) * time.Minute)

							if *validateConfig.State == VALIDATING {
								// Wait for the configuration to finish validating
								for *validateConfig.State == VALIDATING {
									// if the time is greater than the timeout
									// return an error
									if time.Now().After(approvalEndTime) {
										return fmt.Errorf("validation timeout for configuration %s", configName)
									}
									options.Testing.Log(fmt.Sprintf("[PROJECTS] Configuration %s is still validating", configName))
									time.Sleep(30 * time.Second)
									validateConfig, _, validateErr = cloudInfoSvc.GetConfigVersion(*options.currentProject.ID, *currentConfig.ID, *currentConfig.Version)
									if !assert.NoError(options.Testing, validateErr) {
										return validateErr
									}
								}
								if *validateConfig.State != VALIDATED {
									schematicsCrn := validateConfig.Schematics.WorkspaceCrn
									if schematicsCrn != nil {
										options.Testing.Log(fmt.Sprintf("[PROJECTS] Configuration %s failed validation, schematics workspace: %s", configName, *schematicsCrn))
										options.Testing.Log(fmt.Sprintf("[PROJECTS] Result: %s", *validateConfig.LastValidated.Result))
										for _, planErr := range validateConfig.LastValidated.Job.Summary.PlanMessages.ErrorMessages {
											options.Testing.Log(fmt.Sprintf("[PROJECTS] Plan Error: %s", planErr))
										}
									}
									return fmt.Errorf("validation failed for configuration %s last state: %s", configName, *validateConfig.State)
								} else {
									// Approve the configuration
									options.Testing.Log(fmt.Sprintf("[PROJECTS] Approving Configuration %s", configName))
									approveConfig, _, approveErr := cloudInfoSvc.ApproveConfig(*options.currentProject.ID, *currentConfig.ID)
									if assert.NoError(options.Testing, approveErr) {
										if assert.Equal(options.Testing, *approveConfig.State, APPROVED) {
											options.Testing.Log(fmt.Sprintf("[PROJECTS] Approved Configuration %s", configName))
											// Deploy the configuration
											options.Testing.Log(fmt.Sprintf("[PROJECTS] Deploying Configuration %s", configName))
											deployConfig, _, deployErr := cloudInfoSvc.DeployConfig(*options.currentProject.ID, *currentConfig.ID)
											if assert.NoError(options.Testing, deployErr) {
												// Set end time
												deployEndTime := time.Now().Add(time.Duration(options.DeployTimeoutMinutes) * time.Minute)

												if *deployConfig.State == DEPLOYING {
													// Wait for the configuration to finish deploying
													for *deployConfig.State == DEPLOYING {
														// if the time is greater than the timeout
														// return an error
														if time.Now().After(deployEndTime) {
															return fmt.Errorf("deploy timeout for configuration %s", configName)
														}
														options.Testing.Log(fmt.Sprintf("[PROJECTS] Configuration %s is still deploying", configName))
														time.Sleep(30 * time.Second)
														deployConfig, _, deployErr = cloudInfoSvc.GetConfigVersion(*options.currentProject.ID, *currentConfig.ID, *currentConfig.Version)
														if !assert.NoError(options.Testing, deployErr) {
															return deployErr
														}
													}
													if *deployConfig.State != DEPLOYED {
														schematicsCrn := deployConfig.Schematics.WorkspaceCrn
														if schematicsCrn != nil {
															options.Testing.Log(fmt.Sprintf("[PROJECTS] Configuration %s failed deploy, schematics workspace: %s", configName, *schematicsCrn))
															options.Testing.Log(fmt.Sprintf("[PROJECTS] Result: %s", *deployConfig.LastDeployed.Result))
															for _, applyErr := range deployConfig.LastDeployed.Job.Summary.ApplyMessages.ErrorMessages {
																options.Testing.Log(fmt.Sprintf("[PROJECTS] Apply Error: %s", applyErr))
															}
														}
														return fmt.Errorf("deploy failed for configuration %s last state: %s", configName, *deployConfig.State)
													}
													if *deployConfig.State == DEPLOYED {
														options.Testing.Log(fmt.Sprintf("[PROJECTS] Deployed Configuration %s", configName))
													}
												}
											}
										} else {
											options.Testing.Log(fmt.Sprintf("[PROJECTS] Configuration %s failed to approve", configName))
											return fmt.Errorf("error approving configuration %s", configName)
										}
									} else {
										options.Testing.Log(fmt.Sprintf("[PROJECTS] Configuration %s failed to approve", configName))
										return approveErr
									}

								}
							}
						} else {
							options.Testing.Log(fmt.Sprintf("[PROJECTS] Configuration %s is not in validating state", configName))
							return validateErr
						}
					}
				} else {
					options.Testing.Log("[PROJECTS] Failed to deploy Test Stack")
					return fmt.Errorf("error deploying stack statuscode %d details: %s", stackResp.StatusCode, stackResp.String())
				}
			} else {
				options.Testing.Log("[PROJECTS] Failed to deploy Test Stack")
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
							options.Testing.Error("Error getting version locator from stack definition")
							return sdkProblem
						}
						// get inputs for the member config of version
						version, vererr := cloudInfoSvc.GetVersion(versionLocator)
						if !assert.NoError(options.Testing, vererr) {
							options.Testing.Error("Error getting offering")
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
				}
				return stackErr
			}

		}
	} else {
		options.Testing.Log("[PROJECTS] Failed to create Test Project")
		return err
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

			// TODO: Is there a better way to handle this?
			cloudInfoSvc, err := cloudinfo.NewCloudInfoServiceFromEnv("TF_VAR_ibmcloud_api_key", cloudinfo.CloudInfoServiceOptions{})
			if err != nil {
				options.Testing.Errorf("Error creating CloudInfoService: %s", err)
				return
			}
			// Delete the project
			// TODO: Wait until all validation is complete before deleting the project
			//       Delete will fail while jobs are running
			options.Testing.Log("[PROJECTS] Deleting Test Project")
			_, resp, err := cloudInfoSvc.DeleteProject(*options.currentProject.ID)
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
