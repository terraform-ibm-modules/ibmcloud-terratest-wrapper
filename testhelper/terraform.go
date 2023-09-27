package testhelper

import (
	"errors"
	"fmt"
	"github.com/gruntwork-io/terratest/modules/files"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/common"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
)

// RemoveFromStateFile Attempts to remove resource from state file
func RemoveFromStateFile(stateFile string, resourceAddress string) (string, error) {
	var errorMsg string
	if files.PathContainsTerraformState(stateFile) {
		stateDir := filepath.Dir(stateFile)
		log.Printf("Removing %s from Statefile %s\n", resourceAddress, stateFile)
		command := fmt.Sprintf("terraform state rm %s", resourceAddress)
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
				err = fmt.Errorf("%wOutput: %s'%s'%s was not found\n", err, blueBold, key, reset)
			} else {
				err = fmt.Errorf("Output: %s'%s'%s was not found\n", blueBold, key, reset)
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
					err = fmt.Errorf("%wOutput: %s'%s'%s was not expected to be %s\n", err, blueBold, key, reset, expected)
				} else {
					err = fmt.Errorf("Output: %s'%s'%s was not expected to be %s\n", blueBold, key, reset, expected)
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
