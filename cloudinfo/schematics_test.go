package cloudinfo

import (
	"net/http"
	"testing"

	"github.com/IBM/go-sdk-core/v5/core"
	schematics "github.com/IBM/schematics-go-sdk/schematicsv1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock for schematicsService interface
type mockSchematicsService struct {
	mock.Mock
}

// Mock for IiamAuthenticator interface
type mockIamAuthenticator struct {
	mock.Mock
}

func (m *mockIamAuthenticator) GetToken() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *mockIamAuthenticator) AuthenticationType() string {
	args := m.Called()
	return args.String(0)
}

func (m *mockIamAuthenticator) Authenticate(request *http.Request) error {
	args := m.Called(request)
	return args.Error(0)
}

func (m *mockIamAuthenticator) Validate() error {
	args := m.Called()
	return args.Error(0)
}

func (m *mockIamAuthenticator) RequestToken() (*core.IamTokenServerResponse, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*core.IamTokenServerResponse), args.Error(1)
}

func (m *mockSchematicsService) CreateWorkspace(options *schematics.CreateWorkspaceOptions) (*schematics.WorkspaceResponse, *core.DetailedResponse, error) {
	args := m.Called(options)
	if args.Get(0) == nil {
		return nil, args.Get(1).(*core.DetailedResponse), args.Error(2)
	}
	return args.Get(0).(*schematics.WorkspaceResponse), args.Get(1).(*core.DetailedResponse), args.Error(2)
}

func (m *mockSchematicsService) UpdateWorkspace(options *schematics.UpdateWorkspaceOptions) (*schematics.WorkspaceResponse, *core.DetailedResponse, error) {
	args := m.Called(options)
	if args.Get(0) == nil {
		return nil, args.Get(1).(*core.DetailedResponse), args.Error(2)
	}
	return args.Get(0).(*schematics.WorkspaceResponse), args.Get(1).(*core.DetailedResponse), args.Error(2)
}

func (m *mockSchematicsService) DeleteWorkspace(options *schematics.DeleteWorkspaceOptions) (*string, *core.DetailedResponse, error) {
	args := m.Called(options)
	if args.Get(0) == nil {
		return nil, args.Get(1).(*core.DetailedResponse), args.Error(2)
	}
	return args.Get(0).(*string), args.Get(1).(*core.DetailedResponse), args.Error(2)
}

func (m *mockSchematicsService) TemplateRepoUpload(options *schematics.TemplateRepoUploadOptions) (*schematics.TemplateRepoTarUploadResponse, *core.DetailedResponse, error) {
	args := m.Called(options)
	if args.Get(0) == nil {
		return nil, args.Get(1).(*core.DetailedResponse), args.Error(2)
	}
	return args.Get(0).(*schematics.TemplateRepoTarUploadResponse), args.Get(1).(*core.DetailedResponse), args.Error(2)
}

func (m *mockSchematicsService) ReplaceWorkspaceInputs(options *schematics.ReplaceWorkspaceInputsOptions) (*schematics.UserValues, *core.DetailedResponse, error) {
	args := m.Called(options)
	if args.Get(0) == nil {
		return nil, args.Get(1).(*core.DetailedResponse), args.Error(2)
	}
	return args.Get(0).(*schematics.UserValues), args.Get(1).(*core.DetailedResponse), args.Error(2)
}

func (m *mockSchematicsService) ListWorkspaceActivities(options *schematics.ListWorkspaceActivitiesOptions) (*schematics.WorkspaceActivities, *core.DetailedResponse, error) {
	args := m.Called(options)
	if args.Get(0) == nil {
		return nil, args.Get(1).(*core.DetailedResponse), args.Error(2)
	}
	return args.Get(0).(*schematics.WorkspaceActivities), args.Get(1).(*core.DetailedResponse), args.Error(2)
}

func (m *mockSchematicsService) GetWorkspaceActivity(options *schematics.GetWorkspaceActivityOptions) (*schematics.WorkspaceActivity, *core.DetailedResponse, error) {
	args := m.Called(options)
	if args.Get(0) == nil {
		return nil, args.Get(1).(*core.DetailedResponse), args.Error(2)
	}
	return args.Get(0).(*schematics.WorkspaceActivity), args.Get(1).(*core.DetailedResponse), args.Error(2)
}

func (m *mockSchematicsService) PlanWorkspaceCommand(options *schematics.PlanWorkspaceCommandOptions) (*schematics.WorkspaceActivityPlanResult, *core.DetailedResponse, error) {
	args := m.Called(options)
	if args.Get(0) == nil {
		return nil, args.Get(1).(*core.DetailedResponse), args.Error(2)
	}
	return args.Get(0).(*schematics.WorkspaceActivityPlanResult), args.Get(1).(*core.DetailedResponse), args.Error(2)
}

