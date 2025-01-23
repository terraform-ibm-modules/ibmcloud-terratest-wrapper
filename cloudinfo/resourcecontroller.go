package cloudinfo

import (
	"fmt"
	"strings"

	bluemix_crn "github.com/IBM-Cloud/bluemix-go/crn"

	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/platform-services-go-sdk/resourcecontrollerv2"
)

// ListResourcesByCrnServiceName will retrieve all service instances of a provided type, for an account.
// This function will loop through ALL resources of type "service_instance" in the account
// and then filter by looking for the provided service type in the CRN.
func (infoSvc *CloudInfoService) ListResourcesByCrnServiceName(crnServiceName string) ([]resourcecontrollerv2.ResourceInstance, error) {
	listOptions := infoSvc.resourceControllerService.NewListResourceInstancesOptions()
	listOptions.SetType("service_instance")
	listOptions.SetState(resourcecontrollerv2.ListResourceInstancesOptionsStateActiveConst) // only active resources
	listOptions.SetLimit(int64(100))

	allResources, errListingResources := listResourceInstances(infoSvc, listOptions)
	if errListingResources != nil {
		return nil, fmt.Errorf("error listing resources: %w", errListingResources)
	}
	// Loop through all resources and filter by CRN
	var filteredResources []resourcecontrollerv2.ResourceInstance
	for _, resource := range allResources {
		crn, crnErr := bluemix_crn.Parse(*resource.CRN)
		// if crn errors, do it the old fashioned way
		if crnErr == nil {
			// do it the proper way
			if crn.ServiceName == crnServiceName {
				filteredResources = append(filteredResources, resource)
			}
		} else {
			// do it the ugly way
			if strings.Contains(*resource.CRN, crnServiceName) {
				filteredResources = append(filteredResources, resource)
			}
		}
	}
	return filteredResources, nil
}

// ListResourcesByGroupName will retrieve all service instances in a resource group.
func (infoSvc *CloudInfoService) ListResourcesByGroupName(resourceGroupName string) ([]resourcecontrollerv2.ResourceInstance, error) {
	resourceGroupId, err := infoSvc.GetResourceGroupIDByName(resourceGroupName)
	if err != nil {
		return nil, fmt.Errorf("error getting resource group ID: %w", err)
	}
	return infoSvc.ListResourcesByGroupID(resourceGroupId)
}

// ListResourcesByGroupID will retrieve all service instances in a resource group.
func (infoSvc *CloudInfoService) ListResourcesByGroupID(resourceGroupId string) ([]resourcecontrollerv2.ResourceInstance, error) {
	listOptions := infoSvc.resourceControllerService.NewListResourceInstancesOptions()
	listOptions.SetType("resource_instance")
	listOptions.SetLimit(int64(100))
	listOptions.SetResourceGroupID(resourceGroupId)

	allResources, errListingResources := listResourceInstances(infoSvc, listOptions)
	if errListingResources != nil {
		return nil, fmt.Errorf("error listing resources: %w", errListingResources)
	}

	return allResources, nil
}

func (infoSvc *CloudInfoService) GetReclamationIdFromCRN(CRN string) (string, error) {

	parsed_crn := strings.Split(CRN, ":")
	resourceInstanceID := parsed_crn[7]

	listReclamationsOptions := infoSvc.resourceControllerService.NewListReclamationsOptions()
	listReclamationsOptions = listReclamationsOptions.SetResourceInstanceID(resourceInstanceID)
	reclamationsList, _, err := infoSvc.resourceControllerService.ListReclamations(listReclamationsOptions)
	if err != nil {

		return "", err
	}

	if len(reclamationsList.Resources) == 0 {

		return "", nil

	}

	reclamationID := *reclamationsList.Resources[0].ID

	fmt.Println("reclamation id is ", reclamationID)
	return reclamationID, nil
}

func (infoSvc *CloudInfoService) DeleteInstanceFromReclamationId(reclamationID string) (string, error) {

	fmt.Println("Deleting the instance using reclamation id")

	runReclamationActionOptions := infoSvc.resourceControllerService.NewRunReclamationActionOptions(
		reclamationID,
		"reclaim",
	)

	_, _, err := infoSvc.resourceControllerService.RunReclamationAction(runReclamationActionOptions)
	if err != nil {

		return "", err
	}

	return "instance reclaimed successfully", nil
}

