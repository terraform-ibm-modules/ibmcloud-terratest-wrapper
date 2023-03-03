package testhelper

import (
	"errors"
	"fmt"
	"github.com/gruntwork-io/terratest/modules/files"
	"log"
	"os/exec"
	"path/filepath"
	"testing"
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

// GetTerraformOutputs This function takes the output from terraform.OutputAll, 1..N strings.
// // For each string, it gets the value and checks if it could successfully get the value.
// // The function returns a map of the found values with their keys and a list of the output keys that were not found.
func GetTerraformOutputs(t *testing.T, outputs map[string]interface{}, outputKeys ...string) (map[string]interface{}, []string) {
	foundValues := make(map[string]interface{})
	var missingKeys []string

	for _, key := range outputKeys {
		value, ok := outputs[key]
		if !ok {
			missingKeys = append(missingKeys, key)
			t.Errorf("Output %s was not found", key)
		} else {
			if value != nil {
				foundValues[key] = value
			} else {
				t.Errorf("Output %s was not expected to be nil", key)
				missingKeys = append(missingKeys, key)
			}
		}
	}

	return foundValues, missingKeys
}
