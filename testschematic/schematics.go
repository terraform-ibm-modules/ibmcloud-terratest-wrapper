// Package testschematic contains functions that can be used to assist and standardize the execution of unit tests for IBM Cloud Terraform projects
// by using the IBM Cloud Schematics service
package testschematic

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/IBM/go-sdk-core/v5/core"
	schematics "github.com/IBM/schematics-go-sdk/schematicsv1"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/cloudinfo"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/common"
)

// Re-export constants from cloudinfo for backward compatibility
const (
	SchematicsJobTypeUpload       = cloudinfo.SchematicsJobTypeUpload
	SchematicsJobTypeUpdate       = cloudinfo.SchematicsJobTypeUpdate
	SchematicsJobTypePlan         = cloudinfo.SchematicsJobTypePlan
	SchematicsJobTypeApply        = cloudinfo.SchematicsJobTypeApply
	SchematicsJobTypeDestroy      = cloudinfo.SchematicsJobTypeDestroy
	SchematicsJobStatusCompleted  = cloudinfo.SchematicsJobStatusCompleted
	SchematicsJobStatusFailed     = cloudinfo.SchematicsJobStatusFailed
	SchematicsJobStatusCreated    = cloudinfo.SchematicsJobStatusCreated
	SchematicsJobStatusInProgress = cloudinfo.SchematicsJobStatusInProgress
)

// interface for the external schematics service api. Can be mocked for tests
type SchematicsApiSvcI interface {
	CreateWorkspace(*schematics.CreateWorkspaceOptions) (*schematics.WorkspaceResponse, *core.DetailedResponse, error)
	UpdateWorkspace(*schematics.UpdateWorkspaceOptions) (*schematics.WorkspaceResponse, *core.DetailedResponse, error)
	DeleteWorkspace(*schematics.DeleteWorkspaceOptions) (*string, *core.DetailedResponse, error)
	TemplateRepoUpload(*schematics.TemplateRepoUploadOptions) (*schematics.TemplateRepoTarUploadResponse, *core.DetailedResponse, error)
	ReplaceWorkspaceInputs(*schematics.ReplaceWorkspaceInputsOptions) (*schematics.UserValues, *core.DetailedResponse, error)
	ListWorkspaceActivities(*schematics.ListWorkspaceActivitiesOptions) (*schematics.WorkspaceActivities, *core.DetailedResponse, error)
	GetWorkspaceActivity(*schematics.GetWorkspaceActivityOptions) (*schematics.WorkspaceActivity, *core.DetailedResponse, error)
	PlanWorkspaceCommand(*schematics.PlanWorkspaceCommandOptions) (*schematics.WorkspaceActivityPlanResult, *core.DetailedResponse, error)
	ApplyWorkspaceCommand(*schematics.ApplyWorkspaceCommandOptions) (*schematics.WorkspaceActivityApplyResult, *core.DetailedResponse, error)
	DestroyWorkspaceCommand(*schematics.DestroyWorkspaceCommandOptions) (*schematics.WorkspaceActivityDestroyResult, *core.DetailedResponse, error)
	ReplaceWorkspace(*schematics.ReplaceWorkspaceOptions) (*schematics.WorkspaceResponse, *core.DetailedResponse, error)
	GetWorkspaceOutputs(*schematics.GetWorkspaceOutputsOptions) ([]schematics.OutputValuesInner, *core.DetailedResponse, error)
}

// interface for external IBMCloud IAM Authenticator api. Can be mocked for tests
type IamAuthenticatorSvcI interface {
	Authenticate(*http.Request) error
	AuthenticationType() string
	RequestToken() (*core.IamTokenServerResponse, error)
	Validate() error
}

// main data struct for all schematic test methods
type SchematicsTestService struct {
	SchematicsApiSvc          SchematicsApiSvcI           // the main schematics service interface
	ApiAuthenticator          IamAuthenticatorSvcI        // the authenticator used for schematics api calls
	WorkspaceID               string                      // workspace ID used for tests
	WorkspaceName             string                      // name of workspace that was created for test
	WorkspaceNameForLog       string                      // combination of Name and ID useful for log consistency
	WorkspaceLocation         string                      // region the workspace was created in
	TemplateID                string                      // workspace template ID used for tests
	TestOptions               *TestSchematicOptions       // additional testing options
	TerraformTestStarted      bool                        // keeps track of when actual Terraform resource testing has begin, used for proper test teardown logic
	TerraformResourcesCreated bool                        // keeps track of when we start deploying resources, used for proper test teardown logic
	CloudInfoService          cloudinfo.CloudInfoServiceI // reference to a CloudInfoService resource
	BaseTerraformRepo         string                      // the URL of the origin git repository, typically the base that the PR will merge into, used for upgrade test
	BaseTerraformRepoBranch   string                      // the branch name of the main origin branch of the project (main or master), used for upgrade test
	TestTerraformRepo         string                      // the URL of the repo for the pull request, will be either origin or a fork
	TestTerraformRepoBranch   string                      // the branch of the test, usually the current checked out branch of the test run
	BaseTerraformTempDir      string                      // if upgrade test, will contain the temp directory containing clone of base repo
}

