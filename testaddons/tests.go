package testaddons

import (
	"fmt"
	"log"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/IBM/go-sdk-core/v5/core"
	Core "github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/platform-services-go-sdk/catalogmanagementv1"
	"github.com/IBM/project-go-sdk/projectv1"
	"github.com/stretchr/testify/assert"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/cloudinfo"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/common"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testprojects"
)

// parse function is used by matchVersion function to find the most suitable version
// in case it is not pinned in the dependency
func parse(v string) (int, int, int, bool) {
	re := regexp.MustCompile(`^v(\d+)\.(\d+)\.(\d+)$`)
	matches := re.FindStringSubmatch(v)
	if matches == nil {
		return 0, 0, 0, false
	}
	major, _ := strconv.Atoi(matches[1])
	minor, _ := strconv.Atoi(matches[2])
	patch, _ := strconv.Atoi(matches[3])
	return major, minor, patch, true
}

// Main matching function
// This function takes all the versions avialable for a dependency
// and returns the suitable version matching target
// here target can be an actual version or unpinned version like ^v3.0.1 or ~v4.1.4
func matchVersion(versions []string, target string) string {
	operator := ""
	if strings.HasPrefix(target, "^") || strings.HasPrefix(target, "~") {
		operator = string(target[0])
		target = target[1:]
	}

	targetMajor, targetMinor, targetPatch, ok := parse(target)
	if !ok {
		return ""
	}

	candidates := [][]int{}
	versionMap := map[string][]int{}

	for _, v := range versions {
		major, minor, patch, valid := parse(v)
		if !valid {
			continue
		}
		versionTriplet := []int{major, minor, patch}
		versionMap[v] = versionTriplet

		// Handle version matching based on operator
		switch operator {
		case "^":
			if major == targetMajor {
				candidates = append(candidates, versionTriplet)
			}
		case "~":
			if major == targetMajor && minor == targetMinor {
				candidates = append(candidates, versionTriplet)
			}
		default:
			// Exact match
			if major == targetMajor && minor == targetMinor && patch == targetPatch {
				return v
			}
		}
	}

	if len(candidates) == 0 {
		return ""
	}

	// Sort candidates by major, minor, patch descending
	sort.SliceStable(candidates, func(i, j int) bool {
		for k := 0; k < 3; k++ {
			if candidates[i][k] != candidates[j][k] {
				return candidates[i][k] > candidates[j][k]
			}
		}
		return false
	})

	// Convert top candidate back to string and find original version string
	top := candidates[0]
	for ver, parts := range versionMap {
		if parts[0] == top[0] && parts[1] == top[1] && parts[2] == top[2] {
			return ver
		}
	}

	return ""
}

func (options *TestAddonOptions) GetDependencyVersion(depCatalogID string, depOfferingID string, depVersion string, depFlavor string) (string, string, error) {

	_, response, err := options.CloudInfoService.GetOffering(depCatalogID, depOfferingID)
	if err != nil {
		return "", "", fmt.Errorf("unable to get the child offering %s", err)
	}

	depOffering, ok := response.Result.(*catalogmanagementv1.Offering)
	depVersionList := make([]string, 0)
	if ok {

		for _, kind := range depOffering.Kinds {

			if *kind.InstallKind == "terraform" {

				for _, v := range kind.Versions {

					depVersionList = append(depVersionList, *v.Version)
				}
			}
		}
	}

	bestVersion := matchVersion(depVersionList, depVersion)
	if bestVersion == "" {
		return "", "", fmt.Errorf("could not find a matching version for dependency %s ", *depOffering.Name)
	}

	versionLocator := ""

	for _, kind := range depOffering.Kinds {

		if *kind.InstallKind == "terraform" {

			for _, v := range kind.Versions {

				if *v.Version == bestVersion && *v.Flavor.Name == depFlavor {
					versionLocator = *v.VersionLocator
					break
				}
			}
		}
	}

	return bestVersion, versionLocator, nil

}

