package testschematic

import (
	"encoding/json"
	"reflect"
)

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
