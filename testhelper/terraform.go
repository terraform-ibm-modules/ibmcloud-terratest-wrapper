package testhelper

import (
	"errors"
	"fmt"
	"github.com/gruntwork-io/terratest/modules/files"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
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

// CleanTerraformDir removes the .terraform directory and other terraform files .terraform.lock.hcl, terraform.tfstate, terraform.tfstate.backup from the directory
func CleanTerraformDir(directory string) {
	terraformFilesAndDirectories := []string{".terraform", ".terraform.lock.hcl", "terraform.tfstate", "terraform.tfstate.backup"}
	for _, file := range terraformFilesAndDirectories {
		// Check if file exists then remove
		if _, err := os.Stat(filepath.Join(directory, file)); err == nil {
			err := os.RemoveAll(filepath.Join(directory, file))
			if err != nil {
				// ignore errors, just log them
				log.Printf("Error removing file %s: %s", file, err)
			}
		}
	}
}
