package testhelper

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	"github.com/gruntwork-io/terratest/modules/files"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/gruntwork-io/terratest/modules/terraform"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/stretchr/testify/assert"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/common"
)

// RemoveFromStateFile Attempts to remove resource from state file
func RemoveFromStateFile(stateFile string, resourceAddress string) (string, error) {
	return RemoveFromStateFileV2(stateFile, resourceAddress, "terraform")
}

// RemoveFromStateFileV2 Attempts to remove resource from state file
// stateFile: The path to the state file
// resourceAddress: The address of the resource to remove
// tfBinary: The path to the terraform binary
func RemoveFromStateFileV2(stateFile string, resourceAddress string, tfBinary string) (string, error) {
	var errorMsg string
	if tfBinary == "" {
		tfBinary = "terraform"
	}
	if files.PathContainsTerraformState(stateFile) {
		stateDir := filepath.Dir(stateFile)
		log.Printf("Removing %s from Statefile %s\n", resourceAddress, stateFile)
		command := fmt.Sprintf("%s state rm %s", tfBinary, resourceAddress)
		log.Printf("Executing: %s", command)
		cmd := exec.Command("/bin/sh", "-c", command)
		cmd.Dir = stateDir
		out, err := cmd.CombinedOutput()
		if err != nil {
			errorMsg = fmt.Sprintf("An error occured removingresource '%s' from Statefile '%s': %s", resourceAddress, stateFile, err)
			return string(out), errors.New(errorMsg)
		}
		return string(out), nil
	} else {
		errorMsg = fmt.Sprintf("An error occured Statefile '%s' not found", stateFile)

		return "", errors.New(errorMsg)
	}
}

// ValidateTerraformOutputs takes a map of Terraform output keys and values, it checks if all the
// expected output keys are present. The function returns a list of the output keys that were not found
// and an error message that includes details about which keys were missing.
func ValidateTerraformOutputs(outputs map[string]interface{}, expectedKeys ...string) ([]string, error) {
	var missingKeys []string
	var err error
	// Set up ANSI escape codes for blue and bold text
	blueBold := "\033[1;34m"
	reset := "\033[0m"

	for _, key := range expectedKeys {
		value, ok := outputs[key]
		if !ok {
			missingKeys = append(missingKeys, key)
			if err != nil {
				err = fmt.Errorf("%wOutput: %s'%s'%s was not found", err, blueBold, key, reset)
			} else {
				err = fmt.Errorf("output: %s'%s'%s was not found", blueBold, key, reset)
			}
		} else {
			if value == nil || (reflect.TypeOf(value).String() == "string" && len(strings.Trim(value.(string), " ")) == 0) {
				missingKeys = append(missingKeys, key)
				expected := "unknown"
				if value == nil {
					expected = "nil"
				} else if len(strings.Trim(value.(string), " ")) == 0 {
					expected = "blank string"
				}
				if err != nil {
					err = fmt.Errorf("%wOutput: %s'%s'%s was not expected to be %s", err, blueBold, key, reset, expected)
				} else {
					err = fmt.Errorf("output: %s'%s'%s was not expected to be %s", blueBold, key, reset, expected)
				}
			}
		}
	}

	return missingKeys, err
}

// CleanTerraformDir removes the .terraform directory, other Terraform files, and files with the specified format from the directory
func CleanTerraformDir(directory string) {
	terraformFilesAndDirectories := []string{
		".terraform",
		".terraform.lock.hcl",
		"terraform.tfstate",
		"terraform.tfstate.backup",
	}

	// Define a regular expression pattern to match the desired file format
	pattern := `^terratest-plan-file-\d+$`
	re := regexp.MustCompile(pattern)

	// List files in the directory
	files, err := os.ReadDir(directory)
	if err != nil {
		log.Printf("Could not read directory for cleanup %s: %s Skipping...", directory, err)
		return
	}

	for _, file := range files {
		fileName := file.Name()
		filePath := filepath.Join(directory, fileName)

		// Check if it's one of the known Terraform files or a file matching the format
		if common.StrArrayContains(terraformFilesAndDirectories, fileName) || re.MatchString(fileName) {
			if err := os.RemoveAll(filePath); err != nil {
				// Ignore errors, just log them
				log.Printf("Error removing file %s: %s", fileName, err)
			}
		}
	}
}

