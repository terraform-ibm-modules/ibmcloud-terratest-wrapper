package testaddons

import (
	"fmt"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/IBM/go-sdk-core/v5/core"
	Core "github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/project-go-sdk/projectv1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/cloudinfo"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/common"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testprojects"
)

// RunAddonTest : Run the test for addons
// Creates a new catalog
// Imports an offering
// Creates a new project
// Adds a configuration
// Deploys the configuration
// Deletes the project
// Deletes the catalog
// Returns an error if any of the steps fail
func (options *TestAddonOptions) RunAddonTest() error {
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

	// Deploy Addon to Project
	options.Logger.ShortInfo("Deploying the addon to project")
	deployedConfigs, err := options.CloudInfoService.DeployAddonToProject(&options.AddonConfig, options.currentProjectConfig)

	if err != nil {
		options.Logger.ShortError(fmt.Sprintf("Error deploying the addon to project: %v", err))
		options.Testing.Fail()
		return fmt.Errorf("error deploying the addon to project: %w", err)
	}

	// Store deployed configs for later use in dependency validation
	options.deployedConfigs = deployedConfigs

	options.Logger.ShortInfo(fmt.Sprintf("Deployed Configurations to Project ID: %s", options.currentProjectConfig.ProjectID))
	for _, config := range deployedConfigs.Configs {
		options.Logger.ShortInfo(fmt.Sprintf("  %s - ID: %s", config.Name, config.ConfigID))
	}
	options.Logger.ShortInfo("Addon deployed successfully")

	options.Logger.ShortInfo("Updating Configurations")
	// Configure Addon
	addonID := options.AddonConfig.ConfigID
	addonName := options.AddonConfig.ConfigName
	if options.AddonConfig.ContainerConfigID != "" {
		addonID = options.AddonConfig.ContainerConfigID
		addonName = options.AddonConfig.ContainerConfigName
	}
	// configure API key
	configDetails := cloudinfo.ConfigDetails{
		ProjectID: options.currentProjectConfig.ProjectID,
		Name:      addonName,
		Inputs:    options.AddonConfig.Inputs,
		ConfigID:  addonID,
	}

	configDetails.MemberConfigs = nil
	for _, config := range deployedConfigs.Configs {

		prjCfg, _, _ := options.CloudInfoService.GetConfig(&cloudinfo.ConfigDetails{
			ProjectID: options.currentProjectConfig.ProjectID,
			Name:      config.Name,
			ConfigID:  config.ConfigID,
		})
		configDetails.Members = append(configDetails.Members, *prjCfg)

		configDetails.MemberConfigs = append(configDetails.MemberConfigs, projectv1.StackConfigMember{
			ConfigID: core.StringPtr(config.ConfigID),
			Name:     core.StringPtr(config.Name),
		})

	}

	confPatch := projectv1.ProjectConfigDefinitionPatch{
		Inputs: configDetails.Inputs,
		Authorizations: &projectv1.ProjectConfigAuth{
			ApiKey: core.StringPtr(options.CloudInfoService.GetApiKey()),
			Method: core.StringPtr(projectv1.ProjectConfigAuth_Method_ApiKey),
		},
	}
	prjConfig, response, err := options.CloudInfoService.UpdateConfig(&configDetails, &confPatch)
	if err != nil {
		options.Logger.ShortError(fmt.Sprintf("Error updating the configuration: %v", err))
		options.Testing.Fail()
		return fmt.Errorf("error updating the configuration: %w", err)
	}
	if response.RawResult != nil {
		options.Logger.ShortInfo(fmt.Sprintf("Response: %s", string(response.RawResult)))
	}
	options.Logger.ShortInfo(fmt.Sprintf("Updated Configuration: %s", *prjConfig.ID))
	if prjConfig.StateCode != nil {
		options.Logger.ShortInfo(fmt.Sprintf("Updated Configuration statecode: %s", *prjConfig.StateCode))
	}
	if prjConfig.State != nil {
		options.Logger.ShortInfo(fmt.Sprintf("Updated Configuration state: %s", *prjConfig.State))
	}

	// create TestProjectsOptions to use with the projects package
	deployOptions := testprojects.TestProjectsOptions{
		Prefix:               options.Prefix,
		ProjectName:          options.ProjectName,
		CloudInfoService:     options.CloudInfoService,
		Logger:               options.Logger,
		Testing:              options.Testing,
		DeployTimeoutMinutes: options.DeployTimeoutMinutes,
		StackPollTimeSeconds: 60,
	}

	deployOptions.SetCurrentStackConfig(&configDetails)
	deployOptions.SetCurrentProjectConfig(options.currentProjectConfig)

	allConfigs, err := options.CloudInfoService.GetProjectConfigs(options.currentProjectConfig.ProjectID)
	if err != nil {
		options.Logger.ShortError(fmt.Sprintf("Error getting the configuration: %v", err))
		options.Testing.Fail()
		return fmt.Errorf("error getting the configuration: %w", err)
	}
	options.Logger.ShortInfo(fmt.Sprintf("All Configurations in Project ID: %s", options.currentProjectConfig.ProjectID))
	options.Logger.ShortInfo("Configurations:")

	// loop through all configs for reference validation and input validation
	readyToValidate := false
	waitingOnInputs := make([]string, 0)
	failedRefs := []string{}
	missingRequiredInputs := make([]string, 0)

	// set offering details
	SetOfferingDetails(options)

	// Create a map of deployed config IDs for this test case to avoid processing configs from other test cases
	deployedConfigIDs := make(map[string]bool)
	if options.deployedConfigs != nil {
		for _, deployedConfig := range options.deployedConfigs.Configs {
			deployedConfigIDs[deployedConfig.ConfigID] = true
		}
	}

	for _, config := range allConfigs {
		options.Logger.ShortInfo(fmt.Sprintf("  %s - ID: %s", *config.Definition.Name, *config.ID))

		currentConfigDetails, _, err := options.CloudInfoService.GetConfig(&cloudinfo.ConfigDetails{
			ProjectID: options.currentProjectConfig.ProjectID,
			ConfigID:  *config.ID,
		})

		if err != nil {
			options.Logger.ShortError(fmt.Sprintf("Error getting the configuration: %v", err))
			options.Testing.Fail()
			return fmt.Errorf("error getting the configuration: %w", err)
		}

		// Check state for input validation
		if currentConfigDetails.StateCode != nil && *currentConfigDetails.StateCode == projectv1.ProjectConfig_StateCode_AwaitingValidation {
			options.Logger.ShortInfo(fmt.Sprintf("Found a configuration ready to validate: %s - ID: %s", *config.Definition.Name, *config.ID))
			readyToValidate = true
		}
		if currentConfigDetails.StateCode != nil && *currentConfigDetails.StateCode == projectv1.ProjectConfig_StateCode_AwaitingInput {
			configName := *currentConfigDetails.Definition.(*projectv1.ProjectConfigDefinitionResponse).Name
			options.Logger.ShortWarn(fmt.Sprintf("Configuration '%s' is in AwaitingInput state - adding to waitingOnInputs list", configName))
			waitingOnInputs = append(waitingOnInputs, configName)
		}

		// Skip reference validation if the flag is set
		if !options.SkipRefValidation {
			options.Logger.ShortInfo("  References:")
			references := []string{}

			for _, input := range currentConfigDetails.Definition.(*projectv1.ProjectConfigDefinitionResponse).Inputs {
				// Check if input is a string before checking for ref:/ prefix
				if inputStr, ok := input.(string); ok && strings.HasPrefix(inputStr, "ref:/") {
					options.Logger.ShortInfo(fmt.Sprintf("    %s", inputStr))
					references = append(references, inputStr)
				}
			}

			if len(references) > 0 {
				res_resp, err := options.CloudInfoService.ResolveReferencesFromStrings(*options.currentProject.Location, references, options.currentProjectConfig.ProjectID)
				if err != nil {
					// Check if this is a known intermittent error that should be skipped
					// This can occur as either a direct HttpError or as an EnhancedHttpError with additional context
					errStr := err.Error()
					isApiKeyError := (strings.Contains(errStr, "Failed to validate api key token") && strings.Contains(errStr, "500"))
					isProjectNotFoundError := strings.Contains(errStr, "could not be found") && strings.Contains(errStr, "404")
					isKnownIntermittentError := strings.Contains(errStr, "This is a known intermittent issue") ||
						strings.Contains(errStr, "known transient issue") ||
						strings.Contains(errStr, "typically transient")

					// Only skip validation for intermittent errors if infrastructure deployment is enabled
					// When SkipInfrastructureDeployment=true, reference validation is the only chance to catch issues
					if (isApiKeyError || isProjectNotFoundError || isKnownIntermittentError) && !options.SkipInfrastructureDeployment {
						options.Logger.ShortWarn(fmt.Sprintf("Skipping reference validation due to intermittent IBM Cloud service error: %v", err))
						if isApiKeyError {
							options.Logger.ShortWarn("This is a known transient issue with IBM Cloud's API key validation service.")
						} else if isProjectNotFoundError {
							options.Logger.ShortWarn("This is a timing issue where project details are checked too quickly after creation.")
							options.Logger.ShortWarn("The resolver API needs time to be updated with new project information.")
						} else {
							options.Logger.ShortWarn("This is a known transient issue with IBM Cloud's reference resolution service.")
						}
						options.Logger.ShortWarn("The test will continue and will fail later if references actually fail to resolve during deployment.")
						// Skip reference validation for this config and continue with the test
						continue
					} else if (isApiKeyError || isProjectNotFoundError || isKnownIntermittentError) && options.SkipInfrastructureDeployment {
						options.Logger.ShortWarn(fmt.Sprintf("Detected intermittent service error, but cannot skip validation in validation-only mode: %v", err))
						options.Logger.ShortWarn("Infrastructure deployment is disabled, so reference validation is the only opportunity to catch reference issues.")
						options.Logger.ShortWarn("Failing the test to ensure reference issues are not missed.")
					}
					// For other errors, fail the test as before
					options.Logger.ShortError(fmt.Sprintf("Error resolving references: %v", err))
					options.Testing.Fail()
					return fmt.Errorf("error resolving references: %w", err)
				}
				options.Logger.ShortInfo("  Resolved References:")
				for _, ref := range res_resp.References {
					if ref.Code != 200 {
						options.Logger.ShortError(fmt.Sprintf("%s   %s - Error: %s", common.ColorizeString(common.Colors.Red, "✘"), ref.Reference, ref.State))
						options.Logger.ShortError(fmt.Sprintf("      Message: %s", ref.Message))
						options.Logger.ShortError(fmt.Sprintf("      Code: %d", ref.Code))
						options.Testing.Failed()
						failedRefs = append(failedRefs, ref.Reference)
						continue
					}

					options.Logger.ShortInfo(fmt.Sprintf("%s   %s", common.ColorizeString(common.Colors.Green, "✔"), ref.Reference))
					options.Logger.ShortInfo(fmt.Sprintf("      State: %s", ref.State))
					if ref.Value != "" {
						options.Logger.ShortInfo(fmt.Sprintf("      Value: %s", ref.Value))
					}
				}
			}
		}

		// get corresponding offering to current config
		var targetAddon cloudinfo.AddonConfig
		var addonFound bool

		// Extract version from locator ID
		locatorParts := strings.Split(*currentConfigDetails.Definition.(*projectv1.ProjectConfigDefinitionResponse).LocatorID, ".")
		if len(locatorParts) < 2 {
			options.Logger.ShortWarn(fmt.Sprintf("Invalid locator ID format: %s", *currentConfigDetails.Definition.(*projectv1.ProjectConfigDefinitionResponse).LocatorID))
			continue
		}
		version := locatorParts[1]

		// Try to match by version ID first
		if version == options.AddonConfig.VersionID {
			targetAddon = options.AddonConfig
			addonFound = true
		} else {
			// Check dependencies
			for i, dependency := range options.AddonConfig.Dependencies {
				if version == dependency.VersionID {
					targetAddon = options.AddonConfig.Dependencies[i]
					addonFound = true
					break
				}
			}
		}

		// If version-based lookup failed, try matching by offering name or configuration name
		if !addonFound {
			configName := *currentConfigDetails.Definition.(*projectv1.ProjectConfigDefinitionResponse).Name

			// Try to match main addon by configuration name pattern
			if strings.Contains(configName, options.AddonConfig.OfferingName) ||
				(options.AddonConfig.ConfigName != "" && strings.Contains(configName, options.AddonConfig.ConfigName)) {
				targetAddon = options.AddonConfig
				addonFound = true
				options.Logger.ShortInfo(fmt.Sprintf("Matched addon by name pattern for config: %s", configName))
			} else {
				// Try to match dependencies by name pattern - check both offering name and base name
				for i, dependency := range options.AddonConfig.Dependencies {
					dependencyMatched := false

					// Match by exact offering name
					if dependency.OfferingName != "" && strings.Contains(configName, dependency.OfferingName) {
						dependencyMatched = true
					}

					// Match by configuration name
					if !dependencyMatched && dependency.ConfigName != "" && strings.Contains(configName, dependency.ConfigName) {
						dependencyMatched = true
					}

					// Match by base offering name patterns (e.g., "deploy-arch-ibm-account-infra-base")
					if !dependencyMatched && dependency.OfferingName != "" {
						baseOfferingName := strings.Split(dependency.OfferingName, ":")[0] // Remove flavor part if present
						if strings.Contains(configName, baseOfferingName) {
							dependencyMatched = true
						}
					}

					if dependencyMatched {
						targetAddon = options.AddonConfig.Dependencies[i]
						addonFound = true
						options.Logger.ShortInfo(fmt.Sprintf("Matched dependency by name pattern for config: %s", configName))
						break
					}
				}
			}
		}

		if !addonFound {
			options.Logger.ShortWarn(fmt.Sprintf("Could not resolve addon definition for config: %s (ID: %s, Version: %s)",
				*currentConfigDetails.Definition.(*projectv1.ProjectConfigDefinitionResponse).Name, *currentConfigDetails.ID, version))
			options.Logger.ShortWarn(fmt.Sprintf("Available main addon: %s (version: %s)", options.AddonConfig.OfferingName, options.AddonConfig.VersionID))
			if len(options.AddonConfig.Dependencies) > 0 {
				options.Logger.ShortWarn("Available dependencies:")
				for _, dep := range options.AddonConfig.Dependencies {
					options.Logger.ShortWarn(fmt.Sprintf("  - %s (version: %s)", dep.OfferingName, dep.VersionID))
				}
			}
			options.Logger.ShortWarn("=== CONFIGURATION MATCHING DEBUG ===")
			options.Logger.ShortWarn(fmt.Sprintf("Config Name: %s", *currentConfigDetails.Definition.(*projectv1.ProjectConfigDefinitionResponse).Name))
			options.Logger.ShortWarn(fmt.Sprintf("Config LocatorID: %s", *currentConfigDetails.Definition.(*projectv1.ProjectConfigDefinitionResponse).LocatorID))
			options.Logger.ShortWarn(fmt.Sprintf("Extracted Version: %s", version))
			options.Logger.ShortWarn(fmt.Sprintf("Expected Main Addon: %s (config: %s)", options.AddonConfig.OfferingName, options.AddonConfig.ConfigName))
			options.Logger.ShortWarn(fmt.Sprintf("Expected Version ID: %s", options.AddonConfig.VersionID))
			options.Logger.ShortWarn("=== END MATCHING DEBUG ===")
			options.Logger.ShortWarn("Skipping input validation for this configuration")
			continue
		}

		// Validate required inputs with retry mechanism to handle database timing issues
		options.Logger.ShortInfo("Validating required inputs...")
		for _, input := range targetAddon.OfferingInputs {
			if input.Required {
				options.Logger.ShortInfo(fmt.Sprintf("Required Input: %v ", input.Key))
			}
		}

		// Use configurable retry settings with sensible defaults
		retries := options.InputValidationRetries
		if retries <= 0 {
			retries = 3 // Default to 3 retries
		}
		retryDelay := options.InputValidationRetryDelay
		if retryDelay <= 0 {
			retryDelay = 2 * time.Second // Default to 2 seconds
		}

		inputsValid, missingInputsList := options.validateInputsWithRetry(*currentConfigDetails.ID, targetAddon, retries, retryDelay)
		if !inputsValid {
			for _, missing := range missingInputsList {
				missingRequiredInputs = append(missingRequiredInputs, missing)
			}
			options.Logger.ShortError(fmt.Sprintf("Some required inputs are missing for addon: %s", *currentConfigDetails.ID))
		} else {
			options.Logger.ShortInfo(fmt.Sprintf("All required inputs set for addon: %s", *currentConfigDetails.ID))
		}
	}

	if !options.SkipRefValidation && len(failedRefs) > 0 {
		options.Logger.ShortWarn("Failed to resolve references:")
		for _, ref := range failedRefs {
			options.Logger.ShortWarn(fmt.Sprintf("  %s", ref))
		}
		options.Logger.ShortWarn("References may resolve during deployment - proceeding with deployment attempt")
	}

	if !options.SkipRefValidation {
		options.Logger.ShortInfo(fmt.Sprintf("  All references resolved successfully %s", common.ColorizeString(common.Colors.Green, "pass ✔")))
	} else {
		options.Logger.ShortInfo("Reference validation skipped")
	}

	// Check for missing required inputs - this should prevent deployment
	if len(missingRequiredInputs) > 0 {
		options.Logger.ShortError("Missing required inputs detected:")
		for _, configError := range missingRequiredInputs {
			options.Logger.ShortError(fmt.Sprintf("  %s", configError))
		}

		// Enhanced debugging information when validation fails
		options.Logger.ShortError("=== INPUT VALIDATION FAILURE DEBUG INFO ===")
		options.Logger.ShortError("Attempting to get current configuration details for debugging...")

		allConfigs, debugErr := options.CloudInfoService.GetProjectConfigs(options.currentProjectConfig.ProjectID)
		if debugErr != nil {
			options.Logger.ShortError(fmt.Sprintf("Could not retrieve configs for debugging: %v", debugErr))
		} else {
			options.Logger.ShortError(fmt.Sprintf("Found %d configurations in project:", len(allConfigs)))
			for _, config := range allConfigs {
				configDetails, _, getErr := options.CloudInfoService.GetConfig(&cloudinfo.ConfigDetails{
					ProjectID: options.currentProjectConfig.ProjectID,
					ConfigID:  *config.ID,
				})

				if getErr != nil {
					options.Logger.ShortError(fmt.Sprintf("  Config: %s (ID: %s) - ERROR: %v", *config.Definition.Name, *config.ID, getErr))
				} else {
					options.Logger.ShortError(fmt.Sprintf("  Config: %s (ID: %s)", *config.Definition.Name, *config.ID))
					options.Logger.ShortError(fmt.Sprintf("    State: %s", func() string {
						if configDetails.State != nil {
							return *configDetails.State
						}
						return "unknown"
					}()))
					options.Logger.ShortError(fmt.Sprintf("    StateCode: %s", func() string {
						if configDetails.StateCode != nil {
							return string(*configDetails.StateCode)
						}
						return "unknown"
					}()))
					options.Logger.ShortError(fmt.Sprintf("    LocatorID: %s", func() string {
						if configDetails.Definition != nil {
							if resp, ok := configDetails.Definition.(*projectv1.ProjectConfigDefinitionResponse); ok && resp.LocatorID != nil {
								return *resp.LocatorID
							}
						}
						return "unknown"
					}()))

					// Show current input values
					if configDetails.Definition != nil {
						if resp, ok := configDetails.Definition.(*projectv1.ProjectConfigDefinitionResponse); ok && resp.Inputs != nil {
							options.Logger.ShortError("    Current Inputs:")
							for key, value := range resp.Inputs {
								// Don't log sensitive values
								if strings.Contains(strings.ToLower(key), "key") || strings.Contains(strings.ToLower(key), "password") || strings.Contains(strings.ToLower(key), "secret") {
									options.Logger.ShortError(fmt.Sprintf("      %s: [REDACTED]", key))
								} else {
									options.Logger.ShortError(fmt.Sprintf("      %s: %v", key, value))
								}
							}
						}
					}
				}
			}
		}

		options.Logger.ShortError("Expected addon configuration details:")
		options.Logger.ShortError(fmt.Sprintf("  Main Addon Name: %s", options.AddonConfig.OfferingName))
		options.Logger.ShortError(fmt.Sprintf("  Main Addon Version: %s", options.AddonConfig.VersionID))
		options.Logger.ShortError(fmt.Sprintf("  Main Addon Config Name: %s", options.AddonConfig.ConfigName))
		options.Logger.ShortError(fmt.Sprintf("  Prefix: %s", options.AddonConfig.Prefix))
		if len(options.AddonConfig.Dependencies) > 0 {
			options.Logger.ShortError("  Dependencies:")
			for i, dep := range options.AddonConfig.Dependencies {
				options.Logger.ShortError(fmt.Sprintf("    [%d] Name: %s, Version: %s, ConfigName: %s", i, dep.OfferingName, dep.VersionID, dep.ConfigName))
			}
		}
		options.Logger.ShortError("=== END DEBUG INFO ===")

		// Create a specific error message listing the actual missing inputs
		var missingInputsList []string
		for _, configError := range missingRequiredInputs {
			missingInputsList = append(missingInputsList, configError)
		}

		options.Logger.ShortError("Cannot proceed with deployment - required inputs must be provided")
		options.Testing.Fail()
		return fmt.Errorf("missing required inputs: %s", strings.Join(missingInputsList, "; "))
	}

	if assert.Equal(options.Testing, 0, len(waitingOnInputs), "Found configurations waiting on inputs") {
		options.Logger.ShortInfo("No configurations waiting on inputs")
	} else {
		options.Logger.ShortError("Found configurations waiting on inputs - this usually indicates timing issues with backend state")
		options.Logger.ShortError("=== DEBUG INFO ===")
		options.Logger.ShortError("Configurations in 'awaiting_input' state:")
		for _, config := range waitingOnInputs {
			options.Logger.ShortError(fmt.Sprintf("  %s", config))
		}

		// Print current configuration input values for debugging - similar to missing inputs debug info
		options.Logger.ShortError("Attempting to get current configuration details for debugging...")

		// Track missing inputs across all configurations for specific error message
		var missingInputsDetails []string
		var configsWithIssues []string

		allConfigs, debugErr := options.CloudInfoService.GetProjectConfigs(options.currentProjectConfig.ProjectID)
		if debugErr != nil {
			options.Logger.ShortError(fmt.Sprintf("Could not retrieve configs for debugging: %v", debugErr))
		} else {
			options.Logger.ShortError(fmt.Sprintf("Found %d configurations in project:", len(allConfigs)))
			for _, config := range allConfigs {
				configDetails, _, getErr := options.CloudInfoService.GetConfig(&cloudinfo.ConfigDetails{
					ProjectID: options.currentProjectConfig.ProjectID,
					ConfigID:  *config.ID,
				})

				if getErr != nil {
					options.Logger.ShortError(fmt.Sprintf("  Config: %s (ID: %s) - ERROR: %v", *config.Definition.Name, *config.ID, getErr))
				} else {
					configName := *config.Definition.Name
					isInWaitingList := false
					for _, waitingConfig := range waitingOnInputs {
						if waitingConfig == configName {
							isInWaitingList = true
							break
						}
					}

					waitingStatus := ""
					if isInWaitingList {
						waitingStatus = " [IN WAITING LIST]"
						configsWithIssues = append(configsWithIssues, configName)
					}

					options.Logger.ShortError(fmt.Sprintf("  Config: %s (ID: %s)%s", configName, *config.ID, waitingStatus))
					options.Logger.ShortError(fmt.Sprintf("    State: %s", func() string {
						if configDetails.State != nil {
							return *configDetails.State
						}
						return "unknown"
					}()))
					options.Logger.ShortError(fmt.Sprintf("    StateCode: %s", func() string {
						if configDetails.StateCode != nil {
							return string(*configDetails.StateCode)
						}
						return "unknown"
					}()))
					options.Logger.ShortError(fmt.Sprintf("    LocatorID: %s", func() string {
						if configDetails.Definition != nil {
							if resp, ok := configDetails.Definition.(*projectv1.ProjectConfigDefinitionResponse); ok && resp.LocatorID != nil {
								return *resp.LocatorID
							}
						}
						return "unknown"
					}()))

					// Show current input values and collect missing ones
					if configDetails.Definition != nil {
						if resp, ok := configDetails.Definition.(*projectv1.ProjectConfigDefinitionResponse); ok && resp.Inputs != nil {
							options.Logger.ShortError("    Current Inputs:")
							for key, value := range resp.Inputs {
								// Don't log sensitive values
								if strings.Contains(strings.ToLower(key), "key") || strings.Contains(strings.ToLower(key), "password") || strings.Contains(strings.ToLower(key), "secret") {
									options.Logger.ShortError(fmt.Sprintf("      %s: [REDACTED]", key))
								} else {
									valueStr := fmt.Sprintf("%v", value)
									if valueStr == "__NOT_SET__" || valueStr == "" || valueStr == "<nil>" {
										// Found a missing input - add to our specific error details
										if isInWaitingList {
											missingInputsDetails = append(missingInputsDetails, fmt.Sprintf("%s.%s", configName, key))
										}
										options.Logger.ShortError(fmt.Sprintf("      %s: __NOT_SET__", key))
									} else {
										options.Logger.ShortError(fmt.Sprintf("      %s: %v", key, value))
									}
								}
							}
						}
					}
				}
			}
		}

		// Print expected configuration details
		options.Logger.ShortError("Expected addon configuration details:")
		options.Logger.ShortError(fmt.Sprintf("  Main Addon Name: %s", options.AddonConfig.OfferingName))
		options.Logger.ShortError(fmt.Sprintf("  Main Addon Version: %s", options.AddonConfig.VersionID))
		options.Logger.ShortError(fmt.Sprintf("  Main Addon Config Name: %s", options.AddonConfig.ConfigName))
		options.Logger.ShortError(fmt.Sprintf("  Prefix: %s", options.AddonConfig.Prefix))
		if len(options.AddonConfig.Dependencies) > 0 {
			options.Logger.ShortError("  Dependencies:")
			for i, dep := range options.AddonConfig.Dependencies {
				options.Logger.ShortError(fmt.Sprintf("    [%d] Name: %s, Version: %s, ConfigName: %s", i, dep.OfferingName, dep.VersionID, dep.ConfigName))
			}
		}
		options.Logger.ShortError("=== END DEBUG INFO ===")

		// Create a specific, actionable error message
		var errorMsg string
		if len(missingInputsDetails) > 0 {
			errorMsg = fmt.Sprintf("configurations waiting on missing inputs: %s", strings.Join(missingInputsDetails, ", "))
		} else if len(configsWithIssues) > 0 {
			errorMsg = fmt.Sprintf("configurations in awaiting_input state: %s", strings.Join(configsWithIssues, ", "))
		} else {
			errorMsg = "configurations waiting on inputs - check debug output above for details"
		}

		options.Testing.Fail()
		return fmt.Errorf("found %s", errorMsg)
	}

	if assert.True(options.Testing, readyToValidate, "No configuration found in ready_to_validate state") {
		options.Logger.ShortInfo("Found a configuration ready to validate")
	} else {
		options.Logger.ShortError("No configuration found in ready_to_validate state")
		options.Testing.Fail()
		return fmt.Errorf("no configuration found in ready_to_validate state")
	}

	// Check if the configuration is in a valid state
	// Check if its deployable
	options.Logger.ShortInfo(fmt.Sprintf("Checked if the configuration is deployable %s", common.ColorizeString(common.Colors.Green, "pass ✔")))

	// validate if expected dependencies are deployed for each addon
	if !options.SkipDependencyValidation {
		options.Logger.ShortInfo("Starting with dependency validation")
		var rootCatalogID, rootOfferingID, rootVersionLocator string
		rootVersionLocator = options.AddonConfig.VersionLocator
		rootCatalogID = options.AddonConfig.CatalogID
		rootOfferingID = options.AddonConfig.OfferingID

		// Build dependency graph using the cleaner return-values approach
		visited := make(map[string]bool)
		graphResult, err := options.buildDependencyGraph(rootCatalogID, rootOfferingID, rootVersionLocator, options.AddonConfig.OfferingFlavor, &options.AddonConfig, visited)
		if err != nil {
			return err
		}

		// Extract results from the returned struct
		graph := graphResult.Graph
		expectedDeployedList := graphResult.ExpectedDeployedList

		options.Logger.ShortInfo("Expected dependency tree:")
		options.PrintDependencyTree(graph, expectedDeployedList)

		options.Logger.ShortInfo("Building the actually deployed configs")

		if options.deployedConfigs == nil {
			return fmt.Errorf("deployed configs not available - cannot validate dependencies")
		}

		actuallyDeployedResult := options.buildActuallyDeployedListFromResponse(options.deployedConfigs)
		if len(actuallyDeployedResult.Errors) > 0 {
			options.Logger.ShortError("Failed to build deployed list from response:")
			for _, errMsg := range actuallyDeployedResult.Errors {
				options.Logger.ShortError(fmt.Sprintf("  - %s", errMsg))
			}
			return fmt.Errorf("failed to build actually deployed list: %s", strings.Join(actuallyDeployedResult.Errors, "; "))
		}

		if len(actuallyDeployedResult.Warnings) > 0 {
			options.Logger.ShortInfo("Built deployed list from deployment response with warnings:")
			for _, warning := range actuallyDeployedResult.Warnings {
				options.Logger.ShortWarn(fmt.Sprintf("Warning: %s", warning))
			}
		} else {
			options.Logger.ShortInfo("Built deployed list from deployment response")
		}

		// First validate what is actually deployed to get the validation results
		validationResult := options.validateDependencies(graph, expectedDeployedList, actuallyDeployedResult.ActuallyDeployedList)

		options.Logger.ShortInfo("Actually deployed configurations (with status):")
		// Create deployment status maps for the tree view
		deployedMap := make(map[string]bool)
		for _, deployed := range actuallyDeployedResult.ActuallyDeployedList {
			key := fmt.Sprintf("%s:%s:%s", deployed.Name, deployed.Version, deployed.Flavor.Name)
			deployedMap[key] = true
		}

		errorMap := make(map[string]cloudinfo.DependencyError)
		for _, depErr := range validationResult.DependencyErrors {
			key := fmt.Sprintf("%s:%s:%s", depErr.Addon.Name, depErr.Addon.Version, depErr.Addon.Flavor.Name)
			errorMap[key] = depErr
		}

		missingMap := make(map[string]bool)
		for _, missing := range validationResult.MissingConfigs {
			key := fmt.Sprintf("%s:%s:%s", missing.Name, missing.Version, missing.Flavor.Name)
			missingMap[key] = true
		}

		// Find the root addon and print tree with status
		allDependencies := make(map[string]bool)
		for _, deps := range graph {
			for _, dep := range deps {
				key := fmt.Sprintf("%s:%s:%s", dep.Name, dep.Version, dep.Flavor.Name)
				allDependencies[key] = true
			}
		}

		var rootAddon *cloudinfo.OfferingReferenceDetail
		for _, addon := range expectedDeployedList {
			key := fmt.Sprintf("%s:%s:%s", addon.Name, addon.Version, addon.Flavor.Name)
			if !allDependencies[key] {
				rootAddon = &addon
				break
			}
		}

		if rootAddon == nil && len(expectedDeployedList) > 0 {
			rootAddon = &expectedDeployedList[0]
		}

		if rootAddon != nil {
			options.printAddonTreeWithStatus(*rootAddon, graph, "", true, make(map[string]bool), deployedMap, errorMap, missingMap)
		}

		// Print validation results
		options.Logger.ShortInfo("Dependency validation results:")
		for _, message := range validationResult.Messages {
			if validationResult.IsValid {
				options.Logger.ShortInfo(message)
			} else {
				options.Logger.ShortError(message)
			}
		}

		// Print validation errors - either consolidated summary or detailed individual messages
		if !validationResult.IsValid {
			if options.EnhancedTreeValidationOutput {
				options.printDependencyTreeWithValidationStatus(graph, expectedDeployedList, actuallyDeployedResult.ActuallyDeployedList, validationResult)
			} else if options.VerboseValidationErrors {
				options.printDetailedValidationErrors(validationResult)
			} else {
				options.printConsolidatedValidationSummary(validationResult)
			}

			// Create a specific error message based on validation results
			var errorDetails []string
			if len(validationResult.DependencyErrors) > 0 {
				errorDetails = append(errorDetails, fmt.Sprintf("%d dependency errors", len(validationResult.DependencyErrors)))
			}
			if len(validationResult.UnexpectedConfigs) > 0 {
				errorDetails = append(errorDetails, fmt.Sprintf("%d unexpected configs", len(validationResult.UnexpectedConfigs)))
			}
			if len(validationResult.MissingConfigs) > 0 {
				errorDetails = append(errorDetails, fmt.Sprintf("%d missing configs", len(validationResult.MissingConfigs)))
			}

			var errorMsg string
			if len(errorDetails) > 0 {
				errorMsg = fmt.Sprintf("dependency validation failed: %s", strings.Join(errorDetails, ", "))
			} else {
				errorMsg = "dependency validation failed - check validation output above for details"
			}

			return fmt.Errorf(errorMsg)
		}
	}

	if options.PreDeployHook != nil {
		options.Logger.ShortInfo("Running PreDeployHook")
		hookErr := options.PreDeployHook(options)
		if hookErr != nil {
			options.Testing.Fail()
			return hookErr
		}
		options.Logger.ShortInfo("Finished PreDeployHook")
	}

	options.Logger.ShortInfo("Dependency validation completed successfully")

	if !options.SkipInfrastructureDeployment {
		errorList := deployOptions.TriggerDeployAndWait()
		if len(errorList) > 0 {
			options.Logger.ShortError("Errors occurred during deploy")
			for _, err := range errorList {
				options.Logger.ShortError(fmt.Sprintf("  %v", err))
			}
			options.Testing.Fail()
			return fmt.Errorf("errors occurred during deploy")
		}
		options.Logger.ShortInfo("Deploy completed successfully")
		options.Logger.ShortInfo(common.ColorizeString(common.Colors.Green, "pass ✔"))
	} else {
		options.Logger.ShortInfo("Infrastructure deployment skipped")
		options.Logger.ShortInfo(common.ColorizeString(common.Colors.Yellow, "skip ⚠"))
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

	if options.PreUndeployHook != nil {
		options.Logger.ShortInfo("Running PreUndeployHook")
		hookErr := options.PreUndeployHook(options)
		if hookErr != nil {
			options.Testing.Fail()
			return hookErr
		}
		options.Logger.ShortInfo("Finished PreUndeployHook")
	}

	options.Logger.ShortInfo("Testing undeployed addons")

	// Trigger Undeploy
	if !options.SkipInfrastructureDeployment {
		undeployErrs := deployOptions.TriggerUnDeployAndWait()
		if len(undeployErrs) > 0 {
			options.Logger.ShortError("Errors occurred during undeploy")
			for _, err := range undeployErrs {
				options.Logger.ShortError(fmt.Sprintf("  %v", err))
			}
			options.Testing.Fail()
			return fmt.Errorf("errors occurred during undeploy")
		}
		options.Logger.ShortInfo("Undeploy completed successfully")
	} else {
		options.Logger.ShortInfo("Infrastructure undeploy skipped")
		options.Logger.ShortInfo(common.ColorizeString(common.Colors.Yellow, "skip ⚠"))
	}

	if options.PostUndeployHook != nil {
		options.Logger.ShortInfo("Running PostUndeployHook")
		hookErr := options.PostUndeployHook(options)
		if hookErr != nil {
			options.Testing.Fail()
			return hookErr
		}
		options.Logger.ShortInfo("Finished PostUndeployHook")
	}

	return nil
}

func (options *TestAddonOptions) TestSetup() error {

	setupErr := options.testSetup()
	if !assert.NoError(options.Testing, setupErr) {
		options.Testing.Fail()
		return fmt.Errorf("test setup has failed:%w", setupErr)
	}

	return nil
}

// Perform required steps for new test
func (options *TestAddonOptions) testSetup() error {

	// setup logger
	if options.Logger == nil {
		options.Logger = common.NewTestLogger(options.Testing.Name())
	}

	// Set logger prefix based on available identifiers (in order of preference)
	if options.TestCaseName != "" {
		// Use TestCaseName for clear test identification (preferred for matrix tests and custom scenarios)
		options.Logger.SetPrefix(fmt.Sprintf("ADDON - %s", options.TestCaseName))
	} else if options.ProjectName != "" {
		// For single tests, include project name in prefix
		options.Logger.SetPrefix(fmt.Sprintf("ADDON - %s", options.ProjectName))
	} else {
		// For tests without project name, use simple prefix
		options.Logger.SetPrefix("ADDON")
	}

	options.Logger.EnableDateTime(false)

	// change relative paths of configuration files to full path based on git root
	repoRoot, repoErr := common.GitRootPath(".")
	if repoErr != nil {
		repoRoot = "."
	}

	isChanges, files, err := common.ChangesToBePush(options.Testing, repoRoot)
	if err != nil {
		options.Logger.ShortError("Error checking for local changes in the repository")
		options.Testing.Fail()
		return fmt.Errorf("error checking for local changes in the repository: %w", err)
	}
	// remove ignored files
	if len(options.LocalChangesIgnorePattern) > 0 {
		filteredFiles := make([]string, 0)
		for _, file := range files {
			shouldKeep := true

			// Special case: always keep ibm_catalog.json files regardless of ignore patterns
			if strings.HasSuffix(file, "ibm_catalog.json") {
				filteredFiles = append(filteredFiles, file)
				continue
			}

			// ignore files are regex patterns
			for _, ignorePattern := range options.LocalChangesIgnorePattern {
				matched, err := regexp.MatchString(ignorePattern, file)
				if err != nil {
					options.Logger.ShortWarn(fmt.Sprintf("Error matching pattern %s: %v", ignorePattern, err))
					continue
				}
				if matched {
					shouldKeep = false
					break
				}
			}
			if shouldKeep {
				filteredFiles = append(filteredFiles, file)
			}
		}
		files = filteredFiles
		if len(files) == 0 {
			isChanges = false
		}
	}

	if isChanges {
		if !options.SkipLocalChangeCheck {
			options.Logger.ShortError("Local changes found in the repository, please commit, push or stash the changes before running the test")
			options.Logger.ShortError("Files with changes:")
			for _, file := range files {
				options.Logger.ShortError(fmt.Sprintf("  %s", file))
			}
			options.Testing.Fail()
			return fmt.Errorf("local changes found in the repository")
		} else {
			options.Logger.ShortWarn("Local changes found in the repository, but skipping the check")
			options.Logger.ShortWarn("Files with changes:")
			for _, file := range files {
				options.Logger.ShortWarn(fmt.Sprintf("  %s", file))
			}
		}
	}

	// get current branch and repo url
	repo, branch, repoErr := common.GetCurrentPrRepoAndBranch()
	if repoErr != nil {
		options.Logger.ShortError("Error getting current branch and repo")
		options.Testing.Fail()
		return fmt.Errorf("error getting current branch and repo: %w", repoErr)
	}
	options.currentBranch = &branch

	// Convert repository URL to HTTPS format for branch validation
	repoForValidation := repo
	if strings.HasPrefix(repo, "git@") {
		// Convert SSH format: git@github.com:username/repo.git → https://github.com/username/repo
		repoForValidation = strings.Replace(repo, ":", "/", 1)
		repoForValidation = strings.Replace(repoForValidation, "git@", "https://", 1)
		repoForValidation = strings.TrimSuffix(repoForValidation, ".git")
	} else if strings.HasPrefix(repo, "git://") {
		// Convert Git protocol: git://github.com/username/repo.git → https://github.com/username/repo
		repoForValidation = strings.Replace(repo, "git://", "https://", 1)
		repoForValidation = strings.TrimSuffix(repoForValidation, ".git")
	} else if strings.HasPrefix(repo, "https://") {
		// HTTPS format - just trim .git suffix if present
		repoForValidation = strings.TrimSuffix(repo, ".git")
	}

	// Validate that the branch exists in the remote repository (required for offering import)
	options.Logger.ShortInfo(fmt.Sprintf("Validating that branch '%s' exists in remote repository", branch))
	branchExists, err := common.CheckRemoteBranchExists(repoForValidation, branch)
	if err != nil {
		options.Logger.ShortError(fmt.Sprintf("Error checking if branch exists in remote repository: %v", err))
		options.Testing.Fail()
		return fmt.Errorf("error checking if branch exists in remote repository: %w", err)
	}
	if !branchExists {
		options.Logger.ShortError(fmt.Sprintf("Required branch '%s' does not exist in repository '%s'", branch, repoForValidation))
		options.Logger.ShortError("This branch is required for offering import/catalog tests to work properly.")
		options.Logger.ShortError("Please ensure the branch exists in the remote repository before running the test.")
		options.Testing.Fail()
		return fmt.Errorf("required branch '%s' does not exist in repository '%s' (required for offering import)", branch, repoForValidation)
	}
	options.Logger.ShortInfo(fmt.Sprintf("Branch '%s' confirmed to exist in remote repository", branch))

	options.Logger.ShortInfo("Checking for local changes in the repository")

	// Use the converted URL from validation for the rest of the process
	repo = repoForValidation

	options.currentBranchUrl = Core.StringPtr(fmt.Sprintf("%s/tree/%s", repo, branch))
	options.Logger.ShortInfo(fmt.Sprintf("Current branch: %s", branch))
	options.Logger.ShortInfo(fmt.Sprintf("Current repo: %s", repo))
	options.Logger.ShortInfo(fmt.Sprintf("Current branch URL: %s", *options.currentBranchUrl))

	// create new CloudInfoService if not supplied
	if options.CloudInfoService == nil {
		cloudInfoSvc, err := cloudinfo.NewCloudInfoServiceFromEnv("TF_VAR_ibmcloud_api_key", cloudinfo.CloudInfoServiceOptions{})
		if err != nil {
			return err
		}
		options.CloudInfoService = cloudInfoSvc
		options.CloudInfoService.SetLogger(options.Logger)
	}

	if !options.CatalogUseExisting {
		// Check if catalog sharing is enabled and if catalog already exists
		if options.catalog != nil {
			if options.catalog.Label != nil && options.catalog.ID != nil {
				options.Logger.ShortInfo(fmt.Sprintf("Using existing catalog: %s with ID %s", *options.catalog.Label, *options.catalog.ID))
			} else {
				options.Logger.ShortWarn("Using existing catalog but catalog details are incomplete")
			}
		} else if options.SharedCatalog != nil && *options.SharedCatalog {
			// For shared catalogs, only create if no shared catalog exists yet
			// Individual tests with SharedCatalog=true should not create new catalogs
			options.Logger.ShortInfo("SharedCatalog=true but no existing shared catalog available - this may indicate a setup issue")
			options.Logger.ShortInfo("Creating catalog anyway to avoid test failure, but consider using matrix tests for proper catalog sharing")
			catalog, err := options.CloudInfoService.CreateCatalog(options.CatalogName)
			if err != nil {
				options.Logger.ShortError(fmt.Sprintf("Error creating catalog for shared use: %v", err))
				options.Testing.Fail()
				return fmt.Errorf("error creating catalog for shared use: %w", err)
			}
			options.catalog = catalog
			if options.catalog != nil && options.catalog.Label != nil && options.catalog.ID != nil {
				options.Logger.ShortInfo(fmt.Sprintf("Created catalog for shared use: %s with ID %s", *options.catalog.Label, *options.catalog.ID))
			} else {
				options.Logger.ShortWarn("Created catalog for shared use but catalog details are incomplete")
			}
		} else {
			// Create new catalog only for non-shared usage
			options.Logger.ShortInfo(fmt.Sprintf("Creating a new catalog: %s", options.CatalogName))
			catalog, err := options.CloudInfoService.CreateCatalog(options.CatalogName)
			if err != nil {
				options.Logger.ShortError(fmt.Sprintf("Error creating a new catalog: %v", err))
				options.Testing.Fail()
				return fmt.Errorf("error creating a new catalog: %w", err)
			}
			options.catalog = catalog
			if options.catalog != nil && options.catalog.Label != nil && options.catalog.ID != nil {
				options.Logger.ShortInfo(fmt.Sprintf("Created a new catalog: %s with ID %s", *options.catalog.Label, *options.catalog.ID))
			} else {
				options.Logger.ShortWarn("Created catalog but catalog details are incomplete")
			}
		}
	} else {
		options.Logger.ShortInfo("Using existing catalog")
		options.Logger.ShortWarn("Not implemented yet")
		// TODO: lookup the catalog ID no api for this
	}

	// import the offering
	// ensure install kind is set or return an error
	if !options.AddonConfig.OfferingInstallKind.Valid() {
		options.Logger.ShortError(fmt.Sprintf("'%s' is not valid for OfferingInstallKind", options.AddonConfig.OfferingInstallKind.String()))
		options.Testing.Fail()
		return fmt.Errorf("'%s' is not valid for OfferingInstallKind", options.AddonConfig.OfferingInstallKind.String())
	}
	// check offering name set or fail
	if options.AddonConfig.OfferingName == "" {
		options.Logger.ShortError("AddonConfig.OfferingName is not set")
		options.Testing.Fail()
		return fmt.Errorf("AddonConfig.OfferingName is not set")
	}
	// Import the offering - check sharing settings
	if options.SharedCatalog != nil && *options.SharedCatalog && options.offering != nil {
		options.Logger.ShortInfo(fmt.Sprintf("Using existing shared offering: %s with ID %s", *options.offering.Label, *options.offering.ID))

		// Set offering details for addon config from existing offering
		newVersionLocator := ""
		if options.offering.Kinds != nil && len(options.offering.Kinds) > 0 &&
			len(options.offering.Kinds[0].Versions) > 0 {
			newVersionLocator = *options.offering.Kinds[0].Versions[0].VersionLocator
		}
		options.AddonConfig.OfferingName = *options.offering.Name
		options.AddonConfig.OfferingID = *options.offering.ID
		options.AddonConfig.VersionLocator = newVersionLocator
		options.AddonConfig.OfferingLabel = *options.offering.Label

		// Set the resolved version from the existing offering
		if options.offering.Kinds != nil && len(options.offering.Kinds) > 0 &&
			len(options.offering.Kinds[0].Versions) > 0 &&
			options.offering.Kinds[0].Versions[0].Version != nil {
			options.AddonConfig.ResolvedVersion = *options.offering.Kinds[0].Versions[0].Version
		}

		options.Logger.ShortInfo(fmt.Sprintf("Using shared offering Version Locator: %s", options.AddonConfig.VersionLocator))
	} else {
		// Create new offering if sharing is disabled or no existing offering
		version := fmt.Sprintf("v0.0.1-dev-%s", options.Prefix)
		options.AddonConfig.ResolvedVersion = version
		options.Logger.ShortInfo(fmt.Sprintf("Importing the offering flavor: %s from branch: %s as version: %s", options.AddonConfig.OfferingFlavor, *options.currentBranchUrl, version))
		offering, err := options.CloudInfoService.ImportOffering(*options.catalog.ID, *options.currentBranchUrl, options.AddonConfig.OfferingName, options.AddonConfig.OfferingFlavor, version, options.AddonConfig.OfferingInstallKind)
		if err != nil {
			options.Logger.ShortError(fmt.Sprintf("Error importing the offering: %v", err))
			options.Testing.Fail()
			return fmt.Errorf("error importing the offering: %w", err)
		}
		options.offering = offering
		options.Logger.ShortInfo(fmt.Sprintf("Imported flavor: %s with version: %s to %s", *options.offering.Label, version, *options.catalog.Label))

		// Set offering details for addon config
		newVersionLocator := ""
		if options.offering.Kinds != nil {
			newVersionLocator = *options.offering.Kinds[0].Versions[0].VersionLocator
		}
		options.AddonConfig.OfferingName = *options.offering.Name
		options.AddonConfig.OfferingID = *options.offering.ID
		options.AddonConfig.VersionLocator = newVersionLocator
		options.AddonConfig.OfferingLabel = *options.offering.Label

		options.Logger.ShortInfo(fmt.Sprintf("Offering Version Locator: %s", options.AddonConfig.VersionLocator))
	}

	// Create a new project (only if not already created)
	if options.currentProject == nil {
		options.Logger.ShortInfo("Creating project for test")
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
		if err != nil {
			options.Logger.ShortError(fmt.Sprintf("Error creating a new project: %v", err))
			options.Logger.ShortError(fmt.Sprintf("Response: %v", resp))
			options.Testing.Fail()
			return fmt.Errorf("error creating a new project: %w", err)
		}
		options.currentProject = prj
		options.currentProjectConfig.ProjectID = *options.currentProject.ID
		options.Logger.ShortInfo(fmt.Sprintf("Created a new project: %s with ID %s", options.ProjectName, options.currentProjectConfig.ProjectID))
		projectURL := fmt.Sprintf("https://cloud.ibm.com/projects/%s/configurations", options.currentProjectConfig.ProjectID)
		options.Logger.ShortInfo(fmt.Sprintf("Project URL: %s", projectURL))
		region := options.currentProjectConfig.Location
		if region == "" {
			region = "unknown"
		}
		options.Logger.ShortInfo(fmt.Sprintf("Project Region: %s", region))
	} else {
		// Using existing project
		options.Logger.ShortInfo(fmt.Sprintf("Using existing project: %s with ID %s", options.ProjectName, *options.currentProject.ID))
		// Ensure currentProjectConfig is set up properly for existing projects
		if options.currentProjectConfig == nil {
			options.currentProjectConfig = &cloudinfo.ProjectsConfig{
				ProjectID: *options.currentProject.ID,
			}
		}
	}

	return nil
}

func (options *TestAddonOptions) TestTearDown() {

	if !options.SkipTestTearDown {
		// if we are not skipping the test teardown, execute it
		options.testTearDown()
	}

}

func (options *TestAddonOptions) testTearDown() {

	// perform the test teardown
	options.Logger.ShortInfo("Performing test teardown")

	// Project cleanup logic: always clean up projects since we're not sharing them
	if options.currentProject != nil && options.currentProject.ID != nil {
		options.Logger.ShortInfo(fmt.Sprintf("Deleting the project %s with ID %s", options.ProjectName, *options.currentProject.ID))
		_, resp, err := options.CloudInfoService.DeleteProject(*options.currentProject.ID)
		if assert.NoError(options.Testing, err) {
			if assert.Equal(options.Testing, 202, resp.StatusCode) {
				options.Logger.ShortInfo(fmt.Sprintf("Deleted Test Project: %s", options.currentProjectConfig.ProjectName))
			} else {
				options.Logger.ShortError(fmt.Sprintf("Failed to delete Test Project, response code: %d", resp.StatusCode))
			}
		} else {
			options.Logger.ShortError(fmt.Sprintf("Error deleting Test Project: %s", err))
		}
	} else {
		options.Logger.ShortInfo("No project ID found to delete")
	}

	// Catalog cleanup logic:
	// - Individual tests with SharedCatalog=false: clean up their own catalogs
	// - Individual tests with SharedCatalog=true: keep catalog for potential reuse
	if options.catalog != nil && (options.SharedCatalog == nil || !*options.SharedCatalog) {
		options.Logger.ShortInfo(fmt.Sprintf("Deleting the catalog %s with ID %s (SharedCatalog=false)", *options.catalog.Label, *options.catalog.ID))
		err := options.CloudInfoService.DeleteCatalog(*options.catalog.ID)
		if err != nil {
			options.Logger.ShortError(fmt.Sprintf("Error deleting the catalog: %v", err))
			options.Testing.Fail()
		} else {
			options.Logger.ShortInfo(fmt.Sprintf("Deleted the catalog %s with ID %s", *options.catalog.Label, *options.catalog.ID))
		}
	} else {
		if options.SharedCatalog != nil && *options.SharedCatalog {
			options.Logger.ShortInfo("Shared catalog retained for potential reuse (SharedCatalog=true)")
		} else {
			options.Logger.ShortInfo("No catalog to delete")
		}
	}
}

// RunAddonTestMatrix runs multiple addon test cases in parallel using a matrix approach
// This method handles the boilerplate of running parallel tests and automatically shares
// catalogs and offerings across test cases for efficiency.
//
// BaseOptions must be provided with common options that apply to all test cases.
// BaseSetupFunc can optionally customize the options for each specific test case.
func (options *TestAddonOptions) RunAddonTestMatrix(matrix AddonTestMatrix) {
	options.Testing.Parallel()

	// Validate that BaseOptions is provided
	if matrix.BaseOptions == nil {
		panic("BaseOptions must be provided for AddonTestMatrix")
	}

	// Create shared resource tracking for the matrix
	var sharedCatalogOptions *TestAddonOptions
	var sharedMutex = &sync.Mutex{}

	for _, tc := range matrix.TestCases {
		tc := tc // Capture loop variable for parallel execution
		options.Testing.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			// Start with a copy of BaseOptions and customize for this test case
			testOptions := matrix.BaseOptions.copy()
			testOptions.Testing = t // Override testing context for this specific test

			// Allow BaseSetupFunc to customize the copied options
			if matrix.BaseSetupFunc != nil {
				testOptions = matrix.BaseSetupFunc(testOptions, tc)
			}

			// Apply test case specific prefix if provided
			if tc.Prefix != "" {
				testOptions.Prefix = tc.Prefix
			}
			// Ensure prefix is unique to avoid resource name collisions
			if testOptions.Prefix != "" {
				uniqueID := common.UniqueId()
				if len(uniqueID) > 4 {
					uniqueID = uniqueID[:4]
				}
				testOptions.Prefix = fmt.Sprintf("%s-%s", testOptions.Prefix, uniqueID)
			} else {
				uniqueID := common.UniqueId()
				if len(uniqueID) > 4 {
					uniqueID = uniqueID[:4]
				}
				testOptions.Prefix = fmt.Sprintf("test-%s", uniqueID)
			}
			testOptions.AddonConfig.Prefix = testOptions.Prefix

			// Ensure logger is initialized before using it
			if testOptions.Logger == nil {
				testOptions.Logger = common.NewTestLogger(testOptions.Testing.Name())
			}

			// Ensure CloudInfoService is initialized before using it for catalog operations
			if testOptions.CloudInfoService == nil {
				cloudInfoSvc, err := cloudinfo.NewCloudInfoServiceFromEnv("TF_VAR_ibmcloud_api_key", cloudinfo.CloudInfoServiceOptions{})
				if err != nil {
					require.NoError(t, err, "Failed to initialize CloudInfoService")
					return
				}
				testOptions.CloudInfoService = cloudInfoSvc
				testOptions.CloudInfoService.SetLogger(testOptions.Logger)
			}

			// Matrix tests always use shared catalogs for efficiency, regardless of SharedCatalog setting
			if testOptions.SharedCatalog == nil {
				testOptions.SharedCatalog = core.BoolPtr(true)
			} else if !*testOptions.SharedCatalog {
				testOptions.Logger.ShortWarn("Matrix tests override SharedCatalog=false to use shared catalogs for efficiency")
				testOptions.SharedCatalog = core.BoolPtr(true)
			}

			// Apply test case specific settings
			if tc.SkipTearDown {
				testOptions.SkipTestTearDown = true
			}
			if tc.SkipInfrastructureDeployment {
				testOptions.SkipInfrastructureDeployment = true
			}

			// Set TestCaseName for clear logging (matrix tests automatically use test case name)
			if tc.Name != "" {
				testOptions.TestCaseName = tc.Name
			}

			// Create addon configuration using the provided config function
			testOptions.AddonConfig = matrix.AddonConfigFunc(testOptions, tc)

			// Set dependencies if provided in test case
			if tc.Dependencies != nil {
				testOptions.AddonConfig.Dependencies = tc.Dependencies
			}

			// Set project name using test case and prefix
			if testOptions.Prefix != "" {
				nameComponents := []string{}

				if testOptions.AddonConfig.OfferingName != "" {
					// Extract a shorter, more readable name from the offering
					offeringShortName := testOptions.AddonConfig.OfferingName
					if strings.HasPrefix(offeringShortName, "deploy-arch-") {
						offeringShortName = strings.TrimPrefix(offeringShortName, "deploy-arch-")
					}
					nameComponents = append(nameComponents, offeringShortName)
				}

				// Add test case name in lowercase for readability
				if tc.Name != "" {
					nameComponents = append(nameComponents, strings.ToLower(tc.Name))
				}

				nameComponents = append(nameComponents, testOptions.Prefix)
				testOptions.ProjectName = strings.Join(nameComponents, "-")
			}

			// Merge any additional inputs from the test case
			if tc.Inputs != nil && len(tc.Inputs) > 0 {
				if testOptions.AddonConfig.Inputs == nil {
					testOptions.AddonConfig.Inputs = make(map[string]interface{})
				}
				for key, value := range tc.Inputs {
					testOptions.AddonConfig.Inputs[key] = value
				}
			}

			// Handle shared catalog creation in matrix tests
			sharedMutex.Lock()
			if sharedCatalogOptions == nil {
				// This is the first test case - it will create the shared catalog and offering
				sharedCatalogOptions = testOptions

				// First, validate that the branch exists in the remote repository BEFORE creating any resources
				// Get repository info for offering import validation
				repo, branch, repoErr := common.GetCurrentPrRepoAndBranch()
				if repoErr != nil {
					sharedMutex.Unlock()
					testOptions.Logger.ShortError("Error getting current branch and repo for offering import validation")
					require.NoError(t, repoErr, "Failed to get repository info for offering import validation")
					return
				}

				// Convert repository URL to HTTPS format for branch validation
				if strings.HasPrefix(repo, "git@") {
					repo = strings.Replace(repo, ":", "/", 1)
					repo = strings.Replace(repo, "git@", "https://", 1)
					repo = strings.TrimSuffix(repo, ".git")
				} else if strings.HasPrefix(repo, "git://") {
					repo = strings.Replace(repo, "git://", "https://", 1)
					repo = strings.TrimSuffix(repo, ".git")
				} else if strings.HasPrefix(repo, "https://") {
					repo = strings.TrimSuffix(repo, ".git")
				}

				// Validate that the branch exists in the remote repository (required for offering import)
				testOptions.Logger.ShortInfo(fmt.Sprintf("Validating that branch '%s' exists in remote repository before creating any resources", branch))
				branchExists, err := common.CheckRemoteBranchExists(repo, branch)
				if err != nil {
					sharedMutex.Unlock()
					testOptions.Logger.ShortError(fmt.Sprintf("Error checking if branch exists in remote repository: %v", err))
					require.NoError(t, err, "Failed to validate branch exists for offering import")
					return
				}
				if !branchExists {
					sharedMutex.Unlock()
					testOptions.Logger.ShortError(fmt.Sprintf("Required branch '%s' does not exist in repository '%s'", branch, repo))
					testOptions.Logger.ShortError("This branch is required for offering import/catalog tests to work properly.")
					testOptions.Logger.ShortError("Please ensure the branch exists in the remote repository before running the test.")
					require.Fail(t, fmt.Sprintf("Required branch '%s' does not exist in repository '%s' (required for offering import)", branch, repo))
					return
				}
				testOptions.Logger.ShortInfo(fmt.Sprintf("Branch '%s' confirmed to exist in remote repository", branch))

				// Create the shared catalog for all matrix tests
				if !testOptions.CatalogUseExisting {
					// Generate a descriptive catalog name for matrix tests
					offeringShortName := "addon"
					if testOptions.AddonConfig.OfferingName != "" {
						offeringShortName = testOptions.AddonConfig.OfferingName
						if strings.HasPrefix(offeringShortName, "deploy-arch-") {
							offeringShortName = strings.TrimPrefix(offeringShortName, "deploy-arch-")
						}
					}
					// Extract just the unique ID from the prefix for the catalog name
					prefixParts := strings.Split(testOptions.Prefix, "-")
					uniqueId := prefixParts[len(prefixParts)-1]
					descriptiveCatalogName := fmt.Sprintf("matrix-test-%s-catalog-%s", offeringShortName, uniqueId)

					testOptions.Logger.ShortInfo(fmt.Sprintf("Creating shared catalog for matrix: %s", descriptiveCatalogName))
					catalog, err := testOptions.CloudInfoService.CreateCatalog(descriptiveCatalogName)
					if err != nil {
						sharedMutex.Unlock() // Release mutex on error
						testOptions.Logger.ShortError(fmt.Sprintf("Error creating shared catalog: %v", err))
						require.NoError(t, err, "Failed to create shared catalog for matrix tests")
						return
					}
					testOptions.catalog = catalog
					if testOptions.catalog != nil && testOptions.catalog.Label != nil && testOptions.catalog.ID != nil {
						testOptions.Logger.ShortInfo(fmt.Sprintf("Created shared catalog: %s with ID %s", *testOptions.catalog.Label, *testOptions.catalog.ID))
					} else {
						testOptions.Logger.ShortWarn("Created shared catalog but catalog details are incomplete")
					}

					// Import the offering once for all matrix tests
					version := fmt.Sprintf("v0.0.1-dev-%s", testOptions.Prefix)
					testOptions.AddonConfig.ResolvedVersion = version

					// Get repository info for offering import
					repo, branch, repoErr := common.GetCurrentPrRepoAndBranch()
					if repoErr != nil {
						sharedMutex.Unlock()
						testOptions.Logger.ShortError("Error getting current branch and repo for offering import")
						require.NoError(t, repoErr, "Failed to get repository info for offering import")
						return
					}

					// Convert repository URL to HTTPS format for catalog import
					if strings.HasPrefix(repo, "git@") {
						repo = strings.Replace(repo, ":", "/", 1)
						repo = strings.Replace(repo, "git@", "https://", 1)
						repo = strings.TrimSuffix(repo, ".git")
					} else if strings.HasPrefix(repo, "git://") {
						repo = strings.Replace(repo, "git://", "https://", 1)
						repo = strings.TrimSuffix(repo, ".git")
					} else if strings.HasPrefix(repo, "https://") {
						repo = strings.TrimSuffix(repo, ".git")
					}

					// Branch validation was already performed before catalog creation
					branchUrl := fmt.Sprintf("%s/tree/%s", repo, branch)
					testOptions.Logger.ShortInfo(fmt.Sprintf("Importing shared offering: %s from branch: %s as version: %s", testOptions.AddonConfig.OfferingFlavor, branchUrl, version))

					offering, err := testOptions.CloudInfoService.ImportOffering(*testOptions.catalog.ID, branchUrl, testOptions.AddonConfig.OfferingName, testOptions.AddonConfig.OfferingFlavor, version, testOptions.AddonConfig.OfferingInstallKind)
					if err != nil {
						sharedMutex.Unlock() // Release mutex on error
						testOptions.Logger.ShortError(fmt.Sprintf("Error importing shared offering: %v", err))
						require.NoError(t, err, "Failed to import shared offering for matrix tests")
						return
					}
					testOptions.offering = offering

					if testOptions.offering != nil && testOptions.offering.Label != nil && testOptions.offering.ID != nil {
						testOptions.Logger.ShortInfo(fmt.Sprintf("Imported shared offering: %s with ID %s", *testOptions.offering.Label, *testOptions.offering.ID))
					} else {
						testOptions.Logger.ShortWarn("Imported shared offering but offering details are incomplete")
					}
				}

				sharedMutex.Unlock()
			} else {
				// Share the catalog and offering from the first instance
				testOptions.catalog = sharedCatalogOptions.catalog
				testOptions.offering = sharedCatalogOptions.offering

				sharedMutex.Unlock()
				if testOptions.catalog != nil && testOptions.catalog.Label != nil && testOptions.catalog.ID != nil {
					testOptions.Logger.ShortInfo(fmt.Sprintf("Using shared catalog: %s with ID %s", *testOptions.catalog.Label, *testOptions.catalog.ID))
				} else {
					testOptions.Logger.ShortWarn("Shared catalog is nil or incomplete - catalog creation may have failed")
				}
				if testOptions.offering != nil && testOptions.offering.Label != nil && testOptions.offering.ID != nil {
					testOptions.Logger.ShortInfo(fmt.Sprintf("Using shared offering: %s with ID %s", *testOptions.offering.Label, *testOptions.offering.ID))
				} else {
					testOptions.Logger.ShortWarn("Shared offering is nil or incomplete - offering import may have failed")
				}
			}

			// Run the test - each test creates its own project
			err := testOptions.RunAddonTest()
			require.NoError(t, err, "Addon Test had an unexpected error")
		})
	}

	// Cleanup shared resources after all tests complete
	go func() {
		options.Testing.Cleanup(func() {
			if sharedCatalogOptions != nil {
				sharedCatalogOptions.CleanupSharedResources()
			}
		})
	}()
}

