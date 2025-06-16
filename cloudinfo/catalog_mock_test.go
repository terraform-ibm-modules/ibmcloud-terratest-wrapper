package cloudinfo

import (
	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/platform-services-go-sdk/catalogmanagementv1"
	"github.com/stretchr/testify/mock"
)

type catalogServiceMock struct {
	mock.Mock
}

// GetVersion(getVersionOptions *catalogmanagementv1.GetVersionOptions) (result *catalogmanagementv1.Offering, response *core.DetailedResponse, err error)
func (mock *catalogServiceMock) GetVersion(getVersionOptions *catalogmanagementv1.GetVersionOptions) (*catalogmanagementv1.Offering, *core.DetailedResponse, error) {
	args := mock.Called(getVersionOptions)

	var offering *catalogmanagementv1.Offering
	if args.Get(0) != nil {
		offering = args.Get(0).(*catalogmanagementv1.Offering)
	}

	var response *core.DetailedResponse
	if args.Get(1) != nil {
		response = args.Get(1).(*core.DetailedResponse)
	}

	return offering, response, args.Error(2)
}

// CreateCatalog(createCatalogOptions *catalogmanagementv1.CreateCatalogOptions) (result *catalogmanagementv1.Catalog, response *core.DetailedResponse, err error)
func (mock *catalogServiceMock) CreateCatalog(createCatalogOptions *catalogmanagementv1.CreateCatalogOptions) (*catalogmanagementv1.Catalog, *core.DetailedResponse, error) {
	args := mock.Called(createCatalogOptions)

	var catalog *catalogmanagementv1.Catalog
	if args.Get(0) != nil {
		catalog = args.Get(0).(*catalogmanagementv1.Catalog)
	}

	var response *core.DetailedResponse
	if args.Get(1) != nil {
		response = args.Get(1).(*core.DetailedResponse)
	}

	return catalog, response, args.Error(2)
}

// DeleteCatalog(deleteCatalogOptions *catalogmanagementv1.DeleteCatalogOptions) (response *core.DetailedResponse, err error)
func (mock *catalogServiceMock) DeleteCatalog(deleteCatalogOptions *catalogmanagementv1.DeleteCatalogOptions) (*core.DetailedResponse, error) {
	args := mock.Called(deleteCatalogOptions)

	var response *core.DetailedResponse
	if args.Get(0) != nil {
		response = args.Get(0).(*core.DetailedResponse)
	}

	return response, args.Error(1)
}

// ImportOffering(importOfferingOptions *catalogmanagementv1.ImportOfferingOptions) (result *catalogmanagementv1.Offering, response *core.DetailedResponse, err error)
func (mock *catalogServiceMock) ImportOffering(importOfferingOptions *catalogmanagementv1.ImportOfferingOptions) (*catalogmanagementv1.Offering, *core.DetailedResponse, error) {
	args := mock.Called(importOfferingOptions)

	var offering *catalogmanagementv1.Offering
	if args.Get(0) != nil {
		offering = args.Get(0).(*catalogmanagementv1.Offering)
	}

	var response *core.DetailedResponse
	if args.Get(1) != nil {
		response = args.Get(1).(*core.DetailedResponse)
	}

	return offering, response, args.Error(2)
}

func (mock *catalogServiceMock) GetOffering(getOfferingOptions *catalogmanagementv1.GetOfferingOptions) (*catalogmanagementv1.Offering, *core.DetailedResponse, error) {
	args := mock.Called(getOfferingOptions)

	var offering *catalogmanagementv1.Offering
	if args.Get(0) != nil {
		offering = args.Get(0).(*catalogmanagementv1.Offering)
	}

	var response *core.DetailedResponse
	if args.Get(1) != nil {
		response = args.Get(1).(*core.DetailedResponse)
	}

	return offering, response, args.Error(2)
}
