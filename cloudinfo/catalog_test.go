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
)

type CatalogServiceTestSuite struct {
	suite.Suite
	mockService *catalogServiceMock
	infoSvc     *CloudInfoService
}

func (suite *CatalogServiceTestSuite) SetupTest() {
	suite.mockService = new(catalogServiceMock)
	suite.infoSvc = &CloudInfoService{
		catalogService: suite.mockService,
		authenticator: &core.IamAuthenticator{
			ApiKey: "mockApiKey",
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
			expectedError:  nil,
			mockError:      nil,
			expectedResult: &catalogmanagementv1.Version{},
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

func TestCatalogServiceTestSuite(t *testing.T) {
	suite.Run(t, new(CatalogServiceTestSuite))
}
