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

// CreateResourceGroup will create a resource group with a given name using the Resource Manager service.
func (infoSvc *CloudInfoService) CreateResourceGroup(name string) (*resourcemanagerv2.ResCreateResourceGroup, *core.DetailedResponse, error) {
	resourceGroupOptions := infoSvc.resourceManagerService.NewCreateResourceGroupOptions()
	resourceGroupOptions.SetName(name)
	return infoSvc.resourceManagerService.CreateResourceGroup(resourceGroupOptions)
}

// DeleteResourceGroup will delete a resource group with a given ID using the Resource Manager service.
func (infoSvc *CloudInfoService) DeleteResourceGroup(resourceGroupId string) (*core.DetailedResponse, error) {
	resourceGroupOptions := infoSvc.resourceManagerService.NewDeleteResourceGroupOptions(resourceGroupId)
	return infoSvc.resourceManagerService.DeleteResourceGroup(resourceGroupOptions)
}

// WithNewResourceGroup is a context manager that will create a resource group,
// execute a given task that uses the created resource group and deletes the resource group
// example running schematic test in a given resource group:
//
//	err = sharedInfoSvc.WithNewResourceGroup("myResourceGroup", func() error {
//		return options.RunSchematicTest()
//	})
func (infoSvc *CloudInfoService) WithNewResourceGroup(name string, task func() error) error {
	fmt.Println("Running task inside resource group context...")
	resourceGroup, resp, err := infoSvc.CreateResourceGroup(name)
	fmt.Printf("Created resource group %s with ID %s", name, *resourceGroup.ID)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	defer infoSvc.DeleteResourceGroup(*resourceGroup.ID)

	if err := task(); err != nil {
		return fmt.Errorf("task execution failed: %w", err)
	}

	return nil
}
