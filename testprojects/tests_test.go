package testprojects

import (
	"errors"
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
			Logger:             common.NewTestLogger(t.Name()),
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

// mockStackCloudInfoService lets a test drive the return values of
// CreateStackFromConfigFile without reaching the Projects API.
type mockStackCloudInfoService struct {
	cloudinfo.CloudInfoServiceI
	stack *projectv1.StackDefinition
	resp  *core.DetailedResponse
	err   error
}

func (m *mockStackCloudInfoService) CreateStackFromConfigFile(stackConfig *cloudinfo.ConfigDetails, stackConfigPath string, catalogJsonPath string) (*projectv1.StackDefinition, *core.DetailedResponse, error) {
	return m.stack, m.resp, m.err
}

// TestConfigureTestStackReturnsError covers the case where the Projects API fails with an
// error that is not the "stack definition member input" case. That error used to be
// swallowed, so ConfigureTestStack returned nil and the caller dereferenced a nil
// currentStack, panicking with a message that hid the real failure.
func TestConfigureTestStackReturnsError(t *testing.T) {
	newOptions := func(svc cloudinfo.CloudInfoServiceI) *TestProjectsOptions {
		return &TestProjectsOptions{
			Testing:          new(testing.T), // swallow the internal assertions
			Logger:           common.NewTestLogger(t.Name()),
			CloudInfoService: svc,
			currentProject:   &projectv1.Project{ID: core.StringPtr(mockProjectID)},
		}
	}

	t.Run("SDKProblemIsReturnedNotSwallowed", func(t *testing.T) {
		apiErr := core.SDKErrorf(nil, "The config cannot be found", "http-request-err",
			core.NewProblemComponent("project", "v1"))
		o := newOptions(&mockStackCloudInfoService{err: apiErr})

		err := o.ConfigureTestStack()

		assert.Error(t, err, "an API error must be reported, not swallowed")
		assert.Contains(t, err.Error(), "The config cannot be found")
	})

	t.Run("NonSDKErrorIsReturned", func(t *testing.T) {
		o := newOptions(&mockStackCloudInfoService{err: errors.New("some transport failure")})

		err := o.ConfigureTestStack()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "some transport failure")
	})

	t.Run("MissingConfigurationIDIsErrorNotPanic", func(t *testing.T) {
		// 201 with a body that carries no configuration ID: the caller dereferences
		// currentStack.Configuration.ID, so this must be reported as an error.
		o := newOptions(&mockStackCloudInfoService{
			stack: &projectv1.StackDefinition{},
			resp:  &core.DetailedResponse{StatusCode: 201},
		})

		assert.NotPanics(t, func() {
			err := o.ConfigureTestStack()
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "stack configuration ID")
		})
	})

	t.Run("NonCreatedStatusCodeIsReturned", func(t *testing.T) {
		o := newOptions(&mockStackCloudInfoService{
			stack: &projectv1.StackDefinition{},
			resp:  &core.DetailedResponse{StatusCode: 400},
		})

		err := o.ConfigureTestStack()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "400")
	})

	t.Run("SuccessReturnsNil", func(t *testing.T) {
		o := newOptions(&mockStackCloudInfoService{
			stack: &projectv1.StackDefinition{
				Configuration: &projectv1.StackDefinitionMetadataConfiguration{
					ID: core.StringPtr("stack-config-id"),
				},
			},
			resp: &core.DetailedResponse{StatusCode: 201},
		})

		assert.NoError(t, o.ConfigureTestStack())
	})
}
