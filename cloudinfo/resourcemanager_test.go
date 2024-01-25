package cloudinfo

import (
	"github.com/IBM/platform-services-go-sdk/resourcemanagerv2"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetResourceGroupIDByName(t *testing.T) {
	infoSvc := CloudInfoService{
		resourceManagerService: &resourceManagerServiceMock{},
	}

	var groupName1 string = "group-name-1"
	var groupId1 string = "group-id-1"
	var groupName2 string = "group-name-2"
	var groupId2 string = "group-id-2"

	// first test, group name does not exist
	t.Run("GroupNameDoesNotExist", func(t *testing.T) {
		infoSvc.resourceManagerService = &resourceManagerServiceMock{
			mockResourceGroupList: &resourcemanagerv2.ResourceGroupList{
				Resources: []resourcemanagerv2.ResourceGroup{
					{ID: &groupId1, Name: &groupName1},
					{ID: &groupId2, Name: &groupName2},
				},
			},
		}
		_, err := infoSvc.GetResourceGroupIDByName("non-existing-group-name")
		assert.NotNil(t, err)
	})

	// second test, group name exists
	t.Run("GroupNameExists", func(t *testing.T) {
		infoSvc.resourceManagerService = &resourceManagerServiceMock{
			mockResourceGroupList: &resourcemanagerv2.ResourceGroupList{
				Resources: []resourcemanagerv2.ResourceGroup{
					{ID: &groupId1, Name: &groupName1},
					{ID: &groupId2, Name: &groupName2},
				},
			},
		}
		resourceGroupId, err := infoSvc.GetResourceGroupIDByName(groupName1)
		assert.Nil(t, err)
		assert.Equal(t, groupId1, resourceGroupId)
	})
}
