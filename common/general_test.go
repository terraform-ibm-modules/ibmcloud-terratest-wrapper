package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetRequiredEnvVarsSuccess(t *testing.T) {
	t.Setenv("A_REQUIRED_VARIABLE", "The Value")
	t.Setenv("ANOTHER_VARIABLE", "Another Value")

	expected := make(map[string]string)
	expected["A_REQUIRED_VARIABLE"] = "The Value"
	expected["ANOTHER_VARIABLE"] = "Another Value"

	assert.Equal(t, expected, GetRequiredEnvVars(t, []string{"A_REQUIRED_VARIABLE", "ANOTHER_VARIABLE"}))
}

func TestGetRequiredEnvVarsEmptyInput(t *testing.T) {

	expected := make(map[string]string)
	assert.Equal(t, expected, GetRequiredEnvVars(t, []string{}))
}

func TestGetBeforeAfterDiffValidInput(t *testing.T) {
	jsonString := `{"before": {"a": 1, "b": 2}, "after": {"a": 2, "b": 3}}`
	expected := "Before: {\"a\":1,\"b\":2}\nAfter: {\"a\":2,\"b\":3}"
	result := GetBeforeAfterDiff(jsonString)
	if result != expected {
		t.Errorf("TestGetBeforeAfterDiffValidInput(%q) returned %q, expected %q", jsonString, result, expected)
	}
}

func TestGetBeforeAfterDiffMissingBeforeKey(t *testing.T) {
	jsonString := `{"after": {"a": 1, "b": 2}}`
	expected := "Error: missing 'before' or 'after' key in JSON"
	result := GetBeforeAfterDiff(jsonString)
	if result != expected {
		t.Errorf("TestGetBeforeAfterDiffMissingBeforeKey(%q) returned %q, expected %q", jsonString, result, expected)
	}
}

func TestGetBeforeAfterDiffNonObjectBeforeValue(t *testing.T) {
	jsonString := `{"before": ["a", "b"], "after": {"a": 1, "b": 2}}`
	expected := "Error: 'before' value is not an object"
	result := GetBeforeAfterDiff(jsonString)
	if result != expected {
		t.Errorf("TestGetBeforeAfterDiffNonObjectBeforeValue(%q) returned %q, expected %q", jsonString, result, expected)
	}
}

func TestGetBeforeAfterDiffNonObjectAfterValue(t *testing.T) {
	jsonString := `{"before": {"a": 1, "b": 2}, "after": ["a", "b"]}`
	expected := "Error: 'after' value is not an object"
	result := GetBeforeAfterDiff(jsonString)
	if result != expected {
		t.Errorf("TestGetBeforeAfterDiffNonObjectAfterValue(%q) returned %q, expected %q", jsonString, result, expected)
	}
}

func TestGetBeforeAfterDiffInvalidJSON(t *testing.T) {
	jsonString := `{"before": {"a": 1, "b": 2}, "after": {"a": 1, "b": 2}`
	expected := "Error: unable to parse JSON string"
	result := GetBeforeAfterDiff(jsonString)
	if result != expected {
		t.Errorf("TestGetBeforeAfterDiffInvalidJSON(%q) returned %q, expected %q", jsonString, result, expected)
	}
}

func TestConvertArrayJson(t *testing.T) {

	t.Run("GoodArray", func(t *testing.T) {
		goodArr := []interface{}{
			"hello",
			true,
			99,
			"bye",
		}
		goodStr, goodErr := ConvertArrayToJsonString(goodArr)
		if assert.NoError(t, goodErr, "error converting array") {
			assert.NotEmpty(t, goodStr)
		}
	})

	t.Run("NilValue", func(t *testing.T) {
		nullVal, nullErr := ConvertArrayToJsonString(nil)
		if assert.NoError(t, nullErr) {
			assert.Equal(t, "null", nullVal)
		}
	})

	t.Run("NullPointer", func(t *testing.T) {
		var intPtr *int
		ptrVal, ptrErr := ConvertArrayToJsonString(intPtr)
		if assert.NoError(t, ptrErr) {
			assert.Equal(t, "null", ptrVal)
		}
	})
}

func TestIsArray(t *testing.T) {

	t.Run("IsSlice", func(t *testing.T) {
		slice := []int{1, 2, 3}
		isSlice := IsArray(slice)
		assert.True(t, isSlice)
	})

	t.Run("IsArray", func(t *testing.T) {
		arr := [3]int{1, 2, 3}
		isArr := IsArray(arr)
		assert.True(t, isArr)
	})

	t.Run("TryString", func(t *testing.T) {
		val := "hello"
		is := IsArray(val)
		assert.False(t, is)
	})

	t.Run("TryBool", func(t *testing.T) {
		bval := true
		bis := IsArray(bval)
		assert.False(t, bis)
	})

	t.Run("TryNumber", func(t *testing.T) {
		nval := 99.99
		nis := IsArray(nval)
		assert.False(t, nis)
	})

	t.Run("TryStruct", func(t *testing.T) {
		type TestObject struct {
			prop1 string
			prop2 int
		}
		obj := &TestObject{"hello", 99}
		sis := IsArray(*obj)
		assert.False(t, sis)
	})
}

func TestGetTerraformOutputs(t *testing.T) {
	t.Parallel()
	t.Run("All outputs exist", func(t *testing.T) {
		// Generate a map of static key-value pairs for testing purposes.
		outputs := map[string]interface{}{
			"test1": "1234",
			"test2": "5678",
			"test3": "91011",
			"test4": []string{"12", "123"},
		}

		// Extract a slice of the keys for use in the test.
		outputKeys := []string{"test1", "test2", "test3", "test4"}

		foundValues, missingKeys := GetTerraformOutputs(t, outputs, outputKeys...)
		assert.Equal(t, len(outputKeys), len(foundValues))
		assert.Empty(t, missingKeys)
	})

	t.Run("All outputs exist, but with nil value", func(t *testing.T) {
		// Generate a map of static key-value pairs for testing purposes.
		outputs := map[string]interface{}{
			"test1": "1234",
			"test2": "5678",
			"test3": "91011",
			"test4": nil,
		}

		// Extract a slice of the keys for use in the test.
		outputKeys := []string{"test1", "test2", "test3", "test4"}

		failedT := new(testing.T)
		foundValues, missingKeys := GetTerraformOutputs(failedT, outputs, outputKeys...)
		assert.True(t, failedT.Failed(), "test4 should have caused an error because it is nil")

		assert.Contains(t, missingKeys, "test4")
		assert.Equal(t, len(outputKeys)-1, len(foundValues))
	})

	t.Run("Not all outputs exist", func(t *testing.T) {
		// Generate a map of static key-value pairs for testing purposes.
		outputs := map[string]interface{}{
			"test1": "1234",
			"test2": "5678",
			"test3": "91011",
		}

		// Extract a slice of the keys for use in the test.
		outputKeys := []string{"test1", "test2", "test3", "test4"}

		failedT := new(testing.T)
		foundValues, missingKeys := GetTerraformOutputs(failedT, outputs, outputKeys...)
		assert.True(t, failedT.Failed(), "test4 should have caused an error because it is missing")

		assert.Equal(t, len(foundValues), len(outputKeys)-1)
		assert.Contains(t, missingKeys, "test4")
		assert.NotContains(t, missingKeys, "test1")
		assert.NotContains(t, missingKeys, "test2")
		assert.NotContains(t, missingKeys, "test3")
	})
}
