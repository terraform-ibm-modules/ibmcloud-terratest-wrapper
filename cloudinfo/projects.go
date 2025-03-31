package cloudinfo

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/IBM/go-sdk-core/v5/core"
	project "github.com/IBM/project-go-sdk/projectv1"
)

// CreateProjectFromConfig creates a project with the given config
// config: the config to use
// Returns an error if one occurs
func (infoSvc *CloudInfoService) CreateProjectFromConfig(config *ProjectsConfig) (result *project.Project, response *core.DetailedResponse, err error) {
	// Use defaults if not provided
	if config.Location == "" {
		validRegions := []string{"us-south", "us-east", "eu-gb", "eu-de"}
		randomIndex := rand.Intn(len(validRegions))
		config.Location = validRegions[randomIndex]
	}
	if config.ProjectName == "" {
		config.ProjectName = "Test Project"
	}
	if config.ProjectDescription == "" {
		config.ProjectDescription = "Test project description"
	}
	if config.ResourceGroup == "" {
		config.ResourceGroup = "Default"
	}
	if config.Configs == nil {
		config.Configs = []project.ProjectConfigPrototype{}
	}
	if config.Environments == nil {
		config.Environments = []project.EnvironmentPrototype{}
	}
	if config.Headers == nil {
		config.Headers = map[string]string{}
	}

	projectOptions := &project.CreateProjectOptions{
		Definition: &project.ProjectPrototypeDefinition{
			Name:              &config.ProjectName,
			DestroyOnDelete:   core.BoolPtr(config.DestroyOnDelete),
			Description:       &config.ProjectDescription,
			Store:             config.Store,
			MonitoringEnabled: core.BoolPtr(config.MonitoringEnabled),
			AutoDeploy:        core.BoolPtr(config.AutoDeploy),
		},
		Location:      &config.Location,
		ResourceGroup: &config.ResourceGroup,
		Configs:       config.Configs,
		Environments:  config.Environments,
		Headers:       config.Headers,
	}

	return infoSvc.projectsService.CreateProject(projectOptions)
}

func (infoSvc *CloudInfoService) GetProject(projectID string) (result *project.Project, response *core.DetailedResponse, err error) {
	getProjectOptions := &project.GetProjectOptions{
		ID: &projectID,
	}
	return infoSvc.projectsService.GetProject(getProjectOptions)
}

func (infoSvc *CloudInfoService) GetProjectConfigs(projectID string) (results []project.ProjectConfigSummary, err error) {
	listConfigsOptions := &project.ListConfigsOptions{
		ProjectID: &projectID,
		Limit:     core.Int64Ptr(int64(20)),
	}

	pager, err := infoSvc.projectsService.NewConfigsPager(listConfigsOptions)
	if err != nil {
		return nil, err
	}
	var allResults []project.ProjectConfigSummary
	for pager.HasNext() {
		nextPage, err := pager.GetNext()
		if err != nil {
			return nil, err
		}
		allResults = append(allResults, nextPage...)
	}

	return allResults, nil
}

func (infoSvc *CloudInfoService) DeleteProject(projectID string) (result *project.ProjectDeleteResponse, response *core.DetailedResponse, err error) {
	deleteProjectOptions := &project.DeleteProjectOptions{
		ID: &projectID,
	}
	return infoSvc.projectsService.DeleteProject(deleteProjectOptions)
}

// CreateConfig creates a project config
func (infoSvc *CloudInfoService) CreateConfig(configDetails *ConfigDetails) (result *project.ProjectConfig, response *core.DetailedResponse, err error) { //
	// if authatization is not provided, use API key, if its set
	// Best effort to try set an authorization method
	// 1. If the user has provided an authorization method, use it
	// 2. If not try use infoSvc.authenticator.ApiKey
	if configDetails.Authorizations == nil {
		if infoSvc.authenticator != nil {
			if infoSvc.authenticator.ApiKey != "" {
				authMethod := project.ProjectConfigAuth_Method_ApiKey
				configDetails.Authorizations = &project.ProjectConfigAuth{
					ApiKey: &infoSvc.authenticator.ApiKey,
					Method: &authMethod,
				}
			}
		}
	}

	createConfigOptions := &project.CreateConfigOptions{
		ProjectID: &configDetails.ProjectID,
		Definition: &project.ProjectConfigDefinitionPrototype{
			Description:    &configDetails.Description,
			Name:           &configDetails.Name,
			LocatorID:      &configDetails.StackLocatorID,
			Authorizations: configDetails.Authorizations,
		},
	}
	return infoSvc.projectsService.CreateConfig(createConfigOptions)
}

// CreateDaConfig creates a DA project config
func (infoSvc *CloudInfoService) CreateDaConfig(configDetails *ConfigDetails) (result *project.ProjectConfig, response *core.DetailedResponse, err error) {
	createConfigOptions := &project.CreateConfigOptions{
		ProjectID: &configDetails.ProjectID,
		Definition: &project.ProjectConfigDefinitionPrototype{
			Description:    &configDetails.Description,
			Name:           &configDetails.Name,
			LocatorID:      &configDetails.StackLocatorID,
			Authorizations: configDetails.Authorizations,
			Inputs:         configDetails.Inputs,
			Settings:       configDetails.Settings,
		},
	}
	return infoSvc.projectsService.CreateConfig(createConfigOptions)
}

func (infoSvc *CloudInfoService) CreateConfigFromCatalogJson(configDetails *ConfigDetails, catalogJsonPath string) (result *project.ProjectConfig, response *core.DetailedResponse, err error) {
	// TODO: Handle multiple products/flavors
	// Read the catalog JSON file
	jsonFile, err := os.ReadFile(catalogJsonPath)
	if err != nil {
		log.Println("Error reading catalog JSON file:", err)
		return nil, nil, err
	}

	// Unmarshal the JSON data into the config variable
	var catalogConfig CatalogJson
	err = json.Unmarshal(jsonFile, &catalogConfig)
	if err != nil {
		log.Println("Error unmarshaling catalog JSON:", err)
		return nil, nil, err
	}
	// TODO: Validate inputs are valid for the stack

	// TODO: override inputs with values from catalogConfig
	configDetails.Name = catalogConfig.Products[0].Name
	configDetails.Description = catalogConfig.Products[0].Label
	return infoSvc.CreateConfig(configDetails)
}

