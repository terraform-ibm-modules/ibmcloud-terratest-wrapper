package cloudinfo

import (
	"github.com/IBM/platform-services-go-sdk/resourcemanagerv2"
	"testing"

	"github.com/IBM/platform-services-go-sdk/resourcecontrollerv2"
	"github.com/stretchr/testify/assert"
)

func TestListResourcesByCrnSvcName(t *testing.T) {
	infoSvc := CloudInfoService{
		resourceControllerService: &resourceControllerServiceMock{},
	}

	var zeroCount int64 = 0
	var twoCount int64 = 2
	var foundCrn string = "crn:v1:bluemix:public:my-service:theregion:a/accountnum:guid::"
	var notFoundCrn string = "crn:v1:bluemix:public:not-my-service:theregion:a/accountnum:guid::"

	// first test, account has zero resources
	t.Run("ZeroTotalResources", func(t *testing.T) {
		infoSvc.resourceControllerService = &resourceControllerServiceMock{
			mockResourceList: &resourcecontrollerv2.ResourceInstancesList{RowsCount: &zeroCount},
		}
		zeroTotalList, zeroTotalErr := infoSvc.ListResourcesByCrnServiceName("my-service")
		assert.Nil(t, zeroTotalErr)
		assert.Empty(t, zeroTotalList)
	})

	// second test, account has resources but not what we are looking for
	t.Run("ZeroFoundResources", func(t *testing.T) {
		infoSvc.resourceControllerService = &resourceControllerServiceMock{
			mockResourceList: &resourcecontrollerv2.ResourceInstancesList{
				RowsCount: &twoCount,
				Resources: []resourcecontrollerv2.ResourceInstance{
					{CRN: &notFoundCrn},
					{CRN: &notFoundCrn},
				},
			},
		}
		zeroList, zeroErr := infoSvc.ListResourcesByCrnServiceName("my-service")
		assert.Nil(t, zeroErr)
		assert.Empty(t, zeroList)
	})

	// third test, account has one result we are looking for
	t.Run("HasFoundResources", func(t *testing.T) {
		infoSvc.resourceControllerService = &resourceControllerServiceMock{
			mockResourceList: &resourcecontrollerv2.ResourceInstancesList{
				RowsCount: &twoCount,
				Resources: []resourcecontrollerv2.ResourceInstance{
					{CRN: &notFoundCrn},
					{CRN: &foundCrn},
				},
			},
		}
		hasList, hasErr := infoSvc.ListResourcesByCrnServiceName("my-service")
		assert.Nil(t, hasErr)
		assert.NotEmpty(t, hasList)
		assert.Equal(t, len(hasList), 1)
	})
}

func TestListResourcesByGroupID(t *testing.T) {
	infoSvc := CloudInfoService{
		resourceControllerService: &resourceControllerServiceMock{},
	}

	var zeroCount int64 = 0
	var twoCount int64 = 2
	var groupId string = "group-id"

	// first test, group has zero resources
	t.Run("ZeroTotalResources", func(t *testing.T) {
		infoSvc.resourceControllerService = &resourceControllerServiceMock{
			mockResourceList: &resourcecontrollerv2.ResourceInstancesList{RowsCount: &zeroCount},
		}
		zeroTotalList, zeroTotalErr := infoSvc.ListResourcesByGroupID(groupId)
		assert.Nil(t, zeroTotalErr)
		assert.Empty(t, zeroTotalList)
	})

	// second test, group has two resources
	t.Run("TwoTotalResources", func(t *testing.T) {
		infoSvc.resourceControllerService = &resourceControllerServiceMock{
			mockResourceList: &resourcecontrollerv2.ResourceInstancesList{
				RowsCount: &twoCount,
				Resources: []resourcecontrollerv2.ResourceInstance{
					{ResourceGroupID: &groupId},
					{ResourceGroupID: &groupId},
				},
			},
		}
		twoTotalList, twoTotalErr := infoSvc.ListResourcesByGroupID(groupId)
		assert.Nil(t, twoTotalErr)
		assert.NotEmpty(t, twoTotalList)
		assert.Equal(t, len(twoTotalList), 2)
	})
}

func TestListResourcesByGroupName(t *testing.T) {
	infoSvc := CloudInfoService{
		resourceControllerService: &resourceControllerServiceMock{},
		resourceManagerService:    &resourceManagerServiceMock{},
	}

	var groupName1 string = "group-name-1"
	var groupId1 string = "group-id-1"
	var groupName2 string = "group-name-2"
	var groupId2 string = "group-id-2"

	// first test, group has zero resources
	t.Run("ZeroTotalResources", func(t *testing.T) {
		infoSvc.resourceControllerService = &resourceControllerServiceMock{
			mockResourceList: &resourcecontrollerv2.ResourceInstancesList{},
		}
		infoSvc.resourceManagerService = &resourceManagerServiceMock{
			mockResourceGroupList: &resourcemanagerv2.ResourceGroupList{
				Resources: []resourcemanagerv2.ResourceGroup{
					{ID: &groupId1, Name: &groupName1},
				},
			},
			resourceGroups: map[string]string{
				groupName1: groupId1,
			},
		}
		zeroTotalList, zeroTotalErr := infoSvc.ListResourcesByGroupName(groupName1)
		assert.Nil(t, zeroTotalErr)
		assert.Empty(t, zeroTotalList)
	})

	// second test, group has two resources
	t.Run("TwoTotalResources", func(t *testing.T) {
		infoSvc.resourceControllerService = &resourceControllerServiceMock{
			mockResourceList: &resourcecontrollerv2.ResourceInstancesList{
				Resources: []resourcecontrollerv2.ResourceInstance{
					{ResourceGroupID: &groupId1},
					{ResourceGroupID: &groupId2},
				},
			},
		}
		infoSvc.resourceManagerService = &resourceManagerServiceMock{
			mockResourceGroupList: &resourcemanagerv2.ResourceGroupList{
				Resources: []resourcemanagerv2.ResourceGroup{
					{ID: &groupId1, Name: &groupName1},
					{ID: &groupId2, Name: &groupName2},
				},
			},
			resourceGroups: map[string]string{
				groupName1: groupId1,
				groupName2: groupId2,
			},
		}
		twoTotalList, twoTotalErr := infoSvc.ListResourcesByGroupName(groupName1)
		assert.Nil(t, twoTotalErr)
		assert.NotEmpty(t, twoTotalList)
		assert.Equal(t, len(twoTotalList), 2)
	})
}

func TestListResourcesRealGroup(t *testing.T) {
	infoSvc, _ := NewCloudInfoServiceFromEnv("TF_VAR_ibmcloud_api_key", CloudInfoServiceOptions{})

	expectedResourceNames := []string{
		"conjoined-reformat-immobile-frenzy",
		"harbor-underling-bouncy-uneven",
		"retread-gracious-countdown-shrubs",
	}

	resources, listError := infoSvc.ListResourcesByGroupName("vsi-e6ackt-workload-rg")
	assert.Nil(t, listError, "Error listing resources")
	assert.NotEmpty(t, resources, "No resources found")
	resourceNames := make([]string, len(resources))
	if len(resources) > 0 {
		//		create a list of resource names
		for i, resource := range resources {
			resourceNames[i] = *resource.Name
		}
	}
	assert.Equal(t, len(expectedResourceNames), len(resources), "Expected resources not found:\n%v\nGot:\n%v", expectedResourceNames, resourceNames)

}
