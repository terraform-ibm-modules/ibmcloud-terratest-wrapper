package cloudinfo

import (
	"errors"
	"fmt"
	"testing"

	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/platform-services-go-sdk/catalogmanagementv1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/common"
)

// MockCatalogCloudInfoService is a mock implementation for testing catalog functionality
type MockCatalogCloudInfoService struct {
	CloudInfoService
	MockGetComponentReferences func(versionLocator string) (*OfferingReferenceResponse, error)
}

// GetComponentReferences overrides the implementation to use our mock function
func (m *MockCatalogCloudInfoService) GetComponentReferences(versionLocator string) (*OfferingReferenceResponse, error) {
	if m.MockGetComponentReferences != nil {
		return m.MockGetComponentReferences(versionLocator)
	}
	return m.CloudInfoService.GetComponentReferences(versionLocator)
}

// MockComponentReferenceGetter is a simple mock that implements ComponentReferenceGetter interface
type MockComponentReferenceGetter struct {
	MockGetComponentReferences func(versionLocator string) (*OfferingReferenceResponse, error)
}

func (m *MockComponentReferenceGetter) GetComponentReferences(versionLocator string) (*OfferingReferenceResponse, error) {
	if m.MockGetComponentReferences != nil {
		return m.MockGetComponentReferences(versionLocator)
	}
	return nil, fmt.Errorf("not mocked")
}

type CatalogServiceTestSuite struct {
	suite.Suite
	mockService *catalogServiceMock
	infoSvc     *MockCatalogCloudInfoService
}

func (suite *CatalogServiceTestSuite) SetupTest() {
	suite.mockService = new(catalogServiceMock)
	suite.infoSvc = &MockCatalogCloudInfoService{
		CloudInfoService: CloudInfoService{
			catalogService: suite.mockService,
			authenticator: &MockAuthenticator{
				Token: "mock-token",
			},
			Logger: common.NewTestLogger("CatalogServiceTest"),
		},
	}
}

func (suite *CatalogServiceTestSuite) TestGetCatalogVersionByLocator() {
	versionLocator := "test-version-locator"
	mockVersion := &catalogmanagementv1.Version{
		Version: core.StringPtr("1.0.0"),
		ID:      core.StringPtr("version-id"),
	}
	mockOffering := &catalogmanagementv1.Offering{
		ID: core.StringPtr("offering-id"),
		Kinds: []catalogmanagementv1.Kind{
			{
				ID: core.StringPtr("kind-id"),
				Versions: []catalogmanagementv1.Version{
					*mockVersion,
				},
			},
		},
	}
	mockResponse := &core.DetailedResponse{StatusCode: 200}
	mockError := fmt.Errorf("error getting version")

	testCases := []struct {
		name           string
		expectedError  error
		mockError      error
		expectedResult *catalogmanagementv1.Version
		mockResult     *catalogmanagementv1.Offering
		mockResponse   *core.DetailedResponse
	}{
		{
			name:           "Success case",
			expectedError:  nil,
			mockError:      nil,
			expectedResult: mockVersion,
			mockResult:     mockOffering,
			mockResponse:   mockResponse,
		},
		{
			name:           "Failure case - API error",
			expectedError:  mockError,
			mockError:      mockError,
			expectedResult: nil,
			mockResult:     nil,
			mockResponse:   nil,
		},
		{
			name:           "Failure case - empty offering",
			expectedError:  errors.New("version not found"),
			mockError:      nil,
			expectedResult: nil,
			mockResult:     &catalogmanagementv1.Offering{},
			mockResponse:   mockResponse,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Clear previous expectations
			suite.mockService.ExpectedCalls = nil

			// Fix: Use the correct parameter type (GetVersionOptions instead of string)
			suite.mockService.On("GetVersion", mock.MatchedBy(func(opts *catalogmanagementv1.GetVersionOptions) bool {
				return opts != nil && *opts.VersionLocID == versionLocator
			})).Return(tc.mockResult, tc.mockResponse, tc.mockError)

			result, err := suite.infoSvc.GetCatalogVersionByLocator(versionLocator)
			if tc.expectedError != nil {
				assert.Error(suite.T(), err)
				assert.Nil(suite.T(), result)
			} else {
				assert.NoError(suite.T(), err)
				assert.Equal(suite.T(), tc.expectedResult, result)
			}
		})
	}
}

