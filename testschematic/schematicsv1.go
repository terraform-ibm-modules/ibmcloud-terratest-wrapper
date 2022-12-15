package testschematic

import (
	"archive/tar"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/schematics-go-sdk/schematicsv1"
	"github.com/go-openapi/errors"
	"github.com/gruntwork-io/terratest/modules/random"
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

// interface for the external schematics service api. Can be mocked for tests
type SchematicsApiSvcI interface {
	CreateWorkspace(createWorkspaceOptions *schematicsv1.CreateWorkspaceOptions) (result *schematicsv1.WorkspaceResponse, response *core.DetailedResponse, err error)
	DeleteWorkspace(deleteWorkspaceOptions *schematicsv1.DeleteWorkspaceOptions) (result *string, response *core.DetailedResponse, err error)
	TemplateRepoUpload(templateRepoUploadOptions *schematicsv1.TemplateRepoUploadOptions) (result *schematicsv1.TemplateRepoTarUploadResponse, response *core.DetailedResponse, err error)
	ReplaceWorkspaceInputs(replaceWorkspaceInputsOptions *schematicsv1.ReplaceWorkspaceInputsOptions) (result *schematicsv1.UserValues, response *core.DetailedResponse, err error)
	ListWorkspaceActivities(listWorkspaceActivitiesOptions *schematicsv1.ListWorkspaceActivitiesOptions) (result *schematicsv1.WorkspaceActivities, response *core.DetailedResponse, err error)
	GetWorkspaceActivity(getWorkspaceActivityOptions *schematicsv1.GetWorkspaceActivityOptions) (result *schematicsv1.WorkspaceActivity, response *core.DetailedResponse, err error)
	PlanWorkspaceCommand(planWorkspaceCommandOptions *schematicsv1.PlanWorkspaceCommandOptions) (result *schematicsv1.WorkspaceActivityPlanResult, response *core.DetailedResponse, err error)
	ApplyWorkspaceCommand(applyWorkspaceCommandOptions *schematicsv1.ApplyWorkspaceCommandOptions) (result *schematicsv1.WorkspaceActivityApplyResult, response *core.DetailedResponse, err error)
	DestroyWorkspaceCommand(destroyWorkspaceCommandOptions *schematicsv1.DestroyWorkspaceCommandOptions) (result *schematicsv1.WorkspaceActivityDestroyResult, response *core.DetailedResponse, err error)
}

type SchematicsTestService struct {
	SchematicsApiSvc          SchematicsApiSvcI            // the main schematics service interface
	ApiAuthenticator          *core.IamAuthenticator       // the authenticator used for schematics api calls
	SchematicsIamToken        *core.IamTokenServerResponse // needed for refresh_token, for schematic plan/apply/destroy API calls
	WorkspaceID               string                       // workspace ID used for tests
	TemplateID                string                       // workspace template ID used for tests
	TestOptions               *TestSchematicOptions        // additional testing options
	TerraformTestStarted      bool                         // keeps track of when actual Terraform resource testing has begin, used for proper test teardown logic
	TerraformResourcesCreated bool                         // keeps track of when we start deploying resources, used for proper test teardown logic
}

func (svc *SchematicsTestService) CreateAuthenticator(ibmcloudApiKey string) {
	svc.ApiAuthenticator = &core.IamAuthenticator{
		ApiKey: ibmcloudApiKey, // pragma: allowlist secret
		// the user bx:bx is required here for all IAM calls so that a refresh_token is returned, see here: https://cloud.ibm.com/apidocs/schematics/schematics#apply-workspace-command
		ClientId:     "bx", // pragma: allowlist secret
		ClientSecret: "bx", // pragma: allowlist secret
	}
}

func (svc *SchematicsTestService) GetRefreshToken() (string, error) {
	if svc.SchematicsIamToken == nil || len(svc.SchematicsIamToken.RefreshToken) == 0 {
		var err error
		svc.SchematicsIamToken, err = svc.ApiAuthenticator.RequestToken()
		if err != nil {
			return "", err
		}
	}

	return svc.SchematicsIamToken.RefreshToken, nil
}

func (svc *SchematicsTestService) InitializeSchematicsService() error {
	var err error
	svc.SchematicsApiSvc, err = schematicsv1.NewSchematicsV1(&schematicsv1.SchematicsV1Options{
		URL:           svc.TestOptions.SchematicsApiURL,
		Authenticator: svc.ApiAuthenticator,
	})
	if err != nil {
		return err
	}

	return nil
}

func (svc *SchematicsTestService) CreateTestWorkspace(name string, resourceGroup string, tags []string) (*schematicsv1.WorkspaceResponse, error) {

	// create env and input vars template
	templateModel := &schematicsv1.TemplateSourceDataRequest{
		Folder: core.StringPtr("."),
		Type:   core.StringPtr("terraform_v1.2"),
		EnvValues: []interface{}{
			map[string]string{
				"__netrc__": fmt.Sprintf("[['github.ibm.com','%s','%s']]", svc.TestOptions.RequiredEnvironmentVars[gitUser], svc.TestOptions.RequiredEnvironmentVars[gitToken]),
			},
			map[string]string{
				"API_DATA_IS_SENSITIVE": "true", // for RestAPI provider
			},
		},
		EnvValuesMetadata: []schematicsv1.EnvironmentValuesMetadata{
			{Name: core.StringPtr("__netrc__"), Hidden: core.BoolPtr(false), Secure: core.BoolPtr(true)},
			{Name: core.StringPtr("API_DATA_IS_SENSITIVE"), Hidden: core.BoolPtr(false), Secure: core.BoolPtr(false)},
		},
	}

	createWorkspaceOptions := &schematicsv1.CreateWorkspaceOptions{
		Description:   core.StringPtr("Goldeneye CI Test for " + name),
		Name:          core.StringPtr(name),
		TemplateData:  []schematicsv1.TemplateSourceDataRequest{*templateModel},
		Type:          []string{"terraform_v1.2"},
		Location:      core.StringPtr(defaultRegion),
		ResourceGroup: core.StringPtr(resourceGroup),
		Tags:          tags,
	}

	workspace, _, workspaceErr := svc.SchematicsApiSvc.CreateWorkspace(createWorkspaceOptions)
	if workspaceErr != nil {
		return nil, workspaceErr
	}

	// set workspace and template IDs created for later use
	svc.WorkspaceID = *workspace.ID
	svc.TemplateID = *workspace.TemplateData[0].ID

	return workspace, nil
}

func (svc *SchematicsTestService) UpdateTestTemplateVars(vars []TestSchematicTerraformVar) error {

	// set up an array of workspace variables based on TerraformVars supplied.
	var strVal string
	var strErr error
	variables := []schematicsv1.WorkspaceVariableRequest{}
	for _, tfVar := range vars {
		// if tfVal is an array, convert to json array string
		if IsArray(tfVar.Value) {
			strVal, strErr = ConvertArrayToJsonString(tfVar.Value)
			if strErr != nil {
				return strErr
			}
		} else {
			strVal = fmt.Sprintf("%v", tfVar.Value)
		}
		variables = append(variables, schematicsv1.WorkspaceVariableRequest{
			Name:   core.StringPtr(tfVar.Name),
			Value:  core.StringPtr(strVal),
			Type:   core.StringPtr(tfVar.DataType),
			Secure: core.BoolPtr(tfVar.Secure),
		})
	}

	templateModel := &schematicsv1.ReplaceWorkspaceInputsOptions{
		WID:           core.StringPtr(svc.WorkspaceID),
		TID:           core.StringPtr(svc.TemplateID),
		Variablestore: variables,
	}

	// now update template
	_, _, updateErr := svc.SchematicsApiSvc.ReplaceWorkspaceInputs(templateModel)
	if updateErr != nil {
		return updateErr
	}

	return nil
}

func (svc *SchematicsTestService) UploadTarToWorkspace(tarPath string) error {
	fileReader, _ := os.Open(tarPath)
	fileReaderWrapper := io.NopCloser(fileReader)

	uploadTarOptions := &schematicsv1.TemplateRepoUploadOptions{
		WID:             core.StringPtr(svc.WorkspaceID),
		TID:             core.StringPtr(svc.TemplateID),
		File:            fileReaderWrapper,
		FileContentType: core.StringPtr("application/octet-stream"),
	}

	_, _, uploadErr := svc.SchematicsApiSvc.TemplateRepoUpload(uploadTarOptions)
	if uploadErr != nil {
		return uploadErr
	}

	return nil
}

func (svc *SchematicsTestService) CreatePlanJob() (*schematicsv1.WorkspaceActivityPlanResult, error) {
	refreshToken, tokenErr := svc.GetRefreshToken()
	if tokenErr != nil {
		return nil, tokenErr
	}

	planResult, _, err := svc.SchematicsApiSvc.PlanWorkspaceCommand(&schematicsv1.PlanWorkspaceCommandOptions{
		WID:          core.StringPtr(svc.WorkspaceID),
		RefreshToken: core.StringPtr(refreshToken),
	})
	if err != nil {
		return nil, err
	}

	return planResult, nil
}

func (svc *SchematicsTestService) CreateApplyJob() (*schematicsv1.WorkspaceActivityApplyResult, error) {
	refreshToken, tokenErr := svc.GetRefreshToken()
	if tokenErr != nil {
		return nil, tokenErr
	}

	applyResult, _, err := svc.SchematicsApiSvc.ApplyWorkspaceCommand(&schematicsv1.ApplyWorkspaceCommandOptions{
		WID:          core.StringPtr(svc.WorkspaceID),
		RefreshToken: core.StringPtr(refreshToken),
	})
	if err != nil {
		return nil, err
	}

	return applyResult, nil
}

func (svc *SchematicsTestService) CreateDestroyJob() (*schematicsv1.WorkspaceActivityDestroyResult, error) {
	refreshToken, tokenErr := svc.GetRefreshToken()
	if tokenErr != nil {
		return nil, tokenErr
	}

	destroyResult, _, err := svc.SchematicsApiSvc.DestroyWorkspaceCommand(&schematicsv1.DestroyWorkspaceCommandOptions{
		WID:          core.StringPtr(svc.WorkspaceID),
		RefreshToken: core.StringPtr(refreshToken),
	})
	if err != nil {
		return nil, err
	}

	return destroyResult, nil
}

func (svc *SchematicsTestService) FindLatestWorkspaceJobByName(jobName string) (*schematicsv1.WorkspaceActivity, error) {

	// get array of jobs using workspace id
	listResult, _, listErr := svc.SchematicsApiSvc.ListWorkspaceActivities(&schematicsv1.ListWorkspaceActivitiesOptions{
		WID: core.StringPtr(svc.WorkspaceID),
	})
	if listErr != nil {
		return nil, listErr
	}

	// loop through jobs and get latest one that matches name
	var jobResult *schematicsv1.WorkspaceActivity
	for _, job := range listResult.Actions {
		// only match name
		if *job.Name == jobName {
			// keep latest job of svc name
			if jobResult != nil {
				if time.Time(*job.PerformedAt).After(time.Time(*jobResult.PerformedAt)) {
					jobResult = &job
				}
			} else {
				jobResult = &job
			}
		}
	}

	// if jobResult is nil then none were found, throw error
	if jobResult == nil {
		return nil, errors.NotFound("job <%s> not found in workspace", jobName)
	}

	return jobResult, nil
}

func (svc *SchematicsTestService) GetWorkspaceJobDetail(jobID string) (*schematicsv1.WorkspaceActivity, error) {

	// look up job by ID
	activityResponse, _, err := svc.SchematicsApiSvc.GetWorkspaceActivity(&schematicsv1.GetWorkspaceActivityOptions{
		WID:        core.StringPtr(svc.WorkspaceID),
		ActivityID: core.StringPtr(jobID),
	})
	if err != nil {
		return nil, err
	}

	return activityResponse, nil
}

func (svc *SchematicsTestService) WaitForFinalJobStatus(jobID string) (string, error) {
	var status string
	var job *schematicsv1.WorkspaceActivity
	var jobErr error

	// Wait for the job to be complete
	start := time.Now()
	lastLog := int16(0)
	runMinutes := int16(0)
	if svc.TestOptions.WaitJobCompleteMinutes <= 0 {
		svc.TestOptions.WaitJobCompleteMinutes = DefaultWaitJobCompleteMinutes
	}

	for {
		// check for timeout and throw error
		runMinutes = int16(time.Since(start).Minutes())
		if runMinutes > svc.TestOptions.WaitJobCompleteMinutes {
			return "", fmt.Errorf("time exceeded waiting for schematic job to finish")
		}

		// get details of job
		job, jobErr = svc.GetWorkspaceJobDetail(jobID)
		if jobErr != nil {
			return "", jobErr
		}
		// only log svc once a minute or so
		if runMinutes > lastLog {
			log.Printf("[SCHEMATICS] ... still waiting for job %s to complete: %d minutes", *job.Name, runMinutes)
			lastLog = runMinutes
		}

		// check if it is finished
		if job.Status != nil &&
			len(*job.Status) > 0 &&
			*job.Status != SchematicsJobStatusCreated &&
			*job.Status != SchematicsJobStatusInProgress {
			log.Printf("[SCHEMATICS] The status of job %s is: %s", *job.Name, *job.Status)
			break
		}

		// wait 10 seconds
		time.Sleep(10 * time.Second)
	}

	// if we reach svc point the job has finished, return status
	status = *job.Status

	return status, nil
}

func (svc *SchematicsTestService) DeleteWorkspace() (string, error) {

	refreshToken, tokenErr := svc.GetRefreshToken()
	if tokenErr != nil {
		return "", tokenErr
	}

	result, _, err := svc.SchematicsApiSvc.DeleteWorkspace(&schematicsv1.DeleteWorkspaceOptions{
		WID:              core.StringPtr(svc.WorkspaceID),
		RefreshToken:     core.StringPtr(refreshToken),
		DestroyResources: core.StringPtr("false"),
	})
	if err != nil {
		return "", fmt.Errorf("delete of schematic job failed: %w", err)
	}

	return *result, nil
}

func CreateSchematicTar(projectPath string, includePatterns *[]string) (string, error) {

	// create unique tar filename
	target := fmt.Sprintf("%sschematic-test-%s.tar", os.TempDir(), strings.ToLower(random.UniqueId()))

	// files are relative to the root of the project
	chdirErr := os.Chdir(projectPath)
	if chdirErr != nil {
		return "", chdirErr
	}

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

	// start loop through provided list of patterns
	// if none provided, assume just terraform files
	if len(*includePatterns) == 0 {
		includePatterns = &[]string{"*.tf"}
	}
	for _, pattern := range *includePatterns {
		files, _ := filepath.Glob(pattern)

		// loop through files
		for _, fileName := range files {

			// get file info
			info, infoErr := os.Stat(fileName)
			if infoErr != nil {
				return "", infoErr
			}
			fileDir := filepath.Dir(fileName)

			// skip directories, just in case
			if info.IsDir() {
				continue
			}

			hdr, hdrErr := tar.FileInfoHeader(info, info.Name())
			if hdrErr != nil {
				return "", hdrErr
			}

			// the FI header sets the name as base name only, so to preserve the leading directories (if needed)
			// we will alter the name
			if fileDir != "." {
				hdr.Name = filepath.Join(fileDir, hdr.Name)
			}

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
