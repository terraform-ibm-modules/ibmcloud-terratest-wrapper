package cloudinfo

import (
	"errors"
	"testing"

	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/platform-services-go-sdk/resourcecontrollerv2"
	"github.com/stretchr/testify/assert"
)

func TestLeastVpcAllAvailRegions(t *testing.T) {
	vpcService := new(vpcServiceMock)
	resourceControllerService := new(resourceControllerServiceMock)

	// first test, low priority wins
	infoSvc := CloudInfoService{
		vpcService:                vpcService,
		resourceControllerService: resourceControllerService,
		regionsData: []RegionData{
			{Name: "reg-1-10", UseForTest: true, TestPriority: 1},
			{Name: "reg-2-5", UseForTest: true, TestPriority: 2},
			{Name: "reg-3-5", UseForTest: true, TestPriority: 3},
		},
	}

	t.Run("LowestPriorityWins", func(t *testing.T) {
		bestregion, regErr := infoSvc.GetLeastVpcTestRegion()
		if assert.Nil(t, regErr) {
			assert.Equal(t, "reg-2-5", bestregion, "Wrong VPC region returned")
		}
	})

	// second test, region with zero wins no matter
	infoSvc.regionsData = []RegionData{
		{Name: "reg-1-0", UseForTest: true, TestPriority: 1},
		{Name: "reg-2-5", UseForTest: true, TestPriority: 2},
		{Name: "reg-3-3", UseForTest: true, TestPriority: 3},
	}

	t.Run("FirstZeroWins", func(t *testing.T) {
		bestregion, regErr := infoSvc.GetLeastVpcTestRegion()
		if assert.Nil(t, regErr) {
			assert.Equal(t, "reg-1-0", bestregion, "Wrong VPC region returned")
		}
	})

	// third test, do not include non test regions
	infoSvc.regionsData = []RegionData{
		{Name: "reg-3-3", UseForTest: true, TestPriority: 3},
		{Name: "reg-2-1", UseForTest: false, TestPriority: 2},
		{Name: "reg-1-10", UseForTest: true, TestPriority: 1},
	}

	t.Run("ExcludeRegions", func(t *testing.T) {
		bestregion, regErr := infoSvc.GetLeastVpcTestRegion()
		if assert.Nil(t, regErr) {
			assert.Equal(t, "reg-3-3", bestregion, "Wrong VPC region returned")
		}
	})

	// fourth test, use all avail regions if no prefs
	infoSvc.regionsData = []RegionData{}

	t.Run("UseAllAvailRegions", func(t *testing.T) {
		bestregion, regErr := infoSvc.GetLeastVpcTestRegion()
		if assert.Nil(t, regErr) {
			assert.Equal(t, "regavail-3-1", bestregion, "Wrong VPC region returned")
		}
	})

	// fifth test, nothing available
	infoSvc.regionsData = []RegionData{
		{Name: "reg-1-10", UseForTest: false, TestPriority: 1},
		{Name: "reg-2-1", UseForTest: false, TestPriority: 2},
		{Name: "reg-3-3", UseForTest: false, TestPriority: 3},
	}

	t.Run("NoRegionsAvailable", func(t *testing.T) {
		_, regErr := infoSvc.GetLeastVpcTestRegion()
		assert.NotNil(t, regErr, "error expected when no region returned")
	})

	// sixth test, exclude regions with activity tracker
	infoSvc.regionsData = []RegionData{
		{Name: "reg-1-10", UseForTest: true, TestPriority: 1},
		{Name: "reg-2-1", UseForTest: true, TestPriority: 2},
		{Name: "reg-3-3", UseForTest: true, TestPriority: 3},
	}
	var twoCount int64 = 2
	resourceLogCrn := "crn:v1:bluemix:public:logdna:reg-3-3:a/accountnum:guid::"
	resourceATCrn := "crn:v1:bluemix:public:logdnaat:reg-2-1:a/accountnum:guid::"
	infoSvc.resourceControllerService = &resourceControllerServiceMock{
		mockResourceList: &resourcecontrollerv2.ResourceInstancesList{
			RowsCount: &twoCount,
			Resources: []resourcecontrollerv2.ResourceInstance{
				{CRN: &resourceLogCrn, RegionID: &infoSvc.regionsData[2].Name},
				{CRN: &resourceATCrn, RegionID: &infoSvc.regionsData[1].Name},
			},
		},
	}
	t.Run("ActivityTrackerExclude", func(t *testing.T) {
		bestregion, regErr := infoSvc.GetLeastVpcTestRegionWithoutActivityTracker()
		if assert.Nil(t, regErr, "unexpected error returned") {
			assert.Equal(t, "reg-3-3", bestregion, "Wrong VPC region returned")
		}
	})
}
func TestLeastSdnlbAllAvailRegions(t *testing.T) {
	vpcService := new(vpcServiceMock)
	resourceControllerService := new(resourceControllerServiceMock)

	//create main cloud service objects with mock service and region data
	infoSvc := CloudInfoService{
		vpcService:                vpcService,
		resourceControllerService: resourceControllerService,
		regionsData: []RegionData{
			{Name: "reg-1-10", UseForTest: true, TestPriority: 1},
			{Name: "reg-2-5", UseForTest: true, TestPriority: 2},
			{Name: "reg-3-5", UseForTest: true, TestPriority: 3},
		},
	}
	// first test, low priority wins
	t.Run("LowestPriorityWins", func(t *testing.T) {
		bestregion, regErr := infoSvc.GetLeastSdnlbTestRegion("us-south")
		if assert.Nil(t, regErr) {
			assert.Equal(t, "reg-2-5", bestregion, "Wrong SDN LB region returned")
		}
	})

	// second test, region with zero wins no matter
	infoSvc.regionsData = []RegionData{
		{Name: "reg-1-0", UseForTest: true, TestPriority: 1},
		{Name: "reg-2-5", UseForTest: true, TestPriority: 2},
		{Name: "reg-3-3", UseForTest: true, TestPriority: 3},
	}

	t.Run("FirstZeroWins", func(t *testing.T) {
		bestregion, regErr := infoSvc.GetLeastSdnlbTestRegion("us-south")
		if assert.Nil(t, regErr) {
			assert.Equal(t, "reg-1-0", bestregion, "Wrong SDN LB region returned")
		}
	})

	// third test, do not include non test regions
	infoSvc.regionsData = []RegionData{
		{Name: "reg-3-3", UseForTest: true, TestPriority: 3},
		{Name: "reg-2-1", UseForTest: false, TestPriority: 2},
		{Name: "reg-1-10", UseForTest: true, TestPriority: 1},
	}

	t.Run("ExcludeRegions", func(t *testing.T) {
		bestregion, regErr := infoSvc.GetLeastSdnlbTestRegion("us-south")
		if assert.Nil(t, regErr) {
			assert.Equal(t, "reg-3-3", bestregion, "Wrong SDN LB region returned")
		}
	})

	// fourth test, use all avail regions if no prefs
	infoSvc.regionsData = []RegionData{}

	t.Run("UseAllAvailRegions", func(t *testing.T) {
		bestregion, regErr := infoSvc.GetLeastSdnlbTestRegion("us-south")
		if assert.Nil(t, regErr) {
			assert.Equal(t, "regavail-3-1", bestregion, "Wrong SDN LB region returned")
		}
	})

	// fifth test, nothing available - should return default region
	infoSvc.regionsData = []RegionData{
		{Name: "reg-1-10", UseForTest: false, TestPriority: 1},
		{Name: "reg-2-1", UseForTest: false, TestPriority: 2},
		{Name: "reg-3-3", UseForTest: false, TestPriority: 3},
	}

	t.Run("NoRegionsAvailable", func(t *testing.T) {
		bestregion, regErr := infoSvc.GetLeastSdnlbTestRegion("us-south")
		assert.Nil(t, regErr, "should not error when default region provided")
		assert.Equal(t, "us-south", bestregion, "should return default region when no regions available")
	})

	// sixth test, forced failure
	t.Run("ErrorFromGetTestRegionsByPriority", func(t *testing.T) {
		// Create a mock that will fail on GetRegion call
		vpcServiceErr := &vpcServiceMock{
			shouldFailGetRegion: true,
			getRegionError:      errors.New("failed to get region details"),
		}

		infoSvcErr := CloudInfoService{
			vpcService:                vpcServiceErr,
			resourceControllerService: resourceControllerService,
			regionsData: []RegionData{
				{Name: "reg-1-10", UseForTest: true, TestPriority: 1},
			},
		}

		_, err := infoSvcErr.GetLeastSdnlbTestRegion("us-south")
		assert.NotNil(t, err, "expected error from GetTestRegionsByPriority")
	})

	// seventh test, error from SetServiceURL
	t.Run("ErrorFromSetServiceURL", func(t *testing.T) {
		// Create a mock that will fail on SetServiceURL call
		vpcServiceErr := &vpcServiceMock{
			shouldFailSetServiceURL: true,
			setServiceURLError:      errors.New("failed to set service URL"),
		}

		infoSvcErr := CloudInfoService{
			vpcService:                vpcServiceErr,
			resourceControllerService: resourceControllerService,
			regionsData: []RegionData{
				{Name: "reg-1-10", UseForTest: true, TestPriority: 1},
			},
		}

		_, err := infoSvcErr.GetLeastSdnlbTestRegion("us-south")
		assert.NotNil(t, err, "expected error from SetServiceURL")
		assert.Contains(t, err.Error(), "failed to set service URL")
	})

	// eighth test, error from ListLoadBalancers
	t.Run("ErrorFromListLoadBalancers", func(t *testing.T) {
		// Create a mock that will fail on ListLoadBalancers call
		vpcServiceErr := &vpcServiceMock{
			shouldFailListLoadBalancers: true,
			listLoadBalancersError:      errors.New("failed to list load balancers"),
		}

		infoSvcErr := CloudInfoService{
			vpcService:                vpcServiceErr,
			resourceControllerService: resourceControllerService,
			regionsData: []RegionData{
				{Name: "reg-1-10", UseForTest: true, TestPriority: 1},
			},
		}

		_, err := infoSvcErr.GetLeastSdnlbTestRegion("us-south")
		assert.NotNil(t, err, "expected error from ListLoadBalancers")
		assert.Contains(t, err.Error(), "failed to list load balancers")
	})
}