func (suite *CatalogServiceTestSuite) TestCreateCatalog() {
	catalogName := "test-catalog"
	mockCatalog := &catalogmanagementv1.Catalog{
		ID:    core.StringPtr("catalog-id"),
		Label: core.StringPtr(catalogName),
	}
	mockResponse := &core.DetailedResponse{StatusCode: 201}
	mockError := fmt.Errorf("error creating catalog")

	testCases := []struct {
		name           string
		expectedError  error
		mockError      error
		expectedResult *catalogmanagementv1.Catalog
		mockResult     *catalogmanagementv1.Catalog
		mockResponse   *core.DetailedResponse
	}{
		{
			name:           "Success case",
			expectedError:  nil,
			mockError:      nil,
			expectedResult: mockCatalog,
			mockResult:     mockCatalog,
			mockResponse:   mockResponse,
		},
		{
			name:           "Failure case - API error",
			expectedError:  mockError,
			mockError:      mockError,
			expectedResult: nil,
			mockResult:     nil,
			mockResponse:   nil,
		},
		{
			name:           "Failure case - non-201 status code",
			expectedError:  errors.New("failed to create catalog: "),
			mockError:      nil,
			expectedResult: nil,
			mockResult:     mockCatalog,
			mockResponse:   &core.DetailedResponse{StatusCode: 400},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Clear previous expectations
			suite.mockService.ExpectedCalls = nil

			// Improve mock by expecting the specific catalog name
			suite.mockService.On("CreateCatalog", mock.MatchedBy(func(opts *catalogmanagementv1.CreateCatalogOptions) bool {
				return opts != nil && *opts.Label == catalogName
			})).Return(tc.mockResult, tc.mockResponse, tc.mockError)

			result, err := suite.infoSvc.CreateCatalog(catalogName)
			if tc.expectedError != nil {
				assert.Error(suite.T(), err)
				assert.Nil(suite.T(), result)
			} else {
				assert.NoError(suite.T(), err)
				assert.Equal(suite.T(), tc.expectedResult, result)
			}
		})
	}
}

func (suite *CatalogServiceTestSuite) TestDeleteCatalog() {
	catalogID := "test-catalog-id"
	mockResponse := &core.DetailedResponse{StatusCode: 200}
	mockError := fmt.Errorf("error deleting catalog")

	testCases := []struct {
		name          string
		expectedError error
		mockError     error
		mockResponse  *core.DetailedResponse
	}{
		{
			name:          "Success case",
			expectedError: nil,
			mockError:     nil,
			mockResponse:  mockResponse,
		},
		{
			name:          "Failure case - API error",
			expectedError: mockError,
			mockError:     mockError,
			mockResponse:  nil,
		},
		{
			name:          "Failure case - non-200 status code",
			expectedError: errors.New("failed to delete catalog: "),
			mockError:     nil,
			mockResponse:  &core.DetailedResponse{StatusCode: 400},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Clear previous expectations
			suite.mockService.ExpectedCalls = nil

			// Improve mock by expecting the specific catalog ID
			suite.mockService.On("DeleteCatalog", mock.MatchedBy(func(opts *catalogmanagementv1.DeleteCatalogOptions) bool {
				return opts != nil && *opts.CatalogIdentifier == catalogID
			})).Return(tc.mockResponse, tc.mockError)

			err := suite.infoSvc.DeleteCatalog(catalogID)
			if tc.expectedError != nil {
				assert.Error(suite.T(), err)
			} else {
				assert.NoError(suite.T(), err)
			}
		})
	}
}

