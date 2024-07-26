package cloudinfo

import (
	"fmt"
)

// GetResourceGroupIDByName will retrieve the resource group ID for a given resource group name
// resourceGroupName is the name of the resource group
// returns the resource group ID
func (infoSvc *CloudInfoService) GetResourceGroupIDByName(resourceGroupName string) (string, error) {
	listResourceGroupsOptions := infoSvc.resourceManagerService.NewListResourceGroupsOptions()
	resourceGroups, _, err := infoSvc.resourceManagerService.ListResourceGroups(listResourceGroupsOptions)
	if err != nil {
		return "", fmt.Errorf("error listing resource groups: %w", err)
	}

	for _, group := range resourceGroups.Resources {
		if *group.Name == resourceGroupName {
			return *group.ID, nil
		}
	}

	return "", fmt.Errorf("resource group with name %s not found", resourceGroupName)
}
