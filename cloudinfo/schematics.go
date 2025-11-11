package cloudinfo

import (
	"archive/tar"
	"fmt"
	"io"
	"math/rand"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/IBM/go-sdk-core/v5/core"
	schematics "github.com/IBM/schematics-go-sdk/schematicsv1"
	"github.com/IBM/vpc-go-sdk/common"
	"github.com/go-openapi/errors"
	"github.com/gruntwork-io/terratest/modules/random"
	commonpkg "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/common"
)

// IBM Schematics job types
const (
	SchematicsJobTypeUpload  = "TAR_WORKSPACE_UPLOAD"
	SchematicsJobTypeUpdate  = "WORKSPACE_UPDATE"
	SchematicsJobTypePlan    = "PLAN"
	SchematicsJobTypeApply   = "APPLY"
	SchematicsJobTypeDestroy = "DESTROY"
)

// IBM Schematics job status
const (
	SchematicsJobStatusCompleted  = "COMPLETED"
	SchematicsJobStatusFailed     = "FAILED"
	SchematicsJobStatusCreated    = "CREATED"
	SchematicsJobStatusInProgress = "INPROGRESS"
)

// Defaults for API retry mechanic
const (
	defaultApiRetryCount          = 5
	defaultApiRetryWaitSeconds    = 30
	DefaultWaitJobCompleteMinutes = 60
)

// NetrcCredential represents credentials for netrc authentication
type NetrcCredential struct {
	Host     string
	Username string
	Password string
}

// will return a previously configured schematics service based on location
// error returned if location not initialized
// location must be a valid geographical location supported by schematics: "us" or "eu"
func (infoSvc *CloudInfoService) GetSchematicsServiceByLocation(location string) (schematicsService, error) {
	service, isFound := infoSvc.schematicsServices[location]
	if !isFound {
		return nil, fmt.Errorf("could not find Schematics Service for location %s", location)
	}

	return service, nil
}

func (infoSvc *CloudInfoService) GetSchematicsJobLogs(jobID string, location string) (result *schematics.JobLog, response *core.DetailedResponse, err error) {
	return infoSvc.schematicsServices[location].ListJobLogs(
		&schematics.ListJobLogsOptions{
			JobID: core.StringPtr(jobID),
		},
	)
}

// GetSchematicsJobLogsText retrieves the logs of a Schematics job as a string
// The logs are returned as a string, or an error if the operation failed
// This is a temporary workaround until the Schematics GO SDK is fixed, ListJobLogs is broken as the response is text/plain and not application/json
// location must be a valid geographical location supported by schematics: "us" or "eu"
func (infoSvc *CloudInfoService) GetSchematicsJobLogsText(jobID string, location string) (string, error) {

	svc, svcErr := infoSvc.GetSchematicsServiceByLocation(location)
	if svcErr != nil {
		return "", fmt.Errorf("error getting schematics service for location %s:%w", location, svcErr)
	}

	// build up a REST API call for job logs
	pathParamsMap := map[string]string{
		"job_id": jobID,
	}
	builder := core.NewRequestBuilder(core.GET)
	builder.EnableGzipCompression = svc.GetEnableGzipCompression()
	_, builderErr := builder.ResolveRequestURL(svc.GetServiceURL(), `/v2/jobs/{job_id}/logs`, pathParamsMap)
	if builderErr != nil {
		return "", builderErr
	}
	sdkHeaders := common.GetSdkHeaders("schematics", "V1", "ListJobLogs")
	for headerName, headerValue := range sdkHeaders {
		builder.AddHeader(headerName, headerValue)
	}
	builder.AddHeader("Accept", "application/json")

	request, buildErr := builder.Build()
	if buildErr != nil {
		return "", buildErr
	}

	// initialize the IBM Core HTTP service
	baseService, baseSvcErr := core.NewBaseService(&core.ServiceOptions{
		URL:           svc.GetServiceURL(),
		Authenticator: infoSvc.authenticator,
	})
	if baseSvcErr != nil {
		return "", baseSvcErr
	}

	// make the builder request call on the core http service, which is text/plain
	// using response type "**string" to get raw text output
	rawResponse := core.StringPtr("")
	_, requestErr := baseService.Request(request, &rawResponse)
	if requestErr != nil {
		return "", requestErr
	}

	return *rawResponse, nil
}

// GetSchematicsJobFileData will download a specific job file and return a JobFileData structure.
// Allowable values for fileType: state_file, plan_json
// location must be a valid geographical location supported by schematics: "us" or "eu"
func (infoSvc *CloudInfoService) GetSchematicsJobFileData(jobID string, fileType string, location string) (*schematics.JobFileData, error) {
	// setup options
	// file type Allowable values: [template_repo,readme_file,log_file,state_file,plan_json]
	jobFileOptions := &schematics.GetJobFilesOptions{
		JobID:    core.StringPtr(jobID),
		FileType: core.StringPtr(fileType),
	}

	// get a service based on location
	svc, svcErr := infoSvc.GetSchematicsServiceByLocation(location)
	if svcErr != nil {
		return nil, fmt.Errorf("error getting schematics service for location %s:%w", location, svcErr)
	}

	data, _, err := svc.GetJobFiles(jobFileOptions)

	return data, err
}

