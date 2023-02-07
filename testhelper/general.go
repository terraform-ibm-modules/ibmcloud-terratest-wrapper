// Package testhelper contains functions that can be used to assist and standardize the execution of unit tests for IBM Cloud Terraform projects
package testhelper

import (
	"log"
	"os"

	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/cloudinfo"
)

const ForceTestRegionEnvName = "FORCE_TEST_REGION"

// TesthelperTerraformOptions options object for optional variables to set
// primarily used for mocking external services in test cases
type TesthelperTerraformOptions struct {
	CloudInfoService              cloudinfo.CloudInfoServiceI
	ExcludeActivityTrackerRegions bool
}

// GetBestVpcRegion is a method that will determine a region available
// to the caller account that currently contains the least amount of deployed VPCs.
// The determination can be influenced by specifying a prefsFilePath pointed to a valid YAML file.
// If an OS ENV is found called FORCE_TEST_REGION then it will be used without querying.
// This function assumes that all default Options will be used.
// Returns a string representing an IBM Cloud region name, and error.
func GetBestVpcRegion(apiKey string, prefsFilePath string, defaultRegion string) (string, error) {
	return GetBestVpcRegionO(apiKey, prefsFilePath, defaultRegion, TesthelperTerraformOptions{})
}

// GetBestVpcRegionO is a method that will determine a region available
// to the caller account that currently contains the least amount of deployed VPCs.
// The determination can be influenced by specifying a prefsFilePath pointed to a valid YAML file.
// If an OS ENV is found called FORCE_TEST_REGION then it will be used without querying.
// Options data can also be called to supply the service to use that implements the correct interface.
// Returns a string representing an IBM Cloud region name, and error.
func GetBestVpcRegionO(apiKey string, prefsFilePath string, defaultRegion string, options TesthelperTerraformOptions) (string, error) {
	// If there is an OS ENV found to force the region, simply return that value and short-circuit this routine
	forceRegion, isForcePresent := os.LookupEnv(ForceTestRegionEnvName)
	if isForcePresent {
		return forceRegion, nil
	}

	cloudSvc, cloudSvcErr := configureCloudInfoService(apiKey, prefsFilePath, options)
	if cloudSvcErr != nil {
		log.Println("Error creating CloudInfoService for testhelper")
		return defaultRegion, cloudSvcErr
	}

	// THREAD SAFE OPERATION
	// Make this section thread safe with a mutex
	// If multiple parallel tests are using a shared cloudinfo instance, we want this function to only serve them one-at-a-time
	// so that they will not choose same region
	lock := cloudSvc.GetThreadLock()
	lock.Lock()
	defer lock.Unlock()

	// get best region
	var bestregion string
	var getErr error
	if options.ExcludeActivityTrackerRegions {
		bestregion, getErr = cloudSvc.GetLeastVpcTestRegionWithoutActivityTracker()
	} else {
		bestregion, getErr = cloudSvc.GetLeastVpcTestRegion()
	}
	if getErr != nil {
		log.Println("Error getting least vpc region")
		return defaultRegion, getErr
	}

	// regardless of error, if the bestregion returned is empty use default
	if len(bestregion) > 0 {
		log.Println("Best region was found!:", bestregion)
	} else {
		log.Println("Dynamic region not found, using default region:", defaultRegion)
		return defaultRegion, nil
	}

	// no matter how it was chosen, remove the region from further tests within this test run.
	// If multiple parallel tests are sharing the cloudinfo service, this will ensure that another
	// test will NOT select this region.
	cloudSvc.RemoveRegionForTest(bestregion)

	return bestregion, nil
}

// GetBestPowerSystemsRegion is a method that will determine a region available
// to the caller account that currently contains the least amount of deployed PowerVS Cloud Connections.
// The determination can be influenced by specifying a prefsFilePath pointed to a valid YAML file.
// If an OS ENV is found called FORCE_TEST_REGION then it will be used without querying.
// This function assumes that all default Options will be used.
// Returns a string representing an IBM Cloud region name, and error.
func GetBestPowerSystemsRegion(apiKey string, prefsFilePath string, defaultRegion string) (string, error) {
	return GetBestPowerSystemsRegionO(apiKey, prefsFilePath, defaultRegion, TesthelperTerraformOptions{})
}

