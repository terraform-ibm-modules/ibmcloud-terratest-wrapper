// Package common contains general functions that are used by various packages and unit tests in ibmcloud-terratest-wrapper module
package common

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
	"gopkg.in/yaml.v3"

	"github.com/stretchr/testify/require"
)

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

// overwriting duplicate keys
func MergeMaps(maps ...map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}

// Adds value to map[key] only if value != compareValue
func ConditionalAdd(amap map[string]interface{}, key string, value string, compareValue string) {
	if value != compareValue {
		amap[key] = value
	}
}

// ConvertArrayToJsonString is a helper function that will take an array of Golang data types, and return a string
// of the array formatted as a JSON array.
// Helpful to convert Golang arrays into a format that Terraform can consume.
func ConvertArrayToJsonString(arr interface{}) (string, error) {
	// first marshal array into json compatible
	json, jsonErr := json.Marshal(arr)
	if jsonErr != nil {
		return "", jsonErr
	}

	// take json array, wrap as one string, and escape any double quotes inside
	s := string(json)

	return s, nil
}

// IsArray is a simple helper function that will determine if a given Golang value is a slice or array.
func IsArray(v interface{}) bool {

	theType := reflect.TypeOf(v).Kind()

	if (theType == reflect.Slice) || (theType == reflect.Array) {
		return true
	}

	return false
}

// StrArrayContains is a helper function that will check an array and see if a value is already present
func StrArrayContains(arr []string, val string) bool {
	for _, arrVal := range arr {
		if arrVal == val {
			return true
		}
	}

	return false
}

// IntArrayContains is a helper function that will check an array and see if an int value is already present
func IntArrayContains(arr []int, val int) bool {
	for _, arrVal := range arr {
		if arrVal == val {
			return true
		}
	}

	return false
}

// LoadMapFromYaml loads a YAML file into a map[string]interface{}.
// It returns the resulting map and any error encountered.
func LoadMapFromYaml(filePath string) (map[string]interface{}, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("file not found: %w", err)
		}
		return nil, err
	}

	var result map[string]interface{}
	err = yaml.Unmarshal(data, &result)
	if err != nil {
		return nil, fmt.Errorf("error parsing YAML data: %w", err)
	}

	return result, nil
}

// Generate an SSH RSA Keypair (4096 bits), and return the PublicKey in OpenSSH Authorized Key format.
// Used for tests to generate unique throw-away (but valid) SSH key to supply to test inputs.
// SPECIAL NOTE: the newline character at end of key will be trimmed and not included!
func GenerateSshRsaPublicKey() (string, error) {
	// generate a new RSA key
	newkey, keyerr := rsa.GenerateKey(rand.Reader, 4096)
	if keyerr != nil {
		return "", keyerr
	}

	// convert the RSA key into OpenSSH structure
	pubKey, ssherr := ssh.NewPublicKey(&newkey.PublicKey)
	if ssherr != nil {
		return "", ssherr
	}

	// marshall public key into "authorized_key" format string (from binary)
	pubKeyStr := string(ssh.MarshalAuthorizedKey(pubKey))

	// trim all whitespace, including trailing newline
	pubKeyStrTrim := strings.TrimSpace(pubKeyStr)

	return pubKeyStrTrim, nil
}