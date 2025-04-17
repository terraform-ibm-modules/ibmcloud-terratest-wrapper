package cloudinfo

import (
	"fmt"

	"github.com/IBM/go-sdk-core/v5/core"
	projects "github.com/IBM/project-go-sdk/projectv1"
	"github.com/stretchr/testify/mock"
)

// Projects API SERVICE MOCK
type ProjectsServiceMock struct {
	mock.Mock
}

type MockConfigsPager struct {
	mock.Mock
}

func (m *MockConfigsPager) HasNext() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockConfigsPager) GetNext() ([]projects.ProjectConfigSummary, error) {
	args := m.Called()
	return args.Get(0).([]projects.ProjectConfigSummary), args.Error(1)
}

func (mock *ProjectsServiceMock) CreateProject(createProjectOptions *projects.CreateProjectOptions) (result *projects.Project, response *core.DetailedResponse, err error) {
	args := mock.Called(createProjectOptions)

	var project *projects.Project
	if args.Get(0) != nil {
		project = args.Get(0).(*projects.Project)
	}

	var detailedResponse *core.DetailedResponse
	if args.Get(1) != nil {
		detailedResponse = args.Get(1).(*core.DetailedResponse)
	}

	return project, detailedResponse, args.Error(2)
}

func (mock *ProjectsServiceMock) GetProject(getProjectOptions *projects.GetProjectOptions) (result *projects.Project, response *core.DetailedResponse, err error) {
	args := mock.Called(getProjectOptions)
	return args.Get(0).(*projects.Project), args.Get(1).(*core.DetailedResponse), args.Error(2)
}

func (mock *ProjectsServiceMock) UpdateProject(updateProjectOptions *projects.UpdateProjectOptions) (result *projects.Project, response *core.DetailedResponse, err error) {
	args := mock.Called(updateProjectOptions)
	return args.Get(0).(*projects.Project), args.Get(1).(*core.DetailedResponse), args.Error(2)
}

func (mock *ProjectsServiceMock) DeleteProject(deleteProjectOptions *projects.DeleteProjectOptions) (result *projects.ProjectDeleteResponse, response *core.DetailedResponse, err error) {
	args := mock.Called(deleteProjectOptions)
	return args.Get(0).(*projects.ProjectDeleteResponse), args.Get(1).(*core.DetailedResponse), args.Error(2)
}

func (mock *ProjectsServiceMock) NewCreateConfigOptions(projectID string, definition projects.ProjectConfigDefinitionPrototypeIntf) *projects.CreateConfigOptions {
	var createConfigOptions *projects.CreateConfigOptions

	switch def := definition.(type) {
	case *projects.ProjectConfigDefinitionPrototype:
		createConfigOptions = &projects.CreateConfigOptions{
			ProjectID: core.StringPtr(projectID),
			Definition: &projects.ProjectConfigDefinitionPrototype{
				Name:           def.Name,
				Description:    def.Description,
				LocatorID:      def.LocatorID,
				Authorizations: def.Authorizations,
				Inputs:         def.Inputs,
				Members:        def.Members,
			},
		}
	case *projects.ProjectConfigDefinitionPrototypeStackConfigDefinitionProperties:
		createConfigOptions = &projects.CreateConfigOptions{
			ProjectID: core.StringPtr(projectID),
			Definition: &projects.ProjectConfigDefinitionPrototypeStackConfigDefinitionProperties{
				Name:           def.Name,
				Description:    def.Description,
				Authorizations: def.Authorizations,
				EnvironmentID:  def.EnvironmentID,
				Members:        def.Members,
			},
		}
	default:
		panic(fmt.Sprintf("unsupported definition type: %T", definition))
	}

	return createConfigOptions
}

func (mock *ProjectsServiceMock) NewConfigsPager(listConfigsOptions *projects.ListConfigsOptions) (*projects.ConfigsPager, error) {
	args := mock.Called(listConfigsOptions)
	mockPager := args.Get(0).(*projects.ConfigsPager)
	return mockPager, args.Error(1)
}

func (mock *ProjectsServiceMock) NewGetConfigVersionOptions(projectID string, id string, version int64) *projects.GetConfigVersionOptions {
	args := mock.Called(projectID, id, version)
	return args.Get(0).(*projects.GetConfigVersionOptions)
}

func (mock *ProjectsServiceMock) GetConfigVersion(getConfigVersionOptions *projects.GetConfigVersionOptions) (result *projects.ProjectConfigVersion, response *core.DetailedResponse, err error) {
	args := mock.Called(getConfigVersionOptions)
	return args.Get(0).(*projects.ProjectConfigVersion), args.Get(1).(*core.DetailedResponse), args.Error(2)
}

