package testprojects

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCorrectResourceTeardownFlag(t *testing.T) {

	// Test success and no skips
	t.Run("SuccessNoSkip", func(t *testing.T) {
		o := TestProjectsOptions{
			Testing: new(testing.T),
		}
		assert.Equal(t, true, o.executeResourceTearDown())
	})

	t.Run("SuccessWithSkip", func(t *testing.T) {
		o := TestProjectsOptions{
			Testing:           new(testing.T),
			SkipUndeploy:      true,
			SkipProjectDelete: false,
		}
		assert.Equal(t, false, o.executeResourceTearDown())
	})

	t.Run("FailNoSkip", func(t *testing.T) {
		o := TestProjectsOptions{
			Testing:           new(testing.T),
			SkipUndeploy:      false,
			SkipProjectDelete: false,
		}
		o.Testing.Fail()
		assert.Equal(t, true, o.executeResourceTearDown())
	})

	t.Run("FailWithSkip", func(t *testing.T) {
		o := TestProjectsOptions{
			Testing:           new(testing.T),
			SkipUndeploy:      true,
			SkipProjectDelete: false,
		}
		o.Testing.Fail()
		assert.Equal(t, false, o.executeResourceTearDown())
	})

	t.Run("FailNoSkipWithIgnore", func(t *testing.T) {
		o := TestProjectsOptions{
			Testing:           new(testing.T),
			SkipUndeploy:      false,
			SkipProjectDelete: false,
		}
		os.Setenv("DO_NOT_DESTROY_ON_FAILURE", "true")
		o.Testing.Fail()
		assert.Equal(t, false, o.executeResourceTearDown())
		os.Unsetenv("DO_NOT_DESTROY_ON_FAILURE")
	})

	t.Run("FailNoSkipWithIgnoreOff", func(t *testing.T) {
		o := TestProjectsOptions{
			Testing:           new(testing.T),
			SkipUndeploy:      false,
			SkipProjectDelete: false,
		}
		os.Setenv("DO_NOT_DESTROY_ON_FAILURE", "false")
		o.Testing.Fail()
		assert.Equal(t, true, o.executeResourceTearDown())
		os.Unsetenv("DO_NOT_DESTROY_ON_FAILURE")
	})

	t.Run("FailWithSkipWithIgnore", func(t *testing.T) {
		o := TestProjectsOptions{
			Testing:           new(testing.T),
			SkipUndeploy:      false,
			SkipProjectDelete: false,
		}
		os.Setenv("DO_NOT_DESTROY_ON_FAILURE", "true")
		o.Testing.Fail()
		assert.Equal(t, false, o.executeResourceTearDown())
		os.Unsetenv("DO_NOT_DESTROY_ON_FAILURE")
	})
}

func TestCorrectProjectTeardownFlag(t *testing.T) {

	t.Run("SuccessNoSkip", func(t *testing.T) {
		o := TestProjectsOptions{
			Testing: new(testing.T),
		}
		assert.Equal(t, true, o.executeProjectTearDown())
	})

	t.Run("SuccessWithSkip", func(t *testing.T) {
		o := TestProjectsOptions{
			Testing:           new(testing.T),
			SkipUndeploy:      false,
			SkipProjectDelete: true,
		}
		assert.Equal(t, false, o.executeProjectTearDown())
	})

	t.Run("FailNoSkip", func(t *testing.T) {
		o := TestProjectsOptions{
			Testing:           new(testing.T),
			SkipUndeploy:      false,
			SkipProjectDelete: false,
		}
		o.Testing.Fail()
		assert.Equal(t, false, o.executeProjectTearDown())
	})

	t.Run("FailWithSkip", func(t *testing.T) {
		o := TestProjectsOptions{
			Testing:           new(testing.T),
			SkipUndeploy:      true,
			SkipProjectDelete: false,
		}
		o.Testing.Fail()
		assert.Equal(t, false, o.executeProjectTearDown())
	})
}
