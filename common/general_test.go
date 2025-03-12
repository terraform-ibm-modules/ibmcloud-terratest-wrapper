package common

import (
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

	result, err := GetBeforeAfterDiff(jsonString)
	assert.Equal(t, expected, result)
	assert.Nil(t, err)
}

func TestGetBeforeAfterDiffMissingBeforeKey(t *testing.T) {
	jsonString := `{"after": {"a": 1, "b": 2}}`
	expectedErr := "missing 'before' or 'after' key in JSON"
	result, err := GetBeforeAfterDiff(jsonString)
	assert.Equal(t, "", result)
	assert.EqualError(t, err, expectedErr)
}

func TestGetBeforeAfterDiffNonObjectBeforeValue(t *testing.T) {
	jsonString := `{"before": ["a", "b"], "after": {"a": 1, "b": 2}}`
	expectedErr := "'before' value is not an object"
	result, err := GetBeforeAfterDiff(jsonString)
	assert.Equal(t, "", result)
	assert.EqualError(t, err, expectedErr)
}

func TestGetBeforeAfterDiffNonObjectAfterValue(t *testing.T) {
	jsonString := `{"before": {"a": 1, "b": 2}, "after": ["a", "b"]}`
	expectedErr := "'after' value is not an object"
	result, err := GetBeforeAfterDiff(jsonString)
	assert.Equal(t, "", result)
	assert.EqualError(t, err, expectedErr)
}

func TestGetBeforeAfterDiffInvalidJSON(t *testing.T) {
	jsonString := `{"before": {"a": 1, "b": 2}, "after": {"a": 1, "b": 2}`
	expectedErr := "unable to parse JSON string"
	result, err := GetBeforeAfterDiff(jsonString)
	assert.Equal(t, "", result)
	assert.EqualError(t, err, expectedErr)
}

