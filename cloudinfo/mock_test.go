package cloudinfo

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/IBM-Cloud/bluemix-go/api/container/containerv1"
	"github.com/IBM-Cloud/bluemix-go/api/container/containerv2"
	"github.com/IBM/cloud-databases-go-sdk/clouddatabasesv5"
	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/platform-services-go-sdk/contextbasedrestrictionsv1"
	"github.com/IBM/platform-services-go-sdk/iamidentityv1"
	"github.com/IBM/platform-services-go-sdk/iampolicymanagementv1"
	"github.com/IBM/platform-services-go-sdk/resourcecontrollerv2"
	"github.com/IBM/platform-services-go-sdk/resourcemanagerv2"
	"github.com/IBM/vpc-go-sdk/vpcv1"
	"github.com/stretchr/testify/mock"
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
	mockResourceList    *resourcecontrollerv2.ResourceInstancesList
	mockReclamationList *resourcecontrollerv2.ReclamationsList
	mockReclamation     *resourcecontrollerv2.Reclamation
}

func (mock *resourceControllerServiceMock) NewListResourceInstancesOptions() *resourcecontrollerv2.ListResourceInstancesOptions {
	return &resourcecontrollerv2.ListResourceInstancesOptions{}
}

func (mock *resourceControllerServiceMock) NewListReclamationsOptions() *resourcecontrollerv2.ListReclamationsOptions {

	return &resourcecontrollerv2.ListReclamationsOptions{}
}

func (mock *resourceControllerServiceMock) NewRunReclamationActionOptions(id string, action string) *resourcecontrollerv2.RunReclamationActionOptions {

	return &resourcecontrollerv2.RunReclamationActionOptions{ID: core.StringPtr(id), ActionName: core.StringPtr(action)}
}