func (infoSvc *CloudInfoService) CreateNewStack(stackConfig *ConfigDetails) (result *project.StackDefinition, response *core.DetailedResponse, err error) {

	// Create a project config first
	createProjectConfigDefinitionOptions := &project.ProjectConfigDefinitionPrototypeStackConfigDefinitionProperties{
		Description:    &stackConfig.Description,
		Name:           &stackConfig.Name,
		Members:        stackConfig.MemberConfigs,
		Authorizations: stackConfig.Authorizations,
		//Inputs:         stackConfig.Inputs, // Inputs are set in the stack definition this is not valid to set them here
		EnvironmentID: stackConfig.EnvironmentID,
	}
	createConfigOptions := infoSvc.projectsService.NewCreateConfigOptions(
		stackConfig.ProjectID,
		createProjectConfigDefinitionOptions,
	)
	config, configResp, configErr := infoSvc.projectsService.CreateConfig(createConfigOptions)
	if configErr != nil {
		return nil, configResp, configErr
	}

	stackConfig.ConfigID = *config.ID
	// Then apply the stack definition
	stackDefOptions := infoSvc.projectsService.NewCreateStackDefinitionOptions(stackConfig.ProjectID, *config.ID, stackConfig.StackDefinition)
	result, response, err = infoSvc.stackDefinitionCreator.CreateStackDefinitionWrapper(stackDefOptions, stackConfig.Members)

	return result, response, err

}

// CreateStackDefinition is a wrapper around projectv1.CreateStackDefinition to allow correct mocking in the tests
func (infoSvc *CloudInfoService) CreateStackDefinitionWrapper(stackDefOptions *project.CreateStackDefinitionOptions, members []project.ProjectConfig) (result *project.StackDefinition, response *core.DetailedResponse, err error) {

	// dummy use of members
	_ = members

	result, response, err = infoSvc.projectsService.CreateStackDefinition(stackDefOptions)

	return result, response, err

}

func (infoSvc *CloudInfoService) UpdateStackFromConfig(stackConfig *ConfigDetails) (result *project.StackDefinition, response *core.DetailedResponse, err error) {

	updateStackDefinitionOptions := &project.UpdateStackDefinitionOptions{
		ProjectID:       &stackConfig.ProjectID,
		ID:              &stackConfig.ConfigID,
		StackDefinition: stackConfig.StackDefinition,
	}
	return infoSvc.projectsService.UpdateStackDefinition(updateStackDefinitionOptions)
}

func (infoSvc *CloudInfoService) ForceValidateProjectConfig(configDetails *ConfigDetails) (result *project.ProjectConfigVersion, response *core.DetailedResponse, err error) {
	validateConfigOptions := &project.ValidateConfigOptions{
		ProjectID: &configDetails.ProjectID,
		ID:        &configDetails.ConfigID,
	}
	return infoSvc.projectsService.ValidateConfig(validateConfigOptions)
}

func (infoSvc *CloudInfoService) ValidateProjectConfig(configDetails *ConfigDetails) (result *project.ProjectConfigVersion, response *core.DetailedResponse, err error) {
	configVersion, isDeployed := infoSvc.IsConfigDeployed(configDetails)
	if !isDeployed {
		return infoSvc.ForceValidateProjectConfig(configDetails)
	} else {
		return configVersion, nil, nil
	}
}

func (infoSvc *CloudInfoService) GetProjectConfigVersion(configDetails *ConfigDetails, version int64) (result *project.ProjectConfigVersion, response *core.DetailedResponse, err error) {

	getConfigOptions := infoSvc.projectsService.NewGetConfigVersionOptions(configDetails.ProjectID, configDetails.ConfigID, version)
	return infoSvc.projectsService.GetConfigVersion(getConfigOptions)
}

func (infoSvc *CloudInfoService) GetConfig(configDetails *ConfigDetails) (result *project.ProjectConfig, response *core.DetailedResponse, err error) {
	getConfigOptions := &project.GetConfigOptions{
		ProjectID: &configDetails.ProjectID,
		ID:        &configDetails.ConfigID,
	}
	return infoSvc.projectsService.GetConfig(getConfigOptions)
}

// UpdateConfig updates a project config
func (infoSvc *CloudInfoService) UpdateConfig(configDetails *ConfigDetails, configuration project.ProjectConfigDefinitionPatchIntf) (result *project.ProjectConfig, response *core.DetailedResponse, err error) {
	updateConfigOptions := &project.UpdateConfigOptions{
		ProjectID:  &configDetails.ProjectID,
		ID:         &configDetails.ConfigID,
		Definition: configuration,
	}
	return infoSvc.projectsService.UpdateConfig(updateConfigOptions)
}

func (infoSvc *CloudInfoService) UpdateConfigWithHeaders(configDetails *ConfigDetails, configuration project.ProjectConfigDefinitionPatchIntf, headers map[string]string) (result *project.ProjectConfig, response *core.DetailedResponse, err error) {
	updateConfigOptions := &project.UpdateConfigOptions{
		ProjectID:  &configDetails.ProjectID,
		ID:         &configDetails.ConfigID,
		Definition: configuration,
		Headers:    headers,
	}
	return infoSvc.projectsService.UpdateConfig(updateConfigOptions)
}

// DeployConfig deploys a project config
func (infoSvc *CloudInfoService) DeployConfig(configDetails *ConfigDetails) (result *project.ProjectConfigVersion, response *core.DetailedResponse, err error) {
	configVersion, isDeployed := infoSvc.IsConfigDeployed(configDetails)
	if !isDeployed {
		return infoSvc.ForceDeployConfig(configDetails)
	}
	return configVersion, nil, nil
}

// ForceDeployConfig forcefully deploys a project config
func (infoSvc *CloudInfoService) ForceDeployConfig(configDetails *ConfigDetails) (result *project.ProjectConfigVersion, response *core.DetailedResponse, err error) {
	deployConfigOptions := &project.DeployConfigOptions{
		ProjectID: &configDetails.ProjectID,
		ID:        &configDetails.ConfigID,
	}
	return infoSvc.projectsService.DeployConfig(deployConfigOptions)
}

