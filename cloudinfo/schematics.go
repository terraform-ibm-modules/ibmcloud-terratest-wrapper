package cloudinfo

import (
	"fmt"
	"math/rand"
	"net/url"
	"strings"

	"github.com/IBM/go-sdk-core/v5/core"
	schematics "github.com/IBM/schematics-go-sdk/schematicsv1"
	"github.com/IBM/vpc-go-sdk/common"
)

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