func (suite *CatalogServiceTestSuite) TestImportOffering() {
	catalogID := "test-catalog-id"
	zipUrl := "https://example.com/archive.zip"
	offeringName := "test-offering"
	flavorName := "test-flavor"
	version := "1.0.0"
	mockOffering := &catalogmanagementv1.Offering{
		ID: core.StringPtr("offering-id"),
	}
	mockResponse := &core.DetailedResponse{StatusCode: 201}
	mockError := fmt.Errorf("error importing offering")

	testCases := []struct {
		name           string
		expectedError  error
		mockError      error
		expectedResult *catalogmanagementv1.Offering
		mockResult     *catalogmanagementv1.Offering
		mockResponse   *core.DetailedResponse
		installKind    *InstallKind
	}{
		{
			name:           "Success case - Terraform",
			expectedError:  nil,
			mockError:      nil,
			expectedResult: mockOffering,
			mockResult:     mockOffering,
			mockResponse:   mockResponse,
			installKind:    NewInstallKindTerraform(),
		},
		{
			name:           "Success case - Stack",
			expectedError:  nil,
			mockError:      nil,
			expectedResult: mockOffering,
			mockResult:     mockOffering,
			mockResponse:   mockResponse,
			installKind:    NewInstallKindStack(),
		},
		{
			name:           "Failure case - API error",
			expectedError:  mockError,
			mockError:      mockError,
			expectedResult: nil,
			mockResult:     nil,
			mockResponse:   nil,
			installKind:    NewInstallKindTerraform(),
		},
		{
			name:           "Failure case - non-201 status code",
			expectedError:  errors.New("failed to import offering: "),
			mockError:      nil,
			expectedResult: nil,
			mockResult:     mockOffering,
			mockResponse:   &core.DetailedResponse{StatusCode: 400},
			installKind:    NewInstallKindTerraform(),
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Clear previous expectations
			suite.mockService.ExpectedCalls = nil

			// Fix: Use safer parameter checking that won't panic on nil values
			suite.mockService.On("ImportOffering", mock.MatchedBy(func(opts *catalogmanagementv1.ImportOfferingOptions) bool {
				if opts == nil {
					return false
				}

				// Only check required fields that we know should be present
				catalogIDMatch := opts.CatalogIdentifier != nil && *opts.CatalogIdentifier == catalogID
				zipUrlMatch := opts.Zipurl != nil && *opts.Zipurl == zipUrl

				// We can also check these fields if they are important
				// But we use safer nil checks to avoid panics
				return catalogIDMatch && zipUrlMatch
			})).Return(tc.mockResult, tc.mockResponse, tc.mockError)

			// Dereference the pointer to get the InstallKind value
			var installKind InstallKind
			if tc.installKind != nil {
				installKind = *tc.installKind
			}
			result, err := suite.infoSvc.ImportOffering(catalogID, zipUrl, offeringName, flavorName, version, installKind)
			if tc.expectedError != nil {
				assert.Error(suite.T(), err)
				assert.Nil(suite.T(), result)
			} else {
				assert.NoError(suite.T(), err)
				assert.Equal(suite.T(), tc.expectedResult, result)
			}
		})
	}
}

func (suite *CatalogServiceTestSuite) TestGetOffering() {
	catalogID := "test-catalog-id"
	offeringID := "test-offering-id"
	mockOffering := &catalogmanagementv1.Offering{
		ID: core.StringPtr(offeringID),
	}
	mockResponse := &core.DetailedResponse{StatusCode: 200}
	mockError := fmt.Errorf("error getting offering")

	testCases := []struct {
		name           string
		expectedError  error
		mockError      error
		expectedResult *catalogmanagementv1.Offering
		mockResult     *catalogmanagementv1.Offering
		mockResponse   *core.DetailedResponse
		installKind    *InstallKind
	}{
		{
			name:           "Success case - Terraform",
			expectedError:  nil,
			mockError:      nil,
			expectedResult: mockOffering,
			mockResult:     mockOffering,
			mockResponse:   mockResponse,
			installKind:    NewInstallKindTerraform(),
		},
		{
			name:           "Success case - Stack",
			expectedError:  nil,
			mockError:      nil,
			expectedResult: mockOffering,
			mockResult:     mockOffering,
			mockResponse:   mockResponse,
			installKind:    NewInstallKindStack(),
		},
		{
			name:           "Failure case - API error",
			expectedError:  mockError,
			mockError:      mockError,
			expectedResult: nil,
			mockResult:     nil,
			mockResponse:   nil,
			installKind:    NewInstallKindTerraform(),
		},
		{
			name:           "Failure case - non-200 status code",
			expectedError:  errors.New("failed to get offering: "),
			mockError:      nil,
			expectedResult: nil,
			mockResult:     mockOffering,
			mockResponse:   &core.DetailedResponse{StatusCode: 400},
			installKind:    NewInstallKindTerraform(),
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Clear previous expectations
			suite.mockService.ExpectedCalls = nil

			// Fix: Use safer parameter checking that won't panic on nil values
			suite.mockService.On("GetOffering", mock.MatchedBy(func(opts *catalogmanagementv1.GetOfferingOptions) bool {
				if opts == nil {
					return false
				}

				// Only check required fields that we know should be present
				catalogIDMatch := opts.CatalogIdentifier != nil && *opts.CatalogIdentifier == catalogID
				offeringIDMatch := opts.OfferingID != nil && *opts.OfferingID == offeringID

				// We can also check these fields if they are important
				// But we use safer nil checks to avoid panics
				return catalogIDMatch && offeringIDMatch
			})).Return(tc.mockResult, tc.mockResponse, tc.mockError)

			result, _, err := suite.infoSvc.GetOffering(catalogID, offeringID)
			if tc.expectedError != nil {
				assert.Error(suite.T(), err)
				assert.Nil(suite.T(), result)
			} else {
				assert.NoError(suite.T(), err)
				assert.Equal(suite.T(), tc.expectedResult, result)
			}
		})
	}
}

