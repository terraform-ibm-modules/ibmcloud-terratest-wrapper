package cloudinfo

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/IBM/go-sdk-core/v5/core"
	projects "github.com/IBM/project-go-sdk/projectv1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type ProjectsServiceTestSuite struct {
	suite.Suite
	mockService *ProjectsServiceMock
	infoSvc     *CloudInfoService
}

func (suite *ProjectsServiceTestSuite) SetupTest() {
	suite.mockService = new(ProjectsServiceMock)
	suite.infoSvc = &CloudInfoService{
		projectsService: suite.mockService,
		authenticator: &core.IamAuthenticator{
			ApiKey: "mockApiKey",
		},
	}
}

func (suite *ProjectsServiceTestSuite) TestCreateProjectFromConfig() {
	mockProject := &projects.Project{ID: core.StringPtr("mockProjectID")}
	mockResponse := &core.DetailedResponse{StatusCode: 201}
	// mock an sdk error
	mockError := core.RepurposeSDKProblem(fmt.Errorf("error creating  project"), "")

	testCases := []struct {
		name             string
		expectedError    error
		mockError        error
		expectedResult   *projects.Project
		mockResult       *projects.Project
		expectedResponse *core.DetailedResponse
		mockResponse     *core.DetailedResponse
	}{
		{
			name:             "Success case",
			expectedError:    nil,
			mockError:        nil,
			mockResult:       mockProject,
			expectedResult:   mockProject,
			mockResponse:     mockResponse,
			expectedResponse: mockResponse,
		},
		{
			name:             "Failure case",
			expectedError:    mockError,
			mockError:        mockError,
			mockResult:       nil,
			expectedResult:   nil,
			mockResponse:     nil,
			expectedResponse: nil,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Clear previous expectations
			suite.mockService.ExpectedCalls = nil

			suite.mockService.On("CreateProject", mock.Anything).Return(tc.mockResult, tc.mockResponse, tc.mockError)

			result, response, err := suite.infoSvc.CreateProjectFromConfig(&ProjectsConfig{})
			if tc.expectedError != nil {
				assert.Error(suite.T(), err)
				assert.Equal(suite.T(), tc.expectedError, err)
			} else {
				assert.NoError(suite.T(), err)
				assert.Equal(suite.T(), tc.expectedResult, result)
				assert.Equal(suite.T(), tc.expectedResponse, response)
			}
		})
	}
}

func (suite *ProjectsServiceTestSuite) TestGetProject() {
	mockProject := &projects.Project{ID: core.StringPtr("mockProjectID")}
	mockResponse := &core.DetailedResponse{StatusCode: 200}
	mockError := fmt.Errorf("error getting project")

	testCases := []struct {
		name             string
		expectedError    error
		mockError        error
		expectedResult   *projects.Project
		expectedResponse *core.DetailedResponse
	}{
		{
			name:             "Success case",
			expectedError:    nil,
			mockError:        nil,
			expectedResult:   mockProject,
			expectedResponse: mockResponse,
		},
		{
			name:             "Failure case",
			expectedError:    mockError,
			mockError:        mockError,
			expectedResult:   nil,
			expectedResponse: nil,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Clear previous expectations
			suite.mockService.ExpectedCalls = nil

			suite.mockService.On("GetProject", mock.Anything).Return(tc.expectedResult, tc.expectedResponse, tc.mockError)

			result, response, err := suite.infoSvc.GetProject("mockProjectID")
			if tc.expectedError != nil {
				assert.Error(suite.T(), err)
				assert.Equal(suite.T(), tc.expectedError, err)
			} else {
				assert.NoError(suite.T(), err)
				assert.Equal(suite.T(), tc.expectedResult, result)
				assert.Equal(suite.T(), tc.expectedResponse, response)
			}
		})
	}
}

func (suite *ProjectsServiceTestSuite) TestGetConfig() {
	mockConfig := &projects.ProjectConfig{ID: core.StringPtr("mockConfigID")}
	mockResponse := &core.DetailedResponse{StatusCode: 200}
	mockError := fmt.Errorf("error getting config")

	testCases := []struct {
		name             string
		expectedError    error
		mockError        error
		expectedResult   *projects.ProjectConfig
		expectedResponse *core.DetailedResponse
	}{
		{
			name:             "Success case",
			expectedError:    nil,
			mockError:        nil,
			expectedResult:   mockConfig,
			expectedResponse: mockResponse,
		},
		{
			name:             "Failure case",
			expectedError:    mockError,
			mockError:        mockError,
			expectedResult:   nil,
			expectedResponse: nil,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Clear previous expectations
			suite.mockService.ExpectedCalls = nil

			suite.mockService.On("GetConfig", mock.Anything).Return(tc.expectedResult, tc.expectedResponse, tc.mockError)

			result, response, err := suite.infoSvc.GetConfig(&ConfigDetails{})
			if tc.expectedError != nil {
				assert.Error(suite.T(), err)
				assert.Equal(suite.T(), tc.expectedError, err)
			} else {
				assert.NoError(suite.T(), err)
				assert.Equal(suite.T(), tc.expectedResult, result)
				assert.Equal(suite.T(), tc.expectedResponse, response)
			}
		})
	}
}

