package cloudinfo

import (
	"encoding/json"
	"github.com/IBM/go-sdk-core/v5/core"
	project "github.com/IBM/project-go-sdk/projectv1"
	"log"
	"math/rand"
	"os"
	"strconv"
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
	return infoSvc.CreateDefaultProjectWithConfigs(name, description, resourceGroup, []project.ProjectConfigPrototype{})
}

// CreateDefaultProjectWithConfigs creates a default project with the given name and description and the given configs
// name: the name of the project
// description: the description of the project
// resourceGroup: the resource group to use
// configs: the project configs to use
// Returns an error if one occurs
// Chooses a random location
// Delete all resources on delete
// Drift detection is enabled
// Headers are empty
func (infoSvc *CloudInfoService) CreateDefaultProjectWithConfigs(name string, description string, resourceGroup string, configs []project.ProjectConfigPrototype) (result *project.Project, response *core.DetailedResponse, err error) {
	validRegions := []string{"us-south", "us-east", "eu-gb", "eu-de"}

	// Generate a random index within the range of the slice
	randomIndex := rand.Intn(len(validRegions))

	// Select the location at the random index
	location := validRegions[randomIndex]
	projectOptions := &project.CreateProjectOptions{
		Definition: &project.ProjectPrototypeDefinition{
			Name:              &name,
			DestroyOnDelete:   core.BoolPtr(true),
			Description:       &description,
			MonitoringEnabled: core.BoolPtr(true),
		},
		Location:      &location,
		ResourceGroup: &resourceGroup,
		Configs:       configs,
		Environments:  []project.EnvironmentPrototype{},
		Headers:       map[string]string{},
	}

	return infoSvc.projectsService.CreateProject(projectOptions)

}

