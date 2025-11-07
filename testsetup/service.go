package testsetup

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/hashicorp/go-memdb"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/cloudinfo"
)

const defaultTestConfigFileLocation = "/Users/toddgiguere/work/sandbox/cloud-service-region-limits.yaml"

// the main struct for the test setup service
type TestSetupService struct {
	CloudInfoSvc        cloudinfo.CloudInfoServiceI
	TestSetupOptions    *TestSetupServiceOptions
	ResourceDB          *memdb.MemDB
	TestRegions         []string
	ServiceRegionLimits []CloudServiceLimit

	// to control some single-thread functions
	lock sync.Mutex

	// local properties to keep track of things
	dataLoaded  bool
	dataLoadErr error
}

// main interface for the TestSetupService
type TestSetupServiceI interface {
}

// options when setting up a new TestSetupSvc
type TestSetupServiceOptions struct {
	CloudInfoService       cloudinfo.CloudInfoServiceI
	CloudApiKey            *string
	TestConfigFileLocation *string
}

func NewTestSetupService(options *TestSetupServiceOptions) (*TestSetupService, error) {

	// set up new empty service
	svc := new(TestSetupService)

	// set options to the service so that we have access to them later
	svc.TestSetupOptions = options

	// configure a cloudinfoservice if options are included
	if options.CloudInfoService != nil {
		svc.CloudInfoSvc = options.CloudInfoService
	} else if options.CloudApiKey != nil {
		newInfoSvc, newInfoSvcErr := configureCloudInfoService(options.CloudApiKey)
		if newInfoSvcErr != nil {
			return nil, newInfoSvcErr
		} else {
			svc.CloudInfoSvc = newInfoSvc
		}
	}

	return svc, nil
}

// helper function to set up a cloud info service if needed
func configureCloudInfoService(apiKey *string) (cloudinfo.CloudInfoServiceI, error) {

	// check that we have an api key set, if not throw custom error
	if apiKey == nil {
		return nil, fmt.Errorf("no cloud API key has been supplied for configuring CloudInfoService")
	}

	// set up new service based on supplied values
	cloudInfoSvcOptions := cloudinfo.CloudInfoServiceOptions{
		ApiKey: *apiKey, //pragma: allowlist secret
	}

	infoSvc, cloudInfoSvcErr := cloudinfo.NewCloudInfoServiceWithKey(cloudInfoSvcOptions)
	if cloudInfoSvcErr != nil {
		return nil, fmt.Errorf("error creating a new CloudInfoService: %w", cloudInfoSvcErr)
	}

	return infoSvc, nil

}

func (svc *TestSetupService) GetThreadLock() *sync.Mutex {
	return &svc.lock
}

// will get either existing or new CloudInfoService
func (svc *TestSetupService) GetCloudInfoService() (cloudinfo.CloudInfoServiceI, error) {

	var infoSvc cloudinfo.CloudInfoServiceI
	var newSvcErr error

	// if already configured, return
	if svc.CloudInfoSvc != nil {
		infoSvc = svc.CloudInfoSvc
	} else {
		// set up a new info service using supplied key
		infoSvc, newSvcErr = configureCloudInfoService(svc.TestSetupOptions.CloudApiKey)
		if newSvcErr != nil {
			return nil, newSvcErr
		}
		// keep in service for future use
		svc.CloudInfoSvc = infoSvc
	}

	return infoSvc, nil
}

// returns the most appropriate value for file location
func (svc *TestSetupService) getTestConfigFileLocation() string {
	// if user overrides, prefer that
	if svc.TestSetupOptions.TestConfigFileLocation != nil {
		return *svc.TestSetupOptions.TestConfigFileLocation
	}

	return defaultTestConfigFileLocation
}

// return if debug is set.
// turn on debug by having an env set: TESTWRAPPER_SETUP_DEBUG=true
func isDebugSet() bool {
	env := os.Getenv("TESTWRAPPER_SETUP_DEBUG")

	return strings.ToLower(env) == "true"

}