func (suite *ProjectsServiceTestSuite) TestDeleteProject() {
	mockResponse := &projects.ProjectDeleteResponse{}
	mockDetailedResponse := &core.DetailedResponse{}
	mockError := fmt.Errorf("error deleting project")

	testCases := []struct {
		name           string
		expectedError  error
		mockError      error
		expectedResult *projects.ProjectDeleteResponse
	}{
		{
			name:           "Success case",
			expectedError:  nil,
			mockError:      nil,
			expectedResult: mockResponse,
		},
		{
			name:           "Failure case",
			expectedError:  mockError,
			mockError:      mockError,
			expectedResult: nil,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.mockService.ExpectedCalls = nil

			suite.mockService.On("DeleteProject", mock.Anything).Return(tc.expectedResult, mockDetailedResponse, tc.mockError)

			result, _, err := suite.infoSvc.DeleteProject("mockProjectID")
			if tc.expectedError != nil {
				assert.Error(suite.T(), err)
				assert.Equal(suite.T(), tc.expectedError, err)
			} else {
				assert.NoError(suite.T(), err)
				assert.Equal(suite.T(), tc.expectedResult, result)
			}
		})
	}
}

func (suite *ProjectsServiceTestSuite) TestCreateConfig() {
	mockConfig := &projects.ProjectConfig{ID: core.StringPtr("config1")}
	mockResponse := &core.DetailedResponse{}
	mockError := fmt.Errorf("error creating config")

	testCases := []struct {
		name           string
		expectedError  error
		mockError      error
		expectedResult *projects.ProjectConfig
	}{
		{
			name:           "Success case",
			expectedError:  nil,
			mockError:      nil,
			expectedResult: mockConfig,
		},
		{
			name:           "Failure case",
			expectedError:  mockError,
			mockError:      mockError,
			expectedResult: nil,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.mockService.ExpectedCalls = nil

			suite.mockService.On("CreateConfig", mock.Anything).Return(tc.expectedResult, mockResponse, tc.mockError)

			result, _, err := suite.infoSvc.CreateConfig(&ConfigDetails{})
			if tc.expectedError != nil {
				assert.Error(suite.T(), err)
				assert.Equal(suite.T(), tc.expectedError, err)
			} else {
				assert.NoError(suite.T(), err)
				assert.Equal(suite.T(), tc.expectedResult, result)
			}
		})
	}
}

func (suite *ProjectsServiceTestSuite) TestCreateStackFromConfigFile() {
	testCases := []struct {
		name            string
		stackConfig     *ConfigDetails
		stackConfigPath string
		catalogJsonPath string
		expectedInputs  map[string]interface{}
	}{
		{
			name: "Inputs from current stack configuration",
			stackConfig: &ConfigDetails{
				ProjectID: "mockProjectID",
				Inputs: map[string]interface{}{
					"input1": "value1",
					"input2": 2,
				},
			},
			stackConfigPath: "testdata/stack_definition_stack_inputs.json",
			catalogJsonPath: "testdata/ibm_catalog_with_config_overrides.json",
			expectedInputs: map[string]interface{}{
				"input1": "value1",
				"input2": 2,
			},
		},
		{
			name: "Default values from ibm_catalog.json",
			stackConfig: &ConfigDetails{
				ProjectID: "mockProjectID",
			},
			stackConfigPath: "testdata/stack_definition_no_stack_inputs.json",
			catalogJsonPath: "testdata/ibm_catalog_with_config_overrides.json",
			expectedInputs: map[string]interface{}{
				"input1": "default1",
				"input2": 20,
			},
		},
		{
			name: "Default values from stack_definition.json",
			stackConfig: &ConfigDetails{
				ProjectID: "mockProjectID",
			},
			stackConfigPath: "testdata/stack_definition_no_stack_inputs.json",
			catalogJsonPath: "testdata/ibm_catalog_no_config_overrides.json",
			expectedInputs: map[string]interface{}{
				"input1": "defaultValue1",
				"input2": 10,
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Unmarshal stackConfig JSON
			stackJsonFile, err := os.ReadFile(tc.stackConfigPath)
			assert.NoError(suite.T(), err)

			var stackJson Stack
			err = json.Unmarshal(stackJsonFile, &stackJson)
			assert.NoError(suite.T(), err)

			// Unmarshal catalogJson JSON
			catalogJsonFile, err := os.ReadFile(tc.catalogJsonPath)
			assert.NoError(suite.T(), err)

			var catalogConfig CatalogJson
			err = json.Unmarshal(catalogJsonFile, &catalogConfig)
			assert.NoError(suite.T(), err)

			mockStackDefinition := &projects.StackDefinition{
				ID: core.StringPtr("mockStackID"),
			}
			mockResponse := &core.DetailedResponse{StatusCode: 201}

			suite.mockService.On("CreateStackDefinition", mock.Anything).Return(mockStackDefinition, mockResponse, nil)
			suite.mockService.On("CreateConfig", mock.Anything).Return(&projects.ProjectConfig{}, mockResponse, nil)
			suite.mockService.On("NewCreateConfigOptions", mock.Anything, mock.Anything).Return(&projects.CreateConfigOptions{}, nil)

			result, response, err := suite.infoSvc.CreateStackFromConfigFile(tc.stackConfig, tc.stackConfigPath, tc.catalogJsonPath)

			assert.NoError(suite.T(), err)
			assert.Equal(suite.T(), mockStackDefinition, result)
			assert.Equal(suite.T(), mockResponse, response)
			assert.Equal(suite.T(), tc.expectedInputs, tc.stackConfig.Inputs)
			suite.mockService.AssertCalled(suite.T(), "CreateStackDefinition", mock.Anything)
			suite.mockService.AssertCalled(suite.T(), "CreateConfig", mock.Anything)
			suite.mockService.AssertCalled(suite.T(), "NewCreateConfigOptions", mock.Anything, mock.Anything)
		})
	}
}

func TestProjectsServiceTestSuite(t *testing.T) {
	suite.Run(t, new(ProjectsServiceTestSuite))
}