func (m *mockSchematicsService) ApplyWorkspaceCommand(options *schematics.ApplyWorkspaceCommandOptions) (*schematics.WorkspaceActivityApplyResult, *core.DetailedResponse, error) {
	args := m.Called(options)
	if args.Get(0) == nil {
		return nil, args.Get(1).(*core.DetailedResponse), args.Error(2)
	}
	return args.Get(0).(*schematics.WorkspaceActivityApplyResult), args.Get(1).(*core.DetailedResponse), args.Error(2)
}

func (m *mockSchematicsService) DestroyWorkspaceCommand(options *schematics.DestroyWorkspaceCommandOptions) (*schematics.WorkspaceActivityDestroyResult, *core.DetailedResponse, error) {
	args := m.Called(options)
	if args.Get(0) == nil {
		return nil, args.Get(1).(*core.DetailedResponse), args.Error(2)
	}
	return args.Get(0).(*schematics.WorkspaceActivityDestroyResult), args.Get(1).(*core.DetailedResponse), args.Error(2)
}

func (m *mockSchematicsService) GetWorkspaceOutputs(options *schematics.GetWorkspaceOutputsOptions) ([]schematics.OutputValuesInner, *core.DetailedResponse, error) {
	args := m.Called(options)
	if args.Get(0) == nil {
		return nil, args.Get(1).(*core.DetailedResponse), args.Error(2)
	}
	return args.Get(0).([]schematics.OutputValuesInner), args.Get(1).(*core.DetailedResponse), args.Error(2)
}

func (m *mockSchematicsService) GetEnableGzipCompression() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *mockSchematicsService) GetServiceURL() string {
	args := m.Called()
	return args.String(0)
}

func (m *mockSchematicsService) ListJobLogs(options *schematics.ListJobLogsOptions) (*schematics.JobLog, *core.DetailedResponse, error) {
	args := m.Called(options)
	if args.Get(0) == nil {
		return nil, args.Get(1).(*core.DetailedResponse), args.Error(2)
	}
	return args.Get(0).(*schematics.JobLog), args.Get(1).(*core.DetailedResponse), args.Error(2)
}

func (m *mockSchematicsService) GetJobFiles(options *schematics.GetJobFilesOptions) (*schematics.JobFileData, *core.DetailedResponse, error) {
	args := m.Called(options)
	if args.Get(0) == nil {
		return nil, args.Get(1).(*core.DetailedResponse), args.Error(2)
	}
	return args.Get(0).(*schematics.JobFileData), args.Get(1).(*core.DetailedResponse), args.Error(2)
}

func TestCreateSchematicsWorkspace(t *testing.T) {
	mockSvc := new(mockSchematicsService)
	infoSvc := &CloudInfoService{
		schematicsServices: map[string]schematicsService{
			"us-south": mockSvc,
		},
	}

	workspaceID := "ws-12345"
	templateID := "template-12345"

	t.Run("Success", func(t *testing.T) {
		mockSvc.On("CreateWorkspace", mock.Anything).Return(
			&schematics.WorkspaceResponse{
				ID:   core.StringPtr(workspaceID),
				Name: core.StringPtr("test-workspace"),
				TemplateData: []schematics.TemplateSourceDataResponse{
					{ID: core.StringPtr(templateID)},
				},
			},
			&core.DetailedResponse{StatusCode: 201},
			nil,
		).Once()

		result, err := infoSvc.CreateSchematicsWorkspace(
			"test-workspace",
			"default",
			"us-south",
			".",
			"1.5",
			[]string{"test"},
			nil,
			nil,
		)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, workspaceID, *result.ID)
		mockSvc.AssertExpectations(t)
	})
}

func TestDeleteSchematicsWorkspace(t *testing.T) {
	mockSvc := new(mockSchematicsService)
	mockAuth := new(mockIamAuthenticator)

	infoSvc := &CloudInfoService{
		schematicsServices: map[string]schematicsService{
			"us-south": mockSvc,
		},
		authenticator: mockAuth,
	}

	t.Run("Success", func(t *testing.T) {
		mockAuth.On("RequestToken").Return(
			&core.IamTokenServerResponse{
				RefreshToken: "test-refresh-token",
			},
			nil,
		).Once()

		mockSvc.On("DeleteWorkspace", mock.Anything).Return(
			core.StringPtr("deleted"),
			&core.DetailedResponse{StatusCode: 200},
			nil,
		).Once()

		result, err := infoSvc.DeleteSchematicsWorkspace("ws-12345", "us-south", false)

		assert.NoError(t, err)
		assert.Equal(t, "deleted", result)
		mockSvc.AssertExpectations(t)
		mockAuth.AssertExpectations(t)
	})
}