// TestRegionSelector tests the new region selector functions
func TestRegionSelector(t *testing.T) {
	vpcService := new(vpcServiceMock)

	t.Run("GetRegionWithoutService", func(t *testing.T) {
		// Mock: reg-2 and reg-3 have test service, reg-1 doesn't
		region2 := "reg-2-1"
		region3 := "reg-3-1"
		instanceName1 := "test-instance-1"
		instanceName2 := "test-instance-2"
		var twoCount int64 = 2
		serviceCrn1 := "crn:v1:bluemix:public:aiopenscale:reg-2-1:a/account:::"
		serviceCrn2 := "crn:v1:bluemix:public:aiopenscale:reg-3-1:a/account:::"

		resourceControllerService := &resourceControllerServiceMock{
			mockResourceList: &resourcecontrollerv2.ResourceInstancesList{
				RowsCount: &twoCount,
				Resources: []resourcecontrollerv2.ResourceInstance{
					{CRN: &serviceCrn1, RegionID: &region2, Name: &instanceName1},
					{CRN: &serviceCrn2, RegionID: &region3, Name: &instanceName2},
				},
			},
		}

		infoSvc := CloudInfoService{
			vpcService:                vpcService,
			resourceControllerService: resourceControllerService,
			regionsData: []RegionData{
				{Name: "reg-1-0", UseForTest: true, TestPriority: 1},
				{Name: "reg-2-1", UseForTest: true, TestPriority: 2},
				{Name: "reg-3-1", UseForTest: true, TestPriority: 3},
			},
		}

		region, err := infoSvc.GetRegionWithoutService("aiopenscale")
		assert.NoError(t, err)
		assert.Equal(t, "reg-1-0", region, "Should select reg-1-0 (no service instances)")
	})

	t.Run("GetRegionWithLeastResources", func(t *testing.T) {
		// Mock: reg-3 has 3, reg-2 has 1, reg-1 has 0 of a test service
		region2 := "reg-2-1"
		region3 := "reg-3-3"
		serviceName1 := "test-service-1"
		serviceName2 := "test-service-2"
		serviceName3 := "test-service-3"
		serviceName4 := "test-service-4"
		var fourCount int64 = 4
		serviceCrn1 := "crn:v1:bluemix:public:testservice:reg-3-3:a/account:::"
		serviceCrn2 := "crn:v1:bluemix:public:testservice:reg-3-3:a/account:::"
		serviceCrn3 := "crn:v1:bluemix:public:testservice:reg-3-3:a/account:::"
		serviceCrn4 := "crn:v1:bluemix:public:testservice:reg-2-1:a/account:::"

		resourceControllerService := &resourceControllerServiceMock{
			mockResourceList: &resourcecontrollerv2.ResourceInstancesList{
				RowsCount: &fourCount,
				Resources: []resourcecontrollerv2.ResourceInstance{
					{CRN: &serviceCrn1, RegionID: &region3, Name: &serviceName1},
					{CRN: &serviceCrn2, RegionID: &region3, Name: &serviceName2},
					{CRN: &serviceCrn3, RegionID: &region3, Name: &serviceName3},
					{CRN: &serviceCrn4, RegionID: &region2, Name: &serviceName4},
				},
			},
		}

		infoSvc := CloudInfoService{
			vpcService:                vpcService,
			resourceControllerService: resourceControllerService,
			regionsData: []RegionData{
				{Name: "reg-1-0", UseForTest: true, TestPriority: 1},
				{Name: "reg-2-1", UseForTest: true, TestPriority: 2},
				{Name: "reg-3-3", UseForTest: true, TestPriority: 3},
			},
		}

		region, err := infoSvc.GetRegionWithLeastResources("testservice")
		assert.NoError(t, err)
		assert.Equal(t, "reg-1-0", region, "Should select reg-1-0 (zero test service instances)")
	})

}

