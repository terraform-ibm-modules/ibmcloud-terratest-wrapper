package testschematic

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTarCreation(t *testing.T) {

	goodPattern := &[]string{"*_test.go"}

	// good file
	t.Run("GoodTarFile", func(t *testing.T) {
		goodTarFile, goodTarErr := CreateSchematicTar(".", goodPattern)
		if assert.NoError(t, goodTarErr) {
			if assert.NotEmpty(t, goodTarFile) {
				defer os.Remove(goodTarFile)
				info, infoErr := os.Stat(goodTarFile)
				if assert.NoError(t, infoErr) {
					assert.Greater(t, info.Size(), int64(0), "file cannot be empty")
				}
			}
		}
	})

	// bad starting path errors
	t.Run("BadRootPath", func(t *testing.T) {
		_, badRootErr := CreateSchematicTar("/blah_blah_dummy_blah", goodPattern)
		assert.Error(t, badRootErr)
	})

	// include filter that results in empty tar file, which is an error
	t.Run("EmptyFile", func(t *testing.T) {
		emptyPattern := &[]string{"*.foobar"}
		_, emptyFileErr := CreateSchematicTar(".", emptyPattern)
		assert.Error(t, emptyFileErr)
	})
}
