package testschematic

import (
	"net/http"
	"testing"
	"time"

	"github.com/IBM/go-sdk-core/v5/core"
	schematics "github.com/IBM/schematics-go-sdk/schematicsv1"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/strfmt/conv"
	"github.com/stretchr/testify/mock"
)

const mockWorkspaceID = "ws12345"
const mockTemplateID = "template54321"
const mockActivityID = "activity98765"
const mockPlanID = "plan123"
const mockApplyID = "apply123"
const mockDestroyID = "destroy123"
const mockServiceErrorText = "mock_error_from_service"

// mock error returned by the schematics V1 mock service
// this type will be checked by the unit tests to verify the mock service threw the error
type schematicErrorMock struct{}

func (e *schematicErrorMock) Error() string {
	return mockServiceErrorText
}

// SCHEMATIC SERVICE INTERFACE MOCK
type schematicServiceMock struct {
	mock.Mock
	activities                   []schematics.WorkspaceActivity
	failCreateWorkspace          bool
	failDeleteWorkspace          bool
	failTemplateRepoUpload       bool
	failReplaceWorkspaceInputs   bool
	failListWorkspaceActivities  bool
	emptyListWorkspaceActivities bool
	failGetWorkspaceActivity     bool
	failPlanWorkspaceCommand     bool
	failApplyWorkspaceCommand    bool
	failDestroyWorkspaceCommand  bool
	applyComplete                bool
	destroyComplete              bool
	workspaceDeleteComplete      bool
}

// IAM AUTHENTICATOR INTERFACE MOCK
type iamAuthenticatorMock struct {
	mock.Mock
}

// helper function to reset mock values
func mockSchematicServiceReset(mock *schematicServiceMock, options *TestSchematicOptions) {
	mock.failCreateWorkspace = false
	mock.failDeleteWorkspace = false
	mock.failTemplateRepoUpload = false
	mock.failReplaceWorkspaceInputs = false
	mock.failListWorkspaceActivities = false
	mock.emptyListWorkspaceActivities = false
	mock.failGetWorkspaceActivity = false
	mock.failPlanWorkspaceCommand = false
	mock.failApplyWorkspaceCommand = false
	mock.failDestroyWorkspaceCommand = false
	mock.applyComplete = false
	mock.destroyComplete = false
	mock.workspaceDeleteComplete = false

	options.Testing = new(testing.T)
}

// SCHEMATIC SERVICE MOCK FUNCTIONS
func (mock *schematicServiceMock) CreateWorkspace(createWorkspaceOptions *schematics.CreateWorkspaceOptions) (*schematics.WorkspaceResponse, *core.DetailedResponse, error) {
	if mock.failCreateWorkspace {
		return nil, nil, &schematicErrorMock{}
	}

	result := &schematics.WorkspaceResponse{
		ID: core.StringPtr(mockWorkspaceID),
		TemplateData: []schematics.TemplateSourceDataResponse{
			{ID: core.StringPtr(mockTemplateID)},
		},
	}
	response := &core.DetailedResponse{StatusCode: 200}
	return result, response, nil
}

func (mock *schematicServiceMock) UpdateWorkspace(updateWorkspaceOptions *schematics.UpdateWorkspaceOptions) (*schematics.WorkspaceResponse, *core.DetailedResponse, error) {
	if mock.failCreateWorkspace {
		return nil, nil, &schematicErrorMock{}
	}

	result := &schematics.WorkspaceResponse{
		ID: core.StringPtr(mockWorkspaceID),
		TemplateData: []schematics.TemplateSourceDataResponse{
			{ID: core.StringPtr(mockTemplateID)},
		},
	}
	response := &core.DetailedResponse{StatusCode: 200}
	return result, response, nil
}

func (mock *schematicServiceMock) DeleteWorkspace(deleteWorkspaceOptions *schematics.DeleteWorkspaceOptions) (*string, *core.DetailedResponse, error) {
	if mock.failDeleteWorkspace {
		return nil, nil, &schematicErrorMock{}
	}
	result := core.StringPtr("deleted")
	response := &core.DetailedResponse{StatusCode: 200}
	mock.workspaceDeleteComplete = true
	return result, response, nil
}

func (mock *schematicServiceMock) TemplateRepoUpload(templateRepoUploadOptions *schematics.TemplateRepoUploadOptions) (*schematics.TemplateRepoTarUploadResponse, *core.DetailedResponse, error) {
	if mock.failTemplateRepoUpload {
		return nil, nil, &schematicErrorMock{}
	}
	result := &schematics.TemplateRepoTarUploadResponse{
		ID: core.StringPtr(mockWorkspaceID),
	}
	response := &core.DetailedResponse{StatusCode: 200}
	return result, response, nil
}

func (mock *schematicServiceMock) ReplaceWorkspaceInputs(replaceWorkspaceInputsOptions *schematics.ReplaceWorkspaceInputsOptions) (*schematics.UserValues, *core.DetailedResponse, error) {
	if mock.failReplaceWorkspaceInputs {
		return nil, nil, &schematicErrorMock{}
	}
	result := &schematics.UserValues{
		Variablestore: []schematics.WorkspaceVariableResponse{},
	}
	response := &core.DetailedResponse{StatusCode: 200}
	return result, response, nil
}