func TestGetOfferingInputs(t *testing.T) {
	tests := []struct {
		name         string
		versionID    string
		offering     *catalogmanagementv1.Offering
		expectInputs bool
		expectedLog  string
	}{
		{
			name:      "Version found - returns inputs",
			versionID: "v1",
			offering: &catalogmanagementv1.Offering{
				ID: core.StringPtr("off1"),
				Kinds: []catalogmanagementv1.Kind{
					{
						Versions: []catalogmanagementv1.Version{
							{
								ID: core.StringPtr("v1"),
								Configuration: []catalogmanagementv1.Configuration{
									{
										Key:          core.StringPtr("input1"),
										Type:         core.StringPtr("string"),
										DefaultValue: "default",
										Description:  core.StringPtr("An input"),
										Required:     core.BoolPtr(true),
									},
								},
							},
						},
					},
				},
			},
			expectInputs: true,
		},
		{
			name:      "Version not found - returns nil and logs message",
			versionID: "not-found",
			offering: &catalogmanagementv1.Offering{
				ID: core.StringPtr("off2"),
				Kinds: []catalogmanagementv1.Kind{
					{
						Versions: []catalogmanagementv1.Version{
							{
								ID:            core.StringPtr("v1"),
								Configuration: []catalogmanagementv1.Configuration{},
							},
						},
					},
				},
			},
			expectInputs: false,
			expectedLog:  "Error, version not found for offering: off2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := common.NewTestLogger("test")
			service := &CloudInfoService{
				Logger: logger,
			}
			inputs := service.GetOfferingInputs(tt.offering, tt.versionID, *tt.offering.ID)

			if tt.expectInputs {
				assert.NotNil(t, inputs)
				assert.Len(t, inputs, 1)
				assert.Equal(t, "input1", inputs[0].Key)
			} else {
				assert.Nil(t, inputs)
			}
		})
	}
}