func (mock *ProjectsServiceMock) CreateConfig(createConfigOptions *projects.CreateConfigOptions) (result *projects.ProjectConfig, response *core.DetailedResponse, err error) {
	args := mock.Called(createConfigOptions)

	var projectConfig *projects.ProjectConfig
	switch def := createConfigOptions.Definition.(type) {
	case *projects.ProjectConfigDefinitionPrototype:
		projectConfig = &projects.ProjectConfig{
			ID: createConfigOptions.ProjectID,
			Definition: &projects.ProjectConfigDefinitionResponse{
				Name:           def.Name,
				Description:    def.Description,
				LocatorID:      def.LocatorID,
				Authorizations: def.Authorizations,
				Inputs:         def.Inputs,
				Members:        def.Members,
			},
		}
	case *projects.ProjectConfigDefinitionPrototypeStackConfigDefinitionProperties:
		projectConfig = &projects.ProjectConfig{
			ID: createConfigOptions.ProjectID,
			Definition: &projects.ProjectConfigDefinitionResponse{
				Name:           def.Name,
				Description:    def.Description,
				Authorizations: def.Authorizations,
				Inputs:         def.Inputs,
				Members:        def.Members,
			},
		}
	default:
		panic(fmt.Sprintf("unsupported definition type: %T", createConfigOptions.Definition))
	}

	return projectConfig, args.Get(1).(*core.DetailedResponse), args.Error(2)
}
func (mock *ProjectsServiceMock) UpdateConfig(updateConfigOptions *projects.UpdateConfigOptions) (result *projects.ProjectConfig, response *core.DetailedResponse, err error) {
	args := mock.Called(updateConfigOptions)
	return args.Get(0).(*projects.ProjectConfig), args.Get(1).(*core.DetailedResponse), args.Error(2)
}

func (mock *ProjectsServiceMock) GetConfig(getConfigOptions *projects.GetConfigOptions) (result *projects.ProjectConfig, response *core.DetailedResponse, err error) {
	args := mock.Called(getConfigOptions)
	return args.Get(0).(*projects.ProjectConfig), args.Get(1).(*core.DetailedResponse), args.Error(2)
}

func (mock *ProjectsServiceMock) DeleteConfig(deleteConfigOptions *projects.DeleteConfigOptions) (result *projects.ProjectConfigDelete, response *core.DetailedResponse, err error) {
	args := mock.Called(deleteConfigOptions)
	return args.Get(0).(*projects.ProjectConfigDelete), args.Get(1).(*core.DetailedResponse), args.Error(2)
}

func (mock *ProjectsServiceMock) CreateStackDefinition(createStackDefinitionOptions *projects.CreateStackDefinitionOptions) (result *projects.StackDefinition, response *core.DetailedResponse, err error) {
	// Capture the arguments passed during the mock setup
	args := mock.Called(createStackDefinitionOptions)

	// Get the StackDefinitionBlockPrototype from the options
	stackDefinitionBlockPrototype := createStackDefinitionOptions.StackDefinition

	// Dynamically set the Members from the test
	members := args.Get(0).([]projects.StackDefinitionMember)

	// Manually map fields from StackDefinitionBlockPrototype to StackDefinitionBlock
	stackDefinitionBlock := &projects.StackDefinitionBlock{
		Inputs:  stackDefinitionBlockPrototype.Inputs,  // Map Inputs
		Outputs: stackDefinitionBlockPrototype.Outputs, // Map Outputs
		Members: members,                               // Dynamically set Members from args
	}

	// Construct the full StackDefinition response
	stackDefinition := &projects.StackDefinition{
		ID:              createStackDefinitionOptions.ID, // Use the provided ID
		StackDefinition: stackDefinitionBlock,            // Use the manually constructed block
	}

	return stackDefinition, args.Get(1).(*core.DetailedResponse), args.Error(2)
}

func (mock *ProjectsServiceMock) NewCreateStackDefinitionOptions(projectID string, configID string, stackDefinition *projects.StackDefinitionBlockPrototype) *projects.CreateStackDefinitionOptions {

	createStackDefinitionOptions := &projects.CreateStackDefinitionOptions{
		ProjectID:       core.StringPtr(projectID),
		ID:              core.StringPtr(configID),
		StackDefinition: stackDefinition,
	}

	return createStackDefinitionOptions
}

func (mock *ProjectsServiceMock) UpdateStackDefinition(updateStackDefinitionOptions *projects.UpdateStackDefinitionOptions) (result *projects.StackDefinition, response *core.DetailedResponse, err error) {
	args := mock.Called(updateStackDefinitionOptions)
	return args.Get(0).(*projects.StackDefinition), args.Get(1).(*core.DetailedResponse), args.Error(2)
}

func (mock *ProjectsServiceMock) GetStackDefinition(getStackDefinitionOptions *projects.GetStackDefinitionOptions) (result *projects.StackDefinition, response *core.DetailedResponse, err error) {
	args := mock.Called(getStackDefinitionOptions)
	return args.Get(0).(*projects.StackDefinition), args.Get(1).(*core.DetailedResponse), args.Error(2)
}

func (mock *ProjectsServiceMock) ValidateConfig(validateConfigOptions *projects.ValidateConfigOptions) (result *projects.ProjectConfigVersion, response *core.DetailedResponse, err error) {
	args := mock.Called(validateConfigOptions)
	return args.Get(0).(*projects.ProjectConfigVersion), args.Get(1).(*core.DetailedResponse), args.Error(2)
}

