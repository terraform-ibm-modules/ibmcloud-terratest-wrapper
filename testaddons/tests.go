package testaddons

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/IBM/go-sdk-core/v5/core"
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
					// Fallback to testing.T if logger is not available
					options.Testing.Logf("ERROR: %s", panicMsg)
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
		return fmt.Errorf("test setup has failed:%w", setupErr)
	}

	// Deploy Addon to Project
	if options.QuietMode {
		options.Logger.ProgressStage("Deploying Configurations to Project")
	}
	options.Logger.ShortInfo("Deploying the addon to project")
	deployedConfigs, err := options.CloudInfoService.DeployAddonToProject(&options.AddonConfig, options.currentProjectConfig)

	if err != nil {
		options.Logger.ShortError(fmt.Sprintf("Error deploying the addon to project: %v", err))
		options.Logger.MarkFailed()
		options.Logger.FlushOnFailure()
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
			return fmt.Errorf("error retrieving config %s: %w", config.Name, err)
		}
		if prjCfg == nil {
			options.Logger.ShortError(fmt.Sprintf("Retrieved config %s is nil", config.Name))
			options.Logger.MarkFailed()
			options.Logger.FlushOnFailure()
			options.Testing.Fail()
			return fmt.Errorf("retrieved config %s is nil", config.Name)
		}
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
		options.Logger.ShortWarn("No configuration found in ready_to_validate state (will check dependencies first)")
		// Store this issue for later evaluation after dependency validation
		waitingInputIssues = append(waitingInputIssues, "No configuration is in ready_to_validate state")
	}

	// Check if the configuration is in a valid state
	options.Logger.ShortInfo(fmt.Sprintf("Checked if the configuration is deployable %s", common.ColorizeString(common.Colors.Green, "pass ✔")))

	// Now run dependency validation before evaluating the collected validation issues
	if !options.SkipDependencyValidation {
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
			return fmt.Errorf("dependency validation failed: AddonConfig.CatalogID is empty - this may indicate a race condition in parallel test execution or incomplete offering setup. VersionLocator='%s', OfferingName='%s'", rootVersionLocator, options.AddonConfig.OfferingName)
		}
		if rootOfferingID == "" {
			return fmt.Errorf("dependency validation failed: AddonConfig.OfferingID is empty - this may indicate a race condition in parallel test execution or incomplete offering setup. VersionLocator='%s', OfferingName='%s', CatalogID='%s'", rootVersionLocator, options.AddonConfig.OfferingName, rootCatalogID)
		}
		if rootVersionLocator == "" {
			return fmt.Errorf("dependency validation failed: AddonConfig.VersionLocator is empty - this may indicate incomplete offering setup. OfferingName='%s', CatalogID='%s', OfferingID='%s'", options.AddonConfig.OfferingName, rootCatalogID, rootOfferingID)
		}

		options.Logger.ShortInfo(fmt.Sprintf("Dependency validation starting with: catalogID='%s', offeringID='%s', versionLocator='%s', flavor='%s'", rootCatalogID, rootOfferingID, rootVersionLocator, options.AddonConfig.OfferingFlavor))

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

		if options.QuietMode {
			options.Logger.ProgressStage("Analyzing deployed configurations")
		}
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

		if options.QuietMode {
			options.Logger.ProgressStage("Validating dependency compliance")
		}
		// First validate what is actually deployed to get the validation results
		validationResult := options.validateDependencies(graph, expectedDeployedList, actuallyDeployedResult.ActuallyDeployedList)

		// Store the validation result for error reporting
		options.lastValidationResult = &validationResult

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

			return fmt.Errorf("%s", errorMsg)
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

		// Use the new CriticalError helper method for consistent error handling
		errorMessage := fmt.Sprintf("Missing required inputs - %s", strings.Join(inputValidationIssues, "; "))
		options.Logger.CriticalError(errorMessage)
		options.Logger.ShortCustom("Cannot proceed with deployment - required inputs must be provided", common.Colors.Red)
		options.Logger.ShortCustom("Note: Missing inputs may be caused by missing dependencies shown above", common.Colors.Red)

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

		// Create a specific, actionable error message
		var errorMsg string
		if len(missingInputsDetails) > 0 {
			errorMsg = fmt.Sprintf("configurations waiting on missing inputs: %s", strings.Join(missingInputsDetails, ", "))
		} else if len(configsWithIssues) > 0 {
			errorMsg = fmt.Sprintf("configurations in awaiting_input state: %s", strings.Join(configsWithIssues, ", "))
		} else {
			errorMsg = "configurations waiting on inputs - check debug output above for details"
		}

		// Use the new CriticalError helper method for consistent error handling
		options.Logger.CriticalError(fmt.Sprintf("Found %s", errorMsg))
		options.Logger.ShortCustom("Note: Missing inputs may be caused by missing dependencies shown above", common.Colors.Red)

		options.Testing.Fail()
		return fmt.Errorf("found %s", errorMsg)
	}

	options.Logger.ShortInfo("Dependency validation completed successfully")

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

	return nil
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
			t.Parallel()

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

			// Show test start progress in quiet mode
			if matrix.BaseOptions.QuietMode {
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

			// Ensure logger is initialized before using it
			if testOptions.Logger == nil {
				testOptions.Logger = common.CreateSmartAutoBufferingLogger(parentTestName, false)
			}

			// Set quiet mode on the logger
			if testOptions.QuietMode {
				testOptions.Logger.SetQuietMode(true)
				// In quiet mode, don't show individual test start messages
			} else {
				testOptions.Logger.SetQuietMode(false)
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
			err := testOptions.RunAddonTest()

			// If a panic occurred, use the panic error instead
			if panicOccurred {
				err = testErr
			}

			// Thread-safe result collection for parallel subtests
			// Mutex protection is required because multiple parallel subtests may
			// simultaneously append to the shared PermutationTestReport.Results slice.
			// Go's slice operations are not thread-safe for concurrent writes.
			if matrix.BaseOptions.CollectResults && matrix.BaseOptions.PermutationTestReport != nil {
				testResult := testOptions.collectTestResult(tc.Name, tc.Prefix, testOptions.AddonConfig, err)

				// CRITICAL: Protect concurrent access to shared report data
				resultMutex.Lock()
				matrix.BaseOptions.PermutationTestReport.Results = append(matrix.BaseOptions.PermutationTestReport.Results, testResult)
				if testResult.Passed {
					matrix.BaseOptions.PermutationTestReport.PassedTests++
				} else {
					matrix.BaseOptions.PermutationTestReport.FailedTests++
				}
				resultMutex.Unlock()
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

	// Enable quiet mode by default for permutation tests to reduce log noise
	if !options.QuietMode {
		options.QuietMode = true
	}

	// Step 1: Discover dependencies from catalog
	dependencies, err := options.discoverDependencies()
	if err != nil {
		return fmt.Errorf("failed to discover dependencies: %w", err)
	}

	if len(dependencies) == 0 {
		options.Testing.Skip("No dependencies found to test permutations")
		return nil
	}

	// Step 2: Generate all permutations of dependencies
	testCases := options.generatePermutations(dependencies)

	if len(testCases) == 0 {
		options.Testing.Skip("No permutations generated (all would be default configuration)")
		return nil
	}

	// Step 3: Initialize result collection and logging
	if options.Logger == nil {
		options.Logger = common.CreateSmartAutoBufferingLogger("TestDependencyPermutations", false)
	}

	if options.QuietMode {
		options.Logger.SetQuietMode(true)
	} else {
		options.Logger.SetQuietMode(false)
	}

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
	options.Logger.ProgressStage(fmt.Sprintf("Running %d dependency permutation tests for %s (quiet mode - minimal output)...", len(testCases), options.AddonConfig.OfferingName))

	// Step 4: Execute all permutations in parallel using matrix test infrastructure
	matrix := AddonTestMatrix{
		TestCases:   testCases,
		BaseOptions: options,
		BaseSetupFunc: func(baseOptions *TestAddonOptions, testCase AddonTestCase) *TestAddonOptions {
			// Clone base options for each test case
			testOptions := baseOptions.copy()
			testOptions.Prefix = testCase.Prefix
			testOptions.TestCaseName = testCase.Name
			testOptions.SkipInfrastructureDeployment = testCase.SkipInfrastructureDeployment
			// Inherit quiet mode from base options
			testOptions.QuietMode = baseOptions.QuietMode
			testOptions.VerboseOnFailure = baseOptions.VerboseOnFailure
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
	options.RunAddonTestMatrix(matrix)

	return nil
}

// discoverDependencies automatically discovers direct dependencies from the catalog
func (options *TestAddonOptions) discoverDependencies() ([]cloudinfo.AddonConfig, error) {
	// Use existing CloudInfoService if available, otherwise create a new one
	var cloudInfoService cloudinfo.CloudInfoServiceI
	if options.CloudInfoService != nil {
		cloudInfoService = options.CloudInfoService
	} else {
		// Create a temporary CloudInfoService for catalog operations
		service, err := cloudinfo.NewCloudInfoServiceFromEnv("TF_VAR_ibmcloud_api_key", cloudinfo.CloudInfoServiceOptions{
			Logger: options.Logger,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create CloudInfoService: %w", err)
		}
		cloudInfoService = service
	}

	// Create a temporary catalog for dependency discovery
	catalogName := fmt.Sprintf("temp-catalog-%s", common.UniqueId())
	catalog, err := cloudInfoService.CreateCatalog(catalogName)
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary catalog: %w", err)
	}

	// Cleanup catalog after discovery
	defer func() {
		if catalog != nil && catalog.ID != nil {
			_ = cloudInfoService.DeleteCatalog(*catalog.ID)
		}
	}()

	// Import the addon offering to get its dependencies
	offering, err := cloudInfoService.ImportOfferingWithValidation(
		*catalog.ID,
		options.AddonConfig.OfferingName,
		options.AddonConfig.OfferingFlavor,
		"1.0.0", // Use a default version for discovery
		cloudinfo.InstallKindTerraform,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to import offering for dependency discovery: %w", err)
	}

	// Get component references to discover dependencies
	if len(offering.Kinds) == 0 || len(offering.Kinds[0].Versions) == 0 {
		return nil, fmt.Errorf("no versions found in imported offering")
	}

	versionLocator := *offering.Kinds[0].Versions[0].VersionLocator
	componentsReferences, err := cloudInfoService.GetComponentReferences(versionLocator)
	if err != nil {
		return nil, fmt.Errorf("failed to get component references: %w", err)
	}

	// Convert component references to AddonConfig list
	var dependencies []cloudinfo.AddonConfig

	// Process required dependencies first
	for _, component := range componentsReferences.Required.OfferingReferences {
		if component.OfferingReference.DefaultFlavor == "" ||
			component.OfferingReference.DefaultFlavor == component.OfferingReference.Flavor.Name {
			dep := cloudinfo.AddonConfig{
				OfferingName:        component.OfferingReference.Name,
				OfferingFlavor:      component.OfferingReference.Flavor.Name,
				OfferingLabel:       component.OfferingReference.Label,
				CatalogID:           component.OfferingReference.CatalogID,
				OfferingID:          component.OfferingReference.ID,
				VersionLocator:      component.OfferingReference.VersionLocator,
				ResolvedVersion:     component.OfferingReference.Version,
				Enabled:             core.BoolPtr(true), // Required dependencies are always enabled
				Prefix:              options.Prefix,
				Inputs:              make(map[string]interface{}),
				OfferingInstallKind: cloudinfo.InstallKindTerraform,
				Dependencies:        []cloudinfo.AddonConfig{},
			}
			dependencies = append(dependencies, dep)
		}
	}

	// Process optional dependencies
	for _, component := range componentsReferences.Optional.OfferingReferences {
		if component.OfferingReference.DefaultFlavor == "" ||
			component.OfferingReference.DefaultFlavor == component.OfferingReference.Flavor.Name {
			dep := cloudinfo.AddonConfig{
				OfferingName:        component.OfferingReference.Name,
				OfferingFlavor:      component.OfferingReference.Flavor.Name,
				OfferingLabel:       component.OfferingReference.Label,
				CatalogID:           component.OfferingReference.CatalogID,
				OfferingID:          component.OfferingReference.ID,
				VersionLocator:      component.OfferingReference.VersionLocator,
				ResolvedVersion:     component.OfferingReference.Version,
				Enabled:             core.BoolPtr(component.OfferingReference.OnByDefault),
				OnByDefault:         core.BoolPtr(component.OfferingReference.OnByDefault),
				Prefix:              options.Prefix,
				Inputs:              make(map[string]interface{}),
				OfferingInstallKind: cloudinfo.InstallKindTerraform,
				Dependencies:        []cloudinfo.AddonConfig{},
			}
			dependencies = append(dependencies, dep)
		}
	}

	return dependencies, nil
}

// generatePermutations creates all enabled/disabled combinations of dependencies
func (options *TestAddonOptions) generatePermutations(dependencies []cloudinfo.AddonConfig) []AddonTestCase {
	var testCases []AddonTestCase

	// Generate all 2^n permutations of dependencies (root addon is always present)
	numDeps := len(dependencies)
	totalPermutations := 1 << numDeps // 2^n where n = number of dependencies

	// Generate a random prefix once per test run for UI grouping
	randomPrefix := common.UniqueId(6)

	// Sequential counter for unique prefix generation
	permIndex := 0
	basePrefix := options.shortenPrefix(options.Prefix)

	for i := 0; i < totalPermutations; i++ {
		// Create a copy of dependencies for this permutation
		permutationDeps := make([]cloudinfo.AddonConfig, len(dependencies))
		copy(permutationDeps, dependencies)

		// Generate permutation name based on disabled dependencies
		var disabledNames []string

		// Set enabled/disabled based on bit pattern
		for j := 0; j < numDeps; j++ {
			if (i & (1 << j)) == 0 {
				// Bit is 0, disable this dependency
				permutationDeps[j].Enabled = core.BoolPtr(false)
				disabledNames = append(disabledNames, permutationDeps[j].OfferingName)
			} else {
				// Bit is 1, enable this dependency
				permutationDeps[j].Enabled = core.BoolPtr(true)
			}
		}

		// Skip the "on by default" case (default configuration)
		// This excludes the combination where all dependencies match their OnByDefault values
		if options.isDefaultConfiguration(permutationDeps, dependencies) {
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
