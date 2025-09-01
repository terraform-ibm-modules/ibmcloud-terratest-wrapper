package testaddons

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/platform-services-go-sdk/catalogmanagementv1"
	"github.com/IBM/project-go-sdk/projectv1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/cloudinfo"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/common"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testprojects"
)

// ConfigDependencyInfo holds information about a config's dependencies for circular dependency analysis
type ConfigDependencyInfo struct {
	ID                   string            // Config ID
	Name                 string            // Config Name
	InputReferences      []string          // List of input references (ref:/configs/{id}/outputs/{name})
	InputFieldReferences map[string]string // Map of input field names to their reference strings
}

// DetailedDependencyInfo contains enhanced dependency information for better error reporting
type DetailedDependencyInfo struct {
	ConfigID             string
	ConfigName           string
	InputName            string // The input field name that creates the dependency
	ReferencedConfigID   string // The config ID being referenced
	ReferencedConfigName string // The config name being referenced
	ReferencedOutput     string // The output/input field name being referenced
	ReferencedType       string // The type of reference: "outputs", "inputs", etc.
	FullReference        string // The complete reference string
}

// runAddonTest contains the core test execution logic with configurable error reporting
// This private method is used by both RunAddonTest() and matrix tests
// enhancedReporting: if true, shows detailed actionable advice; if false, shows simple error messages
func (options *TestAddonOptions) runAddonTest(enhancedReporting bool) error {
	// Log test execution start with clear markers to ensure every test is tracked
	testName := "Unknown"
	if options.Testing != nil {
		testName = options.Testing.Name()
	}
	// Force log test start regardless of quiet mode to ensure every test is tracked
	if options.Logger != nil {
		if smartLogger, ok := options.Logger.(*common.SmartLogger); ok {
			// SmartLogger - use info for non-error messages
			smartLogger.ImmediateShortInfo(fmt.Sprintf("TEST EXECUTION START: %s", testName))
			smartLogger.ImmediateShortInfo(fmt.Sprintf("Test Configuration: Prefix='%s', OfferingName='%s', QuietMode=%v", options.Prefix, options.AddonConfig.OfferingName, options.QuietMode))
		} else if bufferedLogger, ok := options.Logger.(*common.BufferedTestLogger); ok {
			// BufferedTestLogger - use info for non-error messages
			bufferedLogger.ImmediateShortInfo(fmt.Sprintf("TEST EXECUTION START: %s", testName))
			bufferedLogger.ImmediateShortInfo(fmt.Sprintf("Test Configuration: Prefix='%s', OfferingName='%s', QuietMode=%v", options.Prefix, options.AddonConfig.OfferingName, options.QuietMode))
		} else {
		}
	}

	// Always log test completion, even on early exit or panic
	var testResult string = "UNKNOWN"
	var testError error = nil

	// Helper function to set test result before returning
	setFailureResult := func(err error, stage string) error {
		testResult = fmt.Sprintf("FAILED_AT_%s", stage)
		testError = err
		return err
	}

	defer func() {
		if r := recover(); r != nil {
			testResult = "PANICKED"
			testError = fmt.Errorf("panic: %v", r)
		}

		// If we reached here without setting a result, assume success
		if testResult == "UNKNOWN" {
			testResult = "PASSED"
		}

		// Force log test completion regardless of quiet mode or buffering
		completionMsg := ""
		if testError != nil {
			completionMsg = fmt.Sprintf("TEST EXECUTION END: %s - RESULT: %s (error: %v)", testName, testResult, testError)
		} else {
			completionMsg = fmt.Sprintf("TEST EXECUTION END: %s - RESULT: %s", testName, testResult)
		}

		// Always force output of test completion, bypassing quiet mode
		if options.Logger != nil {
			if smartLogger, ok := options.Logger.(*common.SmartLogger); ok {
				// Use appropriate log level: error for failures, info for success
				if testError != nil {
					smartLogger.ImmediateShortError(completionMsg)
				} else {
					smartLogger.ImmediateShortInfo(completionMsg)
				}
			} else if bufferedLogger, ok := options.Logger.(*common.BufferedTestLogger); ok {
				if testError != nil {
					bufferedLogger.ImmediateShortError(completionMsg)
				} else {
					// Use immediate info for success to ensure it's visible but not red
					bufferedLogger.ImmediateShortInfo(completionMsg)
				}
			} else {
				if testError != nil {
					options.Logger.ShortError(completionMsg)
				} else {
					options.Logger.ShortInfo(completionMsg)
				}
			}
		}

		// Ensure logs are flushed for failed tests
		if testResult != "PASSED" {
			// Debug logger type and state before flushing
			loggerType := "unknown"
			bufferSize := "unknown"
			if options.Logger != nil {
				loggerType = fmt.Sprintf("%T", options.Logger)
				if _, ok := options.Logger.(*common.BufferedTestLogger); ok {
					// Try to get buffer info if available
					bufferSize = "BufferedTestLogger"
				} else if _, ok := options.Logger.(*common.SmartLogger); ok {
					bufferSize = "SmartLogger"
				}
			}

			// Force immediate output of debug info - use immediate output to bypass buffering
			if bufferedLogger, ok := options.Logger.(*common.BufferedTestLogger); ok {
				bufferedLogger.ImmediateShortError(fmt.Sprintf("Buffer flush for failed test. Logger type: %s, Buffer: %s, QuietMode: %v", loggerType, bufferSize, options.QuietMode))
			}

			if options.Logger != nil {
				options.Logger.MarkFailed()
				options.Logger.FlushOnFailure()

				// Verify flush occurred - use immediate output
				if bufferedLogger, ok := options.Logger.(*common.BufferedTestLogger); ok {
					bufferedLogger.ImmediateShortError("Buffer flush completed for failed test")
				}
			} else {
				// Create temporary immediate logger when logger is nil
				if options.Testing != nil {
					tempLogger := common.CreateSmartAutoBufferingLogger(testName, false)
					if bufferedLogger, ok := tempLogger.(*common.BufferedTestLogger); ok {
						bufferedLogger.ImmediateShortError("Cannot flush - logger is nil")
					}
				}
			}
		}
	}()

	if !options.SkipTestTearDown {
		// ensure we always run the test tear down, even if a panic occurs
		defer func() {
			if r := recover(); r != nil {
				// Get the file and line number where the panic occurred
				_, file, line, ok := runtime.Caller(4)

				// Safely handle logger - use Testing.Log if Logger is nil
				panicMsg := ""
				if ok {
					panicMsg = fmt.Sprintf("Recovered from panic: %v\nOccurred at: %s:%d\n", r, file, line)
				} else {
					panicMsg = fmt.Sprintf("Recovered from panic: %v", r)
				}

				if options.Logger != nil {
					options.Logger.ShortError(panicMsg)
					// Mark as failed and flush debug logs on panic
					options.Logger.MarkFailed()
					options.Logger.FlushOnFailure()
				} else {
					// Create temporary immediate logger when logger is not available
					tempLogger := common.CreateSmartAutoBufferingLogger(testName, false)
					if bufferedLogger, ok := tempLogger.(*common.BufferedTestLogger); ok {
						bufferedLogger.ImmediateShortError(fmt.Sprintf("ERROR: %s", panicMsg))
					}
				}

				options.Testing.Fail()
			}
			options.TestTearDown()
		}()
	}

	// Show setup progress in quiet mode
	if options.QuietMode {
		options.Logger.ProgressStage("Setting up test Catalog and Project")
	}

	setupErr := options.testSetup()
	if !assert.NoError(options.Testing, setupErr) {
		options.Logger.MarkFailed()
		options.Logger.FlushOnFailure()
		options.Testing.Fail()
		return setFailureResult(fmt.Errorf("test setup has failed:%w", setupErr), "SETUP")
	}

	// Apply required dependency business logic before deployment
	// This ensures required dependencies are force-enabled before actual deployment
	options.Logger.ShortInfo("VALIDATION STEP: Starting required dependency validation")

	// Debug: Force immediate output to verify validation steps are being logged
	if bufferedLogger, ok := options.Logger.(*common.BufferedTestLogger); ok {
		bufferedLogger.ImmediateShortError(fmt.Sprintf("Validation step logged (QuietMode: %v, Logger: %T)", options.QuietMode, options.Logger))
	}
	if options.QuietMode {
		options.Logger.ProgressStage("Validating required dependencies")
	}
	err := options.validateAndProcessRequiredDependencies()
	if err != nil {
		if bufferedLogger, ok := options.Logger.(*common.BufferedTestLogger); ok {
			bufferedLogger.ImmediateShortError(fmt.Sprintf("REQUIRED_DEPENDENCY_FAILURE: About to mark failed and flush buffer (bufferSize: %d)", bufferedLogger.GetBufferSize()))
		}
		options.Logger.MarkFailed()
		options.Logger.FlushOnFailure()
		if bufferedLogger, ok := options.Logger.(*common.BufferedTestLogger); ok {
			bufferedLogger.ImmediateShortError("REQUIRED_DEPENDENCY_FAILURE: Buffer flush completed")
		}
		return setFailureResult(fmt.Errorf("required dependency validation failed: %w", err), "REQUIRED_DEPENDENCY_VALIDATION")
	}

	// Deploy Addon to Project
	options.Logger.ShortInfo("VALIDATION STEP: Required dependency validation completed successfully")
	options.Logger.ShortInfo("DEPLOYMENT STEP: Starting addon deployment to project")
	if options.QuietMode {
		options.Logger.ProgressStage("Deploying Configurations to Project")
	}
	options.Logger.ShortInfo("Deploying the addon to project")

	deployedConfigs, err := options.CloudInfoService.DeployAddonToProject(&options.AddonConfig, options.currentProjectConfig)

	if err != nil {
		options.Logger.ShortError(fmt.Sprintf("Error deploying the addon to project: %v", err))

		// When deployment fails, attempt to build and log the expected dependency tree for debugging
		// This helps identify what should have been deployed when analyzing failures
		if options.AddonConfig.CatalogID != "" && options.AddonConfig.OfferingID != "" && options.AddonConfig.VersionLocator != "" {
			options.Logger.ShortInfo("Building expected dependency tree for debugging deployment failure...")
			visited := make(map[string]bool)
			graphResult, graphErr := options.buildDependencyGraph(
				options.AddonConfig.CatalogID,
				options.AddonConfig.OfferingID,
				options.AddonConfig.VersionLocator,
				options.AddonConfig.OfferingFlavor,
				&options.AddonConfig,
				visited,
			)
			if graphErr == nil {
				options.Logger.ShortInfo("Expected dependency tree (for debugging):")
				options.PrintDependencyTree(graphResult.Graph, graphResult.ExpectedDeployedList)
			} else {
				options.Logger.ShortError(fmt.Sprintf("Could not build dependency tree for debugging: %v", graphErr))
			}
		}

		options.Logger.MarkFailed()
		options.Logger.FlushOnFailure()
		options.Testing.Fail()
		return setFailureResult(fmt.Errorf("error deploying the addon to project: %w", err), "DEPLOYMENT")
	}

	// Store deployed configs for later use in dependency validation
	options.deployedConfigs = deployedConfigs

	options.Logger.ShortInfo("DEPLOYMENT STEP: Deployment completed successfully")
	options.Logger.ShortInfo(fmt.Sprintf("Deployed Configurations to Project ID: %s", options.currentProjectConfig.ProjectID))
	for _, config := range deployedConfigs.Configs {
		options.Logger.ShortInfo(fmt.Sprintf("  %s - ID: %s", config.Name, config.ConfigID))
	}
	options.Logger.ShortInfo("Addon deployed successfully")

	// Show configuration update progress in quiet mode
	if options.QuietMode {
		options.Logger.ProgressStage("Updating configuration inputs")
	}
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

		prjCfg, _, err := options.CloudInfoService.GetConfig(&cloudinfo.ConfigDetails{
			ProjectID: options.currentProjectConfig.ProjectID,
			Name:      config.Name,
			ConfigID:  config.ConfigID,
		})
		if err != nil {
			options.Logger.ShortError(fmt.Sprintf("Error retrieving config %s: %v", config.Name, err))
			options.Logger.MarkFailed()
			options.Logger.FlushOnFailure()
			options.Testing.Fail()
			return setFailureResult(fmt.Errorf("error retrieving config %s: %w", config.Name, err), "CONFIG_RETRIEVAL")
		}
		if prjCfg == nil {
			options.Logger.ShortError(fmt.Sprintf("Retrieved config %s is nil", config.Name))
			options.Logger.MarkFailed()
			options.Logger.FlushOnFailure()
			options.Testing.Fail()
			return setFailureResult(fmt.Errorf("retrieved config %s is nil", config.Name), "CONFIG_NIL")
		}
		configDetails.Members = append(configDetails.Members, *prjCfg)

		configDetails.MemberConfigs = append(configDetails.MemberConfigs, projectv1.StackConfigMember{
			ConfigID: core.StringPtr(config.ConfigID),
			Name:     core.StringPtr(config.Name),
		})

		// Collect input references for OverrideInputMappings logic (reuse existing GetConfig call)
		if options.configInputReferences == nil {
			options.configInputReferences = make(map[string]map[string]string)
		}
		if resp, ok := prjCfg.Definition.(*projectv1.ProjectConfigDefinitionResponse); ok && resp.Inputs != nil {
			references := make(map[string]string)
			for inputKey, inputValue := range resp.Inputs {
				if strValue, ok := inputValue.(string); ok && strings.HasPrefix(strValue, "ref:") {
					references[inputKey] = strValue
				}
			}
			if len(references) > 0 {
				options.configInputReferences[config.ConfigID] = references
			}
		}
	}

	// Process AddonConfig.Inputs with OverrideInputMappings logic for regular (non-matrix) tests
	// This ensures input override logging and reference preservation works for both matrix and regular tests
	if options.AddonConfig.Inputs != nil && len(options.AddonConfig.Inputs) > 0 {
		// Apply the same reference-aware input processing logic as matrix tests
		if options.OverrideInputMappings != nil && !*options.OverrideInputMappings {
			// Reference preservation mode (default)
			configReferences := options.configInputReferences[addonID]
			preservedCount := 0
			overriddenCount := 0

			for inputKey, inputValue := range options.AddonConfig.Inputs {
				if referenceValue, isReference := configReferences[inputKey]; isReference {
					// Preserve reference value
					if !options.QuietMode {
						options.Logger.ShortInfo(fmt.Sprintf("  Input '%s': preserving reference value '%s' (ignoring test override)", inputKey, referenceValue))
					}
					configDetails.Inputs[inputKey] = referenceValue
					preservedCount++
				} else {
					// Override with new value
					if !options.QuietMode {
						existingValue := configDetails.Inputs[inputKey]
						options.Logger.ShortInfo(fmt.Sprintf("  Input '%s': %v → %v", inputKey, existingValue, inputValue))
					}
					configDetails.Inputs[inputKey] = inputValue
					overriddenCount++
				}
			}

			// Summary logging
			if !options.QuietMode && (preservedCount > 0 || overriddenCount > 0) {
				options.Logger.ShortInfo(fmt.Sprintf("Input merging complete: %d reference(s) preserved, %d input(s) overridden (OverrideInputMappings=false)", preservedCount, overriddenCount))
			}
		} else {
			// Override all mode (legacy behavior)
			overriddenCount := 0
			if !options.QuietMode {
				options.Logger.ShortInfo("Overriding ALL inputs (OverrideInputMappings=true)")
			}

			for inputKey, inputValue := range options.AddonConfig.Inputs {
				if !options.QuietMode {
					existingValue := configDetails.Inputs[inputKey]
					options.Logger.ShortInfo(fmt.Sprintf("  Input '%s': %v → %v", inputKey, existingValue, inputValue))
				}
				configDetails.Inputs[inputKey] = inputValue
				overriddenCount++
			}

			if !options.QuietMode && overriddenCount > 0 {
				options.Logger.ShortInfo(fmt.Sprintf("Input override complete: %d input(s) overridden (OverrideInputMappings=true)", overriddenCount))
			}
		}
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
		options.Logger.MarkFailed()
		options.Logger.FlushOnFailure()
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
		Logger:               options.Logger.GetUnderlyingLogger(),
		Testing:              options.Testing,
		DeployTimeoutMinutes: options.DeployTimeoutMinutes,
		StackPollTimeSeconds: 60,
	}

	deployOptions.SetCurrentStackConfig(&configDetails)
	deployOptions.SetCurrentProjectConfig(options.currentProjectConfig)

	allConfigs, err := options.CloudInfoService.GetProjectConfigs(options.currentProjectConfig.ProjectID)
	if err != nil {
		options.Logger.ShortError(fmt.Sprintf("Error getting the configuration: %v", err))
		options.Logger.MarkFailed()
		options.Logger.FlushOnFailure()
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
	totalReferencesProcessed := 0

	// Collect configs in awaiting_prerequisite state for circular dependency analysis
	awaitingPrerequisiteConfigs := make([]ConfigDependencyInfo, 0)

	// These variables will store the collected validation issues
	// They are declared here but will only be evaluated after dependency validation

	// set offering details
	SetOfferingDetails(options)

	// Create a map of deployed config IDs for this test case to avoid processing configs from other test cases
	deployedConfigIDs := make(map[string]bool)
	if options.deployedConfigs != nil {
		for _, deployedConfig := range options.deployedConfigs.Configs {
			deployedConfigIDs[deployedConfig.ConfigID] = true
		}
	}

	// Show configuration processing progress in quiet mode
	if options.QuietMode {
		options.Logger.ProgressStage("Processing configuration details")
	}

	// Enable batch mode for smart logger if processing multiple configs
	if smartLogger, ok := options.Logger.(*common.SmartLogger); ok && len(allConfigs) > 1 {
		smartLogger.EnableBatchMode()
		defer smartLogger.DisableBatchMode()
	}

	for _, config := range allConfigs {
		options.Logger.ShortInfo(fmt.Sprintf("  %s - ID: %s", *config.Definition.Name, *config.ID))

		currentConfigDetails, _, err := options.CloudInfoService.GetConfig(&cloudinfo.ConfigDetails{
			ProjectID: options.currentProjectConfig.ProjectID,
			ConfigID:  *config.ID,
		})

		if err != nil {
			options.Logger.ShortError(fmt.Sprintf("Error getting the configuration: %v", err))
			options.Logger.MarkFailed()
			options.Logger.FlushOnFailure()
			options.Testing.Fail()
			return fmt.Errorf("error getting the configuration: %w", err)
		}

		// Initialize reference cache if needed
		if options.configInputReferences == nil {
			options.configInputReferences = make(map[string]map[string]string)
		}

		// Collect input references for ALL configs (extends existing pattern from AwaitingPrerequisite logic)
		if currentConfigDetails.Definition != nil {
			if resp, ok := currentConfigDetails.Definition.(*projectv1.ProjectConfigDefinitionResponse); ok && resp.Inputs != nil {
				fieldReferences := make(map[string]string)
				for fieldName, input := range resp.Inputs {
					if inputStr, ok := input.(string); ok && strings.HasPrefix(inputStr, "ref:/") {
						fieldReferences[fieldName] = inputStr
					}
				}
				// Only store if there are references to avoid empty maps
				if len(fieldReferences) > 0 {
					options.configInputReferences[*config.ID] = fieldReferences
				}
			}
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

		// Collect configs in AwaitingPrerequisite state for circular dependency analysis
		if currentConfigDetails.StateCode != nil && *currentConfigDetails.StateCode == projectv1.ProjectConfig_StateCode_AwaitingPrerequisite {
			configName := *currentConfigDetails.Definition.(*projectv1.ProjectConfigDefinitionResponse).Name
			configInfo := ConfigDependencyInfo{
				ID:                   *config.ID,
				Name:                 configName,
				InputReferences:      make([]string, 0),
				InputFieldReferences: make(map[string]string),
			}

			// Collect input references for this config
			for fieldName, input := range currentConfigDetails.Definition.(*projectv1.ProjectConfigDefinitionResponse).Inputs {
				if inputStr, ok := input.(string); ok && strings.HasPrefix(inputStr, "ref:/") {
					configInfo.InputReferences = append(configInfo.InputReferences, inputStr)
					configInfo.InputFieldReferences[fieldName] = inputStr
				}
			}

			awaitingPrerequisiteConfigs = append(awaitingPrerequisiteConfigs, configInfo)
			options.Logger.ShortWarn(fmt.Sprintf("Configuration '%s' is in AwaitingPrerequisite state", configName))
		}

		// Skip reference validation if the flag is set
		if !options.SkipRefValidation {
			// Show reference validation progress in quiet mode (only once for all configs)
			if config == allConfigs[0] && options.QuietMode {
				options.Logger.ProgressStage("Validating configuration references")
			}
			options.Logger.ShortInfo("  References:")
			references := []string{}

			for _, input := range currentConfigDetails.Definition.(*projectv1.ProjectConfigDefinitionResponse).Inputs {
				// Check if input is a string before checking for ref:/ prefix
				if inputStr, ok := input.(string); ok && strings.HasPrefix(inputStr, "ref:/") {
					options.Logger.ShortInfo(fmt.Sprintf("    %s", inputStr))
					references = append(references, inputStr)
					totalReferencesProcessed++
				}
			}

			if len(references) > 0 {
				// Use batch mode to reduce logging verbosity when processing multiple configs
				batchMode := len(allConfigs) > 1
				res_resp, err := options.CloudInfoService.ResolveReferencesFromStringsWithContext(*options.currentProject.Location, references, options.currentProjectConfig.ProjectID, batchMode)
				if err != nil {
					// Check if this is a known intermittent error that should be skipped
					// This can occur as either a direct HttpError or as an EnhancedHttpError with additional context
					errStr := err.Error()

					// Use structured API error classification instead of fragile string matching
					isSkippableError := IsSkippableAPIError(errStr)

					// Only skip validation for intermittent errors if infrastructure deployment is enabled
					// When SkipInfrastructureDeployment=true, reference validation is the only chance to catch issues
					if isSkippableError && !options.SkipInfrastructureDeployment {
						options.Logger.ShortWarn(fmt.Sprintf("Skipping reference validation due to intermittent IBM Cloud service error: %v", err))

						// Use structured error type for specific messaging
						errorType, _ := ClassifyAPIError(errStr)
						switch errorType {
						case APIKeyError:
							options.Logger.ShortWarn("This is a known transient issue with IBM Cloud's API key validation service.")
						case ProjectNotFoundError:
							options.Logger.ShortWarn("This is a timing issue where project details are checked too quickly after creation.")
							options.Logger.ShortWarn("The resolver API needs time to be updated with new project information.")
						case IntermittentError:
							options.Logger.ShortWarn("This is a known transient issue with IBM Cloud's reference resolution service.")
						}

						options.Logger.ShortWarn("The test will continue and will fail later if references actually fail to resolve during deployment.")
						// Skip reference validation for this config and continue with the test
						continue
					} else if isSkippableError && options.SkipInfrastructureDeployment {
						options.Logger.ShortWarn(fmt.Sprintf("Detected intermittent service error, but cannot skip validation in validation-only mode: %v", err))
						options.Logger.ShortWarn("Infrastructure deployment is disabled, so reference validation is the only opportunity to catch reference issues.")
						options.Logger.ShortWarn("Failing the test to ensure reference issues are not missed.")
					}
					// For other errors, fail the test as before
					options.Logger.ShortError(fmt.Sprintf("Error resolving references: %v", err))
					options.Logger.MarkFailed()
					options.Logger.FlushOnFailure()
					options.Testing.Fail()
					return fmt.Errorf("error resolving references: %w", err)
				}
				options.Logger.ShortInfo("  Resolved References:")
				for _, ref := range res_resp.References {
					if ref.Code != 200 {
						// Check if this is a valid reference that cannot be resolved until after member deployment
						// This is a valid scenario and should be treated as a warning, not an error
						if IsMemberDeploymentReference(ref.Message) {
							options.Logger.ShortWarn(fmt.Sprintf("%s   %s - Warning: %s", common.ColorizeString(common.Colors.Yellow, "⚠"), ref.Reference, ref.State))
							options.Logger.ShortWarn(fmt.Sprintf("      Message: %s", ref.Message))
							options.Logger.ShortWarn(fmt.Sprintf("      Code: %d", ref.Code))
							options.Logger.ShortWarn("      This is a valid reference that cannot be resolved until the member configuration is deployed.")
							// This is a warning, not an error, so don't add to failedRefs
						} else {
							options.Logger.ShortWarn(fmt.Sprintf("%s   %s - Error: %s", common.ColorizeString(common.Colors.Red, "✘"), ref.Reference, ref.State))
							options.Logger.ShortWarn(fmt.Sprintf("      Message: %s", ref.Message))
							options.Logger.ShortWarn(fmt.Sprintf("      Code: %d", ref.Code))
							// Store failed ref for later evaluation instead of failing immediately
							failedRefs = append(failedRefs, ref.Reference)
						}
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

			// Use structured configuration matching instead of fragile string patterns
			mainAddonMatcher := NewConfigurationMatcherForAddon(options.AddonConfig)
			if matched, rule := mainAddonMatcher.IsMatch(configName); matched {
				targetAddon = options.AddonConfig
				addonFound = true
				options.Logger.ShortInfo(fmt.Sprintf("Matched addon using %s for config: %s (rule: %s)",
					rule.Strategy.String(), configName, rule.Description))
			} else {
				// Try to match dependencies using structured matching
				for i, dependency := range options.AddonConfig.Dependencies {
					dependencyMatcher := NewConfigurationMatcherForAddon(dependency)
					if matched, rule := dependencyMatcher.IsMatch(configName); matched {
						targetAddon = options.AddonConfig.Dependencies[i]
						addonFound = true
						options.Logger.ShortInfo(fmt.Sprintf("Matched dependency using %s for config: %s (rule: %s)",
							rule.Strategy.String(), configName, rule.Description))
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
			options.Logger.ShortWarn("Skipping input validation for this configuration")
			continue
		}

		// Validate required inputs with retry mechanism to handle database timing issues
		// Show input validation progress in quiet mode (only once for all configs)
		if config == allConfigs[0] && options.QuietMode {
			options.Logger.ProgressStage("Validating required inputs")
		}
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
			options.Logger.ShortWarn(fmt.Sprintf("Some required inputs are missing for addon: %s (will check dependencies first)", *currentConfigDetails.ID))
		} else {
			options.Logger.ShortInfo(fmt.Sprintf("All required inputs set for addon: %s", *currentConfigDetails.ID))
		}
	}

	// Show reference validation completion in quiet mode if references were processed
	if !options.SkipRefValidation && options.QuietMode && len(allConfigs) > 1 && totalReferencesProcessed > 0 {
		options.Logger.ProgressSuccess(fmt.Sprintf("Reference validation completed (%d configurations, %d references)", len(allConfigs), totalReferencesProcessed))
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

	// Collect missing input issues for later evaluation after dependency validation
	var inputValidationIssues []string
	if len(missingRequiredInputs) > 0 {
		options.Logger.ShortWarn("Missing required inputs detected (will check dependencies first):")
		for _, configError := range missingRequiredInputs {
			options.Logger.ShortWarn(fmt.Sprintf("  %s", configError))
			inputValidationIssues = append(inputValidationIssues, configError)
		}
	}

	// Collect waiting input issues for later evaluation after dependency validation
	var waitingInputIssues []string
	if len(waitingOnInputs) == 0 {
		options.Logger.ShortInfo("No configurations waiting on inputs")
	} else {
		options.Logger.ShortWarn("Found configurations waiting on inputs (will check dependencies first):")
		for _, config := range waitingOnInputs {
			options.Logger.ShortWarn(fmt.Sprintf("  %s", config))
			waitingInputIssues = append(waitingInputIssues, config)
		}
	}

	// Check if any configs are ready to validate, but don't fail immediately
	if readyToValidate {
		options.Logger.ShortInfo("Found a configuration ready to validate")
	} else {
		// Analyze configs in awaiting_prerequisite state for circular dependencies
		circularDeps := options.detectCircularDependencies(awaitingPrerequisiteConfigs)
		if len(circularDeps) > 0 {
			// Handle circular dependencies based on StrictMode
			if options.StrictMode == nil || *options.StrictMode {
				// Strict mode (default): Log as errors and fail the test
				options.Logger.ShortError("Circular dependency detected - configs are waiting on each other:")
				for _, cycle := range circularDeps {
					options.Logger.ShortError(fmt.Sprintf("  %s", cycle))
					waitingInputIssues = append(waitingInputIssues, fmt.Sprintf("Circular dependency: %s", cycle))
				}
			} else {
				// Non-strict mode: Log as warnings and add to ValidationResult.Warnings
				options.Logger.ShortWarn("Circular dependency detected (StrictMode=false - test will continue):")
				for _, cycle := range circularDeps {
					options.Logger.ShortWarn(fmt.Sprintf("  %s", cycle))
					// Add to ValidationResult warnings instead of waitingInputIssues
					if options.lastValidationResult == nil {
						options.lastValidationResult = &ValidationResult{
							IsValid:  true, // Still valid in non-strict mode
							Warnings: []string{},
						}
					}
					options.lastValidationResult.Warnings = append(options.lastValidationResult.Warnings, fmt.Sprintf("Circular dependency: %s", cycle))
				}
				options.Logger.ShortInfo("Note: Circular dependencies may cause deployment issues but test will proceed")
			}
		} else if len(awaitingPrerequisiteConfigs) > 0 {
			// Check for unresolved references
			unresolvedRefs := options.findUnresolvedReferences(awaitingPrerequisiteConfigs, allConfigs)
			if len(unresolvedRefs) > 0 {
				options.Logger.ShortError("Found unresolved input references:")
				for _, ref := range unresolvedRefs {
					options.Logger.ShortError(fmt.Sprintf("  %s", ref))
					waitingInputIssues = append(waitingInputIssues, fmt.Sprintf("Unresolved reference: %s", ref))
				}
			} else {
				options.Logger.ShortWarn(fmt.Sprintf("Found %d configurations in awaiting_prerequisite state (will check dependencies first)", len(awaitingPrerequisiteConfigs)))
				waitingInputIssues = append(waitingInputIssues, fmt.Sprintf("%d configurations in awaiting_prerequisite state", len(awaitingPrerequisiteConfigs)))
			}
		} else {
			options.Logger.ShortWarn("No configuration found in ready_to_validate state (will check dependencies first)")
			waitingInputIssues = append(waitingInputIssues, "No configuration is in ready_to_validate state")
		}
	}

	// Check if the configuration is in a valid state
	options.Logger.ShortInfo(fmt.Sprintf("Checked if the configuration is deployable %s", common.ColorizeString(common.Colors.Green, "pass ✔")))

	// Now run dependency validation before evaluating the collected validation issues
	if !options.SkipDependencyValidation {
		options.Logger.ShortInfo("VALIDATION STEP: Starting post-deployment dependency validation")
		if options.QuietMode {
			options.Logger.ProgressStage("Building dependency graph")
		}
		options.Logger.ShortInfo("Starting with dependency validation")
		var rootCatalogID, rootOfferingID, rootVersionLocator string
		rootVersionLocator = options.AddonConfig.VersionLocator
		rootCatalogID = options.AddonConfig.CatalogID
		rootOfferingID = options.AddonConfig.OfferingID

		// Add validation to catch the race condition/uninitialized catalog issue
		if rootCatalogID == "" {
			return setFailureResult(fmt.Errorf("dependency validation failed: AddonConfig.CatalogID is empty - this may indicate a race condition in parallel test execution or incomplete offering setup. VersionLocator='%s', OfferingName='%s'", rootVersionLocator, options.AddonConfig.OfferingName), "POST_DEPLOYMENT_VALIDATION_CATALOGID")
		}
		if rootOfferingID == "" {
			return setFailureResult(fmt.Errorf("dependency validation failed: AddonConfig.OfferingID is empty - this may indicate a race condition in parallel test execution or incomplete offering setup. VersionLocator='%s', OfferingName='%s', CatalogID='%s'", rootVersionLocator, options.AddonConfig.OfferingName, rootCatalogID), "POST_DEPLOYMENT_VALIDATION_OFFERINGID")
		}
		if rootVersionLocator == "" {
			return setFailureResult(fmt.Errorf("dependency validation failed: AddonConfig.VersionLocator is empty - this may indicate incomplete offering setup. OfferingName='%s', CatalogID='%s', OfferingID='%s'", options.AddonConfig.OfferingName, rootCatalogID, rootOfferingID), "POST_DEPLOYMENT_VALIDATION_VERSIONLOCATOR")
		}

		options.Logger.ShortInfo(fmt.Sprintf("Dependency validation starting with: catalogID='%s', offeringID='%s', versionLocator='%s', flavor='%s'", rootCatalogID, rootOfferingID, rootVersionLocator, options.AddonConfig.OfferingFlavor))

		// Build dependency graph using the cleaner return-values approach
		// Note: Required dependency validation has already been applied before deployment
		visited := make(map[string]bool)
		graphResult, err := options.buildDependencyGraph(rootCatalogID, rootOfferingID, rootVersionLocator, options.AddonConfig.OfferingFlavor, &options.AddonConfig, visited)
		if err != nil {
			// Use CriticalError to bypass quiet mode for this critical failure
			options.Logger.CriticalError(fmt.Sprintf("Failed to build dependency graph: %v - This may indicate issues with catalog access, offering metadata, or dependency resolution", err))
			return setFailureResult(err, "DEPENDENCY_GRAPH_BUILD")
		}

		// Extract results from the returned struct
		graph := graphResult.Graph
		expectedDeployedList := graphResult.ExpectedDeployedList

		// Always log the expected dependency tree early to ensure it's available for debugging
		// even if subsequent validation steps fail
		options.Logger.ShortInfo("Expected dependency tree:")
		options.PrintDependencyTree(graph, expectedDeployedList)

		if options.QuietMode {
			options.Logger.ProgressStage("Analyzing deployed configurations")
		}
		options.Logger.ShortInfo("Building the actually deployed configs")

		if options.deployedConfigs == nil {
			// Use CriticalError to bypass quiet mode for this critical failure
			options.Logger.CriticalError("Deployed configs not available for dependency validation - this indicates a serious issue with the deployment process or test setup")
			return setFailureResult(fmt.Errorf("deployed configs not available - cannot validate dependencies"), "MISSING_DEPLOYED_CONFIGS")
		}

		actuallyDeployedResult := options.buildActuallyDeployedListFromResponse(options.deployedConfigs)
		if len(actuallyDeployedResult.Errors) > 0 {
			options.Logger.ShortError("Failed to build deployed list from response:")
			for _, errMsg := range actuallyDeployedResult.Errors {
				options.Logger.ShortError(fmt.Sprintf("  - %s", errMsg))
			}
			return setFailureResult(fmt.Errorf("failed to build actually deployed list: %s", strings.Join(actuallyDeployedResult.Errors, "; ")), "BUILD_DEPLOYED_LIST")
		}

		if len(actuallyDeployedResult.Warnings) > 0 {
			options.Logger.ShortInfo("Built deployed list from deployment response with warnings:")
			for _, warning := range actuallyDeployedResult.Warnings {
				options.Logger.ShortWarn(fmt.Sprintf("Warning: %s", warning))
			}
		} else {
			options.Logger.ShortInfo("Built deployed list from deployment response")
		}

		if options.QuietMode {
			options.Logger.ProgressStage("Validating dependency compliance")
		}
		// First validate what is actually deployed to get the validation results
		validationResult := options.validateDependencies(graph, expectedDeployedList, actuallyDeployedResult.ActuallyDeployedList)

		// Preserve any existing warnings (like circular dependencies) before overwriting
		var existingWarnings []string
		if options.lastValidationResult != nil {
			existingWarnings = options.lastValidationResult.Warnings
		}

		// Store the validation result for error reporting
		options.lastValidationResult = &validationResult

		// Merge preserved warnings with new validation warnings
		if len(existingWarnings) > 0 {
			options.lastValidationResult.Warnings = append(existingWarnings, options.lastValidationResult.Warnings...)
		}

		options.Logger.ShortInfo("Actually deployed configurations (with status):")

		// Create deployment status maps for the tree view
		deployedMap := make(map[string]bool)
		for _, deployed := range actuallyDeployedResult.ActuallyDeployedList {
			key := generateAddonKeyFromDetail(deployed)
			deployedMap[key] = true
		}

		errorMap := make(map[string]cloudinfo.DependencyError)
		for _, depErr := range validationResult.DependencyErrors {
			key := generateAddonKeyFromDependencyError(depErr)
			errorMap[key] = depErr
		}

		missingMap := make(map[string]bool)
		for _, missing := range validationResult.MissingConfigs {
			key := generateAddonKeyFromDetail(missing)
			missingMap[key] = true
		}

		// Find the root addon and print tree with status
		allDependencies := make(map[string]bool)
		for _, deps := range graph {
			for _, dep := range deps {
				key := generateAddonKeyFromDetail(dep)
				allDependencies[key] = true
			}
		}

		var rootAddon *cloudinfo.OfferingReferenceDetail
		for _, addon := range expectedDeployedList {
			key := generateAddonKeyFromDetail(addon)
			if !allDependencies[key] {
				rootAddon = &addon
				break
			}
		}

		if rootAddon == nil && len(expectedDeployedList) > 0 {
			rootAddon = &expectedDeployedList[0]
		}

		// Build a comprehensive tree that shows ALL deployed configurations (expected + unexpected)
		// This helps identify where unexpected configs fit in the dependency hierarchy
		allDeployedTree := options.buildComprehensiveDeploymentTree(actuallyDeployedResult.ActuallyDeployedList, graph, validationResult)

		// Print the comprehensive tree that includes unexpected configurations
		if len(allDeployedTree) > 0 {
			// Find the root configuration (typically the main addon)
			var rootConfig *cloudinfo.OfferingReferenceDetail
			for _, config := range allDeployedTree {
				// Look for the configuration that doesn't appear as a dependency of others
				isRoot := true
				for _, otherConfig := range allDeployedTree {
					if deps, exists := graph[generateAddonKeyFromDetail(otherConfig)]; exists {
						for _, dep := range deps {
							if dep.Name == config.Name && dep.Version == config.Version && dep.Flavor.Name == config.Flavor.Name {
								isRoot = false
								break
							}
						}
					}
					if !isRoot {
						break
					}
				}
				if isRoot {
					rootConfig = &config
					break
				}
			}

			// If we couldn't find a clear root, use the first config
			if rootConfig == nil && len(allDeployedTree) > 0 {
				rootConfig = &allDeployedTree[0]
			}

			if rootConfig != nil {
				options.printComprehensiveTreeWithStatus(*rootConfig, allDeployedTree, graph, "", true, make(map[string]bool), validationResult)
			}
		} else if rootAddon != nil {
			// Fallback to original tree if no comprehensive tree available
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

		// Print warnings if any exist
		if len(validationResult.Warnings) > 0 {
			options.Logger.ShortWarn("Validation warnings:")
			for _, warning := range validationResult.Warnings {
				options.Logger.ShortWarn(fmt.Sprintf("  %s", warning))
			}
		}

		// Handle validation failures
		if !validationResult.IsValid {
			// Mark as failed and flush buffered logs BEFORE showing validation output
			// This ensures ShortInfo messages from dependency tree output are visible
			if bufferedLogger, ok := options.Logger.(*common.BufferedTestLogger); ok {
				bufferedLogger.ImmediateShortError(fmt.Sprintf("DEPENDENCY_VALIDATION_FAILURE: About to mark failed and flush buffer (bufferSize: %d)", bufferedLogger.GetBufferSize()))
			}
			options.Logger.MarkFailed()
			options.Logger.FlushOnFailure()
			if bufferedLogger, ok := options.Logger.(*common.BufferedTestLogger); ok {
				bufferedLogger.ImmediateShortError("DEPENDENCY_VALIDATION_FAILURE: Buffer flush completed")
			}

			// Print validation errors - either consolidated summary or detailed individual messages
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
				// Include specific names of missing configs in the error message
				var missingNames []string
				for _, missing := range validationResult.MissingConfigs {
					missingNames = append(missingNames, fmt.Sprintf("%s (%s, %s)", missing.Name, missing.Version, missing.Flavor.Name))
				}
				errorDetails = append(errorDetails, fmt.Sprintf("%d missing configs: [%s]", len(validationResult.MissingConfigs), strings.Join(missingNames, ", ")))
			}

			var errorMsg string
			if len(errorDetails) > 0 {
				errorMsg = fmt.Sprintf("dependency validation failed: %s", strings.Join(errorDetails, ", "))
			} else {
				errorMsg = "dependency validation failed - check validation output above for details"
			}

			return setFailureResult(fmt.Errorf("%s", errorMsg), "DEPENDENCY_VALIDATION")
		}
	}

	// Now evaluate input validation issues after dependency validation has provided context
	if len(inputValidationIssues) > 0 {
		options.Logger.ShortError("Input validation failed after dependency validation:")
		for _, issue := range inputValidationIssues {
			options.Logger.ShortError(fmt.Sprintf("  %s", issue))
		}

		// Store input validation issues in ValidationResult for proper categorization
		if options.lastValidationResult == nil {
			options.lastValidationResult = &ValidationResult{
				IsValid:             false,
				MissingInputs:       []string{},
				ConfigurationErrors: []string{},
				Messages:            []string{},
			}
		}

		// Add missing inputs to ValidationResult
		options.lastValidationResult.MissingInputs = append(options.lastValidationResult.MissingInputs, inputValidationIssues...)
		options.lastValidationResult.IsValid = false

		// Enhanced debugging information when validation fails
		options.Logger.ShortWarn("=== INPUT VALIDATION FAILURE DEBUG INFO ===")
		options.Logger.ShortWarn(fmt.Sprintf("FAILURE SUMMARY: %d configurations have missing required inputs - %s", len(inputValidationIssues), strings.Join(inputValidationIssues, "; ")))
		options.Logger.ShortWarn("Attempting to get current configuration details for debugging...")

		allConfigs, debugErr := options.CloudInfoService.GetProjectConfigs(options.currentProjectConfig.ProjectID)
		if debugErr != nil {
			options.Logger.ShortWarn(fmt.Sprintf("Could not retrieve configs for debugging: %v", debugErr))
		} else {
			options.Logger.ShortWarn(fmt.Sprintf("Found %d configurations in project:", len(allConfigs)))
			for _, config := range allConfigs {
				configDetails, _, getErr := options.CloudInfoService.GetConfig(&cloudinfo.ConfigDetails{
					ProjectID: options.currentProjectConfig.ProjectID,
					ConfigID:  *config.ID,
				})

				if getErr != nil {
					options.Logger.ShortWarn(fmt.Sprintf("  Config: %s (ID: %s) - ERROR: %v", *config.Definition.Name, *config.ID, getErr))
				} else {
					configName := *config.Definition.Name
					if configName == "" {
						configName = "(unnamed config)"
					}
					options.Logger.ShortWarn(fmt.Sprintf("  Config: %s (ID: %s)", configName, *config.ID))

					stateInfo := func() string {
						if configDetails.State != nil {
							return *configDetails.State
						}
						return "unknown"
					}()
					stateCodeInfo := func() string {
						if configDetails.StateCode != nil {
							return string(*configDetails.StateCode)
						}
						return "unknown"
					}()

					// Add state explanation
					stateExplanation := ""
					switch stateCodeInfo {
					case "awaiting_input":
						stateExplanation = " (waiting for required inputs to be provided)"
					case "awaiting_prerequisite":
						stateExplanation = " (waiting for dependent configurations to complete)"
					case "awaiting_validation":
						stateExplanation = " (ready to validate - inputs complete)"
					case "awaiting_member_deployment":
						stateExplanation = " (waiting for member configurations to deploy)"
					}

					options.Logger.ShortWarn(fmt.Sprintf("    State: %s", stateInfo))
					options.Logger.ShortWarn(fmt.Sprintf("    StateCode: %s%s", stateCodeInfo, stateExplanation))
					options.Logger.ShortWarn(fmt.Sprintf("    LocatorID: %s", func() string {
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
							options.Logger.ShortWarn("    Current Inputs:")
							for key, value := range resp.Inputs {
								// Use structured sensitive data detection instead of fragile string matching
								if IsSensitiveField(key) {
									options.Logger.ShortWarn(fmt.Sprintf("      %s: [REDACTED]", key))
								} else {
									options.Logger.ShortWarn(fmt.Sprintf("      %s: %v", key, value))
								}
							}
						}
					}
				}
			}
		}

		options.Logger.ShortWarn("Expected addon configuration details:")
		options.Logger.ShortWarn(fmt.Sprintf("  Main Addon Name: %s", options.AddonConfig.OfferingName))
		options.Logger.ShortWarn(fmt.Sprintf("  Main Addon Version: %s", options.AddonConfig.VersionID))
		configNameDisplay := options.AddonConfig.ConfigName
		if configNameDisplay == "" {
			configNameDisplay = "(blank - will be auto-generated)"
		}
		options.Logger.ShortWarn(fmt.Sprintf("  Main Addon Config Name: %s", configNameDisplay))
		options.Logger.ShortWarn(fmt.Sprintf("  Prefix: %s", options.AddonConfig.Prefix))
		if len(options.AddonConfig.Dependencies) > 0 {
			options.Logger.ShortWarn("  Dependencies:")
			for i, dep := range options.AddonConfig.Dependencies {
				depConfigName := dep.ConfigName
				if depConfigName == "" {
					depConfigName = "(blank - will be auto-generated)"
				}
				options.Logger.ShortWarn(fmt.Sprintf("    [%d] Name: %s, Version: %s, ConfigName: %s", i, dep.OfferingName, dep.VersionID, depConfigName))
			}
		}
		options.Logger.ShortCustom("=== END DEBUG INFO ===", common.Colors.Cyan)

		// Use enhanced or simple error reporting based on context
		errorMessage := fmt.Sprintf("Missing required inputs - %s", strings.Join(inputValidationIssues, "; "))
		if enhancedReporting {
			// Enhanced error reporting for direct test execution
			options.Logger.CriticalError(errorMessage)
			options.Logger.ShortCustom("Cannot proceed with deployment - required inputs must be provided", common.Colors.Red)
			options.Logger.ShortCustom("Note: Missing inputs may be caused by missing dependencies shown above", common.Colors.Red)
		} else {
			// Simple error reporting for matrix/nested test execution
			options.Logger.ShortError(errorMessage)
		}

		// Mark as failed and flush buffered logs to show complete diagnostic information
		options.Logger.MarkFailed()
		options.Logger.FlushOnFailure()
		options.Testing.Fail()
		return fmt.Errorf("missing required inputs: %s", strings.Join(inputValidationIssues, "; "))
	}

	// Now evaluate waiting input issues after dependency validation has provided context
	if len(waitingInputIssues) > 0 {
		options.Logger.ShortError("Found configurations waiting on inputs after dependency validation:")
		for _, config := range waitingInputIssues {
			options.Logger.ShortError(fmt.Sprintf("  %s", config))
		}

		// Add waiting input issues to the stored ValidationResult
		if options.lastValidationResult != nil {
			options.lastValidationResult.Messages = append(options.lastValidationResult.Messages, waitingInputIssues...)
			options.lastValidationResult.IsValid = false
		}

		// Print current configuration input values for debugging - similar to missing inputs debug info
		options.Logger.ShortError("=== WAITING INPUTS DEBUG INFO ===")
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
					for _, waitingConfig := range waitingInputIssues {
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
								// Use structured sensitive data detection instead of fragile string matching
								if IsSensitiveField(key) {
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
		options.Logger.ShortCustom("=== END DEBUG INFO ===", common.Colors.Cyan)

		// Create a specific, actionable error message with enhanced analysis
		var errorMsg string
		var actionableAdvice []string

		// Analyze configuration states for more specific guidance
		stateAnalysis := make(map[string][]string)
		referenceIssues := make([]string, 0)

		// Re-examine configurations to extract state-specific information
		if debugErr == nil {
			for _, config := range allConfigs {
				configDetails, _, getErr := options.CloudInfoService.GetConfig(&cloudinfo.ConfigDetails{
					ProjectID: options.currentProjectConfig.ProjectID,
					ConfigID:  *config.ID,
				})
				if getErr == nil && configDetails != nil {
					configName := *config.Definition.Name
					if configDetails.StateCode != nil {
						stateCode := *configDetails.StateCode
						stateAnalysis[stateCode] = append(stateAnalysis[stateCode], configName)

						// Check for reference issues in inputs
						if configDetails.Definition != nil {
							if resp, ok := configDetails.Definition.(*projectv1.ProjectConfigDefinitionResponse); ok && resp.Inputs != nil {
								for key, value := range resp.Inputs {
									if valueStr := fmt.Sprintf("%v", value); strings.HasPrefix(valueStr, "ref:/configs/") {
										referenceIssues = append(referenceIssues, fmt.Sprintf("%s.%s->%s", configName, key, valueStr))
									}
								}
							}
						}
					}
				}
			}
		}

		// Check if we have circular dependency details from the waiting input issues
		var circularDependencyDetails []string
		for _, issue := range waitingInputIssues {
			if strings.HasPrefix(issue, "Circular dependency: ") {
				circularDependencyDetails = append(circularDependencyDetails, strings.TrimPrefix(issue, "Circular dependency: "))
			}
		}

		// Build specific error message based on state analysis
		if len(circularDependencyDetails) > 0 {
			// If we have circular dependency details, use them in the final error message
			errorMsg = fmt.Sprintf("circular dependency deadlock detected - %s", strings.Join(circularDependencyDetails, "; "))
			actionableAdvice = append(actionableAdvice, "• Use existing resources instead of creating new ones")
			actionableAdvice = append(actionableAdvice, "• Restructure deployment order by splitting dependencies")
			actionableAdvice = append(actionableAdvice, "• Consider using data sources or external references")
		} else if len(stateAnalysis) > 0 {
			var stateDetails []string
			for state, configs := range stateAnalysis {
				switch state {
				case "awaiting_prerequisite":
					stateDetails = append(stateDetails, fmt.Sprintf("%d config(s) waiting for prerequisites: %s", len(configs), strings.Join(configs, ", ")))
					actionableAdvice = append(actionableAdvice, "• Check if required dependency configurations are enabled and properly configured")
				case "awaiting_member_deployment":
					stateDetails = append(stateDetails, fmt.Sprintf("%d config(s) waiting for member deployment: %s", len(configs), strings.Join(configs, ", ")))
					actionableAdvice = append(actionableAdvice, "• Verify stack member configurations are properly defined and not in error state")
				case "awaiting_input":
					stateDetails = append(stateDetails, fmt.Sprintf("%d config(s) waiting for inputs: %s", len(configs), strings.Join(configs, ", ")))
					actionableAdvice = append(actionableAdvice, "• Provide missing input values or add input mappings for disabled dependencies")
				case "awaiting_validation":
					stateDetails = append(stateDetails, fmt.Sprintf("%d config(s) ready for validation: %s", len(configs), strings.Join(configs, ", ")))
				default:
					stateDetails = append(stateDetails, fmt.Sprintf("%d config(s) in %s state: %s", len(configs), state, strings.Join(configs, ", ")))
				}
			}
			errorMsg = fmt.Sprintf("configuration state deadlock detected - %s", strings.Join(stateDetails, "; "))
		} else if len(missingInputsDetails) > 0 {
			errorMsg = fmt.Sprintf("configurations waiting on missing inputs: %s", strings.Join(missingInputsDetails, ", "))
			actionableAdvice = append(actionableAdvice, "• Provide the missing input values in your test configuration")
		} else if len(configsWithIssues) > 0 {
			errorMsg = fmt.Sprintf("configurations in problematic state: %s", strings.Join(configsWithIssues, ", "))
		} else {
			errorMsg = "configurations waiting on inputs - check debug output above for details"
		}

		// Add reference-specific advice if detected
		if len(referenceIssues) > 0 && len(referenceIssues) <= 3 {
			actionableAdvice = append(actionableAdvice, fmt.Sprintf("• Key reference dependencies: %s", strings.Join(referenceIssues, ", ")))
		}

		// Use enhanced or simple error reporting based on context
		if enhancedReporting {
			// Enhanced error reporting for direct test execution
			options.Logger.CriticalError(fmt.Sprintf("Found %s", errorMsg))
			if len(actionableAdvice) > 0 {
				options.Logger.ShortCustom("RECOMMENDED ACTIONS:", common.Colors.Yellow)
				for _, advice := range actionableAdvice {
					options.Logger.ShortCustom(advice, common.Colors.Red)
				}
			} else {
				options.Logger.ShortCustom("Note: Missing inputs may be caused by missing dependencies shown above", common.Colors.Red)
			}
		} else {
			// Simple error reporting for matrix/nested test execution
			options.Logger.ShortError(fmt.Sprintf("Found %s", errorMsg))
		}

		// Mark as failed and flush buffered logs to show complete diagnostic information
		options.Logger.MarkFailed()
		options.Logger.FlushOnFailure()
		options.Testing.Fail()
		return setFailureResult(fmt.Errorf("found %s", errorMsg), "INPUT_VALIDATION")
	}

	options.Logger.ShortInfo("VALIDATION STEP: Post-deployment dependency validation completed successfully")

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
		if options.QuietMode {
			options.Logger.ProgressStage("Deploying infrastructure")
		}
		errorList := deployOptions.TriggerDeployAndWait()
		if len(errorList) > 0 {
			options.Logger.ShortError("Errors occurred during deploy")
			for _, err := range errorList {
				options.Logger.ShortError(fmt.Sprintf("  %v", err))
			}
			options.Logger.MarkFailed()
			options.Logger.FlushOnFailure()
			options.Testing.Fail()
			return fmt.Errorf("errors occurred during deploy")
		}
		if options.QuietMode {
			options.Logger.ProgressSuccess("Infrastructure deployment completed")
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
			options.Logger.MarkFailed()
			options.Logger.FlushOnFailure()
			options.Testing.Fail()
			return hookErr
		}
		options.Logger.ShortInfo("Finished PostDeployHook")
	}

	if options.PreUndeployHook != nil {
		options.Logger.ShortInfo("Running PreUndeployHook")
		hookErr := options.PreUndeployHook(options)
		if hookErr != nil {
			options.Logger.MarkFailed()
			options.Logger.FlushOnFailure()
			options.Testing.Fail()
			return hookErr
		}
		options.Logger.ShortInfo("Finished PreUndeployHook")
	}

	options.Logger.ShortInfo("Testing undeployed addons")

	// Trigger Undeploy
	if !options.SkipInfrastructureDeployment {
		if options.QuietMode {
			options.Logger.ProgressStage("Cleaning up infrastructure")
		}
		undeployErrs := deployOptions.TriggerUnDeployAndWait()
		if len(undeployErrs) > 0 {
			options.Logger.ShortError("Errors occurred during undeploy")
			for _, err := range undeployErrs {
				options.Logger.ShortError(fmt.Sprintf("  %v", err))
			}
			options.Logger.MarkFailed()
			options.Logger.FlushOnFailure()
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
			options.Logger.MarkFailed()
			options.Logger.FlushOnFailure()
			options.Testing.Fail()
			return hookErr
		}
		options.Logger.ShortInfo("Finished PostUndeployHook")
	}

	// Enhanced reporting: show success message for direct test execution
	if enhancedReporting {
		options.Logger.ProgressSuccess("✅ All tests passed - no actions required")

		// Display strict mode warnings if running in permissive mode and warnings exist
		if options.StrictMode != nil && !*options.StrictMode && options.lastValidationResult != nil && len(options.lastValidationResult.Warnings) > 0 {
			options.displaySingleTestStrictModeWarnings()
		}
	}

	return nil
}

// RunAddonTest : Run the test for addons with enhanced error reporting
// Creates a new catalog
// Imports an offering
// Creates a new project
// Adds a configuration
// Deploys the configuration
// Deletes the project
// Deletes the catalog
// Returns an error if any of the steps fail
// This public method provides enhanced error reporting for direct test execution
func (options *TestAddonOptions) RunAddonTest() error {
	// Use enhanced reporting for direct test execution
	err := options.runAddonTest(true)
	return err
}

// runAddonTestMatrix contains the core matrix execution logic without matrix-level error reporting
// This private method is used by both RunAddonTestMatrix() and permutation tests
func (options *TestAddonOptions) runAddonTestMatrix(matrix AddonTestMatrix) {
	// Note: Parent test is NOT made parallel to avoid blocking other tests from starting
	// while all subtests are being created. Subtests are still parallel with each other.
	// options.Testing.Parallel()

	// Validate that BaseOptions is provided
	if matrix.BaseOptions == nil {
		panic("BaseOptions must be provided for AddonTestMatrix")
	}

	// Capture the parent test name to avoid duplication in logger when creating subtests
	parentTestName := options.Testing.Name()

	// Set default stagger delay if not specified
	staggerDelay := 10 * time.Second
	if matrix.StaggerDelay != nil {
		staggerDelay = *matrix.StaggerDelay
	}

	// Set default batch configuration for staggered execution
	batchSize := 8
	if matrix.StaggerBatchSize != nil {
		batchSize = *matrix.StaggerBatchSize
	}

	withinBatchDelay := 2 * time.Second
	if matrix.WithinBatchDelay != nil {
		withinBatchDelay = *matrix.WithinBatchDelay
	}

	// Create shared resource tracking for the matrix
	var sharedCatalogOptions *TestAddonOptions
	var sharedMutex = &sync.Mutex{}

	// CRITICAL SYNCHRONIZATION SETUP for parallel test result collection
	// This solves the "parallel within parallel" execution problem where:
	// 1. Parent test (options.Testing.Parallel()) becomes parallel
	// 2. Each subtest (t.Parallel()) also becomes parallel
	// 3. Go's execution model causes parent to complete before subtests finish
	// 4. Standard cleanup (defer/t.Cleanup) would execute before result collection completes
	//
	// WaitGroup coordination ensures report generation waits for ALL subtests to complete
	var resultWg sync.WaitGroup
	var resultMutex sync.Mutex // Protect concurrent result collection from parallel subtests

	// CRITICAL TIMING: Register cleanup BEFORE starting parallel subtests
	// This ensures cleanup is properly scheduled in the parent's cleanup chain
	// before parallel execution begins. If registered after subtests start,
	// Go's parallel test execution model may cause cleanup to never execute.
	if options.CollectResults && options.PermutationTestReport != nil {
		options.Testing.Cleanup(func() {
			// Wait for all parallel subtests to complete result collection
			// Timeout protection prevents hanging if subtests fail to signal completion
			done := make(chan struct{})
			go func() {
				resultWg.Wait() // Wait for all subtests to complete
				close(done)
			}()

			// Wait for completion or timeout after 30 seconds
			select {
			case <-done:
				// All subtests completed normally
			case <-time.After(30 * time.Second):
				// Timeout protection: generate report with available results
				options.Logger.ShortWarn("Timeout waiting for all subtests to complete - generating report with available results")
			}

			// Generate final report after all matrix tests complete (or timeout)
			resultMutex.Lock() // Protect against any remaining concurrent access
			options.PermutationTestReport.EndTime = time.Now()
			resultMutex.Unlock()

			// Use SmartLogger if available for consistent formatting
			if smartLogger, ok := options.Logger.(*common.SmartLogger); ok {
				options.PermutationTestReport.PrintPermutationReport(smartLogger)
			} else {
				// Fallback: log to standard output if SmartLogger is not available
				fmt.Printf("\n=== PERMUTATION TEST REPORT ===\n")
				fmt.Printf("Total: %d, Passed: %d, Failed: %d\n",
					options.PermutationTestReport.TotalTests,
					options.PermutationTestReport.PassedTests,
					options.PermutationTestReport.FailedTests)
				if options.PermutationTestReport.FailedTests > 0 {
					fmt.Printf("See individual test failures above for details.\n")
				}
				fmt.Printf("===============================\n\n")
			}

			// Fail the overall test if there were any individual test failures
			// This ensures proper test result while still showing the complete report
			if options.PermutationTestReport.FailedTests > 0 {
				options.Testing.Errorf("Matrix tests failed: %d out of %d tests failed - see comprehensive report above for details",
					options.PermutationTestReport.FailedTests, options.PermutationTestReport.TotalTests)
			}
		})
	}

	// Generate a random prefix once per matrix test run for UI grouping
	randomPrefix := common.UniqueId(6)

	// Initialize TotalTests count for accurate report generation
	if options.CollectResults && options.PermutationTestReport != nil {
		resultMutex.Lock()
		options.PermutationTestReport.TotalTests = len(matrix.TestCases)
		resultMutex.Unlock()
	}

	for i, tc := range matrix.TestCases {
		tc := tc       // Capture loop variable for parallel execution
		testIndex := i // Capture index for staggering

		// Increment WaitGroup for each subtest to ensure proper synchronization
		// Each parallel subtest must signal completion for reliable report generation
		if options.CollectResults && options.PermutationTestReport != nil {
			resultWg.Add(1)
		}

		options.Testing.Run(tc.Name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					// Don't re-panic, just log it and let test fail gracefully
					t.Errorf("Matrix test %s failed due to unhandled panic: %v", tc.Name, r)
				}
			}()

			// Variable to capture any panic as an error for result collection
			var testErr error
			var panicOccurred bool

			// Ensure WaitGroup.Done() is called even if subtest panics
			// This prevents the parent cleanup from hanging indefinitely
			if options.CollectResults && options.PermutationTestReport != nil {
				defer func() {
					if r := recover(); r != nil {
						// Convert panic to error for result collection
						testErr = fmt.Errorf("panic occurred: %v", r)
						panicOccurred = true
						options.Logger.ShortError(fmt.Sprintf("Subtest %s panicked: %v", tc.Name, r))

						// Fail the test but don't re-panic to allow graceful cleanup
						t.Errorf("Test failed due to panic: %v", r)
					}
					resultWg.Done() // Always signal completion
				}()
			}

			// Implement staggered start to prevent rate limiting
			// Use batched approach to prevent excessive delays for large test suites
			if staggerDelay > 0 && testIndex > 0 {
				var staggerWait time.Duration
				var batchNumber, inBatchIndex int

				if batchSize > 0 {
					// Batched staggering: group tests into batches with smaller delays within batches
					batchNumber = testIndex / batchSize
					inBatchIndex = testIndex % batchSize
					staggerWait = time.Duration(batchNumber)*staggerDelay + time.Duration(inBatchIndex)*withinBatchDelay
				} else {
					// Linear staggering: original behavior when batch size is 0
					staggerWait = time.Duration(testIndex) * staggerDelay
					batchNumber = 0
					inBatchIndex = testIndex
				}

				// Only log stagger messages in verbose mode
				if !matrix.BaseOptions.QuietMode {
					if batchSize > 0 {
						matrix.BaseOptions.Logger.ShortInfo(fmt.Sprintf("[%s - STAGGER] Delaying test start by %v to prevent rate limiting (batch %d, position %d/%d, test %d/%d)",
							tc.Name, staggerWait, batchNumber, inBatchIndex+1, batchSize, testIndex+1, len(matrix.TestCases)))
					} else {
						matrix.BaseOptions.Logger.ShortInfo(fmt.Sprintf("[%s - STAGGER] Delaying test start by %v to prevent rate limiting (test %d/%d)",
							tc.Name, staggerWait, testIndex+1, len(matrix.TestCases)))
					}
				}
				time.Sleep(staggerWait)
			}

			// Enable parallel execution after stagger delay to ensure tests are released in order
			t.Parallel()

			// Show test start progress for permutation tests or when in quiet mode
			if matrix.BaseOptions.QuietMode || matrix.BaseOptions.PermutationTestReport != nil {
				matrix.BaseOptions.Logger.ProgressStage(fmt.Sprintf("Starting test: %s", tc.Name))
			}

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

			// Ensure logger is initialized before using it - create unique logger per test case
			if testOptions.Logger == nil {
				// Use specific test case name for better identification in logs
				testCaseName := fmt.Sprintf("%s/%s", parentTestName, tc.Name)
				testOptions.Logger = common.CreateSmartAutoBufferingLogger(testCaseName, testOptions.QuietMode)
			} else {
				// Preserve existing logger (it may have buffered content) but ensure QuietMode is correct
				testOptions.Logger.SetQuietMode(testOptions.QuietMode)
			}

			if !testOptions.QuietMode {
				// Show individual test start messages in verbose mode
				testOptions.Logger.ShortInfo(fmt.Sprintf("Running test: %s", tc.Name))
			}

			// Ensure CloudInfoService is initialized before using it for catalog operations
			if testOptions.CloudInfoService == nil {
				cloudInfoSvc, err := cloudinfo.NewCloudInfoServiceFromEnv("TF_VAR_ibmcloud_api_key", cloudinfo.CloudInfoServiceOptions{
					Logger: testOptions.Logger,
				})
				if err != nil {
					require.NoError(t, err, "Failed to initialize CloudInfoService")
					return
				}
				testOptions.CloudInfoService = cloudInfoSvc
			} else {
				// Update the existing CloudInfoService logger with quiet mode setting
				testOptions.CloudInfoService.SetLogger(testOptions.Logger.GetUnderlyingLogger())
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

			// Set project name using test case and prefix with abbreviations
			if testOptions.Prefix != "" {
				nameComponents := []string{randomPrefix}

				if testOptions.AddonConfig.OfferingName != "" {
					// Use abbreviation for offering name
					offeringAbbrev := testOptions.createInitialAbbreviation(testOptions.AddonConfig.OfferingName)
					nameComponents = append(nameComponents, offeringAbbrev)
				}

				// Add abbreviated test case name for readability
				if tc.Name != "" {
					testCaseAbbrev := testOptions.createInitialAbbreviation(strings.ToLower(tc.Name))
					nameComponents = append(nameComponents, testCaseAbbrev)
				}

				nameComponents = append(nameComponents, testOptions.Prefix)
				testOptions.ProjectName = strings.Join(nameComponents, "-")
			}

			// Merge any additional inputs from the test case
			if tc.Inputs != nil && len(tc.Inputs) > 0 {
				if testOptions.AddonConfig.Inputs == nil {
					testOptions.AddonConfig.Inputs = make(map[string]interface{})
				}

				// Check OverrideInputMappings flag behavior
				if testOptions.OverrideInputMappings != nil && !*testOptions.OverrideInputMappings {
					// Use cached reference information (zero additional API calls)
					configReferences := testOptions.configInputReferences[testOptions.AddonConfig.ConfigID]

					for key, newValue := range tc.Inputs {
						if referenceValue, isReference := configReferences[key]; isReference {
							if !testOptions.QuietMode {
								testOptions.Logger.ShortInfo(fmt.Sprintf("Preserving reference value for input '%s': %s", key, referenceValue))
							}
							// Keep the existing reference value
							testOptions.AddonConfig.Inputs[key] = referenceValue
						} else {
							// Safe to override - not a reference
							if !testOptions.QuietMode {
								existingValue := testOptions.AddonConfig.Inputs[key]
								testOptions.Logger.ShortInfo(fmt.Sprintf("Overriding input '%s': %v → %v", key, existingValue, newValue))
							}
							testOptions.AddonConfig.Inputs[key] = newValue
						}
					}
				} else {
					// Current behavior - override all inputs
					if !testOptions.QuietMode {
						testOptions.Logger.ShortInfo(fmt.Sprintf("OverrideInputMappings=true: overriding %d input(s)", len(tc.Inputs)))
					}
					for key, value := range tc.Inputs {
						if !testOptions.QuietMode {
							existingValue := testOptions.AddonConfig.Inputs[key]
							testOptions.Logger.ShortInfo(fmt.Sprintf("Overriding input '%s': %v → %v", key, existingValue, value))
						}
						testOptions.AddonConfig.Inputs[key] = value
					}
				}

				// Log summary of input merging behavior (non-quiet mode only)
				if !testOptions.QuietMode && tc.Inputs != nil && len(tc.Inputs) > 0 {
					preserveRefs := testOptions.OverrideInputMappings != nil && !*testOptions.OverrideInputMappings
					if preserveRefs {
						testOptions.Logger.ShortInfo(fmt.Sprintf("Input merging: preserve-references mode, processed %d input(s)", len(tc.Inputs)))
					} else {
						testOptions.Logger.ShortInfo(fmt.Sprintf("Input merging: override-all mode, processed %d input(s)", len(tc.Inputs)))
					}
				}
			}

			// Handle shared catalog creation in matrix tests
			sharedMutex.Lock()
			if sharedCatalogOptions == nil {
				// This is the first test case - it will create the shared catalog and offering
				sharedCatalogOptions = testOptions

				// First, validate that the branch exists in the remote repository BEFORE creating any resources
				// Use the new cloudinfo helper for offering import preparation
				_, _, _, err := testOptions.CloudInfoService.PrepareOfferingImport()
				if err != nil {
					sharedMutex.Unlock()
					testOptions.Logger.ShortError(fmt.Sprintf("Failed to prepare offering import: %v", err))
					require.NoError(t, err, "Failed to prepare offering import")
					return
				}

				// Create the shared catalog for matrix tests
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

					// Import the offering using the new helper function
					version := fmt.Sprintf("v0.0.1-dev-%s", testOptions.Prefix)
					testOptions.AddonConfig.ResolvedVersion = version

					testOptions.Logger.ShortInfo(fmt.Sprintf("Importing shared offering: %s as version: %s", testOptions.AddonConfig.OfferingFlavor, version))

					offering, err := testOptions.CloudInfoService.ImportOfferingWithValidation(
						*testOptions.catalog.ID,
						testOptions.AddonConfig.OfferingName,
						testOptions.AddonConfig.OfferingFlavor,
						version,
						testOptions.AddonConfig.OfferingInstallKind,
					)
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
			// Use runAddonTest() with simple error messages for matrix execution
			err := testOptions.runAddonTest(false)

			// If a panic occurred, use the panic error instead
			if panicOccurred {
				err = testErr
			}

			// Force log completion to main test logger to ensure it appears in logs
			completionStatus := "PASSED"
			if err != nil {
				completionStatus = fmt.Sprintf("FAILED (error: %v)", err)
			}
			if matrix.BaseOptions.Logger != nil {
				if smartLogger, ok := matrix.BaseOptions.Logger.(*common.SmartLogger); ok {
					if err != nil {
						smartLogger.ImmediateShortError(fmt.Sprintf("MATRIX TEST COMPLETION: %s - %s", tc.Name, completionStatus))
					} else {
						smartLogger.ImmediateShortInfo(fmt.Sprintf("MATRIX TEST COMPLETION: %s - %s", tc.Name, completionStatus))
					}
				} else if bufferedLogger, ok := matrix.BaseOptions.Logger.(*common.BufferedTestLogger); ok {
					if err != nil {
						bufferedLogger.ImmediateShortError(fmt.Sprintf("MATRIX TEST COMPLETION: %s - %s", tc.Name, completionStatus))
					} else {
						bufferedLogger.ImmediateShortInfo(fmt.Sprintf("MATRIX TEST COMPLETION: %s - %s", tc.Name, completionStatus))
					}
				} else {
					if err != nil {
						matrix.BaseOptions.Logger.ShortError(fmt.Sprintf("MATRIX TEST COMPLETION: %s - %s", tc.Name, completionStatus))
					} else {
						matrix.BaseOptions.Logger.ShortInfo(fmt.Sprintf("MATRIX TEST COMPLETION: %s - %s", tc.Name, completionStatus))
					}
				}
			}

			// Thread-safe result collection for parallel subtests
			// Mutex protection is required because multiple parallel subtests may
			// simultaneously append to the shared PermutationTestReport.Results slice.
			// Go's slice operations are not thread-safe for concurrent writes.
			if testOptions.CollectResults && testOptions.PermutationTestReport != nil {
				// Create a clean AddonConfig for reporting that shows only the original test case dependencies
				// This ensures the summary correctly shows 4 direct dependencies, not the 6 after processing
				reportAddonConfig := cloudinfo.AddonConfig{
					OfferingName:   testOptions.AddonConfig.OfferingName,
					OfferingID:     testOptions.AddonConfig.OfferingID,
					OfferingLabel:  testOptions.AddonConfig.OfferingLabel,
					VersionLocator: testOptions.AddonConfig.VersionLocator,
					Dependencies:   tc.Dependencies, // Use original test case dependencies, not processed ones
				}
				testResult := testOptions.collectTestResult(tc.Name, tc.Prefix, reportAddonConfig, err)

				// CRITICAL: Protect concurrent access to shared report data
				resultMutex.Lock()
				testOptions.PermutationTestReport.Results = append(testOptions.PermutationTestReport.Results, testResult)
				if testResult.Passed {
					testOptions.PermutationTestReport.PassedTests++
					testOptions.Logger.ShortInfo(fmt.Sprintf("Collected PASSED result for: %s (total results: %d)", tc.Name, len(testOptions.PermutationTestReport.Results)))
				} else {
					testOptions.PermutationTestReport.FailedTests++
					testOptions.Logger.ShortInfo(fmt.Sprintf("Collected FAILED result for: %s (total results: %d, error: %v)", tc.Name, len(testOptions.PermutationTestReport.Results), err))
				}
				resultMutex.Unlock()
			} else {
				// Log when result collection is skipped to help debug missing results
				if !testOptions.CollectResults {
					testOptions.Logger.ShortWarn(fmt.Sprintf("Skipping result collection for %s: CollectResults=false", tc.Name))
				} else if testOptions.PermutationTestReport == nil {
					testOptions.Logger.ShortWarn(fmt.Sprintf("Skipping result collection for %s: PermutationTestReport=nil", tc.Name))
				}
			}

			// Handle result display in quiet mode
			if testOptions.QuietMode {
				if err != nil {
					testOptions.Logger.ShortError(fmt.Sprintf("✗ Failed: %s (error: %v)", tc.Name, err))
				} else {
					testOptions.Logger.ProgressSuccess(fmt.Sprintf("Passed: %s", tc.Name))
				}
			}

			// Don't fail individual tests - we collect all results and report at the end
			// This ensures the final comprehensive report is always generated
		})
	}

	// NOTE: Report generation is now handled by t.Cleanup() registered BEFORE subtests
	// This was moved to ensure proper timing in Go's parallel test execution model

	// Cleanup shared resources after all tests complete
	go func() {
		options.Testing.Cleanup(func() {
			if sharedCatalogOptions != nil {
				sharedCatalogOptions.CleanupSharedResources()
			}
		})
	}()
}

// RunAddonTestMatrix runs multiple addon test cases in parallel using a matrix approach
// This method handles the boilerplate of running parallel tests and automatically shares
// catalogs and offerings across test cases for efficiency.
//
// BaseOptions must be provided with common options that apply to all test cases.
// BaseSetupFunc can optionally customize the options for each specific test case.
// This public method provides matrix-level error reporting for direct matrix execution
func (options *TestAddonOptions) RunAddonTestMatrix(matrix AddonTestMatrix) {
	// Enable quiet mode by default for matrix tests to reduce log noise
	// Allow user to override by explicitly setting QuietMode = false
	// If Logger is already set and has QuietMode false, preserve user's choice
	// Preserve user's QuietMode setting - do not override it
	// If user hasn't explicitly set QuietMode and no logger exists, default to true for matrix tests
	if options.Logger == nil && !options.QuietMode {
		// Only default to quiet mode if neither QuietMode nor Logger were explicitly set
		options.QuietMode = true
	}

	// If logger exists, respect its QuietMode setting
	if options.Logger != nil && !options.Logger.IsQuietMode() {
		// Logger exists and is not in quiet mode - respect this setting
		options.QuietMode = false
	}

	options.runAddonTestMatrix(matrix)
	// Matrix-level error reporting could be added here in the future
	// For now, individual test case errors are handled within the matrix execution
}

// RunAddonPermutationTest executes all permutations of direct dependencies for the addon
// without manual configuration. It automatically discovers dependencies and generates
// all enabled/disabled combinations, excluding the default "on by default" case.
// All permutations skip infrastructure deployment for efficiency.
func (options *TestAddonOptions) RunAddonPermutationTest() error {
	require.NotNil(options.Testing, options.Testing, "Testing is required")
	require.NotEmpty(options.Testing, options.AddonConfig.OfferingName, "AddonConfig.OfferingName is required")
	require.NotEmpty(options.Testing, options.AddonConfig.OfferingFlavor, "AddonConfig.OfferingFlavor is required")
	require.NotEmpty(options.Testing, options.Prefix, "Prefix is required")

	// Ensure OfferingInstallKind is set to a valid value, defaulting to Terraform
	if !options.AddonConfig.OfferingInstallKind.Valid() {
		options.AddonConfig.OfferingInstallKind = cloudinfo.InstallKindTerraform
	}

	// For permutation tests, we apply quiet mode per-test (not globally)
	// This allows STAGGER messages and test phases to be visible
	// while suppressing verbose details within each individual test

	// Step 1: Discover dependency names and flavors from catalog
	dependenciesWithFlavors, err := options.getDependenciesWithFlavors()
	if err != nil {
		return fmt.Errorf("failed to discover dependencies with flavors: %w", err)
	}

	if len(dependenciesWithFlavors) == 0 {
		options.Testing.Skip("No dependencies found to test permutations")
		return nil
	}

	// Step 2: Generate all permutations of dependencies including flavor combinations
	testCases := options.generatePermutationsWithFlavors(dependenciesWithFlavors)

	if len(testCases) == 0 {
		options.Testing.Skip("No permutations generated (all would be default configuration)")
		return nil
	}

	// Step 3: Initialize result collection and logging
	if options.Logger == nil {
		options.Logger = common.CreateSmartAutoBufferingLogger("TestDependencyPermutations", false)
	}

	// For permutation tests, keep the global logger in verbose mode to show STAGGER messages and test phases
	// Individual test quiet mode will be applied in BaseSetupFunc to suppress verbose details within each test
	options.Logger.SetQuietMode(false)

	// Initialize result collection for final report
	options.CollectResults = true
	options.PermutationTestReport = &PermutationTestReport{
		TotalTests:  len(testCases),
		PassedTests: 0,
		FailedTests: 0,
		Results:     make([]PermutationTestResult, 0, len(testCases)),
		StartTime:   time.Now(),
	}

	// Show permutation test start progress
	options.Logger.ProgressStage(fmt.Sprintf("Running %d dependency permutation tests for %s (per-test quiet mode)...", len(testCases), options.AddonConfig.OfferingName))

	// Step 4: Execute all permutations in parallel using matrix test infrastructure
	matrix := AddonTestMatrix{
		TestCases:        testCases,
		BaseOptions:      options,
		StaggerDelay:     options.StaggerDelay,
		StaggerBatchSize: options.StaggerBatchSize,
		WithinBatchDelay: options.WithinBatchDelay,
		BaseSetupFunc: func(baseOptions *TestAddonOptions, testCase AddonTestCase) *TestAddonOptions {
			// Clone base options for each test case
			testOptions := baseOptions.copy()
			testOptions.Prefix = testCase.Prefix
			testOptions.TestCaseName = testCase.Name
			testOptions.SkipInfrastructureDeployment = testCase.SkipInfrastructureDeployment
			// Apply per-test quiet mode to suppress verbose details within each test
			// while preserving STAGGER messages and test phases visibility
			testOptions.QuietMode = true
			testOptions.VerboseOnFailure = baseOptions.VerboseOnFailure

			// Synchronize logger quiet mode with the boolean flag
			if testOptions.Logger != nil {
				testOptions.Logger.SetQuietMode(testOptions.QuietMode)
			}

			return testOptions
		},
		AddonConfigFunc: func(testOptions *TestAddonOptions, testCase AddonTestCase) cloudinfo.AddonConfig {
			// Create a proper copy that preserves the original inputs
			config := testOptions.AddonConfig

			// Copy the Inputs map to avoid sharing reference
			if config.Inputs != nil {
				inputsCopy := make(map[string]interface{})
				for k, v := range config.Inputs {
					inputsCopy[k] = v
				}
				config.Inputs = inputsCopy
			}

			// Set permutation-specific values
			config.Dependencies = testCase.Dependencies
			config.Prefix = testOptions.Prefix

			return config
		},
	}

	// Execute the matrix test - the final report will be generated by the matrix cleanup
	// Use runAddonTestMatrix() to avoid matrix-level reporting in permutation tests
	options.runAddonTestMatrix(matrix)

	return nil
}

// getDirectDependencyNames discovers just the names of direct dependencies from the local ibm_catalog.json file
// This replaces the expensive catalog import operations with lightweight local file parsing
func (options *TestAddonOptions) getDirectDependencyNames() ([]string, error) {
	// Check if test has injected a custom dependency names function (for testing)
	if options.GetDirectDependencyNames != nil {
		return options.GetDirectDependencyNames()
	}

	// Find the git root directory to locate ibm_catalog.json
	gitRoot, err := common.GitRootPath(".")
	if err != nil {
		return nil, fmt.Errorf("failed to find git root: %w", err)
	}

	// Construct the path to ibm_catalog.json
	catalogPath := filepath.Join(gitRoot, "ibm_catalog.json")

	// Read the local ibm_catalog.json file
	jsonFile, err := os.ReadFile(catalogPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read ibm_catalog.json from %s: %w", catalogPath, err)
	}

	// Parse the JSON into CatalogJson struct
	var catalogConfig cloudinfo.CatalogJson
	err = json.Unmarshal(jsonFile, &catalogConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ibm_catalog.json: %w", err)
	}

	// Find the matching product by OfferingName
	var targetProductIndex = -1
	for i := range catalogConfig.Products {
		if catalogConfig.Products[i].Name == options.AddonConfig.OfferingName {
			targetProductIndex = i
			break
		}
	}

	if targetProductIndex == -1 {
		return nil, fmt.Errorf("product '%s' not found in ibm_catalog.json", options.AddonConfig.OfferingName)
	}

	// Find the matching flavor by OfferingFlavor
	var targetFlavorIndex = -1
	for i := range catalogConfig.Products[targetProductIndex].Flavors {
		if catalogConfig.Products[targetProductIndex].Flavors[i].Name == options.AddonConfig.OfferingFlavor {
			targetFlavorIndex = i
			break
		}
	}

	if targetFlavorIndex == -1 {
		return nil, fmt.Errorf("flavor '%s' not found in product '%s' in ibm_catalog.json",
			options.AddonConfig.OfferingFlavor, options.AddonConfig.OfferingName)
	}

	// Extract dependency names from the dependencies array
	var dependencyNames []string
	targetFlavor := catalogConfig.Products[targetProductIndex].Flavors[targetFlavorIndex]
	for _, dependency := range targetFlavor.Dependencies {
		if dependency.Name != "" {
			dependencyNames = append(dependencyNames, dependency.Name)
		}
	}

	return dependencyNames, nil
}

// DependencyWithFlavors represents a dependency with all its available flavors
type DependencyWithFlavors struct {
	Name    string
	Flavors []string
}

// getDependenciesWithFlavors discovers direct dependencies and their available flavors from the local ibm_catalog.json file
func (options *TestAddonOptions) getDependenciesWithFlavors() ([]DependencyWithFlavors, error) {
	// Find the git root directory to locate ibm_catalog.json
	gitRoot, err := common.GitRootPath(".")
	if err != nil {
		return nil, fmt.Errorf("failed to find git root: %w", err)
	}

	// Construct the path to ibm_catalog.json
	catalogPath := filepath.Join(gitRoot, "ibm_catalog.json")

	// Read the local ibm_catalog.json file
	jsonFile, err := os.ReadFile(catalogPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read ibm_catalog.json from %s: %w", catalogPath, err)
	}

	// Parse the JSON into CatalogJson struct
	var catalogConfig cloudinfo.CatalogJson
	err = json.Unmarshal(jsonFile, &catalogConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ibm_catalog.json: %w", err)
	}

	// Find the matching product by OfferingName
	var targetProductIndex = -1
	for i := range catalogConfig.Products {
		if catalogConfig.Products[i].Name == options.AddonConfig.OfferingName {
			targetProductIndex = i
			break
		}
	}

	if targetProductIndex == -1 {
		return nil, fmt.Errorf("product '%s' not found in ibm_catalog.json", options.AddonConfig.OfferingName)
	}

	// Find the matching flavor by OfferingFlavor
	var targetFlavorIndex = -1
	for i := range catalogConfig.Products[targetProductIndex].Flavors {
		if catalogConfig.Products[targetProductIndex].Flavors[i].Name == options.AddonConfig.OfferingFlavor {
			targetFlavorIndex = i
			break
		}
	}

	if targetFlavorIndex == -1 {
		return nil, fmt.Errorf("flavor '%s' not found in product '%s' in ibm_catalog.json",
			options.AddonConfig.OfferingFlavor, options.AddonConfig.OfferingName)
	}

	// Extract dependencies with their flavors
	var dependenciesWithFlavors []DependencyWithFlavors
	targetFlavor := catalogConfig.Products[targetProductIndex].Flavors[targetFlavorIndex]

	for _, dependency := range targetFlavor.Dependencies {
		if dependency.Name != "" {
			dependenciesWithFlavors = append(dependenciesWithFlavors, DependencyWithFlavors{
				Name:    dependency.Name,
				Flavors: dependency.Flavors,
			})
		}
	}

	return dependenciesWithFlavors, nil
}

// validateAndProcessRequiredDependencies applies the required dependency business logic to manually configured dependencies
// This ensures consistent behavior with the catalog processing that happens for discovered dependencies
func (options *TestAddonOptions) validateAndProcessRequiredDependencies() error {
	// Log CloudInfoService availability - respect quiet mode for this informational message
	if options.CloudInfoService == nil {
		if !options.Logger.IsQuietMode() {
			options.Logger.ShortInfo("CloudInfoService not available - processing dependencies with existing metadata only")
		}
	} else {
		if !options.Logger.IsQuietMode() {
			options.Logger.ShortInfo("CloudInfoService is available - will query catalog metadata")
		}
	}

	// Process the root addon and all its dependencies recursively
	err := options.processRequiredDependenciesRecursively(&options.AddonConfig, options.AddonConfig.OfferingName)
	if err != nil {
		return fmt.Errorf("failed to process required dependencies: %w", err)
	}

	return nil
}

// processRequiredDependenciesRecursively walks through dependencies and applies business rules for required dependencies
func (options *TestAddonOptions) processRequiredDependenciesRecursively(config *cloudinfo.AddonConfig, parentName string) error {
	var forceEnabledDeps []string
	var requiredDeps []string
	var optionalDeps []string

	// Log the processing header if we have dependencies - respect quiet mode for verbose information
	if len(config.Dependencies) > 0 && !options.Logger.IsQuietMode() {
		msg := fmt.Sprintf("Processing dependencies for %s:", parentName)
		options.Logger.ShortInfo(msg)
	}

	for i := range config.Dependencies {
		dep := &config.Dependencies[i]

		// Determine if this dependency is required
		isRequired := false
		if dep.Enabled != nil && !*dep.Enabled {
			// Check if this dependency is required by getting its catalog metadata
			var err error
			isRequired, err = options.checkIfDependencyIsRequired(dep, parentName)
			if err != nil {
				options.Logger.ShortWarn(fmt.Sprintf("Could not check if dependency %s is required: %v", dep.OfferingName, err))
				continue
			}

			if isRequired {
				// Apply business rule: force-enable required dependencies
				dep.Enabled = core.BoolPtr(true)
				dep.IsRequired = core.BoolPtr(true)
				dep.RequiredBy = []string{parentName}
				forceEnabledDeps = append(forceEnabledDeps, dep.OfferingName)

				// Handle StrictMode logging
				if options.StrictMode == nil || *options.StrictMode {
					// Strict mode: warn but continue
					options.Logger.ShortWarn(fmt.Sprintf("Required dependency %s was force-enabled despite being disabled", dep.OfferingName))
					options.Logger.ShortWarn(fmt.Sprintf("  Required by: %s", parentName))
					options.Logger.ShortWarn("  Use StrictMode=false to suppress this warning")
				} else {
					// Non-strict mode: informational message and capture warning for final report
					options.Logger.ShortInfo(fmt.Sprintf("Required dependency %s was force-enabled (required by %s)", dep.OfferingName, parentName))

					// Add to validation warnings for final report display
					if options.lastValidationResult == nil {
						options.lastValidationResult = &ValidationResult{
							IsValid:  true, // Still valid in non-strict mode
							Warnings: []string{},
						}
					}
					warningMsg := fmt.Sprintf("Required dependency %s was force-enabled despite being disabled (required by %s)", dep.OfferingName, parentName)
					options.lastValidationResult.Warnings = append(options.lastValidationResult.Warnings, warningMsg)
				}
			}
		} else {
			// Check if already marked as required from previous processing
			if dep.IsRequired != nil && *dep.IsRequired {
				isRequired = true
			}
		}

		// Log individual dependency status with tree structure
		var status string
		if isRequired {
			requiredDeps = append(requiredDeps, dep.OfferingName)
			status = fmt.Sprintf("├── %s (REQUIRED by %s)", dep.OfferingName, parentName)
		} else {
			optionalDeps = append(optionalDeps, dep.OfferingName)
			status = fmt.Sprintf("└── %s (OPTIONAL)", dep.OfferingName)
		}
		// Only show individual dependency status in verbose mode
		if !options.Logger.IsQuietMode() {
			options.Logger.ShortInfo(status)
		}

		// Recursively process nested dependencies
		err := options.processRequiredDependenciesRecursively(dep, dep.OfferingName)
		if err != nil {
			return err
		}
	}

	// Log comprehensive summary - respect quiet mode for verbose information
	if len(config.Dependencies) > 0 && !options.Logger.IsQuietMode() {
		totalDeps := len(config.Dependencies)
		requiredCount := len(requiredDeps)
		optionalCount := len(optionalDeps)
		forceEnabledCount := len(forceEnabledDeps)

		summaryMsg := fmt.Sprintf("Summary: %d dependencies processed, %d required (%d force-enabled), %d optional",
			totalDeps, requiredCount, forceEnabledCount, optionalCount)

		options.Logger.ShortInfo(summaryMsg)
	}

	return nil
}

// checkIfDependencyIsRequired checks if a dependency is marked as required in the catalog metadata
func (options *TestAddonOptions) checkIfDependencyIsRequired(dep *cloudinfo.AddonConfig, parentName string) (bool, error) {
	// If we already have the IsRequired metadata, use it
	if dep.IsRequired != nil {
		return *dep.IsRequired, nil
	}

	// If CloudInfoService is not available, we can't check catalog metadata
	if options.CloudInfoService == nil {
		// Default to not required when we can't check catalog
		return false, nil
	}

	// Query the parent offering's catalog information to check if this dependency is required
	// We need to find the parent AddonConfig to get its catalog information
	parentConfig := options.findParentAddonConfig(parentName)
	if parentConfig == nil {
		// Parent config not found - default to not required (this is fine for simple dependency configs)
		return false, nil
	}

	// Check if we have the necessary catalog metadata for the lookup
	if parentConfig.CatalogID == "" || parentConfig.OfferingID == "" {
		// No catalog metadata available - this is expected for simple dependency configurations
		// Default to not required without warning (test setup already logs catalog details)
		return false, nil
	}

	// Get the parent offering details from catalog
	offering, _, err := options.CloudInfoService.GetOffering(parentConfig.CatalogID, parentConfig.OfferingID)
	if err != nil {
		// Catalog lookup failed - default to not required without warning
		// (test setup already shows catalog creation details)
		return false, nil
	}

	// Find the version that matches the parent config
	var version *catalogmanagementv1.Version
	for _, kind := range offering.Kinds {
		if *kind.InstallKind == "terraform" {
			for _, v := range kind.Versions {
				if *v.VersionLocator == parentConfig.VersionLocator {
					version = &v
					break
				}
			}
		}
		if version != nil {
			break
		}
	}

	if version == nil {
		// Version not found - default to not required
		return false, nil
	}

	// Check the catalog dependencies to see if this dependency is marked as required
	for _, catalogDep := range version.SolutionInfo.Dependencies {
		if *catalogDep.Name == dep.OfferingName {
			// Found the dependency in catalog metadata
			// A dependency is required if it's NOT optional (required = !optional)
			if catalogDep.Optional != nil {
				return !*catalogDep.Optional, nil
			}
			// If Optional field is not set, check OnByDefault as fallback
			// Dependencies that are on by default but optional can be disabled
			if catalogDep.OnByDefault != nil {
				// If it's on by default but no explicit optional field, assume it's optional
				return false, nil
			}
		}
	}

	// If dependency not found in catalog metadata, default to not required
	return false, nil
}

// findParentAddonConfig finds the AddonConfig for a given offering name
// This helps locate the catalog information needed to check dependency requirements
func (options *TestAddonOptions) findParentAddonConfig(offeringName string) *cloudinfo.AddonConfig {
	// Check if this is the root addon
	if options.AddonConfig.OfferingName == offeringName {
		return &options.AddonConfig
	}

	// Recursively search through dependencies
	return options.findAddonConfigRecursively(&options.AddonConfig, offeringName)
}

// findAddonConfigRecursively searches for an AddonConfig by offering name in the dependency tree
func (options *TestAddonOptions) findAddonConfigRecursively(config *cloudinfo.AddonConfig, offeringName string) *cloudinfo.AddonConfig {
	if config.OfferingName == offeringName {
		return config
	}

	// Search in dependencies
	for i := range config.Dependencies {
		if result := options.findAddonConfigRecursively(&config.Dependencies[i], offeringName); result != nil {
			return result
		}
	}

	return nil
}

// generatePermutations creates all enabled/disabled combinations of dependencies
func (options *TestAddonOptions) generatePermutations(dependencyNames []string) []AddonTestCase {

	var testCases []AddonTestCase

	// Generate 2^n - 1 permutations of dependencies (skips the all-enabled case)
	// Example: For 2 dependencies [dep1, dep2], generates 3 permutations:
	// - dep1 disabled, dep2 disabled
	// - dep1 enabled, dep2 disabled
	// - dep1 disabled, dep2 enabled
	// (skips: dep1 enabled, dep2 enabled)
	numDeps := len(dependencyNames)
	totalPermutations := 1 << numDeps // 2^n where n = number of dependencies

	// Generate a random prefix once per test run for UI grouping
	randomPrefix := common.UniqueId(6)

	// Sequential counter for unique prefix generation
	permIndex := 0
	basePrefix := options.shortenPrefix(options.Prefix)

	for i := 0; i < totalPermutations; i++ {
		// Create simple dependency configs for this permutation (like manual tests)
		var permutationDeps []cloudinfo.AddonConfig
		var disabledNames []string

		// Set enabled/disabled based on bit pattern - create simple configs like manual tests
		for j, depName := range dependencyNames {
			enabled := (i & (1 << j)) != 0

			// Create simple dependency config (same format as manual tests)
			dep := cloudinfo.AddonConfig{
				OfferingName: depName,
				Enabled:      core.BoolPtr(enabled),
			}
			permutationDeps = append(permutationDeps, dep)

			if !enabled {
				disabledNames = append(disabledNames, depName)
			}
		}

		// Skip the "all enabled" case (default configuration)
		// This excludes the combination where all dependencies are enabled
		if len(disabledNames) == 0 {
			continue
		}

		// Create test case name using abbreviations with collision resolution
		mainOfferingAbbrev := options.createInitialAbbreviation(options.AddonConfig.OfferingName)

		testCaseName := fmt.Sprintf("%s-%s-%d", randomPrefix, mainOfferingAbbrev, permIndex)
		if len(disabledNames) > 0 {
			// Use abbreviation with collision resolution for disabled dependency names
			abbreviatedDisabledNames := options.abbreviateWithCollisionResolution(disabledNames)
			testCaseName = fmt.Sprintf("%s-%s-%d-disable-%s", randomPrefix, mainOfferingAbbrev, permIndex,
				strings.Join(abbreviatedDisabledNames, "-"))

		}

		// Generate unique prefix using random prefix and sequential counter
		uniquePrefix := fmt.Sprintf("%s-%s%d", randomPrefix, basePrefix, permIndex)

		// Build enabled deps view for skip matching (name+flavor)
		var enabledForMatch []cloudinfo.AddonConfig
		for _, cfg := range permutationDeps {
			if cfg.Enabled != nil && *cfg.Enabled {
				enabledForMatch = append(enabledForMatch, cloudinfo.AddonConfig{
					OfferingName:   cfg.OfferingName,
					OfferingFlavor: cfg.OfferingFlavor,
				})
			}
		}

		// Skip if this enabled set matches a configured skip permutation
		if options != nil && options.shouldSkipPermutation(enabledForMatch) {
			if options.Logger != nil {
				options.Logger.ShortInfo(fmt.Sprintf("Skipping permutation: %s", testCaseName))
			}
			continue
		}

		testCase := AddonTestCase{
			Name:                         testCaseName,
			Prefix:                       uniquePrefix,
			Dependencies:                 permutationDeps,
			SkipInfrastructureDeployment: true, // Always skip infrastructure deployment
		}

		testCases = append(testCases, testCase)
		permIndex++ // Only increment for generated test cases
	}

	return testCases
}

// generatePermutationsWithFlavors creates enabled/disabled permutations and, for enabled deps,
// expands into all valid flavor combinations discovered from the catalog (programmatic, no hardcoding).
func (options *TestAddonOptions) generatePermutationsWithFlavors(dependenciesWithFlavors []DependencyWithFlavors) []AddonTestCase {
	var testCases []AddonTestCase

	// Generate permutations of enabled/disabled dependencies with flavor variations
	// For each of the 2^n - 1 enabled/disabled combinations (skips all-enabled),
	// expand into all flavor combinations for enabled dependencies.
	// Example: For 2 deps where dep1 has flavors [a,b] and dep2 has flavor [x]:
	// - Both disabled: 1 test case
	// - dep1[a] enabled, dep2 disabled: 1 test case
	// - dep1[b] enabled, dep2 disabled: 1 test case
	// - dep1 disabled, dep2[x] enabled: 1 test case
	// Total: 4 test cases (not 2^2=4 by coincidence, but from flavor expansion)
	numDeps := len(dependenciesWithFlavors)
	totalPermutations := 1 << numDeps // 2^n where n = number of dependencies

	// Group UI results per run
	randomPrefix := common.UniqueId(6)
	permIndex := 0
	basePrefix := options.shortenPrefix(options.Prefix)

	// Helper: ensure we always have at least one flavor to pick
	pickDefaultFlavor := func(dep DependencyWithFlavors) string {
		if len(dep.Flavors) > 0 {
			return dep.Flavors[0]
		}
		return "fully-configurable"
	}

	for mask := 0; mask < totalPermutations; mask++ {
		// Build enabled/disabled lists for this permutation
		var enabledDeps []DependencyWithFlavors
		var disabledDeps []DependencyWithFlavors
		var disabledNames []string

		for j, dep := range dependenciesWithFlavors {
			enabled := (mask & (1 << j)) != 0
			if enabled {
				enabledDeps = append(enabledDeps, dep)
			} else {
				disabledDeps = append(disabledDeps, dep)
				disabledNames = append(disabledNames, dep.Name)
			}
		}

		// Skip the all-enabled case (default configuration)
		if len(disabledNames) == 0 {
			continue
		}

		// Compute total flavor combinations for enabled deps (Cartesian product)
		totalComb := 1
		for _, dep := range enabledDeps {
			n := len(dep.Flavors)
			if n == 0 {
				n = 1 // treat as one implicit default flavor
			}
			totalComb *= n
		}

		// Enumerate all combinations of flavors for enabled deps
		for comb := 0; comb < totalComb; comb++ {
			var depsForCase []cloudinfo.AddonConfig

			// Mixed-radix expansion over enabled deps
			tmp := comb
			for _, dep := range enabledDeps {
				var flavor string
				if len(dep.Flavors) > 0 {
					idx := tmp % len(dep.Flavors)
					tmp /= len(dep.Flavors)
					flavor = dep.Flavors[idx]
				} else {
					flavor = pickDefaultFlavor(dep)
				}
				depsForCase = append(depsForCase, cloudinfo.AddonConfig{
					OfferingName:   dep.Name,
					OfferingFlavor: flavor,
					Enabled:        core.BoolPtr(true),
				})
			}

			// Add disabled deps with a deterministic flavor (first/default) so configs are complete
			for _, dep := range disabledDeps {
				depsForCase = append(depsForCase, cloudinfo.AddonConfig{
					OfferingName:   dep.Name,
					OfferingFlavor: pickDefaultFlavor(dep),
					Enabled:        core.BoolPtr(false),
				})
			}

			// Name: include disabled abbreviations; append concise flavor tags for enabled deps
			mainAbbrev := options.createInitialAbbreviation(options.AddonConfig.OfferingName)
			name := fmt.Sprintf("%s-%s-%d", randomPrefix, mainAbbrev, permIndex)
			if len(disabledNames) > 0 {
				disabledAbbrevs := options.abbreviateWithCollisionResolution(disabledNames)
				name = fmt.Sprintf("%s-%s-%d-disable-%s", randomPrefix, mainAbbrev, permIndex, strings.Join(disabledAbbrevs, "-"))
			}

			// Append flavor suffix only when at least one enabled dep has >1 flavors
			var flavorParts []string
			for _, dep := range enabledDeps {
				if len(dep.Flavors) > 1 { // only annotate multi-flavor deps
					// find chosen flavor in depsForCase
					for _, cfg := range depsForCase {
						if cfg.OfferingName == dep.Name && cfg.Enabled != nil && *cfg.Enabled {
							flavorParts = append(flavorParts, fmt.Sprintf("%s-%s",
								options.createInitialAbbreviation(dep.Name), options.abbreviateFlavor(cfg.OfferingFlavor)))
							break
						}
					}
				}
			}
			if len(flavorParts) > 0 {
				name = fmt.Sprintf("%s[%s]", name, strings.Join(flavorParts, ","))
			}

			uniquePrefix := fmt.Sprintf("%s-%s%d", randomPrefix, basePrefix, permIndex)
			// Build enabled deps view for skip matching (name+flavor)
			var enabledForMatch []cloudinfo.AddonConfig
			for _, cfg := range depsForCase {
				if cfg.Enabled != nil && *cfg.Enabled {
					enabledForMatch = append(enabledForMatch, cloudinfo.AddonConfig{
						OfferingName:   cfg.OfferingName,
						OfferingFlavor: cfg.OfferingFlavor,
					})
				}
			}

			// Skip if this enabled set matches a configured skip permutation
			if options != nil && options.shouldSkipPermutation(enabledForMatch) {
				if options.Logger != nil {
					options.Logger.ShortInfo(fmt.Sprintf("Skipping permutation: %s", name))
				}
				continue
			}

			testCases = append(testCases, AddonTestCase{
				Name:                         name,
				Prefix:                       uniquePrefix,
				Dependencies:                 depsForCase,
				SkipInfrastructureDeployment: true,
			})
			permIndex++
		}
	}

	return testCases
}

// shouldSkipPermutation returns true if the given set of enabled dependencies matches any skip entry.
// Matching rules:
// - Exact set of enabled OfferingName must match (order independent)
// - For each item in the skip entry, OfferingFlavor must match exactly when provided (non-empty). Empty flavor is wildcard
func (options *TestAddonOptions) shouldSkipPermutation(enabled []cloudinfo.AddonConfig) bool {
	if options == nil || len(options.SkipPermutations) == 0 {
		return false
	}

	// Build map of enabled name -> flavor
	enabledMap := make(map[string]string, len(enabled))
	for _, e := range enabled {
		enabledMap[e.OfferingName] = e.OfferingFlavor
	}

	for _, skipSet := range options.SkipPermutations {
		if len(skipSet) != len(enabled) {
			continue // size must match for exact set
		}

		match := true
		for _, s := range skipSet {
			chosenFlavor, ok := enabledMap[s.OfferingName]
			if !ok {
				match = false
				break
			}
			if s.OfferingFlavor != "" && s.OfferingFlavor != chosenFlavor {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// abbreviateFlavor creates a short abbreviation for flavor names
func (options *TestAddonOptions) abbreviateFlavor(flavor string) string {
	switch flavor {
	case "fully-configurable":
		return "fc"
	case "resource-group-only":
		return "rgo"
	case "resource-groups-with-account-settings":
		return "rgas"
	case "instance":
		return "inst"
	default:
		// Create abbreviation from first letters
		return options.createInitialAbbreviation(flavor)
	}
}

// isDefaultConfiguration checks if a permutation matches the default "on by default" configuration
func (options *TestAddonOptions) isDefaultConfiguration(permutation []cloudinfo.AddonConfig, originalDependencies []cloudinfo.AddonConfig) bool {
	if len(permutation) != len(originalDependencies) {
		return false
	}

	for i, dep := range permutation {
		originalDep := originalDependencies[i]

		// Check if this dependency's enabled state matches its OnByDefault value
		expectedEnabled := false
		if originalDep.OnByDefault != nil {
			expectedEnabled = *originalDep.OnByDefault
		}

		actualEnabled := false
		if dep.Enabled != nil {
			actualEnabled = *dep.Enabled
		}

		if expectedEnabled != actualEnabled {
			return false
		}
	}

	return true
}

// shortenPrefix truncates a prefix to fit within the 8-character limit
// while preserving enough information to identify the test case
func (options *TestAddonOptions) shortenPrefix(prefix string) string {
	const maxPrefixLength = 8

	// If already short enough, return as is
	if len(prefix) <= maxPrefixLength-2 { // Reserve 2 characters for numbering (0-99)
		return prefix
	}

	// Truncate to leave room for numbering
	maxBaseLength := maxPrefixLength - 2
	return prefix[:maxBaseLength]
}

// joinNames joins a slice of strings with a separator, handling empty slices
func (options *TestAddonOptions) joinNames(names []string, separator string) string {
	if len(names) == 0 {
		return ""
	}
	if len(names) == 1 {
		return names[0]
	}

	result := names[0]
	for i := 1; i < len(names); i++ {
		result += separator + names[i]
	}
	return result
}

// abbreviateWithCollisionResolution creates abbreviated names for components, resolving collisions
// by progressively adding characters until all names are unique within the given list
func (options *TestAddonOptions) abbreviateWithCollisionResolution(names []string) []string {
	if len(names) == 0 {
		return names
	}

	abbreviated := make([]string, len(names))

	// First pass: create initial abbreviations (first character of each dash-separated part)
	for i, name := range names {
		abbreviated[i] = options.createInitialAbbreviation(name)
	}

	// Resolve collisions by progressively adding characters
	return options.resolveCollisions(names, abbreviated)
}

// createInitialAbbreviation creates first-character abbreviation for a component name
func (options *TestAddonOptions) createInitialAbbreviation(name string) string {
	if name == "" {
		return ""
	}

	// Handle common prefixes first and replace with abbreviated versions
	shortName := name
	if strings.HasPrefix(shortName, "deploy-arch-ibm-") {
		shortName = strings.TrimPrefix(shortName, "deploy-arch-ibm-")
		shortName = "dai-" + shortName
	} else if strings.HasPrefix(shortName, "deploy-arch-") {
		shortName = strings.TrimPrefix(shortName, "deploy-arch-")
		shortName = "da-" + shortName
	}

	// Split by dashes and take first character of each part
	parts := strings.Split(shortName, "-")
	var result strings.Builder

	for i, part := range parts {
		if part == "" {
			continue
		}
		if i > 0 {
			result.WriteString("-")
		}
		// Keep numbers and preserve certain keywords as-is
		if options.isNumericOrKeyword(part) {
			result.WriteString(part)
		} else {
			result.WriteByte(part[0])
		}
	}

	return result.String()
}

// isNumericOrKeyword checks if a part should be kept as-is
func (options *TestAddonOptions) isNumericOrKeyword(part string) bool {
	// Keep parts that start with numbers or contain digits (like v2, v10, etc.)
	if len(part) > 0 {
		for _, char := range part {
			if char >= '0' && char <= '9' {
				return true
			}
		}
	}

	// Keep important keywords and our abbreviated prefixes
	keywords := []string{"disable", "enable", "test", "basic", "advanced", "dai", "da"}
	for _, keyword := range keywords {
		if part == keyword {
			return true
		}
	}

	return false
}

// detectCircularDependencies analyzes configs in awaiting_prerequisite state to find circular dependency chains
func (options *TestAddonOptions) detectCircularDependencies(awaitingConfigs []ConfigDependencyInfo) []string {
	if len(awaitingConfigs) == 0 {
		return nil
	}

	// Build a map of config ID to config info for quick lookup
	configMap := make(map[string]ConfigDependencyInfo)
	for _, config := range awaitingConfigs {
		configMap[config.ID] = config
	}

	// Build detailed dependency information for enhanced error reporting
	detailedDependencies := make([]DetailedDependencyInfo, 0)

	// Build dependency graph: config ID -> list of config IDs it depends on
	dependencyGraph := make(map[string][]string)
	for _, config := range awaitingConfigs {
		dependencies := make([]string, 0)
		for _, ref := range config.InputReferences {
			// Parse reference format: ref:/configs/{config_id}/outputs/{output_name}
			if referencedConfigID := options.parseConfigIDFromReference(ref); referencedConfigID != "" {
				// Only consider dependencies on other awaiting configs (potential circular deps)
				if referencedConfig, exists := configMap[referencedConfigID]; exists {
					dependencies = append(dependencies, referencedConfigID)

					// Extract detailed dependency information with actual field names
					inputName := options.findInputFieldNameFromReference(config, ref)

					// Parse the reference to understand what we're pointing to
					refDetails := options.parseReferenceDetails(ref)
					var referencedField string
					var referencedType string

					if refDetails.IsValid {
						// Use the dynamically parsed details
						referencedField = refDetails.FieldName
						referencedType = refDetails.ReferenceType
					} else {
						// Fallback for backward compatibility
						referencedField = options.parseOutputNameFromReference(ref)
						referencedType = "outputs" // assume outputs for backward compatibility
					}

					detailedDep := DetailedDependencyInfo{
						ConfigID:             config.ID,
						ConfigName:           config.Name,
						InputName:            inputName,
						ReferencedConfigID:   referencedConfigID,
						ReferencedConfigName: referencedConfig.Name,
						ReferencedOutput:     referencedField,
						ReferencedType:       referencedType,
						FullReference:        ref,
					}
					detailedDependencies = append(detailedDependencies, detailedDep)
				}
			}
		}
		dependencyGraph[config.ID] = dependencies
	}

	// Detect cycles using DFS (Depth First Search)
	var cycles []string
	visited := make(map[string]bool)
	recursionStack := make(map[string]bool)

	for configID := range dependencyGraph {
		if !visited[configID] {
			if cycle := options.findCycleDFS(configID, dependencyGraph, configMap, detailedDependencies, visited, recursionStack, []string{}); cycle != "" {
				cycles = append(cycles, cycle)
			}
		}
	}

	return cycles
}

// parseConfigIDFromReference extracts the config ID from a reference string
// Format: ref:/configs/{config_id}/outputs/{output_name}
func (options *TestAddonOptions) parseConfigIDFromReference(reference string) string {
	// Expected format: ref:/configs/{config_id}/outputs/{output_name}
	if !strings.HasPrefix(reference, "ref:/configs/") {
		return ""
	}

	// Remove "ref:/configs/" prefix
	remaining := strings.TrimPrefix(reference, "ref:/configs/")

	// Find the next "/" to separate config ID from the rest
	parts := strings.Split(remaining, "/")
	if len(parts) >= 1 {
		return parts[0]
	}

	return ""
}

// findInputFieldNameFromReference finds the input field name that contains the given reference
func (options *TestAddonOptions) findInputFieldNameFromReference(config ConfigDependencyInfo, reference string) string {
	// Look through the field mappings to find which field has this reference
	for fieldName, fieldRef := range config.InputFieldReferences {
		if fieldRef == reference {
			return fieldName
		}
	}

	// If we can't find the field name, fall back to unknown with reference info for debugging
	return "unknown_input"
}

// ReferenceDetails contains the parsed components of a reference string
type ReferenceDetails struct {
	ConfigID      string // The config ID being referenced
	ReferenceType string // "inputs", "outputs", or other type
	FieldName     string // The field name being referenced
	IsValid       bool   // Whether the reference was successfully parsed
}

// parseReferenceDetails dynamically parses any reference format
func (options *TestAddonOptions) parseReferenceDetails(reference string) ReferenceDetails {
	// Expected format: ref:/configs/{config_id}/{type}/{field_name}
	// Where type can be "inputs", "outputs", or any future type
	result := ReferenceDetails{IsValid: false}

	if !strings.HasPrefix(reference, "ref:/configs/") {
		return result
	}

	// Extract components from the reference
	parts := strings.Split(reference, "/")
	if len(parts) >= 5 && parts[0] == "ref:" && parts[1] == "configs" {
		result.ConfigID = parts[2]
		result.ReferenceType = parts[3]
		result.FieldName = parts[4]
		result.IsValid = true
	}

	return result
}

// parseOutputNameFromReference extracts just the output name from a reference string (legacy compatibility)
func (options *TestAddonOptions) parseOutputNameFromReference(reference string) string {
	details := options.parseReferenceDetails(reference)
	if !details.IsValid {
		return "unknown_output"
	}

	// For backward compatibility, return the field name regardless of whether it's input or output
	return details.FieldName
}

// findCycleDFS performs depth-first search to detect cycles in the dependency graph
func (options *TestAddonOptions) findCycleDFS(configID string, graph map[string][]string, configMap map[string]ConfigDependencyInfo, detailedDeps []DetailedDependencyInfo, visited map[string]bool, recursionStack map[string]bool, currentPath []string) string {
	visited[configID] = true
	recursionStack[configID] = true
	currentPath = append(currentPath, configID)

	// Check all dependencies of the current config
	for _, dependencyID := range graph[configID] {
		if !visited[dependencyID] {
			if cycle := options.findCycleDFS(dependencyID, graph, configMap, detailedDeps, visited, recursionStack, currentPath); cycle != "" {
				return cycle
			}
		} else if recursionStack[dependencyID] {
			// Found a cycle! Build the cycle description
			return options.buildCycleDescription(currentPath, dependencyID, configMap, detailedDeps)
		}
	}

	recursionStack[configID] = false
	return ""
}

// buildCycleDescription creates a human-readable description of the circular dependency with detailed input/output information
func (options *TestAddonOptions) buildCycleDescription(path []string, cycleStart string, configMap map[string]ConfigDependencyInfo, detailedDeps []DetailedDependencyInfo) string {
	// Find where the cycle starts in the path
	cycleStartIndex := -1
	for i, id := range path {
		if id == cycleStart {
			cycleStartIndex = i
			break
		}
	}

	if cycleStartIndex == -1 {
		return fmt.Sprintf("Circular dependency detected involving config %s", cycleStart)
	}

	// Build a map of dependencies for quick lookup
	depMap := make(map[string]map[string]DetailedDependencyInfo)
	for _, dep := range detailedDeps {
		if depMap[dep.ConfigID] == nil {
			depMap[dep.ConfigID] = make(map[string]DetailedDependencyInfo)
		}
		depMap[dep.ConfigID][dep.ReferencedConfigID] = dep
	}

	// Build enhanced cycle description with input/output details
	var cycleDetails []string
	cycleConfigs := path[cycleStartIndex:]
	cycleConfigs = append(cycleConfigs, cycleStart) // Add the starting config to complete the cycle

	for i := 0; i < len(cycleConfigs)-1; i++ {
		currentConfigID := cycleConfigs[i]
		nextConfigID := cycleConfigs[i+1]

		currentConfig, exists := configMap[currentConfigID]
		if !exists {
			continue
		}

		// Find the specific dependency causing this link in the cycle
		if depInfo, found := depMap[currentConfigID][nextConfigID]; found {
			// Use actual field names if available, otherwise provide fallback message
			if depInfo.InputName != "unknown_input" && depInfo.ReferencedOutput != "unknown_output" {
				// Build dynamic description based on what type of reference this is
				var referenceDescription string
				if depInfo.ReferencedType == "outputs" {
					referenceDescription = fmt.Sprintf("%s.output: %s", depInfo.ReferencedConfigName, depInfo.ReferencedOutput)
				} else if depInfo.ReferencedType == "inputs" {
					referenceDescription = fmt.Sprintf("%s.input: %s", depInfo.ReferencedConfigName, depInfo.ReferencedOutput)
				} else if depInfo.ReferencedType == "" {
					// Backward compatibility: assume output if type is not set
					referenceDescription = fmt.Sprintf("%s.output: %s", depInfo.ReferencedConfigName, depInfo.ReferencedOutput)
				} else {
					// Handle any future reference types generically
					referenceDescription = fmt.Sprintf("%s.%s: %s", depInfo.ReferencedConfigName, depInfo.ReferencedType, depInfo.ReferencedOutput)
				}

				cycleDetails = append(cycleDetails, fmt.Sprintf("%s (input: %s needs %s)",
					currentConfig.Name,
					depInfo.InputName,
					referenceDescription))
			} else {
				// Fallback when field parsing fails but circular dependency is detected
				cycleDetails = append(cycleDetails, fmt.Sprintf("%s (circular dependency detected but field details unavailable - check configuration references)",
					currentConfig.Name))
			}
		} else {
			cycleDetails = append(cycleDetails, currentConfig.Name)
		}
	}

	// Create the enhanced error message with resolution guidance
	cycleChain := strings.Join(cycleDetails, " → ")

	resolutionGuidance := "\n\n💡 RESOLUTION OPTIONS:\n" +
		"• Use existing resources instead of creating new ones\n" +
		"• Restructure deployment order by splitting dependencies\n" +
		"• Consider using data sources or external references"

	return fmt.Sprintf("🔍 CIRCULAR DEPENDENCY DETECTED: %s%s", cycleChain, resolutionGuidance)
}

// findUnresolvedReferences identifies input references that point to non-existent configs or outputs
func (options *TestAddonOptions) findUnresolvedReferences(awaitingConfigs []ConfigDependencyInfo, allConfigs []projectv1.ProjectConfigSummary) []string {
	// Build a map of existing config IDs for quick lookup
	existingConfigIDs := make(map[string]bool)
	for _, config := range allConfigs {
		if config.ID != nil {
			existingConfigIDs[*config.ID] = true
		}
	}

	var unresolvedRefs []string

	for _, config := range awaitingConfigs {
		for _, ref := range config.InputReferences {
			referencedConfigID := options.parseConfigIDFromReference(ref)
			if referencedConfigID != "" {
				// Check if the referenced config exists
				if !existingConfigIDs[referencedConfigID] {
					unresolvedRefs = append(unresolvedRefs, fmt.Sprintf("%s → references non-existent config %s", config.Name, referencedConfigID))
				}
				// Note: We could also check if the referenced output actually exists on the target config,
				// but that would require querying the catalog for each config's outputs, which might be expensive.
				// For now, we just verify the config exists.
			}
		}
	}

	return unresolvedRefs
}

// resolveCollisions progressively adds characters to resolve naming conflicts
func (options *TestAddonOptions) resolveCollisions(originalNames []string, abbreviated []string) []string {
	result := make([]string, len(abbreviated))
	copy(result, abbreviated)

	// Find collisions and resolve them
	for {
		collisions := options.findCollisions(result)
		if len(collisions) == 0 {
			break
		}

		// For each collision group, extend the later abbreviations (keep first as-is)
		for _, indices := range collisions {
			if len(indices) < 2 {
				continue
			}

			// Extend all but the first abbreviation in the collision group
			for i := 1; i < len(indices); i++ {
				idx := indices[i]
				result[idx] = options.extendAbbreviation(originalNames[idx], result[idx])
			}
		}
	}

	return result
}

// findCollisions returns a map of collision groups (abbreviated name -> list of indices)
func (options *TestAddonOptions) findCollisions(abbreviated []string) map[string][]int {
	collisions := make(map[string][]int)

	for i, abbrev := range abbreviated {
		collisions[abbrev] = append(collisions[abbrev], i)
	}

	// Remove entries that don't have collisions
	for abbrev, indices := range collisions {
		if len(indices) <= 1 {
			delete(collisions, abbrev)
		}
	}

	return collisions
}

// extendAbbreviation adds one more character to an abbreviation
func (options *TestAddonOptions) extendAbbreviation(originalName, currentAbbrev string) string {
	// Find the last part that was abbreviated and extend it
	originalParts := strings.Split(originalName, "-")
	abbrevParts := strings.Split(currentAbbrev, "-")

	if len(abbrevParts) == 0 {
		return currentAbbrev
	}

	// Work backwards to find the first abbreviated part we can extend
	for i := len(abbrevParts) - 1; i >= 0; i-- {
		if i < len(originalParts) {
			originalPart := originalParts[i]
			abbrevPart := abbrevParts[i]

			// Skip if this part is a keyword or number (shouldn't be extended)
			if options.isNumericOrKeyword(originalPart) {
				continue
			}

			// Skip if already fully expanded
			if abbrevPart == originalPart {
				continue
			}

			// Extend this part by one character
			if len(abbrevPart) < len(originalPart) {
				abbrevParts[i] = originalPart[:len(abbrevPart)+1]
				return strings.Join(abbrevParts, "-")
			}
		}
	}

	// If we can't extend any part, just append a character from the original
	return currentAbbrev + "x"
}

// displaySingleTestStrictModeWarnings displays a simple warning summary for single tests
// when StrictMode=false and warnings occurred during test execution
func (options *TestAddonOptions) displaySingleTestStrictModeWarnings() {
	if options.lastValidationResult == nil || len(options.lastValidationResult.Warnings) == 0 {
		return
	}

	// Count different types of warnings
	circularDependencyCount := 0
	forceEnabledDependencyCount := 0
	var circularWarnings []string
	var forceEnabledWarnings []string

	for _, warning := range options.lastValidationResult.Warnings {
		if strings.Contains(warning, "Circular dependency") {
			circularDependencyCount++
			// Extract clean circular dependency info
			cleanWarning := strings.TrimPrefix(warning, "Circular dependency: ")
			circularWarnings = append(circularWarnings, cleanWarning)
		} else if strings.Contains(warning, "force-enabled despite being disabled") {
			forceEnabledDependencyCount++
			forceEnabledWarnings = append(forceEnabledWarnings, warning)
		}
	}

	// Only display if we have warnings to show
	if circularDependencyCount > 0 || forceEnabledDependencyCount > 0 {
		options.Logger.ProgressInfo("\n⚠️ STRICT MODE DISABLED - The following would have failed in strict mode:")

		// Display circular dependency warnings
		if circularDependencyCount > 0 {
			for _, warning := range circularWarnings {
				options.Logger.ProgressInfo(fmt.Sprintf("• Circular dependency detected: %s", warning))
			}
		}

		// Display force-enabled dependency warnings
		if forceEnabledDependencyCount > 0 {
			for _, warning := range forceEnabledWarnings {
				options.Logger.ProgressInfo(fmt.Sprintf("• %s", warning))
			}
		}

		options.Logger.ProgressInfo("")
	}
}