func TestLoadRegionPrefs(t *testing.T) {
	infoSvc := CloudInfoService{}

	t.Run("LoadDefaultFromYaml", func(t *testing.T) {
		err := infoSvc.LoadRegionPrefsFromFile("testdata/region-default-prefs.yaml")
		if assert.Nil(t, err) {
			assert.Equal(t, 9, len(infoSvc.regionsData), "invalid record count")
			assert.Equal(t, "ca-tor", infoSvc.regionsData[2].Name, "wrong name in array")
			assert.Equal(t, 2, infoSvc.regionsData[2].TestPriority, "wrong priority in array")
		}
	})

	t.Run("FileNotExist", func(t *testing.T) {
		err := infoSvc.LoadRegionPrefsFromFile("testdata/not-exist.yaml")
		assert.NotNil(t, err, "expected error on missing file")
	})

	t.Run("NotYamlFile", func(t *testing.T) {
		err := infoSvc.LoadRegionPrefsFromFile("testdata/region-bad-format.txt")
		assert.NotNil(t, err, "expected error on bad file format")
	})
}

func TestLeastPowerConnectionZone(t *testing.T) {
	infoSvc := CloudInfoService{}

	// first test, there is no provided region preference data
	t.Run("NoRegionData", func(t *testing.T) {
		bestZone, bestErr := infoSvc.GetLeastPowerConnectionZone()
		assert.NotNil(t, bestErr)
		assert.Empty(t, bestZone)
		assert.ErrorContains(t, bestErr, "no available zones")
	})
}