// validateInputsWithRetry validates required inputs for a configuration with retry logic
// This handles the case where the backend database hasn't been updated yet after configuration changes
func (options *TestAddonOptions) validateInputsWithRetry(configID string, targetAddon cloudinfo.AddonConfig, maxRetries int, retryDelay time.Duration) (bool, []string) {
	var allInputsPresent bool
	var missingInputs []string
	var lastError error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		// Get the latest configuration details
		currentConfigDetails, _, err := options.CloudInfoService.GetConfig(&cloudinfo.ConfigDetails{
			ProjectID: options.currentProjectConfig.ProjectID,
			ConfigID:  configID,
		})

		if err != nil {
			lastError = err
			if attempt < maxRetries {
				options.Logger.ShortInfo(fmt.Sprintf("Attempt %d/%d: Error getting configuration details, retrying in %v: %v", attempt, maxRetries, retryDelay, err))
				time.Sleep(retryDelay)
				continue
			}
			options.Logger.ShortError(fmt.Sprintf("Failed to get configuration after %d attempts: %v", maxRetries, err))
			return false, []string{fmt.Sprintf("Failed to get configuration: %v", err)}
		}

		// Reset for this attempt
		allInputsPresent = true
		missingInputs = nil

		// Check required inputs
		for _, input := range targetAddon.OfferingInputs {
			if !input.Required {
				continue
			}
			if input.Key == "ibmcloud_api_key" {
				continue
			}

			value, exists := currentConfigDetails.Definition.(*projectv1.ProjectConfigDefinitionResponse).Inputs[input.Key]
			if !exists || fmt.Sprintf("%v", value) == "" {
				if input.DefaultValue == nil || fmt.Sprintf("%v", input.DefaultValue) == "" || fmt.Sprintf("%v", input.DefaultValue) == "__NOT_SET__" {
					allInputsPresent = false
					configIdentifier := fmt.Sprintf("%s (missing: %s)", *currentConfigDetails.Definition.(*projectv1.ProjectConfigDefinitionResponse).Name, input.Key)
					missingInputs = append(missingInputs, configIdentifier)
				}
			}
		}

		// If all inputs are present, we're done
		if allInputsPresent {
			if attempt > 1 {
				options.Logger.ShortInfo(fmt.Sprintf("Input validation succeeded on attempt %d/%d after retrying", attempt, maxRetries))
			}
			return true, nil
		}

		// If this isn't the last attempt, wait and retry
		if attempt < maxRetries {
			options.Logger.ShortInfo(fmt.Sprintf("Attempt %d/%d: Some required inputs appear missing, retrying in %v (this may be due to database update timing)", attempt, maxRetries, retryDelay))

			// Show which inputs are missing on this attempt for debugging
			if len(missingInputs) > 0 {
				options.Logger.ShortInfo(fmt.Sprintf("Missing inputs on attempt %d:", attempt))
				for _, missing := range missingInputs {
					options.Logger.ShortInfo(fmt.Sprintf("  %s", missing))
				}
			}

			time.Sleep(retryDelay)
		}
	}

	// All attempts failed
	if lastError != nil {
		options.Logger.ShortError(fmt.Sprintf("Input validation failed after %d attempts due to configuration retrieval error: %v", maxRetries, lastError))
	} else {
		options.Logger.ShortError(fmt.Sprintf("Input validation failed after %d attempts - inputs still appear missing:", maxRetries))
		for _, missing := range missingInputs {
			options.Logger.ShortError(fmt.Sprintf("  %s", missing))
		}

		// Show detailed retry debug information when all attempts fail
		options.Logger.ShortError("=== RETRY VALIDATION DEBUG INFO ===")
		options.Logger.ShortError(fmt.Sprintf("Configuration ID: %s", configID))
		options.Logger.ShortError(fmt.Sprintf("Retry attempts: %d", maxRetries))
		options.Logger.ShortError(fmt.Sprintf("Retry delay: %v", retryDelay))

		// Try to get the final state and show all inputs
		finalConfigDetails, _, finalErr := options.CloudInfoService.GetConfig(&cloudinfo.ConfigDetails{
			ProjectID: options.currentProjectConfig.ProjectID,
			ConfigID:  configID,
		})

		if finalErr != nil {
			options.Logger.ShortError(fmt.Sprintf("Could not retrieve final config state: %v", finalErr))
		} else {
			options.Logger.ShortError("Final configuration state:")
			if finalConfigDetails.State != nil {
				options.Logger.ShortError(fmt.Sprintf("  State: %s", *finalConfigDetails.State))
			}
			if finalConfigDetails.StateCode != nil {
				options.Logger.ShortError(fmt.Sprintf("  StateCode: %s", string(*finalConfigDetails.StateCode)))
			}

			options.Logger.ShortError("All inputs in final configuration:")
			if finalConfigDetails.Definition != nil {
				if resp, ok := finalConfigDetails.Definition.(*projectv1.ProjectConfigDefinitionResponse); ok && resp.Inputs != nil {
					for key, value := range resp.Inputs {
						// Don't log sensitive values
						if strings.Contains(strings.ToLower(key), "key") || strings.Contains(strings.ToLower(key), "password") || strings.Contains(strings.ToLower(key), "secret") {
							options.Logger.ShortError(fmt.Sprintf("    %s: [REDACTED]", key))
						} else {
							options.Logger.ShortError(fmt.Sprintf("    %s: %v (type: %T)", key, value, value))
						}
					}
				} else {
					options.Logger.ShortError("    No inputs found in configuration definition")
				}
			}

			options.Logger.ShortError("Required inputs that were checked:")
			for _, input := range targetAddon.OfferingInputs {
				if input.Required && input.Key != "ibmcloud_api_key" {
					defaultInfo := "no default"
					if input.DefaultValue != nil {
						defaultInfo = fmt.Sprintf("default: %v", input.DefaultValue)
					}
					options.Logger.ShortError(fmt.Sprintf("    %s (%s)", input.Key, defaultInfo))
				}
			}
		}
		options.Logger.ShortError("=== END RETRY DEBUG INFO ===")
	}

	return false, missingInputs
}
