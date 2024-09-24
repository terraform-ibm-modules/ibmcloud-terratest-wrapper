package cloudinfo

import (
	"fmt"
	"github.com/IBM/go-sdk-core/v5/core"
	schematics "github.com/IBM/schematics-go-sdk/schematicsv1"
	"io"
	"net/http"
	"strings"
	"time"
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