// checkConsistency Fails the test if any destroys are detected and the resource is not exempt.
// If any addresses are provided in IgnoreUpdates.List then fail on updates too unless the resource is exempt
// Returns TRUE if there were consistency changes that were identified
func CheckConsistency(plan *terraform.PlanStruct, testOptions CheckConsistencyOptionsI) bool {
	validChange := false

	// extract consistency options from base set of options (schematic or terratest)
	options := testOptions.GetCheckConsistencyOptions()

	for _, resource := range plan.ResourceChangesMap {
		// get JSON string of full changes for the logs
		changesBytes, changesErr := json.MarshalIndent(resource.Change, "", "  ")
		// if it errors in the marshall step, just put a placeholder and move on, not important
		changesJson := "--UNAVAILABLE--"
		if changesErr == nil {
			changesJson = string(changesBytes)
		}

		var resourceDetails string

		// Treat all keys in the BeforeSensitive and AfterSensitive maps as sensitive
		// Assuming BeforeSensitive and AfterSensitive are of type interface{}
		beforeSensitive, beforeSensitiveOK := resource.Change.BeforeSensitive.(map[string]interface{})
		afterSensitive, afterSensitiveOK := resource.Change.AfterSensitive.(map[string]interface{})

		// Create the mergedSensitive map
		mergedSensitive := make(map[string]interface{})

		// Check if BeforeSensitive is of the expected type
		if beforeSensitiveOK {
			// Copy the keys and values from BeforeSensitive to the mergedSensitive map.
			for key, value := range beforeSensitive {
				// if value is non boolean, that means the terraform attribute was a map.
				// if a map, then it is only valid if it has fields assigned.
				// Terraform will leave the map empty if there are no sensitive fields, but still list the map itself.
				if isSanitizationSensitiveValue(value) {
					// take the safe route and assume anything else is sensitive
					mergedSensitive[key] = value
				}
			}
		}

		// Check if AfterSensitive is of the expected type
		if afterSensitiveOK {
			// Copy the keys and values from AfterSensitive to the mergedSensitive map.
			for key, value := range afterSensitive {
				// if value is non boolean, that means the terraform attribute was a map.
				// if a map, then it is only valid if it has fields assigned.
				// Terraform will leave the map empty if there are no sensitive fields, but still list the map itself.
				if isSanitizationSensitiveValue(value) {
					mergedSensitive[key] = value
				}
			}
		}

		// Perform sanitization
		sanitizedChangesJson, err := sanitizeResourceChanges(resource.Change, mergedSensitive)
		if err != nil {
			sanitizedChangesJson = "Error sanitizing sensitive data"
			logger.Log(options.Testing, sanitizedChangesJson)
		}
		formatChangesJson, err := common.FormatJsonStringPretty(sanitizedChangesJson)

		var formatChangesJsonString string
		if err != nil {
			logger.Log(options.Testing, "Error formatting JSON, use unformatted")
			formatChangesJsonString = sanitizedChangesJson
		} else {
			formatChangesJsonString = string(formatChangesJson)
		}

		diff, diffErr := common.GetBeforeAfterDiff(changesJson)

		if diffErr != nil {
			diff = fmt.Sprintf("Error getting diff: %s", diffErr)
		} else {
			// Split the changesJson into "Before" and "After" parts
			beforeAfter := strings.Split(diff, "After: ")

			// Perform sanitization on "After" part
			var after string
			if len(beforeAfter) > 1 {
				after, err = common.SanitizeSensitiveData(beforeAfter[1], mergedSensitive)
				handleSanitizationError(err, "after diff", options)
			} else {
				after = "Could not parse after from diff" // dont print incase diff contains sensitive values
			}

			// Perform sanitization on "Before" part
			var before string
			if len(beforeAfter) > 0 {
				before, err = common.SanitizeSensitiveData(strings.TrimPrefix(beforeAfter[0], "Before: "), mergedSensitive)
				handleSanitizationError(err, "before diff", options)
			} else {
				before = "Could not parse before from diff" // dont print incase diff contains sensitive values
			}

			// Reassemble the sanitized diff string
			diff = "  Before: \n\t" + before + "\n  After: \n\t" + after
		}
		resourceDetails = fmt.Sprintf("\nName: %s\nAddress: %s\nActions: %s\nDIFF:\n%s\n\nChange Detail:\n%s", resource.Name, resource.Address, resource.Change.Actions, diff, formatChangesJsonString)

		var errorMessage string
		if !options.IgnoreDestroys.IsExemptedResource(resource.Address) {
			errorMessage = fmt.Sprintf("Resource(s) identified to be destroyed %s", resourceDetails)
			assert.False(options.Testing, resource.Change.Actions.Delete(), errorMessage)
			assert.False(options.Testing, resource.Change.Actions.DestroyBeforeCreate(), errorMessage)
			assert.False(options.Testing, resource.Change.Actions.CreateBeforeDestroy(), errorMessage)
			validChange = true
		}
		if !options.IgnoreUpdates.IsExemptedResource(resource.Address) {
			errorMessage = fmt.Sprintf("Resource(s) identified to be updated %s", resourceDetails)
			assert.False(options.Testing, resource.Change.Actions.Update(), errorMessage)
			validChange = true
		}
		// We only want to check pure Adds (creates without destroy) if the consistency test is
		// NOT the result of an Upgrade, as some adds are expected when doing the Upgrade test
		// (such as new resources were added as part of the pull request)
		if !options.IsUpgradeTest {
			if !options.IgnoreAdds.IsExemptedResource(resource.Address) {
				errorMessage = fmt.Sprintf("Resource(s) identified to be created %s", resourceDetails)
				assert.False(options.Testing, resource.Change.Actions.Create(), errorMessage)
				validChange = true
			}
		}
	}

	return validChange
}