// CreateAuthenticator will accept a valid IBM cloud API key, and
// set a valid Authenticator object that will be used in the external provider service for schematics.
func (svc *SchematicsTestService) CreateAuthenticator(ibmcloudApiKey string) {
	svc.ApiAuthenticator = &core.IamAuthenticator{
		ApiKey: ibmcloudApiKey, // pragma: allowlist secret
		// the user bx:bx is required here for all IAM calls so that a refresh_token is returned, see here: https://cloud.ibm.com/apidocs/schematics/schematics#apply-workspace-command
		ClientId:     "bx", // pragma: allowlist secret
		ClientSecret: "bx", // pragma: allowlist secret
	}
}

// GetRefreshToken will use a previously established Authenticator to create a new IAM Token object,
// if existing is not valid, and return the refresh token propery from the token object.
func (svc *SchematicsTestService) GetRefreshToken() (string, error) {
	response, err := svc.ApiAuthenticator.RequestToken()
	if err != nil {
		return "", err
	}
	if len(response.RefreshToken) == 0 {
		// this shouldn't happen
		return "", fmt.Errorf("refresh token is empty (invalid)")
	}

	return response.RefreshToken, nil
}

// InitializeSchematicsService will initialize the external service object
// for schematicsv1 and assign it to a property of the receiver for later use.
func (svc *SchematicsTestService) InitializeSchematicsService() error {
	var err error
	var getUrlErr error
	var schematicsURL string // will default to empty which is ok

	// if override of URL was not provided, determine correct one by workspace region that was chosen
	if len(svc.TestOptions.SchematicsApiURL) > 0 {
		schematicsURL = svc.TestOptions.SchematicsApiURL
	} else {
		if len(svc.WorkspaceLocation) > 0 {
			schematicsURL, getUrlErr = cloudinfo.GetSchematicServiceURLForRegion(svc.WorkspaceLocation)
			if getUrlErr != nil {
				return fmt.Errorf("error getting schematics URL for region %s - %w", svc.WorkspaceLocation, getUrlErr)
			}
		} else {
			schematicsURL = schematics.DefaultServiceURL
		}
	}
	svc.TestOptions.Testing.Logf("[SCHEMATICS] Schematics API for region %s: %s", svc.WorkspaceLocation, schematicsURL)

	svc.SchematicsApiSvc, err = schematics.NewSchematicsV1(&schematics.SchematicsV1Options{
		URL:           schematicsURL,
		Authenticator: svc.ApiAuthenticator,
	})
	if err != nil {
		return err
	}

	return nil
}

