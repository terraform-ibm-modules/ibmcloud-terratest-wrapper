package common

// Compare the JSON values (intended to compare override json and config output
// in case of SLZ but can be used anywhere)
import (
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