func (options *TestAddonOptions) buildDependencyGraph(catalogID string, offeringID string, versionLocator string, flavor string, graph map[cloudinfo.OfferingNameVersionFlavor][]cloudinfo.OfferingNameVersionFlavor, visited map[string]bool) {

	if visited[versionLocator] {
		return
	}

	visited[versionLocator] = true
	_, response, err := options.CloudInfoService.GetOffering(catalogID, offeringID)
	if err != nil {
		log.Fatal(err)
	}

	offering, ok := response.Result.(*catalogmanagementv1.Offering)
	var version catalogmanagementv1.Version
	found := false
	if ok {

		for _, kind := range offering.Kinds {

			if *kind.InstallKind == "terraform" {

				for _, v := range kind.Versions {

					if *v.VersionLocator == versionLocator {
						version = v
						found = true
						break
					}
				}

			}
			if found {
				break
			}
		}
	}

	offeringVersion := *version.Version
	offeringName := *offering.Name

	addon := cloudinfo.OfferingNameVersionFlavor{
		Name:    offeringName,
		Version: offeringVersion,
		Flavor:  flavor,
	}

	for _, dep := range version.SolutionInfo.Dependencies {

		depCatalogID := *dep.CatalogID
		depOfferingID := *dep.ID
		depFlavor := dep.Flavors[0]
		// GetDependecyVersion function is needed to find VersionLocator of dependency tile
		// which will be used by current addon and we will recursively process for dependency
		// this function is also going to handle the case in which dependency version is not pinned
		depVersion, depVersionLocator, err := options.GetDependencyVersion(depCatalogID, depOfferingID, *dep.Version, depFlavor)

		if err != nil {
			fmt.Println(err)
			return
		}

		child := cloudinfo.OfferingNameVersionFlavor{
			Name:    *dep.Name,
			Version: depVersion,
			Flavor:  depFlavor,
		}

		graph[addon] = append(graph[addon], child)

		options.buildDependencyGraph(depCatalogID, depOfferingID, depVersionLocator, depFlavor, graph, visited)

	}

}

