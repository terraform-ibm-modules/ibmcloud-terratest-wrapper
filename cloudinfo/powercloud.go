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

// extend the CloudConnection struct to add zone (region or datacenter)
type PowerCloudConnectionDetail struct {
	*ibmpimodels.CloudConnection
	Zone *string
}

// ListPowerWorkspaces will retrieve all PowerCloud parent instances for an account.
// There is no API to retrieve all Powercloud instances for an account.
// This function will loop through ALL resources of type "service_instance" in the account
// and then filter by looking for "power-iaas" in the CRN.
func (infoSvc *CloudInfoService) ListPowerWorkspaces() ([]resourcecontrollerv2.ResourceInstance, error) {

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

// ListPowerWorkspaceConnections will return an array of CloudConnection for a given existing connection client
// NOTE: the client passed into this method is derived from a specific PowerCloud Instance ID and Region/Zone.
// NOTE 2: the client object parameter is a reference to an interface to allow for unit test mocking
func (infoSvc *CloudInfoService) ListPowerWorkspaceConnections(client ibmPICloudConnectionClient) ([]*ibmpimodels.CloudConnection, error) {

	// get all connections
	allResp, err := client.GetAll()
	if err != nil {
		// there is issue with powercloud instances not being fully removed from the resource controller database.
		// this results in IDs being reported that are no longer valid.
		// checking for this case and ignoring.
		if strings.Contains(err.Error(), "unable to find cloud instance id") {
			return nil, nil
		} else {
			return nil, fmt.Errorf("error retrieving Powercloud Connections: %w", err)
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

// ListPowercloudConnectionsForAccount will return an array of CloudConnection that contains all unique connections
// in the current account (current account determined by API Key used)
func (infoSvc *CloudInfoService) ListPowercloudConnectionsForAccount() ([]*PowerCloudConnectionDetail, error) {

	var uniqueConnections []*PowerCloudConnectionDetail

	// first we need a list of all workspaces for the account
	wsList, wsErr := infoSvc.ListPowerWorkspaces()
	if wsErr != nil {
		return nil, wsErr
	}

	// for each workspace in account, get connections
	for _, powerWs := range wsList {
		// the sessions are for specific zone/region, so we need new session for each iteration of this
		sess, sessErr := infoSvc.CreatePowercloudSession(*powerWs.RegionID)
		if sessErr != nil {
			return nil, sessErr
		}

		// use session to get a cloud connection client, which also requires the ID of each workspace
		ccClient := infoSvc.CreatePowercloudConnectionClient(*powerWs.GUID, sess)

		// get a list of connections for the workspace
		ccList, ccErr := infoSvc.ListPowerWorkspaceConnections(ccClient)
		if ccErr != nil {
			return nil, ccErr
		}

		// build a final list of connections plus extended data, unique by connection id
		for _, cc := range ccList {
			// only add to list if ID is not already present
			// NOTE: this is required because connections in same zone/region are shared across workspaces!
			if !cloudConnectionDetailExists(cc, uniqueConnections) {
				ccDetail := &PowerCloudConnectionDetail{
					CloudConnection: cc,
					Zone:            powerWs.RegionID,
				}
				uniqueConnections = append(uniqueConnections, ccDetail)
			}
		}
	}

	return uniqueConnections, nil
}

// cloudConnectionDetailExists is a simple helper function to check if a given cloud connection object already exists in detail array
func cloudConnectionDetailExists(connection *ibmpimodels.CloudConnection, connectionList []*PowerCloudConnectionDetail) bool {

	for _, cc := range connectionList {
		if strings.Compare(*cc.CloudConnectionID, *connection.CloudConnectionID) == 0 {
			return true
		}
	}
	return false
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
		return nil, fmt.Errorf("error creating PI session: %w", err)
	}

	return session, nil
}

// CreatePowercloudConnectionClient will return a Power PI Connection Client based on an existing session object and a Powercloud instance Id
func (infoSvc *CloudInfoService) CreatePowercloudConnectionClient(instanceId string, powerSession *ibmpisession.IBMPISession) *ibmpiinstance.IBMPICloudConnectionClient {
	// get a powercloud connection client
	ccClient := ibmpiinstance.NewIBMPICloudConnectionClient(context.Background(), powerSession, instanceId)

	return ccClient
}
