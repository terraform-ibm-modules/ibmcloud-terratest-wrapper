package testaddons

import (
	"fmt"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"

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

	// set offering details
	SetOfferingDetails(options)
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
			waitingOnInputs = append(waitingOnInputs, *currentConfigDetails.Definition.(*projectv1.ProjectConfigDefinitionResponse).Name)
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
		version := strings.Split(*currentConfigDetails.Definition.(*projectv1.ProjectConfigDefinitionResponse).LocatorID, ".")[1]
		if version == options.AddonConfig.VersionID {
			targetAddon = options.AddonConfig
		} else {
			for i, dependency := range options.AddonConfig.Dependencies {
				if version == dependency.VersionID {
					targetAddon = options.AddonConfig.Dependencies[i]
					break
				}
			}
		}
		if targetAddon.VersionID == "" {
			options.Logger.ShortError(fmt.Sprintf("Error resolving addon: %v", *currentConfigDetails.ID))
			options.Testing.Failed()
		}

		// check if any required inputs are not set
		allInputsPresent := true
		for _, input := range targetAddon.OfferingInputs {
			if !input.Required {
				continue
			}
			options.Logger.ShortInfo(fmt.Sprintf("Required Input: %v ", input.Key))
			if input.Key == "ibmcloud_api_key" {
				continue
			}

			value, exists := currentConfigDetails.Definition.(*projectv1.ProjectConfigDefinitionResponse).Inputs[input.Key]
			if !exists || fmt.Sprintf("%v", value) == "" {
				if input.DefaultValue == nil || fmt.Sprintf("%v", input.DefaultValue) == "" || fmt.Sprintf("%v", input.DefaultValue) == "__NOT_SET__" {
					options.Logger.ShortError(fmt.Sprintf("Missing or empty required input: %s\n", input.Key))
					allInputsPresent = false
				}
			}

		}
		if allInputsPresent {
			options.Logger.ShortInfo(fmt.Sprintf("All required inputs set for addon: %s\n", *currentConfigDetails.ID))
		} else {
			options.Logger.ShortError(fmt.Sprintf("Error, some required inputs are missing or empty for addon: %s\n", *currentConfigDetails.ID))
			options.Testing.Fail()
		}
	}

	if !options.SkipRefValidation && len(failedRefs) > 0 {
		options.Logger.ShortError("Failed to resolve references:")
		for _, ref := range failedRefs {
			options.Logger.ShortError(fmt.Sprintf("  %s", ref))
		}
		options.Testing.Failed()
		return fmt.Errorf("failed to resolve references")
	}

	if !options.SkipRefValidation {
		options.Logger.ShortInfo(fmt.Sprintf("  All references resolved successfully %s", common.ColorizeString(common.Colors.Green, "pass ✔")))
	} else {
		options.Logger.ShortInfo("Reference validation skipped")
	}

	if assert.Equal(options.Testing, 0, len(waitingOnInputs), "Found configurations waiting on inputs") {
		options.Logger.ShortInfo("No configurations waiting on inputs")
	} else {
		options.Logger.ShortError("Found configurations waiting on inputs")
		for _, config := range waitingOnInputs {
			options.Logger.ShortError(fmt.Sprintf("  %s", config))
		}
		options.Testing.Fail()
		return fmt.Errorf("found configurations waiting on inputs project not correctly configured")
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
			return fmt.Errorf("expected configurations and actual configurations are not same")
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

	if options.ProjectName != "" && options.TestCaseName == "" {
		// For single tests, include project name in prefix
		options.Logger.SetPrefix(fmt.Sprintf("ADDON - %s", options.ProjectName))
	} else if options.TestCaseName != "" {
		// For matrix tests, include test case name in prefix
		options.Logger.SetPrefix(fmt.Sprintf("ADDON - %s", options.TestCaseName))
	} else {
		// For tests without project name or test case name, use simple prefix
		options.Logger.SetPrefix("ADDON")
	}

	options.Logger.EnableDateTime(false)

	// change relative paths of configuration files to full path based on git root
	repoRoot, repoErr := common.GitRootPath(".")
	if repoErr != nil {
		repoRoot = "."
	}

	options.Logger.ShortInfo("Checking for local changes in the repository")

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

	// Convert repository URL to HTTPS format for catalog import
	if strings.HasPrefix(repo, "git@") {
		// Convert SSH format: git@github.com:username/repo.git → https://github.com/username/repo
		repo = strings.Replace(repo, ":", "/", 1)
		repo = strings.Replace(repo, "git@", "https://", 1)
		repo = strings.TrimSuffix(repo, ".git")
	} else if strings.HasPrefix(repo, "git://") {
		// Convert Git protocol: git://github.com/username/repo.git → https://github.com/username/repo
		repo = strings.Replace(repo, "git://", "https://", 1)
		repo = strings.TrimSuffix(repo, ".git")
	} else if strings.HasPrefix(repo, "https://") {
		// HTTPS format - just trim .git suffix if present
		repo = strings.TrimSuffix(repo, ".git")
	}

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
		} else {
			// Create new catalog only if no existing catalog is available
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
	if *options.SharedCatalog && options.offering != nil {
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

	// Create a new project (only if not already created or shared)
	if options.currentProject == nil {
		if options.isMatrixTest && options.SharedProject != nil && *options.SharedProject {
			options.Logger.ShortInfo("Shared project should have been set up but was not - creating individual project as fallback")
		} else {
			options.Logger.ShortInfo("Creating individual project for test")
		}
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
		// Using shared or existing project
		if options.isMatrixTest && options.SharedProject != nil && *options.SharedProject {
			options.Logger.ShortInfo(fmt.Sprintf("Using shared project: %s with ID %s", options.currentProjectConfig.ProjectName, *options.currentProject.ID))
		} else {
			options.Logger.ShortInfo(fmt.Sprintf("Using existing project: %s with ID %s", options.ProjectName, *options.currentProject.ID))
		}
		// Ensure currentProjectConfig is set up properly for shared projects
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

	// Project cleanup logic:
	// - Matrix tests: handled centrally via CleanupSharedResources()
	// - Individual tests with SharedProject=false: clean up their own projects
	// - Individual tests with SharedProject=true: keep project for potential reuse
	if options.currentProject != nil && options.currentProject.ID != nil && !options.isMatrixTest && !*options.SharedProject {
		options.Logger.ShortInfo(fmt.Sprintf("Deleting the project %s with ID %s (SharedProject=false)", options.ProjectName, *options.currentProject.ID))
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
		if options.isMatrixTest {
			options.Logger.ShortInfo("Matrix test project will be cleaned up centrally")
		} else if options.SharedProject != nil && *options.SharedProject {
			options.Logger.ShortInfo("Shared project retained for potential reuse (SharedProject=true)")
		} else {
			options.Logger.ShortInfo("No project ID found to delete")
		}
	}

	// Catalog cleanup logic:
	// - Matrix tests: handled centrally via CleanupSharedResources()
	// - Individual tests with SharedCatalog=false: clean up their own catalogs
	// - Individual tests with SharedCatalog=true: keep catalog for potential reuse
	if options.catalog != nil && !options.isMatrixTest && !*options.SharedCatalog {
		options.Logger.ShortInfo(fmt.Sprintf("Deleting the catalog %s with ID %s (SharedCatalog=false)", *options.catalog.Label, *options.catalog.ID))
		err := options.CloudInfoService.DeleteCatalog(*options.catalog.ID)
		if err != nil {
			options.Logger.ShortError(fmt.Sprintf("Error deleting the catalog: %v", err))
			options.Testing.Fail()
		} else {
			options.Logger.ShortInfo(fmt.Sprintf("Deleted the catalog %s with ID %s", *options.catalog.Label, *options.catalog.ID))
		}
	} else {
		if options.isMatrixTest {
			options.Logger.ShortInfo("Matrix test catalog will be cleaned up centrally")
		} else if *options.SharedCatalog {
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
// The method supports two API patterns:
// 1. Legacy: Only BaseSetupFunc provided, creates options from scratch for each test case
// 2. Enhanced: BaseOptions + BaseSetupFunc, reduces boilerplate by providing common options
func (options *TestAddonOptions) RunAddonTestMatrix(matrix AddonTestMatrix) {
	options.Testing.Parallel()

	// Create shared resource tracking for the matrix
	var sharedCatalogOptions *TestAddonOptions
	var sharedMutex = &sync.Mutex{}

	// Pre-analyze all test cases to determine if any will need shared projects
	// This ensures we create the shared project during the first test setup if needed
	needsSharedProject := false
	for _, preTC := range matrix.TestCases {
		// Create a temporary testOptions to determine SharedProject setting for this test case
		var tempTestOptions *TestAddonOptions
		if matrix.BaseOptions != nil {
			tempTestOptions = matrix.BaseOptions.copy()
			if matrix.BaseSetupFunc != nil {
				tempTestOptions = matrix.BaseSetupFunc(tempTestOptions, preTC)
			}
		} else {
			if matrix.BaseSetupFunc == nil {
				panic("Either BaseOptions must be provided or BaseSetupFunc must be provided")
			}
			tempTestOptions = matrix.BaseSetupFunc(nil, preTC)
		}

		// Apply the same SharedProject logic as the main loop
		// Matrix tests default to shared projects for efficiency - override any BaseOptions setting
		if preTC.SkipInfrastructureDeployment {
			// Validation-only tests always share a project since they don't deploy infrastructure
			tempTestOptions.SharedProject = core.BoolPtr(true)
			needsSharedProject = true
		} else {
			// Deployment tests also default to sharing in matrix mode for efficiency
			tempTestOptions.SharedProject = core.BoolPtr(true)
			needsSharedProject = true
		}
	}

	// Debug log the shared project determination
	// Don't initialize logger here - let each test case initialize its own logger with correct prefix
	// This ensures matrix test cases get proper "[TestName - ADDON - TestCaseName]" prefixes

	if needsSharedProject {
		// Only log if we have a logger available, otherwise skip logging at matrix level
		if options.Logger != nil {
			options.Logger.ShortInfo("Matrix tests determined to need shared project - will create one")
		}
	} else {
		// Only log if we have a logger available, otherwise skip logging at matrix level
		if options.Logger != nil {
			options.Logger.ShortInfo("Matrix tests determined to NOT need shared project")
		}
	}

	for _, tc := range matrix.TestCases {
		tc := tc // Capture loop variable for parallel execution
		options.Testing.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			var testOptions *TestAddonOptions

			// Support both legacy and enhanced API patterns
			if matrix.BaseOptions != nil {
				// Enhanced API: Start with a copy of BaseOptions
				testOptions = matrix.BaseOptions.copy()
				testOptions.Testing = t // Override testing context for this specific test

				// Allow BaseSetupFunc to customize the copied options
				if matrix.BaseSetupFunc != nil {
					testOptions = matrix.BaseSetupFunc(testOptions, tc)
				}
			} else {
				// Legacy API: BaseSetupFunc creates options from scratch
				if matrix.BaseSetupFunc == nil {
					panic("Either BaseOptions must be provided or BaseSetupFunc must be provided")
				}
				testOptions = matrix.BaseSetupFunc(nil, tc)
			}

			// Apply test case specific prefix if provided
			if tc.Prefix != "" {
				testOptions.Prefix = tc.Prefix
			}
			// Ensure prefix is unique to avoid project name collisions
			// This is critical for matrix tests where multiple tests run in parallel
			// Keep test case name for logging and debugging, but use shorter prefix for resources
			if testOptions.Prefix != "" {
				// Use shorter prefix for resources: just base prefix + unique ID
				// Test case name is preserved in TestCaseName for logging purposes
				testOptions.Prefix = fmt.Sprintf("%s-%s", testOptions.Prefix, common.UniqueId())
			} else {
				// Generate prefix without test case name if no base prefix provided to keep it short
				testOptions.Prefix = fmt.Sprintf("test-%s", common.UniqueId())
			}
			testOptions.AddonConfig.Prefix = testOptions.Prefix

			// Set the test case name for logging
			testOptions.TestCaseName = tc.Name

			// Mark as matrix test so teardown doesn't delete shared catalog
			testOptions.isMatrixTest = true

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

			// Create addon configuration using the provided config function FIRST
			// This ensures we have offering details available for project naming and shared offering import
			testOptions.AddonConfig = matrix.AddonConfigFunc(testOptions, tc)

			// Set dependencies if provided in test case
			if tc.Dependencies != nil {
				testOptions.AddonConfig.Dependencies = tc.Dependencies

				// Ensure unique configuration names for dependencies as well
				for i := range testOptions.AddonConfig.Dependencies {
					dep := &testOptions.AddonConfig.Dependencies[i]
					if dep.ConfigName == "" {
						// Build unique dependency config name
						depConfigNameParts := []string{}
						if dep.Prefix != "" {
							depConfigNameParts = append(depConfigNameParts, dep.Prefix)
						}
						if dep.OfferingName != "" {
							depConfigNameParts = append(depConfigNameParts, dep.OfferingName)
						}
						if dep.OfferingFlavor != "" {
							depConfigNameParts = append(depConfigNameParts, dep.OfferingFlavor)
						}
						if tc.Name != "" {
							depConfigNameParts = append(depConfigNameParts, tc.Name)
						}

						if len(depConfigNameParts) > 0 {
							dep.ConfigName = strings.Join(depConfigNameParts, "-")
						} else {
							dep.ConfigName = fmt.Sprintf("dep-config-%s", common.UniqueId())
						}
					}
				}
			}

			// Ensure unique configuration names to avoid conflicts when sharing projects
			// Include test case name and offering details for better identification
			if testOptions.AddonConfig.ConfigName == "" {
				// Build a unique configuration name that includes test case and offering info
				configNameParts := []string{}
				if testOptions.AddonConfig.Prefix != "" {
					configNameParts = append(configNameParts, testOptions.AddonConfig.Prefix)
				}
				if testOptions.AddonConfig.OfferingName != "" {
					configNameParts = append(configNameParts, testOptions.AddonConfig.OfferingName)
				}
				if tc.Name != "" {
					configNameParts = append(configNameParts, tc.Name)
				}

				// If we have parts, join them; otherwise use a fallback
				if len(configNameParts) > 0 {
					testOptions.AddonConfig.ConfigName = strings.Join(configNameParts, "-")
				} else {
					testOptions.AddonConfig.ConfigName = fmt.Sprintf("config-%s", common.UniqueId())
				}
			}

			// Matrix tests default to shared projects for efficiency now that we have unique config names
			// Override any BaseOptions setting for matrix tests since they can safely share
			if tc.SkipInfrastructureDeployment {
				// Validation-only tests always share a project since they don't deploy infrastructure
				testOptions.SharedProject = core.BoolPtr(true)
				testOptions.Logger.ShortInfo("Matrix validation test using shared project for efficiency")
			} else {
				// Deployment tests also default to sharing in matrix mode for efficiency
				// Configuration names include test case names to prevent conflicts
				testOptions.SharedProject = core.BoolPtr(true)
				testOptions.Logger.ShortInfo("Matrix deployment test using shared project for efficiency")
			}

			// Now that SharedProject is determined, set the appropriate project name
			if testOptions.Prefix != "" {
				// For shared projects, use a consistent name across all test cases
				if testOptions.SharedProject != nil && *testOptions.SharedProject {
					// All shared projects use the same base name so they reference the same project
					// We'll use the offering name from the first test case that creates the shared project
					// The actual project name will be set when the shared project is created
					// For now, use a placeholder that will be overridden when the shared project is assigned
					testOptions.ProjectName = "shared-project-placeholder"
				} else { // Individual projects use descriptive names showing the specific test case
					// Format: "{offering-short-name}-{test-case}-{base-prefix}-{unique-id}"
					nameComponents := []string{}

					if testOptions.AddonConfig.OfferingName != "" {
						// Extract a shorter, more readable name from the offering
						offeringShortName := testOptions.AddonConfig.OfferingName
						if strings.HasPrefix(offeringShortName, "deploy-arch-") {
							// Strip "deploy-arch-" prefix if present (e.g., "deploy-arch-ibm-observability" -> "ibm-observability")
							offeringShortName = strings.TrimPrefix(offeringShortName, "deploy-arch-")
						}
						nameComponents = append(nameComponents, offeringShortName)
					}

					// Add test case name in lowercase for readability
					if tc.Name != "" {
						nameComponents = append(nameComponents, strings.ToLower(tc.Name))
					}

					// Extract base prefix and unique ID
					prefixParts := strings.Split(testOptions.Prefix, "-")
					if len(prefixParts) >= 3 {
						// We have base-prefix-testname-uniqueid, so take base prefix and unique ID
						basePrefix := prefixParts[0]
						uniqueId := prefixParts[len(prefixParts)-1]
						nameComponents = append(nameComponents, basePrefix, uniqueId)
					} else {
						// Fallback to the full prefix
						nameComponents = append(nameComponents, testOptions.Prefix)
					}

					testOptions.ProjectName = strings.Join(nameComponents, "-")
				}
			}

			// Log project sharing behavior for debugging
			testOptions.Logger.ShortInfo(fmt.Sprintf("Test case %s: SharedProject=%t, SkipInfrastructureDeployment=%t, ProjectName=%s", tc.Name, *testOptions.SharedProject, tc.SkipInfrastructureDeployment, testOptions.ProjectName))
			if *testOptions.SharedProject {
				if tc.SkipInfrastructureDeployment {
					testOptions.Logger.ShortInfo("Matrix validation test using shared project for efficiency")
				} else {
					testOptions.Logger.ShortInfo("Matrix deployment test using shared project")
				}
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

			// Handle shared catalog AND offering creation in matrix tests (single mutex)
			sharedMutex.Lock()
			if sharedCatalogOptions == nil {
				// This is the first test case - it will create the shared catalog and offering
				sharedCatalogOptions = testOptions

				// Create the shared catalog for all matrix tests
				if !testOptions.CatalogUseExisting {
					// Generate a more descriptive catalog name for matrix tests
					offeringShortName := "addon"
					if testOptions.AddonConfig.OfferingName != "" {
						// Extract a shorter, more readable name from the offering
						offeringShortName = testOptions.AddonConfig.OfferingName
						if strings.HasPrefix(offeringShortName, "deploy-arch-") {
							// Strip "deploy-arch-" prefix if present (e.g., "deploy-arch-ibm-observability" -> "ibm-observability")
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

					// Import the offering once for all matrix tests (now that we have addon config)
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

				// Create shared project if any test case in the matrix needs one
				if needsSharedProject {
					// Use a descriptive shared project name that includes offering information
					offeringShortName := "addon"
					if testOptions.AddonConfig.OfferingName != "" {
						// Strip the "deploy-arch-" prefix if present, otherwise use the full name
						if strings.HasPrefix(testOptions.AddonConfig.OfferingName, "deploy-arch-") {
							offeringShortName = strings.TrimPrefix(testOptions.AddonConfig.OfferingName, "deploy-arch-")
						} else {
							offeringShortName = testOptions.AddonConfig.OfferingName
						}
					}
					// Extract just the unique ID from the prefix (last part after the final hyphen)
					prefixParts := strings.Split(testOptions.Prefix, "-")
					uniqueId := prefixParts[len(prefixParts)-1]
					sharedProjectName := fmt.Sprintf("shared-%s-validation-%s", offeringShortName, uniqueId)
					testOptions.Logger.ShortInfo(fmt.Sprintf("Creating shared project for matrix tests: %s", sharedProjectName))

					// Set up project configuration for shared validation project
					if testOptions.ProjectDestroyOnDelete == nil {
						testOptions.ProjectDestroyOnDelete = core.BoolPtr(true)
					}
					if testOptions.ProjectAutoDeploy == nil {
						testOptions.ProjectAutoDeploy = core.BoolPtr(false)
					}
					if testOptions.ProjectMonitoringEnabled == nil {
						testOptions.ProjectMonitoringEnabled = core.BoolPtr(false)
					}

					sharedProjectConfig := &cloudinfo.ProjectsConfig{
						Location:           testOptions.ProjectLocation,
						ProjectName:        sharedProjectName,
						ProjectDescription: "Shared project for matrix tests",
						ResourceGroup:      testOptions.ResourceGroup,
						DestroyOnDelete:    *testOptions.ProjectDestroyOnDelete,
						MonitoringEnabled:  *testOptions.ProjectMonitoringEnabled,
						AutoDeploy:         *testOptions.ProjectAutoDeploy,
						Environments:       testOptions.ProjectEnvironments,
					}

					prj, _, err := testOptions.CloudInfoService.CreateProjectFromConfig(sharedProjectConfig)
					if err != nil {
						sharedMutex.Unlock() // Release mutex on error
						testOptions.Logger.ShortError(fmt.Sprintf("Error creating shared project: %v", err))
						require.NoError(t, err, "Failed to create shared project for matrix tests")
						return
					}

					// Store the shared project in sharedCatalogOptions for other tests to use
					sharedCatalogOptions.currentProject = prj
					sharedCatalogOptions.currentProjectConfig = sharedProjectConfig
					sharedCatalogOptions.currentProjectConfig.ProjectID = *prj.ID

					if prj != nil && prj.ID != nil {
						testOptions.Logger.ShortInfo(fmt.Sprintf("Created shared project: %s with ID %s", sharedProjectName, *prj.ID))
						projectURL := fmt.Sprintf("https://cloud.ibm.com/projects/%s/configurations", *prj.ID)
						testOptions.Logger.ShortInfo(fmt.Sprintf("Shared Project URL: %s", projectURL))
					} else {
						testOptions.Logger.ShortWarn("Created shared project but project details are incomplete")
					}
				} else {
					testOptions.Logger.ShortInfo(fmt.Sprintf("No shared project needed for matrix tests"))
				}

				sharedMutex.Unlock() // Release mutex only AFTER catalog, offering, AND project creation is complete
			} else {
				// Share the catalog, offering, and project tracking fields from the first instance
				// At this point, we know both catalog and offering creation is complete because the mutex ensures it
				testOptions.catalog = sharedCatalogOptions.catalog
				testOptions.offering = sharedCatalogOptions.offering

				// Share project if SharedProject is enabled
				if *testOptions.SharedProject {
					testOptions.currentProject = sharedCatalogOptions.currentProject
					testOptions.currentProjectConfig = sharedCatalogOptions.currentProjectConfig
					// Override the placeholder project name with the actual shared project name
					testOptions.ProjectName = sharedCatalogOptions.currentProjectConfig.ProjectName
					if testOptions.currentProject != nil && testOptions.currentProject.ID != nil {
						testOptions.Logger.ShortInfo(fmt.Sprintf("Test case %s: Shared project assigned: %s with ID %s", tc.Name, testOptions.ProjectName, *testOptions.currentProject.ID))
					} else {
						testOptions.Logger.ShortWarn(fmt.Sprintf("Test case %s: Shared project assignment failed - currentProject is nil", tc.Name))
					}
				} else {
					testOptions.Logger.ShortInfo(fmt.Sprintf("Test case %s: SharedProject=false, will create individual project", tc.Name))
				}

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
				if *testOptions.SharedProject && testOptions.currentProject != nil && testOptions.currentProject.ID != nil {
					testOptions.Logger.ShortInfo(fmt.Sprintf("Using shared project: %s with ID %s", testOptions.currentProjectConfig.ProjectName, *testOptions.currentProject.ID))
				}
			}

			// Run the test
			err := testOptions.RunAddonTest()
			require.NoError(t, err, "Addon Test had an unexpected error")
		})
	}

	// Cleanup shared resources after all tests complete
	// Use a separate goroutine that waits for all test goroutines to complete
	go func() {
		options.Testing.Cleanup(func() {
			if sharedCatalogOptions != nil {
				sharedCatalogOptions.CleanupSharedResources()
			}
		})
	}()
}
