// Package cloudinfo contains functions and methods for search and detailing various resources located in the IBM Cloud
package cloudinfo

import (
	"errors"
	"log"
	"os"

	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/vpc-go-sdk/vpcv1"
)

// CloudInfoService is a structure that is used as the receiver to many methods in this package.
// It contains references to other important services and data structures needed to perform these methods.
type CloudInfoService struct {
	vpcService  vpcService
	regionsData []RegionData
}

// CloudInfoServiceOptions structure used as input params for service constructor.
type CloudInfoServiceOptions struct {
	ApiKey      string
	VpcService  vpcService
	RegionPrefs []RegionData
}

// RegionData is a data structure used for holding configurable information about a region.
// Most of this data is configured by the caller in order to affect certain processing routines.
type RegionData struct {
	Name         string
	UseForTest   bool `yaml:"useForTest"`
	TestPriority int  `yaml:"testPriority"`
	Endpoint     string
	Status       string
	VpcCount     int
}

// vpcService interface for an external VPC Service API. Used for mocking external service in tests.
type vpcService interface {
	ListRegions(*vpcv1.ListRegionsOptions) (*vpcv1.RegionCollection, *core.DetailedResponse, error)
	GetRegion(*vpcv1.GetRegionOptions) (*vpcv1.Region, *core.DetailedResponse, error)
	NewGetRegionOptions(string) *vpcv1.GetRegionOptions
	ListVpcs(*vpcv1.ListVpcsOptions) (*vpcv1.VPCCollection, *core.DetailedResponse, error)
	SetServiceURL(string) error
}

// SortedRegionsDataByPriority is an array of RegionData struct that is used as a receiver to implement the
// sort interface (Len/Less/Swap) with supplied methods to sort the array on the field RegionData.TestPriority.
type SortedRegionsDataByPriority []RegionData

func (regions SortedRegionsDataByPriority) Len() int { return len(regions) }
func (regions SortedRegionsDataByPriority) Less(i, j int) bool {
	return regions[i].TestPriority < regions[j].TestPriority
}
func (regions SortedRegionsDataByPriority) Swap(i, j int) {
	regions[i], regions[j] = regions[j], regions[i]
}

// NewCloudInfoServiceWithKey is a factory function used for creating a new initialized service structure.
// This function can be called if an IBM Cloud API Key is known and passed in directly.
// Returns a pointer to an initialized CloudInfoService and error.
func NewCloudInfoServiceWithKey(options CloudInfoServiceOptions) (*CloudInfoService, error) {
	infoSvc := new(CloudInfoService)

	// need a valid key
	if len(options.ApiKey) == 0 {
		log.Println("ERROR: empty API KEY")
		return nil, errors.New("Empty API Key supplied")
	}

	// if vpcService is not supplied, use default of external service
	if options.VpcService != nil {
		infoSvc.vpcService = options.VpcService
	} else {
		// Instantiate the service with an API key based IAM authenticator
		vpcService, vpcErr := vpcv1.NewVpcV1(&vpcv1.VpcV1Options{
			Authenticator: &core.IamAuthenticator{
				ApiKey: options.ApiKey,
			},
		})
		if vpcErr != nil {
			log.Println("ERROR: Could not create NewVpcV1 service:", vpcErr)
			return nil, vpcErr
		}

		infoSvc.vpcService = vpcService
	}

	return infoSvc, nil
}

// NewCloudInfoServiceFromEnv is a factory function used for creating a new initialized service structure.
// This function can be called if the IBM Cloud API Key should be extracted from an existing OS level environment variable.
// Returns a pointer to an initialized CloudInfoService and error.
func NewCloudInfoServiceFromEnv(apiKeyEnv string, options CloudInfoServiceOptions) (*CloudInfoService, error) {
	apiKey := os.Getenv(apiKeyEnv)
	if apiKey == "" {
		return nil, errors.New("No API key Environment variable set")
	}

	options.ApiKey = apiKey

	return NewCloudInfoServiceWithKey(options)
}
