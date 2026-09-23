package cloudinfo

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
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
	mockCreator *MockStackDefinitionCreator
}

func (suite *ProjectsServiceTestSuite) SetupTest() {
	suite.mockService = new(ProjectsServiceMock)
	suite.mockCreator = new(MockStackDefinitionCreator)

	suite.infoSvc = &CloudInfoService{
		projectsService: suite.mockService,
		authenticator: &core.IamAuthenticator{
			ApiKey: "mockApiKey",
		},
		ApiKey:                 "mockApiKey",
		stackDefinitionCreator: suite.mockCreator,
	}
	suite.mockCreator = new(MockStackDefinitionCreator)
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
	// Use a non-retryable error (401) so it fails immediately without retries
	mockError := fmt.Errorf("401 unauthorized: error getting config")

	testCases := []struct {
		name             string
		configDetails    *ConfigDetails
		expectedError    error
		mockError        error
		expectedResult   *projects.ProjectConfig
		expectedResponse *core.DetailedResponse
	}{
		{
			name: "Success case",
			configDetails: &ConfigDetails{
				ProjectID: "test-project-id",
				ConfigID:  "test-config-id",
			},
			expectedError:    nil,
			mockError:        nil,
			expectedResult:   mockConfig,
			expectedResponse: mockResponse,
		},
		{
			name: "Failure case",
			configDetails: &ConfigDetails{
				ProjectID: "test-project-id",
				ConfigID:  "test-config-id",
			},
			expectedError:    mockError,
			mockError:        mockError,
			expectedResult:   nil,
			expectedResponse: nil,
		},
		{
			name: "Empty ProjectID",
			configDetails: &ConfigDetails{
				ProjectID: "",
				ConfigID:  "test-config-id",
			},
			expectedError:    fmt.Errorf("ProjectID cannot be empty"),
			mockError:        nil,
			expectedResult:   nil,
			expectedResponse: nil,
		},
		{
			name: "Empty ConfigID",
			configDetails: &ConfigDetails{
				ProjectID: "test-project-id",
				ConfigID:  "",
			},
			expectedError:    fmt.Errorf("ConfigID cannot be empty"),
			mockError:        nil,
			expectedResult:   nil,
			expectedResponse: nil,
		},
		{
			name:             "Nil ConfigDetails",
			configDetails:    nil,
			expectedError:    fmt.Errorf("configDetails cannot be nil"),
			mockError:        nil,
			expectedResult:   nil,
			expectedResponse: nil,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Clear previous expectations
			suite.mockService.ExpectedCalls = nil

			// Only set up mock if we expect it to be called (valid inputs)
			if tc.configDetails != nil && tc.configDetails.ProjectID != "" && tc.configDetails.ConfigID != "" {
				suite.mockService.On("GetConfig", mock.Anything).Return(tc.expectedResult, tc.expectedResponse, tc.mockError)
			}

			result, response, err := suite.infoSvc.GetConfig(tc.configDetails)
			if tc.expectedError != nil {
				assert.Error(suite.T(), err)
				assert.Equal(suite.T(), tc.expectedError.Error(), err.Error())
			} else {
				assert.NoError(suite.T(), err)
				assert.Equal(suite.T(), tc.expectedResult, result)
				assert.Equal(suite.T(), tc.expectedResponse, response)
			}
		})
	}
}
func (suite *ProjectsServiceTestSuite) TestGetConfigWithRetry() {
	mockConfig := &projects.ProjectConfig{ID: core.StringPtr("mockConfigID")}
	mockResponse := &core.DetailedResponse{StatusCode: 200}

	testCases := []struct {
		name             string
		configDetails    *ConfigDetails
		mockError        error
		expectedError    bool
		expectedResult   *projects.ProjectConfig
		expectedResponse *core.DetailedResponse
	}{
		{
			name: "Success on first attempt",
			configDetails: &ConfigDetails{
				ProjectID: "test-project-id",
				ConfigID:  "test-config-id",
			},
			mockError:        nil,
			expectedError:    false,
			expectedResult:   mockConfig,
			expectedResponse: mockResponse,
		},
		{
			name: "IAM authorization error is retryable - succeeds on second attempt",
			configDetails: &ConfigDetails{
				ProjectID: "test-project-id",
				ConfigID:  "test-config-id",
			},
			mockError: fmt.Errorf("An error occurred while asking authorization on IAM: error when asking authorization - errors array does not exist or is empty"),
			// This test verifies the IAM error is retryable by having it succeed on the second attempt
			// This proves the error was identified as retryable and a retry was attempted
			expectedError:    false,
			expectedResult:   mockConfig,
			expectedResponse: mockResponse,
		},
		{
			name: "Non-retryable 401 error fails immediately",
			configDetails: &ConfigDetails{
				ProjectID: "test-project-id",
				ConfigID:  "test-config-id",
			},
			mockError:        fmt.Errorf("401 unauthorized"),
			expectedError:    true,
			expectedResult:   nil,
			expectedResponse: nil,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Clear previous expectations
			suite.mockService.ExpectedCalls = nil

			if tc.mockError == nil {
				// Success case - just return success
				suite.mockService.On("GetConfig", mock.Anything).Return(mockConfig, mockResponse, nil).Once()
			} else if tc.expectedError {
				// Non-retryable error - will fail immediately, set up one call
				suite.mockService.On("GetConfig", mock.Anything).Return((*projects.ProjectConfig)(nil), (*core.DetailedResponse)(nil), tc.mockError).Once()
			} else {
				// Retryable error - fail once, then succeed
				// The retry logic should stop after success
				suite.mockService.On("GetConfig", mock.Anything).Return((*projects.ProjectConfig)(nil), (*core.DetailedResponse)(nil), tc.mockError).Once()
				suite.mockService.On("GetConfig", mock.Anything).Return(mockConfig, mockResponse, nil).Once()
				// Add Maybe() to catch any unexpected extra calls (shouldn't happen if retry stops on success)
				suite.mockService.On("GetConfig", mock.Anything).Return(mockConfig, mockResponse, nil).Maybe()
			}

			result, response, err := suite.infoSvc.GetConfig(tc.configDetails)

			// Verify expectations
			if tc.expectedError {
				assert.Error(suite.T(), err)
				assert.Nil(suite.T(), result)
				assert.Nil(suite.T(), response)
			} else {
				assert.NoError(suite.T(), err)
				assert.Equal(suite.T(), tc.expectedResult, result)
				assert.Equal(suite.T(), tc.expectedResponse, response)
			}

			// Verify all expected calls were made
			suite.mockService.AssertExpectations(suite.T())
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
	mockConfig := &projects.ProjectConfig{
		ID: core.StringPtr(""),
		Definition: &projects.ProjectConfigDefinitionResponse{
			LocatorID:   core.StringPtr(""),
			Description: core.StringPtr(""),
			Name:        core.StringPtr(""),
			Authorizations: &projects.ProjectConfigAuth{
				Method: core.StringPtr("api_key"),
				ApiKey: core.StringPtr("mockApiKey"),
			},
		},
	}
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
		expectedConfig  *projects.StackDefinition
		expectedError   error
	}{
		{
			name: "Inputs from current stack configuration, these should override all other values",
			stackConfig: &ConfigDetails{
				ProjectID: "mockProjectID",
				ConfigID:  "54321",
				Inputs: map[string]interface{}{
					"input1": "test_value1",
					"input2": 2,
				},
			},
			stackConfigPath: "testdata/stack_definition_stack_inputs.json",
			catalogJsonPath: "testdata/ibm_catalog_multiple_products_flavors.json",
			expectedConfig: &projects.StackDefinition{
				ID: core.StringPtr("mockProjectID"), // This would be generated on the server side and not part of the input
				StackDefinition: &projects.StackDefinitionBlock{
					Inputs: []projects.StackDefinitionInputVariable{
						{
							Name:        core.StringPtr("input1"),
							Type:        core.StringPtr("string"),
							Required:    core.BoolPtr(true),
							Default:     core.StringPtr("test_value1"),
							Description: core.StringPtr(""),
							Hidden:      core.BoolPtr(false),
						},
						{
							Name:        core.StringPtr("input2"),
							Type:        core.StringPtr("int"),
							Required:    core.BoolPtr(false),
							Default:     core.Int64Ptr(2),
							Description: core.StringPtr(""),
							Hidden:      core.BoolPtr(false),
						},
						{
							Name:        core.StringPtr("input3"),
							Type:        core.StringPtr("array"),
							Required:    core.BoolPtr(false),
							Default:     core.StringPtr("[\"stack_def_arr_value1\", \"stack_def_arr_value2\"]"),
							Description: core.StringPtr(""),
							Hidden:      core.BoolPtr(false),
						},
					},
					Outputs: []projects.StackDefinitionOutputVariable{
						{Name: core.StringPtr("output1"), Value: core.StringPtr("ref:../members/member1/outputs/output1")},
						{Name: core.StringPtr("output2"), Value: core.StringPtr("ref:../members/member2/outputs/output2")},
					},
					Members: []projects.StackDefinitionMember{
						{
							Name:           core.StringPtr("member1"),
							VersionLocator: core.StringPtr("version1"),
							Inputs: []projects.StackDefinitionMemberInput{
								{Name: core.StringPtr("input1"), Value: core.StringPtr("ref:../../inputs/input1")},
								{Name: core.StringPtr("input2"), Value: core.StringPtr("20")},
								{Name: core.StringPtr("input3"), Value: core.StringPtr("stack_def_value3")},
							},
						},
						{
							Name:           core.StringPtr("member2"),
							VersionLocator: core.StringPtr("version2"),
							Inputs: []projects.StackDefinitionMemberInput{
								{Name: core.StringPtr("input1"), Value: core.StringPtr("ref:../../inputs/input2")},
								{Name: core.StringPtr("input2"), Value: core.StringPtr("30")},
								{Name: core.StringPtr("input3"), Value: core.StringPtr("stack_def_value4")},
							},
						},
					},
				},
			},
			expectedError: nil,
		},
		{
			name: "Inputs from current stack configuration with member configs, these should override all other values",
			stackConfig: &ConfigDetails{
				ProjectID: "mockProjectID",
				ConfigID:  "54321",
				Inputs: map[string]interface{}{
					"input1": "test_value1",
					"input2": 2,
				},
				MemberConfigDetails: []ConfigDetails{
					{
						Name: "member1",
						Inputs: map[string]interface{}{
							"input1": "member1_input1",
							"input2": 5,
							"input3": "[\"member1_input3_value1\", \"member1_input3_value2\"]",
						},
					},
				},
			},
			stackConfigPath: "testdata/stack_definition_stack_inputs.json",
			catalogJsonPath: "testdata/ibm_catalog_multiple_products_flavors.json",
			expectedConfig: &projects.StackDefinition{
				ID: core.StringPtr("mockProjectID"), // This would be generated on the server side and not part of the input
				StackDefinition: &projects.StackDefinitionBlock{
					Inputs: []projects.StackDefinitionInputVariable{
						{
							Name:        core.StringPtr("input1"),
							Type:        core.StringPtr("string"),
							Required:    core.BoolPtr(true),
							Default:     core.StringPtr("test_value1"),
							Description: core.StringPtr(""),
							Hidden:      core.BoolPtr(false),
						},
						{
							Name:        core.StringPtr("input2"),
							Type:        core.StringPtr("int"),
							Required:    core.BoolPtr(false),
							Default:     core.Int64Ptr(2),
							Description: core.StringPtr(""),
							Hidden:      core.BoolPtr(false),
						},
						{
							Name:        core.StringPtr("input3"),
							Type:        core.StringPtr("array"),
							Required:    core.BoolPtr(false),
							Default:     core.StringPtr("[\"stack_def_arr_value1\", \"stack_def_arr_value2\"]"),
							Description: core.StringPtr(""),
							Hidden:      core.BoolPtr(false),
						},
					},
					Outputs: []projects.StackDefinitionOutputVariable{
						{Name: core.StringPtr("output1"), Value: core.StringPtr("ref:../members/member1/outputs/output1")},
						{Name: core.StringPtr("output2"), Value: core.StringPtr("ref:../members/member2/outputs/output2")},
					},
					Members: []projects.StackDefinitionMember{
						{
							Name:           core.StringPtr("member1"),
							VersionLocator: core.StringPtr("version1"),
							Inputs: []projects.StackDefinitionMemberInput{
								{Name: core.StringPtr("input1"), Value: core.StringPtr("member1_input1")},
								{Name: core.StringPtr("input2"), Value: core.StringPtr("5")},
								{Name: core.StringPtr("input3"), Value: core.StringPtr("[\"member1_input3_value1\", \"member1_input3_value2\"]")},
							},
						},
						{
							Name:           core.StringPtr("member2"),
							VersionLocator: core.StringPtr("version2"),
							Inputs: []projects.StackDefinitionMemberInput{
								{Name: core.StringPtr("input1"), Value: core.StringPtr("ref:../../inputs/input2")},
								{Name: core.StringPtr("input2"), Value: core.StringPtr("30")},
								{Name: core.StringPtr("input3"), Value: core.StringPtr("stack_def_value4")},
							},
						},
					},
				},
			},
			expectedError: nil,
		},
		{
			name: "Default values from ibm_catalog.json, should override values from stack_definition.json",
			stackConfig: &ConfigDetails{
				ProjectID:          "mockProjectID",
				ConfigID:           "54321",
				CatalogProductName: "Product Name",
				CatalogFlavorName:  "Flavor Name",
			},
			stackConfigPath: "testdata/stack_definition_stack_inputs_extended.json",
			catalogJsonPath: "testdata/ibm_catalog_with_config_overrides.json",
			expectedConfig: &projects.StackDefinition{
				ID: core.StringPtr("mockProjectID"), // This would be generated on the server side and not part of the input
				StackDefinition: &projects.StackDefinitionBlock{
					Inputs: []projects.StackDefinitionInputVariable{
						{
							Name:        core.StringPtr("input1"),
							Type:        core.StringPtr("string"),
							Required:    core.BoolPtr(true),
							Default:     core.StringPtr("catalog_default1"),
							Description: core.StringPtr(""),
							Hidden:      core.BoolPtr(false),
						},
						{
							Name:        core.StringPtr("input2"),
							Type:        core.StringPtr("int"),
							Required:    core.BoolPtr(false),
							Default:     core.Int64Ptr(80),
							Description: core.StringPtr(""),
							Hidden:      core.BoolPtr(false),
						},
						{
							Name:        core.StringPtr("input3"),
							Type:        core.StringPtr("array"),
							Required:    core.BoolPtr(false),
							Default:     core.StringPtr("[\"catalog_arr_value1\", \"catalog_arr_value2\"]"),
							Description: core.StringPtr(""),
							Hidden:      core.BoolPtr(false),
						},
						{
							Name:        core.StringPtr("input4"),
							Type:        core.StringPtr("bool"),
							Required:    core.BoolPtr(false),
							Default:     core.BoolPtr(true),
							Description: core.StringPtr(""),
							Hidden:      core.BoolPtr(false),
						},
					},
					Outputs: []projects.StackDefinitionOutputVariable{
						{Name: core.StringPtr("output1"), Value: core.StringPtr("ref:../members/member1/outputs/output1")},
						{Name: core.StringPtr("output2"), Value: core.StringPtr("ref:../members/member2/outputs/output2")},
					},
					Members: []projects.StackDefinitionMember{
						{
							Name:           core.StringPtr("member1"),
							VersionLocator: core.StringPtr("version1"),
							Inputs: []projects.StackDefinitionMemberInput{
								{Name: core.StringPtr("input1"), Value: core.StringPtr("ref:../../inputs/input1")},
								{Name: core.StringPtr("input2"), Value: core.StringPtr("20")},
								{Name: core.StringPtr("input3"), Value: core.StringPtr("stack_def_value3")},
							},
						},
						{
							Name:           core.StringPtr("member2"),
							VersionLocator: core.StringPtr("version2"),
							Inputs: []projects.StackDefinitionMemberInput{
								{Name: core.StringPtr("input1"), Value: core.StringPtr("ref:../../inputs/input2")},
								{Name: core.StringPtr("input2"), Value: core.StringPtr("30")},
								{Name: core.StringPtr("input3"), Value: core.StringPtr("stack_def_value4")},
							},
						},
					},
				},
			},
			expectedError: nil,
		},
		{
			name: "Default values from ibm_catalog.json with a default not set, should override values from stack_definition.json",
			stackConfig: &ConfigDetails{
				ProjectID: "mockProjectID",
				ConfigID:  "54321",
			},
			stackConfigPath: "testdata/stack_definition_stack_inputs_extended.json",
			catalogJsonPath: "testdata/ibm_catalog_with_config_overrides_and_defaults_not_set.json",
			expectedConfig: &projects.StackDefinition{
				ID: core.StringPtr("mockProjectID"), // This would be generated on the server side and not part of the input
				StackDefinition: &projects.StackDefinitionBlock{
					Inputs: []projects.StackDefinitionInputVariable{
						{
							Name:        core.StringPtr("input1"),
							Type:        core.StringPtr("string"),
							Required:    core.BoolPtr(true),
							Default:     core.StringPtr("stack_def_Value1"),
							Description: core.StringPtr(""),
							Hidden:      core.BoolPtr(false),
						},
						{
							Name:        core.StringPtr("input2"),
							Type:        core.StringPtr("int"),
							Required:    core.BoolPtr(false),
							Default:     core.Int64Ptr(80),
							Description: core.StringPtr(""),
							Hidden:      core.BoolPtr(false),
						},
						{
							Name:        core.StringPtr("input3"),
							Type:        core.StringPtr("array"),
							Required:    core.BoolPtr(false),
							Default:     core.StringPtr("[\"stack_def_arr_value1\", \"stack_def_arr_value2\"]"),
							Description: core.StringPtr(""),
							Hidden:      core.BoolPtr(false),
						},
						{
							Name:        core.StringPtr("input4"),
							Type:        core.StringPtr("bool"),
							Required:    core.BoolPtr(false),
							Default:     core.BoolPtr(false),
							Description: core.StringPtr(""),
							Hidden:      core.BoolPtr(false),
						},
					},
					Outputs: []projects.StackDefinitionOutputVariable{
						{Name: core.StringPtr("output1"), Value: core.StringPtr("ref:../members/member1/outputs/output1")},
						{Name: core.StringPtr("output2"), Value: core.StringPtr("ref:../members/member2/outputs/output2")},
					},
					Members: []projects.StackDefinitionMember{
						{
							Name:           core.StringPtr("member1"),
							VersionLocator: core.StringPtr("version1"),
							Inputs: []projects.StackDefinitionMemberInput{
								{Name: core.StringPtr("input1"), Value: core.StringPtr("ref:../../inputs/input1")},
								{Name: core.StringPtr("input2"), Value: core.StringPtr("20")},
								{Name: core.StringPtr("input3"), Value: core.StringPtr("stack_def_value3")},
							},
						},
						{
							Name:           core.StringPtr("member2"),
							VersionLocator: core.StringPtr("version2"),
							Inputs: []projects.StackDefinitionMemberInput{
								{Name: core.StringPtr("input1"), Value: core.StringPtr("ref:../../inputs/input2")},
								{Name: core.StringPtr("input2"), Value: core.StringPtr("30")},
								{Name: core.StringPtr("input3"), Value: core.StringPtr("stack_def_value4")},
							},
						},
					},
				},
			},
			expectedError: nil,
		},
		{
			name: "Default values from stack_definition.json, this should be the default values if no other values are provided",
			stackConfig: &ConfigDetails{
				ProjectID: "mockProjectID",
				ConfigID:  "54321",
			},
			stackConfigPath: "testdata/stack_definition_stack_inputs.json",
			catalogJsonPath: "testdata/ibm_catalog_no_config_overrides.json",
			expectedConfig: &projects.StackDefinition{
				ID: core.StringPtr("mockProjectID"), // This would be generated on the server side and not part of the input
				StackDefinition: &projects.StackDefinitionBlock{
					Inputs: []projects.StackDefinitionInputVariable{
						{
							Name:        core.StringPtr("input1"),
							Type:        core.StringPtr("string"),
							Required:    core.BoolPtr(true),
							Default:     core.StringPtr("stack_def_Value1"),
							Description: core.StringPtr(""),
							Hidden:      core.BoolPtr(false),
						},
						{
							Name:        core.StringPtr("input2"),
							Type:        core.StringPtr("int"),
							Required:    core.BoolPtr(false),
							Default:     core.Int64Ptr(10),
							Description: core.StringPtr(""),
							Hidden:      core.BoolPtr(false),
						},
						{
							Name:        core.StringPtr("input3"),
							Type:        core.StringPtr("array"),
							Required:    core.BoolPtr(false),
							Default:     core.StringPtr("[\"stack_def_arr_value1\", \"stack_def_arr_value2\"]"),
							Description: core.StringPtr(""),
							Hidden:      core.BoolPtr(false),
						},
					},
					Outputs: []projects.StackDefinitionOutputVariable{
						{Name: core.StringPtr("output1"), Value: core.StringPtr("ref:../members/member1/outputs/output1")},
						{Name: core.StringPtr("output2"), Value: core.StringPtr("ref:../members/member2/outputs/output2")},
					},
					Members: []projects.StackDefinitionMember{
						{
							Name:           core.StringPtr("member1"),
							VersionLocator: core.StringPtr("version1"),
							Inputs: []projects.StackDefinitionMemberInput{
								{Name: core.StringPtr("input1"), Value: core.StringPtr("ref:../../inputs/input1")},
								{Name: core.StringPtr("input2"), Value: core.StringPtr("20")},
								{Name: core.StringPtr("input3"), Value: core.StringPtr("stack_def_value3")},
							},
						},
						{
							Name:           core.StringPtr("member2"),
							VersionLocator: core.StringPtr("version2"),
							Inputs: []projects.StackDefinitionMemberInput{
								{Name: core.StringPtr("input1"), Value: core.StringPtr("ref:../../inputs/input2")},
								{Name: core.StringPtr("input2"), Value: core.StringPtr("30")},
								{Name: core.StringPtr("input3"), Value: core.StringPtr("stack_def_value4")},
							},
						},
					},
				},
			},
			expectedError: nil,
		},
		{
			name: "catalog multiple products, checking values for correct product are selected",
			stackConfig: &ConfigDetails{
				ProjectID:          "mockProjectID",
				ConfigID:           "54321",
				CatalogProductName: "Second Product Name",
			},
			stackConfigPath: "testdata/stack_definition_stack_inputs.json",
			catalogJsonPath: "testdata/ibm_catalog_multiple_products_flavors.json",
			expectedConfig: &projects.StackDefinition{
				ID: core.StringPtr("mockProjectID"), // This would be generated on the server side and not part of the input
				StackDefinition: &projects.StackDefinitionBlock{
					Inputs: []projects.StackDefinitionInputVariable{
						{
							Name:        core.StringPtr("input1"),
							Type:        core.StringPtr("string"),
							Required:    core.BoolPtr(true),
							Default:     core.StringPtr("catalog_product2_default_flavor1"),
							Description: core.StringPtr(""),
							Hidden:      core.BoolPtr(false),
						},
						{
							Name:        core.StringPtr("input2"),
							Type:        core.StringPtr("int"),
							Required:    core.BoolPtr(false),
							Default:     core.Int64Ptr(85),
							Description: core.StringPtr(""),
							Hidden:      core.BoolPtr(false),
						},
						{
							Name:     core.StringPtr("input3"),
							Type:     core.StringPtr("array"),
							Required: core.BoolPtr(false),
							// not set in the catalog so should be the stack definition default
							Default:     core.StringPtr("[\"stack_def_arr_value1\", \"stack_def_arr_value2\"]"),
							Description: core.StringPtr(""),
							Hidden:      core.BoolPtr(false),
						},
					},
					Outputs: []projects.StackDefinitionOutputVariable{
						{Name: core.StringPtr("output1"), Value: core.StringPtr("ref:../members/member1/outputs/output1")},
						{Name: core.StringPtr("output2"), Value: core.StringPtr("ref:../members/member2/outputs/output2")},
					},
					// catalog can only configure stack level inputs, so the member inputs should be the same as the stack definition
					Members: []projects.StackDefinitionMember{
						{
							Name:           core.StringPtr("member1"),
							VersionLocator: core.StringPtr("version1"),
							Inputs: []projects.StackDefinitionMemberInput{
								{Name: core.StringPtr("input1"), Value: core.StringPtr("ref:../../inputs/input1")},
								{Name: core.StringPtr("input2"), Value: core.StringPtr("20")},
								{Name: core.StringPtr("input3"), Value: core.StringPtr("stack_def_value3")},
							},
						},
						{
							Name:           core.StringPtr("member2"),
							VersionLocator: core.StringPtr("version2"),
							Inputs: []projects.StackDefinitionMemberInput{
								{Name: core.StringPtr("input1"), Value: core.StringPtr("ref:../../inputs/input2")},
								{Name: core.StringPtr("input2"), Value: core.StringPtr("30")},
								{Name: core.StringPtr("input3"), Value: core.StringPtr("stack_def_value4")},
							},
						},
					},
				},
			},
			expectedError: nil,
		},
		{
			name: "catalog multiple flavors, checking values for correct flavor are selected",
			stackConfig: &ConfigDetails{
				ProjectID:          "mockProjectID",
				ConfigID:           "54321",
				CatalogProductName: "Second Product Name",
				CatalogFlavorName:  "Second Flavor Name",
			},
			stackConfigPath: "testdata/stack_definition_stack_inputs.json",
			catalogJsonPath: "testdata/ibm_catalog_multiple_products_flavors.json",
			expectedConfig: &projects.StackDefinition{
				ID: core.StringPtr("mockProjectID"), // This would be generated on the server side and not part of the input
				StackDefinition: &projects.StackDefinitionBlock{
					Inputs: []projects.StackDefinitionInputVariable{
						{
							Name:        core.StringPtr("input1"),
							Type:        core.StringPtr("string"),
							Required:    core.BoolPtr(true),
							Default:     core.StringPtr("product2_default_flavor2"),
							Description: core.StringPtr(""),
							Hidden:      core.BoolPtr(false),
						},
						{
							Name:        core.StringPtr("input2"),
							Type:        core.StringPtr("int"),
							Required:    core.BoolPtr(false),
							Default:     core.Int64Ptr(95),
							Description: core.StringPtr(""),
							Hidden:      core.BoolPtr(false),
						},
						{
							Name:     core.StringPtr("input3"),
							Type:     core.StringPtr("array"),
							Required: core.BoolPtr(false),
							// not set in the catalog so should be the stack definition default
							Default:     core.StringPtr("[\"stack_def_arr_value1\", \"stack_def_arr_value2\"]"),
							Description: core.StringPtr(""),
							Hidden:      core.BoolPtr(false),
						},
					},
					Outputs: []projects.StackDefinitionOutputVariable{
						{Name: core.StringPtr("output1"), Value: core.StringPtr("ref:../members/member1/outputs/output1")},
						{Name: core.StringPtr("output2"), Value: core.StringPtr("ref:../members/member2/outputs/output2")},
					},
					// catalog can only configure stack level inputs, so the member inputs should be the same as the stack definition
					Members: []projects.StackDefinitionMember{
						{
							Name:           core.StringPtr("member1"),
							VersionLocator: core.StringPtr("version1"),
							Inputs: []projects.StackDefinitionMemberInput{
								{Name: core.StringPtr("input1"), Value: core.StringPtr("ref:../../inputs/input1")},
								{Name: core.StringPtr("input2"), Value: core.StringPtr("20")},
								{Name: core.StringPtr("input3"), Value: core.StringPtr("stack_def_value3")},
							},
						},
						{
							Name:           core.StringPtr("member2"),
							VersionLocator: core.StringPtr("version2"),
							Inputs: []projects.StackDefinitionMemberInput{
								{Name: core.StringPtr("input1"), Value: core.StringPtr("ref:../../inputs/input2")},
								{Name: core.StringPtr("input2"), Value: core.StringPtr("30")},
								{Name: core.StringPtr("input3"), Value: core.StringPtr("stack_def_value4")},
							},
						},
					},
				},
			},
			expectedError: nil,
		},
		{
			name: "catalog multiple products with member configs set, checking values for correct product are selected",
			stackConfig: &ConfigDetails{
				ProjectID:          "mockProjectID",
				ConfigID:           "54321",
				CatalogProductName: "Second Product Name",
				MemberConfigDetails: []ConfigDetails{
					{
						Name: "member1",
						Inputs: map[string]interface{}{
							"input1": "member1_input1",
							"input2": 5,
							"input3": "[\"member1_input3_value1\", \"member1_input3_value2\"]",
						},
					},
					{
						Name: "member2",
						Inputs: map[string]interface{}{
							"input1": "member2_input1",
							"input2": 6,
							"input3": "[\"member2_input3_value1\", \"member2_input3_value2\"]",
						},
					},
				},
			},
			stackConfigPath: "testdata/stack_definition_stack_inputs.json",
			catalogJsonPath: "testdata/ibm_catalog_multiple_products_flavors.json",
			expectedConfig: &projects.StackDefinition{
				ID: core.StringPtr("mockProjectID"), // This would be generated on the server side and not part of the input
				StackDefinition: &projects.StackDefinitionBlock{
					Inputs: []projects.StackDefinitionInputVariable{
						{
							Name:        core.StringPtr("input1"),
							Type:        core.StringPtr("string"),
							Required:    core.BoolPtr(true),
							Default:     core.StringPtr("catalog_product2_default_flavor1"),
							Description: core.StringPtr(""),
							Hidden:      core.BoolPtr(false),
						},
						{
							Name:        core.StringPtr("input2"),
							Type:        core.StringPtr("int"),
							Required:    core.BoolPtr(false),
							Default:     core.Int64Ptr(85),
							Description: core.StringPtr(""),
							Hidden:      core.BoolPtr(false),
						},
						{
							Name:     core.StringPtr("input3"),
							Type:     core.StringPtr("array"),
							Required: core.BoolPtr(false),
							// not set in the catalog so should be the stack definition default
							Default:     core.StringPtr("[\"stack_def_arr_value1\", \"stack_def_arr_value2\"]"),
							Description: core.StringPtr(""),
							Hidden:      core.BoolPtr(false),
						},
					},
					Outputs: []projects.StackDefinitionOutputVariable{
						{Name: core.StringPtr("output1"), Value: core.StringPtr("ref:../members/member1/outputs/output1")},
						{Name: core.StringPtr("output2"), Value: core.StringPtr("ref:../members/member2/outputs/output2")},
					},
					// catalog can only configure stack level inputs, so the member inputs should be the same as the stack definition
					Members: []projects.StackDefinitionMember{
						{
							Name:           core.StringPtr("member1"),
							VersionLocator: core.StringPtr("version1"),
							Inputs: []projects.StackDefinitionMemberInput{
								{Name: core.StringPtr("input1"), Value: core.StringPtr("member1_input1")},
								{Name: core.StringPtr("input2"), Value: core.StringPtr("5")},
								{Name: core.StringPtr("input3"), Value: core.StringPtr("[\"member1_input3_value1\", \"member1_input3_value2\"]")},
							},
						},
						{
							Name:           core.StringPtr("member2"),
							VersionLocator: core.StringPtr("version2"),
							Inputs: []projects.StackDefinitionMemberInput{
								{Name: core.StringPtr("input1"), Value: core.StringPtr("member2_input1")},
								{Name: core.StringPtr("input2"), Value: core.StringPtr("6")},
								{Name: core.StringPtr("input3"), Value: core.StringPtr("[\"member2_input3_value1\", \"member2_input3_value2\"]")},
							},
						},
					},
				},
			},
			expectedError: nil,
		},
		{
			name: "catalog multiple flavors, checking values for correct flavor are selected",
			stackConfig: &ConfigDetails{
				ProjectID:          "mockProjectID",
				ConfigID:           "54321",
				CatalogProductName: "Second Product Name",
				CatalogFlavorName:  "Second Flavor Name",
			},
			stackConfigPath: "testdata/stack_definition_stack_inputs.json",
			catalogJsonPath: "testdata/ibm_catalog_multiple_products_flavors.json",
			expectedConfig: &projects.StackDefinition{
				ID: core.StringPtr("mockProjectID"), // This would be generated on the server side and not part of the input
				StackDefinition: &projects.StackDefinitionBlock{
					Inputs: []projects.StackDefinitionInputVariable{
						{
							Name:        core.StringPtr("input1"),
							Type:        core.StringPtr("string"),
							Required:    core.BoolPtr(true),
							Default:     core.StringPtr("product2_default_flavor2"),
							Description: core.StringPtr(""),
							Hidden:      core.BoolPtr(false),
						},
						{
							Name:        core.StringPtr("input2"),
							Type:        core.StringPtr("int"),
							Required:    core.BoolPtr(false),
							Default:     core.Int64Ptr(95),
							Description: core.StringPtr(""),
							Hidden:      core.BoolPtr(false),
						},
						{
							Name:     core.StringPtr("input3"),
							Type:     core.StringPtr("array"),
							Required: core.BoolPtr(false),
							// not set in the catalog so should be the stack definition default
							Default:     core.StringPtr("[\"stack_def_arr_value1\", \"stack_def_arr_value2\"]"),
							Description: core.StringPtr(""),
							Hidden:      core.BoolPtr(false),
						},
					},
					Outputs: []projects.StackDefinitionOutputVariable{
						{Name: core.StringPtr("output1"), Value: core.StringPtr("ref:../members/member1/outputs/output1")},
						{Name: core.StringPtr("output2"), Value: core.StringPtr("ref:../members/member2/outputs/output2")},
					},
					// catalog can only configure stack level inputs, so the member inputs should be the same as the stack definition
					Members: []projects.StackDefinitionMember{
						{
							Name:           core.StringPtr("member1"),
							VersionLocator: core.StringPtr("version1"),
							Inputs: []projects.StackDefinitionMemberInput{
								{Name: core.StringPtr("input1"), Value: core.StringPtr("ref:../../inputs/input1")},
								{Name: core.StringPtr("input2"), Value: core.StringPtr("20")},
								{Name: core.StringPtr("input3"), Value: core.StringPtr("stack_def_value3")},
							},
						},
						{
							Name:           core.StringPtr("member2"),
							VersionLocator: core.StringPtr("version2"),
							Inputs: []projects.StackDefinitionMemberInput{
								{Name: core.StringPtr("input1"), Value: core.StringPtr("ref:../../inputs/input2")},
								{Name: core.StringPtr("input2"), Value: core.StringPtr("30")},
								{Name: core.StringPtr("input3"), Value: core.StringPtr("stack_def_value4")},
							},
						},
					},
				},
			},
			expectedError: nil,
		},
		{
			name: "duplicate stack inputs, should return an error",
			stackConfig: &ConfigDetails{
				ProjectID: "mockProjectID",
				ConfigID:  "54321",
			},
			stackConfigPath: "testdata/stack_definition_duplicate_stack_inputs.json",
			catalogJsonPath: "testdata/ibm_catalog_no_config_overrides.json",
			expectedConfig:  nil,
			expectedError: fmt.Errorf("duplicate stack input variable found: input1\n" +
				"duplicate stack input variable found: input2"),
		},
		{
			name: "duplicate stack outputs, should return an error",
			stackConfig: &ConfigDetails{
				ProjectID: "mockProjectID",
				ConfigID:  "54321",
			},
			stackConfigPath: "testdata/stack_definition_duplicate_stack_outputs.json",
			catalogJsonPath: "testdata/ibm_catalog_no_config_overrides.json",
			expectedConfig:  nil,
			expectedError:   fmt.Errorf("duplicate stack output variable found: output1"),
		},
		{
			name: "duplicate member inputs, should return an error",
			stackConfig: &ConfigDetails{
				ProjectID: "mockProjectID",
				ConfigID:  "54321",
			},
			stackConfigPath: "testdata/stack_definition_duplicate_member_inputs.json",
			catalogJsonPath: "testdata/ibm_catalog_no_config_overrides.json",
			expectedConfig:  nil,
			expectedError:   fmt.Errorf("duplicate member input variable found member: member1 input: input1"),
		},
		{
			name: "catalog input not found in stack definition, should return an error",
			stackConfig: &ConfigDetails{
				ProjectID: "mockProjectID",
				ConfigID:  "54321",
			},
			stackConfigPath: "testdata/stack_definition_stack_inputs.json",
			catalogJsonPath: "testdata/ibm_catalog_extra_input.json",
			expectedConfig:  nil,
			expectedError:   fmt.Errorf("extra catalog input variable not found in stack definition in product 'Product Name', flavor 'Flavor Name': input5"),
		},
		{
			name: "catalog input duplicate found, should return an error",
			stackConfig: &ConfigDetails{
				ProjectID: "mockProjectID",
				ConfigID:  "54321",
			},
			stackConfigPath: "testdata/stack_definition_stack_inputs.json",
			catalogJsonPath: "testdata/ibm_catalog_duplicate_input.json",
			expectedConfig:  nil,
			expectedError:   fmt.Errorf("duplicate catalog input variable found in product 'Product Name', flavor 'Flavor Name': input1"),
		},
		{
			name: "catalog input type mismatch, should return an error",
			stackConfig: &ConfigDetails{
				ProjectID: "mockProjectID",
				ConfigID:  "54321",
			},
			stackConfigPath: "testdata/stack_definition_stack_inputs_extended.json",
			catalogJsonPath: "testdata/ibm_catalog_with_config_overrides_type_mismatch.json",
			expectedConfig:  nil,
			expectedError: fmt.Errorf("catalog configuration type mismatch in product 'Product Name', flavor 'Flavor Name': input1 expected type: string, got: array\n" +
				"catalog configuration type mismatch in product 'Product Name', flavor 'Flavor Name': input2 expected type: int, got: string\n" +
				"catalog configuration type mismatch in product 'Product Name', flavor 'Flavor Name': input3 expected type: array, got: bool\n" +
				"catalog configuration type mismatch in product 'Product Name', flavor 'Flavor Name': input4 expected type: bool, got: array"),
		},
		{
			name: "catalog input type_metadata mismatch, should return an error",
			stackConfig: &ConfigDetails{
				ProjectID: "mockProjectID",
				ConfigID:  "54321",
			},
			stackConfigPath: "testdata/stack_definition_with_type_metadata_only.json",
			catalogJsonPath: "testdata/ibm_catalog_with_type_metadata_only.json",
			expectedConfig:  nil,
			expectedError: fmt.Errorf("catalog configuration type_metadata mismatch in product 'Product Name', flavor 'Flavor Name': input5 expected type: string, got: int\n" +
				"catalog configuration type_metadata mismatch in product 'Product Name', flavor 'Flavor Name': input6 expected type: string, got: bool"),
		},
		{
			name: "catalog input with both type and type_metadata matching, should succeed",
			stackConfig: &ConfigDetails{
				ProjectID: "mockProjectID",
				ConfigID:  "54321",
			},
			stackConfigPath: "testdata/stack_definition_with_both_type_fields.json",
			catalogJsonPath: "testdata/ibm_catalog_with_both_type_fields_matching.json",
			expectedConfig: &projects.StackDefinition{
				ID: core.StringPtr("mockProjectID"),
				StackDefinition: &projects.StackDefinitionBlock{
					Inputs: []projects.StackDefinitionInputVariable{
						{
							Name:        core.StringPtr("input1"),
							Type:        core.StringPtr("string"),
							Required:    core.BoolPtr(false),
							Default:     core.StringPtr("default_value_1"),
							Description: core.StringPtr(""),
							Hidden:      core.BoolPtr(false),
						},
						{
							Name:        core.StringPtr("input2"),
							Type:        core.StringPtr("string"),
							Required:    core.BoolPtr(false),
							Default:     core.StringPtr("default_value_2"),
							Description: core.StringPtr(""),
							Hidden:      core.BoolPtr(false),
						},
						{
							Name:        core.StringPtr("input3"),
							Type:        core.StringPtr("string"),
							Required:    core.BoolPtr(false),
							Default:     core.StringPtr("default_value_3"),
							Description: core.StringPtr(""),
							Hidden:      core.BoolPtr(false),
						},
					},
					Outputs: []projects.StackDefinitionOutputVariable{
						{Name: core.StringPtr("output1"), Value: core.StringPtr("ref:../members/member1/outputs/output1")},
					},
					Members: []projects.StackDefinitionMember{
						{
							Name:           core.StringPtr("member1"),
							VersionLocator: core.StringPtr("version1"),
							Inputs: []projects.StackDefinitionMemberInput{
								{Name: core.StringPtr("input1"), Value: core.StringPtr("ref:../../inputs/input1")},
							},
						},
					},
				},
			},
			expectedError: nil,
		},
		{
			name: "catalog input with both type and type_metadata conflicting, should return error",
			stackConfig: &ConfigDetails{
				ProjectID: "mockProjectID",
				ConfigID:  "54321",
			},
			stackConfigPath: "testdata/stack_definition_with_both_type_fields.json",
			catalogJsonPath: "testdata/ibm_catalog_with_both_type_fields.json",
			expectedConfig:  nil,
			expectedError:   fmt.Errorf("catalog configuration type mismatch in product 'Product Name', flavor 'Flavor Name': input2 expected type: string, got: int"),
		},
		{
			// This is checking the type of the actual default value
			name: "catalog input default type mismatch, should return an error",
			stackConfig: &ConfigDetails{
				ProjectID: "mockProjectID",
				ConfigID:  "54321",
			},
			stackConfigPath: "testdata/stack_definition_stack_inputs_extended.json",
			catalogJsonPath: "testdata/ibm_catalog_with_config_overrides_value_type_mismatch.json",
			expectedConfig:  nil,
			expectedError: fmt.Errorf("catalog configuration default value type mismatch in product 'Product Name', flavor 'Flavor Name': input1 expected type: string, got: bool\n" +
				"catalog configuration default value type mismatch in product 'Product Name', flavor 'Flavor Name': input2 expected type: int, got: string\n" +
				"catalog configuration default value type mismatch in product 'Product Name', flavor 'Flavor Name': input3 expected type: array, got: string\n" +
				"catalog configuration default value type mismatch in product 'Product Name', flavor 'Flavor Name': input4 expected type: bool, got: string"),
		},
		{
			name: "multiple duplicates or extra inputs, should return a single error with multiple messages",
			stackConfig: &ConfigDetails{
				ProjectID: "mockProjectID",
				ConfigID:  "54321",
			},
			stackConfigPath: "testdata/stack_definition_multiple_stack_errors.json",
			catalogJsonPath: "testdata/ibm_catalog_multiple_errors.json",
			expectedConfig:  nil,
			expectedError: fmt.Errorf(
				"duplicate stack input variable found: input1\n" +
					"duplicate stack input variable found: input2\n" +
					"duplicate stack output variable found: output1\n" +
					"duplicate member input variable found member: member1 input: input1\n" +
					"duplicate catalog input variable found in product 'Product Name', flavor 'Flavor Name': input1\n" +
					"catalog configuration default value type mismatch in product 'Product Name', flavor 'Flavor Name': input2 expected type: int, got: string\n" +
					"extra catalog input variable not found in stack definition in product 'Product Name', flavor 'Flavor Name': input5"),
		},
		{
			name: "invalid product name, should return an error",
			stackConfig: &ConfigDetails{
				ProjectID:          "mockProjectID",
				ConfigID:           "54321",
				CatalogProductName: "Non-Existent Product",
			},
			stackConfigPath: "testdata/stack_definition_stack_inputs.json",
			catalogJsonPath: "testdata/ibm_catalog_multiple_products_flavors.json",
			expectedConfig:  nil,
			expectedError:   fmt.Errorf("product name 'Non-Existent Product' not found in catalog JSON"),
		},
		{
			name: "invalid flavor name, should return an error",
			stackConfig: &ConfigDetails{
				ProjectID:          "mockProjectID",
				ConfigID:           "54321",
				CatalogProductName: "Second Product Name",
				CatalogFlavorName:  "Non-Existent Flavor",
			},
			stackConfigPath: "testdata/stack_definition_stack_inputs.json",
			catalogJsonPath: "testdata/ibm_catalog_multiple_products_flavors.json",
			expectedConfig:  nil,
			expectedError:   fmt.Errorf("flavor name 'Non-Existent Flavor' not found in catalog JSON for product 'Second Product Name'"),
		},
		{
			// custom_config widget types with no top-level type and no default should resolve to "string".
			name: "catalog with custom_config region fields (no top-level type, no default), should resolve to string",
			stackConfig: &ConfigDetails{
				ProjectID: "mockProjectID",
				ConfigID:  "54321",
			},
			stackConfigPath: "testdata/stack_definition_custom_config_region.json",
			catalogJsonPath: "testdata/ibm_catalog_custom_config_region.json",
			expectedConfig: &projects.StackDefinition{
				ID: core.StringPtr("mockProjectID"),
				StackDefinition: &projects.StackDefinitionBlock{
					Inputs: []projects.StackDefinitionInputVariable{
						{
							Name:        core.StringPtr("cos_region"),
							Type:        core.StringPtr("string"),
							Required:    core.BoolPtr(true),
							Default:     core.StringPtr("__NULL__"),
							Description: core.StringPtr(""),
							Hidden:      core.BoolPtr(false),
						},
						{
							Name:        core.StringPtr("region"),
							Type:        core.StringPtr("string"),
							Required:    core.BoolPtr(true),
							Default:     core.StringPtr("__NULL__"),
							Description: core.StringPtr(""),
							Hidden:      core.BoolPtr(false),
						},
					},
					Outputs: []projects.StackDefinitionOutputVariable{
						{Name: core.StringPtr("output1"), Value: core.StringPtr("ref:../members/member1/outputs/output1")},
					},
					Members: []projects.StackDefinitionMember{
						{
							Name:           core.StringPtr("member1"),
							VersionLocator: core.StringPtr("version1"),
							Inputs: []projects.StackDefinitionMemberInput{
								{Name: core.StringPtr("region"), Value: core.StringPtr("ref:../../inputs/region")},
							},
						},
					},
				},
			},
			expectedError: nil,
		},
		{
			// Fields with both top-level type:"string" and custom_config.type:"region", required, no default should resolve to "string".
			name: "catalog with top-level type:string and custom_config.type:region (no default), should resolve to string",
			stackConfig: &ConfigDetails{
				ProjectID: "mockProjectID",
				ConfigID:  "54321",
			},
			stackConfigPath: "testdata/stack_definition_custom_config_region.json",
			catalogJsonPath: "testdata/ibm_catalog_custom_config_with_type.json",
			expectedConfig: &projects.StackDefinition{
				ID: core.StringPtr("mockProjectID"),
				StackDefinition: &projects.StackDefinitionBlock{
					Inputs: []projects.StackDefinitionInputVariable{
						{
							Name:        core.StringPtr("cos_region"),
							Type:        core.StringPtr("string"),
							Required:    core.BoolPtr(true),
							Default:     core.StringPtr("__NULL__"),
							Description: core.StringPtr(""),
							Hidden:      core.BoolPtr(false),
						},
						{
							Name:        core.StringPtr("region"),
							Type:        core.StringPtr("string"),
							Required:    core.BoolPtr(true),
							Default:     core.StringPtr("__NULL__"),
							Description: core.StringPtr(""),
							Hidden:      core.BoolPtr(false),
						},
					},
					Outputs: []projects.StackDefinitionOutputVariable{
						{Name: core.StringPtr("output1"), Value: core.StringPtr("ref:../members/member1/outputs/output1")},
					},
					Members: []projects.StackDefinitionMember{
						{
							Name:           core.StringPtr("member1"),
							VersionLocator: core.StringPtr("version1"),
							Inputs: []projects.StackDefinitionMemberInput{
								{Name: core.StringPtr("region"), Value: core.StringPtr("ref:../../inputs/region")},
							},
						},
					},
				},
			},
			expectedError: nil,
		},
		{
			// Proves the effectiveCatalogType fallback is active: without it, type "" skips the mismatch
			// check silently; with it, type "string" is compared against stack "int" and an error is returned.
			name: "catalog with custom_config region fields (no top-level type) mismatching stack int type, should return error",
			stackConfig: &ConfigDetails{
				ProjectID: "mockProjectID",
				ConfigID:  "54321",
			},
			stackConfigPath: "testdata/stack_definition_custom_config_region_type_mismatch.json",
			catalogJsonPath: "testdata/ibm_catalog_custom_config_region.json",
			expectedConfig:  nil,
			expectedError:   fmt.Errorf("catalog configuration type mismatch in product 'Product Name', flavor 'Flavor Name': cos_region expected type: int, got: string\ncatalog configuration type mismatch in product 'Product Name', flavor 'Flavor Name': region expected type: int, got: string"),
		},
		{
			name: "catalog with HCL string defaults for array/object types, should pass validation",
			stackConfig: &ConfigDetails{
				ProjectID: "mockProjectID",
				ConfigID:  "54321",
			},
			stackConfigPath: "testdata/stack_definition_hcl_string_default.json",
			catalogJsonPath: "testdata/ibm_catalog_hcl_string_default.json",
			expectedConfig: &projects.StackDefinition{
				ID: core.StringPtr("mockProjectID"),
				StackDefinition: &projects.StackDefinitionBlock{
					Inputs: []projects.StackDefinitionInputVariable{
						{
							Name:        core.StringPtr("config_object"),
							Type:        core.StringPtr("object"),
							Required:    core.BoolPtr(false),
							Default:     core.StringPtr("{\n    setting1 = \"value1\"\n    setting2 = 42\n    nested   = {\n      key = \"value\"\n    }\n  }"),
							Description: core.StringPtr(""),
							Hidden:      core.BoolPtr(false),
						},
						{
							Name:        core.StringPtr("region"),
							Type:        core.StringPtr("string"),
							Required:    core.BoolPtr(true),
							Default:     core.StringPtr("us-south"),
							Description: core.StringPtr(""),
							Hidden:      core.BoolPtr(false),
						},
						{
							Name:        core.StringPtr("secret_groups"),
							Type:        core.StringPtr("array"),
							Required:    core.BoolPtr(false),
							Default:     core.StringPtr("[\n    {\n      secret_group_name        = \"General\"\n      secret_group_description = \"A general purpose secrets group with an associated access group which has a secrets reader role\"\n      create_access_group      = true\n      access_group_name        = \"general-secrets-group-access-group\"\n      access_group_roles       = [\"SecretsReader\"]\n    }\n  ]"),
							Description: core.StringPtr(""),
							Hidden:      core.BoolPtr(false),
						},
					},
					Outputs: []projects.StackDefinitionOutputVariable{
						{Name: core.StringPtr("output1"), Value: core.StringPtr("ref:../members/member1/outputs/output1")},
					},
					Members: []projects.StackDefinitionMember{
						{
							Name:           core.StringPtr("member1"),
							VersionLocator: core.StringPtr("version1"),
							Inputs: []projects.StackDefinitionMemberInput{
								{Name: core.StringPtr("region"), Value: core.StringPtr("ref:../../inputs/region")},
							},
						},
					},
				},
			},
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {

			// Mock the CreateConfig call
			suite.mockService.On("CreateConfig", mock.Anything).Return(
				&projects.ProjectConfig{},
				&core.DetailedResponse{}, nil)

			// Mock the NewCreateConfigOptions call
			suite.mockService.On("NewCreateConfigOptions", mock.Anything, mock.Anything).Return(
				&projects.CreateConfigOptions{})

			// Mock the NewCreateStackDefinitionOptions call
			suite.mockService.On("NewCreateStackDefinitionOptions", mock.Anything, mock.Anything).Return(
				&projects.CreateStackDefinitionOptions{})

			// Mock the CreateStackDefinition call
			suite.mockCreator.On("CreateStackDefinitionWrapper", mock.Anything, mock.Anything).Return(
				nil, &core.DetailedResponse{}, nil)

			result, _, err := suite.infoSvc.CreateStackFromConfigFile(tc.stackConfig, tc.stackConfigPath, tc.catalogJsonPath)

			if tc.expectedError == nil {
				if assert.NoError(suite.T(), err) {
					assert.EqualValues(suite.T(), SortStackDefinition(tc.expectedConfig), SortStackDefinition(result))
				}
			} else {
				if assert.Error(suite.T(), err) {
					assert.Equal(suite.T(), tc.expectedError.Error(), err.Error())
				}
			}

		})
	}
}

func (suite *ProjectsServiceTestSuite) TestGetMemberWithWorkspaceInfo_Success() {
	projectID := "test-project-id"
	configID := "test-config-id"

	// Mock member config with workspace info and job ID populated
	mockMember := &projects.ProjectConfig{
		ID: core.StringPtr(configID),
		Schematics: &projects.SchematicsMetadata{
			WorkspaceCrn: core.StringPtr("crn:v1:bluemix:public:schematics:us-south:a/abc123::workspace:ws-123"),
		},
		LastValidated: &projects.LastValidatedActionWithSummary{
			Href: core.StringPtr("https://example.com"),
			Job: &projects.ActionJobWithIdAndSummary{
				ID:      core.StringPtr("job-123"),
				Summary: &projects.ActionJobSummary{},
			},
		},
	}

	mockResponse := &core.DetailedResponse{StatusCode: 200}

	// Mock GetConfig to return member with workspace info on first try
	suite.mockService.On("GetConfig", mock.MatchedBy(func(opts *projects.GetConfigOptions) bool {
		return *opts.ProjectID == projectID && *opts.ID == configID
	})).Return(mockMember, mockResponse, nil)

	// Call the function
	result, err := suite.infoSvc.GetMemberWithWorkspaceInfo(projectID, configID)

	// Verify
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), configID, *result.ID)
	assert.NotNil(suite.T(), result.Schematics)
	assert.NotNil(suite.T(), result.Schematics.WorkspaceCrn)
	assert.NotNil(suite.T(), result.LastValidated)
	assert.NotNil(suite.T(), result.LastValidated.Job)
	assert.NotNil(suite.T(), result.LastValidated.Job.ID)
}

