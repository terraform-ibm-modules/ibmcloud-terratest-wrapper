package testschematic

import (
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/platform-services-go-sdk/catalogmanagementv1"
	projects "github.com/IBM/project-go-sdk/projectv1"
	schematics "github.com/IBM/schematics-go-sdk/schematicsv1"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/strfmt/conv"
	"github.com/stretchr/testify/mock"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/cloudinfo"
)

const mockWorkspaceID = "ws12345"
const mockWorkspaceName = "NewWorkspace-12345"
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
	failReplaceWorkspace         bool
	failDeleteWorkspace          bool
	failTemplateRepoUpload       bool
	failReplaceWorkspaceInputs   bool
	failListWorkspaceActivities  bool
	emptyListWorkspaceActivities bool
	failGetWorkspaceActivity     bool
	failPlanWorkspaceCommand     bool
	failApplyWorkspaceCommand    bool
	failDestroyWorkspaceCommand  bool
	failGetOutputsCommand        bool
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
	mock.failReplaceWorkspace = false
	mock.failTemplateRepoUpload = false
	mock.failReplaceWorkspaceInputs = false
	mock.failListWorkspaceActivities = false
	mock.emptyListWorkspaceActivities = false
	mock.failGetWorkspaceActivity = false
	mock.failPlanWorkspaceCommand = false
	mock.failApplyWorkspaceCommand = false
	mock.failDestroyWorkspaceCommand = false
	mock.failGetOutputsCommand = false
	mock.applyComplete = false
	mock.destroyComplete = false
	mock.workspaceDeleteComplete = false

	options.Testing = new(testing.T)
}

// SCHEMATIC SERVICE MOCK FUNCTIONS
func (mock *schematicServiceMock) CreateWorkspace(createWorkspaceOptions *schematics.CreateWorkspaceOptions) (*schematics.WorkspaceResponse, *core.DetailedResponse, error) {
	if mock.failCreateWorkspace {
		return nil, &core.DetailedResponse{StatusCode: 404}, &schematicErrorMock{}
	}

	result := &schematics.WorkspaceResponse{
		ID:   core.StringPtr(mockWorkspaceID),
		Name: core.StringPtr(mockWorkspaceName),
		TemplateData: []schematics.TemplateSourceDataResponse{
			{ID: core.StringPtr(mockTemplateID)},
		},
	}
	response := &core.DetailedResponse{StatusCode: 200}
	return result, response, nil
}