// TestFlattenDependencies tests the flattenDependencies function that recursively collects all dependencies
func (suite *CatalogServiceTestSuite) TestFlattenDependencies() {
	// Test cases
	testCases := []struct {
		name           string
		addonConfig    *AddonConfig
		expectedLength int
	}{
		{
			name: "No dependencies",
			addonConfig: &AddonConfig{
				OfferingName:   "main-offering",
				VersionLocator: "locator-1",
				Dependencies:   []AddonConfig{},
			},
			expectedLength: 0,
		},
		{
			name: "Direct dependencies only",
			addonConfig: &AddonConfig{
				OfferingName:   "main-offering",
				VersionLocator: "locator-1",
				Dependencies: []AddonConfig{
					{
						OfferingName:   "dep-1",
						VersionLocator: "dep-locator-1",
						Dependencies:   []AddonConfig{},
					},
					{
						OfferingName:   "dep-2",
						VersionLocator: "dep-locator-2",
						Dependencies:   []AddonConfig{},
					},
				},
			},
			expectedLength: 2,
		},
		{
			name: "Nested dependencies",
			addonConfig: &AddonConfig{
				OfferingName:   "main-offering",
				VersionLocator: "locator-1",
				Dependencies: []AddonConfig{
					{
						OfferingName:   "dep-1",
						VersionLocator: "dep-locator-1",
						Dependencies: []AddonConfig{
							{
								OfferingName:   "nested-dep-1",
								VersionLocator: "nested-locator-1",
								Dependencies:   []AddonConfig{},
							},
						},
					},
					{
						OfferingName:   "dep-2",
						VersionLocator: "dep-locator-2",
						Dependencies: []AddonConfig{
							{
								OfferingName:   "nested-dep-2",
								VersionLocator: "nested-locator-2",
								Dependencies:   []AddonConfig{},
							},
						},
					},
				},
			},
			expectedLength: 4,
		},
		{
			name: "Duplicated dependencies are only included once",
			addonConfig: &AddonConfig{
				OfferingName:   "main-offering",
				VersionLocator: "locator-1",
				Dependencies: []AddonConfig{
					{
						OfferingName:   "dep-1",
						VersionLocator: "dep-locator-1",
						Dependencies: []AddonConfig{
							{
								OfferingName:   "shared-dep",
								VersionLocator: "shared-locator",
								Dependencies:   []AddonConfig{},
							},
						},
					},
					{
						OfferingName:   "dep-2",
						VersionLocator: "dep-locator-2",
						Dependencies: []AddonConfig{
							{
								OfferingName:   "shared-dep",
								VersionLocator: "shared-locator", // Same version locator
								Dependencies:   []AddonConfig{},
							},
						},
					},
				},
			},
			expectedLength: 3, // Only 3 because the shared dependency is only counted once
		},
	}

	// Run the test cases
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			result := flattenDependencies(tc.addonConfig)
			assert.Equal(suite.T(), tc.expectedLength, len(result), "Expected %d dependencies, got %d", tc.expectedLength, len(result))

			// Check that there are no duplicate version locators
			locatorMap := make(map[string]bool)
			for _, dep := range result {
				assert.False(suite.T(), locatorMap[dep.VersionLocator], "Duplicate version locator found: %s", dep.VersionLocator)
				locatorMap[dep.VersionLocator] = true
			}
		})
	}
}

func TestCatalogServiceTestSuite(t *testing.T) {
	suite.Run(t, new(CatalogServiceTestSuite))
}

// TestUpdateConfigInfoFromResponse tests the updateConfigInfoFromResponse function
func (suite *CatalogServiceTestSuite) TestUpdateConfigInfoFromResponse() {
	testCases := []struct {
		name                string
		addonConfig         *AddonConfig
		dependencies        []AddonConfig
		response            *DeployedAddonsDetails
		expectedMainConfig  string
		expectedContainerId string
		expectedDepIds      map[string]string
	}{
		{
			name: "Main addon and dependencies",
			addonConfig: &AddonConfig{
				ConfigName: "main-addon",
			},
			dependencies: []AddonConfig{
				{
					ConfigName: "dep-1",
				},
				{
					ConfigName: "dep-2",
				},
			},
			response: &DeployedAddonsDetails{
				ProjectID: "project-123",
				Configs: []struct {
					Name     string `json:"name"`
					ConfigID string `json:"config_id"`
				}{
					{
						Name:     "main-addon",
						ConfigID: "config-main",
					},
					{
						Name:     "main-addon Container",
						ConfigID: "container-main",
					},
					{
						Name:     "dep-1",
						ConfigID: "config-dep-1",
					},
					{
						Name:     "dep-2",
						ConfigID: "config-dep-2",
					},
					{
						Name:     "dep-2 Container",
						ConfigID: "container-dep-2",
					},
				},
			},
			expectedMainConfig:  "config-main",
			expectedContainerId: "container-main",
			expectedDepIds: map[string]string{
				"dep-1": "config-dep-1",
				"dep-2": "config-dep-2",
			},
		},
		{
			name: "Only main addon without container",
			addonConfig: &AddonConfig{
				ConfigName: "main-addon",
			},
			dependencies: []AddonConfig{},
			response: &DeployedAddonsDetails{
				ProjectID: "project-123",
				Configs: []struct {
					Name     string `json:"name"`
					ConfigID string `json:"config_id"`
				}{
					{
						Name:     "main-addon",
						ConfigID: "config-main",
					},
				},
			},
			expectedMainConfig:  "config-main",
			expectedContainerId: "",
			expectedDepIds:      map[string]string{},
		},
		{
			name: "Unmatched config names",
			addonConfig: &AddonConfig{
				ConfigName: "main-addon",
			},
			dependencies: []AddonConfig{
				{
					ConfigName: "dep-1",
				},
			},
			response: &DeployedAddonsDetails{
				ProjectID: "project-123",
				Configs: []struct {
					Name     string `json:"name"`
					ConfigID string `json:"config_id"`
				}{
					{
						Name:     "other-addon",
						ConfigID: "config-other",
					},
					{
						Name:     "dep-999",
						ConfigID: "config-dep-999",
					},
				},
			},
			expectedMainConfig:  "",
			expectedContainerId: "",
			expectedDepIds:      map[string]string{},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Create copies of the configs for modification during test
			addonConfig := *tc.addonConfig
			// Set up dependencies in the addonConfig structure
			addonConfig.Dependencies = make([]AddonConfig, len(tc.dependencies))
			copy(addonConfig.Dependencies, tc.dependencies)

			// Call the function to test
			updateConfigInfoFromResponse(&addonConfig, tc.response)

			// Check the main addon's config
			assert.Equal(suite.T(), tc.expectedMainConfig, addonConfig.ConfigID)
			assert.Equal(suite.T(), tc.expectedContainerId, addonConfig.ContainerConfigID)

			// If container ID is expected to be set, check that container name is set correctly
			if tc.expectedContainerId != "" {
				assert.Equal(suite.T(), addonConfig.ConfigName+" Container", addonConfig.ContainerConfigName)
			} else {
				assert.Empty(suite.T(), addonConfig.ContainerConfigName)
			}

			// Check the dependencies
			for i, dep := range addonConfig.Dependencies {
				expectedID, exists := tc.expectedDepIds[dep.ConfigName]
				if exists {
					assert.Equal(suite.T(), expectedID, addonConfig.Dependencies[i].ConfigID, "Dependency %s has incorrect ConfigID", dep.ConfigName)
				} else {
					assert.Empty(suite.T(), addonConfig.Dependencies[i].ConfigID, "Dependency %s should have empty ConfigID", dep.ConfigName)
				}
			}
		})
	}
}

