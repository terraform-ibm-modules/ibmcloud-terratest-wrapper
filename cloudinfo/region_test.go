package cloudinfo

import (
	"testing"

	transitgatewayapisv1 "github.com/IBM/networking-go-sdk/transitgatewayapisv1"
	"github.com/IBM/platform-services-go-sdk/resourcecontrollerv2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
	t.Run("Success - Returns region with least transit gateways", func(t *testing.T) {
		// Create mock transit gateway service
		mockTGService := new(transitGatewayServiceMock)

		// Create mock VPC service
		mockVpcService := new(vpcServiceMock)

		// Setup test data - 3 transit gateways in different regions
		// Use region names that match the mock VPC service regions
		location1 := "regavail-1-5"
		location2 := "regavail-2-10"
		location3 := "regavail-1-5"

		tgw1 := transitgatewayapisv1.TransitGateway{Location: &location1}
		tgw2 := transitgatewayapisv1.TransitGateway{Location: &location2}
		tgw3 := transitgatewayapisv1.TransitGateway{Location: &location3}

		tgwCollection := &transitgatewayapisv1.TransitGatewayCollection{
			TransitGateways: []transitgatewayapisv1.TransitGateway{tgw1, tgw2, tgw3},
		}

		// Mock the ListTransitGateways call - returns (collection, error)
		mockTGService.On("ListTransitGateways", mock.Anything).Return(tgwCollection, nil)

		// Create service with mocks
		infoSvc := &CloudInfoService{
			transitGatewayService: mockTGService,
			vpcService:            mockVpcService,
		}

		// Call the method
		region, err := infoSvc.GetRegionWithLeastTransitGateways()

		// Assertions
		assert.NoError(t, err)
		// regavail-3-1 and regavail-4-30 have 0 TGWs, regavail-2-10 has 1 TGW, regavail-1-5 has 2 TGWs
		// Should return first region with 0 TGWs
		assert.Equal(t, "regavail-3-1", region)
		mockTGService.AssertExpectations(t)
	})

	t.Run("Error - Transit gateway service returns error", func(t *testing.T) {
		// Create mock transit gateway service
		mockTGService := new(transitGatewayServiceMock)

		// Mock the ListTransitGateways call to return error - returns (collection, error)
		mockTGService.On("ListTransitGateways", mock.Anything).Return(nil, assert.AnError)

		// Create service with mock
		infoSvc := &CloudInfoService{
			transitGatewayService: mockTGService,
		}

		// Call the method
		region, err := infoSvc.GetRegionWithLeastTransitGateways()

		// Assertions
		assert.Error(t, err)
		assert.Empty(t, region)
		assert.Contains(t, err.Error(), "failed to list transit gateways")
		mockTGService.AssertExpectations(t)
	})

	t.Run("Success - No transit gateways found", func(t *testing.T) {
		// Create mock transit gateway service
		mockTGService := new(transitGatewayServiceMock)

		// Create mock VPC service
		mockVpcService := new(vpcServiceMock)

		// Empty transit gateway collection
		tgwCollection := &transitgatewayapisv1.TransitGatewayCollection{
			TransitGateways: []transitgatewayapisv1.TransitGateway{},
		}

		// Mock the ListTransitGateways call - returns (collection, error)
		mockTGService.On("ListTransitGateways", mock.Anything).Return(tgwCollection, nil)

		// Create service with mocks
		infoSvc := &CloudInfoService{
			transitGatewayService: mockTGService,
			vpcService:            mockVpcService,
		}

		// Call the method
		region, err := infoSvc.GetRegionWithLeastTransitGateways()

		// Assertions
		assert.NoError(t, err)
		assert.NotEmpty(t, region) // Should return first available region
		mockTGService.AssertExpectations(t)
	})
}
