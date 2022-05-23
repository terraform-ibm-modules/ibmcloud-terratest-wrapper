package cloudinfo

import (
	"log"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/vpc-go-sdk/vpcv1"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

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

func TestNewServiceWithKey(t *testing.T) {
	serviceOptions := CloudInfoServiceOptions{
		ApiKey:     "dummy_key",
		VpcService: new(vpcServiceMock),
	}

	_, err := NewCloudInfoServiceWithKey(serviceOptions)

	require.Nil(t, err, "Error returned getting new service")
}

func TestNewServiceWithEnv(t *testing.T) {
	serviceOptions := CloudInfoServiceOptions{
		VpcService: new(vpcServiceMock),
	}

	os.Setenv("TEST_KEY_VAL", "dummy_key")
	_, err := NewCloudInfoServiceFromEnv("TEST_KEY_VAL", serviceOptions)

	require.Nil(t, err, "Error returned getting new service")

}

func TestNewServiceWithEmptyKey(t *testing.T) {
	serviceOptions := CloudInfoServiceOptions{
		VpcService: new(vpcServiceMock),
	}

	_, err := NewCloudInfoServiceWithKey(serviceOptions)

	require.NotNil(t, err, "Empty key should have resulted in error")
}

func TestNewServiceWithEmptyEnv(t *testing.T) {
	serviceOptions := CloudInfoServiceOptions{
		VpcService: new(vpcServiceMock),
	}

	_, err := NewCloudInfoServiceFromEnv("", serviceOptions)

	require.NotNil(t, err, "Empty Environment key should have resulted in error")

}
