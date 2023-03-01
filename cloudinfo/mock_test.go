package cloudinfo

import (
	"errors"
	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/platform-services-go-sdk/contextbasedrestrictionsv1"
	"github.com/IBM/platform-services-go-sdk/iamidentityv1"
	"github.com/IBM/platform-services-go-sdk/resourcecontrollerv2"
	"github.com/IBM/vpc-go-sdk/vpcv1"
	"github.com/stretchr/testify/mock"
	"log"
	"strconv"
	"strings"
)

// VPC SERVICE INTERFACE MOCK
type vpcServiceMock struct {
	mock.Mock
	mockRegionUrl string
}

func (mock *vpcServiceMock) ListRegions(options *vpcv1.ListRegionsOptions) (*vpcv1.RegionCollection, *core.DetailedResponse, error) {
	var regions []vpcv1.Region
	status := regionStatusAvailable
	var newRegion vpcv1.Region

	// set up some fake regions
	regionNames := []string{"regavail-1-5", "regavail-2-10", "regavail-3-1", "regavail-4-30"}
	for _, regName := range regionNames {
		newName := regName // need reassignment here to get new address for using below, don't use address of regName
		newRegion = vpcv1.Region{Name: &newName, Endpoint: &newName, Href: &newName, Status: &status}
		regions = append(regions, newRegion)
	}

	regionCol := vpcv1.RegionCollection{
		Regions: regions,
	}

	return &regionCol, nil, nil
}

func (mock *vpcServiceMock) GetRegion(options *vpcv1.GetRegionOptions) (*vpcv1.Region, *core.DetailedResponse, error) {
	status := regionStatusAvailable
	// keep it simple for the mock, the url is just the name
	region := vpcv1.Region{
		Name:     options.Name,
		Endpoint: options.Name,
		Href:     options.Name,
		Status:   &status,
	}
	return &region, nil, nil
}

func (mock *vpcServiceMock) NewGetRegionOptions(name string) *vpcv1.GetRegionOptions {
	options := vpcv1.GetRegionOptions{
		Name: &name,
	}
	return &options
}

func (mock *vpcServiceMock) ListVpcs(options *vpcv1.ListVpcsOptions) (*vpcv1.VPCCollection, *core.DetailedResponse, error) {
	// the "count" of VPCs for a region in the mock will just be the suffix of the region url (which is a simple name)
	urlParts := strings.Split(mock.mockRegionUrl, "/") // we only want first part of url pathing
	nameParts := strings.Split(urlParts[0], "-")
	count, _ := strconv.ParseInt(nameParts[len(nameParts)-1], 0, 64)
	log.Println("Count of", mock.mockRegionUrl, " = ", count)
	vpcCol := vpcv1.VPCCollection{
		TotalCount: &count,
	}
	return &vpcCol, nil, nil
}

func (mock *vpcServiceMock) SetServiceURL(url string) error {
	mock.mockRegionUrl = url
	return nil
}

// IAM SERVICE MOCK
type iamIdentityServiceMock struct {
	mock.Mock
}

func (mock *iamIdentityServiceMock) GetAPIKeysDetails(options *iamidentityv1.GetAPIKeysDetailsOptions) (*iamidentityv1.APIKey, *core.DetailedResponse, error) {
	id := "MOCK_ID"
	name := "MOCK_NAME"
	acctId := "MOCK_ACCOUNT_ID"

	// if the api key in option is ERROR then pass error back
	if *options.IamAPIKey == "ERROR" {
		return nil, nil, errors.New("mock API key is bad")
	}

	return &iamidentityv1.APIKey{
		ID:        &id,
		Name:      &name,
		AccountID: &acctId,
	}, nil, nil
}

type resourceControllerServiceMock struct {
	mock.Mock
	mockResourceList *resourcecontrollerv2.ResourceInstancesList
}

func (mock *resourceControllerServiceMock) NewListResourceInstancesOptions() *resourcecontrollerv2.ListResourceInstancesOptions {
	return &resourcecontrollerv2.ListResourceInstancesOptions{}
}

func (mock *resourceControllerServiceMock) ListResourceInstances(options *resourcecontrollerv2.ListResourceInstancesOptions) (*resourcecontrollerv2.ResourceInstancesList, *core.DetailedResponse, error) {
	var retList *resourcecontrollerv2.ResourceInstancesList
	var mockCount int64 = 0

	if options.Name != nil && *options.Name == "ERROR" {
		return nil, nil, errors.New("mock Resource is error")
	}

	if mock.mockResourceList == nil {
		retList = &resourcecontrollerv2.ResourceInstancesList{
			RowsCount: &mockCount,
		}
	} else {
		retList = mock.mockResourceList
	}

	return retList, nil, nil
}

// Mock CBR
type cbrServiceMock struct {
	mock.Mock
	rule             *contextbasedrestrictionsv1.Rule
	zone             *contextbasedrestrictionsv1.Zone
	detailedResponse *core.DetailedResponse
	err              error
}

func (mock *cbrServiceMock) GetZone(options *contextbasedrestrictionsv1.GetZoneOptions) (*contextbasedrestrictionsv1.Zone, *core.DetailedResponse, error) {
	return mock.zone, mock.detailedResponse, mock.err
}

func (mock *cbrServiceMock) GetRule(options *contextbasedrestrictionsv1.GetRuleOptions) (*contextbasedrestrictionsv1.Rule, *core.DetailedResponse, error) {
	return mock.rule, mock.detailedResponse, mock.err
}
