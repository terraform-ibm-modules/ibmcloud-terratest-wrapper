package cloudinfo

import (
	"testing"

	"github.com/IBM/platform-services-go-sdk/resourcecontrollerv2"
	"github.com/stretchr/testify/assert"
)

func TestListPowerWorkspaces(t *testing.T) {
	infoSvc := CloudInfoService{
		resourceControllerService: &resourceControllerServiceMock{},
	}

	var zeroCount int64 = 0
	var twoCount int64 = 2
	var powerCrn string = "crn:v1:bluemix:public:power-iaas:theregion:a/accountnum:guid::"
	var notPowerCrn string = "crn:v1:bluemix:public:logdna:theregion:a/accountnum:guid::"

	// first test, account has zero resources
	t.Run("ZeroTotalResources", func(t *testing.T) {
		infoSvc.resourceControllerService = &resourceControllerServiceMock{
			mockResourceList: &resourcecontrollerv2.ResourceInstancesList{RowsCount: &zeroCount},
		}
		zeroTotalList, zeroTotalErr := infoSvc.ListPowerWorkspaces()
		assert.Nil(t, zeroTotalErr)
		assert.Empty(t, zeroTotalList)
	})

	// second test, account has resources but no power iaas
	t.Run("ZeroPowerResources", func(t *testing.T) {
		infoSvc.resourceControllerService = &resourceControllerServiceMock{
			mockResourceList: &resourcecontrollerv2.ResourceInstancesList{
				RowsCount: &twoCount,
				Resources: []resourcecontrollerv2.ResourceInstance{
					{CRN: &notPowerCrn},
					{CRN: &notPowerCrn},
				},
			},
		}
		zeroList, zeroErr := infoSvc.ListPowerWorkspaces()
		assert.Nil(t, zeroErr)
		assert.Empty(t, zeroList)
	})

	// third test, account has a power iaas service single page of results
	t.Run("HasPowerResources", func(t *testing.T) {
		infoSvc.resourceControllerService = &resourceControllerServiceMock{
			mockResourceList: &resourcecontrollerv2.ResourceInstancesList{
				RowsCount: &twoCount,
				Resources: []resourcecontrollerv2.ResourceInstance{
					{CRN: &notPowerCrn},
					{CRN: &powerCrn},
				},
			},
		}
		hasList, hasErr := infoSvc.ListPowerWorkspaces()
		assert.Nil(t, hasErr)
		assert.NotEmpty(t, hasList)
		assert.Equal(t, len(hasList), 1)
	})
}
