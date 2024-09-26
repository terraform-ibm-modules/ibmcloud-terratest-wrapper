package cloudinfo

import (
	"github.com/IBM/go-sdk-core/v5/core"
	projects "github.com/IBM/project-go-sdk/projectv1"
	"github.com/stretchr/testify/mock"
)

const mockProjectID = "mockProjectID"

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
	args := mock.Called(projectID, definition)
	return args.Get(0).(*projects.CreateConfigOptions)
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
	return args.Get(0).(*projects.ProjectConfig), args.Get(1).(*core.DetailedResponse), args.Error(2)
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
	args := mock.Called(createStackDefinitionOptions)
	return args.Get(0).(*projects.StackDefinition), args.Get(1).(*core.DetailedResponse), args.Error(2)
}

func (mock *ProjectsServiceMock) NewCreateStackDefinitionOptions(projectID string, id string, stackDefinition *projects.StackDefinitionBlockPrototype) *projects.CreateStackDefinitionOptions {
	args := mock.Called(projectID, id, stackDefinition)
	return args.Get(0).(*projects.CreateStackDefinitionOptions)
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