func (suite *CatalogServiceTestSuite) TestGetOfferingVersionLocatorByConstraint_TableDriven() {
	catalogID := "test-catalog-id"
	offeringID := "test-offering-id"

	// Expanded mock offering with multiple versions and flavors
	mockOffering := &catalogmanagementv1.Offering{
		ID:   core.StringPtr(offeringID),
		Name: core.StringPtr("mock-offering"),
		Kinds: []catalogmanagementv1.Kind{
			{
				InstallKind: core.StringPtr("terraform"),
				Versions: []catalogmanagementv1.Version{
					{
						Version: core.StringPtr("v8.20.2"),
						Flavor: &catalogmanagementv1.Flavor{
							Name:  core.StringPtr("instance"),
							Label: core.StringPtr("Single instance"),
							Index: core.Int64Ptr(0),
						},
						OfferingID:     core.StringPtr("test-offering-id"),
						CatalogID:      core.StringPtr("test-catalog-id"),
						VersionLocator: core.StringPtr("locator-v8.20.2"),
					},
					{
						Version: core.StringPtr("8.18.0"),
						Flavor: &catalogmanagementv1.Flavor{
							Name:  core.StringPtr("instance"),
							Label: core.StringPtr("Single instance"),
							Index: core.Int64Ptr(0),
						},
						OfferingID:     core.StringPtr("test-offering-id"),
						CatalogID:      core.StringPtr("test-catalog-id"),
						VersionLocator: core.StringPtr("locator-8.18.0"),
					},
					{
						Version: core.StringPtr("v7.50.1"),
						Flavor: &catalogmanagementv1.Flavor{
							Name:  core.StringPtr("instance"),
							Label: core.StringPtr("Single instance"),
							Index: core.Int64Ptr(0),
						},
						OfferingID:     core.StringPtr("test-offering-id"),
						CatalogID:      core.StringPtr("test-catalog-id"),
						VersionLocator: core.StringPtr("locator-v7.50.1"),
					},
					{
						Version: core.StringPtr("v8.18.0"),
						Flavor: &catalogmanagementv1.Flavor{
							Name:  core.StringPtr("multi-instance"),
							Label: core.StringPtr("Multi instance"),
							Index: core.Int64Ptr(1),
						},
						OfferingID:     core.StringPtr("test-offering-id"),
						CatalogID:      core.StringPtr("test-catalog-id"),
						VersionLocator: core.StringPtr("locator-v8.18.0-multi"),
					},
				},
			},
		},
	}

	mockResponse := &core.DetailedResponse{StatusCode: 200}
	var mockError error

	// Setup the mock once
	suite.mockService.ExpectedCalls = nil
	suite.mockService.On("GetOffering", mock.MatchedBy(func(opts *catalogmanagementv1.GetOfferingOptions) bool {
		if opts == nil {
			return false
		}
		return opts.CatalogIdentifier != nil && *opts.CatalogIdentifier == catalogID &&
			opts.OfferingID != nil && *opts.OfferingID == offeringID
	})).Return(mockOffering, mockResponse, mockError)

	// Test cases table
	testCases := []struct {
		name            string
		requestedVer    string
		requestedFlavor string
		expectedVer     string
		expectedLocator string
		expectErr       bool
	}{
		{
			name:            "Exact version match",
			requestedVer:    "v8.20.2",
			requestedFlavor: "instance",
			expectedVer:     "v8.20.2",
			expectedLocator: "locator-v8.20.2",
			expectErr:       false,
		},
		{
			name:            "Caret version match ^v8.18.0 (allow patch updates)",
			requestedVer:    "^v8.18.0",
			requestedFlavor: "instance",
			expectedVer:     "v8.20.2", // latest >= 8.18.0 and <9.0.0
			expectedLocator: "locator-v8.20.2",
			expectErr:       false,
		},
		{
			name:            "Tilde version match ~v8.18.0 (allow patch updates only)",
			requestedVer:    "~v8.18.0",
			requestedFlavor: "instance",
			expectedVer:     "8.18.0",
			expectedLocator: "locator-8.18.0",
			expectErr:       false,
		},
		{
			name:            "Range version match >=v8.15.0,<=v8.22.0 (allow patch updates only)",
			requestedVer:    ">=8.15.0,<=8.22.0",
			requestedFlavor: "instance",
			expectedVer:     "v8.20.2",
			expectedLocator: "locator-v8.20.2",
			expectErr:       false,
		},
		{
			name:            "Range version match <=v7.90.0,>=1.22.0 (allow patch updates only)",
			requestedVer:    "<=7.90.0,>=1.22.0",
			requestedFlavor: "instance",
			expectedVer:     "v7.50.1",
			expectedLocator: "locator-v7.50.1",
			expectErr:       false,
		},
		{
			name:            "Flavor multi instance",
			requestedVer:    "v8.18.0",
			requestedFlavor: "multi-instance",
			expectedVer:     "v8.18.0",
			expectedLocator: "locator-v8.18.0-multi",
			expectErr:       false,
		},
		{
			name:            "No matching version",
			requestedVer:    "v9.0.0",
			requestedFlavor: "instance",
			expectErr:       true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			bestVersion, versionLocator, err := suite.infoSvc.GetOfferingVersionLocatorByConstraint(
				catalogID, offeringID, tc.requestedVer, tc.requestedFlavor,
			)

			if tc.expectErr {
				suite.Error(err)
			} else {
				suite.NoError(err)
				suite.Equal(tc.expectedVer, bestVersion)
				suite.Equal(tc.expectedLocator, versionLocator)
			}
		})
	}

	suite.mockService.AssertExpectations(suite.T())
}

