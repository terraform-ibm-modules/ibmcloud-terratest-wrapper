package testprojects

import (
	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/project-go-sdk/projectv1"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/cloudinfo"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/common"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCorrectResourceTeardownFlag(t *testing.T) {

	// Test success and no skips
	t.Run("SuccessNoSkip", func(t *testing.T) {
		o := TestProjectsOptions{
			Testing:            new(testing.T),
			currentStackConfig: &cloudinfo.ConfigDetails{ConfigID: "1234"},
			Logger:             common.NewTestLogger(t.Name()),
		}
		assert.Equal(t, true, o.executeResourceTearDown())
	})

	t.Run("SuccessWithSkip", func(t *testing.T) {
		o := TestProjectsOptions{
			Testing:            new(testing.T),
			SkipUndeploy:       true,
			SkipProjectDelete:  false,
			currentStackConfig: &cloudinfo.ConfigDetails{ConfigID: "1234"},
			Logger:             common.NewTestLogger(t.Name()),
		}
		assert.Equal(t, false, o.executeResourceTearDown())
	})

	t.Run("SuccessNoConfig", func(t *testing.T) {
		o := TestProjectsOptions{
			Testing:            new(testing.T),
			SkipUndeploy:       false,
			SkipProjectDelete:  false,
			currentStackConfig: nil,
			Logger:             common.NewTestLogger(t.Name()),
		}
		assert.Equal(t, false, o.executeResourceTearDown())
	})

	t.Run("FailNoSkip", func(t *testing.T) {
		o := TestProjectsOptions{
			Testing:            new(testing.T),
			SkipUndeploy:       false,
			SkipProjectDelete:  false,
			currentStackConfig: &cloudinfo.ConfigDetails{ConfigID: "1234"},
			Logger:             &common.TestLogger{},
		}
		o.Testing.Fail()
		assert.Equal(t, true, o.executeResourceTearDown())
	})

	t.Run("FailWithSkip", func(t *testing.T) {
		o := TestProjectsOptions{
			Testing:            new(testing.T),
			SkipUndeploy:       true,
			SkipProjectDelete:  false,
			currentStackConfig: &cloudinfo.ConfigDetails{ConfigID: "1234"},
			Logger:             common.NewTestLogger(t.Name()),
		}
		o.Testing.Fail()
		assert.Equal(t, false, o.executeResourceTearDown())
	})

	t.Run("FailNoSkipWithIgnore", func(t *testing.T) {
		o := TestProjectsOptions{
			Testing:            new(testing.T),
			SkipUndeploy:       false,
			SkipProjectDelete:  false,
			currentStackConfig: &cloudinfo.ConfigDetails{ConfigID: "1234"},
			Logger:             common.NewTestLogger(t.Name()),
		}
		os.Setenv("DO_NOT_DESTROY_ON_FAILURE", "true")
		o.Testing.Fail()
		assert.Equal(t, false, o.executeResourceTearDown())
		os.Unsetenv("DO_NOT_DESTROY_ON_FAILURE")
	})

	t.Run("FailNoSkipWithIgnoreOff", func(t *testing.T) {
		o := TestProjectsOptions{
			Testing:            new(testing.T),
			SkipUndeploy:       false,
			SkipProjectDelete:  false,
			currentStackConfig: &cloudinfo.ConfigDetails{ConfigID: "1234"},
			Logger:             common.NewTestLogger(t.Name()),
		}
		os.Setenv("DO_NOT_DESTROY_ON_FAILURE", "false")
		o.Testing.Fail()
		assert.Equal(t, true, o.executeResourceTearDown())
		os.Unsetenv("DO_NOT_DESTROY_ON_FAILURE")
	})

	t.Run("FailWithSkipWithIgnore", func(t *testing.T) {
		o := TestProjectsOptions{
			Testing:            new(testing.T),
			SkipUndeploy:       false,
			SkipProjectDelete:  false,
			currentStackConfig: &cloudinfo.ConfigDetails{ConfigID: "1234"},
			Logger:             common.NewTestLogger(t.Name()),
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
			Testing:        new(testing.T),
			currentProject: &projectv1.Project{ID: core.StringPtr("1234")},
			Logger:         common.NewTestLogger(t.Name()),
		}
		assert.Equal(t, true, o.executeProjectTearDown())
	})

	t.Run("SuccessWithSkip", func(t *testing.T) {
		o := TestProjectsOptions{
			Testing:           new(testing.T),
			SkipUndeploy:      false,
			SkipProjectDelete: true,
			currentProject:    &projectv1.Project{ID: core.StringPtr("1234")},
			Logger:            common.NewTestLogger(t.Name()),
		}
		assert.Equal(t, false, o.executeProjectTearDown())
	})

	t.Run("SuccessNoProject", func(t *testing.T) {
		o := TestProjectsOptions{
			Testing:           new(testing.T),
			SkipUndeploy:      false,
			SkipProjectDelete: false,
			currentProject:    nil,
			Logger:            common.NewTestLogger(t.Name()),
		}
		assert.Equal(t, false, o.executeProjectTearDown())
	})

	t.Run("FailNoSkip", func(t *testing.T) {
		o := TestProjectsOptions{
			Testing:           new(testing.T),
			SkipUndeploy:      false,
			SkipProjectDelete: false,
			currentProject:    &projectv1.Project{ID: core.StringPtr("1234")},
			Logger:            common.NewTestLogger(t.Name()),
		}
		o.Testing.Fail()
		assert.Equal(t, false, o.executeProjectTearDown())
	})

	t.Run("FailWithSkip", func(t *testing.T) {
		o := TestProjectsOptions{
			Testing:           new(testing.T),
			SkipUndeploy:      true,
			SkipProjectDelete: false,
			currentProject:    &projectv1.Project{ID: core.StringPtr("1234")},
			Logger:            common.NewTestLogger(t.Name()),
		}
		o.Testing.Fail()
		assert.Equal(t, false, o.executeProjectTearDown())
	})
}