// IsConfigDeployed checks if the config is deployed
func (infoSvc *CloudInfoService) IsConfigDeployed(configDetails *ConfigDetails) (projectConfig *project.ProjectConfigVersion, isDeployed bool) {
	config, _, err := infoSvc.GetConfig(configDetails)
	if err != nil {
		log.Println("Error getting config:", err)
		return nil, false
	}
	configVersion, _, err := infoSvc.GetProjectConfigVersion(configDetails, *config.Version)
	if err != nil {
		log.Println("Error getting config version:", err)
		return nil, false
	}
	if config != nil {
		if config.State != nil {
			if *config.State == project.ProjectConfig_State_Deployed {

				return configVersion, true
			} else {
				return configVersion, false
			}
		}
	}
	return configVersion, false
}

// UndeployConfig undeploys a project config
func (infoSvc *CloudInfoService) UndeployConfig(details *ConfigDetails) (result *project.ProjectConfigVersion, response *core.DetailedResponse, err error) {
	undeployConfigOptions := &project.UndeployConfigOptions{
		ProjectID: &details.ProjectID,
		ID:        &details.ConfigID,
	}
	return infoSvc.projectsService.UndeployConfig(undeployConfigOptions)
}

// IsUndeploying checks if the config is undeploying
func (infoSvc *CloudInfoService) IsUndeploying(details *ConfigDetails) (projectConfig *project.ProjectConfigVersion, isUndeploying bool) {
	config, _, err := infoSvc.GetConfig(details)
	if err != nil {
		log.Println("Error getting config:", err)
		return nil, false

	}
	configVersion, _, err := infoSvc.GetProjectConfigVersion(details, *config.Version)
	if err != nil {
		log.Println("Error getting config version:", err)
		return nil, false
	}
	if config != nil {
		if config.State != nil {
			if *config.State == project.ProjectConfig_State_Undeploying {
				return configVersion, true
			} else {
				return configVersion, false
			}
		}
	}
	return configVersion, false
}

// CreateStackFromConfigFile creates a stack from a config file
// This method orchestrates the stack creation process by managing stack configurations, member inputs, and handling catalog configurations.
func (infoSvc *CloudInfoService) CreateStackFromConfigFile(stackConfig *ConfigDetails, stackConfigPath string, catalogJsonPath string) (stackDefinition *project.StackDefinition, response *core.DetailedResponse, err error) {
	if stackConfig.Authorizations == nil {
		// Set default authorizations if not provided
		stackConfig.Authorizations = &project.ProjectConfigAuth{
			ApiKey: &infoSvc.authenticator.ApiKey,
			Method: core.StringPtr(project.ProjectConfigAuth_Method_ApiKey),
		}
	}

	// Track inputs that should not be overridden
	doNotOverrideInputs := trackStackInputs(stackConfig)

	// Read the stack configuration from the provided file path
	stackJson, err := readStackConfig(stackConfigPath)
	if err != nil {
		return nil, nil, err
	}

	// Check for any duplicate stack inputs or outputs
	errorMessages := checkStackForDuplicates(stackJson)

	// Set up member inputs from the stack configuration
	memberInputsMap := setMemberInputsMap(stackConfig, doNotOverrideInputs)

	// Process each member in the stack configuration
	stackConfig, err = processMembers(stackJson, stackConfig, memberInputsMap, doNotOverrideInputs, infoSvc)
	if err != nil {
		return nil, nil, err
	}

	// Define stack inputs and outputs
	stackInputsDef, stackOutputsDef := defineStackIO(stackJson, stackConfig, doNotOverrideInputs)

	stackConfig.StackDefinition = &project.StackDefinitionBlockPrototype{
		Inputs:  stackInputsDef,
		Outputs: stackOutputsDef,
	}

	// Read the catalog configuration
	catalogConfig, catalogProductIndex, catalogFlavorIndex, err := readCatalogConfig(catalogJsonPath, stackConfig, &errorMessages)
	if err != nil {
		return nil, nil, err
	}

	// Update stack inputs from catalog configuration
	updateInputsFromCatalog(stackConfig, catalogConfig, catalogProductIndex, catalogFlavorIndex, doNotOverrideInputs)

	// Check if all catalog inputs exist in the stack definition and match types
	validateCatalogInputsInStackDefinition(stackJson, catalogConfig, catalogProductIndex, catalogFlavorIndex, &errorMessages)

	// If there are any errors from the validation process, return them
	if len(errorMessages) > 0 {
		return nil, nil, fmt.Errorf(strings.Join(errorMessages, "\n"))
	}

	// Sort stack inputs by name for consistency
	sortInputsByName(stackConfig)

	// Set stack name and description using catalog product and flavor labels, ensure the name is valid. For the test the name does not really matter so strip invalid characters.
	// Define the pattern to match valid characters
	var validNamePattern = regexp.MustCompile(`[^a-zA-Z0-9-_ ]`)

	// Generate the stack name and strip invalid characters
	rawName := fmt.Sprintf("%s-%s", catalogConfig.Products[catalogProductIndex].Label, catalogConfig.Products[catalogProductIndex].Flavors[catalogFlavorIndex].Label)
	stackConfig.Name = validNamePattern.ReplaceAllString(rawName, "")
	stackConfig.Description = fmt.Sprintf("%s-%s", catalogConfig.Products[catalogProductIndex].Label, catalogConfig.Products[catalogProductIndex].Flavors[catalogFlavorIndex].Label)

	// Create the new stack
	return infoSvc.CreateNewStack(stackConfig)
}

// trackStackInputs tracks inputs that should not be overridden.
// This function initializes a map of inputs that should be preserved in the stack configuration.
func trackStackInputs(stackConfig *ConfigDetails) map[string]map[string]interface{} {
	doNotOverrideInputs := make(map[string]map[string]interface{})
	doNotOverrideInputs["stack"] = make(map[string]interface{})
	if stackConfig.Inputs != nil {
		for key, value := range stackConfig.Inputs {
			doNotOverrideInputs["stack"][key] = value
		}
	}
	return doNotOverrideInputs
}

// readStackConfig reads the stack configuration from a JSON file.
// This function reads and unmarshals the stack configuration from the given file path.
func readStackConfig(stackConfigPath string) (Stack, error) {
	jsonFile, err := os.ReadFile(stackConfigPath)
	if err != nil {
		log.Println("Error reading config JSON file:", err)
		return Stack{}, err
	}

	var stackJson Stack
	err = json.Unmarshal(jsonFile, &stackJson)
	if err != nil {
		log.Println("Error unmarshalling JSON:", err)
		return Stack{}, err
	}
	return stackJson, nil
}

