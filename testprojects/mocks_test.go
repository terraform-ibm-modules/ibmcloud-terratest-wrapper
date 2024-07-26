package testprojects

import (
	"github.com/IBM/go-sdk-core/v5/core"
	projects "github.com/IBM/project-go-sdk/projectv1"
	"github.com/stretchr/testify/mock"
	"net/http"
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

func (mock *projectsServiceMock) CreateStackDefinition(createStackDefinitionOptions *projects.CreateStackDefinitionOptions) (result *projects.StackDefinition, response *core.DetailedResponse, err error) {
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

// IAM AUTHENTICATOR INTERFACE MOCK
type iamAuthenticatorMock struct {
	mock.Mock
}

// IAM AUTHENTIATOR INTERFACE MOCK FUNCTIONS
func (mock *iamAuthenticatorMock) Authenticate(request *http.Request) error {
	return nil
}

func (mock *iamAuthenticatorMock) AuthenticationType() string {
	return core.AUTHTYPE_IAM
}

func (mock *iamAuthenticatorMock) Validate() error {
	return nil
}

func (mock *iamAuthenticatorMock) RequestToken() (*core.IamTokenServerResponse, error) {
	retval := &core.IamTokenServerResponse{
		AccessToken:  "fake-token",
		RefreshToken: "fake-refresh-token",
	}

	return retval, nil
}