// sanitizeResourceChanges sanitizes the sensitive data in a Terraform JSON Change and returns the sanitized JSON.
func sanitizeResourceChanges(change *tfjson.Change, mergedSensitive map[string]interface{}) (string, error) {
	// Marshal the Change to JSON bytes
	changesBytes, err := json.MarshalIndent(change, "", "  ")
	if err != nil {
		return "", err
	}
	changesJson := string(changesBytes)

	// Perform sanitization of sensitive data
	changesJson, err = common.SanitizeSensitiveData(changesJson, mergedSensitive)
	return changesJson, err
}

// handleSanitizationError logs an error message if a sanitization error occurs.
func handleSanitizationError(err error, location string, options *CheckConsistencyOptions) {
	if err != nil {
		errorMessage := fmt.Sprintf("Error sanitizing sensitive data in %s", location)
		logger.Log(options.Testing, errorMessage)
	}
}

// isSanitizationSensitiveValue will look at the value data type of an attribute identified as sensitive in a TF plan
// only boolean values or maps with one or more fields are considered sensitive.
func isSanitizationSensitiveValue(value interface{}) bool {
	isSensitive := true // take safe route
	// if value is non boolean, that means the terraform attribute was a map.
	// if a map, then it is only valid if it has fields assigned.
	// Terraform will leave the map empty if there are no sensitive fields, but still list the map itself.

	//lint:ignore S1034 we do not have need for the value of the type
	switch value.(type) {
	case bool:
		isSensitive = true
	case map[string]interface{}:
		// if a map, check if length > 0 to see if this map has at least one sensitive field
		if len(value.(map[string]interface{})) > 0 {
			isSensitive = true
		} else {
			isSensitive = false
		}
	default:
		// take the safe route and assume anything else is sensitive
		isSensitive = true
	}

	return isSensitive
}
