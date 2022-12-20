package testschematic

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