func (mock *schematicServiceMock) UpdateWorkspace(updateWorkspaceOptions *schematics.UpdateWorkspaceOptions) (*schematics.WorkspaceResponse, *core.DetailedResponse, error) {
	if mock.failCreateWorkspace {
		return nil, &core.DetailedResponse{StatusCode: 404}, &schematicErrorMock{}
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

func (mock *schematicServiceMock) ReplaceWorkspace(replaceWorkspaceOptions *schematics.ReplaceWorkspaceOptions) (*schematics.WorkspaceResponse, *core.DetailedResponse, error) {
	if mock.failReplaceWorkspace {
		return nil, &core.DetailedResponse{StatusCode: 404}, &schematicErrorMock{}
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
		return nil, &core.DetailedResponse{StatusCode: 404}, &schematicErrorMock{}
	}
	result := core.StringPtr("deleted")
	response := &core.DetailedResponse{StatusCode: 200}
	mock.workspaceDeleteComplete = true
	return result, response, nil
}

func (mock *schematicServiceMock) TemplateRepoUpload(templateRepoUploadOptions *schematics.TemplateRepoUploadOptions) (*schematics.TemplateRepoTarUploadResponse, *core.DetailedResponse, error) {
	if mock.failTemplateRepoUpload {
		return nil, &core.DetailedResponse{StatusCode: 404}, &schematicErrorMock{}
	}
	result := &schematics.TemplateRepoTarUploadResponse{
		ID: core.StringPtr(mockWorkspaceID),
	}
	response := &core.DetailedResponse{StatusCode: 200}
	return result, response, nil
}

func (mock *schematicServiceMock) ReplaceWorkspaceInputs(replaceWorkspaceInputsOptions *schematics.ReplaceWorkspaceInputsOptions) (*schematics.UserValues, *core.DetailedResponse, error) {
	if mock.failReplaceWorkspaceInputs {
		return nil, &core.DetailedResponse{StatusCode: 404}, &schematicErrorMock{}
	}
	result := &schematics.UserValues{
		Variablestore: []schematics.WorkspaceVariableResponse{},
	}
	response := &core.DetailedResponse{StatusCode: 200}
	return result, response, nil
}

func (mock *schematicServiceMock) ListWorkspaceActivities(listWorkspaceactivitiesOptions *schematics.ListWorkspaceActivitiesOptions) (*schematics.WorkspaceActivities, *core.DetailedResponse, error) {
	if mock.failListWorkspaceActivities {
		return nil, &core.DetailedResponse{StatusCode: 404}, &schematicErrorMock{}
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
		return nil, &core.DetailedResponse{StatusCode: 404}, &schematicErrorMock{}
	}

	if getWorkspaceActivityOptions.WID == nil || getWorkspaceActivityOptions.ActivityID == nil {
		return nil, &core.DetailedResponse{StatusCode: 404}, &schematicErrorMock{}
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
		return nil, &core.DetailedResponse{StatusCode: 404}, &schematicErrorMock{}
	}
	result := &schematics.WorkspaceActivityPlanResult{
		Activityid: core.StringPtr(mockPlanID),
	}
	response := &core.DetailedResponse{StatusCode: 200}
	return result, response, nil
}

func (mock *schematicServiceMock) ApplyWorkspaceCommand(applyWorkspaceCommandOptions *schematics.ApplyWorkspaceCommandOptions) (*schematics.WorkspaceActivityApplyResult, *core.DetailedResponse, error) {
	if mock.failApplyWorkspaceCommand {
		return nil, &core.DetailedResponse{StatusCode: 404}, &schematicErrorMock{}
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
		return nil, &core.DetailedResponse{StatusCode: 404}, &schematicErrorMock{}
	}
	result := &schematics.WorkspaceActivityDestroyResult{
		Activityid: core.StringPtr(mockDestroyID),
	}
	response := &core.DetailedResponse{StatusCode: 200}
	mock.destroyComplete = true
	return result, response, nil
}

func (mock *schematicServiceMock) GetWorkspaceOutputs(getWorkspaceOutputsOptions *schematics.GetWorkspaceOutputsOptions) ([]schematics.OutputValuesInner, *core.DetailedResponse, error) {
	if mock.failGetOutputsCommand {
		return nil, &core.DetailedResponse{StatusCode: 404}, &schematicErrorMock{}
	}

	result := []schematics.OutputValuesInner{
		{
			Folder: core.StringPtr("examples/basic"),
			OutputValues: []map[string]interface{}{
				{"mock_output": "the_mock_value"},
			},
		},
	}
	response := &core.DetailedResponse{StatusCode: 200}
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

/**** START MOCK CloudInfoService ****/
type cloudInfoServiceMock struct {
	mock.Mock
	cloudinfo.CloudInfoServiceI
	lock sync.Mutex
}

func (mock *cloudInfoServiceMock) CreateStackDefinitionWrapper(stackDefOptions *projects.CreateStackDefinitionOptions, members []projects.StackConfigMember) (result *projects.StackDefinition, response *core.DetailedResponse, err error) {
	return nil, nil, nil
}

func (mock *cloudInfoServiceMock) LoadRegionPrefsFromFile(prefsFile string) error {
	return nil
}

func (mock *cloudInfoServiceMock) GetLeastVpcTestRegion() (string, error) {
	return "us-south", nil
}

func (mock *cloudInfoServiceMock) GetLeastVpcTestRegionWithoutActivityTracker() (string, error) {
	return "us-east", nil
}

func (mock *cloudInfoServiceMock) GetLeastPowerConnectionZone() (string, error) {
	return "us-south", nil
}

func (mock *cloudInfoServiceMock) HasRegionData() bool {
	return false
}

func (mock *cloudInfoServiceMock) RemoveRegionForTest(regionID string) {
	// nothing to really do here
}

func (mock *cloudInfoServiceMock) GetThreadLock() *sync.Mutex {
	return &mock.lock
}

func (mock *cloudInfoServiceMock) GetCatalogVersionByLocator(string) (*catalogmanagementv1.Version, error) {
	return nil, nil
}
func (mock *cloudInfoServiceMock) CreateProjectFromConfig(*cloudinfo.ProjectsConfig) (*projects.Project, *core.DetailedResponse, error) {
	return nil, nil, nil
}

func (mock *cloudInfoServiceMock) GetProject(string) (*projects.Project, *core.DetailedResponse, error) {
	return nil, nil, nil
}

func (mock *cloudInfoServiceMock) GetProjectConfigs(string) ([]projects.ProjectConfigSummary, error) {
	return nil, nil
}

func (mock *cloudInfoServiceMock) GetConfig(*cloudinfo.ConfigDetails) (result *projects.ProjectConfig, response *core.DetailedResponse, err error) {
	return nil, nil, nil
}

func (mock *cloudInfoServiceMock) DeleteProject(string) (*projects.ProjectDeleteResponse, *core.DetailedResponse, error) {
	return nil, nil, nil
}

func (mock *cloudInfoServiceMock) CreateConfig(*cloudinfo.ConfigDetails) (result *projects.ProjectConfig, response *core.DetailedResponse, err error) {
	return nil, nil, nil
}

func (mock *cloudInfoServiceMock) DeployConfig(*cloudinfo.ConfigDetails) (result *projects.ProjectConfigVersion, response *core.DetailedResponse, err error) {
	return nil, nil, nil
}

func (mock *cloudInfoServiceMock) CreateDaConfig(*cloudinfo.ConfigDetails) (result *projects.ProjectConfig, response *core.DetailedResponse, err error) {
	return nil, nil, nil
}

func (mock *cloudInfoServiceMock) CreateConfigFromCatalogJson(*cloudinfo.ConfigDetails, string) (result *projects.ProjectConfig, response *core.DetailedResponse, err error) {
	return nil, nil, nil
}

func (mock *cloudInfoServiceMock) UpdateConfig(*cloudinfo.ConfigDetails, projects.ProjectConfigDefinitionPatchIntf) (result *projects.ProjectConfig, response *core.DetailedResponse, err error) {
	return nil, nil, nil
}

func (mock *cloudInfoServiceMock) ValidateProjectConfig(*cloudinfo.ConfigDetails) (result *projects.ProjectConfigVersion, response *core.DetailedResponse, err error) {
	return nil, nil, nil
}

func (mock *cloudInfoServiceMock) IsConfigDeployed(*cloudinfo.ConfigDetails) (projectConfig *projects.ProjectConfigVersion, isDeployed bool) {
	return nil, false
}

func (mock *cloudInfoServiceMock) UndeployConfig(*cloudinfo.ConfigDetails) (result *projects.ProjectConfigVersion, response *core.DetailedResponse, err error) {
	return nil, nil, nil
}

func (mock *cloudInfoServiceMock) IsUndeploying(*cloudinfo.ConfigDetails) (projectConfig *projects.ProjectConfigVersion, isUndeploying bool) {
	return nil, false
}

func (mock *cloudInfoServiceMock) CreateStackFromConfigFile(*cloudinfo.ConfigDetails, string, string) (result *projects.StackDefinition, response *core.DetailedResponse, err error) {
	return nil, nil, nil
}

func (mock *cloudInfoServiceMock) GetProjectConfigVersion(*cloudinfo.ConfigDetails, int64) (result *projects.ProjectConfigVersion, response *core.DetailedResponse, err error) {
	return nil, nil, nil
}

func (mock *cloudInfoServiceMock) GetStackMembers(stackConfig *cloudinfo.ConfigDetails) ([]*projects.ProjectConfig, error) {
	return nil, nil
}

func (mock *cloudInfoServiceMock) SyncConfig(projectID string, configID string) (response *core.DetailedResponse, err error) {
	return nil, nil
}

func (mock *cloudInfoServiceMock) LookupMemberNameByID(stackDetails *projects.ProjectConfig, memberID string) (string, error) {
	return "", nil
}

func (mock *cloudInfoServiceMock) GetClusterIngressStatus(string) (string, error) {
	return "", nil
}

func (mock *cloudInfoServiceMock) GetSchematicsJobLogs(string, string) (*schematics.JobLog, *core.DetailedResponse, error) {
	return nil, nil, nil

}
func (mock *cloudInfoServiceMock) GetSchematicsJobLogsText(string, string) (string, error) {
	return "", nil
}

func (mock *cloudInfoServiceMock) ArePipelineActionsRunning(stackConfig *cloudinfo.ConfigDetails) (bool, error) {
	return false, nil
}

func (mock *cloudInfoServiceMock) GetSchematicsJobLogsForMember(member *projects.ProjectConfig, memberName string, projectRegion string) (string, string) {
	return "", ""
}

// special mock for CreateStackDefinition
// we do not have enough information when mocking projectv1.CreateStackDefinition to return a valid response
// to get around this we create a wrapper that can take in the missing list of members that can be used in the mock
// to return a valid response

func (mock *cloudInfoServiceMock) CreateStackDefinition(stackDefOptions *projects.CreateStackDefinitionOptions, members []projects.StackConfigMember) (result *projects.StackDefinition, response *core.DetailedResponse, err error) {
	args := mock.Called(stackDefOptions, members)
	return args.Get(0).(*projects.StackDefinition), args.Get(1).(*core.DetailedResponse), args.Error(2)
}

func (mock *cloudInfoServiceMock) GetSchematicsJobFileData(jobID string, fileType string, location string) (*schematics.JobFileData, error) {
	dummyFile := &schematics.JobFileData{
		JobID:       core.StringPtr(jobID),
		FileContent: core.StringPtr("testing 1 2 3"),
	}
	return dummyFile, nil
}

func (mock *cloudInfoServiceMock) GetSchematicsJobPlanJson(jobID string, location string) (string, error) {
	// needed a valid json for marshalling, this is the terraform-ibm-resource-group plan with no changes
	return "{\"format_version\":\"1.2\",\"terraform_version\":\"1.9.2\",\"variables\":{\"ibmcloud_api_key\":{\"value\":\"dummy-key\"},\"resource_group_name\":{\"value\":\"geretain-test-resources\"}},\"planned_values\":{\"outputs\":{\"resource_group_id\":{\"sensitive\":false,\"type\":\"string\",\"value\":\"292170bc79c94f5e9019e46fb48f245a\"},\"resource_group_name\":{\"sensitive\":false,\"type\":\"string\",\"value\":\"geretain-test-resources\"}},\"root_module\":{}},\"output_changes\":{\"resource_group_id\":{\"actions\":[\"no-op\"],\"before\":\"292170bc79c94f5e9019e46fb48f245a\",\"after\":\"292170bc79c94f5e9019e46fb48f245a\",\"after_unknown\":false,\"before_sensitive\":false,\"after_sensitive\":false},\"resource_group_name\":{\"actions\":[\"no-op\"],\"before\":\"geretain-test-resources\",\"after\":\"geretain-test-resources\",\"after_unknown\":false,\"before_sensitive\":false,\"after_sensitive\":false}},\"prior_state\":{\"format_version\":\"1.0\",\"terraform_version\":\"1.9.2\",\"values\":{\"outputs\":{\"resource_group_id\":{\"sensitive\":false,\"value\":\"292170bc79c94f5e9019e46fb48f245a\",\"type\":\"string\"},\"resource_group_name\":{\"sensitive\":false,\"value\":\"geretain-test-resources\",\"type\":\"string\"}},\"root_module\":{\"child_modules\":[{\"resources\":[{\"address\":\"module.resource_group.data.ibm_resource_group.existing_resource_group[0]\",\"mode\":\"data\",\"type\":\"ibm_resource_group\",\"name\":\"existing_resource_group\",\"index\":0,\"provider_name\":\"registry.terraform.io/ibm-cloud/ibm\",\"schema_version\":0,\"values\":{\"account_id\":\"abac0df06b644a9cabc6e44f55b3880e\",\"created_at\":\"2022-08-04T16:52:02.227Z\",\"crn\":\"crn:v1:bluemix:public:resource-controller::a/abac0df06b644a9cabc6e44f55b3880e::resource-group:292170bc79c94f5e9019e46fb48f245a\",\"id\":\"292170bc79c94f5e9019e46fb48f245a\",\"is_default\":false,\"name\":\"geretain-test-resources\",\"payment_methods_url\":null,\"quota_id\":\"a3d7b8d01e261c24677937c29ab33f3c\",\"quota_url\":\"/v2/quota_definitions/a3d7b8d01e261c24677937c29ab33f3c\",\"resource_linkages\":[],\"state\":\"ACTIVE\",\"teams_url\":null,\"updated_at\":\"2022-08-04T16:52:02.227Z\"},\"sensitive_values\":{\"resource_linkages\":[]}}],\"address\":\"module.resource_group\"}]}}},\"configuration\":{\"provider_config\":{\"ibm\":{\"name\":\"ibm\",\"full_name\":\"registry.terraform.io/ibm-cloud/ibm\",\"version_constraint\":\"1.49.0\",\"expressions\":{\"ibmcloud_api_key\":{\"references\":[\"var.ibmcloud_api_key\"]}}}},\"root_module\":{\"outputs\":{\"resource_group_id\":{\"expression\":{\"references\":[\"module.resource_group.resource_group_id\",\"module.resource_group\"]},\"description\":\"Resource group ID\"},\"resource_group_name\":{\"expression\":{\"references\":[\"module.resource_group.resource_group_name\",\"module.resource_group\"]},\"description\":\"Resource group name\"}},\"module_calls\":{\"resource_group\":{\"source\":\"../../\",\"expressions\":{\"existing_resource_group_name\":{\"references\":[\"var.resource_group_name\"]}},\"module\":{\"outputs\":{\"resource_group_id\":{\"expression\":{\"references\":[\"var.existing_resource_group_name\",\"data.ibm_resource_group.existing_resource_group[0].id\",\"data.ibm_resource_group.existing_resource_group[0]\",\"data.ibm_resource_group.existing_resource_group\",\"ibm_resource_group.resource_group[0].id\",\"ibm_resource_group.resource_group[0]\",\"ibm_resource_group.resource_group\"]},\"description\":\"Resource group ID\"},\"resource_group_name\":{\"expression\":{\"references\":[\"var.existing_resource_group_name\",\"data.ibm_resource_group.existing_resource_group[0].name\",\"data.ibm_resource_group.existing_resource_group[0]\",\"data.ibm_resource_group.existing_resource_group\",\"ibm_resource_group.resource_group[0].name\",\"ibm_resource_group.resource_group[0]\",\"ibm_resource_group.resource_group\"]},\"description\":\"Resource group name\"}},\"resources\":[{\"address\":\"ibm_resource_group.resource_group\",\"mode\":\"managed\",\"type\":\"ibm_resource_group\",\"name\":\"resource_group\",\"provider_config_key\":\"ibm\",\"expressions\":{\"name\":{\"references\":[\"var.resource_group_name\"]},\"quota_id\":{\"constant_value\":null}},\"schema_version\":0,\"count_expression\":{\"references\":[\"var.existing_resource_group_name\"]}},{\"address\":\"data.ibm_resource_group.existing_resource_group\",\"mode\":\"data\",\"type\":\"ibm_resource_group\",\"name\":\"existing_resource_group\",\"provider_config_key\":\"ibm\",\"expressions\":{\"name\":{\"references\":[\"var.existing_resource_group_name\"]}},\"schema_version\":0,\"count_expression\":{\"references\":[\"var.existing_resource_group_name\"]}}],\"variables\":{\"existing_resource_group_name\":{\"default\":null,\"description\":\"Name of the existing resource group.  Required if not creating new resource group\"},\"resource_group_name\":{\"default\":null,\"description\":\"Name of the resource group to create. Required if not using existing resource group\"}}}}},\"variables\":{\"ibmcloud_api_key\":{\"description\":\"The IBM Cloud API Token\",\"sensitive\":true},\"resource_group_name\":{\"description\":\"Resource group name\"}}}},\"relevant_attributes\":[{\"resource\":\"module.resource_group.data.ibm_resource_group.existing_resource_group[0]\",\"attribute\":[\"name\"]},{\"resource\":\"module.resource_group.ibm_resource_group.resource_group[0]\",\"attribute\":[\"name\"]},{\"resource\":\"module.resource_group.data.ibm_resource_group.existing_resource_group[0]\",\"attribute\":[\"id\"]},{\"resource\":\"module.resource_group.ibm_resource_group.resource_group[0]\",\"attribute\":[\"id\"]}],\"timestamp\":\"2024-11-13T21:02:28Z\",\"applyable\":false,\"complete\":true,\"errored\":false}", nil
}

/**** END MOCK CloudInfoService ****/
