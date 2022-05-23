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