// CreateDefaultProjectWithConfigsFromFile creates a default project with the given name and description and the configs from the given file
// name: the name of the project
// description: the description of the project
// resourceGroup: the resource group to use
// configFile: the file containing the project configs
// Returns an error if one occurs
// Chooses a random location
// Delete all resources on delete
// Drift detection is enabled
// Headers are empty
func (infoSvc *CloudInfoService) CreateDefaultProjectWithConfigsFromFile(name string, description string, resourceGroup string, configFile string) (result *project.Project, response *core.DetailedResponse, err error) {
	// Read the JSON file
	jsonFile, err := os.ReadFile(configFile)
	if err != nil {
		log.Println("Error reading JSON file:", err)
		return nil, nil, err
	}

	// Create a new variable of type ProjectConfigPrototype
	var configs []project.ProjectConfigPrototype

	// Unmarshal the JSON data into the config variable
	err = json.Unmarshal(jsonFile, &configs)
	if err != nil {
		log.Println("Error unmarshaling JSON:", err)
		return nil, nil, err
	}

	return infoSvc.CreateDefaultProjectWithConfigs(name, description, resourceGroup, configs)
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

func (infoSvc *CloudInfoService) CreateConfig(projectID string, name string, description string, stackLocatorID string) (result *project.ProjectConfig, response *core.DetailedResponse, err error) {
	authMethod := project.ProjectConfigAuth_Method_ApiKey
	createConfigOptions := &project.CreateConfigOptions{
		ProjectID: &projectID,
		Definition: &project.ProjectConfigDefinitionPrototype{
			Description: &description,
			Name:        &name,
			LocatorID:   &stackLocatorID,
			Authorizations: &project.ProjectConfigAuth{
				ApiKey: &infoSvc.authenticator.ApiKey,
				Method: &authMethod,
			},
		},
	}
	return infoSvc.projectsService.CreateConfig(createConfigOptions)
}

func (infoSvc *CloudInfoService) CreateDaConfig(projectID string, locatorID string, name string, description string, authorizations project.ProjectConfigAuth, inputs map[string]interface{}, settings map[string]interface{}) (result *project.ProjectConfig, response *core.DetailedResponse, err error) {
	createConfigOptions := &project.CreateConfigOptions{
		ProjectID: &projectID,
		Definition: &project.ProjectConfigDefinitionPrototype{
			Description:    &description,
			Name:           &name,
			LocatorID:      &locatorID,
			Authorizations: &authorizations,
			Inputs:         inputs,
			Settings:       settings,
		},
	}
	return infoSvc.projectsService.CreateConfig(createConfigOptions)
}

func (infoSvc *CloudInfoService) CreateConfigFromCatalogJson(projectID string, catalogJsonPath string, stackLocatorID string) (result *project.ProjectConfig, response *core.DetailedResponse, err error) {
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

	return infoSvc.CreateConfig(projectID, catalogConfig.Products[0].Name, catalogConfig.Products[0].Label, stackLocatorID)
}

func (infoSvc *CloudInfoService) AddStackFromConfig(projectID string, configID string, stackConfig *project.StackDefinitionBlockPrototype) (result *project.StackDefinition, response *core.DetailedResponse, err error) {

	createStackDefinitionOptions := &project.CreateStackDefinitionOptions{
		ProjectID:       &projectID,
		ID:              &configID,
		StackDefinition: stackConfig,
	}

	return infoSvc.projectsService.CreateStackDefinition(createStackDefinitionOptions)
}

func (infoSvc *CloudInfoService) CreateNewStack(projectID string, stackName string, stackDescription string, stackConfig *project.StackDefinitionBlockPrototype, stackMembers []project.StackConfigMember) (result *project.ProjectConfig, response *core.DetailedResponse, err error) {

	//convert inputs to map[string]interface{}
	var stackInputs = make(map[string]interface{})
	for _, input := range stackConfig.Inputs {
		stackInputs[*input.Name] = input.Default
	}

	createStackDefinitionOptions := &project.ProjectConfigDefinitionPrototypeStackConfigDefinitionProperties{
		Description: &stackDescription,
		Name:        &stackName,
		Inputs:      stackInputs,
		Members:     stackMembers,
	}
	createConfigOptions := infoSvc.projectsService.NewCreateConfigOptions(
		projectID,
		createStackDefinitionOptions,
	)

	return infoSvc.projectsService.CreateConfig(createConfigOptions)
}
func (infoSvc *CloudInfoService) UpdateStackFromConfig(projectID string, configID string, stackConfig *project.StackDefinitionBlockPrototype) (result *project.StackDefinition, response *core.DetailedResponse, err error) {

	updateStackDefinitionOptions := &project.UpdateStackDefinitionOptions{
		ProjectID:       &projectID,
		ID:              &configID,
		StackDefinition: stackConfig,
	}
	return infoSvc.projectsService.UpdateStackDefinition(updateStackDefinitionOptions)
}

func (infoSvc *CloudInfoService) AddStackFromFile(projectID string, configFilePath string, catalogJsonPath string, stackLocatorID string) (result *project.StackDefinition, response *core.DetailedResponse, err error) {
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

	projectConfig, _, configErr := infoSvc.CreateConfigFromCatalogJson(projectID, catalogJsonPath, stackLocatorID)
	if configErr != nil {
		log.Println("Error creating config from catalog JSON:", configErr)
		return nil, nil, configErr
	}

	// Create a new variable of type ProjectConfigPrototype
	var stackConfig project.StackDefinitionBlockPrototype

	// Unmarshal the JSON data into the config variable
	err = json.Unmarshal(jsonFile, &stackConfig)
	if err != nil {
		log.Fatal("Error unmarshaling JSON:", err)
		return nil, nil, err
	}

	return infoSvc.UpdateStackFromConfig(projectID, *projectConfig.ID, &stackConfig)

}

func (infoSvc *CloudInfoService) ValidateConfig(projectID string, configID string) (result *project.ProjectConfigVersion, response *core.DetailedResponse, err error) {
	validateConfigOptions := &project.ValidateConfigOptions{
		ProjectID: &projectID,
		ID:        &configID,
	}
	return infoSvc.projectsService.ValidateConfig(validateConfigOptions)
}

func (infoSvc *CloudInfoService) GetConfigVersion(projectID string, configID string, version int64) (result *project.ProjectConfigVersion, response *core.DetailedResponse, err error) {

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

func (infoSvc *CloudInfoService) ApproveConfig(projectID string, configID string) (result *project.ProjectConfigVersion, response *core.DetailedResponse, err error) {
	approveOptions := &project.ApproveOptions{
		ProjectID: &projectID,
		ID:        &configID,
	}
	approveOptions.SetComment("Approving the changes by test wrapper")
	return infoSvc.projectsService.Approve(approveOptions)
}

func (infoSvc *CloudInfoService) DeployConfig(projectID string, configID string) (result *project.ProjectConfigVersion, response *core.DetailedResponse, err error) {
	deployConfigOptions := &project.DeployConfigOptions{
		ProjectID: &projectID,
		ID:        &configID,
	}
	return infoSvc.projectsService.DeployConfig(deployConfigOptions)
}

func (infoSvc *CloudInfoService) CreateStackFromConfigFile(projectID string, stackConfigPath string, catalogJsonPath string) (result *project.ProjectConfig, response *core.DetailedResponse, err error) {
	// create configs from members
	// Read the config JSON file
	jsonFile, err := os.ReadFile(stackConfigPath)
	if err != nil {
		log.Println("Error reading config JSON file:", err)
		return nil, nil, err
	}

	var stackConfig Stack

	// Unmarshal the JSON data into the config variable
	err = json.Unmarshal(jsonFile, &stackConfig)
	if err != nil {
		log.Println("Error unmarshaling JSON:", err)
		return nil, nil, err
	}

	// variable to store da configs
	var daConfigMembers []project.StackDefinitionMemberPrototype
	var daStackMembers []project.StackConfigMember
	// loop members in stackConfig and create config
	for _, member := range stackConfig.Members {
		inputs := make(map[string]interface{})
		for _, input := range member.Inputs {
			inputs[input.Name] = input.Value
		}

		daProjectConfig, _, createDaErr := infoSvc.CreateDaConfig(projectID, member.VersionLocator, member.Name, member.Name, project.ProjectConfigAuth{
			ApiKey: &infoSvc.authenticator.ApiKey,
			Method: core.StringPtr(project.ProjectConfigAuth_Method_ApiKey),
		}, inputs, nil)

		if createDaErr != nil {
			log.Println("Error creating config from catalog JSON:", err)
			return nil, nil, err
		}
		// Assuming StackDefinitionMemberInputPrototype has fields Name and Value
		inputPrototypes := make([]project.StackDefinitionMemberInputPrototype, 0, len(inputs))
		for name, _ := range inputs {
			inputPrototypes = append(inputPrototypes, project.StackDefinitionMemberInputPrototype{
				Name: &name,
			})
		}
		// create stack member
		daConfigMembers = append(daConfigMembers, project.StackDefinitionMemberPrototype{
			Inputs: inputPrototypes,
			Name:   &member.Name,
		})
		daStackMembers = append(daStackMembers, project.StackConfigMember{
			Name:     &member.Name,
			ConfigID: daProjectConfig.ID,
		})
	}

	// convert stackConfig inputs to []StackDefinitionInputVariable
	var stackInputs []project.StackDefinitionInputVariable
	for _, input := range stackConfig.Inputs {
		// Assuming StackDefinitionInputVariablePrototype has fields Name and Description
		// create stack input
		stackInputs = append(stackInputs, project.StackDefinitionInputVariable{
			Name:    &input.Name,
			Type:    core.StringPtr(GetProjectInputType(input.Value)),
			Default: input.Value,
		})
	}

	// create stack and add da configs as members
	stackDefinitionBlockPrototype := &project.StackDefinitionBlockPrototype{
		Inputs:  stackInputs,
		Members: daConfigMembers,
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

	return infoSvc.CreateNewStack(projectID, catalogConfig.Products[0].Name, catalogConfig.Products[0].Name, stackDefinitionBlockPrototype, daStackMembers)
}

func GetProjectInputType(input string) (inputType string) {
	// infer the type of the input array,boolean,float,int,number,string,object
	if input == "true" || input == "false" {
		return "boolean"
	}
	// if input starts with { and ends with } then it is an object
	if input[0] == '{' && input[len(input)-1] == '}' {
		return "object"
	}
	// if input starts with [ and ends with ] then it is an array
	if input[0] == '[' && input[len(input)-1] == ']' {
		return "array"
	}
	// if input is a float then it is a float
	if _, err := strconv.ParseFloat(input, 64); err == nil {
		return "float"
	}
	// if input is integer then it is an int
	if _, err := strconv.ParseInt(input, 10, 64); err == nil {
		return "int"
	}

	// default to string
	return "string"
}
