package testhelper

import (
	"errors"
	"fmt"
	"github.com/gruntwork-io/terratest/modules/files"
	"log"
	"os/exec"
	"path/filepath"
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
	for _, key := range expectedKeys {
		value, ok := outputs[key]
		if !ok {
			missingKeys = append(missingKeys, key)
			if err != nil {
				err = fmt.Errorf("%wOutput %s was not found\n", err, key)
			} else {
				err = fmt.Errorf("Output %s was not found\n", key)
			}
		} else {
			if value == nil {
				missingKeys = append(missingKeys, key)
				if err != nil {
					err = fmt.Errorf("%wOutput %s was not expected to be nil\n", err, key)
				} else {
					err = fmt.Errorf("Output %s was not expected to be nil\n", key)
				}
			}
		}
	}

	return missingKeys, err
}
