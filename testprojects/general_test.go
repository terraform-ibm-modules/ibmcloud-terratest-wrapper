package testprojects

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

const ValidStackDefinition = "testdata/stack_definition_valid.json"

func TestGetVersionLocatorFromStackDefinitionForMemberName(t *testing.T) {
	dir, _ := os.Getwd()
	ValidStackDefinitionPath := filepath.Join(dir, ValidStackDefinition)

	tests := []struct {
		Name          string
		Path          string
		MemberName    string
		ExpectedError error
		Expected      string
	}{
		{
			Name:       "ValidMemberName",
			Path:       ValidStackDefinitionPath,
			MemberName: "primary-da",
			Expected:   "7df1e4ca-d54c-4fd0-82ce-3d13247308cd.a8887a40-ff3f-4ee8-bcfe-bcfd55360074",
		},
		{
			Name:          "InvalidMemberName",
			Path:          ValidStackDefinitionPath,
			MemberName:    "invalidMemberName",
			ExpectedError: fmt.Errorf("member not found"),
			Expected:      "",
		},
		{
			Name:          "EmptyMemberName",
			Path:          ValidStackDefinitionPath,
			MemberName:    "",
			ExpectedError: fmt.Errorf("member not found"),
			Expected:      "",
		},
		{
			Name:          "InvalidPath",
			Path:          "invalidPath",
			MemberName:    "primary-da",
			ExpectedError: fmt.Errorf("open invalidPath: no such file or directory"),
			Expected:      "",
		},
		{
			Name:          "EmptyPath",
			Path:          "",
			MemberName:    "primary-da",
			ExpectedError: fmt.Errorf("open : no such file or directory"),
			Expected:      "",
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			locator, err := GetVersionLocatorFromStackDefinitionForMemberName(test.Path, test.MemberName)
			if test.ExpectedError != nil {
				assert.Error(t, err)
				if os.IsNotExist(err) {
					assert.Equal(t, test.ExpectedError.Error(), err.Error())
				} else {
					assert.Equal(t, test.ExpectedError, err)
				}
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, test.Expected, locator)
		})
	}
}

func TestProjectOptionsDefaultAuthDefaultsToApiKey(t *testing.T) {
	t.Setenv(ibmcloudApiKeyVar, "test-api-key")

	options := TestProjectOptionsDefault(&TestProjectsOptions{
		Testing: t,
		Prefix:  "test",
	})

	if assert.NotNil(t, options.StackAuthorizations) {
		assert.Equal(t, "api_key", *options.StackAuthorizations.Method)
		assert.Equal(t, "test-api-key", *options.StackAuthorizations.ApiKey)
		assert.Nil(t, options.StackAuthorizations.TrustedProfileID)
	}
}

func TestProjectOptionsDefaultAuthUsesTrustedProfileField(t *testing.T) {
	t.Setenv(ibmcloudApiKeyVar, "test-api-key")

	options := TestProjectOptionsDefault(&TestProjectsOptions{
		Testing:          t,
		Prefix:           "test",
		TrustedProfileID: "profile-123",
	})

	if assert.NotNil(t, options.StackAuthorizations) {
		assert.Equal(t, "trusted_profile", *options.StackAuthorizations.Method)
		assert.Equal(t, "profile-123", *options.StackAuthorizations.TrustedProfileID)
		assert.Nil(t, options.StackAuthorizations.ApiKey)
	}
}

func TestProjectOptionsDefaultAuthUsesTrustedProfileEnvVar(t *testing.T) {
	t.Setenv(ibmcloudApiKeyVar, "test-api-key")
	t.Setenv(trustedProfileIDVar, "env-profile-456")

	options := TestProjectOptionsDefault(&TestProjectsOptions{
		Testing: t,
		Prefix:  "test",
	})

	if assert.NotNil(t, options.StackAuthorizations) {
		assert.Equal(t, "trusted_profile", *options.StackAuthorizations.Method)
		assert.Equal(t, "env-profile-456", *options.StackAuthorizations.TrustedProfileID)
		assert.Nil(t, options.StackAuthorizations.ApiKey)
	}
}

func TestProjectOptionsDefaultAuthFieldTakesPrecedenceOverEnvVar(t *testing.T) {
	t.Setenv(ibmcloudApiKeyVar, "test-api-key")
	t.Setenv(trustedProfileIDVar, "env-profile-456")

	options := TestProjectOptionsDefault(&TestProjectsOptions{
		Testing:          t,
		Prefix:           "test",
		TrustedProfileID: "field-profile-123",
	})

	if assert.NotNil(t, options.StackAuthorizations) {
		assert.Equal(t, "trusted_profile", *options.StackAuthorizations.Method)
		assert.Equal(t, "field-profile-123", *options.StackAuthorizations.TrustedProfileID)
	}
}