func TestConvertArrayJson(t *testing.T) {

	t.Run("GoodArray", func(t *testing.T) {
		goodArr := []interface{}{
			"hello",
			true,
			99,
			"bye",
		}
		goodStr, goodErr := ConvertValueToJsonString(goodArr)
		if assert.NoError(t, goodErr, "error converting array") {
			assert.NotEmpty(t, goodStr)
		}
	})

	t.Run("NilValue", func(t *testing.T) {
		nullVal, nullErr := ConvertValueToJsonString(nil)
		if assert.NoError(t, nullErr) {
			assert.Equal(t, "null", nullVal)
		}
	})

	t.Run("NullPointer", func(t *testing.T) {
		var intPtr *int
		ptrVal, ptrErr := ConvertValueToJsonString(intPtr)
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

func TestIsComposite(t *testing.T) {

	t.Run("IsSlice", func(t *testing.T) {
		slice := []int{1, 2, 3}
		isSlice := IsCompositeType(slice)
		assert.True(t, isSlice)
	})

	t.Run("IsArray", func(t *testing.T) {
		arr := [3]int{1, 2, 3}
		isArr := IsCompositeType(arr)
		assert.True(t, isArr)
	})

	t.Run("TryString", func(t *testing.T) {
		val := "hello"
		is := IsCompositeType(val)
		assert.False(t, is)
	})

	t.Run("TryBool", func(t *testing.T) {
		bval := true
		bis := IsCompositeType(bval)
		assert.False(t, bis)
	})

	t.Run("TryNumber", func(t *testing.T) {
		nval := 99.99
		nis := IsCompositeType(nval)
		assert.False(t, nis)
	})

	t.Run("TryStruct", func(t *testing.T) {
		type TestObject struct {
			prop1 string
			prop2 int
		}
		obj := &TestObject{"hello", 99}
		sis := IsCompositeType(*obj)
		assert.True(t, sis)
	})

	t.Run("TryMap", func(t *testing.T) {
		mapVal := map[string]string{"one": "1", "two": "2"}
		sis := IsCompositeType(mapVal)
		assert.True(t, sis)
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

func TestCopyDirectory(t *testing.T) {

	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	// Clean up: remove temporary directory
	defer os.RemoveAll(tmpDir)
	// Create a source directory with some files and permissions
	sourceDir := filepath.Join(tmpDir, "source")
	if err := os.Mkdir(sourceDir, 0755); err != nil {
		t.Fatal(err)
	}
	sourceFileTxt := filepath.Join(sourceDir, "file.txt")
	if _, err := os.Create(sourceFileTxt); err != nil {
		t.Fatal(err)
	}
	// Set permissions on the source file txt
	if err := os.Chmod(sourceFileTxt, 0755); err != nil {
		t.Fatal(err)
	}

	sourceFileTf := filepath.Join(sourceDir, "file.tf")
	if _, err := os.Create(sourceFileTf); err != nil {
		t.Fatal(err)
	}
	// Set permissions on the source file tf
	if err := os.Chmod(sourceFileTf, 0755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		Name                      string
		Source                    string
		Destination               string
		FileFilter                func(string) bool
		ExpectedError             string
		ExpectedFileInDestination int
	}{
		{
			Name:                      "Copy Directory",
			Source:                    sourceDir,
			Destination:               "testdata/destination",
			FileFilter:                nil,
			ExpectedError:             "",
			ExpectedFileInDestination: 2,
		},
		{
			Name:        "Copy Directory with Filter",
			Source:      sourceDir,
			Destination: "testdata/destination",
			FileFilter: func(path string) bool {
				return filepath.Ext(path) == ".tf"
			},
			ExpectedError:             "",
			ExpectedFileInDestination: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			// cleanup destination directory
			defer os.RemoveAll(test.Destination)

			err := CopyDirectory(test.Source, test.Destination, test.FileFilter)
			if test.ExpectedError != "" {
				assert.EqualError(t, err, test.ExpectedError)
			} else {
				assert.NoError(t, err)
			}
			files, err := os.ReadDir(test.Destination)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, test.ExpectedFileInDestination, len(files))
		})
	}
}

func TestCopyFile(t *testing.T) {

	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	// Clean up: remove temporary directory
	defer os.RemoveAll(tmpDir)
	// Create a source directory with some files and permissions
	sourceDir := filepath.Join(tmpDir, "source")
	if err := os.Mkdir(sourceDir, 0755); err != nil {
		t.Fatal(err)
	}
	sourceFileTxt := filepath.Join(sourceDir, "file.txt")
	if _, err := os.Create(sourceFileTxt); err != nil {
		t.Fatal(err)
	}
	// create destination directory
	destinationDir := filepath.Join(tmpDir, "destination")
	if err := os.Mkdir(destinationDir, 0755); err != nil {
		t.Fatal(err)
	}
	// cleanup destination directory
	defer os.RemoveAll(destinationDir)

	tests := []struct {
		Name          string
		Source        string
		Destination   string
		ExpectedError string
	}{
		{
			Name:          "Copy File",
			Source:        sourceFileTxt,
			Destination:   filepath.Join(destinationDir, "file.txt"),
			ExpectedError: "",
		},
		{
			Name:          "Copy File with Invalid Source",
			Source:        "testdata/source/invalid.txt",
			Destination:   "testdata/destination/file.txt",
			ExpectedError: "source path testdata/source/invalid.txt does not exist: stat testdata/source/invalid.txt: no such file or directory",
		},
		{
			Name:          "Copy File with Invalid Destination",
			Source:        sourceFileTxt,
			Destination:   "testdata/destination/invalid/file.txt",
			ExpectedError: "failed to create destination file: open testdata/destination/invalid/file.txt: no such file or directory",
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			// cleanup destination directory
			defer os.RemoveAll(test.Destination)

			err := CopyFile(test.Source, test.Destination)
			if test.ExpectedError != "" {
				assert.EqualError(t, err, test.ExpectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
func TestStringContainsIgnoreCase(t *testing.T) {
	tests := []struct {
		Name     string
		Input    string
		Contains string
		Expected bool
	}{
		{
			Name:     "Empty String",
			Input:    "",
			Contains: "test",
			Expected: false,
		},
		{
			Name:     "Contains",
			Input:    "test",
			Contains: "es",
			Expected: true,
		},
		{
			Name:     "Does Not Contain",
			Input:    "test",
			Contains: "nope",
			Expected: false,
		},
		{
			Name:     "Contains Upper Case",
			Input:    "test",
			Contains: "ES",
			Expected: true,
		},
		{
			Name:     "Contains Mixed Case",
			Input:    "test",
			Contains: "Es",
			Expected: true,
		},
		{
			Name:     "Contains Lower Case",
			Input:    "TEST",
			Contains: "es",
			Expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			result := StringContainsIgnoreCase(test.Input, test.Contains)
			assert.Equal(t, test.Expected, result)
		})
	}
}