func TestCreateSchematicsPlanJob(t *testing.T) {
	mockSvc := new(mockSchematicsService)
	mockAuth := new(mockIamAuthenticator)

	infoSvc := &CloudInfoService{
		schematicsServices: map[string]schematicsService{
			"us-south": mockSvc,
		},
		authenticator: mockAuth,
	}

	activityID := "activity-12345"

	t.Run("Success", func(t *testing.T) {
		mockAuth.On("RequestToken").Return(
			&core.IamTokenServerResponse{
				RefreshToken: "test-refresh-token",
			},
			nil,
		).Once()

		mockSvc.On("PlanWorkspaceCommand", mock.Anything).Return(
			&schematics.WorkspaceActivityPlanResult{
				Activityid: core.StringPtr(activityID),
			},
			&core.DetailedResponse{StatusCode: 202},
			nil,
		).Once()

		result, err := infoSvc.CreateSchematicsPlanJob("ws-12345", "us-south")

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, activityID, *result.Activityid)
		mockSvc.AssertExpectations(t)
		mockAuth.AssertExpectations(t)
	})
}

func TestCreateSchematicsApplyJob(t *testing.T) {
	mockSvc := new(mockSchematicsService)
	mockAuth := new(mockIamAuthenticator)

	infoSvc := &CloudInfoService{
		schematicsServices: map[string]schematicsService{
			"us-south": mockSvc,
		},
		authenticator: mockAuth,
	}

	activityID := "activity-12345"

	t.Run("Success", func(t *testing.T) {
		mockAuth.On("RequestToken").Return(
			&core.IamTokenServerResponse{
				RefreshToken: "test-refresh-token",
			},
			nil,
		).Once()

		mockSvc.On("ApplyWorkspaceCommand", mock.Anything).Return(
			&schematics.WorkspaceActivityApplyResult{
				Activityid: core.StringPtr(activityID),
			},
			&core.DetailedResponse{StatusCode: 202},
			nil,
		).Once()

		result, err := infoSvc.CreateSchematicsApplyJob("ws-12345", "us-south")

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, activityID, *result.Activityid)
		mockSvc.AssertExpectations(t)
		mockAuth.AssertExpectations(t)
	})
}

func TestCreateSchematicsDestroyJob(t *testing.T) {
	mockSvc := new(mockSchematicsService)
	mockAuth := new(mockIamAuthenticator)

	infoSvc := &CloudInfoService{
		schematicsServices: map[string]schematicsService{
			"us-south": mockSvc,
		},
		authenticator: mockAuth,
	}

	activityID := "activity-12345"

	t.Run("Success", func(t *testing.T) {
		mockAuth.On("RequestToken").Return(
			&core.IamTokenServerResponse{
				RefreshToken: "test-refresh-token",
			},
			nil,
		).Once()

		mockSvc.On("DestroyWorkspaceCommand", mock.Anything).Return(
			&schematics.WorkspaceActivityDestroyResult{
				Activityid: core.StringPtr(activityID),
			},
			&core.DetailedResponse{StatusCode: 202},
			nil,
		).Once()

		result, err := infoSvc.CreateSchematicsDestroyJob("ws-12345", "us-south")

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, activityID, *result.Activityid)
		mockSvc.AssertExpectations(t)
		mockAuth.AssertExpectations(t)
	})
}

func TestGetSchematicsWorkspaceOutputs(t *testing.T) {
	mockSvc := new(mockSchematicsService)

	infoSvc := &CloudInfoService{
		schematicsServices: map[string]schematicsService{
			"us-south": mockSvc,
		},
	}

	t.Run("Success", func(t *testing.T) {
		mockSvc.On("GetWorkspaceOutputs", mock.Anything).Return(
			[]schematics.OutputValuesInner{
				{
					Folder: core.StringPtr("examples/basic"),
					OutputValues: []map[string]interface{}{
						{"output1": "value1"},
					},
				},
			},
			&core.DetailedResponse{StatusCode: 200},
			nil,
		).Once()

		result, err := infoSvc.GetSchematicsWorkspaceOutputs("ws-12345", "us-south")

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "value1", result["output1"])
		mockSvc.AssertExpectations(t)
	})
}
