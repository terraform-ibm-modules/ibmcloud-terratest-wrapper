// Package testhelper General helper functions that can be used in tests
package testhelper

import (
	"github.com/stretchr/testify/require"
	"log"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/cloudinfo"
)

const ForceTestRegionEnvName = "FORCE_TEST_REGION"

// TesthelperTerraformOptions options object for optional variables to set
// primarily used for mocking external services in test cases
type TesthelperTerraformOptions struct {
	CloudInfoService cloudInfoServiceI
}

// interface for the cloudinfo service (can be mocked in tests)
type cloudInfoServiceI interface {
	GetLeastVpcTestRegion() (string, error)
	LoadRegionPrefsFromFile(string) error
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
	// set up initial best region as default
	var cloudSvc cloudInfoServiceI

	// If there is an OS ENV found to force the region, simply return that value and short-circuit this routine
	forceRegion, isForcePresent := os.LookupEnv(ForceTestRegionEnvName)
	if isForcePresent {
		return forceRegion, nil
	}

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
			return defaultRegion, svcErr
		}
		cloudSvc = cloudSvcRef
	}

	// load a region prefs file if supplied
	if len(prefsFilePath) > 0 {
		loadErr := cloudSvc.LoadRegionPrefsFromFile(prefsFilePath)
		if loadErr != nil {
			log.Println("Error loading CloudInfoService file, using default region:", defaultRegion)
			return defaultRegion, loadErr
		}
	}

	// get best region
	bestregion, getErr := cloudSvc.GetLeastVpcTestRegion()
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

	return bestregion, nil
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
