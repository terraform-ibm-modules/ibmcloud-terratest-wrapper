package common

import (
	"fmt"
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
