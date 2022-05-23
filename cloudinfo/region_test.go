package cloudinfo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLeastVpcAllAvailRegions(t *testing.T) {
	vpcService := new(vpcServiceMock)

	// first test, low priority wins
	infoSvc := CloudInfoService{
		vpcService: vpcService,
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
