package cloudinfo

import (
	"fmt"

	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/platform-services-go-sdk/resourcemanagerv2"
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

func (infoSvc *CloudInfoService) CreateResourceGroup(name string) (*resourcemanagerv2.ResCreateResourceGroup, error) {
	fmt.Println("Creating resource group: ", name)
	resourceGroupOptions := infoSvc.resourceManagerService.NewCreateResourceGroupOptions()
	resourceGroupOptions.SetName(name)
	resourceGroup, _, err := infoSvc.resourceManagerService.CreateResourceGroup(resourceGroupOptions)

	if err != nil {
		infoSvc.Logger.Error(fmt.Sprintf("Could not create resource group: %v", err))
	}

	return resourceGroup, err
}

func (infoSvc *CloudInfoService) DeleteResourceGroup(resourceGroupId string) (*core.DetailedResponse, error) {
	fmt.Println("Deleting resource group: ", resourceGroupId)
	resourceGroupOptions := infoSvc.resourceManagerService.NewDeleteResourceGroupOptions(resourceGroupId)
	resp, err := infoSvc.resourceManagerService.DeleteResourceGroup(resourceGroupOptions)

	if err != nil {
		infoSvc.Logger.Error(fmt.Sprintf("Could not delete resource group: %v", err))
	}

	return resp, err
}

func (infoSvc *CloudInfoService) WithNewResourceGroup(name string, task func() error) error {
	fmt.Println("Running task inside resource group context...")
	resourceGroup, _ := infoSvc.CreateResourceGroup(name)
	defer infoSvc.DeleteResourceGroup(*resourceGroup.ID)

	if err := task(); err != nil {
		return fmt.Errorf("task execution failed: %w", err)
	}

	return nil
}