func (suite *ProjectsServiceTestSuite) TestGetMemberWithWorkspaceInfo_EventualConsistency() {
	// Set environment variable to skip retry delays for faster testing
	originalSkipDelays := os.Getenv("SKIP_RETRY_DELAYS")
	os.Setenv("SKIP_RETRY_DELAYS", "true")
	defer os.Setenv("SKIP_RETRY_DELAYS", originalSkipDelays)

	projectID := "test-project-id"
	configID := "test-config-id"

	// First call: Member without workspace info (eventual consistency)
	memberWithoutInfo := &projects.ProjectConfig{
		ID:         core.StringPtr(configID),
		Schematics: nil, // No workspace info yet
	}

	// Second call: Member with workspace info populated
	memberWithInfo := &projects.ProjectConfig{
		ID: core.StringPtr(configID),
		Schematics: &projects.SchematicsMetadata{
			WorkspaceCrn: core.StringPtr("crn:v1:bluemix:public:schematics:us-south:a/abc123::workspace:ws-123"),
		},
		LastDeployed: &projects.LastActionWithSummary{
			Href: core.StringPtr("https://example.com"),
			Job: &projects.ActionJobWithIdAndSummary{
				ID:      core.StringPtr("job-456"),
				Summary: &projects.ActionJobSummary{},
			},
		},
	}

	mockResponse := &core.DetailedResponse{StatusCode: 200}

	// Mock GetConfig to return different results on sequential calls
	suite.mockService.On("GetConfig", mock.MatchedBy(func(opts *projects.GetConfigOptions) bool {
		return *opts.ProjectID == projectID && *opts.ID == configID
	})).Return(memberWithoutInfo, mockResponse, nil).Once()

	suite.mockService.On("GetConfig", mock.MatchedBy(func(opts *projects.GetConfigOptions) bool {
		return *opts.ProjectID == projectID && *opts.ID == configID
	})).Return(memberWithInfo, mockResponse, nil)

	// Call the function - it should retry and eventually get the complete data
	result, err := suite.infoSvc.GetMemberWithWorkspaceInfo(projectID, configID)

	// Verify
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), configID, *result.ID)
	assert.NotNil(suite.T(), result.Schematics)
	assert.NotNil(suite.T(), result.Schematics.WorkspaceCrn)
	assert.NotNil(suite.T(), result.LastDeployed)
	assert.NotNil(suite.T(), result.LastDeployed.Job)
	assert.NotNil(suite.T(), result.LastDeployed.Job.ID)

	// Verify GetConfig was called at least twice (retry happened)
	suite.mockService.AssertNumberOfCalls(suite.T(), "GetConfig", 2)
}