// CreateTestWorkspace will create a new IBM Schematics Workspace that will be used for testing.
// This is a test-specific wrapper that uses the mocked SchematicsApiSvc directly.
// For production use, consider using cloudinfo.CreateSchematicsWorkspace instead.
func (svc *SchematicsTestService) CreateTestWorkspace(name string, resourceGroup string, region string, templateFolder string, terraformVersion string, tags []string) (*schematics.WorkspaceResponse, error) {
	// initialize empty environment structures
	envValues := make([]map[string]interface{}, 0)
	envMetadata := []schematics.EnvironmentValuesMetadata{}

	// add env needed for restapi provider by default
	cloudinfo.AddWorkspaceEnv(&envValues, &envMetadata, "API_DATA_IS_SENSITIVE", "true", false, false)

	// add additional env values that were set in test options
	for _, envEntry := range svc.TestOptions.WorkspaceEnvVars {
		cloudinfo.AddWorkspaceEnv(&envValues, &envMetadata, envEntry.Key, envEntry.Value, envEntry.Hidden, envEntry.Secure)
	}

	// add netrc credentials if required
	if len(svc.TestOptions.NetrcSettings) > 0 { // pragma: allowlist secret
		// Convert NetrcCredential to cloudinfo.NetrcCredential
		netrcCreds := make([]cloudinfo.NetrcCredential, len(svc.TestOptions.NetrcSettings)) // pragma: allowlist secret
		for i, cred := range svc.TestOptions.NetrcSettings {                                // pragma: allowlist secret
			netrcCreds[i] = cloudinfo.NetrcCredential{ // pragma: allowlist secret
				Host:     cred.Host,
				Username: cred.Username,
				Password: cred.Password, // pragma: allowlist secret
			}
		}
		cloudinfo.AddNetrcToWorkspaceEnv(&envValues, &envMetadata, netrcCreds) // pragma: allowlist secret
	}

	// Use the mocked SchematicsApiSvc directly for testing
	var folder *string
	var version *string
	var wsVersion []string

	if len(templateFolder) == 0 {
		folder = core.StringPtr(".")
	} else {
		folder = core.StringPtr(templateFolder)
	}

	if len(terraformVersion) > 0 {
		version = core.StringPtr(terraformVersion)
		wsVersion = []string{terraformVersion}
	}

	templateModel := &schematics.TemplateSourceDataRequest{
		Folder:            folder,
		Type:              version,
		EnvValues:         envValues,
		EnvValuesMetadata: envMetadata,
	}

	createWorkspaceOptions := &schematics.CreateWorkspaceOptions{
		Description:   core.StringPtr("Goldeneye CI Test for " + name),
		Name:          core.StringPtr(name),
		TemplateData:  []schematics.TemplateSourceDataRequest{*templateModel},
		Type:          wsVersion,
		Location:      core.StringPtr(region),
		ResourceGroup: core.StringPtr(resourceGroup),
		Tags:          tags,
	}

	workspace, _, err := svc.SchematicsApiSvc.CreateWorkspace(createWorkspaceOptions)
	if err != nil {
		return nil, err
	}

	// set workspace and template IDs created for later use
	svc.WorkspaceID = *workspace.ID
	svc.TemplateID = *workspace.TemplateData[0].ID
	svc.WorkspaceName = *workspace.Name
	svc.TestOptions.Testing.Logf("[SCHEMATICS] Created workspace: %s (ID: %s)", *workspace.Name, *workspace.ID)

	return workspace, nil
}

// CreateUploadTarFile will create a tar file with terraform code, based on include patterns set in options.
// Returns the full tarball name that was created on local system (path included).
func (svc *SchematicsTestService) CreateUploadTarFile(projectPath string) (string, error) {
	svc.TestOptions.Testing.Log("[SCHEMATICS] Creating TAR file")
	tarballName, tarballErr := cloudinfo.CreateSchematicsTar(projectPath, svc.TestOptions.TarIncludePatterns)
	if tarballErr != nil {
		return "", fmt.Errorf("error creating tar file: %s", tarballErr.Error())
	}

	svc.TestOptions.Testing.Log("[SCHEMATICS] Uploading TAR file")
	uploadErr := svc.UploadTarToWorkspace(tarballName)
	if uploadErr != nil {
		return tarballName, fmt.Errorf("error uploading tar file to workspace: %s - %s", uploadErr.Error(), svc.WorkspaceNameForLog)
	}

	// -------- UPLOAD TAR FILE ----------
	// find the tar upload job
	uploadJob, uploadJobErr := svc.FindLatestWorkspaceJobByName(SchematicsJobTypeUpload)
	if uploadJobErr != nil {
		return tarballName, fmt.Errorf("error finding the upload tar action: %s - %s", uploadJobErr.Error(), svc.WorkspaceNameForLog)
	}
	// wait for it to finish
	uploadJobStatus, uploadJobStatusErr := svc.WaitForFinalJobStatus(*uploadJob.ActionID)
	if uploadJobStatusErr != nil {
		return tarballName, fmt.Errorf("error waiting for upload of tar to finish: %s - %s", uploadJobStatusErr.Error(), svc.WorkspaceNameForLog)
	}
	// check if complete
	if uploadJobStatus != SchematicsJobStatusCompleted {
		return tarballName, fmt.Errorf("tar upload has failed with status %s - %s", uploadJobStatus, svc.WorkspaceNameForLog)
	}

	return tarballName, nil
}

