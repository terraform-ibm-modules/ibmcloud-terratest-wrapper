package testschematic

import (
	"testing"
	"time"

	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/schematics-go-sdk/schematicsv1"
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
type schematicv1ErrorMock struct{}

func (e *schematicv1ErrorMock) Error() string {
	return mockServiceErrorText
}

// VPC SERVICE INTERFACE MOCK
type schematicv1ServiceMock struct {
	mock.Mock
	activities                   []schematicsv1.WorkspaceActivity
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

// helper function to reset mock values
func mockSchematicv1ServiceReset(mock *schematicv1ServiceMock, options *TestSchematicOptions) {
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

func (mock *schematicv1ServiceMock) CreateWorkspace(createWorkspaceOptions *schematicsv1.CreateWorkspaceOptions) (*schematicsv1.WorkspaceResponse, *core.DetailedResponse, error) {
	if mock.failCreateWorkspace {
		return nil, nil, &schematicv1ErrorMock{}
	}

	result := &schematicsv1.WorkspaceResponse{
		ID: core.StringPtr(mockWorkspaceID),
		TemplateData: []schematicsv1.TemplateSourceDataResponse{
			{ID: core.StringPtr(mockTemplateID)},
		},
	}
	response := &core.DetailedResponse{StatusCode: 200}
	return result, response, nil
}

func (mock *schematicv1ServiceMock) UpdateWorkspace(updateWorkspaceOptions *schematicsv1.UpdateWorkspaceOptions) (*schematicsv1.WorkspaceResponse, *core.DetailedResponse, error) {
	if mock.failCreateWorkspace {
		return nil, nil, &schematicv1ErrorMock{}
	}

	result := &schematicsv1.WorkspaceResponse{
		ID: core.StringPtr(mockWorkspaceID),
		TemplateData: []schematicsv1.TemplateSourceDataResponse{
			{ID: core.StringPtr(mockTemplateID)},
		},
	}
	response := &core.DetailedResponse{StatusCode: 200}
	return result, response, nil
}

func (mock *schematicv1ServiceMock) DeleteWorkspace(deleteWorkspaceOptions *schematicsv1.DeleteWorkspaceOptions) (*string, *core.DetailedResponse, error) {
	if mock.failDeleteWorkspace {
		return nil, nil, &schematicv1ErrorMock{}
	}
	result := core.StringPtr("deleted")
	response := &core.DetailedResponse{StatusCode: 200}
	mock.workspaceDeleteComplete = true
	return result, response, nil
}

func (mock *schematicv1ServiceMock) TemplateRepoUpload(templateRepoUploadOptions *schematicsv1.TemplateRepoUploadOptions) (*schematicsv1.TemplateRepoTarUploadResponse, *core.DetailedResponse, error) {
	if mock.failTemplateRepoUpload {
		return nil, nil, &schematicv1ErrorMock{}
	}
	result := &schematicsv1.TemplateRepoTarUploadResponse{
		ID: core.StringPtr(mockWorkspaceID),
	}
	response := &core.DetailedResponse{StatusCode: 200}
	return result, response, nil
}

func (mock *schematicv1ServiceMock) ReplaceWorkspaceInputs(replaceWorkspaceInputsOptions *schematicsv1.ReplaceWorkspaceInputsOptions) (*schematicsv1.UserValues, *core.DetailedResponse, error) {
	if mock.failReplaceWorkspaceInputs {
		return nil, nil, &schematicv1ErrorMock{}
	}
	result := &schematicsv1.UserValues{
		Variablestore: []schematicsv1.WorkspaceVariableResponse{},
	}
	response := &core.DetailedResponse{StatusCode: 200}
	return result, response, nil
}

func (mock *schematicv1ServiceMock) ListWorkspaceActivities(listWorkspaceactivitiesOptions *schematicsv1.ListWorkspaceActivitiesOptions) (*schematicsv1.WorkspaceActivities, *core.DetailedResponse, error) {
	if mock.failListWorkspaceActivities {
		return nil, nil, &schematicv1ErrorMock{}
	}
	var result *schematicsv1.WorkspaceActivities
	if mock.emptyListWorkspaceActivities {
		result = &schematicsv1.WorkspaceActivities{
			WorkspaceID: core.StringPtr(mockWorkspaceID),
			Actions:     []schematicsv1.WorkspaceActivity{},
		}
	} else {
		if len(mock.activities) == 0 {
			result = &schematicsv1.WorkspaceActivities{
				WorkspaceID: core.StringPtr(mockWorkspaceID),
				Actions: []schematicsv1.WorkspaceActivity{
					{ActionID: core.StringPtr(mockActivityID)},
				},
			}
		} else {
			result = &schematicsv1.WorkspaceActivities{
				WorkspaceID: core.StringPtr(mockWorkspaceID),
				Actions:     mock.activities,
			}
		}
	}
	response := &core.DetailedResponse{StatusCode: 200}
	return result, response, nil
}

func (mock *schematicv1ServiceMock) GetWorkspaceActivity(getWorkspaceActivityOptions *schematicsv1.GetWorkspaceActivityOptions) (*schematicsv1.WorkspaceActivity, *core.DetailedResponse, error) {
	if mock.failGetWorkspaceActivity {
		return nil, nil, &schematicv1ErrorMock{}
	}

	if getWorkspaceActivityOptions.WID == nil || getWorkspaceActivityOptions.ActivityID == nil {
		return nil, nil, &schematicv1ErrorMock{}
	}

	var result *schematicsv1.WorkspaceActivity
	if len(mock.activities) == 0 {
		result = &schematicsv1.WorkspaceActivity{
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

func (mock *schematicv1ServiceMock) findActivity(id string) *schematicsv1.WorkspaceActivity {
	for _, activity := range mock.activities {
		if *activity.ActionID == id {
			return &activity
		}
	}
	return &mock.activities[0]
}

func (mock *schematicv1ServiceMock) PlanWorkspaceCommand(planWorkspaceCommandOptions *schematicsv1.PlanWorkspaceCommandOptions) (*schematicsv1.WorkspaceActivityPlanResult, *core.DetailedResponse, error) {
	if mock.failPlanWorkspaceCommand {
		return nil, nil, &schematicv1ErrorMock{}
	}
	result := &schematicsv1.WorkspaceActivityPlanResult{
		Activityid: core.StringPtr(mockPlanID),
	}
	response := &core.DetailedResponse{StatusCode: 200}
	return result, response, nil
}

func (mock *schematicv1ServiceMock) ApplyWorkspaceCommand(applyWorkspaceCommandOptions *schematicsv1.ApplyWorkspaceCommandOptions) (*schematicsv1.WorkspaceActivityApplyResult, *core.DetailedResponse, error) {
	if mock.failApplyWorkspaceCommand {
		return nil, nil, &schematicv1ErrorMock{}
	}
	result := &schematicsv1.WorkspaceActivityApplyResult{
		Activityid: core.StringPtr(mockApplyID),
	}
	response := &core.DetailedResponse{StatusCode: 200}
	mock.applyComplete = true
	return result, response, nil
}

func (mock *schematicv1ServiceMock) DestroyWorkspaceCommand(destroyWorkspaceCommandOptions *schematicsv1.DestroyWorkspaceCommandOptions) (*schematicsv1.WorkspaceActivityDestroyResult, *core.DetailedResponse, error) {
	if mock.failDestroyWorkspaceCommand {
		return nil, nil, &schematicv1ErrorMock{}
	}
	result := &schematicsv1.WorkspaceActivityDestroyResult{
		Activityid: core.StringPtr(mockDestroyID),
	}
	response := &core.DetailedResponse{StatusCode: 200}
	mock.destroyComplete = true
	return result, response, nil
}
