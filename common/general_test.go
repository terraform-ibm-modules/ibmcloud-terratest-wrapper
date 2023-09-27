package common

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
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

// / TestLoadMapFromYaml tests the LoadMapFromYaml function.
func TestLoadMapFromYaml(t *testing.T) {
	yamlData := `
        name: John
        age: 30
        isMarried: true
        hobbies:
          - reading
          - running
          - swimming
        address:
          street: 123 Main St.
          city: Anytown
          state: CA
          zip: "12345"
    `
	expectedOutput := map[string]interface{}{
		"name":      "John",
		"age":       30,
		"isMarried": true,
		"hobbies": []interface{}{
			"reading",
			"running",
			"swimming",
		},
		"address": map[string]interface{}{
			"street": "123 Main St.",
			"city":   "Anytown",
			"state":  "CA",
			"zip":    "12345",
		},
	}

	t.Run("valid yaml", func(t *testing.T) {
		tempFile, err := os.CreateTemp("", "example.yaml")
		if assert.Nilf(t, err, "Failed to create temporary file: %v", err) {

			defer os.Remove(tempFile.Name())

			_, err = tempFile.WriteString(yamlData)

			if assert.Nilf(t, err, "Failed to write YAML data to file: %v", err) {
				output, err := LoadMapFromYaml(tempFile.Name())
				assert.Nilf(t, err, "Unexpected error: %v", err)
				assert.Truef(t, reflect.DeepEqual(output, expectedOutput), "Unexpected output. Got %v, expected %v", output, expectedOutput)
			}
		}
	})

	t.Run("nonexistent file", func(t *testing.T) {
		_, err := LoadMapFromYaml("nonexistent.yaml")
		assert.Errorf(t, err, "Unexpected error. Got %v, expected %v", err, os.ErrNotExist)
	})

	t.Run("invalid yaml", func(t *testing.T) {
		tempFile, err := os.CreateTemp("", "example.yaml")
		if assert.Nilf(t, err, "Failed to create temporary file: %v", err) {
			defer os.Remove(tempFile.Name())

			_, err = tempFile.WriteString("invalid yaml data")
			if err != nil {
				t.Fatalf("Failed to write YAML data to file: %v", err)
			}

			_, err = LoadMapFromYaml(tempFile.Name())
			if err == nil {
				t.Error("Expected an error, but got none")
			}
		}
	})
}

func TestGenerateSshPublicKey(t *testing.T) {
	newKey, err := GenerateSshRsaPublicKey()
	assert.NoErrorf(t, err, "Failed to create key: %v", err)
	if assert.NotEmpty(t, newKey) {
		// make sure there are no newlines
		assert.NotContains(t, newKey, "\n")
	}
}

func TestCopyDirectoryAndFile(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	// Create a source directory with some files and permissions
	sourceDir := filepath.Join(tmpDir, "source")
	if err := os.Mkdir(sourceDir, 0755); err != nil {
		t.Fatal(err)
	}
	sourceFile := filepath.Join(sourceDir, "file.txt")
	if _, err := os.Create(sourceFile); err != nil {
		t.Fatal(err)
	}
	// Set permissions on the source file
	if err := os.Chmod(sourceFile, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a destination directory
	destDir := filepath.Join(tmpDir, "destination")

	// Log permissions before copying
	srcFileInfo, err := os.Stat(sourceFile)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("Source File Permissions: %v\n", srcFileInfo.Mode())

	// Copy the source directory to the destination
	if err := CopyDirectory(sourceDir, destDir); err != nil {
		t.Fatal(err)
	}

	// Check if the destination directory and file exist
	_, err = os.Stat(destDir)
	if os.IsNotExist(err) {
		t.Fatal("Destination directory does not exist")
	}

	destFile := filepath.Join(destDir, "file.txt")
	_, err = os.Stat(destFile)
	if os.IsNotExist(err) {
		t.Fatal("Destination file does not exist")
	}

	// Log permissions after copying
	destFileInfo, err := os.Stat(destFile)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("Destination File Permissions: %v\n", destFileInfo.Mode())

	// Check if permissions are preserved
	if srcFileInfo.Mode() != destFileInfo.Mode() {
		t.Fatalf("File permissions are not preserved. Expected: %v, Got: %v", srcFileInfo.Mode(), destFileInfo.Mode())
	}

	// Clean up: remove temporary directory
	os.RemoveAll(tmpDir)
}