func TestRegionHasActivityTracker(t *testing.T) {
	id1, id2, id3 := "1", "2", "3"
	region1, region2, region3 := "region-1", "region-2", "region-3"

	atList := []resourcecontrollerv2.ResourceInstance{
		{ID: &id1, RegionID: &region1},
		{ID: &id2, RegionID: &region2},
		{ID: &id3, RegionID: &region3},
	}

	t.Run("ActivityTrackerRegionNotFound", func(t *testing.T) {
		wasNotFound := regionHasActivityTracker("region-notfound", atList)
		assert.False(t, wasNotFound)
	})

	t.Run("ActivityTrackerRegionFound", func(t *testing.T) {
		wasFound := regionHasActivityTracker(region2, atList)
		assert.True(t, wasFound)
	})

	t.Run("EmptyList", func(t *testing.T) {
		wasEmpty := regionHasActivityTracker(region1, []resourcecontrollerv2.ResourceInstance{})
		assert.False(t, wasEmpty)
	})
}

func TestRemoveRegionForTest(t *testing.T) {
	infoSvc := CloudInfoService{}

	t.Run("EmptyRegionList", func(t *testing.T) {
		infoSvc.RemoveRegionForTest("test-region")
		assert.Empty(t, infoSvc.regionsData)
	})

	infoSvc.regionsData = []RegionData{
		{Name: "test-region-1", UseForTest: true},
		{Name: "test-region-2", UseForTest: true},
		{Name: "test-region-3", UseForTest: true},
	}

	t.Run("RegionNotFound", func(t *testing.T) {
		infoSvc.RemoveRegionForTest("not-found-region")
		// all should be true still
		assert.True(t, infoSvc.regionsData[0].UseForTest)
		assert.True(t, infoSvc.regionsData[1].UseForTest)
		assert.True(t, infoSvc.regionsData[2].UseForTest)
	})

	t.Run("RegionFound", func(t *testing.T) {
		infoSvc.RemoveRegionForTest("test-region-2")
		// only one should be false
		assert.True(t, infoSvc.regionsData[0].UseForTest)
		assert.False(t, infoSvc.regionsData[1].UseForTest)
		assert.True(t, infoSvc.regionsData[2].UseForTest)
	})
}

