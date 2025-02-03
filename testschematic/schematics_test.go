package testschematic

import (
	"os"
	"testing"
	"time"

	"github.com/IBM/go-sdk-core/v5/core"
	schematics "github.com/IBM/schematics-go-sdk/schematicsv1"
	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/strfmt/conv"
	"github.com/stretchr/testify/assert"
)

func TestSchematicTarCreation(t *testing.T) {

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

func TestSchematicCreateWorkspace(t *testing.T) {
	zero := 0
	schematicSvc := new(schematicServiceMock)
	svc := &SchematicsTestService{
		SchematicsApiSvc: schematicSvc,
		TestOptions: &TestSchematicOptions{
			Testing: new(testing.T),
			RequiredEnvironmentVars: map[string]string{
				"GIT_TOKEN_USER": "tester",   // pragma: allowlist secret
				"GIT_TOKEN":      "99999999", // pragma: allowlist secret
			},
			SchematicSvcRetryCount:       &zero,
			SchematicSvcRetryWaitSeconds: &zero,
		},
	}
	mockErrorType := new(schematicErrorMock)

	t.Run("WorkspaceCreated", func(t *testing.T) {
		result, err := svc.CreateTestWorkspace("good", "any-rg", "us-south", ".", "terraform_v1.2", []string{"tag1", "tag2"})
		if assert.NoError(t, err) {
			assert.Equal(t, mockWorkspaceID, *result.ID)
		}
	})

	t.Run("WorkspaceCreatedEmptyDefaults", func(t *testing.T) {
		result, err := svc.CreateTestWorkspace("good", "any-rg", "", "", "", []string{"tag1", "tag2"})
		if assert.NoError(t, err) {
			assert.Equal(t, mockWorkspaceID, *result.ID)
		}
	})

	t.Run("ExternalServiceError", func(t *testing.T) {
		schematicSvc.failCreateWorkspace = true
		_, err := svc.CreateTestWorkspace("error", "any-rg", "us-south", ".", "terraform_v1.2", []string{"tag1"})
		assert.ErrorAs(t, err, &mockErrorType)
	})
}

func TestSchematicUpdateWorkspace(t *testing.T) {
	zero := 0
	schematicSvc := new(schematicServiceMock)
	svc := &SchematicsTestService{
		SchematicsApiSvc: schematicSvc,
		WorkspaceID:      mockWorkspaceID,
		TemplateID:       mockTemplateID,
		TestOptions: &TestSchematicOptions{
			Testing:                      new(testing.T),
			SchematicSvcRetryCount:       &zero,
			SchematicSvcRetryWaitSeconds: &zero,
		},
	}
	mockErrorType := new(schematicErrorMock)
	vars := []TestSchematicTerraformVar{
		{Name: "string1", Value: "hello", DataType: "string", Secure: false},
	}

	t.Run("UpdateSimpleVariables", func(t *testing.T) {
		err := svc.UpdateTestTemplateVars(vars)
		assert.NoError(t, err)
	})

	t.Run("UpdateEmptyVariables", func(t *testing.T) {
		err := svc.UpdateTestTemplateVars([]TestSchematicTerraformVar{})
		assert.NoError(t, err)
	})

	t.Run("ComplexVariables", func(t *testing.T) {
		complex := []TestSchematicTerraformVar{
			{Name: "string1", Value: "hello", DataType: "string", Secure: true},
			{Name: "bool1", Value: true, DataType: "bool"},
			{Name: "stringlist", Value: []string{"hello", "goodbye"}, DataType: "list(string)"},
			{Name: "number1", Value: 99.9, DataType: "number"},
			{Name: "map1", Value: map[string]interface{}{"name1": "value1", "name2": false, "name3": 88.8}, DataType: "map(any)"},
		}
		err := svc.UpdateTestTemplateVars(complex)
		assert.NoError(t, err)
	})

	t.Run("ErrorFromService", func(t *testing.T) {
		schematicSvc.failReplaceWorkspaceInputs = true
		err := svc.UpdateTestTemplateVars(vars)
		assert.ErrorAs(t, err, &mockErrorType)
	})
}

func TestUploadSchematicTarFile(t *testing.T) {
	zero := 0
	schematicSvc := new(schematicServiceMock)
	svc := &SchematicsTestService{
		SchematicsApiSvc: schematicSvc,
		WorkspaceID:      mockWorkspaceID,
		TemplateID:       mockTemplateID,
		TestOptions: &TestSchematicOptions{
			Testing:                      new(testing.T),
			SchematicSvcRetryCount:       &zero,
			SchematicSvcRetryWaitSeconds: &zero,
		},
	}
	pathError := new(os.PathError)
	mockErrorType := new(schematicErrorMock)

	t.Run("GoodFile", func(t *testing.T) {
		err := svc.UploadTarToWorkspace("./mock_test.go")
		assert.NoError(t, err)
	})

	t.Run("NoFileFound", func(t *testing.T) {
		err := svc.UploadTarToWorkspace("/dummy-this/dummy-that/i-dont-exist.tar")
		if assert.Error(t, err) {
			assert.ErrorAs(t, err, &pathError)
		}
	})

	t.Run("EmptyFilePath", func(t *testing.T) {
		err := svc.UploadTarToWorkspace("")
		if assert.Error(t, err) {
			assert.ErrorAs(t, err, &pathError)
		}
	})

	t.Run("ErrorFromService", func(t *testing.T) {
		schematicSvc.failTemplateRepoUpload = true
		err := svc.UploadTarToWorkspace("./mock_test.go")
		assert.ErrorAs(t, err, &mockErrorType)
	})
}

func TestSchematicCreatePlanJob(t *testing.T) {
	zero := 0
	schematicSvc := new(schematicServiceMock)
	authSvc := new(iamAuthenticatorMock)
	svc := &SchematicsTestService{
		SchematicsApiSvc: schematicSvc,
		ApiAuthenticator: authSvc,
		WorkspaceID:      mockWorkspaceID,
		TemplateID:       mockTemplateID,
		TestOptions: &TestSchematicOptions{
			Testing:                      new(testing.T),
			SchematicSvcRetryCount:       &zero,
			SchematicSvcRetryWaitSeconds: &zero,
		},
	}
	mockErrorType := new(schematicErrorMock)

	t.Run("CreateSuccess", func(t *testing.T) {
		result, err := svc.CreatePlanJob()
		if assert.NoError(t, err) {
			assert.Equal(t, mockPlanID, *result.Activityid)
		}
	})

	t.Run("ErrorFromService", func(t *testing.T) {
		schematicSvc.failPlanWorkspaceCommand = true
		_, err := svc.CreatePlanJob()
		assert.ErrorAs(t, err, &mockErrorType)
	})
}

func TestSchematicCreateApplyJob(t *testing.T) {
	zero := 0
	schematicSvc := new(schematicServiceMock)
	authSvc := new(iamAuthenticatorMock)
	svc := &SchematicsTestService{
		SchematicsApiSvc: schematicSvc,
		ApiAuthenticator: authSvc,
		WorkspaceID:      mockWorkspaceID,
		TemplateID:       mockTemplateID,
		TestOptions: &TestSchematicOptions{
			Testing:                      new(testing.T),
			SchematicSvcRetryCount:       &zero,
			SchematicSvcRetryWaitSeconds: &zero,
		},
	}
	mockErrorType := new(schematicErrorMock)

	t.Run("CreateSuccess", func(t *testing.T) {
		result, err := svc.CreateApplyJob()
		if assert.NoError(t, err) {
			assert.Equal(t, mockApplyID, *result.Activityid)
			assert.True(t, schematicSvc.applyComplete)
		}
	})

	t.Run("ErrorFromService", func(t *testing.T) {
		schematicSvc.failApplyWorkspaceCommand = true
		_, err := svc.CreateApplyJob()
		assert.ErrorAs(t, err, &mockErrorType)
	})
}

func TestSchematicCreateDestroyJob(t *testing.T) {
	zero := 0
	schematicSvc := new(schematicServiceMock)
	authSvc := new(iamAuthenticatorMock)
	svc := &SchematicsTestService{
		SchematicsApiSvc: schematicSvc,
		ApiAuthenticator: authSvc,
		WorkspaceID:      mockWorkspaceID,
		TemplateID:       mockTemplateID,
		TestOptions: &TestSchematicOptions{
			Testing:                      new(testing.T),
			SchematicSvcRetryCount:       &zero,
			SchematicSvcRetryWaitSeconds: &zero,
		},
	}
	mockErrorType := new(schematicErrorMock)

	t.Run("CreateSuccess", func(t *testing.T) {
		result, err := svc.CreateDestroyJob()
		if assert.NoError(t, err) {
			assert.Equal(t, mockDestroyID, *result.Activityid)
			assert.True(t, schematicSvc.destroyComplete)
		}
	})

	t.Run("ErrorFromService", func(t *testing.T) {
		schematicSvc.failDestroyWorkspaceCommand = true
		_, err := svc.CreateDestroyJob()
		assert.ErrorAs(t, err, &mockErrorType)
	})
}

func TestSchematicFindJob(t *testing.T) {
	zero := 0
	schematicSvc := new(schematicServiceMock)
	authSvc := new(iamAuthenticatorMock)
	svc := &SchematicsTestService{
		SchematicsApiSvc: schematicSvc,
		ApiAuthenticator: authSvc,
		WorkspaceID:      mockWorkspaceID,
		TemplateID:       mockTemplateID,
		TestOptions: &TestSchematicOptions{
			Testing:                      new(testing.T),
			SchematicSvcRetryCount:       &zero,
			SchematicSvcRetryWaitSeconds: &zero,
		},
	}
	mockErrorType := new(schematicErrorMock)
	notFoundErrorType := errors.NotFound("mock")

	t.Run("SingleResultFound", func(t *testing.T) {
		schematicSvc.activities = []schematics.WorkspaceActivity{
			{ActionID: core.StringPtr(mockActivityID), Name: core.StringPtr("TEST_ACTION"), PerformedAt: conv.DateTime(strfmt.DateTime(time.Now().Add(-time.Second * 1)))},
			{ActionID: core.StringPtr("not-the-answer"), Name: core.StringPtr("NOT_TEST_ACTION"), PerformedAt: conv.DateTime(strfmt.DateTime(time.Now().Add(-time.Second * 2)))},
		}
		result, err := svc.FindLatestWorkspaceJobByName("TEST_ACTION")
		if assert.NoError(t, err) {
			assert.Equal(t, mockActivityID, *result.ActionID)
		}
	})

	t.Run("LatestJobReturned", func(t *testing.T) {
		schematicSvc.activities = []schematics.WorkspaceActivity{
			{ActionID: core.StringPtr("also-not-answer"), Name: core.StringPtr("TEST_ACTION"), PerformedAt: conv.DateTime(strfmt.DateTime(time.Now().Add(-time.Second * 3)))},
			{ActionID: core.StringPtr(mockActivityID), Name: core.StringPtr("TEST_ACTION"), PerformedAt: conv.DateTime(strfmt.DateTime(time.Now().Add(-time.Second * 1)))},
			{ActionID: core.StringPtr("not-the-answer"), Name: core.StringPtr("TEST_ACTION"), PerformedAt: conv.DateTime(strfmt.DateTime(time.Now().Add(-time.Second * 2)))},
		}
		result, err := svc.FindLatestWorkspaceJobByName("TEST_ACTION")
		if assert.NoError(t, err) {
			assert.Equal(t, mockActivityID, *result.ActionID)
		}
	})

	t.Run("JobNotFound", func(t *testing.T) {
		_, err := svc.FindLatestWorkspaceJobByName("I_WILL_NOT_BE_FOUND")
		assert.ErrorAs(t, err, &notFoundErrorType)
	})

	t.Run("EmptyJobList", func(t *testing.T) {
		schematicSvc.emptyListWorkspaceActivities = true
		_, err := svc.FindLatestWorkspaceJobByName("TEST_ACTION")
		assert.ErrorAs(t, err, &notFoundErrorType)
		schematicSvc.emptyListWorkspaceActivities = false
	})

	t.Run("FailRetrievingJobs", func(t *testing.T) {
		schematicSvc.failListWorkspaceActivities = true
		_, err := svc.FindLatestWorkspaceJobByName("TEST_ACTION")
		assert.ErrorAs(t, err, &mockErrorType)
		schematicSvc.failListWorkspaceActivities = false
	})
}

func TestSchematicGetJobDetail(t *testing.T) {
	zero := 0
	schematicSvc := new(schematicServiceMock)
	authSvc := new(iamAuthenticatorMock)
	svc := &SchematicsTestService{
		SchematicsApiSvc: schematicSvc,
		ApiAuthenticator: authSvc,
		WorkspaceID:      mockWorkspaceID,
		TemplateID:       mockTemplateID,
		TestOptions: &TestSchematicOptions{
			Testing:                      new(testing.T),
			SchematicSvcRetryCount:       &zero,
			SchematicSvcRetryWaitSeconds: &zero,
		},
	}
	mockErrorType := new(schematicErrorMock)

	t.Run("JobFound", func(t *testing.T) {
		schematicSvc.activities = []schematics.WorkspaceActivity{
			{ActionID: core.StringPtr("also-not-answer"), Name: core.StringPtr("TEST_ACTION"), PerformedAt: conv.DateTime(strfmt.DateTime(time.Now().Add(-time.Second * 3)))},
			{ActionID: core.StringPtr(mockActivityID), Name: core.StringPtr("TEST_ACTION"), PerformedAt: conv.DateTime(strfmt.DateTime(time.Now().Add(-time.Second * 1)))},
			{ActionID: core.StringPtr("not-the-answer"), Name: core.StringPtr("TEST_ACTION"), PerformedAt: conv.DateTime(strfmt.DateTime(time.Now().Add(-time.Second * 2)))},
		}
		result, err := svc.GetWorkspaceJobDetail(mockActivityID)
		if assert.NoError(t, err) {
			assert.Equal(t, mockActivityID, *result.ActionID)
		}
	})

	t.Run("ServiceError", func(t *testing.T) {
		schematicSvc.failGetWorkspaceActivity = true
		_, err := svc.GetWorkspaceJobDetail(mockActivityID)
		assert.ErrorAs(t, err, &mockErrorType)
	})
}

func TestSchematicGetWorkspaceOutputs(t *testing.T) {
	zero := 0
	schematicSvc := new(schematicServiceMock)
	authSvc := new(iamAuthenticatorMock)
	svc := &SchematicsTestService{
		SchematicsApiSvc: schematicSvc,
		ApiAuthenticator: authSvc,
		WorkspaceID:      mockWorkspaceID,
		TemplateID:       mockTemplateID,
		TestOptions: &TestSchematicOptions{
			Testing:                      new(testing.T),
			SchematicSvcRetryCount:       &zero,
			SchematicSvcRetryWaitSeconds: &zero,
		},
	}
	mockErrorType := new(schematicErrorMock)

	t.Run("OutputsReturned", func(t *testing.T) {
		result, err := svc.GetLatestWorkspaceOutputs()
		if assert.NoError(t, err) {
			if assert.NotNil(t, result) {
				if assert.Len(t, result, 1) {
					assert.Equal(t, "the_mock_value", result["mock_output"])
				}
			}
		}
	})

	t.Run("ServiceError", func(t *testing.T) {
		schematicSvc.failGetOutputsCommand = true
		_, err := svc.GetLatestWorkspaceOutputs()
		assert.ErrorAs(t, err, &mockErrorType)
	})
}

func TestSchematicDeleteWorkspace(t *testing.T) {
	zero := 0
	schematicSvc := new(schematicServiceMock)
	authSvc := new(iamAuthenticatorMock)
	svc := &SchematicsTestService{
		SchematicsApiSvc: schematicSvc,
		ApiAuthenticator: authSvc,
		WorkspaceID:      mockWorkspaceID,
		TemplateID:       mockTemplateID,
		TestOptions: &TestSchematicOptions{
			Testing:                      new(testing.T),
			SchematicSvcRetryCount:       &zero,
			SchematicSvcRetryWaitSeconds: &zero,
		},
	}
	mockErrorType := new(schematicErrorMock)

	t.Run("DeleteComplete", func(t *testing.T) {
		result, err := svc.DeleteWorkspace()
		if assert.NoError(t, err) {
			assert.Equal(t, "deleted", result)
			assert.True(t, schematicSvc.workspaceDeleteComplete)
		}
	})

	t.Run("ServiceError", func(t *testing.T) {
		schematicSvc.failDeleteWorkspace = true
		_, err := svc.DeleteWorkspace()
		assert.ErrorAs(t, err, &mockErrorType)
	})
}

func TestSchematicWaitForJobFinish(t *testing.T) {
	zero := 0
	schematicSvc := new(schematicServiceMock)
	authSvc := new(iamAuthenticatorMock)
	svc := &SchematicsTestService{
		SchematicsApiSvc: schematicSvc,
		ApiAuthenticator: authSvc,
		WorkspaceID:      mockWorkspaceID,
		TemplateID:       mockTemplateID,
		TestOptions: &TestSchematicOptions{
			Testing:                      new(testing.T),
			SchematicSvcRetryCount:       &zero,
			SchematicSvcRetryWaitSeconds: &zero,
		},
	}
	mockErrorType := new(schematicErrorMock)

	t.Run("JobReadyNoWait", func(t *testing.T) {
		schematicSvc.activities = []schematics.WorkspaceActivity{
			{ActionID: core.StringPtr(mockActivityID), Name: core.StringPtr("TEST_ACTION"), Status: core.StringPtr(SchematicsJobStatusCompleted), PerformedAt: conv.DateTime(strfmt.DateTime(time.Now().Add(-time.Second * 1)))},
		}
		result, err := svc.WaitForFinalJobStatus(mockActivityID)
		if assert.NoError(t, err) {
			assert.Equal(t, SchematicsJobStatusCompleted, result)
		}
	})

	t.Run("ServiceError", func(t *testing.T) {
		schematicSvc.failGetWorkspaceActivity = true
		_, err := svc.WaitForFinalJobStatus(mockActivityID)
		assert.ErrorAs(t, err, &mockErrorType)
	})
}

func TestSchematicApiRetry(t *testing.T) {
	retry := 1
	svc := &SchematicsTestService{
		TestOptions: &TestSchematicOptions{
			SchematicSvcRetryCount:       &retry,
			SchematicSvcRetryWaitSeconds: &retry,
		},
	}

	testErr := errors.NotFound("not found")

	t.Run("NoErrorNoRetry", func(t *testing.T) {
		wasRetry := svc.retryApiCall(nil, 200, 0)
		assert.False(t, wasRetry)
	})

	t.Run("ErrorMaxRetries", func(t *testing.T) {
		wasRetry := svc.retryApiCall(testErr, 404, 1)
		assert.False(t, wasRetry)
	})

	t.Run("RetryOnError", func(t *testing.T) {
		wasRetry := svc.retryApiCall(testErr, 404, 0)
		assert.True(t, wasRetry)
	})

	t.Run("ErrorNoRetryException", func(t *testing.T) {
		wasRetry := svc.retryApiCall(testErr, getApiRetryStatusExceptions()[0], 0)
		assert.False(t, wasRetry)
	})

	t.Run("VerifyDefaultUsed", func(t *testing.T) {
		zero := 0
		svc.TestOptions = &TestSchematicOptions{
			SchematicSvcRetryWaitSeconds: &zero,
		}
		loops := 0
		for {
			wasRetry := svc.retryApiCall(testErr, 404, loops)
			if wasRetry {
				loops++
			} else {
				break
			}
		}
		assert.Equal(t, defaultApiRetryCount, loops)
	})

	t.Run("VerifyCanTurnOffRetry", func(t *testing.T) {
		zero := 0
		svc.TestOptions = &TestSchematicOptions{
			SchematicSvcRetryCount:       &zero,
			SchematicSvcRetryWaitSeconds: &zero,
		}
		wasRetry := svc.retryApiCall(testErr, 404, 0)
		assert.False(t, wasRetry)
	})
}
