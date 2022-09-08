package cloudinfo

import (
	"context"
	"fmt"
	"log"
	"strings"

	ibmpiinstance "github.com/IBM-Cloud/power-go-client/clients/instance"
	"github.com/IBM-Cloud/power-go-client/ibmpisession"
	ibmpimodels "github.com/IBM-Cloud/power-go-client/power/models"
	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/platform-services-go-sdk/resourcecontrollerv2"
)

// ListPowervsInstances will retrieve all PowerCloud parent instances for an account.
// There is no API to retrieve all Powercloud instances for an account.
// This function will loop through ALL resources of type "service_instance" in the account
// and then filter by looking for "power-iaas" in the CRN.
func (infoSvc *CloudInfoService) ListPowervsInstances() ([]resourcecontrollerv2.ResourceInstance, error) {

	listOptions := infoSvc.resourceControllerService.NewListResourceInstancesOptions()
	listOptions.SetType("service_instance")
	listOptions.SetLimit(int64(100))

	// this API is paginated, but there is no pager support in the library at this time.
	// we are compensating by inspecting the NextURL and Start values supplied by the API
	// to keep looping through pages
	maxPages := 100
	countPages := 0
	listPowerInstance := []resourcecontrollerv2.ResourceInstance{}
	moreData := true

	for moreData {
		listPage, _, err := infoSvc.resourceControllerService.ListResourceInstances(listOptions)
		if err != nil {
			return nil, fmt.Errorf("error listing PowerVS instances: %w", err)
		}
		countPages += 1

		// search through instances on current page looking for only power-iaas
		for _, svcInstance := range listPage.Resources {
			if strings.Contains(*svcInstance.CRN, "power-iaas") {
				listPowerInstance = append(listPowerInstance, svcInstance)
			}
		}

		// get next page of results if necessary
		// see: https://cloud.ibm.com/apidocs/resource-controller/resource-controller?code=go#list-resource-instances
		if (countPages < maxPages) && listPage.NextURL != nil && *listPage.NextURL != "" {
			moreData = true
			newStart, errNext := core.GetQueryParam(listPage.NextURL, "start")
			if errNext != nil || newStart == nil {
				return nil, fmt.Errorf("error in fetching start value from next_url: %w", errNext)
			}
			listOptions.SetStart(*newStart)
		} else {
			moreData = false
		}
	}

	return listPowerInstance, nil
}

// ListPowervsInstanceConnections will return an array of CloudConnection for a given existing connection client
// NOTE: the client passed into this method is derived from a specific PowerCloud Instance ID and Region/Zone.
// NOTE 2: the client object parameter is a reference to an interface to allow for unit test mocking
func (infoSvc *CloudInfoService) ListPowervsInstanceConnections(client ibmPICloudConnectionClient) ([]*ibmpimodels.CloudConnection, error) {

	// get all connections
	allResp, err := client.GetAll()
	if err != nil {
		// there is issue with powercloud instances not being fully removed from the resource controller database.
		// this results in IDs being reported that are no longer valid.
		// checking for this case and ignoring.
		if strings.Contains(err.Error(), "unable to find cloud instance id") {
			return nil, nil
		} else {
			log.Println("Error retrieving Powercloud Connections: ", err)
			return nil, err
		}
	}
	// allResp here is not an array, its an object which contains an array, but we really want to return an array.
	// so first we check to see if the parent is nil, in which case we return a nil for the array return.
	if allResp == nil {
		return nil, nil
	}

	// allResp is an object containing an array, but we want to return the actual array (for count purposes later)
	return allResp.CloudConnections, nil
}

// CreatePowercloudSession will return a PowerPI session object that is tailored to a specific cloud account and region/zone
func (infoSvc *CloudInfoService) CreatePowercloudSession(instanceRegion string) (*ibmpisession.IBMPISession, error) {
	// get current auth cloud account_id needed for this API
	apiKeyDetail, keyErr := infoSvc.getApiKeyDetail()
	if keyErr != nil || apiKeyDetail == nil || apiKeyDetail.AccountID == nil {
		// if we are unable to get accountId we will not be able to proceed
		log.Println("ERROR: unable to retrieve valid ACCOUNT_ID, cannot proceed with Powercloud query")
		return nil, fmt.Errorf("unable to retrieve valid ACCOUNT_ID")
	}

	// get powercloud client
	sessionOptions := &ibmpisession.IBMPIOptions{
		Authenticator: infoSvc.authenticator,
		UserAccount:   *apiKeyDetail.AccountID,
		Zone:          instanceRegion,
	}
	session, err := ibmpisession.NewIBMPISession(sessionOptions)
	if err != nil {
		log.Println("Error creating PI session: ", err)
		return nil, err
	}

	return session, nil
}

// CreatePowercloudConnectionClient will return a Power PI Connection Client based on an existing session object and a Powercloud instance Id
func (infoSvc *CloudInfoService) CreatePowercloudConnectionClient(instanceId string, powerSession *ibmpisession.IBMPISession) *ibmpiinstance.IBMPICloudConnectionClient {
	// get a powercloud connection client
	ccClient := ibmpiinstance.NewIBMPICloudConnectionClient(context.Background(), powerSession, instanceId)

	return ccClient
}