// UpdateTestTemplateVars will update an existing Schematics Workspace terraform template with a
// Variablestore, which will set terraform input variables for test runs.
func (svc *SchematicsTestService) UpdateTestTemplateVars(vars []TestSchematicTerraformVar) error {
	// set up an array of workspace variables based on TerraformVars supplied.
	var strVal string
	var strErr error
	variables := []schematics.WorkspaceVariableRequest{}
	for _, tfVar := range vars {
		// if tfVal is an array or map, convert to json string
		if common.IsCompositeType(tfVar.Value) {
			strVal, strErr = common.ConvertValueToJsonString(tfVar.Value)
			if strErr != nil {
				return strErr
			}
		} else {
			strVal = fmt.Sprintf("%v", tfVar.Value)
		}
		variables = append(variables, schematics.WorkspaceVariableRequest{
			Name:   core.StringPtr(tfVar.Name),
			Value:  core.StringPtr(strVal),
			Type:   core.StringPtr(tfVar.DataType),
			Secure: core.BoolPtr(tfVar.Secure),
		})
	}

	// Use mocked SchematicsApiSvc directly for testing
	templateModel := &schematics.ReplaceWorkspaceInputsOptions{
		WID:           core.StringPtr(svc.WorkspaceID),
		TID:           core.StringPtr(svc.TemplateID),
		Variablestore: variables,
	}

	_, _, err := svc.SchematicsApiSvc.ReplaceWorkspaceInputs(templateModel)
	if err != nil {
		return err
	}

	svc.TestOptions.Testing.Logf("[SCHEMATICS] Updated variables for workspace: %s", svc.WorkspaceID)
	return nil
}

// UploadTarToWorkspace will accept a file path for an existing TAR file, containing files for a
// Terraform test case, and upload it to an existing Schematics Workspace.
func (svc *SchematicsTestService) UploadTarToWorkspace(tarPath string) error {
	fileReader, fileErr := os.Open(tarPath)
	if fileErr != nil {
		return fmt.Errorf("error opening reader for tar path: %w", fileErr)
	}
	defer fileReader.Close()
	fileReaderWrapper := io.NopCloser(fileReader)

	uploadTarOptions := &schematics.TemplateRepoUploadOptions{
		WID:             core.StringPtr(svc.WorkspaceID),
		TID:             core.StringPtr(svc.TemplateID),
		File:            fileReaderWrapper,
		FileContentType: core.StringPtr("application/octet-stream"),
	}

	_, _, err := svc.SchematicsApiSvc.TemplateRepoUpload(uploadTarOptions)
	if err != nil {
		return err
	}

	svc.TestOptions.Testing.Logf("[SCHEMATICS] Uploaded TAR to workspace: %s", svc.WorkspaceID)
	return nil
}

// CreatePlanJob will initiate a new PLAN action on an existing terraform Schematics Workspace.
// Will return a result object containing details about the new action.
func (svc *SchematicsTestService) CreatePlanJob() (*schematics.WorkspaceActivityPlanResult, error) {
	return svc.CloudInfoService.CreateSchematicsPlanJob(
		svc.WorkspaceID,
		svc.WorkspaceLocation,
		nil, // logger - nil is acceptable, cloudinfo handles it
	)
}

// CreateApplyJob will initiate a new APPLY action on an existing terraform Schematics Workspace.
// Will return a result object containing details about the new action.
func (svc *SchematicsTestService) CreateApplyJob() (*schematics.WorkspaceActivityApplyResult, error) {
	return svc.CloudInfoService.CreateSchematicsApplyJob(
		svc.WorkspaceID,
		svc.WorkspaceLocation,
		nil, // logger - nil is acceptable, cloudinfo handles it
	)
}

// CreateDestroyJob will initiate a new DESTROY action on an existing terraform Schematics Workspace.
// Will return a result object containing details about the new action.
func (svc *SchematicsTestService) CreateDestroyJob() (*schematics.WorkspaceActivityDestroyResult, error) {
	return svc.CloudInfoService.CreateSchematicsDestroyJob(
		svc.WorkspaceID,
		svc.WorkspaceLocation,
		nil, // logger - nil is acceptable, cloudinfo handles it
	)
}

// FindLatestWorkspaceJobByName will find the latest executed job of the type supplied and return data about that job.
// This can be used to find a job by its type when the jobID is not known.
// A "NotFound" error will be thrown if there are no existing jobs of the provided type.
func (svc *SchematicsTestService) FindLatestWorkspaceJobByName(jobName string) (*schematics.WorkspaceActivity, error) {
	return svc.CloudInfoService.FindLatestSchematicsJobByName(
		svc.WorkspaceID,
		jobName,
		svc.WorkspaceLocation,
		nil, // logger - nil is acceptable, cloudinfo handles it
	)
}