func TestProjectsServiceTestSuite(t *testing.T) {
	suite.Run(t, new(ProjectsServiceTestSuite))
}

func TestValidateCatalogNames(t *testing.T) {
	catalogPath := "testdata/ibm_catalog_multiple_products_flavors.json"

	tests := []struct {
		name        string
		productName string
		flavorName  string
		// jsonInput is set for edge cases that need an inline catalog struct
		// instead of the file-based catalogPath. When set, lookupCatalogIndices
		// is called directly to avoid needing fixture files.
		jsonInput string
		expectErr string
	}{
		{
			name:        "valid product and flavor",
			productName: "Second Product Name",
			flavorName:  "Second Flavor Name",
		},
		{
			name:        "valid product, empty flavor defaults to first",
			productName: "First Product Name",
		},
		{
			name: "empty product and flavor, defaults to first of each",
		},
		{
			name:        "invalid product name",
			productName: "Non-Existent Product",
			expectErr:   "product name 'Non-Existent Product' not found in catalog JSON",
		},
		{
			name:        "valid product, invalid flavor name",
			productName: "Second Product Name",
			flavorName:  "Non-Existent Flavor",
			expectErr:   "flavor name 'Non-Existent Flavor' not found in catalog JSON for product 'Second Product Name'",
		},
		{
			name:      "catalog with no products returns error not panic",
			jsonInput: `{"products":[]}`,
			expectErr: "catalog JSON contains no products",
		},
		{
			name:      "catalog with no flavors returns error not panic",
			jsonInput: `{"products":[{"name":"Empty Product","flavors":[]}]}`,
			expectErr: "catalog JSON contains no flavors for product 'Empty Product'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			if tt.jsonInput != "" {
				var catalog CatalogJson
				if assert.NoError(t, json.Unmarshal([]byte(tt.jsonInput), &catalog)) {
					_, _, err = lookupCatalogIndices(catalog, "", "")
				} else {
					return
				}
			} else {
				err = ValidateCatalogNames(catalogPath, tt.productName, tt.flavorName)
			}
			if tt.expectErr == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tt.expectErr)
			}
		})
	}
}