// Returns a string of the unmarshalled `Terraform Plan JSON` produced by a schematics job
// location must be a valid geographical location supported by schematics: "us" or "eu"
func (infoSvc *CloudInfoService) GetSchematicsJobPlanJson(jobID string, location string) (string, error) {
	// get the plan_json file for the job
	data, dataErr := infoSvc.GetSchematicsJobFileData(jobID, "plan_json", location)

	// check for multiple error conditions
	if dataErr != nil {
		return "", dataErr
	}
	if data == nil {
		return "", fmt.Errorf("job file data object is nil, which is unexpected")
	}
	if data.FileContent == nil {
		return "", fmt.Errorf("file content is nil, which is unexpected")
	}

	// extract the plan file content and return
	contentPtr := data.FileContent

	return *contentPtr, nil
}

// returns a random selected region that is valid for Schematics Workspace creation
func GetRandomSchematicsLocation() string {
	validLocations := GetSchematicsLocations()
	randomIndex := rand.Intn(len(validLocations))
	return validLocations[randomIndex]
}

// returns the appropriate schematics API endpoint based on specific region
// the region can be geographic (us or eu) or specific (us-south)
func GetSchematicServiceURLForRegion(region string) (string, error) {
	// the service URLs are simply the region in front of base default

	// first, get the default URL from official service
	url, parseErr := url.Parse(schematics.DefaultServiceURL)
	if parseErr != nil {
		return "", fmt.Errorf("error parsing default schematics URL: %w", parseErr)
	}

	// prefix the region in front of existing host
	url.Host = strings.ToLower(region) + "." + url.Host

	return url.String(), nil
}

