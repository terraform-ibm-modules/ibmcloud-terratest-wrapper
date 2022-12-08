package testschematic

import (
	"archive/tar"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/schematics-go-sdk/schematicsv1"
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
		return "", errors.New("tar file is empty, no files added")
	}

	return target, nil
}

func ConvertArrayToJsonString(arr interface{}) (string, error) {
	// first marshal array into json compatible
	json, jsonErr := json.Marshal(arr)
	if jsonErr != nil {
		return "", jsonErr
	}

	// take json array, wrap as one string, and escape any double quotes inside
	s := string(json)

	return s, nil
}

func IsArray(v interface{}) bool {

	theType := reflect.TypeOf(v).Kind()

	if (theType == reflect.Slice) || (theType == reflect.Array) {
		return true
	}

	return false
}