// GetWorkspaceJobDetail will return a data structure with full details about an existing Schematics Workspace activity for the
// given Job ID.
func (svc *SchematicsTestService) GetWorkspaceJobDetail(jobID string) (*schematics.WorkspaceActivity, error) {
	return svc.CloudInfoService.GetSchematicsWorkspaceJobDetail(
		svc.WorkspaceID,
		jobID,
		svc.WorkspaceLocation,
		nil, // logger - nil is acceptable, cloudinfo handles it
	)
}

// WaitForFinalJobStatus will look up details about the given activity and check the status value. If the status implies that the activity
// has not completed yet, this function will keep checking the status value until either the activity has finished, or a configured time threshold has
// been reached.
// Returns the final status value of the activity when it has finished.
// Returns an error if the activity does not finish before the configured time threshold.
func (svc *SchematicsTestService) WaitForFinalJobStatus(jobID string) (string, error) {
	waitMinutes := DefaultWaitJobCompleteMinutes
	if svc.TestOptions != nil && svc.TestOptions.WaitJobCompleteMinutes > 0 {
		waitMinutes = svc.TestOptions.WaitJobCompleteMinutes
	}

	return svc.CloudInfoService.WaitForSchematicsJobCompletion(
		svc.WorkspaceID,
		jobID,
		svc.WorkspaceLocation,
		int(waitMinutes),
		nil, // logger - nil is acceptable, cloudinfo handles it
	)
}

// GetLatestWorkspaceOutputs will return a map of current terraform outputs stored in the workspace
func (svc *SchematicsTestService) GetLatestWorkspaceOutputs() (map[string]interface{}, error) {
	return svc.CloudInfoService.GetSchematicsWorkspaceOutputs(
		svc.WorkspaceID,
		svc.WorkspaceLocation,
		nil, // logger - nil is acceptable, cloudinfo handles it
	)
}

// DeleteWorkspace will delete the existing workspace created for the test service.
func (svc *SchematicsTestService) DeleteWorkspace() (string, error) {
	return svc.CloudInfoService.DeleteSchematicsWorkspace(
		svc.WorkspaceID,
		svc.WorkspaceLocation,
		false, // destroyResources = false
		nil,   // logger - nil is acceptable, cloudinfo handles it
	)
}

// variable validation function for validating if some variable is passed to test which is not
// declared in variables.tf. Currently schematics does not fail the test in such a case where
// normal terraform run would give an error saying passed variable does not exist in variables.tf file
func (svc *SchematicsTestService) validateVariables(terraformDir string) error {

	entries, err := os.ReadDir(terraformDir)
	if err != nil {
		return fmt.Errorf("error reading directory: %v", err)
	}

	re := regexp.MustCompile(`variable\s+"([^"]+)"`)
	declaredVars := []string{}

	for _, entry := range entries { // loop all .tf files in terraform directory
		if entry.IsDir() {
			continue // Skip directories
		}

		if strings.HasSuffix(entry.Name(), ".tf") {
			filePath := filepath.Join(terraformDir, entry.Name())

			content, err := os.ReadFile(filePath)
			if err != nil {
				return fmt.Errorf("error reading file %s: %v", filePath, err)
			}

			matches := re.FindAllStringSubmatch(string(content), -1)
			for _, match := range matches {
				if len(match) > 1 {
					declaredVars = append(declaredVars, match[1])
				}
			}
		}
	}

	optionVars := svc.TestOptions.TerraformVars
	passedVars := make([]string, 0)

	for _, varInfo := range optionVars {

		passedVars = append(passedVars, varInfo.Name)
	}

	extraVariables := make([]string, 0)
	// check if there is some variable passed to the test but is not declared in variables.tf

	for _, passedVar := range passedVars {

		found := false

		for _, declaredVar := range declaredVars {
			if passedVar == declaredVar {
				found = true
				break
			}
		}

		if !found {

			extraVariables = append(extraVariables, passedVar)
		}
	}
	if len(extraVariables) > 0 {

		vars := strings.Join(extraVariables, ", ")
		return fmt.Errorf("variable [%s] passed in test but not declared in variables.tf", vars)

	}
	return nil

}