// TestProcessComponentReferencesUserOverrides tests that user-defined dependency settings are preserved
// while metadata fields are populated from component references
func (suite *CatalogServiceTestSuite) TestProcessComponentReferencesUserOverrides() {
	// Create a mock service that returns component references
	mockReferences := &OfferingReferenceResponse{
		Optional: OptionalReferences{
			OfferingReferences: []OfferingReferenceItem{
				{
					Name: "database",
					OfferingReference: OfferingReferenceDetail{
						Name:           "database",
						VersionLocator: "catalog123.version456",
						ID:             "offering789",
						Version:        "1.0.0",
						OnByDefault:    true, // Component reference says it should be on by default
						Flavor: Flavor{
							Name: "standard",
						},
						Label: "Database Service",
					},
				},
				{
					Name: "monitoring",
					OfferingReference: OfferingReferenceDetail{
						Name:           "monitoring",
						VersionLocator: "catalog123.version999",
						ID:             "offering888",
						Version:        "2.0.0",
						OnByDefault:    true, // Component reference says it should be on by default
						Flavor: Flavor{
							Name: "basic",
						},
						Label: "Monitoring Service",
					},
				},
			},
		},
	}

	// Create a mock component reference getter
	mockGetter := &MockComponentReferenceGetter{
		MockGetComponentReferences: func(versionLocator string) (*OfferingReferenceResponse, error) {
			if versionLocator == "main-locator" {
				return mockReferences, nil
			}
			// For dependency version locators, return empty references (no nested dependencies)
			if versionLocator == "catalog123.version456" || versionLocator == "catalog123.version999" {
				return &OfferingReferenceResponse{
					Required: RequiredReferences{
						OfferingReferences: []OfferingReferenceItem{},
					},
					Optional: OptionalReferences{
						OfferingReferences: []OfferingReferenceItem{},
					},
				}, nil
			}
			return nil, fmt.Errorf("unexpected version locator: %s", versionLocator)
		},
	}

	// Create addon config with user-defined dependencies
	addonConfig := &AddonConfig{
		VersionLocator: "main-locator",
		Dependencies: []AddonConfig{
			{
				OfferingName: "database",
				Enabled:      core.BoolPtr(false), // User explicitly disabled this
				OnByDefault:  core.BoolPtr(false), // User explicitly set to false
				// VersionLocator is empty - framework should populate metadata
			}, {
				OfferingName: "monitoring",
				Enabled:      core.BoolPtr(true), // User explicitly enabled this
				// OnByDefault is nil - framework can set this
				// VersionLocator is empty - framework should populate metadata
			},
		},
	}

	processedLocators := make(map[string]bool)

	// Call processComponentReferencesWithGetter with the mock
	disabledOfferings := make(map[string]bool)
	err := suite.infoSvc.processComponentReferencesWithGetter(addonConfig, processedLocators, disabledOfferings, mockGetter)
	assert.NoError(suite.T(), err)

	// Verify user settings are preserved
	assert.NotNil(suite.T(), addonConfig.Dependencies[0].Enabled, "Enabled should not be nil")
	assert.False(suite.T(), *addonConfig.Dependencies[0].Enabled, "User-defined Enabled: false should be preserved")

	assert.NotNil(suite.T(), addonConfig.Dependencies[0].OnByDefault, "OnByDefault should not be nil")
	assert.False(suite.T(), *addonConfig.Dependencies[0].OnByDefault, "User-defined OnByDefault: false should be preserved")

	assert.NotNil(suite.T(), addonConfig.Dependencies[1].Enabled, "Enabled should not be nil")
	assert.True(suite.T(), *addonConfig.Dependencies[1].Enabled, "User-defined Enabled: true should be preserved")

	// Verify framework populated OnByDefault for the dependency where user didn't set it
	assert.NotNil(suite.T(), addonConfig.Dependencies[1].OnByDefault, "OnByDefault should be populated by framework")
	assert.True(suite.T(), *addonConfig.Dependencies[1].OnByDefault, "Framework should set OnByDefault from component reference")

	// Verify metadata fields are populated from component references
	assert.Equal(suite.T(), "catalog123.version456", addonConfig.Dependencies[0].VersionLocator, "VersionLocator should be populated")
	assert.Equal(suite.T(), "offering789", addonConfig.Dependencies[0].OfferingID, "OfferingID should be populated")
	assert.Equal(suite.T(), "1.0.0", addonConfig.Dependencies[0].ResolvedVersion, "ResolvedVersion should be populated")
	assert.Equal(suite.T(), "Database Service", addonConfig.Dependencies[0].OfferingLabel, "OfferingLabel should be populated")

	assert.Equal(suite.T(), "catalog123.version999", addonConfig.Dependencies[1].VersionLocator, "VersionLocator should be populated")
	assert.Equal(suite.T(), "offering888", addonConfig.Dependencies[1].OfferingID, "OfferingID should be populated")
	assert.Equal(suite.T(), "2.0.0", addonConfig.Dependencies[1].ResolvedVersion, "ResolvedVersion should be populated")
	assert.Equal(suite.T(), "Monitoring Service", addonConfig.Dependencies[1].OfferingLabel, "OfferingLabel should be populated")
}