// checkStackForDuplicates checks for duplicate inputs and outputs in the stack configuration.
// This function validates the stack configuration for any duplicate inputs or outputs and returns any errors found.
func checkStackForDuplicates(stackJson Stack) []string {
	errorMessages := []string{}

	inputNames := make(map[string]bool)
	for _, input := range stackJson.Inputs {
		if _, exists := inputNames[input.Name]; exists {
			errorMessages = append(errorMessages, fmt.Sprintf("duplicate stack input variable found: %s", input.Name))
		} else {
			inputNames[input.Name] = true
		}
	}

	outputNames := make(map[string]bool)
	for _, output := range stackJson.Outputs {
		if _, exists := outputNames[output.Name]; exists {
			errorMessages = append(errorMessages, fmt.Sprintf("duplicate stack output variable found: %s", output.Name))
		} else {
			outputNames[output.Name] = true
		}
	}

	for _, member := range stackJson.Members {
		memberInputNames := make(map[string]bool)
		for _, input := range member.Inputs {
			if _, exists := memberInputNames[input.Name]; exists {
				errorMessages = append(errorMessages, fmt.Sprintf("duplicate member input variable found member: %s input: %s", member.Name, input.Name))
			} else {
				memberInputNames[input.Name] = true
			}
		}
	}

	return errorMessages
}

// setMemberInputsMap sets up the inputs for each member in the stack configuration.
// This function creates a map of inputs for each member and also tracks inputs that should not be overridden.
func setMemberInputsMap(stackConfig *ConfigDetails, doNotOverrideInputs map[string]map[string]interface{}) map[string]map[string]interface{} {
	memberInputsMap := make(map[string]map[string]interface{})
	for _, memberConfig := range stackConfig.MemberConfigDetails {
		memberInputs := make(map[string]interface{})
		for key, value := range memberConfig.Inputs {
			memberInputs[key] = value
			if _, ok := doNotOverrideInputs[memberConfig.Name]; !ok {
				doNotOverrideInputs[memberConfig.Name] = make(map[string]interface{})
			}
			doNotOverrideInputs[memberConfig.Name][key] = value
		}
		memberInputsMap[memberConfig.Name] = memberInputs
	}
	return memberInputsMap
}

// processMembers processes each member in the stack configuration.
// This function iterates over each member, sets up its configuration, and creates the necessary project configurations.
func processMembers(stackJson Stack, stackConfig *ConfigDetails, memberInputsMap map[string]map[string]interface{}, doNotOverrideInputs map[string]map[string]interface{}, infoSvc *CloudInfoService) (*ConfigDetails, error) {
	for _, member := range stackJson.Members {
		inputs := make(map[string]interface{})
		for _, input := range member.Inputs {
			val := input.Value
			inputs[input.Name] = val
		}

		if memberInputs, exists := memberInputsMap[member.Name]; exists {
			for key, value := range memberInputs {
				inputs[key] = value
			}
		}

		newConfig := ConfigDetails{
			ProjectID:      stackConfig.ProjectID,
			Name:           member.Name,
			Description:    member.Name,
			StackLocatorID: member.VersionLocator,
			Inputs:         inputs,
			Authorizations: stackConfig.Authorizations,
		}
		daProjectConfig, _, createDaErr := infoSvc.CreateDaConfig(&newConfig)
		if createDaErr != nil {
			log.Println("Error creating config from JSON:", createDaErr)
			log.Printf("Current Member Name: %s\n", member.Name)
			log.Printf("Current Member description: %s\n", member.Name)
			log.Printf("Current Member VersionLocator: %s\n", member.VersionLocator)
			log.Printf("Current Member Inputs: %v\n", inputs)
			return nil, createDaErr
		}

		curMemberNameVar := member.Name
		curMemberName := &curMemberNameVar

		curDaProjectConfig := daProjectConfig.ID
		stackConfig.Members = append(stackConfig.Members, *daProjectConfig)
		stackConfig.MemberConfigs = append(stackConfig.MemberConfigs, project.StackConfigMember{
			Name:     curMemberName,
			ConfigID: curDaProjectConfig,
		})
	}
	return stackConfig, nil
}

// defineStackIO defines the inputs and outputs for the stack configuration.
// This function creates the stack input and output definitions based on the provided stack configuration and JSON data.
func defineStackIO(stackJson Stack, stackConfig *ConfigDetails, doNotOverrideInputs map[string]map[string]interface{}) ([]project.StackDefinitionInputVariable, []project.StackDefinitionOutputVariable) {
	var stackInputsDef []project.StackDefinitionInputVariable
	var stackOutputsDef []project.StackDefinitionOutputVariable
	for _, input := range stackJson.Inputs {
		nameVar := input.Name
		name := &nameVar
		inputTypeVar := input.Type
		inputType := &inputTypeVar
		requiredVar := input.Required
		required := &requiredVar
		var inputDefault interface{}
		if _, exists := doNotOverrideInputs["stack"][input.Name]; !exists {
			if stackConfig.Inputs != nil {
				if val, ok := stackConfig.Inputs[input.Name]; ok {
					inputDefault = val
				}
			}
			if inputDefault == nil {
				if input.Default == nil {
					inputDefault = "__NULL__"
				} else {
					inputDefault = input.Default
				}
			}
		} else {
			inputDefault = doNotOverrideInputs["stack"][input.Name]
		}
		inputDefault = convertSliceToString(inputDefault)
		descriptionVar := input.Description
		description := &descriptionVar
		hiddenVar := input.Hidden
		hidden := &hiddenVar

		stackInputsDef = append(stackInputsDef, project.StackDefinitionInputVariable{
			Name:        name,
			Type:        inputType,
			Required:    required,
			Default:     inputDefault,
			Description: description,
			Hidden:      hidden,
		})
	}

	for _, output := range stackJson.Outputs {
		nameVar := output.Name
		name := &nameVar
		valueVar := output.Value
		value := &valueVar
		stackOutputsDef = append(stackOutputsDef, project.StackDefinitionOutputVariable{
			Name:  name,
			Value: value,
		})
	}

	return stackInputsDef, stackOutputsDef
}

