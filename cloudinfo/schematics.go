package cloudinfo

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/IBM/go-sdk-core/v5/core"
	schematics "github.com/IBM/schematics-go-sdk/schematicsv1"
)

func (infoSvc *CloudInfoService) GetSchematicsJobLogs(jobID string) (result *schematics.JobLog, response *core.DetailedResponse, err error) {
	return infoSvc.schematicsService.ListJobLogs(
		&schematics.ListJobLogsOptions{
			JobID: core.StringPtr(jobID),
		},
	)
}

// GetSchematicsJobLogsText retrieves the logs of a Schematics job as a string
// The logs are returned as a string, or an error if the operation failed
// This is a temporary workaround until the Schematics GO SDK is fixed, ListJobLogs is broken as the response is text/plain and not application/json
func (infoSvc *CloudInfoService) GetSchematicsJobLogsText(jobID string) (logs string, err error) {
	const maxRetries = 3
	const retryDelay = 2 * time.Second

	url := fmt.Sprintf("https://schematics.cloud.ibm.com/v2/jobs/%s/logs", jobID)
	var retryErrors []string

	for attempt := 1; attempt <= maxRetries; attempt++ {
		// Create the request
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return "", fmt.Errorf("failed to create request: %v", err)
		}

		// Authenticate the request
		err = infoSvc.authenticator.Authenticate(req)
		if err != nil {
			return "", fmt.Errorf("failed to authenticate: %v", err)
		}

		// Make the request
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			retryErrors = append(retryErrors, fmt.Sprintf("attempt %d: failed to make request: %v", attempt, err))
			if attempt < maxRetries {
				time.Sleep(retryDelay)
				continue
			}
			return "", fmt.Errorf("exceeded maximum retries, attempt failures:\n%s", strings.Join(retryErrors, "\n"))
		}
		defer resp.Body.Close()

		// Check if the response status is successful
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			// Read the response body
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return "", fmt.Errorf("failed to read response body: %v", err)
			}
			return string(body), nil
		} else {
			retryErrors = append(retryErrors, fmt.Sprintf("attempt %d: request failed with status code: %d", attempt, resp.StatusCode))
			if attempt < maxRetries {
				time.Sleep(retryDelay)
				continue
			}
			return "", fmt.Errorf("exceeded maximum retries, attempt failures:\n%s", strings.Join(retryErrors, "\n"))
		}
	}

	return "", fmt.Errorf("exceeded maximum retries, attempt failures:\n%s", strings.Join(retryErrors, "\n"))
}

// GetSchematicsJobFileData will download a specific job file and return a JobFileData structure.
// Allowable values for fileType: template_repo, readme_file, log_file, state_file, plan_json
func (infoSvc *CloudInfoService) GetSchematicsJobFileData(jobID string, fileType string) (*schematics.JobFileData, error) {
	// setup options
	// file type Allowable values: [template_repo,readme_file,log_file,state_file,plan_json]
	jobFileOptions := &schematics.GetJobFilesOptions{
		JobID:    core.StringPtr(jobID),
		FileType: core.StringPtr(fileType),
	}

	data, _, err := infoSvc.schematicsService.GetJobFiles(jobFileOptions)

	return data, err
}

func (infoSvc *CloudInfoService) GetSchematicsJobPlanJson(jobID string) (string, error) {
	// get the plan_json file for the job
	data, dataErr := infoSvc.GetSchematicsJobFileData(jobID, "plan_json")

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