// SortStackDefinition Helper function to sort the StackDefinition and all nested slices
// Sorts StackDefinition and all nested slices, this is needed because the order of the elements in the JSON file is not guaranteed
// and the order of the elements in the StackDefinition is important for the tests
func SortStackDefinition(stackDef *projects.StackDefinition) *projects.StackDefinition {
	if stackDef == nil {
		return nil
	}

	// Sort the StackDefinitionBlock (Inputs, Outputs, Members)
	if stackDef.StackDefinition != nil {
		stackDef.StackDefinition.Inputs = SortStackDefinitionInputVariables(stackDef.StackDefinition.Inputs)
		stackDef.StackDefinition.Outputs = SortStackDefinitionOutputVariables(stackDef.StackDefinition.Outputs)
		stackDef.StackDefinition.Members = SortStackDefinitionMembers(stackDef.StackDefinition.Members)
	}

	return stackDef
}

// Sorts a slice of StackDefinitionInputVariable by Name
func SortStackDefinitionInputVariables(inputs []projects.StackDefinitionInputVariable) []projects.StackDefinitionInputVariable {
	sort.SliceStable(inputs, func(i, j int) bool {
		return *inputs[i].Name < *inputs[j].Name
	})
	return inputs
}

// Sorts a slice of StackDefinitionOutputVariable by Name
func SortStackDefinitionOutputVariables(outputs []projects.StackDefinitionOutputVariable) []projects.StackDefinitionOutputVariable {
	sort.SliceStable(outputs, func(i, j int) bool {
		return *outputs[i].Name < *outputs[j].Name
	})
	return outputs
}

