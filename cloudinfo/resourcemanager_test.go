package cloudinfo

import (
	"errors"
	"testing"

	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/platform-services-go-sdk/resourcemanagerv2"
	"github.com/stretchr/testify/assert"
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

func TestCreateResourceGroup(t *testing.T) {
	infoSvc := CloudInfoService{
		resourceManagerService: &resourceManagerServiceMock{
			mockResCreateResourceGroup: &resourcemanagerv2.ResCreateResourceGroup{
				ID: core.StringPtr("test-id"),
			},
		},
		iamIdentityService: &iamIdentityServiceMock{},
		ApiKey:             "mockapikey",
	}

	t.Run("CreateResourceGroup_Success", func(t *testing.T) {
		resourceGroup, resp, err := infoSvc.CreateResourceGroup("test-group")
		assert.NotNil(t, resp)
		assert.Nil(t, err)
		assert.Equal(t, "test-id", *resourceGroup.ID)
	})
}

func TestDeleteResourceGroup(t *testing.T) {
	infoSvc := CloudInfoService{
		resourceManagerService: &resourceManagerServiceMock{},
	}

	t.Run("DeleteResourceGroup", func(t *testing.T) {
		infoSvc.resourceManagerService = &resourceManagerServiceMock{
			mockDeleteResourceGroup: &core.DetailedResponse{StatusCode: 200},
		}
		resp, err := infoSvc.DeleteResourceGroup("test-group")
		assert.Equal(t, resp.StatusCode, 200)
		assert.Nil(t, err)
	})
}

func TestWithNewResourceGroup(t *testing.T) {

	t.Run("WithNewResourceGroup_Success", func(t *testing.T) {
		infoSvc := CloudInfoService{
			ApiKey: "mockapikey",
			resourceManagerService: &resourceManagerServiceMock{
				mockResCreateResourceGroup: &resourcemanagerv2.ResCreateResourceGroup{
					ID: core.StringPtr("test-id"),
				},
			},
			iamIdentityService: &iamIdentityServiceMock{},
		}

		task := func() error {
			// Simulate successful task
			return nil
		}

		err := infoSvc.WithNewResourceGroup("test-group", task)
		assert.Nil(t, err)
	})

	t.Run("WithNewResourceGroup_TaskFails", func(t *testing.T) {
		infoSvc := CloudInfoService{
			ApiKey: "mockapikey",
			resourceManagerService: &resourceManagerServiceMock{
				mockResCreateResourceGroup: &resourcemanagerv2.ResCreateResourceGroup{
					ID: core.StringPtr("test-id"),
				},
			},
			iamIdentityService: &iamIdentityServiceMock{},
		}

		task := func() error {
			return errors.New("task failed")
		}

		err := infoSvc.WithNewResourceGroup("test-group", task)
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "task execution failed")
	})
}
