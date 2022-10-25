package cloudinfo

import (
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
