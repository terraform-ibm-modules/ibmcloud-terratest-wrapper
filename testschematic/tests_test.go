package testschematic

import (
	"testing"
	"time"

	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/schematics-go-sdk/schematicsv1"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/strfmt/conv"
	"github.com/stretchr/testify/assert"
)

func TestSchematicFullTest(t *testing.T) {
	schematicSvc := new(schematicv1ServiceMock)
	svc := &SchematicsTestService{
		SchematicsApiSvc: schematicSvc,
		WorkspaceID:      mockWorkspaceID,
		TemplateID:       mockTemplateID,
		SchematicsIamToken: &core.IamTokenServerResponse{
			AccessToken:  "fake-token",
			RefreshToken: "fake-refresh-token",
		},
	}
	//mockErrorType := new(schematicv1ErrorMock)

	options := &TestSchematicOptions{
		Testing:                 new(testing.T),
		Prefix:                  "unit-test",
		DefaultRegion:           "test",
		Region:                  "test",
		RequiredEnvironmentVars: map[string]string{ibmcloudApiKeyVar: "XXX-XXXXXXX", gitUser: "some_git_user", gitToken: "fake_git_token"},
		TerraformVars: []TestSchematicTerraformVar{
			{Name: "var1", Value: "val1", DataType: "string", Secure: false},
			{Name: "var2", Value: "val2", DataType: "string", Secure: false},
		},
		Tags:                   []string{"unit-test"},
		TarIncludePatterns:     []string{"*.md"},
		WaitJobCompleteMinutes: 1,
		DeleteWorkspaceOnFail:  false,
		SchematicsApiSvc:       schematicSvc,
		schematicsTestSvc:      svc,
	}

	// mock at least one good tar upload and one other completed activity
	schematicSvc.activities = []schematicsv1.WorkspaceActivity{
		{ActionID: core.StringPtr(mockActivityID), Name: core.StringPtr(SchematicsJobTypeUpload), PerformedAt: conv.DateTime(strfmt.DateTime(time.Now().Add(-time.Second * 5))), Status: core.StringPtr(SchematicsJobStatusCompleted)},
		{ActionID: core.StringPtr(mockPlanID), Name: core.StringPtr("TEST-PLAN-JOB"), PerformedAt: conv.DateTime(strfmt.DateTime(time.Now().Add(-time.Second * 4))), Status: core.StringPtr(SchematicsJobStatusCompleted)},
		{ActionID: core.StringPtr(mockApplyID), Name: core.StringPtr("TEST-APPLY-JOB"), PerformedAt: conv.DateTime(strfmt.DateTime(time.Now().Add(-time.Second * 3))), Status: core.StringPtr(SchematicsJobStatusCompleted)},
		{ActionID: core.StringPtr(mockDestroyID), Name: core.StringPtr("TEST-DESTROY-JOB"), PerformedAt: conv.DateTime(strfmt.DateTime(time.Now().Add(-time.Second * 2))), Status: core.StringPtr(SchematicsJobStatusCompleted)},
	}

	t.Run("CleanRun", func(t *testing.T) {
		err := options.RunSchematicTest()
		assert.NoError(t, err)
		assert.True(t, schematicSvc.applyComplete)
		assert.True(t, schematicSvc.destroyComplete)
		assert.True(t, schematicSvc.workspaceDeleteComplete)
	})

	t.Run("WorkspaceCreateFail", func(t *testing.T) {
		mockSchematicv1ServiceReset(schematicSvc, options)
		options.DeleteWorkspaceOnFail = false // shouldn't matter
		schematicSvc.failCreateWorkspace = true
		options.RunSchematicTest()
		assert.False(t, schematicSvc.applyComplete)
		assert.False(t, schematicSvc.destroyComplete)
		assert.False(t, schematicSvc.workspaceDeleteComplete)
	})

	t.Run("WorkspaceSetupFail", func(t *testing.T) {
		mockSchematicv1ServiceReset(schematicSvc, options)
		schematicSvc.failReplaceWorkspaceInputs = true // after workspace create but before terraform
		options.DeleteWorkspaceOnFail = false          // shouldn't matter
		options.RunSchematicTest()
		assert.False(t, schematicSvc.applyComplete)
		assert.False(t, schematicSvc.destroyComplete)
		assert.True(t, schematicSvc.workspaceDeleteComplete) // delete workspace on fail if terraform isn't started
	})

	t.Run("PlanFailedLeaveWorkspace", func(t *testing.T) {
		mockSchematicv1ServiceReset(schematicSvc, options)
		schematicSvc.failPlanWorkspaceCommand = true
		options.DeleteWorkspaceOnFail = false // should leave workspace
		options.RunSchematicTest()
		assert.False(t, schematicSvc.applyComplete)
		assert.False(t, schematicSvc.destroyComplete)
		assert.False(t, schematicSvc.workspaceDeleteComplete)
	})

	t.Run("PlanFailedRemoveWorkspace", func(t *testing.T) {
		mockSchematicv1ServiceReset(schematicSvc, options)
		schematicSvc.failPlanWorkspaceCommand = true
		options.DeleteWorkspaceOnFail = true // should remove workspace
		options.RunSchematicTest()
		assert.False(t, schematicSvc.applyComplete)
		assert.False(t, schematicSvc.destroyComplete)
		assert.True(t, schematicSvc.workspaceDeleteComplete)
	})

	t.Run("ApplyCreateFailedRemoveWorkspace", func(t *testing.T) {
		mockSchematicv1ServiceReset(schematicSvc, options)
		schematicSvc.failApplyWorkspaceCommand = true
		options.DeleteWorkspaceOnFail = true // should remove workspace
		options.RunSchematicTest()
		assert.False(t, schematicSvc.applyComplete)
		assert.False(t, schematicSvc.destroyComplete)
		assert.True(t, schematicSvc.workspaceDeleteComplete)
	})

	t.Run("ApplyCreateFailedLeaveWorkspace", func(t *testing.T) {
		mockSchematicv1ServiceReset(schematicSvc, options)
		schematicSvc.failApplyWorkspaceCommand = true
		options.DeleteWorkspaceOnFail = false // should leave workspace
		options.RunSchematicTest()
		assert.False(t, schematicSvc.applyComplete)
		assert.False(t, schematicSvc.destroyComplete)
		assert.False(t, schematicSvc.workspaceDeleteComplete)
	})

	t.Run("DestroyCreateFailedLeaveWorkspace", func(t *testing.T) {
		mockSchematicv1ServiceReset(schematicSvc, options)
		schematicSvc.failDestroyWorkspaceCommand = true
		options.DeleteWorkspaceOnFail = false // should leave workspace
		options.RunSchematicTest()
		assert.True(t, schematicSvc.applyComplete)
		assert.False(t, schematicSvc.destroyComplete)
		assert.False(t, schematicSvc.workspaceDeleteComplete)
	})

	// set apply to failed
	schematicSvc.activities = []schematicsv1.WorkspaceActivity{
		{ActionID: core.StringPtr(mockActivityID), Name: core.StringPtr(SchematicsJobTypeUpload), PerformedAt: conv.DateTime(strfmt.DateTime(time.Now().Add(-time.Second * 5))), Status: core.StringPtr(SchematicsJobStatusCompleted)},
		{ActionID: core.StringPtr(mockPlanID), Name: core.StringPtr("TEST-PLAN-JOB"), PerformedAt: conv.DateTime(strfmt.DateTime(time.Now().Add(-time.Second * 4))), Status: core.StringPtr(SchematicsJobStatusCompleted)},
		{ActionID: core.StringPtr(mockApplyID), Name: core.StringPtr("TEST-APPLY-JOB"), PerformedAt: conv.DateTime(strfmt.DateTime(time.Now().Add(-time.Second * 3))), Status: core.StringPtr(SchematicsJobStatusFailed)},
		{ActionID: core.StringPtr(mockDestroyID), Name: core.StringPtr("TEST-DESTROY-JOB"), PerformedAt: conv.DateTime(strfmt.DateTime(time.Now().Add(-time.Second * 2))), Status: core.StringPtr(SchematicsJobStatusCompleted)},
	}

	t.Run("ApplyTerraformFailedLeaveWorkspace", func(t *testing.T) {
		mockSchematicv1ServiceReset(schematicSvc, options)
		options.DeleteWorkspaceOnFail = false
		options.RunSchematicTest()
		assert.True(t, schematicSvc.applyComplete)
		assert.True(t, schematicSvc.destroyComplete)
		assert.False(t, schematicSvc.workspaceDeleteComplete)
	})

	t.Run("ApplyTerraformFailedRemoveWorkspace", func(t *testing.T) {
		mockSchematicv1ServiceReset(schematicSvc, options)
		options.DeleteWorkspaceOnFail = true
		options.RunSchematicTest()
		assert.True(t, schematicSvc.applyComplete)
		assert.True(t, schematicSvc.destroyComplete)
		assert.True(t, schematicSvc.workspaceDeleteComplete)
	})

	// set destroy to failed
	schematicSvc.activities = []schematicsv1.WorkspaceActivity{
		{ActionID: core.StringPtr(mockActivityID), Name: core.StringPtr(SchematicsJobTypeUpload), PerformedAt: conv.DateTime(strfmt.DateTime(time.Now().Add(-time.Second * 5))), Status: core.StringPtr(SchematicsJobStatusCompleted)},
		{ActionID: core.StringPtr(mockPlanID), Name: core.StringPtr("TEST-PLAN-JOB"), PerformedAt: conv.DateTime(strfmt.DateTime(time.Now().Add(-time.Second * 4))), Status: core.StringPtr(SchematicsJobStatusCompleted)},
		{ActionID: core.StringPtr(mockApplyID), Name: core.StringPtr("TEST-APPLY-JOB"), PerformedAt: conv.DateTime(strfmt.DateTime(time.Now().Add(-time.Second * 3))), Status: core.StringPtr(SchematicsJobStatusCompleted)},
		{ActionID: core.StringPtr(mockDestroyID), Name: core.StringPtr("TEST-DESTROY-JOB"), PerformedAt: conv.DateTime(strfmt.DateTime(time.Now().Add(-time.Second * 2))), Status: core.StringPtr(SchematicsJobStatusFailed)},
	}

	t.Run("DestroyTerraformFailedLeaveWorkspace", func(t *testing.T) {
		mockSchematicv1ServiceReset(schematicSvc, options)
		options.DeleteWorkspaceOnFail = false
		options.RunSchematicTest()
		assert.True(t, schematicSvc.applyComplete)
		assert.True(t, schematicSvc.destroyComplete)
		assert.False(t, schematicSvc.workspaceDeleteComplete)
	})

	t.Run("DestroyTerraformFailedRemoveWorkspace", func(t *testing.T) {
		mockSchematicv1ServiceReset(schematicSvc, options)
		options.DeleteWorkspaceOnFail = true
		options.RunSchematicTest()
		assert.True(t, schematicSvc.applyComplete)
		assert.True(t, schematicSvc.destroyComplete)
		assert.True(t, schematicSvc.workspaceDeleteComplete)
	})
}
