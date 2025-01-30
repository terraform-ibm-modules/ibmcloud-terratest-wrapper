package common

// Compare the JSON values (intended to compare override json and config output
// in case of SLZ but can be used anywhere)
import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// SortSlice sorts the slices recursively.
// It also takes care of any nested slices or maps inside the slice.
func SortSlice(slice []interface{}) {
	for i, item := range slice {
		switch item := item.(type) {
		case []interface{}: // If the item is a slice, sort it
			SortSlice(item)
		case map[string]interface{}: // If the item is a map, sort it
			SortMap(item)
		}
		slice[i] = item
	}
	// Sort the slice itself. Uses string representation for comparison.
	sort.SliceStable(slice, func(i, j int) bool {
		return fmt.Sprintf("%s", slice[i]) < fmt.Sprintf("%s", slice[j])
	})
}

// SortMapKeys sorts the keys of a map and returns them as a slice.
func SortMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// SortMap sorts the map and any nested structures inside it.
// It modifies the original map.
func SortMap(m map[string]interface{}) {
	for key, val := range m {
		switch val := val.(type) {
		case []interface{}: // If the value is a slice, sort it
			SortSlice(val)
			m[key] = val
		case map[string]interface{}: // If the value is a map, sort it
			SortMap(val)
		}
	}
	// Sort the map itself based on the keys
	keys := SortMapKeys(m)
	sortedMap := make(map[string]interface{})
	for _, key := range keys {
		sortedMap[key] = m[key]
	}
	// Update the original map with sorted key-value pairs
	for k := range m {
		delete(m, k)
	}
	for k, v := range sortedMap {
		m[k] = v
	}
}

// IsJsonEqual validates whether the two JSON files are equal or not
func IsJsonEqual(jsonFile1 string, jsonFile2 string) (bool, error) {
	// Read JSON from files
	jsonData1, err := os.ReadFile(jsonFile1)
	if err != nil {
		newErr := fmt.Errorf("error reading json file %s :  %w", jsonFile1, err)
		return false, newErr
	}

	jsonData2, err := os.ReadFile(jsonFile2)
	if err != nil {
		newErr := fmt.Errorf("error reading json file %s :  %w", jsonFile2, err)
		return false, newErr
	}

	// Unmarshal JSON data into generic map[string]interface{}
	var data1, data2 map[string]interface{}
	err = json.Unmarshal(jsonData1, &data1)
	if err != nil {
		newErr := fmt.Errorf("error while parsing %s :  %w", jsonFile1, err)
		return false, newErr
	}

	err = json.Unmarshal(jsonData2, &data2)
	if err != nil {
		newErr := fmt.Errorf("error while parsing %s :  %w", jsonFile2, err)
		return false, newErr
	}

	// Sort the maps to ensure the keys are in a consistent order
	SortMap(data1)
	SortMap(data2)

	// Compare the maps using go-cmp with a custom slice comparator and float tolerance
	diff := cmp.Diff(data1, data2, cmpopts.EquateEmpty(), cmpopts.EquateApprox(0.0, 0.00001))
	if diff != "" {
		// If diff is not empty, create an error object with the diff string
		return false, errors.New(diff)
	}
	return true, nil
}

func FormatJsonStringPretty(jsonString string) (string, error) {
	var out bytes.Buffer
	err := json.Indent(&out, []byte(jsonString), "", "  ")
	if err != nil {
		return "", err
	}
	return out.String(), nil
}

// SANITIZE_STRING is the string used to replace sensitive values.
const SANITIZE_STRING = "SECURE_VALUE_HIDDEN_HASH:"

// SanitizeSensitiveData takes a JSON string and a list of sensitive keys
// and replaces the values of the sensitive keys with a predefined string.
func SanitizeSensitiveData(inputJSON string, secureList map[string]interface{}, sensitiveKeys []string) (string, error) {
	// Unmarshal the input JSON into a generic data structure.
	var data interface{}
	if err := json.Unmarshal([]byte(inputJSON), &data); err != nil {
		return "", err
	}

	// Recursively sanitize the JSON data.
	sanitizeJSON(data, secureList, sensitiveKeys)

	// Marshal the sanitized data back into a JSON string.
	sanitizedJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}

	return string(sanitizedJSON), nil
}

// sanitizeJSON is a recursive function that traverses a JSON data structure
// and replaces sensitive keys with SANITIZE_STRING.
func sanitizeJSON(data interface{}, secureList map[string]interface{}, sensitiveKeys []string) {
	switch v := data.(type) {
	case map[string]interface{}:
		for key := range v {
			// NOTE: before and after sensitive sections do not contain values, only booleans denoting sensitive, so skip these sections
			if key == "before_sensitive" || key == "after_sensitive" || key == "after_unknown" {
				// Check for sensitive keys within these keys and sanitize them
				if nestedMap, ok := v[key].(map[string]interface{}); ok {
					for nestedKey := range nestedMap {
						if contains(sensitiveKeys, nestedKey) {
							nestedMap[nestedKey] = SANITIZE_STRING
						}
					}
				}
			} else {
				if _, ok := secureList[key]; ok {
					// Generate a random salt value
					salt := make([]byte, 16) // You can choose the salt length as needed
					_, err := rand.Read(salt)
					if err != nil {
						fmt.Println("Error generating salt:", err)
						return
					}

					// Concatenate the salt and input
					saltedInput := append(salt, []byte(fmt.Sprintf("%v", v[key]))...)
					// Replace sensitive values with SANITIZE_STRING+Hash of the value.
					hashedValue := sha256.Sum224(saltedInput)
					v[key] = SANITIZE_STRING + fmt.Sprintf("-%x", hashedValue)
				} else {
					// Recursively sanitize nested data.
					sanitizeJSON(v[key], secureList, sensitiveKeys)
				}
			}
		}
	case []interface{}:
		for i, item := range v {
			// Recursively sanitize each item in the array.
			sanitizeJSON(item, secureList, sensitiveKeys)
			v[i] = item
		}
	}
}

// contains checks if a slice contains a specific element.
func contains(slice []string, element string) bool {
	for _, item := range slice {
		if item == element {
			return true
		}
	}
	return false
}

// PrintStructAsJson prints a struct as a formatted JSON string
func PrintStructAsJson(data interface{}) {
	b, _ := json.MarshalIndent(data, "", "  ")
	fmt.Println(string(b))
}