// CreateSchematicsTar creates a TAR file containing Terraform project files based on include patterns.
// projectPath is the path to the Terraform project directory.
// includePatterns is a slice of file patterns to include (e.g., []string{"*.tf", "*.tfvars"}).
// Returns the full path to the created TAR file and any error encountered.
func CreateSchematicsTar(projectPath string, includePatterns []string) (string, error) {
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
	if len(includePatterns) == 0 {
		includePatterns = []string{"*.tf"}
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

	for _, pattern := range includePatterns {
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
				if !commonpkg.StrArrayContains(dirsAdded, relFileDir) {
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

// AddWorkspaceEnv adds an environment variable to workspace environment values and metadata.
// values is a pointer to the slice of environment value maps.
// metadata is a pointer to the slice of environment metadata.
// key is the environment variable name.
// value is the environment variable value.
// hidden indicates if the variable should be hidden in UI.
// secure indicates if the variable contains sensitive data.
func AddWorkspaceEnv(
	values *[]map[string]interface{},
	metadata *[]schematics.EnvironmentValuesMetadata,
	key string,
	value string,
	hidden bool,
	secure bool,
) {
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

// AddNetrcToWorkspaceEnv adds netrc credentials to workspace environment variables.
// values is a pointer to the slice of environment value maps.
// metadata is a pointer to the slice of environment metadata.
// netrcEntries is a slice of NetrcCredential structs containing host, username, and password.
func AddNetrcToWorkspaceEnv(
	values *[]map[string]interface{},
	metadata *[]schematics.EnvironmentValuesMetadata,
	netrcEntries []NetrcCredential,
) {
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
	netrcValueStr, _ := commonpkg.ConvertValueToJsonString(netrcValue)
	// Add the slice of netrc entries to env with "__netrc__" as the key
	*values = append(*values, map[string]interface{}{"__netrc__": netrcValueStr})

	// Add a metadata entry for the sensitive value
	*metadata = append(*metadata, schematics.EnvironmentValuesMetadata{
		Name:   core.StringPtr("__netrc__"),
		Hidden: core.BoolPtr(false), // Set to false as it's not hidden
		Secure: core.BoolPtr(true),  // Set to true as it's considered secure
	})
}

// GetDetailedResponseStatusCode extracts the HTTP status code from an IBM SDK DetailedResponse.
// If the response is nil (typically due to an error), returns 500.
// resp is the DetailedResponse from an IBM SDK API call.
// Returns the HTTP status code.
func GetDetailedResponseStatusCode(resp *core.DetailedResponse) int {
	if resp != nil {
		return resp.StatusCode
	}
	return 500
}

// getApiRetryStatusExceptions returns a list of HTTP status codes that should not trigger retries.
// These are typically authentication/authorization errors that won't be resolved by retrying.
func getApiRetryStatusExceptions() []int {
	return []int{401, 403}
}

func (infoSvc *CloudInfoService) GetSchematicsWorkspace(workspaceID string, location string) (*schematics.WorkspaceResponse, error) {
	getWorkspaceOptions := &schematics.GetWorkspaceOptions{
		WID: core.StringPtr(workspaceID),
	}

	// Get the appropriate schematics service for the region
	svc, svcErr := infoSvc.GetSchematicsServiceByLocation(location)
	if svcErr != nil {
		return nil, fmt.Errorf("error getting schematics service for location %s: %w", location, svcErr)
	}

	// Use common retry mechanism
	retryConfig := commonpkg.DefaultRetryConfig()
	retryConfig.MaxRetries = defaultApiRetryCount
	retryConfig.InitialDelay = time.Duration(defaultApiRetryWaitSeconds) * time.Second
	retryConfig.OperationName = "GetSchematicsWorkspace"

	workspace, err := commonpkg.RetryWithConfig(retryConfig, func() (*schematics.WorkspaceResponse, error) {
		ws, resp, wsErr := svc.GetWorkspace(getWorkspaceOptions)
		if wsErr != nil {
			statusCode := GetDetailedResponseStatusCode(resp)
			// Don't retry on auth errors
			if commonpkg.IntArrayContains(getApiRetryStatusExceptions(), statusCode) {
				return nil, wsErr
			}
			if infoSvc.Logger != nil {
				infoSvc.Logger.Info(fmt.Sprintf("[SCHEMATICS] RETRY GetWorkspace, status code: %d", statusCode))
			}
			return nil, wsErr
		}
		return ws, nil
	})

	if err != nil {
		return nil, err
	}

	if infoSvc.Logger != nil {
		infoSvc.Logger.Info(fmt.Sprintf("[SCHEMATICS] Got workspace: %s (ID: %s))", *workspace.Name, *workspace.ID))
	}
	return workspace, nil
}

// CreateSchematicsWorkspace creates a new IBM Schematics Workspace for testing.
// name is the workspace name.
// resourceGroup is the resource group ID.
// region is the workspace location/region.
// templateFolder is the folder containing Terraform files (use "." for root).
// terraformVersion is the Terraform version to use (empty string for default).
// tags is a slice of tags to apply to the workspace.
// envVars is a slice of environment variable maps.
// envMetadata is a slice of environment variable metadata.
// Returns the created workspace and any error encountered.
func (infoSvc *CloudInfoService) CreateSchematicsWorkspace(
	name string,
	resourceGroup string,
	region string,
	templateFolder string,
	terraformVersion string,
	tags []string,
	envVars []map[string]interface{},
	envMetadata []schematics.EnvironmentValuesMetadata,
) (*schematics.WorkspaceResponse, error) {
	var folder *string
	var version *string
	var wsVersion []string

	if len(templateFolder) == 0 {
		folder = core.StringPtr(".")
	} else {
		folder = core.StringPtr(templateFolder)
	}

	// choose nil default for version if not supplied, so that they omit from template setup
	// (schematics should then determine defaults)
	if len(terraformVersion) > 0 {
		version = core.StringPtr(terraformVersion)
		wsVersion = []string{terraformVersion}
	}

	// create env and input vars template
	templateModel := &schematics.TemplateSourceDataRequest{
		Folder:            folder,
		Type:              version,
		EnvValues:         envVars,
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

	// Get the appropriate schematics service for the region
	svc, svcErr := infoSvc.GetSchematicsServiceByLocation(region)
	if svcErr != nil {
		return nil, fmt.Errorf("error getting schematics service for location %s: %w", region, svcErr)
	}

	// Use common retry mechanism
	retryConfig := commonpkg.DefaultRetryConfig()
	retryConfig.MaxRetries = defaultApiRetryCount
	retryConfig.InitialDelay = time.Duration(defaultApiRetryWaitSeconds) * time.Second
	retryConfig.OperationName = "CreateSchematicsWorkspace"

	workspace, err := commonpkg.RetryWithConfig(retryConfig, func() (*schematics.WorkspaceResponse, error) {
		ws, resp, wsErr := svc.CreateWorkspace(createWorkspaceOptions)
		if wsErr != nil {
			statusCode := GetDetailedResponseStatusCode(resp)
			// Don't retry on auth errors
			if commonpkg.IntArrayContains(getApiRetryStatusExceptions(), statusCode) {
				return nil, wsErr
			}
			if infoSvc.Logger != nil {
				infoSvc.Logger.Info(fmt.Sprintf("[SCHEMATICS] RETRY CreateWorkspace, status code: %d", statusCode))
			}
			return nil, wsErr
		}
		return ws, nil
	})

	if err != nil {
		return nil, err
	}

	if infoSvc.Logger != nil {
		infoSvc.Logger.Info(fmt.Sprintf("[SCHEMATICS] Created workspace: %s (ID: %s))", *workspace.Name, *workspace.ID))
	}

	return workspace, nil
}

// DeleteSchematicsWorkspace deletes an existing Schematics workspace.
// workspaceID is the ID of the workspace to delete.
// location is the workspace location (e.g., "us", "eu").
// destroyResources indicates whether to destroy Terraform resources before deleting workspace.
// Returns the deletion result string and any error encountered.
func (infoSvc *CloudInfoService) DeleteSchematicsWorkspace(
	workspaceID string,
	location string,
	destroyResources bool,
) (string, error) {
	// Get refresh token
	response, err := infoSvc.authenticator.RequestToken()
	if err != nil {
		return "", fmt.Errorf("error getting refresh token: %w", err)
	}
	if len(response.RefreshToken) == 0 {
		return "", fmt.Errorf("refresh token is empty (invalid)")
	}

	// Get the appropriate schematics service for the location
	svc, svcErr := infoSvc.GetSchematicsServiceByLocation(location)
	if svcErr != nil {
		return "", fmt.Errorf("error getting schematics service for location %s: %w", location, svcErr)
	}

	destroyResourcesStr := "false"
	if destroyResources {
		destroyResourcesStr = "true"
	}

	// Use common retry mechanism
	retryConfig := commonpkg.DefaultRetryConfig()
	retryConfig.MaxRetries = defaultApiRetryCount
	retryConfig.InitialDelay = time.Duration(defaultApiRetryWaitSeconds) * time.Second
	retryConfig.OperationName = "DeleteSchematicsWorkspace"

	result, err := commonpkg.RetryWithConfig(retryConfig, func() (*string, error) {
		res, resp, delErr := svc.DeleteWorkspace(&schematics.DeleteWorkspaceOptions{
			WID:              core.StringPtr(workspaceID),
			RefreshToken:     core.StringPtr(response.RefreshToken),
			DestroyResources: core.StringPtr(destroyResourcesStr),
		})
		if delErr != nil {
			statusCode := GetDetailedResponseStatusCode(resp)
			// Don't retry on auth errors
			if commonpkg.IntArrayContains(getApiRetryStatusExceptions(), statusCode) {
				return nil, delErr
			}
			if infoSvc.Logger != nil {
				infoSvc.Logger.Info(fmt.Sprintf("[SCHEMATICS] RETRY DeleteWorkspace, status code: %d", statusCode))
			}
			return nil, delErr
		}
		return res, nil
	})

	if err != nil {
		return "", fmt.Errorf("delete of schematic workspace failed: %w", err)
	}

	if infoSvc.Logger != nil {
		infoSvc.Logger.Info(fmt.Sprintf("[SCHEMATICS] Deleted workspace: %s", workspaceID))
	}

	return *result, nil
}

// UploadTarToSchematicsWorkspace uploads a TAR file to an existing Schematics workspace.
// workspaceID is the ID of the workspace.
// templateID is the ID of the workspace template.
// tarPath is the file path to the TAR file to upload.
// location is the workspace location (e.g., "us", "eu").
// Returns any error encountered.
func (infoSvc *CloudInfoService) UploadTarToSchematicsWorkspace(
	workspaceID string,
	templateID string,
	tarPath string,
	location string,
) error {
	fileReader, fileErr := os.Open(tarPath)
	if fileErr != nil {
		return fmt.Errorf("error opening reader for tar path: %w", fileErr)
	}
	defer fileReader.Close()
	fileReaderWrapper := io.NopCloser(fileReader)

	uploadTarOptions := &schematics.TemplateRepoUploadOptions{
		WID:             core.StringPtr(workspaceID),
		TID:             core.StringPtr(templateID),
		File:            fileReaderWrapper,
		FileContentType: core.StringPtr("application/octet-stream"),
	}

	// Get the appropriate schematics service for the location
	svc, svcErr := infoSvc.GetSchematicsServiceByLocation(location)
	if svcErr != nil {
		return fmt.Errorf("error getting schematics service for location %s: %w", location, svcErr)
	}

	// Use common retry mechanism
	retryConfig := commonpkg.DefaultRetryConfig()
	retryConfig.MaxRetries = defaultApiRetryCount
	retryConfig.InitialDelay = time.Duration(defaultApiRetryWaitSeconds) * time.Second
	retryConfig.OperationName = "UploadTarToSchematicsWorkspace"

	_, err := commonpkg.RetryWithConfig(retryConfig, func() (*schematics.TemplateRepoTarUploadResponse, error) {
		res, resp, uploadErr := svc.TemplateRepoUpload(uploadTarOptions)
		if uploadErr != nil {
			statusCode := GetDetailedResponseStatusCode(resp)
			// Don't retry on auth errors
			if commonpkg.IntArrayContains(getApiRetryStatusExceptions(), statusCode) {
				return nil, uploadErr
			}
			if infoSvc.Logger != nil {
				infoSvc.Logger.Info(fmt.Sprintf("[SCHEMATICS] RETRY TemplateRepoUpload, status code: %d", statusCode))
			}
			return nil, uploadErr
		}
		return res, nil
	})

	if err != nil {
		return err
	}

	if infoSvc.Logger != nil {
		infoSvc.Logger.Info(fmt.Sprintf("[SCHEMATICS] Uploaded TAR to workspace: %s", workspaceID))
	}

	return nil
}

// UpdateSchematicsWorkspaceVariables updates the input variables for a Schematics workspace template.
// workspaceID is the ID of the workspace.
// templateID is the ID of the workspace template.
// variables is a slice of workspace variable requests.
// location is the workspace location (e.g., "us", "eu").
// Returns any error encountered.
func (infoSvc *CloudInfoService) UpdateSchematicsWorkspaceVariables(
	workspaceID string,
	templateID string,
	variables []schematics.WorkspaceVariableRequest,
	location string,
) error {
	templateModel := &schematics.ReplaceWorkspaceInputsOptions{
		WID:           core.StringPtr(workspaceID),
		TID:           core.StringPtr(templateID),
		Variablestore: variables,
	}

	// Get the appropriate schematics service for the location
	svc, svcErr := infoSvc.GetSchematicsServiceByLocation(location)
	if svcErr != nil {
		return fmt.Errorf("error getting schematics service for location %s: %w", location, svcErr)
	}

	// Use common retry mechanism
	retryConfig := commonpkg.DefaultRetryConfig()
	retryConfig.MaxRetries = defaultApiRetryCount
	retryConfig.InitialDelay = time.Duration(defaultApiRetryWaitSeconds) * time.Second
	retryConfig.OperationName = "UpdateSchematicsWorkspaceVariables"

	_, err := commonpkg.RetryWithConfig(retryConfig, func() (*schematics.UserValues, error) {
		res, resp, updateErr := svc.ReplaceWorkspaceInputs(templateModel)
		if updateErr != nil {
			statusCode := GetDetailedResponseStatusCode(resp)
			// Don't retry on auth errors
			if commonpkg.IntArrayContains(getApiRetryStatusExceptions(), statusCode) {
				return nil, updateErr
			}
			if infoSvc.Logger != nil {
				infoSvc.Logger.Info(fmt.Sprintf("[SCHEMATICS] RETRY ReplaceWorkspaceInputs, status code: %d", statusCode))
			}
			return nil, updateErr
		}
		return res, nil
	})

	if err != nil {
		return err
	}

	if infoSvc.Logger != nil {
		infoSvc.Logger.Info(fmt.Sprintf("[SCHEMATICS] Updated variables for workspace: %s", workspaceID))
	}

	return nil
}

// GetSchematicsWorkspaceOutputs retrieves the current Terraform outputs from a Schematics workspace.
// workspaceID is the ID of the workspace.
// location is the workspace location (e.g., "us", "eu").
// Returns a map of output names to values and any error encountered.
func (infoSvc *CloudInfoService) GetSchematicsWorkspaceOutputs(
	workspaceID string,
	location string,
) (map[string]interface{}, error) {
	// Get the appropriate schematics service for the location
	svc, svcErr := infoSvc.GetSchematicsServiceByLocation(location)
	if svcErr != nil {
		return nil, fmt.Errorf("error getting schematics service for location %s: %w", location, svcErr)
	}

	// Use common retry mechanism
	retryConfig := commonpkg.DefaultRetryConfig()
	retryConfig.MaxRetries = defaultApiRetryCount
	retryConfig.InitialDelay = time.Duration(defaultApiRetryWaitSeconds) * time.Second
	retryConfig.OperationName = "GetSchematicsWorkspaceOutputs"

	outputResponse, err := commonpkg.RetryWithConfig(retryConfig, func() ([]schematics.OutputValuesInner, error) {
		res, resp, outputErr := svc.GetWorkspaceOutputs(&schematics.GetWorkspaceOutputsOptions{
			WID: core.StringPtr(workspaceID),
		})
		if outputErr != nil {
			statusCode := GetDetailedResponseStatusCode(resp)
			// Don't retry on auth errors
			if commonpkg.IntArrayContains(getApiRetryStatusExceptions(), statusCode) {
				return nil, outputErr
			}
			if infoSvc.Logger != nil {
				infoSvc.Logger.Info(fmt.Sprintf("[SCHEMATICS] RETRY GetWorkspaceOutputs, status code: %d", statusCode))
			}
			return nil, outputErr
		}
		return res, nil
	})

	if err != nil {
		return make(map[string]interface{}), err
	}

	// DEV NOTE: the return type from SDK is an array of output wrapper, inside is an array of output maps.
	// Through testing I only saw one set of outputs (outputResponse[0].OutputValues[0]),
	// but implementing a loop/merge here for safety.
	allOutputs := make(map[string]interface{})
	for _, outputWrapper := range outputResponse {
		for _, outputInner := range outputWrapper.OutputValues {
			for k, v := range outputInner {
				allOutputs[k] = v
			}
		}
	}

	if infoSvc.Logger != nil {
		infoSvc.Logger.Info(fmt.Sprintf("[SCHEMATICS] Retrieved %d outputs from workspace: %s", len(allOutputs), workspaceID))
	}

	return allOutputs, nil
}

// CreateSchematicsPlanJob initiates a new PLAN action on a Schematics workspace.
// workspaceID is the ID of the workspace.
// location is the workspace location (e.g., "us", "eu").
// Returns the plan result and any error encountered.
func (infoSvc *CloudInfoService) CreateSchematicsPlanJob(
	workspaceID string,
	location string,
) (*schematics.WorkspaceActivityPlanResult, error) {
	// Get refresh token
	response, err := infoSvc.authenticator.RequestToken()
	if err != nil {
		return nil, fmt.Errorf("error getting refresh token: %w", err)
	}
	if len(response.RefreshToken) == 0 {
		return nil, fmt.Errorf("refresh token is empty (invalid)")
	}

	// Get the appropriate schematics service for the location
	svc, svcErr := infoSvc.GetSchematicsServiceByLocation(location)
	if svcErr != nil {
		return nil, fmt.Errorf("error getting schematics service for location %s: %w", location, svcErr)
	}

	// Use common retry mechanism
	retryConfig := commonpkg.DefaultRetryConfig()
	retryConfig.MaxRetries = defaultApiRetryCount
	retryConfig.InitialDelay = time.Duration(defaultApiRetryWaitSeconds) * time.Second
	retryConfig.OperationName = "CreateSchematicsPlanJob"

	planResult, err := commonpkg.RetryWithConfig(retryConfig, func() (*schematics.WorkspaceActivityPlanResult, error) {
		res, resp, planErr := svc.PlanWorkspaceCommand(&schematics.PlanWorkspaceCommandOptions{
			WID:          core.StringPtr(workspaceID),
			RefreshToken: core.StringPtr(response.RefreshToken),
		})
		if planErr != nil {
			statusCode := GetDetailedResponseStatusCode(resp)
			// Don't retry on auth errors
			if commonpkg.IntArrayContains(getApiRetryStatusExceptions(), statusCode) {
				return nil, planErr
			}
			if infoSvc.Logger != nil {
				infoSvc.Logger.Info(fmt.Sprintf("[SCHEMATICS] RETRY PlanWorkspaceCommand, status code: %d", statusCode))
			}
			return nil, planErr
		}
		return res, nil
	})

	if err != nil {
		return nil, err
	}

	if infoSvc.Logger != nil && planResult.Activityid != nil {
		infoSvc.Logger.Info(fmt.Sprintf("[SCHEMATICS] Created plan job: %s for workspace: %s", *planResult.Activityid, workspaceID))
	}

	return planResult, nil
}

// CreateSchematicsApplyJob initiates a new APPLY action on a Schematics workspace.
// workspaceID is the ID of the workspace.
// location is the workspace location (e.g., "us", "eu").
// Returns the apply result and any error encountered.
func (infoSvc *CloudInfoService) CreateSchematicsApplyJob(
	workspaceID string,
	location string,
) (*schematics.WorkspaceActivityApplyResult, error) {
	// Get refresh token
	response, err := infoSvc.authenticator.RequestToken()
	if err != nil {
		return nil, fmt.Errorf("error getting refresh token: %w", err)
	}
	if len(response.RefreshToken) == 0 {
		return nil, fmt.Errorf("refresh token is empty (invalid)")
	}

	// Get the appropriate schematics service for the location
	svc, svcErr := infoSvc.GetSchematicsServiceByLocation(location)
	if svcErr != nil {
		return nil, fmt.Errorf("error getting schematics service for location %s: %w", location, svcErr)
	}

	// Use common retry mechanism
	retryConfig := commonpkg.DefaultRetryConfig()
	retryConfig.MaxRetries = defaultApiRetryCount
	retryConfig.InitialDelay = time.Duration(defaultApiRetryWaitSeconds) * time.Second
	retryConfig.OperationName = "CreateSchematicsApplyJob"

	applyResult, err := commonpkg.RetryWithConfig(retryConfig, func() (*schematics.WorkspaceActivityApplyResult, error) {
		res, resp, applyErr := svc.ApplyWorkspaceCommand(&schematics.ApplyWorkspaceCommandOptions{
			WID:          core.StringPtr(workspaceID),
			RefreshToken: core.StringPtr(response.RefreshToken),
		})
		if applyErr != nil {
			statusCode := GetDetailedResponseStatusCode(resp)
			// Don't retry on auth errors
			if commonpkg.IntArrayContains(getApiRetryStatusExceptions(), statusCode) {
				return nil, applyErr
			}
			if infoSvc.Logger != nil {
				infoSvc.Logger.Info(fmt.Sprintf("[SCHEMATICS] RETRY ApplyWorkspaceCommand, status code: %d", statusCode))
			}
			return nil, applyErr
		}
		return res, nil
	})

	if err != nil {
		return nil, err
	}

	if infoSvc.Logger != nil && applyResult.Activityid != nil {
		infoSvc.Logger.Info(fmt.Sprintf("[SCHEMATICS] Created apply job: %s for workspace: %s", *applyResult.Activityid, workspaceID))
	}

	return applyResult, nil
}

// CreateSchematicsDestroyJob initiates a new DESTROY action on a Schematics workspace.
// workspaceID is the ID of the workspace.
// location is the workspace location (e.g., "us", "eu").
// Returns the destroy result and any error encountered.
func (infoSvc *CloudInfoService) CreateSchematicsDestroyJob(
	workspaceID string,
	location string,
) (*schematics.WorkspaceActivityDestroyResult, error) {
	// Get refresh token
	response, err := infoSvc.authenticator.RequestToken()
	if err != nil {
		return nil, fmt.Errorf("error getting refresh token: %w", err)
	}
	if len(response.RefreshToken) == 0 {
		return nil, fmt.Errorf("refresh token is empty (invalid)")
	}

	// Get the appropriate schematics service for the location
	svc, svcErr := infoSvc.GetSchematicsServiceByLocation(location)
	if svcErr != nil {
		return nil, fmt.Errorf("error getting schematics service for location %s: %w", location, svcErr)
	}

	// Use common retry mechanism
	retryConfig := commonpkg.DefaultRetryConfig()
	retryConfig.MaxRetries = defaultApiRetryCount
	retryConfig.InitialDelay = time.Duration(defaultApiRetryWaitSeconds) * time.Second
	retryConfig.OperationName = "CreateSchematicsDestroyJob"

	destroyResult, err := commonpkg.RetryWithConfig(retryConfig, func() (*schematics.WorkspaceActivityDestroyResult, error) {
		res, resp, destroyErr := svc.DestroyWorkspaceCommand(&schematics.DestroyWorkspaceCommandOptions{
			WID:          core.StringPtr(workspaceID),
			RefreshToken: core.StringPtr(response.RefreshToken),
		})
		if destroyErr != nil {
			statusCode := GetDetailedResponseStatusCode(resp)
			// Don't retry on auth errors
			if commonpkg.IntArrayContains(getApiRetryStatusExceptions(), statusCode) {
				return nil, destroyErr
			}
			if infoSvc.Logger != nil {
				infoSvc.Logger.Info(fmt.Sprintf("[SCHEMATICS] RETRY DestroyWorkspaceCommand, status code: %d", statusCode))
			}
			return nil, destroyErr
		}
		return res, nil
	})

	if err != nil {
		return nil, err
	}

	if infoSvc.Logger != nil && destroyResult.Activityid != nil {
		infoSvc.Logger.Info(fmt.Sprintf("[SCHEMATICS] Created destroy job: %s for workspace: %s", *destroyResult.Activityid, workspaceID))
	}

	return destroyResult, nil
}

// GetSchematicsWorkspaceJobDetail retrieves full details about a Schematics workspace activity.
// workspaceID is the ID of the workspace.
// jobID is the ID of the job/activity.
// location is the workspace location (e.g., "us", "eu").
// Returns the workspace activity details and any error encountered.
func (infoSvc *CloudInfoService) GetSchematicsWorkspaceJobDetail(
	workspaceID string,
	jobID string,
	location string,
) (*schematics.WorkspaceActivity, error) {
	// Get the appropriate schematics service for the location
	svc, svcErr := infoSvc.GetSchematicsServiceByLocation(location)
	if svcErr != nil {
		return nil, fmt.Errorf("error getting schematics service for location %s: %w", location, svcErr)
	}

	// Use common retry mechanism
	retryConfig := commonpkg.DefaultRetryConfig()
	retryConfig.MaxRetries = defaultApiRetryCount
	retryConfig.InitialDelay = time.Duration(defaultApiRetryWaitSeconds) * time.Second
	retryConfig.OperationName = "GetSchematicsWorkspaceJobDetail"

	activityResponse, err := commonpkg.RetryWithConfig(retryConfig, func() (*schematics.WorkspaceActivity, error) {
		res, resp, actErr := svc.GetWorkspaceActivity(&schematics.GetWorkspaceActivityOptions{
			WID:        core.StringPtr(workspaceID),
			ActivityID: core.StringPtr(jobID),
		})
		if actErr != nil {
			statusCode := GetDetailedResponseStatusCode(resp)
			// Don't retry on auth errors
			if commonpkg.IntArrayContains(getApiRetryStatusExceptions(), statusCode) {
				return nil, actErr
			}
			if infoSvc.Logger != nil {
				infoSvc.Logger.Info(fmt.Sprintf("[SCHEMATICS] RETRY GetWorkspaceActivity, status code: %d", statusCode))
			}
			return nil, actErr
		}
		return res, nil
	})

	if err != nil {
		return nil, err
	}

	return activityResponse, nil
}

// FindLatestSchematicsJobByName finds the latest executed job of the specified type.
// workspaceID is the ID of the workspace.
// jobName is the job type name (e.g., SchematicsJobTypePlan, SchematicsJobTypeApply).
// location is the workspace location (e.g., "us", "eu").
// Returns the workspace activity and any error encountered.
func (infoSvc *CloudInfoService) FindLatestSchematicsJobByName(
	workspaceID string,
	jobName string,
	location string,
) (*schematics.WorkspaceActivity, error) {
	// Get the appropriate schematics service for the location
	svc, svcErr := infoSvc.GetSchematicsServiceByLocation(location)
	if svcErr != nil {
		return nil, fmt.Errorf("error getting schematics service for location %s: %w", location, svcErr)
	}

	// Create a custom retry config for this operation
	retryConfig := commonpkg.DefaultRetryConfig()
	retryConfig.MaxRetries = 3
	retryConfig.OperationName = "FindLatestSchematicsJobByName"

	job, err := commonpkg.RetryWithConfig(retryConfig, func() (*schematics.WorkspaceActivity, error) {
		// get array of jobs using workspace id
		listResult, _, listErr := svc.ListWorkspaceActivities(&schematics.ListWorkspaceActivitiesOptions{
			WID: core.StringPtr(workspaceID),
		})

		if listErr != nil {
			return nil, listErr
		}

		// loop through jobs and get latest one that matches name
		var jobResult *schematics.WorkspaceActivity
		var availableJobTypes []string // Track available job types for better error messages

		for i, job := range listResult.Actions {
			// Add job type to available types list if it has a name
			if job.Name != nil {
				availableJobTypes = append(availableJobTypes, *job.Name)
			}

			// only match name
			if job.Name != nil && *job.Name == jobName {
				// keep latest job of this name
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
			// Log available job types for debugging
			if len(availableJobTypes) > 0 && infoSvc.Logger != nil {
				infoSvc.Logger.Info(fmt.Sprintf("[SCHEMATICS] Job <%s> not found, retrying. Available job types: %v",
					jobName, availableJobTypes))
			}
			return nil, errors.NotFound("job <%s> not found in workspace", jobName)
		}

		return jobResult, nil
	})

	return job, err
}

// WaitForSchematicsJobCompletion waits for a Schematics job to complete.
// workspaceID is the ID of the workspace.
// jobID is the ID of the job/activity.
// location is the workspace location (e.g., "us", "eu").
// timeoutMinutes is the maximum time to wait for job completion.
// Returns the final job status and any error encountered.
func (infoSvc *CloudInfoService) WaitForSchematicsJobCompletion(
	workspaceID string,
	jobID string,
	location string,
	timeoutMinutes int,
) (string, error) {
	var status string
	var job *schematics.WorkspaceActivity
	var jobErr error

	if timeoutMinutes <= 0 {
		timeoutMinutes = DefaultWaitJobCompleteMinutes
	}

	// Wait for the job to be complete
	start := time.Now()
	lastLog := int16(0)
	runMinutes := int16(0)

	for {
		// check for timeout and throw error
		runMinutes = int16(time.Since(start).Minutes())
		if runMinutes > int16(timeoutMinutes) {
			return "", fmt.Errorf("time exceeded waiting for schematic job to finish")
		}

		// get details of job
		job, jobErr = infoSvc.GetSchematicsWorkspaceJobDetail(workspaceID, jobID, location)
		if jobErr != nil {
			return "", jobErr
		}
		// only log this once a minute or so
		if runMinutes > lastLog && infoSvc.Logger != nil {
			infoSvc.Logger.Info(fmt.Sprintf("[SCHEMATICS] ... still waiting for job %s to complete: %d minutes", *job.Name, runMinutes))
			lastLog = runMinutes
		}

		// check if it is finished
		if job.Status != nil &&
			len(*job.Status) > 0 &&
			*job.Status != SchematicsJobStatusCreated &&
			*job.Status != SchematicsJobStatusInProgress {
			if infoSvc.Logger != nil {
				infoSvc.Logger.Info(fmt.Sprintf("[SCHEMATICS] The status of job %s is: %s", *job.Name, *job.Status))
			}
			break
		}

		// wait 60 seconds
		time.Sleep(60 * time.Second)
	}

	// if we reach this point the job has finished, return status
	status = *job.Status

	return status, nil
}