func (mock *schematicServiceMock) ListWorkspaceActivities(listWorkspaceactivitiesOptions *schematics.ListWorkspaceActivitiesOptions) (*schematics.WorkspaceActivities, *core.DetailedResponse, error) {
	if mock.failListWorkspaceActivities {
		return nil, nil, &schematicErrorMock{}
	}
	var result *schematics.WorkspaceActivities
	if mock.emptyListWorkspaceActivities {
		result = &schematics.WorkspaceActivities{
			WorkspaceID: core.StringPtr(mockWorkspaceID),
			Actions:     []schematics.WorkspaceActivity{},
		}
	} else {
		if len(mock.activities) == 0 {
			result = &schematics.WorkspaceActivities{
				WorkspaceID: core.StringPtr(mockWorkspaceID),
				Actions: []schematics.WorkspaceActivity{
					{ActionID: core.StringPtr(mockActivityID)},
				},
			}
		} else {
			result = &schematics.WorkspaceActivities{
				WorkspaceID: core.StringPtr(mockWorkspaceID),
				Actions:     mock.activities,
			}
		}
	}
	response := &core.DetailedResponse{StatusCode: 200}
	return result, response, nil
}

func (mock *schematicServiceMock) GetWorkspaceActivity(getWorkspaceActivityOptions *schematics.GetWorkspaceActivityOptions) (*schematics.WorkspaceActivity, *core.DetailedResponse, error) {
	if mock.failGetWorkspaceActivity {
		return nil, nil, &schematicErrorMock{}
	}

	if getWorkspaceActivityOptions.WID == nil || getWorkspaceActivityOptions.ActivityID == nil {
		return nil, nil, &schematicErrorMock{}
	}

	var result *schematics.WorkspaceActivity
	if len(mock.activities) == 0 {
		result = &schematics.WorkspaceActivity{
			ActionID:    core.StringPtr(mockActivityID),
			Name:        getWorkspaceActivityOptions.WID,
			PerformedAt: conv.DateTime(strfmt.DateTime(time.Now())),
			Status:      core.StringPtr(SchematicsJobStatusCompleted),
		}
	} else {
		result = mock.findActivity(*getWorkspaceActivityOptions.ActivityID)
	}
	response := &core.DetailedResponse{StatusCode: 200}
	return result, response, nil
}

func (mock *schematicServiceMock) findActivity(id string) *schematics.WorkspaceActivity {
	for _, activity := range mock.activities {
		if *activity.ActionID == id {
			return &activity
		}
	}
	return &mock.activities[0]
}

func (mock *schematicServiceMock) PlanWorkspaceCommand(planWorkspaceCommandOptions *schematics.PlanWorkspaceCommandOptions) (*schematics.WorkspaceActivityPlanResult, *core.DetailedResponse, error) {
	if mock.failPlanWorkspaceCommand {
		return nil, nil, &schematicErrorMock{}
	}
	result := &schematics.WorkspaceActivityPlanResult{
		Activityid: core.StringPtr(mockPlanID),
	}
	response := &core.DetailedResponse{StatusCode: 200}
	return result, response, nil
}

func (mock *schematicServiceMock) ApplyWorkspaceCommand(applyWorkspaceCommandOptions *schematics.ApplyWorkspaceCommandOptions) (*schematics.WorkspaceActivityApplyResult, *core.DetailedResponse, error) {
	if mock.failApplyWorkspaceCommand {
		return nil, nil, &schematicErrorMock{}
	}
	result := &schematics.WorkspaceActivityApplyResult{
		Activityid: core.StringPtr(mockApplyID),
	}
	response := &core.DetailedResponse{StatusCode: 200}
	mock.applyComplete = true
	return result, response, nil
}

func (mock *schematicServiceMock) DestroyWorkspaceCommand(destroyWorkspaceCommandOptions *schematics.DestroyWorkspaceCommandOptions) (*schematics.WorkspaceActivityDestroyResult, *core.DetailedResponse, error) {
	if mock.failDestroyWorkspaceCommand {
		return nil, nil, &schematicErrorMock{}
	}
	result := &schematics.WorkspaceActivityDestroyResult{
		Activityid: core.StringPtr(mockDestroyID),
	}
	response := &core.DetailedResponse{StatusCode: 200}
	mock.destroyComplete = true
	return result, response, nil
}

// IAM AUTHENTIATOR INTERFACE MOCK FUNCTIONS
func (mock *iamAuthenticatorMock) Authenticate(request *http.Request) error {
	return nil
}

func (mock *iamAuthenticatorMock) AuthenticationType() string {
	return core.AUTHTYPE_IAM
}

func (mock *iamAuthenticatorMock) Validate() error {
	return nil
}

func (mock *iamAuthenticatorMock) RequestToken() (*core.IamTokenServerResponse, error) {
	retval := &core.IamTokenServerResponse{
		AccessToken:  "fake-token",
		RefreshToken: "fake-refresh-token",
	}

	return retval, nil
}
