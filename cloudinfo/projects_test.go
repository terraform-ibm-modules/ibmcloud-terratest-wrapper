package cloudinfo

import (
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