func (mock *ProjectsServiceMock) Approve(approveOptions *projects.ApproveOptions) (result *projects.ProjectConfigVersion, response *core.DetailedResponse, err error) {
	args := mock.Called(approveOptions)
	return args.Get(0).(*projects.ProjectConfigVersion), args.Get(1).(*core.DetailedResponse), args.Error(2)
}

func (mock *ProjectsServiceMock) DeployConfig(deployConfigOptions *projects.DeployConfigOptions) (result *projects.ProjectConfigVersion, response *core.DetailedResponse, err error) {
	args := mock.Called(deployConfigOptions)
	return args.Get(0).(*projects.ProjectConfigVersion), args.Get(1).(*core.DetailedResponse), args.Error(2)
}

func (mock *ProjectsServiceMock) UndeployConfig(unDeployConfigOptions *projects.UndeployConfigOptions) (result *projects.ProjectConfigVersion, response *core.DetailedResponse, err error) {
	args := mock.Called(unDeployConfigOptions)
	return args.Get(0).(*projects.ProjectConfigVersion), args.Get(1).(*core.DetailedResponse), args.Error(2)
}

func (mock *ProjectsServiceMock) SyncConfig(syncConfigOptions *projects.SyncConfigOptions) (response *core.DetailedResponse, err error) {
	args := mock.Called(syncConfigOptions)
	return args.Get(0).(*core.DetailedResponse), args.Error(1)
}

func (mock *ProjectsServiceMock) NewListProjectsOptions() *projects.ListProjectsOptions {
	args := mock.Called()
	return args.Get(0).(*projects.ListProjectsOptions)
}

func (mock *ProjectsServiceMock) ListProjects(listProjectsOptions *projects.ListProjectsOptions) (result *projects.ProjectCollection, response *core.DetailedResponse, err error) {
	args := mock.Called(listProjectsOptions)
	return args.Get(0).(*projects.ProjectCollection), args.Get(1).(*core.DetailedResponse), args.Error(2)
}

type MockStackDefinitionCreator struct {
	mock.Mock
}

func (m *MockStackDefinitionCreator) CreateStackDefinitionWrapper(options *projects.CreateStackDefinitionOptions, members []projects.ProjectConfig) (*projects.StackDefinition, *core.DetailedResponse, error) {
	var mockedStackDefinition *projects.StackDefinition

	var stackDefinitionMembers []projects.StackDefinitionMember

	for _, member := range members {
		// Type assert 'Definition' to '*projects.ProjectConfigDefinitionResponse'
		def, ok := member.Definition.(*projects.ProjectConfigDefinitionResponse)
		if !ok {
			// Log or handle the case where the assertion fails
			fmt.Printf("Definition type assertion failed for member ID: %s\n", *member.ID)
			continue
		}

		// Extract 'Name' and 'VersionLocator'
		memberName := def.Name
		versionLocator := def.LocatorID

		// Convert 'Inputs' from 'map[string]interface{}' to '[]StackDefinitionMemberInput'
		var inputs []projects.StackDefinitionMemberInput
		if def.Inputs != nil {
			for inputName, inputValue := range def.Inputs {
				inputValueStr := fmt.Sprintf("%v", inputValue)
				inputs = append(inputs, projects.StackDefinitionMemberInput{
					Name:  &inputName,
					Value: &inputValueStr,
				})
			}
		}

		// Create a 'StackDefinitionMember'
		stackMember := projects.StackDefinitionMember{
			Name:           memberName,
			VersionLocator: versionLocator,
			Inputs:         inputs,
		}

		stackDefinitionMembers = append(stackDefinitionMembers, stackMember)
	}

	// Adjust the 'Inputs' in 'StackDefinition'
	adjustedInputs := adjustInputVariables(options.StackDefinition.Inputs)

	// Construct the 'StackDefinition'
	mockedStackDefinition = &projects.StackDefinition{
		ID: options.ID,
		StackDefinition: &projects.StackDefinitionBlock{
			Inputs:  adjustedInputs,
			Outputs: options.StackDefinition.Outputs,
			Members: stackDefinitionMembers,
		},
	}

	return mockedStackDefinition, nil, nil
}

// adjustMemberInputs converts the map[string]interface{} to []projects.StackDefinitionMemberInput
func adjustInputVariables(inputs []projects.StackDefinitionInputVariable) []projects.StackDefinitionInputVariable {
	var adjusted []projects.StackDefinitionInputVariable
	for _, input := range inputs {
		adjustedInput := input
		switch v := input.Default.(type) {
		case string:
			val := v
			adjustedInput.Default = &val
		case bool:
			val := v
			adjustedInput.Default = &val
		case int:
			val := int64(v)
			adjustedInput.Default = &val
		case float64:
			val := int64(v)
			adjustedInput.Default = &val
		default:
			// Handle other types as needed
			adjustedInput.Default = input.Default
		}
		adjusted = append(adjusted, adjustedInput)
	}
	return adjusted
}
