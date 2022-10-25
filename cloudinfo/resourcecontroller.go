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

	// this API is paginated, but there is no pager support in the library at this time.
	// we are compensating by inspecting the NextURL and Start values supplied by the API
	// to keep looping through pages
	maxPages := 100
	countPages := 0
	listResourceInstance := []resourcecontrollerv2.ResourceInstance{}
	moreData := true

	for moreData {
		listPage, _, err := infoSvc.resourceControllerService.ListResourceInstances(listOptions)
		if err != nil {
			return nil, fmt.Errorf("error listing PowerVS instances: %w", err)
		}
		countPages += 1

		// search through instances on current page looking for only power-iaas
		for _, svcInstance := range listPage.Resources {
			crn, crnErr := bluemix_crn.Parse(*svcInstance.CRN)
			// if crn errors, do it the old fashioned way
			if crnErr == nil {
				// do it the proper way
				if crn.ServiceName == crnServiceName {
					listResourceInstance = append(listResourceInstance, svcInstance)
				}
			} else {
				// do it the ugly way
				if strings.Contains(*svcInstance.CRN, crnServiceName) {
					listResourceInstance = append(listResourceInstance, svcInstance)
				}
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

	return listResourceInstance, nil
}