func (options *TestAddonOptions) printactuallydeployed(src cloudinfo.AddonConfig, visited map[string]bool) {

	if visited[src.VersionLocator] {
		return
	} else {

		visited[src.VersionLocator] = true

		fmt.Println(src.OfferingName, src.ResolvedVersion)
		for _, dep := range src.Dependencies {

			options.printactuallydeployed(dep, visited)
		}
	}
}

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
				// if input starts with ref:/
				if strings.HasPrefix(input.(string), "ref:/") {
					options.Logger.ShortInfo(fmt.Sprintf("    %s", input))
					references = append(references, input.(string))
				}
			}

			if len(references) > 0 {
				res_resp, err := options.CloudInfoService.ResolveReferencesFromStrings(*options.currentProject.Location, references, options.currentProjectConfig.ProjectID)
				if err != nil {
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

		// check if any required inputs are not set
		for _, input := range currentConfigDetails.Definition.(*projectv1.ProjectConfigDefinitionResponse).Inputs {
			options.Logger.ShortInfo(fmt.Sprintf("Input: %v ", input))
			// TODO: check if input is required and not set
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

	// Check if all refs are valid
	if options.PreDeployHook != nil {
		options.Logger.ShortInfo("Running PreDeployHook")
		hookErr := options.PreDeployHook(options)
		if hookErr != nil {
			options.Testing.Fail()
			return hookErr
		}
		options.Logger.ShortInfo("Finished PreDeployHook")
	}

	// Trigger Deploy
	var rootCatalogID, rootOfferingID, rootVersionLocator string
	for _, config := range allConfigs {

		if *config.ID == configDetails.ConfigID {
			rootVersionLocator = *config.Definition.LocatorID
			break
		}
	}
	// get catalog ID from version locator
	rootCatalogID = strings.SplitN(rootVersionLocator, ".", 2)[0]
	rootOfferingID = options.AddonConfig.OfferingID

	graph := make(map[cloudinfo.OfferingNameVersionFlavor][]cloudinfo.OfferingNameVersionFlavor)
	visited := make(map[string]bool)
	options.buildDependencyGraph(rootCatalogID, rootOfferingID, rootVersionLocator, options.AddonConfig.OfferingFlavor, graph, visited)

	for key, value := range graph {

		fmt.Printf("{%s %s}---> ", key.Name, key.Version)

		for _, dep := range value {

			fmt.Printf("{%s %s} ", dep.Name, dep.Version)
		}
		fmt.Println()
		fmt.Println()
	}

	visited2 := make(map[string]bool)

	fmt.Println("printing the actually deployed configs")

	options.printactuallydeployed(options.AddonConfig, visited2)

	// now validate what is actually deployed by iterating over options.AddonConfig
	// what is expected which is present in the dependency graph

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

	if options.ProjectName != "" {
		options.Logger.SetPrefix(fmt.Sprintf("ADDON - %s", options.ProjectName))
	} else {
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
		options.Logger.ShortInfo(fmt.Sprintf("Creating a new catalog: %s", options.CatalogName))
		catalog, err := options.CloudInfoService.CreateCatalog(options.CatalogName)
		if err != nil {
			options.Logger.ShortError(fmt.Sprintf("Error creating a new catalog: %v", err))
			options.Testing.Fail()
			return fmt.Errorf("error creating a new catalog: %w", err)
		}
		options.catalog = catalog
		options.Logger.ShortInfo(fmt.Sprintf("Created a new catalog: %s with ID %s", *options.catalog.Label, *options.catalog.ID))
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
	version := fmt.Sprintf("v0.0.1-dev-%s", options.Prefix)
	options.Logger.ShortInfo(fmt.Sprintf("Importing the offering flavor: %s from branch: %s as version: %s", options.AddonConfig.OfferingFlavor, *options.currentBranchUrl, version))
	offering, err := options.CloudInfoService.ImportOffering(*options.catalog.ID, *options.currentBranchUrl, options.AddonConfig.OfferingName, options.AddonConfig.OfferingFlavor, version, options.AddonConfig.OfferingInstallKind)
	if err != nil {
		options.Logger.ShortError(fmt.Sprintf("Error importing the offering: %v", err))
		options.Testing.Fail()
		return fmt.Errorf("error importing the offering: %w", err)
	}
	options.offering = offering
	options.Logger.ShortInfo(fmt.Sprintf("Imported flavor: %s with version: %s to %s", *options.offering.Label, version, *options.catalog.Label))
	newVersionLocator := ""
	if options.offering.Kinds != nil {
		newVersionLocator = *options.offering.Kinds[0].Versions[0].VersionLocator
	}
	options.AddonConfig.OfferingName = *options.offering.Name
	options.AddonConfig.OfferingID = *options.offering.ID
	options.AddonConfig.VersionLocator = newVersionLocator
	options.AddonConfig.OfferingLabel = *options.offering.Label

	options.Logger.ShortInfo(fmt.Sprintf("Offering Version Locator: %s", options.AddonConfig.VersionLocator))

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
	if options.currentProject != nil && options.currentProject.ID != nil {
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
	// Delete Catalog
	if options.catalog != nil {
		options.Logger.ShortInfo(fmt.Sprintf("Deleting the catalog %s with ID %s", *options.catalog.Label, *options.catalog.ID))
		err := options.CloudInfoService.DeleteCatalog(*options.catalog.ID)
		if err != nil {
			options.Logger.ShortError(fmt.Sprintf("Error deleting the catalog: %v", err))
			options.Testing.Fail()
		} else {
			options.Logger.ShortInfo(fmt.Sprintf("Deleted the catalog %s with ID %s", *options.catalog.Label, *options.catalog.ID))
		}
	} else {
		options.Logger.ShortInfo("No catalog to delete")
	}
}