// GetBestPowerSystemsRegionO is a method that will determine a region available
// to the caller account that currently contains the least amount of deployed PowerVS Cloud Connections.
// The determination can be influenced by specifying a prefsFilePath pointed to a valid YAML file.
// If an OS ENV is found called FORCE_TEST_REGION then it will be used without querying.
// Options data can also be called to supply the service to use that implements the correct interface.
// Returns a string representing an IBM Cloud region name, and error.
func GetBestPowerSystemsRegionO(apiKey string, prefsFilePath string, defaultRegion string, options TesthelperTerraformOptions) (string, error) {
	// set up initial best region as default

	// If there is an OS ENV found to force the region, simply return that value and short-circuit this routine
	forceRegion, isForcePresent := os.LookupEnv(ForceTestRegionEnvName)
	if isForcePresent {
		return forceRegion, nil
	}

	cloudSvc, cloudSvcErr := configureCloudInfoService(apiKey, prefsFilePath, options)
	if cloudSvcErr != nil {
		log.Println("Error creating CloudInfoService for testhelper")
		return defaultRegion, cloudSvcErr
	}

	// THREAD SAFE OPERATION
	// Make this section thread safe with a mutex
	// If multiple parallel tests are using a shared cloudinfo instance, we want this function to only serve them one-at-a-time
	// so that they will not choose same region
	lock := cloudSvc.GetThreadLock()
	lock.Lock()
	defer lock.Unlock()

	// get best region
	bestregion, getErr := cloudSvc.GetLeastPowerConnectionZone()
	if getErr != nil {
		log.Println("Error getting least PowerConnection region")
		return defaultRegion, getErr
	}

	// regardless of error, if the bestregion returned is empty use default
	if len(bestregion) > 0 {
		log.Println("Best region was found!:", bestregion)
	} else {
		log.Println("Dynamic region not found, using default region:", defaultRegion)
		return defaultRegion, nil
	}

	// no matter how it was chosen, remove the region from further tests within this test run.
	// If multiple parallel tests are sharing the cloudinfo service, this will ensure that another
	// test will NOT select this region.
	cloudSvc.RemoveRegionForTest(bestregion)

	return bestregion, nil
}

// configureCloudInfoService is a private function that will configure and set up a new CloudInfoService for testhelper
func configureCloudInfoService(apiKey string, prefsFilePath string, options TesthelperTerraformOptions) (cloudinfo.CloudInfoServiceI, error) {
	var cloudSvc cloudinfo.CloudInfoServiceI

	// configure new cloudinfosvc if required (not supplied in options)
	if options.CloudInfoService != nil {
		cloudSvc = options.CloudInfoService
	} else {
		// set up new service based on supplied values
		svcOptions := cloudinfo.CloudInfoServiceOptions{
			ApiKey: apiKey, //pragma: allowlist secret
		}
		cloudSvcRef, svcErr := cloudinfo.NewCloudInfoServiceWithKey(svcOptions)
		if svcErr != nil {
			log.Println("Error creating new CloudInfoService, using default region:", defaultRegion)
			return nil, svcErr
		}
		cloudSvc = cloudSvcRef
	}

	// THREAD SAFE OPERATION
	// Make this section thread safe with a mutex
	// If multiple parallel tests are using a shared cloudinfo instance, we want this function to only serve them one-at-a-time
	// so that they will not overwrite a previously loaded region list
	lock := cloudSvc.GetThreadLock()
	lock.Lock()
	defer lock.Unlock()

	// load a region prefs file if supplied and data does not already exist
	if len(prefsFilePath) > 0 && !cloudSvc.HasRegionData() {
		loadErr := cloudSvc.LoadRegionPrefsFromFile(prefsFilePath)
		if loadErr != nil {
			log.Println("Error loading CloudInfoService file, using default region:", defaultRegion)
			return nil, loadErr
		}
	}

	return cloudSvc, nil
}
