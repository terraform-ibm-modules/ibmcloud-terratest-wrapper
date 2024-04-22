package cloudinfo

import (
	"github.com/IBM/go-sdk-core/v5/core"
	projects "github.com/IBM/project-go-sdk/projectv1"
	"github.com/stretchr/testify/mock"
)

const mockProjectID = "mockProjectID"

// Projects API SERVICE MOCK
type projectsServiceMock struct {
	mock.Mock
}

func (mock *projectsServiceMock) CreateProject(createProjectOptions *projects.CreateProjectOptions) (result *projects.Project, response *core.DetailedResponse, err error) {
	//TODO implement me
	panic("implement me")
}

func (mock *projectsServiceMock) GetProject(getProjectOptions *projects.GetProjectOptions) (result *projects.Project, response *core.DetailedResponse, err error) {
	//TODO implement me
	panic("implement me")
}

func (mock *projectsServiceMock) UpdateProject(updateProjectOptions *projects.UpdateProjectOptions) (result *projects.Project, response *core.DetailedResponse, err error) {
	//TODO implement me
	panic("implement me")
}

func (mock *projectsServiceMock) DeleteProject(deleteProjectOptions *projects.DeleteProjectOptions) (result *projects.ProjectDeleteResponse, response *core.DetailedResponse, err error) {
	//TODO implement me
	panic("implement me")
}

func (mock *projectsServiceMock) CreateConfig(createConfigOptions *projects.CreateConfigOptions) (result *projects.ProjectConfig, response *core.DetailedResponse, err error) {
	//TODO implement me
	panic("implement me")
}

func (mock *projectsServiceMock) GetConfig(getConfigOptions *projects.GetConfigOptions) (result *projects.ProjectConfig, response *core.DetailedResponse, err error) {
	//TODO implement me
	panic("implement me")
}

func (mock *projectsServiceMock) DeleteConfig(deleteConfigOptions *projects.DeleteConfigOptions) (result *projects.ProjectConfigDelete, response *core.DetailedResponse, err error) {
	//TODO implement me
	panic("implement me")
}

func (mock *projectsServiceMock) GetStackDefinition(getStackDefinitionOptions *projects.GetStackDefinitionOptions) (result *projects.StackDefinition, response *core.DetailedResponse, err error) {
	//TODO implement me
	panic("implement me")
}

func (mock *projectsServiceMock) ValidateConfig(validateConfigOptions *projects.ValidateConfigOptions) (result *projects.ProjectConfigVersion, response *core.DetailedResponse, err error) {
	//TODO implement me
	panic("implement me")
}

func (mock *projectsServiceMock) Approve(approveOptions *projects.ApproveOptions) (result *projects.ProjectConfigVersion, response *core.DetailedResponse, err error) {
	//TODO implement me
	panic("implement me")
}

func (mock *projectsServiceMock) DeployConfig(deployConfigOptions *projects.DeployConfigOptions) (result *projects.ProjectConfigVersion, response *core.DetailedResponse, err error) {
	//TODO implement me
	panic("implement me")
}