func TestGetRegionWithLeastTransitGateways(t *testing.T) {
	t.Run("SelectsRegionWithFewestInstances", func(t *testing.T) {
		// Test with transit gateways distributed across regions
		vpcService := new(vpcServiceMock)
		resourceControllerService := new(resourceControllerServiceMock)

		usSouth := "us-south"
		usEast := "us-east"

		mockInstances := []resourcecontrollerv2.ResourceInstance{
			// 3 transit gateways in us-south
			{RegionID: &usSouth, Name: core.StringPtr("tg-1"), CRN: core.StringPtr("crn:v1:bluemix:public:transit:us-south:a/account1:::")},
			{RegionID: &usSouth, Name: core.StringPtr("tg-2"), CRN: core.StringPtr("crn:v1:bluemix:public:transit:us-south:a/account1:::")},
			{RegionID: &usSouth, Name: core.StringPtr("tg-3"), CRN: core.StringPtr("crn:v1:bluemix:public:transit:us-south:a/account1:::")},
			// 1 transit gateway in us-east (should be selected)
			{RegionID: &usEast, Name: core.StringPtr("tg-4"), CRN: core.StringPtr("crn:v1:bluemix:public:transit:us-east:a/account1:::")},
		}

		mockCount := int64(len(mockInstances))
		resourceControllerService.mockResourceList = &resourcecontrollerv2.ResourceInstancesList{
			RowsCount: &mockCount,
			Resources: mockInstances,
		}

		infoSvc := CloudInfoService{
			vpcService:                vpcService,
			resourceControllerService: resourceControllerService,
			regionsData: []RegionData{
				{Name: "us-south", UseForTest: true, TestPriority: 1},
				{Name: "us-east", UseForTest: true, TestPriority: 2},
			},
		}

		region, err := infoSvc.GetRegionWithLeastTransitGateways()
		assert.NoError(t, err)
		assert.Equal(t, "us-east", region, "Should select region with fewest transit gateways")
	})

	t.Run("RespectsRegionPriority", func(t *testing.T) {
		// Test that priority is respected when counts are equal
		vpcService := new(vpcServiceMock)
		resourceControllerService := new(resourceControllerServiceMock)

		usSouth := "us-south"
		usEast := "us-east"

		mockInstances := []resourcecontrollerv2.ResourceInstance{
			// 2 in each region
			{RegionID: &usSouth, Name: core.StringPtr("tg-1"), CRN: core.StringPtr("crn:v1:bluemix:public:transit:us-south:a/account1:::")},
			{RegionID: &usSouth, Name: core.StringPtr("tg-2"), CRN: core.StringPtr("crn:v1:bluemix:public:transit:us-south:a/account1:::")},
			{RegionID: &usEast, Name: core.StringPtr("tg-3"), CRN: core.StringPtr("crn:v1:bluemix:public:transit:us-east:a/account1:::")},
			{RegionID: &usEast, Name: core.StringPtr("tg-4"), CRN: core.StringPtr("crn:v1:bluemix:public:transit:us-east:a/account1:::")},
		}

		mockCount := int64(len(mockInstances))
		resourceControllerService.mockResourceList = &resourcecontrollerv2.ResourceInstancesList{
			RowsCount: &mockCount,
			Resources: mockInstances,
		}

		infoSvc := CloudInfoService{
			vpcService:                vpcService,
			resourceControllerService: resourceControllerService,
			regionsData: []RegionData{
				{Name: "us-south", UseForTest: true, TestPriority: 1}, // Higher priority
				{Name: "us-east", UseForTest: true, TestPriority: 2},
			},
		}

		region, err := infoSvc.GetRegionWithLeastTransitGateways()
		assert.NoError(t, err)
		assert.Equal(t, "us-south", region, "Should select highest priority region when counts are equal")
	})
}

func stringPtr(s string) *string {
	return &s
}
