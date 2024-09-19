// Package testschematic contains functions that can be used to assist and standardize the execution of unit tests for IBM Cloud Terraform projects
// by using the IBM Cloud Schematics service
package testschematic

import (
	"archive/tar"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/IBM/go-sdk-core/v5/core"
	schematics "github.com/IBM/schematics-go-sdk/schematicsv1"
	"github.com/go-openapi/errors"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/common"
)

// IBM schematics job types
const SchematicsJobTypeUpload = "TAR_WORKSPACE_UPLOAD"
const SchematicsJobTypePlan = "PLAN"
const SchematicsJobTypeApply = "APPLY"
const SchematicsJobTypeDestroy = "DESTROY"

// IBM schematics job status
const SchematicsJobStatusCompleted = "COMPLETED"
const SchematicsJobStatusFailed = "FAILED"
const SchematicsJobStatusCreated = "CREATED"
const SchematicsJobStatusInProgress = "INPROGRESS"

// Defaults for API retry mechanic
const defaultApiRetryCount int = 5
const defaultApiRetryWaitSeconds int = 10

// golang does not support constant array/slice, this is our constant
func getApiRetryStatusExceptions() []int {
	return []int{401, 403}
}

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
	SchematicsApiSvc          SchematicsApiSvcI     // the main schematics service interface
	ApiAuthenticator          IamAuthenticatorSvcI  // the authenticator used for schematics api calls
	WorkspaceID               string                // workspace ID used for tests
	WorkspaceName             string                // name of workspace that was created for test
	TemplateID                string                // workspace template ID used for tests
	TestOptions               *TestSchematicOptions // additional testing options
	TerraformTestStarted      bool                  // keeps track of when actual Terraform resource testing has begin, used for proper test teardown logic
	TerraformResourcesCreated bool                  // keeps track of when we start deploying resources, used for proper test teardown logic
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
	svc.SchematicsApiSvc, err = schematics.NewSchematicsV1(&schematics.SchematicsV1Options{
		URL:           svc.TestOptions.SchematicsApiURL,
		Authenticator: svc.ApiAuthenticator,
	})
	if err != nil {
		return err
	}

	return nil
}

// CreateTestWorkspace will create a new IBM Schematics Workspace that will be used for testing.
func (svc *SchematicsTestService) CreateTestWorkspace(name string, resourceGroup string, templateFolder string, terraformVersion string, tags []string) (*schematics.WorkspaceResponse, error) {

	var folder *string
	var version *string
	var wsVersion []string
	// choose nil default for version if not supplied, so that they omit from template setup
	// (schematics should then determine defaults)
	if len(templateFolder) == 0 {
		folder = core.StringPtr(".")
	} else {
		folder = core.StringPtr(templateFolder)
	}

	if len(terraformVersion) > 0 {
		version = core.StringPtr(terraformVersion)
		wsVersion = []string{terraformVersion}
	}

	// initialize empty environment structures
	envValues := make([]map[string]interface{}, 0)
	envMetadata := []schematics.EnvironmentValuesMetadata{}

	// add env needed for restapi provider by default
	addWorkspaceEnv(&envValues, &envMetadata, "API_DATA_IS_SENSITIVE", "true", false, false)

	// add additional env values that were set in test options
	for _, envEntry := range svc.TestOptions.WorkspaceEnvVars {
		addWorkspaceEnv(&envValues, &envMetadata, envEntry.Key, envEntry.Value, envEntry.Hidden, envEntry.Secure)
	}

	// add netrc credientials if required
	if len(svc.TestOptions.NetrcSettings) > 0 {
		addNetrcToWorkspaceEnv(&envValues, &envMetadata, svc.TestOptions.NetrcSettings)
	}

	// create env and input vars template
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
		Location:      core.StringPtr(defaultRegion),
		ResourceGroup: core.StringPtr(resourceGroup),
		Tags:          tags,
	}

	var workspace *schematics.WorkspaceResponse
	var resp *core.DetailedResponse
	var workspaceErr error
	retries := 0
	for {
		workspace, resp, workspaceErr = svc.SchematicsApiSvc.CreateWorkspace(createWorkspaceOptions)
		if svc.retryApiCall(workspaceErr, resp.StatusCode, retries) {
			retries++
			svc.TestOptions.Testing.Logf("[SCHEMATICS] RETRY CreateWorkspace, status code: %d", resp.StatusCode)
		} else {
			break
		}
	}

	if workspaceErr != nil {
		return nil, workspaceErr
	}

	// set workspace and template IDs created for later use
	svc.WorkspaceID = *workspace.ID
	svc.TemplateID = *workspace.TemplateData[0].ID
	svc.WorkspaceName = *workspace.Name

	return workspace, nil
}

