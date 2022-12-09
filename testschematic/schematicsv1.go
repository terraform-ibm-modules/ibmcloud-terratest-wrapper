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
type SchematicsSvcI interface {
	CreateWorkspace(createWorkspaceOptions *schematicsv1.CreateWorkspaceOptions) (result *schematicsv1.WorkspaceResponse, response *core.DetailedResponse, err error)
	TemplateRepoUpload(templateRepoUploadOptions *schematicsv1.TemplateRepoUploadOptions) (result *schematicsv1.TemplateRepoTarUploadResponse, response *core.DetailedResponse, err error)
	ReplaceWorkspaceInputs(replaceWorkspaceInputsOptions *schematicsv1.ReplaceWorkspaceInputsOptions) (result *schematicsv1.UserValues, response *core.DetailedResponse, err error)
	ListWorkspaceActivities(listWorkspaceActivitiesOptions *schematicsv1.ListWorkspaceActivitiesOptions) (result *schematicsv1.WorkspaceActivities, response *core.DetailedResponse, err error)
	GetWorkspaceActivity(getWorkspaceActivityOptions *schematicsv1.GetWorkspaceActivityOptions) (result *schematicsv1.WorkspaceActivity, response *core.DetailedResponse, err error)
}

func CreateSchematicsService(ibmcloudApiKey string) (SchematicsSvcI, error) {

	schematicsSvc, newErr := schematicsv1.NewSchematicsV1(&schematicsv1.SchematicsV1Options{
		URL: "https://schematics.cloud.ibm.com",
		Authenticator: &core.IamAuthenticator{
			ApiKey: ibmcloudApiKey, // pragma: allowlist secret
		},
	})
	if newErr != nil {
		return nil, newErr
	}

	return schematicsSvc, nil
}

func CreateTestWorkspace(svc SchematicsSvcI, name string, resourceGroup string, tags []string, options *TestSchematicOptions) (*schematicsv1.WorkspaceResponse, error) {

	// create env and input vars template
	templateModel := &schematicsv1.TemplateSourceDataRequest{
		Folder: core.StringPtr("."),
		Type:   core.StringPtr("terraform_v1.2"),
		EnvValues: []interface{}{
			map[string]string{
				"__netrc__": fmt.Sprintf("[['github.ibm.com','%s','%s']]", options.RequiredEnvironmentVars[gitUser], options.RequiredEnvironmentVars[gitToken]),
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

	workspace, _, workspaceErr := svc.CreateWorkspace(createWorkspaceOptions)
	if workspaceErr != nil {
		return nil, workspaceErr
	}

	return workspace, nil
}

func UpdateTestTemplateVars(svc SchematicsSvcI, workspaceID string, templateID string, vars []TestSchematicTerraformVar) error {

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
		WID:           core.StringPtr(workspaceID),
		TID:           core.StringPtr(templateID),
		Variablestore: variables,
	}

	// now update template
	_, _, updateErr := svc.ReplaceWorkspaceInputs(templateModel)
	if updateErr != nil {
		return updateErr
	}

	return nil
}

func UploadTarToWorkspace(svc SchematicsSvcI, workspaceID string, templateID string, tarPath string) error {
	fileReader, _ := os.Open(tarPath)
	fileReaderWrapper := io.NopCloser(fileReader)

	uploadTarOptions := &schematicsv1.TemplateRepoUploadOptions{
		WID:             core.StringPtr(workspaceID),
		TID:             core.StringPtr(templateID),
		File:            fileReaderWrapper,
		FileContentType: core.StringPtr("application/octet-stream"),
	}

	_, _, uploadErr := svc.TemplateRepoUpload(uploadTarOptions)
	if uploadErr != nil {
		return uploadErr
	}

	return nil
}

func FindLatestWorkspaceJobByName(svc SchematicsSvcI, workspaceID string, jobName string) (*schematicsv1.WorkspaceActivity, error) {

	// get array of jobs using workspace id
	listResult, _, listErr := svc.ListWorkspaceActivities(&schematicsv1.ListWorkspaceActivitiesOptions{
		WID: core.StringPtr(workspaceID),
	})
	if listErr != nil {
		return nil, listErr
	}

	// loop through jobs and get latest one that matches name
	var jobResult *schematicsv1.WorkspaceActivity
	for _, job := range listResult.Actions {
		// only match name
		if *job.Name == jobName {
			// keep latest job of this name
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

func GetWorkspaceJobDetail(svc SchematicsSvcI, workspaceID string, jobID string) (*schematicsv1.WorkspaceActivity, error) {

	// look up job by ID
	activityResponse, _, err := svc.GetWorkspaceActivity(&schematicsv1.GetWorkspaceActivityOptions{
		WID:        core.StringPtr(workspaceID),
		ActivityID: core.StringPtr(jobID),
	})
	if err != nil {
		return nil, err
	}

	return activityResponse, nil
}

func WaitForFinalJobStatus(svc SchematicsSvcI, workspaceID string, templateID string, jobID string, options *TestSchematicOptions) (string, error) {
	var status string
	var job *schematicsv1.WorkspaceActivity
	var jobErr error

	// Wait for the job to be complete
	start := time.Now()
	if options.WaitJobCompleteMinutes <= 0 {
		options.WaitJobCompleteMinutes = DefaultWaitJobCompleteMinutes
	}

	for {
		// check for timeout and throw error
		if time.Since(start).Minutes() > float64(options.WaitJobCompleteMinutes) {
			return "", errors.New(99, "time exceeded waiting for schematic job to finish")
		}

		// get details of job
		job, jobErr = GetWorkspaceJobDetail(svc, workspaceID, jobID)
		if jobErr != nil {
			return "", jobErr
		}
		log.Println("... still waiting for job", *job.Name, "to complete")

		// check if it is finished
		if job.Status != nil &&
			len(*job.Status) > 0 &&
			*job.Status != SchematicsJobStatusCreated &&
			*job.Status != SchematicsJobStatusInProgress {
			log.Printf("... the status of job %s is: %s", *job.Name, *job.Status)
			break
		}

		// wait 10 seconds
		time.Sleep(10 * time.Second)
	}

	// if we reach this point the job has finished, return status
	status = *job.Status

	return status, nil
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