func (mock *resourceControllerServiceMock) ListResourceInstances(options *resourcecontrollerv2.ListResourceInstancesOptions) (*resourcecontrollerv2.ResourceInstancesList, *core.DetailedResponse, error) {
	var retList *resourcecontrollerv2.ResourceInstancesList
	mockCount := int64(0)

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

func (mock *resourceControllerServiceMock) ListReclamations(options *resourcecontrollerv2.ListReclamationsOptions) (*resourcecontrollerv2.ReclamationsList, *core.DetailedResponse, error) {
	recList := &resourcecontrollerv2.ReclamationsList{}
	mockID := "mock-reclamation-id"
	mockReclamation := resourcecontrollerv2.Reclamation{ID: &mockID}
	mockReclamationList := []resourcecontrollerv2.Reclamation{mockReclamation}

	if options.ResourceInstanceID != nil && *options.ResourceInstanceID == "ERROR" {
		return nil, nil, errors.New("mock Resource is error")
	}

	if mock.mockReclamationList == nil {
		recList = &resourcecontrollerv2.ReclamationsList{
			Resources: mockReclamationList,
		}
	} else {
		recList = mock.mockReclamationList
	}

	return recList, nil, nil
}

func (mock *resourceControllerServiceMock) RunReclamationAction(options *resourcecontrollerv2.RunReclamationActionOptions) (*resourcecontrollerv2.Reclamation, *core.DetailedResponse, error) {
	var reclamation *resourcecontrollerv2.Reclamation
	mockID := "mock-reclamation-id"
	mockReclamation := resourcecontrollerv2.Reclamation{ID: &mockID}

	if mock.mockReclamation == nil {
		reclamation = &mockReclamation
	} else {
		reclamation = mock.mockReclamation
	}

	return reclamation, nil, nil
}

// Resource Manager mock
type resourceManagerServiceMock struct {
	mockResourceGroupList             *resourcemanagerv2.ResourceGroupList
	resourceGroups                    map[string]string // map of resource group names to IDs
	mockResCreateResourceGroup        *resourcemanagerv2.ResCreateResourceGroup
	mockNewDeleteResourceGroupOptions *resourcemanagerv2.DeleteResourceGroupOptions
	mockDeleteResourceGroup           *core.DetailedResponse
}

func (s *resourceManagerServiceMock) NewListResourceGroupsOptions() *resourcemanagerv2.ListResourceGroupsOptions {
	return &resourcemanagerv2.ListResourceGroupsOptions{}
}

func (s *resourceManagerServiceMock) ListResourceGroups(*resourcemanagerv2.ListResourceGroupsOptions) (*resourcemanagerv2.ResourceGroupList, *core.DetailedResponse, error) {
	return s.mockResourceGroupList, nil, nil
}

func (s *resourceManagerServiceMock) NewCreateResourceGroupOptions() *resourcemanagerv2.CreateResourceGroupOptions {
	return &resourcemanagerv2.CreateResourceGroupOptions{
		Name: core.StringPtr(""),
	}
}

func (s *resourceManagerServiceMock) CreateResourceGroup(*resourcemanagerv2.CreateResourceGroupOptions) (*resourcemanagerv2.ResCreateResourceGroup, *core.DetailedResponse, error) {
	resp := &core.DetailedResponse{
		StatusCode: 200,
		Headers:    map[string][]string{"Content-Type": {"application/json"}},
	}

	return s.mockResCreateResourceGroup, resp, nil
}

func (s *resourceManagerServiceMock) NewDeleteResourceGroupOptions(string) *resourcemanagerv2.DeleteResourceGroupOptions {
	return s.mockNewDeleteResourceGroupOptions
}

func (s *resourceManagerServiceMock) DeleteResourceGroup(*resourcemanagerv2.DeleteResourceGroupOptions) (*core.DetailedResponse, error) {
	return s.mockDeleteResourceGroup, nil
}

func (s *resourceManagerServiceMock) GetResourceGroupIDByName(name string) (string, error) {
	id, ok := s.resourceGroups[name]
	if !ok {
		return "", fmt.Errorf("resource group %s not found", name)
	}
	return id, nil
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

func (mock *containerClientMock) Albs() containerv2.Alb {
	args := mock.Called()
	return args.Get(0).(containerv2.Alb) // Cast to the expected return type
}

type AlbsMock struct {
	mock.Mock
}

func (m *AlbsMock) AddIgnoredIngressStatusErrors(ignoredErrorsReq containerv2.IgnoredIngressStatusErrors, target containerv2.ClusterTargetHeader) error {
	args := m.Called(ignoredErrorsReq, target)
	return args.Error(0)
}

func (m *AlbsMock) CreateAlb(albCreateReq containerv2.AlbCreateReq, target containerv2.ClusterTargetHeader) (containerv2.AlbCreateResp, error) {
	args := m.Called(albCreateReq, target)
	return args.Get(0).(containerv2.AlbCreateResp), args.Error(1)
}

func (m *AlbsMock) DisableAlb(disableAlbReq containerv2.AlbConfig, target containerv2.ClusterTargetHeader) error {
	args := m.Called(disableAlbReq, target)
	return args.Error(0)
}

func (m *AlbsMock) EnableAlb(enableAlbReq containerv2.AlbConfig, target containerv2.ClusterTargetHeader) error {
	args := m.Called(enableAlbReq, target)
	return args.Error(0)
}

func (m *AlbsMock) GetALBAutoscaleConfiguration(clusterNameOrID, albID string, target containerv2.ClusterTargetHeader) (containerv2.AutoscaleDetails, error) {
	args := m.Called(clusterNameOrID, albID, target)
	return args.Get(0).(containerv2.AutoscaleDetails), args.Error(1)
}

func (m *AlbsMock) GetAlb(albid string, target containerv2.ClusterTargetHeader) (containerv2.AlbConfig, error) {
	args := m.Called(albid, target)
	return args.Get(0).(containerv2.AlbConfig), args.Error(1)
}

func (m *AlbsMock) GetAlbClusterHealthCheckConfig(clusterNameOrID string, target containerv2.ClusterTargetHeader) (containerv2.ALBClusterHealthCheckConfig, error) {
	args := m.Called(clusterNameOrID, target)
	return args.Get(0).(containerv2.ALBClusterHealthCheckConfig), args.Error(1)
}

func (m *AlbsMock) GetIgnoredIngressStatusErrors(clusterNameOrID string, target containerv2.ClusterTargetHeader) (containerv2.IgnoredIngressStatusErrors, error) {
	args := m.Called(clusterNameOrID, target)
	return args.Get(0).(containerv2.IgnoredIngressStatusErrors), args.Error(1)
}

func (m *AlbsMock) GetIngressLoadBalancerConfig(clusterNameOrID, lbType string, target containerv2.ClusterTargetHeader) (containerv2.ALBLBConfig, error) {
	args := m.Called(clusterNameOrID, lbType, target)
	return args.Get(0).(containerv2.ALBLBConfig), args.Error(1)
}

func (m *AlbsMock) GetIngressStatus(clusterNameOrID string, target containerv2.ClusterTargetHeader) (containerv2.IngressStatus, error) {
	args := m.Called(clusterNameOrID, target)
	return args.Get(0).(containerv2.IngressStatus), args.Error(1)
}

func (m *AlbsMock) ListAlbImages(target containerv2.ClusterTargetHeader) (containerv2.AlbImageVersions, error) {
	args := m.Called(target)
	return args.Get(0).(containerv2.AlbImageVersions), args.Error(1)
}

func (m *AlbsMock) ListClusterAlbs(clusterNameOrID string, target containerv2.ClusterTargetHeader) ([]containerv2.AlbConfig, error) {
	args := m.Called(clusterNameOrID, target)
	return args.Get(0).([]containerv2.AlbConfig), args.Error(1)
}

func (m *AlbsMock) RemoveALBAutoscaleConfiguration(clusterNameOrID, albID string, target containerv2.ClusterTargetHeader) error {
	args := m.Called(clusterNameOrID, albID, target)
	return args.Error(0)
}

func (m *AlbsMock) RemoveIgnoredIngressStatusErrors(ignoredErrorsReq containerv2.IgnoredIngressStatusErrors, target containerv2.ClusterTargetHeader) error {
	args := m.Called(ignoredErrorsReq, target)
	return args.Error(0)
}

func (m *AlbsMock) SetALBAutoscaleConfiguration(clusterNameOrID, albID string, autoscaleDetails containerv2.AutoscaleDetails, target containerv2.ClusterTargetHeader) error {
	args := m.Called(clusterNameOrID, albID, autoscaleDetails, target)
	return args.Error(0)
}

func (m *AlbsMock) SetAlbClusterHealthCheckConfig(albHealthCheckReq containerv2.ALBClusterHealthCheckConfig, target containerv2.ClusterTargetHeader) error {
	args := m.Called(albHealthCheckReq, target)
	return args.Error(0)
}

func (m *AlbsMock) SetIngressStatusState(ingressStatusStateReq containerv2.IngressStatusState, target containerv2.ClusterTargetHeader) error {
	args := m.Called(ingressStatusStateReq, target)
	return args.Error(0)
}

func (m *AlbsMock) UpdateAlb(updateAlbReq containerv2.UpdateALBReq, target containerv2.ClusterTargetHeader) error {
	args := m.Called(updateAlbReq, target)
	return args.Error(0)
}

func (m *AlbsMock) UpdateIngressLoadBalancerConfig(lbConfig containerv2.ALBLBConfig, target containerv2.ClusterTargetHeader) error {
	args := m.Called(lbConfig, target)
	return args.Error(0)
}

// ICD Versions mock
type icdServiceMock struct {
	mockListDeployablesResponse *clouddatabasesv5.ListDeployablesResponse
}

func (s *icdServiceMock) NewListDeployablesOptions() *clouddatabasesv5.ListDeployablesOptions {
	return &clouddatabasesv5.ListDeployablesOptions{}
}

func (s *icdServiceMock) ListDeployables(*clouddatabasesv5.ListDeployablesOptions) (*clouddatabasesv5.ListDeployablesResponse, *core.DetailedResponse, error) {
	return s.mockListDeployablesResponse, nil, nil
}