// readCatalogConfig reads the catalog configuration from a JSON file.
// This function reads and unmarshals the catalog configuration, and identifies the product and flavor indices based on the stack configuration.
func readCatalogConfig(catalogJsonPath string, stackConfig *ConfigDetails, errorMessages *[]string) (CatalogJson, int, int, error) {
	jsonFile, err := os.ReadFile(catalogJsonPath)
	duplicateErrorMessages := []string{}

	if err != nil {
		log.Println("Error reading catalog JSON file:", err)
		return CatalogJson{}, 0, 0, err
	}

	var catalogConfig CatalogJson
	err = json.Unmarshal(jsonFile, &catalogConfig)
	if err != nil {
		log.Println("Error unmarshaling catalog JSON:", err)
		return CatalogJson{}, 0, 0, err
	}

	var catalogProductIndex int
	if stackConfig.CatalogProductName == "" {
		catalogProductIndex = 0
	} else {
		for i, product := range catalogConfig.Products {
			if product.Name == stackConfig.CatalogProductName {
				catalogProductIndex = i
				break
			}
		}
	}

	var catalogFlavorIndex int
	if stackConfig.CatalogFlavorName == "" {
		catalogFlavorIndex = 0
	} else {
		for i, flavor := range catalogConfig.Products[catalogProductIndex].Flavors {
			if flavor.Name == stackConfig.CatalogFlavorName {
				catalogFlavorIndex = i
				break
			}
		}
	}

	catalogInputNames := make(map[string]bool)
	productName := catalogConfig.Products[catalogProductIndex].Name
	flavorName := catalogConfig.Products[catalogProductIndex].Flavors[catalogFlavorIndex].Name
	for _, input := range catalogConfig.Products[catalogProductIndex].Flavors[catalogFlavorIndex].Configuration {
		if _, exists := catalogInputNames[input.Key]; exists {
			duplicateErrorMessages = append(duplicateErrorMessages, fmt.Sprintf("duplicate catalog input variable found in product '%s', flavor '%s': %s", productName, flavorName, input.Key))
		} else {
			catalogInputNames[input.Key] = true
		}
	}
	if len(duplicateErrorMessages) > 0 {

		*errorMessages = append(*errorMessages, strings.Join(duplicateErrorMessages, "\n"))
	}

	return catalogConfig, catalogProductIndex, catalogFlavorIndex, nil
}

// updateInputsFromCatalog updates stack inputs based on the catalog configuration.
// This function updates the stack configuration's inputs using the values from the catalog configuration, respecting the precedence rules.
func updateInputsFromCatalog(stackConfig *ConfigDetails, catalogConfig CatalogJson, catalogProductIndex int, catalogFlavorIndex int, doNotOverrideInputs map[string]map[string]interface{}) {
	for _, input := range catalogConfig.Products[catalogProductIndex].Flavors[catalogFlavorIndex].Configuration {
		if stackConfig.Inputs == nil {
			stackConfig.Inputs = make(map[string]interface{})
		}
		var inputDefault interface{}

		if _, exists := doNotOverrideInputs["stack"][input.Key]; !exists {
			if stackConfig.Inputs != nil {
				if val, ok := stackConfig.Inputs[input.Key]; ok {
					inputDefault = val
				} else {
					inputDefault = input.DefaultValue
				}
			}
		} else {
			inputDefault = doNotOverrideInputs["stack"][input.Key]
		}

		// Skip updating if the default value is nil
		if inputDefault == nil {
			continue
		}

		inputDefault = convertSliceToString(inputDefault)

		found := false
		for i := range stackConfig.StackDefinition.Inputs {
			if *stackConfig.StackDefinition.Inputs[i].Name == input.Key {
				switch *stackConfig.StackDefinition.Inputs[i].Type {
				case "int":
					var int64Val int64
					if val, ok := inputDefault.(int); ok {
						int64Val = int64(val)
						stackConfig.StackDefinition.Inputs[i].Default = &int64Val
					} else if val, ok := inputDefault.(float64); ok {
						int64Val = int64(val)
						stackConfig.StackDefinition.Inputs[i].Default = &int64Val
					} else if val, err := strconv.ParseInt(inputDefault.(string), 10, 64); err == nil {
						int64Val = val
						stackConfig.StackDefinition.Inputs[i].Default = &int64Val
					} else {
						stackConfig.StackDefinition.Inputs[i].Default = &val
					}
				case "string", "password", "array":
					if val, ok := inputDefault.(string); ok {
						stackConfig.StackDefinition.Inputs[i].Default = &val
					}
				case "bool", "boolean":
					if val, ok := inputDefault.(bool); ok {
						stackConfig.StackDefinition.Inputs[i].Default = &val
					} else if val, err := strconv.ParseBool(inputDefault.(string)); err == nil {
						stackConfig.StackDefinition.Inputs[i].Default = &val
					}
				}
				found = true
				break
			}
		}
		if !found {
			stackConfig.StackDefinition.Inputs = append(stackConfig.StackDefinition.Inputs, project.StackDefinitionInputVariable{
				Name:        core.StringPtr(input.Key),
				Type:        core.StringPtr(input.Type),
				Required:    core.BoolPtr(input.Required),
				Default:     &input.DefaultValue,
				Description: core.StringPtr(input.Description),
				Hidden:      core.BoolPtr(false),
			})
		}
	}
}