// UpdateTestTemplateVars will update an existing Schematics Workspace terraform template with a
// Variablestore, which will set terraform input variables for test runs.
func (svc *SchematicsTestService) UpdateTestTemplateVars(vars []TestSchematicTerraformVar) error {

	// set up an array of workspace variables based on TerraformVars supplied.
	var strVal string
	var strErr error
	variables := []schematics.WorkspaceVariableRequest{}
	for _, tfVar := range vars {
		// if tfVal is an array, convert to json array string
		if common.IsArray(tfVar.Value) {
			strVal, strErr = common.ConvertArrayToJsonString(tfVar.Value)
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

	templateModel := &schematics.ReplaceWorkspaceInputsOptions{
		WID:           core.StringPtr(svc.WorkspaceID),
		TID:           core.StringPtr(svc.TemplateID),
		Variablestore: variables,
	}

	// now update template
	var resp *core.DetailedResponse
	var updateErr error
	retries := 0
	for {
		_, resp, updateErr = svc.SchematicsApiSvc.ReplaceWorkspaceInputs(templateModel)
		if svc.retryApiCall(updateErr, resp.StatusCode, retries) {
			retries++
			svc.TestOptions.Testing.Logf("[SCHEMATICS] RETRY ReplaceWorkspaceInputs, status code: %d", resp.StatusCode)
		} else {
			break
		}
	}

	if updateErr != nil {
		return updateErr
	}

	return nil
}

// UploadTarToWorkspace will accept a file path for an existing TAR file, containing files for a
// Terraform test case, and upload it to an existing Schematics Workspace.
func (svc *SchematicsTestService) UploadTarToWorkspace(tarPath string) error {
	fileReader, fileErr := os.Open(tarPath)
	if fileErr != nil {
		return fmt.Errorf("error opening reader for tar path: %w", fileErr)
	}
	fileReaderWrapper := io.NopCloser(fileReader)

	uploadTarOptions := &schematics.TemplateRepoUploadOptions{
		WID:             core.StringPtr(svc.WorkspaceID),
		TID:             core.StringPtr(svc.TemplateID),
		File:            fileReaderWrapper,
		FileContentType: core.StringPtr("application/octet-stream"),
	}

	var resp *core.DetailedResponse
	var uploadErr error
	retries := 0
	for {
		_, resp, uploadErr = svc.SchematicsApiSvc.TemplateRepoUpload(uploadTarOptions)
		if svc.retryApiCall(uploadErr, resp.StatusCode, retries) {
			retries++
			svc.TestOptions.Testing.Logf("[SCHEMATICS] RETRY TemplateRepoUpload, status code: %d", resp.StatusCode)
		} else {
			break
		}
	}
	if uploadErr != nil {
		return uploadErr
	}

	return nil
}

// CreatePlanJob will initiate a new PLAN action on an existing terraform Schematics Workspace.
// Will return a result object containing details about the new action.
func (svc *SchematicsTestService) CreatePlanJob() (*schematics.WorkspaceActivityPlanResult, error) {
	refreshToken, tokenErr := svc.GetRefreshToken()
	if tokenErr != nil {
		return nil, tokenErr
	}

	var planResult *schematics.WorkspaceActivityPlanResult
	var resp *core.DetailedResponse
	var err error
	retries := 0
	for {
		planResult, resp, err = svc.SchematicsApiSvc.PlanWorkspaceCommand(&schematics.PlanWorkspaceCommandOptions{
			WID:          core.StringPtr(svc.WorkspaceID),
			RefreshToken: core.StringPtr(refreshToken),
		})
		if svc.retryApiCall(err, resp.StatusCode, retries) {
			retries++
			svc.TestOptions.Testing.Logf("[SCHEMATICS] RETRY PlanWorkspaceCommand, status code: %d", resp.StatusCode)
		} else {
			break
		}
	}
	if err != nil {
		return nil, err
	}

	return planResult, nil
}

// CreateApplyJob will initiate a new APPLY action on an existing terraform Schematics Workspace.
// Will return a result object containing details about the new action.
func (svc *SchematicsTestService) CreateApplyJob() (*schematics.WorkspaceActivityApplyResult, error) {
	refreshToken, tokenErr := svc.GetRefreshToken()
	if tokenErr != nil {
		return nil, tokenErr
	}

	var applyResult *schematics.WorkspaceActivityApplyResult
	var resp *core.DetailedResponse
	var err error
	retries := 0
	for {
		applyResult, resp, err = svc.SchematicsApiSvc.ApplyWorkspaceCommand(&schematics.ApplyWorkspaceCommandOptions{
			WID:          core.StringPtr(svc.WorkspaceID),
			RefreshToken: core.StringPtr(refreshToken),
		})
		if svc.retryApiCall(err, resp.StatusCode, retries) {
			retries++
			svc.TestOptions.Testing.Logf("[SCHEMATICS] RETRY ApplyWorkspaceCommand, status code: %d", resp.StatusCode)
		} else {
			break
		}
	}
	if err != nil {
		return nil, err
	}

	return applyResult, nil
}

// CreateDestroyJob will initiate a new DESTROY action on an existing terraform Schematics Workspace.
// Will return a result object containing details about the new action.
func (svc *SchematicsTestService) CreateDestroyJob() (*schematics.WorkspaceActivityDestroyResult, error) {
	refreshToken, tokenErr := svc.GetRefreshToken()
	if tokenErr != nil {
		return nil, tokenErr
	}

	var destroyResult *schematics.WorkspaceActivityDestroyResult
	var resp *core.DetailedResponse
	var err error
	retries := 0
	for {
		destroyResult, resp, err = svc.SchematicsApiSvc.DestroyWorkspaceCommand(&schematics.DestroyWorkspaceCommandOptions{
			WID:          core.StringPtr(svc.WorkspaceID),
			RefreshToken: core.StringPtr(refreshToken),
		})
		if svc.retryApiCall(err, resp.StatusCode, retries) {
			retries++
			svc.TestOptions.Testing.Logf("[SCHEMATICS] RETRY DestroyWorkspaceCommand, status code: %d", resp.StatusCode)
		} else {
			break
		}
	}
	if err != nil {
		return nil, err
	}

	return destroyResult, nil
}

// FindLatestWorkspaceJobByName will find the latest executed job of the type supplied and return data about that job.
// This can be used to find a job by its type when the jobID is not known.
// A "NotFound" error will be thrown if there are no existing jobs of the provided type.
func (svc *SchematicsTestService) FindLatestWorkspaceJobByName(jobName string) (*schematics.WorkspaceActivity, error) {

	// get array of jobs using workspace id
	var listResult *schematics.WorkspaceActivities
	var resp *core.DetailedResponse
	var listErr error
	retries := 0
	for {
		listResult, resp, listErr = svc.SchematicsApiSvc.ListWorkspaceActivities(&schematics.ListWorkspaceActivitiesOptions{
			WID: core.StringPtr(svc.WorkspaceID),
		})
		if svc.retryApiCall(listErr, resp.StatusCode, retries) {
			retries++
			svc.TestOptions.Testing.Logf("[SCHEMATICS] RETRY ListWorkspaceActivities, status code: %d", resp.StatusCode)
		} else {
			break
		}
	}

	if listErr != nil {
		return nil, listErr
	}

	// loop through jobs and get latest one that matches name
	var jobResult *schematics.WorkspaceActivity
	for i, job := range listResult.Actions {
		// only match name
		if *job.Name == jobName {
			// keep latest job of svc name
			if jobResult != nil {
				if time.Time(*job.PerformedAt).After(time.Time(*jobResult.PerformedAt)) {
					jobResult = &listResult.Actions[i]
				}
			} else {
				jobResult = &listResult.Actions[i]
			}
		}
	}

	// if jobResult is nil then none were found, throw error
	if jobResult == nil {
		return nil, errors.NotFound("job <%s> not found in workspace", jobName)
	}

	return jobResult, nil
}

// GetWorkspaceJobDetail will return a data structure with full details about an existing Schematics Workspace activity for the
// given Job ID.
func (svc *SchematicsTestService) GetWorkspaceJobDetail(jobID string) (*schematics.WorkspaceActivity, error) {

	// look up job by ID
	var activityResponse *schematics.WorkspaceActivity
	var resp *core.DetailedResponse
	var err error
	retries := 0
	for {
		activityResponse, resp, err = svc.SchematicsApiSvc.GetWorkspaceActivity(&schematics.GetWorkspaceActivityOptions{
			WID:        core.StringPtr(svc.WorkspaceID),
			ActivityID: core.StringPtr(jobID),
		})
		if svc.retryApiCall(err, resp.StatusCode, retries) {
			retries++
			svc.TestOptions.Testing.Logf("[SCHEMATICS] RETRY GetWorkspaceActivity, status code: %d", resp.StatusCode)
		} else {
			break
		}
	}

	if err != nil {
		return nil, err
	}

	return activityResponse, nil
}

// WaitForFinalJobStatus will look up details about the given activity and check the status value. If the status implies that the activity
// has not completed yet, this function will keep checking the status value until either the activity has finished, or a configured time threshold has
// been reached.
// Returns the final status value of the activity when it has finished.
// Returns an error if the activity does not finish before the configured time threshold.
func (svc *SchematicsTestService) WaitForFinalJobStatus(jobID string) (string, error) {
	var status string
	var job *schematics.WorkspaceActivity
	var jobErr error

	// Wait for the job to be complete
	start := time.Now()
	lastLog := int16(0)
	runMinutes := int16(0)
	waitMinutes := DefaultWaitJobCompleteMinutes
	if svc.TestOptions != nil && svc.TestOptions.WaitJobCompleteMinutes > 0 {
		waitMinutes = svc.TestOptions.WaitJobCompleteMinutes
	}

	for {
		// check for timeout and throw error
		runMinutes = int16(time.Since(start).Minutes())
		if runMinutes > waitMinutes {
			return "", fmt.Errorf("time exceeded waiting for schematic job to finish")
		}

		// get details of job
		job, jobErr = svc.GetWorkspaceJobDetail(jobID)
		if jobErr != nil {
			return "", jobErr
		}
		// only log svc once a minute or so
		if runMinutes > lastLog {
			svc.TestOptions.Testing.Logf("[SCHEMATICS] ... still waiting for job %s to complete: %d minutes", *job.Name, runMinutes)
			lastLog = runMinutes
		}

		// check if it is finished
		if job.Status != nil &&
			len(*job.Status) > 0 &&
			*job.Status != SchematicsJobStatusCreated &&
			*job.Status != SchematicsJobStatusInProgress {
			svc.TestOptions.Testing.Logf("[SCHEMATICS] The status of job %s is: %s", *job.Name, *job.Status)
			break
		}

		// wait 60 seconds
		time.Sleep(60 * time.Second)
	}

	// if we reach svc point the job has finished, return status
	status = *job.Status

	return status, nil
}

// DeleteWorkspace will delete the existing workspace created for the test service.
func (svc *SchematicsTestService) DeleteWorkspace() (string, error) {

	refreshToken, tokenErr := svc.GetRefreshToken()
	if tokenErr != nil {
		return "", tokenErr
	}

	var result *string
	var resp *core.DetailedResponse
	var err error
	retries := 0
	for {
		result, resp, err = svc.SchematicsApiSvc.DeleteWorkspace(&schematics.DeleteWorkspaceOptions{
			WID:              core.StringPtr(svc.WorkspaceID),
			RefreshToken:     core.StringPtr(refreshToken),
			DestroyResources: core.StringPtr("false"),
		})
		if svc.retryApiCall(err, resp.StatusCode, retries) {
			retries++
			svc.TestOptions.Testing.Logf("[SCHEMATICS] RETRY DeleteWorkspace, status code: %d", resp.StatusCode)
		} else {
			break
		}
	}

	if err != nil {
		return "", fmt.Errorf("delete of schematic job failed: %w", err)
	}

	return *result, nil
}

// CreateSchematicTar will accept a path to a Terraform project and an array of file patterns to include,
// and will create a TAR file in a temporary location that contains all of the project's files that match the
// supplied file patterns. This TAR file can then be uploaded to a Schematics Workspace template.
// Returns a string of the complete TAR file path and file name.
// Error is returned if any issues happen while creating TAR file.
func CreateSchematicTar(projectPath string, includePatterns *[]string) (string, error) {

	// create unique tar filename
	target := filepath.Join(os.TempDir(), fmt.Sprintf("schematic-test-%s.tar", strings.ToLower(random.UniqueId())))

	// set up tarfile on filesystem
	tarfile, fileErr := os.Create(target)
	if fileErr != nil {
		return "", fileErr
	}
	defer tarfile.Close()

	// create a tar file writer
	tw := tar.NewWriter(tarfile)
	defer tw.Close()

	// track files added
	totalFiles := 0
	// track directories added, we only want to add them once
	dirsAdded := []string{}

	// start loop through provided list of patterns
	// if none provided, assume just terraform files
	if len(*includePatterns) == 0 {
		includePatterns = &[]string{"*.tf"}
	}

	// schematics needs an outer folder in the tar file to contain everything, create that here and add this to everything later
	// use current head of project path directory for dir info
	parentDirInfo, parentDirInfoErr := os.Stat(projectPath)
	if parentDirInfoErr != nil {
		return "", parentDirInfoErr
	}
	parentDirHdr, parentDirHdrErr := tar.FileInfoHeader(parentDirInfo, parentDirInfo.Name())
	if parentDirHdrErr != nil {
		return "", parentDirHdrErr
	}
	if tarWriteOuterDirErr := tw.WriteHeader(parentDirHdr); tarWriteOuterDirErr != nil {
		return "", tarWriteOuterDirErr
	}

	for _, pattern := range *includePatterns {
		// for glob search, use full path plus pattern
		patternPath := filepath.Join(projectPath, pattern)
		files, _ := filepath.Glob(patternPath)

		// loop through files
		for _, fileName := range files {
			// get file info
			info, infoErr := os.Stat(fileName)
			if infoErr != nil {
				return "", infoErr
			}
			// keep the full path to file, and a relative path based on project path
			// full path used for info lookups, relative is used for inside TAR file
			fullFileDir := filepath.Dir(fileName)
			relFileDir, relFileDirErr := filepath.Rel(projectPath, fullFileDir)
			if relFileDirErr != nil {
				return "", relFileDirErr
			}

			// skip directories that were directly found by the Glob, we will add those another way (see below)
			if info.IsDir() {
				continue
			}

			hdr, hdrErr := tar.FileInfoHeader(info, info.Name())
			if hdrErr != nil {
				return "", hdrErr
			}

			// the FI header sets the name as base name only, so to preserve the leading relative directories (if needed)
			// we will alter the name
			if relFileDir != "." {
				hdr.Name = filepath.Join(parentDirInfo.Name(), relFileDir, hdr.Name)
				hdr.Linkname = hdr.Name

				// if the file resides in subdirectory, add that directory to tar file so that extraction works correctly
				if !common.StrArrayContains(dirsAdded, relFileDir) {
					dirInfo, dirInfoErr := os.Stat(fullFileDir)
					if dirInfoErr != nil {
						return "", dirInfoErr
					}
					dirHdr, dirHdrErr := tar.FileInfoHeader(dirInfo, filepath.Join(parentDirInfo.Name(), relFileDir))
					if dirHdrErr != nil {
						return "", dirHdrErr
					}
					dirHdr.Name = filepath.Join(parentDirInfo.Name(), relFileDir) // use full realative path
					if tarWriteDirErr := tw.WriteHeader(dirHdr); tarWriteDirErr != nil {
						return "", tarWriteDirErr
					}
					dirsAdded = append(dirsAdded, relFileDir)
				}
			} else {
				// file is at root level, put it right below parent dir
				hdr.Name = filepath.Join(parentDirInfo.Name(), hdr.Name)
				hdr.Linkname = hdr.Name
			}

			// prefer GNU format
			hdr.Format = tar.FormatGNU

			// start writing to tarball
			if tarWriteErr := tw.WriteHeader(hdr); tarWriteErr != nil {
				return "", tarWriteErr
			}

			// now open file and copy contents to tarball
			file, fileErr := os.Open(fileName)
			if fileErr != nil {
				return "", fileErr
			}
			defer file.Close()
			_, writeErr := io.Copy(tw, file)
			if writeErr != nil {
				return "", writeErr
			}

			// keep track of files added
			totalFiles = totalFiles + 1
		}
	}

	// if there were zero files added to the tar we need to error, as it will be empty
	// also just delete the file, we don't want it hanging around
	if totalFiles == 0 {
		defer os.Remove(target)
		return "", fmt.Errorf("tar file is empty, no files added")
	}

	return target, nil
}

func addWorkspaceEnv(values *[]map[string]interface{}, metadata *[]schematics.EnvironmentValuesMetadata, key string, value string, hidden bool, secure bool) {
	// Create a map of type map[string]interface{} to store key-value pair
	envValue := map[string]interface{}{key: value}

	// Append the map to the values slice
	*values = append(*values, envValue)

	// Add a metadata entry for the sensitive value
	*metadata = append(*metadata, schematics.EnvironmentValuesMetadata{
		Name:   core.StringPtr(key),
		Hidden: core.BoolPtr(hidden),
		Secure: core.BoolPtr(secure),
	})
}

func addNetrcToWorkspaceEnv(values *[]map[string]interface{}, metadata *[]schematics.EnvironmentValuesMetadata, netrcEntries []NetrcCredential) {
	// Create a slice to store netrc entries
	netrcValue := [][]string{}

	// Loop through provided entries and add to the slice
	for _, netrc := range netrcEntries {
		entry := []string{
			netrc.Host,
			netrc.Username,
			netrc.Password,
		}
		netrcValue = append(netrcValue, entry)
	}

	// turn entire array into string
	netrcValueStr, _ := common.ConvertArrayToJsonString(netrcValue)
	// Add the slice of netrc entries to env with "__netrc__" as the key
	*values = append(*values, map[string]interface{}{"__netrc__": netrcValueStr})

	// Add a metadata entry for the sensitive value
	*metadata = append(*metadata, schematics.EnvironmentValuesMetadata{
		Name:   core.StringPtr("__netrc__"),
		Hidden: core.BoolPtr(false), // Set to false as it's not hidden
		Secure: core.BoolPtr(true),  // Set to true as it's considered secure
	})
}

func (svc *SchematicsTestService) retryApiCall(apiError error, apiStatusCode int, currentRetryCount int) bool {
	// set up defaults
	maxRetries := defaultApiRetryCount
	maxWait := defaultApiRetryWaitSeconds

	// override defaults if provided
	if svc.TestOptions.SchematicSvcRetryCount != nil {
		maxRetries = *svc.TestOptions.SchematicSvcRetryCount
	}
	if svc.TestOptions.SchematicSvcRetryWaitSeconds != nil {
		maxWait = *svc.TestOptions.SchematicSvcRetryWaitSeconds
	}

	// if we are at our max retry count, do not retry
	if currentRetryCount >= maxRetries {
		return false
	}

	// if no error was returned, or if it was and we should ignore status code, do not retry
	if apiError == nil {
		return false
	} else {
		if common.IntArrayContains(getApiRetryStatusExceptions(), apiStatusCode) {
			return false
		}
	}

	// wait and retry
	time.Sleep(time.Duration(maxWait) * time.Second)
	return true
}
