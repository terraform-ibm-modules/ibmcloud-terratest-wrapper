package cloudinfo

import (
	"errors"
	"github.com/IBM-Cloud/bluemix-go/api/container/containerv1"
	"github.com/IBM-Cloud/bluemix-go/api/container/containerv2"
	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/platform-services-go-sdk/contextbasedrestrictionsv1"
	"github.com/IBM/platform-services-go-sdk/iamidentityv1"
	"github.com/IBM/platform-services-go-sdk/iampolicymanagementv1"
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

// IAM POLICY SERVICE MOCK
type iamPolicyServiceMock struct {
	mock.Mock
}

func (mock *iamPolicyServiceMock) DeletePolicy(deletePolicyOptions *iampolicymanagementv1.DeletePolicyOptions) (*core.DetailedResponse, error) {
	args := mock.Called(deletePolicyOptions)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*core.DetailedResponse), args.Error(1)
}

// RESOURCE CONTROLLER SERVICE MOCK
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

func (mock *cbrServiceMock) ReplaceRule(options *contextbasedrestrictionsv1.ReplaceRuleOptions) (*contextbasedrestrictionsv1.Rule, *core.DetailedResponse, error) {
	return mock.rule, mock.detailedResponse, mock.err
}

func (mock *cbrServiceMock) GetZone(options *contextbasedrestrictionsv1.GetZoneOptions) (*contextbasedrestrictionsv1.Zone, *core.DetailedResponse, error) {
	return mock.zone, mock.detailedResponse, mock.err
}

func (mock *cbrServiceMock) GetRule(options *contextbasedrestrictionsv1.GetRuleOptions) (*contextbasedrestrictionsv1.Rule, *core.DetailedResponse, error) {
	return mock.rule, mock.detailedResponse, mock.err
}

// Mock Container Client
type containerClientMock struct {
	mock.Mock
}

func (mock *containerClientMock) Clusters() containerv2.Clusters {
	args := mock.Called()
	return args.Get(0).(containerv2.Clusters) // Cast to the expected return type
}

type ClustersMock struct {
	mock.Mock
}

func (m *ClustersMock) Create(params containerv2.ClusterCreateRequest, target containerv2.ClusterTargetHeader) (containerv2.ClusterCreateResponse, error) {
	args := m.Called(params, target)
	return args.Get(0).(containerv2.ClusterCreateResponse), args.Error(1)
}

func (m *ClustersMock) List(target containerv2.ClusterTargetHeader) ([]containerv2.ClusterInfo, error) {
	args := m.Called(target)
	return args.Get(0).([]containerv2.ClusterInfo), args.Error(1)
}

func (m *ClustersMock) Delete(name string, target containerv2.ClusterTargetHeader, deleteDependencies ...bool) error {
	args := m.Called(name, target, deleteDependencies)
	return args.Error(0)
}

func (m *ClustersMock) GetCluster(name string, target containerv2.ClusterTargetHeader) (*containerv2.ClusterInfo, error) {
	args := m.Called(name, target)
	return args.Get(0).(*containerv2.ClusterInfo), args.Error(1)
}

func (m *ClustersMock) GetClusterConfigDetail(name, homeDir string, admin bool, target containerv2.ClusterTargetHeader, endpointType string) (containerv1.ClusterKeyInfo, error) {
	args := m.Called(name, homeDir, admin, target, endpointType)
	return args.Get(0).(containerv1.ClusterKeyInfo), args.Error(1)
}

func (m *ClustersMock) StoreConfigDetail(name, baseDir string, admin bool, createCalicoConfig bool, target containerv2.ClusterTargetHeader, endpointType string) (string, containerv1.ClusterKeyInfo, error) {
	args := m.Called(name, baseDir, admin, createCalicoConfig, target, endpointType)
	return args.String(0), args.Get(1).(containerv1.ClusterKeyInfo), args.Error(2)
}

func (m *ClustersMock) EnableImageSecurityEnforcement(name string, target containerv2.ClusterTargetHeader) error {
	args := m.Called(name, target)
	return args.Error(0)
}

func (m *ClustersMock) DisableImageSecurityEnforcement(name string, target containerv2.ClusterTargetHeader) error {
	args := m.Called(name, target)
	return args.Error(0)
}
