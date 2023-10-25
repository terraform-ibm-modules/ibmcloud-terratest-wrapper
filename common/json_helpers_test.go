package common

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ######### SortSlice ##############
// Validate results with simple slice
func TestSortSlice(t *testing.T) {
	slice := []interface{}{9, 4, 7, 2, 1, 5, 8, 3, 6}
	expected := []interface{}{1, 2, 3, 4, 5, 6, 7, 8, 9}
	SortSlice(slice)
	assert.Equal(t, expected, slice)
}

// Validate results with nested slice: (slice of slices)
func TestSortSliceNestedPositive(t *testing.T) {

	slice := []interface{}{
		[]interface{}{9, 4, 7},
		[]interface{}{2, 1, 5},
		[]interface{}{8, 3, 6},
	}
	expected := []interface{}{
		[]interface{}{1, 2, 5},
		[]interface{}{3, 6, 8},
		[]interface{}{4, 7, 9},
	}
	SortSlice(slice)
	assert.Equal(t, expected, slice)
}

func TestSortSliceNestedNegative(t *testing.T) {

	slice := []interface{}{
		[]interface{}{9, 4, 7},
		[]interface{}{2, 1, 5},
		[]interface{}{8, 3, 6},
	}
	expected := []interface{}{
		[]interface{}{1, 2, 3},
		[]interface{}{4, 5, 7},
		[]interface{}{6, 8, 9},
	}
	SortSlice(slice)
	assert.NotEqual(t, expected, slice)
}

// Validate results with nested slice: (slice of maps)
func TestSortSliceOfMaps(t *testing.T) {

	slice := []interface{}{
		map[string]interface{}{"user": "non-admin", "pwd": "Hello@123"},
		map[string]interface{}{"user": "admin", "pwd": "Bye@098"},
	}
	expected := []interface{}{
		map[string]interface{}{"pwd": "Bye@098", "user": "admin"},
		map[string]interface{}{"pwd": "Hello@123", "user": "non-admin"},
	}
	SortSlice(slice)
	assert.Equal(t, expected, slice)
}

func TestSortMapKeys(t *testing.T) {
	testmap := map[string]interface{}{"Name": "Robocop", "Gender": "Male"}
	expected := []string{"Gender", "Name"}
	assert.Equal(t, expected, SortMapKeys(testmap))
}

func TestSortMap(t *testing.T) {
	testmap := map[string]interface{}{
		"key2": "value2",
		"key1": "value1",
	}
	expected := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}
	SortMap(testmap)
	assert.Equal(t, expected, testmap)
}

func TestIsJsonEqual(t *testing.T) {

	var json1 = "../sample/config/sample1.json"
	var json2 = "../sample/config/sample2.json"

	result, err := IsJsonEqual(json1, json2)
	if err != nil {
		fmt.Println(err)
	}
	assert.Equal(t, true, result)
}

// Verify two unequal json files
func TestIsJsonNotEqual(t *testing.T) {

	var json1 = "../sample/config/sample1.json"
	var json3 = "../sample/config/sample3.json"

	result, err := IsJsonEqual(json1, json3)
	if err != nil {
		fmt.Println(err)
	}
	assert.Equal(t, false, result)
}

func TestSanitizeSensitiveData(t *testing.T) {
	tests := []struct {
		Name       string
		InputJSON  string
		SecureList map[string]interface{}
		Expected   string
		Err        string
	}{
		{
			Name:       "Empty JSON",
			InputJSON:  `{}`,
			SecureList: map[string]interface{}{},
			Expected:   `{}`,
			Err:        "",
		},
		{
			Name:       "Empty Secure List",
			InputJSON:  `{"key": "sensitive_value"}`,
			SecureList: map[string]interface{}{},
			Expected:   `{"key":"sensitive_value"}`,
			Err:        "",
		},
		{
			Name:       "Sanitize Sensitive Value",
			InputJSON:  `{"password": "sensitive_value"}`,
			SecureList: map[string]interface{}{"password": true},
			Expected:   `{"password":"SECURE_VALUE_HIDDEN_HASH:"}`,
			Err:        "",
		},
		{
			Name:       "Sanitize Sensitive Value with other keys",
			InputJSON:  `{"key1": "value1", "password": "sensitive_value", "key2": "value2"}`,
			SecureList: map[string]interface{}{"password": true},
			Expected:   `{"key1":"value1","password":"SECURE_VALUE_HIDDEN_HASH:","key2":"value2"}`,
			Err:        "",
		},
		{
			Name:       "Sanitize Multiple Sensitive Values",
			InputJSON:  `{"password": "sensitive_value", "token": "sensitive_value"}`,
			SecureList: map[string]interface{}{"password": true, "token": true},
			Expected:   `{"password":"SECURE_VALUE_HIDDEN_HASH:","token":"SECURE_VALUE_HIDDEN_HASH:"}`,
			Err:        "",
		},
		{
			Name:       "Nested JSON",
			InputJSON:  `{"nested": {"key": "sensitive_value"}}`,
			SecureList: map[string]interface{}{"key": true},
			Expected:   `{"nested":{"key":"SECURE_VALUE_HIDDEN_HASH:"}}`,
			Err:        "",
		},
		{
			Name:       "Nested JSON with nested values",
			InputJSON:  `{"nested": {"key": {"subkey": "sensitive_value"}}}`,
			SecureList: map[string]interface{}{"subkey": true},
			Expected:   `{"nested":{"key":{"subkey":"SECURE_VALUE_HIDDEN_HASH:"}}}`,
			Err:        "",
		},
		{
			Name:       "JSON Parsing Error",
			InputJSON:  "{malformed json}",
			SecureList: map[string]interface{}{},
			Expected:   "", // The expected result is an empty string because parsing fails
			Err:        "invalid character 'm' looking for beginning of object key string",
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			result, err := SanitizeSensitiveData(test.InputJSON, test.SecureList)

			if err != nil {
				assert.Equal(t, test.Err, err.Error(), "Error mismatch")
			} else {
				// Unmarshal the JSON strings into maps
				var expectedMap map[string]interface{}
				var resultMap map[string]interface{}

				if err := json.Unmarshal([]byte(test.Expected), &expectedMap); err != nil {
					t.Errorf("Error unmarshalling expected JSON: %v", err)
				}

				if err := json.Unmarshal([]byte(result), &resultMap); err != nil {
					t.Errorf("Error unmarshalling result JSON: %v", err)
				}

				// Strip hashed values from the maps
				expectedMap = stripHashesFromMap(expectedMap)
				resultMap = stripHashesFromMap(resultMap)

				// Compare the maps
				assert.Equal(t, expectedMap, resultMap, "Result mismatch")
			}
		})
	}
}

// Helper function to strip hashed values from a map
func stripHashesFromMap(inputMap map[string]interface{}) map[string]interface{} {
	for key, value := range inputMap {
		if str, ok := value.(string); ok {
			if strings.HasPrefix(str, "SECURE_VALUE_HIDDEN_HASH:") {
				// Replace hashed values with an empty string
				inputMap[key] = "SECURE_VALUE_HIDDEN_HASH:"
			}
		} else if subMap, ok := value.(map[string]interface{}); ok {
			// Recursively strip hashed values from sub-maps
			inputMap[key] = stripHashesFromMap(subMap)
		}
	}
	return inputMap
}
