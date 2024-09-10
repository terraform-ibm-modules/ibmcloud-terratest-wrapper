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

// CreateDefaultProject creates a default project with the given name and description
// name: the name of the project
// description: the description of the project
// resourceGroup: the resource group to use
// Returns an error if one occurs
// Chooses a random location
// Delete all resources on delete
// Drift detection is enabled
// Configs and environments are empty
// Headers are empty
func (infoSvc *CloudInfoService) CreateDefaultProject(name string, description string, resourceGroup string) (result *project.Project, response *core.DetailedResponse, err error) {
	return infoSvc.CreateProjectFromConfig(ProjectsConfig{
		ProjectName:        name,
		ProjectDescription: description,
		ResourceGroup:      resourceGroup,
	})
}

// CreateProjectFromConfig creates a project with the given config
// config: the config to use
// Returns an error if one occurs
func (infoSvc *CloudInfoService) CreateProjectFromConfig(config ProjectsConfig) (result *project.Project, response *core.DetailedResponse, err error) {
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

//// CreateDefaultProjectWithConfigs creates a default project with the given name and description and the given configs
//// name: the name of the project
//// description: the description of the project
//// resourceGroup: the resource group to use
//// configs: the project configs to use
//// Returns an error if one occurs
//// Chooses a random location
//// Delete all resources on delete
//// Drift detection is enabled
//// Headers are empty
//func (infoSvc *CloudInfoService) CreateDefaultProjectWithConfigs(name string, description string, resourceGroup string, configs []project.ProjectConfigPrototype) (result *project.Project, response *core.DetailedResponse, err error) {
//	validRegions := []string{"us-south", "us-east", "eu-gb", "eu-de"}
//
//	// Generate a random index within the range of the slice
//	randomIndex := rand.Intn(len(validRegions))
//
//	// Select the location at the random index
//	location := validRegions[randomIndex]
//	projectOptions := &project.CreateProjectOptions{
//		Definition: &project.ProjectPrototypeDefinition{
//			Name:              &name,
//			DestroyOnDelete:   core.BoolPtr(true),
//			Description:       &description,
//			MonitoringEnabled: core.BoolPtr(true),
//		},
//		Location:      &location,
//		ResourceGroup: &resourceGroup,
//		Configs:       configs,
//		Environments:  []project.EnvironmentPrototype{},
//		Headers:       map[string]string{},
//	}
//
//	return infoSvc.projectsService.CreateProject(projectOptions)
//
//}

// // CreateDefaultProjectWithConfigsFromFile creates a default project with the given name and description and the configs from the given file
// // name: the name of the project
// // description: the description of the project
// // resourceGroup: the resource group to use
// // configFile: the file containing the project configs
// // Returns an error if one occurs
// // Chooses a random location
// // Delete all resources on delete
// // Drift detection is enabled
// // Headers are empty
//
//	func (infoSvc *CloudInfoService) CreateDefaultProjectWithConfigsFromFile(name string, description string, resourceGroup string, configFile string) (result *project.Project, response *core.DetailedResponse, err error) {
//		// Read the JSON file
//		jsonFile, err := os.ReadFile(configFile)
//		if err != nil {
//			log.Println("Error reading JSON file:", err)
//			return nil, nil, err
//		}
//
//		// Create a new variable of type ProjectConfigPrototype
//		var configs []project.ProjectConfigPrototype
//
//		// Unmarshal the JSON data into the config variable
//		err = json.Unmarshal(jsonFile, &configs)
//		if err != nil {
//			log.Println("Error unmarshaling JSON:", err)
//			return nil, nil, err
//		}
//
//		return infoSvc.CreateDefaultProjectWithConfigs(name, description, resourceGroup, configs)
//	}
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

func (infoSvc *CloudInfoService) CreateConfig(configDetails ConfigDetails) (result *project.ProjectConfig, response *core.DetailedResponse, err error) {
	authMethod := project.ProjectConfigAuth_Method_ApiKey
	createConfigOptions := &project.CreateConfigOptions{
		ProjectID: &configDetails.ProjectID,
		Definition: &project.ProjectConfigDefinitionPrototype{
			Description: &configDetails.Description,
			Name:        &configDetails.Name,
			LocatorID:   &configDetails.StackLocatorID,
			Authorizations: &project.ProjectConfigAuth{
				ApiKey: &infoSvc.authenticator.ApiKey,
				Method: &authMethod,
			},
		},
	}
	return infoSvc.projectsService.CreateConfig(createConfigOptions)
}

func (infoSvc *CloudInfoService) CreateDaConfig(configDetails ConfigDetails) (result *project.ProjectConfig, response *core.DetailedResponse, err error) {
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

func (infoSvc *CloudInfoService) CreateConfigFromCatalogJson(configDetails ConfigDetails, catalogJsonPath string) (result *project.ProjectConfig, response *core.DetailedResponse, err error) {
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

func (infoSvc *CloudInfoService) AddStackFromConfig(configDetails ConfigDetails) (result *project.StackDefinition, response *core.DetailedResponse, err error) {

	createStackDefinitionOptions := &project.CreateStackDefinitionOptions{
		ProjectID:       &configDetails.ProjectID,
		ID:              &configDetails.ConfigID,
		StackDefinition: configDetails.StackDefinition,
	}

	return infoSvc.projectsService.CreateStackDefinition(createStackDefinitionOptions)
}

func (infoSvc *CloudInfoService) CreateNewStack(stackConfig ConfigDetails) (result *project.StackDefinition, response *core.DetailedResponse, err error) {

	// Create a project config first
	createProjectConfigDefinitionOptions := &project.ProjectConfigDefinitionPrototypeStackConfigDefinitionProperties{
		Description: &stackConfig.Description,
		Name:        &stackConfig.Name,
		Members:     stackConfig.Members,
	}
	createConfigOptions := infoSvc.projectsService.NewCreateConfigOptions(
		stackConfig.ProjectID,
		createProjectConfigDefinitionOptions,
	)
	config, configResp, configErr := infoSvc.projectsService.CreateConfig(createConfigOptions)
	if configErr != nil {
		return nil, configResp, configErr
	}

	// Then apply the stack definition
	stackDefOptions := infoSvc.projectsService.NewCreateStackDefinitionOptions(stackConfig.ProjectID, *config.ID, stackConfig.StackDefinition)

	return infoSvc.projectsService.CreateStackDefinition(stackDefOptions)

}
func (infoSvc *CloudInfoService) UpdateStackFromConfig(stackConfig ConfigDetails) (result *project.StackDefinition, response *core.DetailedResponse, err error) {

	updateStackDefinitionOptions := &project.UpdateStackDefinitionOptions{
		ProjectID:       &stackConfig.ProjectID,
		ID:              &stackConfig.ConfigID,
		StackDefinition: stackConfig.StackDefinition,
	}
	return infoSvc.projectsService.UpdateStackDefinition(updateStackDefinitionOptions)
}

func (infoSvc *CloudInfoService) AddStackFromFile(stackConfig ConfigDetails, configFilePath string, catalogJsonPath string) (result *project.StackDefinition, response *core.DetailedResponse, err error) {
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

func (infoSvc *CloudInfoService) ForceValidateProjectConfig(projectID string, configID string) (result *project.ProjectConfigVersion, response *core.DetailedResponse, err error) {
	validateConfigOptions := &project.ValidateConfigOptions{
		ProjectID: &projectID,
		ID:        &configID,
	}
	return infoSvc.projectsService.ValidateConfig(validateConfigOptions)
}

func (infoSvc *CloudInfoService) ValidateProjectConfig(projectID string, configID string) (result *project.ProjectConfigVersion, response *core.DetailedResponse, err error) {
	configVersion, isDeployed := infoSvc.IsConfigDeployed(projectID, configID)
	if !isDeployed {
		return infoSvc.ForceValidateProjectConfig(projectID, configID)
	} else {
		return configVersion, nil, nil
	}
}

//func (infoSvc *CloudInfoService) IsConfigValidated(projectID string, configID string) (projectConfig *project.ProjectConfigVersion, isValidated bool) {
//	config, _, _ := infoSvc.GetConfig(projectID, configID)
//	configVersion, _, _ := infoSvc.GetProjectConfigVersion(projectID, configID, *config.Version)
//	if config != nil {
//		if config.State != nil {
//			if *config.State == project.ProjectConfig_State_Validated {
//
//				return configVersion, true
//			} else {
//				return configVersion, false
//			}
//		}
//	}
//	return configVersion, false
//
//}

func (infoSvc *CloudInfoService) GetProjectConfigVersion(projectID string, configID string, version int64) (result *project.ProjectConfigVersion, response *core.DetailedResponse, err error) {

	getConfigOptions := infoSvc.projectsService.NewGetConfigVersionOptions(projectID, configID, version)
	return infoSvc.projectsService.GetConfigVersion(getConfigOptions)
}

func (infoSvc *CloudInfoService) GetConfig(projectID string, configID string) (result *project.ProjectConfig, response *core.DetailedResponse, err error) {
	getConfigOptions := &project.GetConfigOptions{
		ProjectID: &projectID,
		ID:        &configID,
	}
	return infoSvc.projectsService.GetConfig(getConfigOptions)
}

func (infoSvc *CloudInfoService) UpdateConfig(projectID string, configID string, configuration project.ProjectConfigDefinitionPatchIntf) (result *project.ProjectConfig, response *core.DetailedResponse, err error) {
	return infoSvc.UpdateConfigWithHeaders(projectID, configID, configuration, map[string]string{})
}

func (infoSvc *CloudInfoService) UpdateConfigWithHeaders(projectID string, configID string, configuration project.ProjectConfigDefinitionPatchIntf, headers map[string]string) (result *project.ProjectConfig, response *core.DetailedResponse, err error) {
	updateConfigOptions := &project.UpdateConfigOptions{
		ProjectID:  &projectID,
		ID:         &configID,
		Definition: configuration,
		Headers:    headers,
	}
	return infoSvc.projectsService.UpdateConfig(updateConfigOptions)
}

//func (infoSvc *CloudInfoService) ForceApproveConfig(projectID string, configID string) (result *project.ProjectConfigVersion, response *core.DetailedResponse, err error) {
//
//	approveOptions := &project.ApproveOptions{
//		ProjectID: &projectID,
//		ID:        &configID,
//	}
//	approveOptions.SetComment("Approving the changes by test wrapper")
//	return infoSvc.projectsService.Approve(approveOptions)
//
//}

//func (infoSvc *CloudInfoService) ApproveConfig(projectID string, configID string) (result *project.ProjectConfigVersion, response *core.DetailedResponse, err error) {
//	// if is validated then approve
//	_, isValidated := infoSvc.IsConfigValidated(projectID, configID)
//	configVersion, isApproved := infoSvc.IsConfigApproved(projectID, configID)
//	if isValidated {
//		if !isApproved {
//			return infoSvc.ForceApproveConfig(projectID, configID)
//		} else {
//			return configVersion, nil, nil
//		}
//	} else {
//		return nil, nil, fmt.Errorf("Config is not validated, cannot approve")
//
//	}
//}

//// IsConfigApproved checks if the config is approved
//func (infoSvc *CloudInfoService) IsConfigApproved(projectID string, configID string) (projectConfig *project.ProjectConfigVersion, isApproved bool) {
//	config, _, _ := infoSvc.GetConfig(projectID, configID)
//	configVersion, _, _ := infoSvc.GetProjectConfigVersion(projectID, configID, *config.Version)
//	if config != nil {
//		if config.State != nil {
//			if *config.State == project.ProjectConfig_State_Approved {
//
//				return configVersion, true
//			} else {
//				return configVersion, false
//			}
//		}
//	}
//	return configVersion, false
//}

// DeployConfig deploy if not already deployed
func (infoSvc *CloudInfoService) DeployConfig(projectID string, configID string) (result *project.ProjectConfigVersion, response *core.DetailedResponse, err error) {
	configVersion, isDeployed := infoSvc.IsConfigDeployed(projectID, configID)
	if !isDeployed {
		return infoSvc.ForceDeployConfig(projectID, configID)
	} else {
		return configVersion, nil, nil

	}
}

// ForceDeployConfig force deploy even if already deployed
func (infoSvc *CloudInfoService) ForceDeployConfig(projectID string, configID string) (result *project.ProjectConfigVersion, response *core.DetailedResponse, err error) {
	deployConfigOptions := &project.DeployConfigOptions{
		ProjectID: &projectID,
		ID:        &configID,
	}
	return infoSvc.projectsService.DeployConfig(deployConfigOptions)
}

// IsConfigDeployed checks if the config is deployed
func (infoSvc *CloudInfoService) IsConfigDeployed(projectID string, configID string) (projectConfig *project.ProjectConfigVersion, isDeployed bool) {
	config, _, err := infoSvc.GetConfig(projectID, configID)
	if err != nil {
		log.Println("Error getting config:", err)
		return nil, false
	}
	configVersion, _, err := infoSvc.GetProjectConfigVersion(projectID, configID, *config.Version)
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

func (infoSvc *CloudInfoService) UndeployConfig(projectID string, configID string) (result *project.ProjectConfigVersion, response *core.DetailedResponse, err error) {
	undeployConfigOptions := &project.UndeployConfigOptions{
		ProjectID: &projectID,
		ID:        &configID,
	}
	return infoSvc.projectsService.UndeployConfig(undeployConfigOptions)
}

func (infoSvc *CloudInfoService) IsUndeploying(projectID string, configID string) (projectConfig *project.ProjectConfigVersion, isUndeploying bool) {
	config, _, err := infoSvc.GetConfig(projectID, configID)
	if err != nil {
		log.Println("Error getting config:", err)
		return nil, false

	}
	configVersion, _, err := infoSvc.GetProjectConfigVersion(projectID, configID, *config.Version)
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

func (infoSvc *CloudInfoService) CreateStackFromConfigFile(stackConfig ConfigDetails, stackConfigPath string, catalogJsonPath string) (result *project.StackDefinition, response *core.DetailedResponse, err error) {

	if stackConfig.Authorizations == nil {
		stackConfig.Authorizations = &project.ProjectConfigAuth{
			ApiKey: &infoSvc.authenticator.ApiKey,
			Method: core.StringPtr(project.ProjectConfigAuth_Method_ApiKey),
		}
	}

	// create configs from members
	// Read the config JSON file
	jsonFile, err := os.ReadFile(stackConfigPath)
	if err != nil {
		log.Println("Error reading config JSON file:", err)
		return nil, nil, err
	}

	var stackJson Stack

	// Unmarshal the JSON data into the config variable
	err = json.Unmarshal(jsonFile, &stackJson)
	if err != nil {
		log.Println("Error unmarshalling JSON:", err)
		return nil, nil, err
	}

	// loop members in stackConfig and create config
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
		daProjectConfig, _, createDaErr := infoSvc.CreateDaConfig(newConfig)

		if createDaErr != nil {
			log.Println("Error creating config from JSON:", createDaErr)
			// log current member details in pretty format
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

	// convert stackConfig inputs to []StackDefinitionInputVariable
	var stackInputsDef []project.StackDefinitionInputVariable
	var stackOutputsDef []project.StackDefinitionOutputVariable
	for _, input := range stackJson.Inputs {
		// Create new variables this avoids the issue of the same address being used for all the variables
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
			// if input.Default is nil, set inputDefault to "__NULL__" so it evaluates correctly by the time it reaches Terraform
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
		// Use the addresses of the new variables
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
		// Create new variables this avoids the issue of the same address being used for all the variables
		nameVar := output.Name
		name := &nameVar
		valueVar := output.Value
		value := &valueVar
		// Use the addresses of the new variables
		stackOutputsDef = append(stackOutputsDef, project.StackDefinitionOutputVariable{
			Name:  name,
			Value: value,
		})
	}

	curStackInputsDef := stackInputsDef

	// create stack
	stackConfig.StackDefinition = &project.StackDefinitionBlockPrototype{
		Inputs:  curStackInputsDef,
		Outputs: stackOutputsDef,
	}

	// load catalog json to get stack name and description
	// Read the catalog JSON file
	jsonFile, err = os.ReadFile(catalogJsonPath)
	if err != nil {
		log.Println("Error reading catalog JSON file:", err)
		return nil, nil, err
	}
	var catalogConfig CatalogJson
	// Unmarshal the JSON data into the config variable
	err = json.Unmarshal(jsonFile, &catalogConfig)
	if err != nil {
		log.Println("Error unmarshaling catalog JSON:", err)
		return nil, nil, err
	}
	// TODO: handle multiple products/flavors
	stackConfig.Name = catalogConfig.Products[0].Name
	stackConfig.Description = catalogConfig.Products[0].Label

	return infoSvc.CreateNewStack(stackConfig)
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
