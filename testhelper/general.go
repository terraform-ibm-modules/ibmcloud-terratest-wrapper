// Package testhelper General helper functions that can be used in tests
package testhelper

import (
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/cloudinfo"
)

const ForceTestRegionEnvName = "FORCE_TEST_REGION"

// var lock sync.Mutex // use for thread-safe operations

// TesthelperTerraformOptions options object for optional variables to set
// primarily used for mocking external services in test cases
type TesthelperTerraformOptions struct {
	CloudInfoService              CloudInfoServiceI
	ExcludeActivityTrackerRegions bool
}

// interface for the cloudinfo service (can be mocked in tests)
type CloudInfoServiceI interface {
	GetLeastVpcTestRegion() (string, error)
	GetLeastVpcTestRegionWithoutActivityTracker() (string, error)
	GetLeastPowerConnectionZone() (string, error)
	LoadRegionPrefsFromFile(string) error
	HasRegionData() bool
	RemoveRegionForTest(string)
	GetThreadLock() *sync.Mutex
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
func configureCloudInfoService(apiKey string, prefsFilePath string, options TesthelperTerraformOptions) (CloudInfoServiceI, error) {
	var cloudSvc CloudInfoServiceI

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

// GetRequiredEnvVars returns a map containing required environment variables and their values
// Fails the test if any are missing
func GetRequiredEnvVars(t *testing.T, variableNames []string) map[string]string {
	var missingVariables []string
	envVars := make(map[string]string)

	for _, variableName := range variableNames {
		val, present := os.LookupEnv(variableName)
		if present {
			envVars[variableName] = val
		} else {
			missingVariables = append(missingVariables, variableName)
		}
	}
	require.Empty(t, missingVariables, "The following environment variables must be set: %v", missingVariables)

	return envVars
}

// GitRootPath gets the path to the current git repos root directory
func GitRootPath(fromPath string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = fromPath
	path, err := cmd.Output()

	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(path)), nil
}

// GetBeforeAfterDiff takes a JSON string as input and returns a string with the differences
// between the "before" and "after" objects in the JSON.
//
// For example, given the JSON string:
//
//	{"before": {"a": 1, "b": 2}, "after": {"a": 2, "b": 3}}
//
// the function would return the string:
//
//	"Before: {"b": 2}\nAfter: {"a": 2, "b": 3}"
func GetBeforeAfterDiff(jsonString string) string {
	// Parse the JSON string into a map
	var jsonMap map[string]interface{}
	err := json.Unmarshal([]byte(jsonString), &jsonMap)
	if err != nil {
		return "Error: unable to parse JSON string"
	}

	// Get the "before" and "after" values from the map
	before, beforeOk := jsonMap["before"]
	after, afterOk := jsonMap["after"]
	if !beforeOk || !afterOk {
		return "Error: missing 'before' or 'after' key in JSON"
	}

	// Check if the "before" and "after" values are objects
	beforeObject, beforeOk := before.(map[string]interface{})
	if !beforeOk {
		return "Error: 'before' value is not an object"
	}
	afterObject, afterOk := after.(map[string]interface{})
	if !afterOk {
		return "Error: 'after' value is not an object"
	}

	// Find the differences between the two objects
	diffsBefore := make(map[string]interface{})
	for key, value := range beforeObject {
		if !reflect.DeepEqual(afterObject[key], value) {
			diffsBefore[key] = value
		}
	}

	// Convert the diffs map to a JSON string
	diffsJson, err := json.Marshal(diffsBefore)
	if err != nil {
		return "Error: unable to convert diffs to JSON"
	}

	// Find the differences between the two objects
	diffsAfter := make(map[string]interface{})
	for key, value := range afterObject {
		if !reflect.DeepEqual(beforeObject[key], value) {
			diffsAfter[key] = value
		}
	}

	// Convert the diffs map to a JSON string
	diffsJson2, err := json.Marshal(diffsAfter)
	if err != nil {
		return "Error: unable to convert diffs2 to JSON"
	}

	return "Before: " + string(diffsJson) + "\nAfter: " + string(diffsJson2)
}