// validateCatalogInputsInStackDefinition validates that all catalog inputs exist in the stack definition and their types match.
// This function ensures that each input defined in the catalog configuration is also present in the stack definition with the correct type,
// thereby ensuring consistency between the catalog and stack configurations.
func validateCatalogInputsInStackDefinition(stackJson Stack, catalogConfig CatalogJson, catalogProductIndex, catalogFlavorIndex int, errorMessages *[]string) {
	catalogInputs := catalogConfig.Products[catalogProductIndex].Flavors[catalogFlavorIndex].Configuration
	productName := catalogConfig.Products[catalogProductIndex].Name
	flavorName := catalogConfig.Products[catalogProductIndex].Flavors[catalogFlavorIndex].Name
	typeMismatches := []string{}
	defaultTypeMismatches := []string{}
	extraInputs := []string{}

	// Iterate over each catalog input and check if it exists in the stack definition and if the types match
	for _, catalogInput := range catalogInputs {
		// Skip the ibmcloud_api_key input as it may not part of the stack definition, but is always valid
		if catalogInput.Key == "ibmcloud_api_key" {
			continue
		}
		found := false
		for _, stackInput := range stackJson.Inputs {
			if catalogInput.Key == stackInput.Name {
				found = true
				expectedType := convertGoTypeToExpectedType(stackInput.Type)
				if !isValidType(catalogInput.Type, expectedType) {
					typeMismatches = append(typeMismatches, fmt.Sprintf("catalog configuration type mismatch in product '%s', flavor '%s': %s expected type: %s, got: %s", productName, flavorName, catalogInput.Key, expectedType, catalogInput.Type))
				}
				// Check if the default value type matches the expected type
				if catalogInput.DefaultValue != nil {
					defaultValueType := reflect.TypeOf(catalogInput.DefaultValue).String()
					expectedDefaultValueType := convertGoTypeToExpectedType(defaultValueType)
					if !isValidType(expectedType, expectedDefaultValueType) {
						defaultTypeMismatches = append(defaultTypeMismatches, fmt.Sprintf("catalog configuration default value type mismatch in product '%s', flavor '%s': %s expected type: %s, got: %s", productName, flavorName, catalogInput.Key, expectedType, expectedDefaultValueType))
					}
				}
				break
			}
		}
		if !found {
			extraInputs = append(extraInputs, fmt.Sprintf("extra catalog input variable not found in stack definition in product '%s', flavor '%s': %s", productName, flavorName, catalogInput.Key))
		}
	}

	if len(typeMismatches) > 0 {
		*errorMessages = append(*errorMessages, strings.Join(typeMismatches, "\n"))
	}

	if len(defaultTypeMismatches) > 0 {
		*errorMessages = append(*errorMessages, strings.Join(defaultTypeMismatches, "\n"))
	}

	if len(extraInputs) > 0 {
		*errorMessages = append(*errorMessages, strings.Join(extraInputs, "\n"))
	}
}

// convertGoTypeToExpectedType converts Go types to the expected type names as defined in catalog json.
func convertGoTypeToExpectedType(goType string) string {
	switch goType {
	case "string":
		return "string"
	case "int", "int64", "float32", "float64":
		return "int"
	case "[]interface {}":
		return "array"
	case "map[string]interface {}":
		return "object"
	case "bool":
		return "bool"
	default:
		return goType
	}
}

// isValidType checks if the provided type is valid considering additional valid types
func isValidType(expectedType, actualType string) bool {
	validTypes := map[string][]string{
		"password": {"password", "string"},
		"boolean":  {"boolean", "bool"},
		"array":    {"array"},
		"object":   {"object"},
		"int":      {"int", "int64", "float32", "float64"},
		"string":   {"string"},
		"float":    {"float32", "float64"},
		// Add more type mappings as needed
	}

	if valid, exists := validTypes[expectedType]; exists {
		for _, t := range valid {
			if t == actualType {
				return true
			}
		}
		return false
	}
	return expectedType == actualType
}

// sortInputsByName sorts the stack inputs by their name.
// This function ensures that the stack inputs are sorted by name for consistent ordering.
func sortInputsByName(stackConfig *ConfigDetails) {
	sort.Slice(stackConfig.StackDefinition.Inputs, func(i, j int) bool {
		if stackConfig.StackDefinition.Inputs[i].Name == nil || stackConfig.StackDefinition.Inputs[j].Name == nil {
			return false
		}
		return *stackConfig.StackDefinition.Inputs[i].Name < *stackConfig.StackDefinition.Inputs[j].Name
	})
}

// GetStackMembers gets the members of a stack
func (infoSvc *CloudInfoService) GetStackMembers(stackConfig *ConfigDetails) (members []*project.ProjectConfig, err error) {
	members = make([]*project.ProjectConfig, 0)
	if stackConfig.Members == nil {
		return members, nil
	}
	for _, member := range stackConfig.MemberConfigs {
		config, _, err := infoSvc.GetConfig(&ConfigDetails{
			ProjectID: stackConfig.ProjectID,
			ConfigID:  *member.ConfigID,
		})
		if err != nil {
			return nil, err
		}
		members = append(members, config)
	}
	return members, nil
}

// SyncConfig syncs a project config with schematics
func (infoSvc *CloudInfoService) SyncConfig(projectID string, configID string) (response *core.DetailedResponse, err error) {

	syncOptions := &project.SyncConfigOptions{
		ProjectID: core.StringPtr(projectID),
		ID:        core.StringPtr(configID),
	}

	return infoSvc.projectsService.SyncConfig(syncOptions)

}

// LookupMemberNameByID looks up the member name using the member ID from the stackDetails definition member list.
func (infoSvc *CloudInfoService) LookupMemberNameByID(stackDetails *project.ProjectConfig, memberID string) (string, error) {
	if stackDetails == nil || stackDetails.Definition == nil {
		return "", fmt.Errorf("invalid stack details or definition")
	}
	def := stackDetails.Definition.(*project.ProjectConfigDefinitionResponse)
	for _, member := range def.Members {
		if member.ConfigID != nil && *member.ConfigID == memberID {
			return *member.Name, nil
		}
	}
	// api lookup the name
	member, _, err := infoSvc.GetConfig(&ConfigDetails{
		ProjectID: *stackDetails.Project.ID,
		ConfigID:  memberID,
	})
	if err != nil {
		return "", fmt.Errorf("member ID %s not found in stack details", memberID)
	}
	return *member.Definition.(*project.ProjectConfigDefinitionResponse).Name, nil
}