// Sorts a slice of StackDefinitionMember by Name, and sorts their Inputs
func SortStackDefinitionMembers(members []projects.StackDefinitionMember) []projects.StackDefinitionMember {
	sort.SliceStable(members, func(i, j int) bool {
		return *members[i].Name < *members[j].Name
	})

	// Sort the Inputs within each Member
	for i := range members {
		members[i].Inputs = SortStackDefinitionMemberInputs(members[i].Inputs)
	}
	return members
}

// Sorts a slice of StackDefinitionMemberInput by Name
func SortStackDefinitionMemberInputs(inputs []projects.StackDefinitionMemberInput) []projects.StackDefinitionMemberInput {
	sort.SliceStable(inputs, func(i, j int) bool {
		return *inputs[i].Name < *inputs[j].Name
	})
	return inputs
}

// TestCreateStackDefinitionWrapperRetry covers the transient "The config cannot be found"
// response the Projects API can return while the configs created moments earlier in
// processMembers become readable. That 404 is retried; anything else is not, because
// creating a stack definition is not idempotent.
func (suite *ProjectsServiceTestSuite) TestCreateStackDefinitionWrapperRetry() {
	// Keep the retry counting but skip the backoff sleeps.
	// common.calculateDelay skips when SKIP_RETRY_DELAYS == "true"
	suite.T().Setenv("SKIP_RETRY_DELAYS", "true")

	mockResponse := &core.DetailedResponse{StatusCode: 201}
	stackDefOptions := &projects.CreateStackDefinitionOptions{
		ProjectID: core.StringPtr("test-project-id"),
		ID:        core.StringPtr("test-config-id"),
		StackDefinition: &projects.StackDefinitionBlockPrototype{
			Inputs: []projects.StackDefinitionInputVariable{},
		},
	}

	suite.Run("ConfigNotFoundIsRetriedThenSucceeds", func() {
		suite.mockService.ExpectedCalls = nil
		suite.mockService.Calls = nil
		notFound := core.SDKErrorf(nil, "The config cannot be found", "http-request-err",
			core.NewProblemComponent("project", "v1"))

		suite.mockService.On("CreateStackDefinition", mock.Anything).
			Return([]projects.StackDefinitionMember{}, (*core.DetailedResponse)(nil), notFound).Once()
		suite.mockService.On("CreateStackDefinition", mock.Anything).
			Return([]projects.StackDefinitionMember{}, mockResponse, nil).Once()

		result, response, err := suite.infoSvc.CreateStackDefinitionWrapper(stackDefOptions, nil)

		assert.NoError(suite.T(), err)
		assert.NotNil(suite.T(), result)
		assert.Equal(suite.T(), mockResponse, response)
		suite.mockService.AssertExpectations(suite.T())
	})

	suite.Run("OtherErrorIsNotRetried", func() {
		suite.mockService.ExpectedCalls = nil
		suite.mockService.Calls = nil
		// A non-idempotent create must not be re-sent on an ambiguous failure.
		otherErr := core.SDKErrorf(nil, "A stack definition member input foo was not found in the configuration bar.",
			"http-request-err", core.NewProblemComponent("project", "v1"))

		suite.mockService.On("CreateStackDefinition", mock.Anything).
			Return([]projects.StackDefinitionMember{}, (*core.DetailedResponse)(nil), otherErr).Once()

		_, _, err := suite.infoSvc.CreateStackDefinitionWrapper(stackDefOptions, nil)

		assert.Error(suite.T(), err)
		assert.Contains(suite.T(), err.Error(), "was not found in the configuration")
		// Exactly one call: no retry was attempted.
		suite.mockService.AssertNumberOfCalls(suite.T(), "CreateStackDefinition", 1)
	})

	suite.Run("SuccessOnFirstAttemptMakesOneCall", func() {
		suite.mockService.ExpectedCalls = nil
		suite.mockService.Calls = nil
		suite.mockService.On("CreateStackDefinition", mock.Anything).
			Return([]projects.StackDefinitionMember{}, mockResponse, nil).Once()

		result, response, err := suite.infoSvc.CreateStackDefinitionWrapper(stackDefOptions, nil)

		assert.NoError(suite.T(), err)
		assert.NotNil(suite.T(), result)
		assert.Equal(suite.T(), mockResponse, response)
		suite.mockService.AssertNumberOfCalls(suite.T(), "CreateStackDefinition", 1)
	})
}