func (infoSvc *CloudInfoService) DeleteInstanceFromReclamationByCRN(CRN string) (string, error) {

	reclamationID, err := infoSvc.GetReclamationIdFromCRN(CRN)

	if err != nil {

		return "", err
	}

	if reclamationID == "" {

		fmt.Println("No reclamation found for the given CRN")
		return "No reclamation found for the given CRN", nil
	}

	_, err = infoSvc.DeleteInstanceFromReclamationId(reclamationID)

	if err != nil {
		return "", err
	}

	return "Instance reclaimed successfully", nil

}

// listResourceInstances will retrieve all resources of a given type for an account
func listResourceInstances(infoSvc *CloudInfoService, options *resourcecontrollerv2.ListResourceInstancesOptions) ([]resourcecontrollerv2.ResourceInstance, error) {
	// this API is paginated, but there is no pager support in the library at this time.
	// we are compensating by inspecting the NextURL and Start values supplied by the API
	// to keep looping through pages
	maxPages := 100
	countPages := 0
	listResourceInstance := []resourcecontrollerv2.ResourceInstance{}
	moreData := true

	for moreData {
		listPage, _, err := infoSvc.resourceControllerService.ListResourceInstances(options)
		if err != nil {
			return nil, fmt.Errorf("error listing PowerVS instances: %w", err)
		}
		countPages += 1

		// add all resources to list
		for _, svcInstance := range listPage.Resources {
			listResourceInstance = append(listResourceInstance, svcInstance)
		}

		// get next page of results if necessary
		// see: https://cloud.ibm.com/apidocs/resource-controller/resource-controller?code=go#list-resource-instances
		if (countPages < maxPages) && listPage.NextURL != nil && *listPage.NextURL != "" {
			moreData = true
			newStart, errNext := core.GetQueryParam(listPage.NextURL, "start")
			if errNext != nil || newStart == nil {
				return nil, fmt.Errorf("error in fetching start value from next_url: %w", errNext)
			}
			options.SetStart(*newStart)
		} else {
			moreData = false
		}
	}

	return listResourceInstance, nil
}

// PrintResources will print a formatted list of resources to stdout
// resources is a list of resources to print
func PrintResources(resources []resourcecontrollerv2.ResourceInstance) {
	// Order of keys to print
	keys := []string{
		"Name",
		"Location",
		"State",
		"ResourceID",
		"CRN",
		"ResourceGroupID",
		"ResourcePlanID",
		"SubType",
		"Type",
		"URL",
		"AccountID",
		"CreatedBy",
		"DashboardURL",
		"GUID",
		"LastOperationState",
		"LastOperationType",
		"ScheduledReclaimBy",
	}
	PrintResourceKey(keys, resources)
}

// PrintResourceKey will print a formatted list of resources to stdout
// keyList is an ordered list of keys to print
// available keys are:
// Name, Location, State, ResourceID, CRN, ResourceGroupID, ResourcePlanID, SubType, Type, URL, AccountID, CreatedBy, DashboardURL, GUID, LastOperationState, LastOperationType, ScheduledReclaimBy
// resources is a list of resources to print
func PrintResourceKey(keyList []string, resources []resourcecontrollerv2.ResourceInstance) {
	for _, resource := range resources {
		resourceMap := map[string]interface{}{
			"Name":               core.StringNilMapper(resource.Name),
			"Location":           core.StringNilMapper(resource.RegionID),
			"State":              core.StringNilMapper(resource.State),
			"ResourceID":         core.StringNilMapper(resource.ResourceID),
			"CRN":                core.StringNilMapper(resource.CRN),
			"ResourceGroupID":    core.StringNilMapper(resource.ResourceGroupID),
			"ResourcePlanID":     core.StringNilMapper(resource.ResourcePlanID),
			"SubType":            core.StringNilMapper(resource.SubType),
			"Type":               core.StringNilMapper(resource.Type),
			"URL":                core.StringNilMapper(resource.URL),
			"AccountID":          core.StringNilMapper(resource.AccountID),
			"CreatedBy":          core.StringNilMapper(resource.CreatedBy),
			"DashboardURL":       core.StringNilMapper(resource.DashboardURL),
			"GUID":               core.StringNilMapper(resource.GUID),
			"LastOperationState": core.StringNilMapper(resource.LastOperation.State),
			"LastOperationType":  core.StringNilMapper(resource.LastOperation.Type),
			"ScheduledReclaimBy": core.StringNilMapper(resource.ScheduledReclaimBy),
		}

		for _, key := range keyList {
			value, ok := resourceMap[key]
			if !ok {
				fmt.Printf("Invalid key: %s\n", key)
				continue
			}
			if value == "" {
				fmt.Printf("%s: N/A\n", key)
			} else {
				fmt.Printf("%s: %v\n", key, value)
			}
		}
		fmt.Println("--------------------")
	}

}