// GetSchematicsJobLogsForMember gets the schematics job logs for a member
func (infoSvc *CloudInfoService) GetSchematicsJobLogsForMember(member *project.ProjectConfig, memberName string, projectRegion string) (details string, terraformLogs string) {
	var logMessage strings.Builder
	var terraformLogMessage strings.Builder

	// determine schematics geo location from project region
	schematicsLocation := projectRegion[0:2]

	logMessage.WriteString(fmt.Sprintf("Schematics job logs for member: %s", memberName))

	if member.Schematics != nil && member.Schematics.WorkspaceCrn != nil {
		schematicsWorkspaceCrn := member.Schematics.WorkspaceCrn
		logMessage.WriteString(fmt.Sprintf(", Schematics Workspace CRN: %s", *schematicsWorkspaceCrn))
	} else {
		logMessage.WriteString(", Unknown Schematics Workspace CRN")
	}

	if member.LastUndeployed != nil {
		logMessage.WriteString(fmt.Sprintf("\n\t(%s) failed Undeployment", memberName))
		if member.LastUndeployed.Job != nil {
			jobID := "unknown"
			if member.LastUndeployed.Job.ID != nil {
				jobID = *member.LastUndeployed.Job.ID
			}
			logMessage.WriteString(fmt.Sprintf(", Schematics Undeploy Job ID: %s", jobID))

			if member.NeedsAttentionState != nil {
				var url string
				for _, state := range member.NeedsAttentionState {
					if state.ActionURL != nil {
						url = *state.ActionURL
						break
					}
				}
				logMessage.WriteString(fmt.Sprintf("\nSchematics workspace URL: %s", url))

				if jobID != "unknown" {
					jobURL := strings.Split(url, "/jobs?region=")[0]
					jobURL = fmt.Sprintf("%s/log/%s", jobURL, jobID)
					logMessage.WriteString(fmt.Sprintf("\nSchematics Job URL: %s", jobURL))
					logs, errGetLogs := infoSvc.GetSchematicsJobLogsText(jobID, schematicsLocation)
					if errGetLogs != nil {
						terraformLogMessage.WriteString(fmt.Sprintf("\nError getting job logs for Job ID: %s member: %s, error: %s", jobID, memberName, errGetLogs))
					} else {
						terraformLogMessage.WriteString(fmt.Sprintf("\nJob logs for Job ID: %s member: %s\n%s", jobID, memberName, logs))
					}
				}
			}
			if member.LastUndeployed.Result != nil {
				logMessage.WriteString(fmt.Sprintf("\n\t(%s) Undeployment result: %s", memberName, *member.LastUndeployed.Result))
			} else {
				logMessage.WriteString(fmt.Sprintf("\n\t(%s) Undeployment result: nil", memberName))
			}
			if member.LastUndeployed.Job.Summary != nil && member.LastUndeployed.Job.Summary.DestroySummary != nil && member.LastUndeployed.Job.Summary.DestroySummary.Resources != nil && member.LastUndeployed.Job.Summary.DestroySummary.Resources.Failed != nil {
				for _, failedResource := range member.LastUndeployed.Job.Summary.DestroySummary.Resources.Failed {
					logMessage.WriteString(fmt.Sprintf("\n\t(%s) Failed resource: %s", memberName, failedResource))
				}
			} else {
				logMessage.WriteString(fmt.Sprintf("\n\t(%s) failed Deployment, no failed resources returned", memberName))
			}

			if member.LastUndeployed.Job.Summary != nil && member.LastUndeployed.Job.Summary.DestroyMessages != nil && member.LastUndeployed.Job.Summary.DestroyMessages.ErrorMessages != nil {
				for _, applyError := range member.LastUndeployed.Job.Summary.DestroyMessages.ErrorMessages {
					logMessage.WriteString(fmt.Sprintf("\n\t(%s) Deployment error:\n", memberName))
					for key, value := range applyError.GetProperties() {
						logMessage.WriteString(fmt.Sprintf("\t\t%s: %v\n", key, value))
					}
				}
			} else {
				logMessage.WriteString(fmt.Sprintf("\n\t(%s) failed Deployment, no failed plan messages returned", memberName))
			}
		}
	} else if member.LastDeployed != nil {
		jobID := "unknown"
		if member.LastDeployed.Job != nil && member.LastDeployed.Job.ID != nil {
			jobID = *member.LastDeployed.Job.ID
		}
		logMessage.WriteString(fmt.Sprintf(", Schematics Deploy Job ID: %s", jobID))

		if member.NeedsAttentionState != nil {
			var url string
			for _, state := range member.NeedsAttentionState {
				if state.ActionURL != nil {
					url = *state.ActionURL
					break
				}
			}
			logMessage.WriteString(fmt.Sprintf("\nSchematics workspace URL: %s", url))

			if jobID != "unknown" {
				jobURL := strings.Split(url, "/jobs?region=")[0]
				jobURL = fmt.Sprintf("%s/log/%s", jobURL, jobID)
				logMessage.WriteString(fmt.Sprintf("\nSchematics Job URL: %s", jobURL))
				logs, errGetLogs := infoSvc.GetSchematicsJobLogsText(jobID, schematicsLocation)
				if errGetLogs != nil {
					terraformLogMessage.WriteString(fmt.Sprintf("\nError getting job logs for Job ID: %s member: %s, error: %s", jobID, memberName, errGetLogs))
				} else {
					terraformLogMessage.WriteString(fmt.Sprintf("\nJob logs for Job ID: %s member: %s\n%s", jobID, memberName, logs))
				}
			}
		}
		if member.LastDeployed.Result != nil {
			logMessage.WriteString(fmt.Sprintf("\n\t(%s) Deployment result: %s", memberName, *member.LastDeployed.Result))
		} else {
			logMessage.WriteString(fmt.Sprintf("\n\t(%s) Deployment result: nil", memberName))
		}
		if member.LastDeployed.Job.Summary != nil && member.LastDeployed.Job.Summary.ApplySummary != nil && member.LastDeployed.Job.Summary.ApplySummary.FailedResources != nil {
			for _, failedResource := range member.LastDeployed.Job.Summary.ApplySummary.FailedResources {
				logMessage.WriteString(fmt.Sprintf("\n\t(%s) Failed resource: %s", memberName, failedResource))
			}
		} else {
			logMessage.WriteString(fmt.Sprintf("\n\t(%s) failed Deployment, no failed resources returned", memberName))
		}

		if member.LastDeployed.Job.Summary != nil && member.LastDeployed.Job.Summary.ApplyMessages != nil && member.LastDeployed.Job.Summary.ApplyMessages.ErrorMessages != nil {
			for _, applyError := range member.LastDeployed.Job.Summary.ApplyMessages.ErrorMessages {
				logMessage.WriteString(fmt.Sprintf("\n\t(%s) Deployment error:\n", memberName))
				for key, value := range applyError.GetProperties() {
					logMessage.WriteString(fmt.Sprintf("\t\t%s: %v\n", key, value))
				}
			}
		} else {
			logMessage.WriteString(fmt.Sprintf("\n\t(%s) failed Deployment, no failed plan messages returned", memberName))
		}
	} else if member.LastValidated != nil {
		jobID := "unknown"
		if member.LastValidated.Job != nil && member.LastValidated.Job.ID != nil {
			jobID = *member.LastValidated.Job.ID
		}
		logMessage.WriteString(fmt.Sprintf(", Schematics Validate Job ID: %s", jobID))

		if member.NeedsAttentionState != nil {
			var url string
			for _, state := range member.NeedsAttentionState {
				if state.ActionURL != nil {
					url = *state.ActionURL
					break
				}
			}
			logMessage.WriteString(fmt.Sprintf("\nSchematics workspace URL: %s", url))

			if jobID != "unknown" {
				jobURL := strings.Split(url, "/jobs?region=")[0]
				jobURL = fmt.Sprintf("%s/log/%s", jobURL, jobID)
				logMessage.WriteString(fmt.Sprintf("\nSchematics Job URL: %s", jobURL))
				logs, errGetLogs := infoSvc.GetSchematicsJobLogsText(jobID, schematicsLocation)
				if errGetLogs != nil {
					terraformLogMessage.WriteString(fmt.Sprintf("\nError getting job logs for Job ID: %s member: %s, error: %s", jobID, memberName, errGetLogs))
				} else {
					terraformLogMessage.WriteString(fmt.Sprintf("\nJob logs for Job ID: %s member: %s\n%s", jobID, memberName, logs))
				}
			}
		}

		if member.LastValidated.Result != nil {
			logMessage.WriteString(fmt.Sprintf("\n\t(%s) Validation result: %s", memberName, *member.LastValidated.Result))
		} else {
			logMessage.WriteString(fmt.Sprintf("\n\t(%s) Validation result: nil", memberName))
		}

		if member.LastValidated.Job.Summary != nil && member.LastValidated.Job.Summary.PlanSummary != nil && member.LastValidated.Job.Summary.PlanSummary.FailedResources != nil {
			for _, failedResource := range member.LastValidated.Job.Summary.PlanSummary.FailedResources {
				logMessage.WriteString(fmt.Sprintf("\n\t(%s) Failed resource: %s", memberName, failedResource))
			}
		} else {
			logMessage.WriteString(fmt.Sprintf("\n\t(%s) failed Validation, no failed resources returned", memberName))
		}

		if member.LastValidated.Job.Summary != nil && member.LastValidated.Job.Summary.PlanMessages != nil && member.LastValidated.Job.Summary.PlanMessages.ErrorMessages != nil {
			for _, planError := range member.LastValidated.Job.Summary.PlanMessages.ErrorMessages {
				logMessage.WriteString(fmt.Sprintf("\n\t(%s) Validation error:\n", memberName))
				for key, value := range planError.GetProperties() {
					logMessage.WriteString(fmt.Sprintf("\t\t%s: %v\n", key, value))
				}
			}
		} else {
			logMessage.WriteString(fmt.Sprintf("\n\t(%s) No failed plan messages returned", memberName))
		}
	}
	return logMessage.String(), terraformLogMessage.String()
}

