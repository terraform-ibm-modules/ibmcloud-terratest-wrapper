package cloudinfo

import (
	"encoding/json"
	"fmt"
	"github.com/IBM/go-sdk-core/v5/core"
	project "github.com/IBM/project-go-sdk/projectv1"
	"log"
	"math/rand"
	"os"
	"reflect"
	"strings"
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

	// TODO: override inputs with values from catalogConfig
	configDetails.Name = catalogConfig.Products[0].Name
	configDetails.Description = catalogConfig.Products[0].Label
	return infoSvc.CreateConfig(configDetails)
}

func (infoSvc *CloudInfoService) AddStackFromConfig(configDetails *ConfigDetails) (result *project.StackDefinition, response *core.DetailedResponse, err error) {

	createStackDefinitionOptions := &project.CreateStackDefinitionOptions{
		ProjectID:       &configDetails.ProjectID,
		ID:              &configDetails.ConfigID,
		StackDefinition: configDetails.StackDefinition,
	}

	return infoSvc.projectsService.CreateStackDefinition(createStackDefinitionOptions)
}

func (infoSvc *CloudInfoService) CreateNewStack(stackConfig *ConfigDetails) (result *project.StackDefinition, response *core.DetailedResponse, err error) {

	// Create a project config first
	createProjectConfigDefinitionOptions := &project.ProjectConfigDefinitionPrototypeStackConfigDefinitionProperties{
		Description:       &stackConfig.Description,
		Name:              &stackConfig.Name,
		Members:           stackConfig.Members,
		Authorizations:    stackConfig.Authorizations,
		Inputs:            stackConfig.Inputs,
		Settings:          stackConfig.Settings,
		ComplianceProfile: stackConfig.ComplianceProfile,
		EnvironmentID:     stackConfig.EnvironmentID,
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
	result, response, err = infoSvc.projectsService.CreateStackDefinition(stackDefOptions)
	if err != nil {
		return nil, nil, err
	}
	return result, response, nil

}
func (infoSvc *CloudInfoService) UpdateStackFromConfig(stackConfig *ConfigDetails) (result *project.StackDefinition, response *core.DetailedResponse, err error) {

	updateStackDefinitionOptions := &project.UpdateStackDefinitionOptions{
		ProjectID:       &stackConfig.ProjectID,
		ID:              &stackConfig.ConfigID,
		StackDefinition: stackConfig.StackDefinition,
	}
	return infoSvc.projectsService.UpdateStackDefinition(updateStackDefinitionOptions)
}

func (infoSvc *CloudInfoService) AddStackFromFile(stackConfig *ConfigDetails, configFilePath string, catalogJsonPath string) (result *project.StackDefinition, response *core.DetailedResponse, err error) {
	// Read the config JSON file
	jsonFile, err := os.ReadFile(configFilePath)
	if err != nil {
		log.Println("Error reading config JSON file:", err)
		return nil, nil, err
	}

	// Create a new variable of type Struct
	var config Stack

	// Unmarshal the JSON data into the config variable
	err = json.Unmarshal(jsonFile, &config)
	if err != nil {
		log.Println("Error unmarshaling JSON:", err)
		return nil, nil, err
	}

	projectConfig, _, configErr := infoSvc.CreateConfigFromCatalogJson(stackConfig, catalogJsonPath)
	if configErr != nil {
		log.Println("Error creating config from catalog JSON:", configErr)
		return nil, nil, configErr
	}

	stackConfig.ConfigID = *projectConfig.ID
	if stackConfig.StackDefinition == nil {
		stackConfig.StackDefinition = &project.StackDefinitionBlockPrototype{}
	}

	// Unmarshal the JSON data into the config variable
	err = json.Unmarshal(jsonFile, &stackConfig)
	if err != nil {
		log.Fatal("Error unmarshaling JSON:", err)
		return nil, nil, err
	}

	return infoSvc.UpdateStackFromConfig(stackConfig)

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
func (infoSvc *CloudInfoService) CreateStackFromConfigFile(stackConfig *ConfigDetails, stackConfigPath string, catalogJsonPath string) (stackDefinition *project.StackDefinition, response *core.DetailedResponse, err error) {
	if stackConfig.Authorizations == nil {
		stackConfig.Authorizations = &project.ProjectConfigAuth{
			ApiKey: &infoSvc.authenticator.ApiKey,
			Method: core.StringPtr(project.ProjectConfigAuth_Method_ApiKey),
		}
	}

	jsonFile, err := os.ReadFile(stackConfigPath)
	if err != nil {
		log.Println("Error reading config JSON file:", err)
		return nil, nil, err
	}

	var stackJson Stack
	err = json.Unmarshal(jsonFile, &stackJson)
	if err != nil {
		log.Println("Error unmarshalling JSON:", err)
		return nil, nil, err
	}

	for _, member := range stackJson.Members {
		inputs := make(map[string]interface{})
		for _, input := range member.Inputs {
			val := input.Value
			inputs[input.Name] = val
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
			return nil, nil, createDaErr
		}

		curMemberNameVar := member.Name
		curMemberName := &curMemberNameVar

		curDaProjectConfig := daProjectConfig.ID

		stackConfig.Members = append(stackConfig.Members, project.StackConfigMember{
			Name:     curMemberName,
			ConfigID: curDaProjectConfig,
		})
	}

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

	stackConfig.StackDefinition = &project.StackDefinitionBlockPrototype{
		Inputs:  stackInputsDef,
		Outputs: stackOutputsDef,
	}

	jsonFile, err = os.ReadFile(catalogJsonPath)
	if err != nil {
		log.Println("Error reading catalog JSON file:", err)
		return nil, nil, err
	}
	var catalogConfig CatalogJson
	err = json.Unmarshal(jsonFile, &catalogConfig)
	if err != nil {
		log.Println("Error unmarshaling catalog JSON:", err)
		return nil, nil, err
	}
	stackConfig.Name = catalogConfig.Products[0].Name
	stackConfig.Description = catalogConfig.Products[0].Label

	return infoSvc.CreateNewStack(stackConfig)
}

func (infoSvc *CloudInfoService) GetStackMembers(stackConfig *ConfigDetails) (members []*project.ProjectConfig, err error) {
	members = make([]*project.ProjectConfig, 0)
	if stackConfig.Members == nil {
		return members, nil
	}
	for _, member := range stackConfig.Members {
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
	return "", fmt.Errorf("member ID %s not found in stack details", memberID)
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
