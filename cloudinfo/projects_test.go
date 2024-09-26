package cloudinfo

import (
	"fmt"
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

// TODO: Figure out how to mock NewConfigsPager
//func (suite *ProjectsServiceTestSuite) TestGetProjectConfigs() {
//	mockConfigSummaries := []projects.ProjectConfigSummary{
//		{ID: core.StringPtr("config1")},
//		{ID: core.StringPtr("config2")},
//	}
//	mockError := fmt.Errorf("error getting project configs")
//
//	testCases := []struct {
//		name           string
//		expectedError  error
//		mockError      error
//		expectedResult []projects.ProjectConfigSummary
//	}{
//		{
//			name:           "Success case",
//			expectedError:  nil,
//			mockError:      nil,
//			expectedResult: mockConfigSummaries,
//		},
//		{
//			name:           "Failure case",
//			expectedError:  mockError,
//			mockError:      mockError,
//			expectedResult: nil,
//		},
//	}
//
//	for _, tc := range testCases {
//		suite.Run(tc.name, func() {
//			// Clear previous expectations
//			suite.mockService.ExpectedCalls = nil
//
//			// Create a mock ConfigsPager
//			mockPager := new(MockConfigsPager)
//			mockPager.On("HasNext").Return(true).Once()
//			mockPager.On("HasNext").Return(false).Once()
//			mockPager.On("GetNext").Return(tc.expectedResult, tc.mockError)
//
//			// Set up the mock expectation for NewConfigsPager
//			suite.mockService.On("NewConfigsPager", mock.Anything).Return(&mockPager, tc.mockError)
//
//			// Set up the mock expectation for GetProjectConfigs
//			suite.mockService.On("GetProjectConfigs", mock.Anything).Return(tc.expectedResult, tc.mockError)
//
//			result, err := suite.infoSvc.GetProjectConfigs("mockProjectID")
//			if tc.expectedError != nil {
//				assert.Error(suite.T(), err)
//				assert.Equal(suite.T(), tc.expectedError, err)
//			} else {
//				assert.NoError(suite.T(), err)
//				assert.Equal(suite.T(), tc.expectedResult, result)
//			}
//		})
//	}
//}

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

func TestProjectsServiceTestSuite(t *testing.T) {
	suite.Run(t, new(ProjectsServiceTestSuite))
}