// ProjectsMemberIsDeploying checks if a member is in a state that indicates it is currently deploying.
func ProjectsMemberIsDeploying(member *project.ProjectConfig) bool {
	if member.State == nil {
		return false
	}
	return *member.State == project.ProjectConfig_State_Deploying ||
		*member.State == project.ProjectConfig_State_Validating ||
		*member.State == project.ProjectConfig_State_Approved ||
		*member.State == project.ProjectConfig_State_Validated
}

// ProjectsMemberIsUndeployed checks if a member is in an undeployed state.
func ProjectsMemberIsUndeployed(member *project.ProjectConfig) bool {
	if member.State == nil {
		return false
	}
	return *member.State == project.ProjectConfig_State_Approved ||
		*member.State == project.ProjectConfig_State_Draft ||
		*member.State == project.ProjectConfig_State_Validated ||
		*member.State == project.ProjectConfig_State_Deleted

}

// ProjectsMemberIsDeployFailed checks if a member is in a state that indicates deployment failure.
func ProjectsMemberIsDeployFailed(member *project.ProjectConfig) bool {
	if member.State == nil {
		return false
	}
	return *member.State == project.ProjectConfig_State_DeployingFailed ||
		*member.State == project.ProjectConfig_State_ValidatingFailed
}

// ArePipelineActionsRunning checks if any pipeline actions are running for the given stack
func (infoSvc *CloudInfoService) ArePipelineActionsRunning(stackConfig *ConfigDetails) (bool, error) {
	stackMembers, err := infoSvc.GetStackMembers(stackConfig)
	if err != nil {
		return false, err
	}

	for _, member := range stackMembers {
		if member.State != nil && (*member.State == project.ProjectConfig_State_Deploying || *member.State == project.ProjectConfig_State_Validating || *member.State == project.ProjectConfig_State_Undeploying) {
			return true, nil
		}
	}
	return false, nil
}

// convertSliceToString
// The `convertSliceToString` function takes an input of any type and checks if it is a slice.
// If the input is a slice, it recursively converts each element of the slice to a string.
// If any element within the slice is itself a slice, the function will recursively convert that nested slice to a string as well.
// The function returns a string representation of the slice, with each element enclosed in double quotes and separated by commas.
// If the input is not a slice, it returns the input as is.
//
// Parameters:
// - `input` (interface\{\}): The input value to be converted. This can be of any type.
//
// Returns:
// - `interface\{\}`: A string representation of the input if it is a slice, otherwise the input itself.
func convertSliceToString(input interface{}) interface{} {
	if reflect.TypeOf(input).Kind() == reflect.Slice {
		slice := reflect.ValueOf(input)
		if slice.Len() == 0 {
			return "[]"
		}
		elements := make([]string, slice.Len())
		for i := 0; i < slice.Len(); i++ {
			element := slice.Index(i).Interface()
			if reflect.TypeOf(element).Kind() == reflect.Slice {
				elements[i] = fmt.Sprintf("%v", convertSliceToString(element))
			} else {
				elements[i] = fmt.Sprintf("\"%v\"", element)
			}
		}
		return fmt.Sprintf("[%s]", strings.Join(elements, ", "))
	}
	return input
}
