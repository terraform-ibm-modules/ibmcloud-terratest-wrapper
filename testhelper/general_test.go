package testhelper

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

/**** START MOCK CloudInfoService ****/
type cloudInfoServiceMock struct {
	mock.Mock
	prefsFileName                     string
	loadFileCalled                    bool
	getLeastVpcTestRegionCalled       bool
	getLeastVpcNoATTestRegionCalled   bool
	getLeastPowerConnectionZoneCalled bool
}

func (mock *cloudInfoServiceMock) LoadRegionPrefsFromFile(prefsFile string) error {
	mock.prefsFileName = prefsFile
	mock.loadFileCalled = true

	if prefsFile == "badfile" {
		return errors.New("Bad File")
	} else {
		return nil
	}
}

func (mock *cloudInfoServiceMock) GetLeastVpcTestRegion() (string, error) {
	mock.getLeastVpcTestRegionCalled = true

	switch mock.prefsFileName {
	case "goodfile":
		return "best-region", nil
	case "badfile":
		return "ok-region", nil
	case "":
		return "all-region", nil
	case "empty-region":
		return "", nil
	case "errorfile":
		return "", errors.New("mock Error Msg")
	}
	return "", errors.New("mock no matching file name")
}

func (mock *cloudInfoServiceMock) GetLeastVpcTestRegionWithoutActivityTracker() (string, error) {
	mock.getLeastVpcNoATTestRegionCalled = true

	switch mock.prefsFileName {
	case "goodfile":
		return "best-region-no-at", nil
	case "badfile":
		return "ok-region", nil
	case "":
		return "all-region", nil
	case "empty-region":
		return "", nil
	case "errorfile":
		return "", errors.New("mock Error Msg")
	}
	return "", errors.New("mock no matching file name")
}

func (mock *cloudInfoServiceMock) GetLeastPowerConnectionZone() (string, error) {
	mock.getLeastPowerConnectionZoneCalled = true

	switch mock.prefsFileName {
	case "goodfile":
		return "best-region", nil
	case "badfile":
		return "ok-region", nil
	case "":
		return "all-region", nil
	case "empty-region":
		return "", nil
	case "errorfile":
		return "", errors.New("mock Error Msg")
	}
	return "", errors.New("mock no matching file name")
}

/**** END MOCK CloudInfoService ****/

func TestLeastVpcRegionFound(t *testing.T) {
	infoSvc := cloudInfoServiceMock{loadFileCalled: false, getLeastVpcTestRegionCalled: false}
	options := TesthelperTerraformOptions{CloudInfoService: &infoSvc}

	bestregion, err := GetBestVpcRegionO("FAKEKEY", "goodfile", "default-region", options)

	assert.True(t, infoSvc.getLeastVpcTestRegionCalled, "GetLeastVpcTestRegion() should have been called")
	assert.Nil(t, err, "Must not return error")
	assert.Equal(t, "best-region", bestregion, "Should return best region")
}

func TestLeastVpcRegionNoActivityTrackerFound(t *testing.T) {
	infoSvc := cloudInfoServiceMock{loadFileCalled: false, getLeastVpcTestRegionCalled: false}
	options := TesthelperTerraformOptions{
		CloudInfoService:              &infoSvc,
		ExcludeActivityTrackerRegions: true,
	}

	bestregion, err := GetBestVpcRegionO("FAKEKEY", "goodfile", "default-region", options)

	assert.True(t, infoSvc.getLeastVpcNoATTestRegionCalled, "GetLeastVpcTestRegionWithoutActivityTracker() should have been called")
	assert.Nil(t, err, "Must not return error")
	assert.Equal(t, "best-region-no-at", bestregion, "Should return best region")
}

func TestLeastVpcRegionDefault(t *testing.T) {
	infoSvc := cloudInfoServiceMock{loadFileCalled: false, getLeastVpcTestRegionCalled: false}
	options := TesthelperTerraformOptions{CloudInfoService: &infoSvc}

	// error returned, should default
	bestregion1, err1 := GetBestVpcRegionO("FAKEKEY", "errorfile", "default-region", options)
	assert.NotNil(t, err1, "Error condition should have returned error")
	assert.Equal(t, "default-region", bestregion1, "Error condition should return default region")

	// empty region returned, should default
	bestregion2, err2 := GetBestVpcRegionO("FAKEKEY", "empty-region", "default-region", options)
	assert.Nil(t, err2, "Empty condition should NOT have returned error")
	assert.Equal(t, "default-region", bestregion2, "Empty condition should return default region")
}

func TestLeastVpcRegionWithFile(t *testing.T) {
	infoSvc := cloudInfoServiceMock{loadFileCalled: false, getLeastVpcTestRegionCalled: false}
	options := TesthelperTerraformOptions{CloudInfoService: &infoSvc}

	_, err := GetBestVpcRegionO("FAKEKEY", "goodfile", "default-region", options)
	assert.Nil(t, err, "Error should not be returned")
	assert.Equal(t, true, infoSvc.loadFileCalled, "Load file function should be called")
}

func TestLeastVpcRegionNoFile(t *testing.T) {
	infoSvc := cloudInfoServiceMock{loadFileCalled: false, getLeastVpcTestRegionCalled: false}
	options := TesthelperTerraformOptions{CloudInfoService: &infoSvc}

	bestregion, err := GetBestVpcRegionO("FAKEKEY", "", "default-region", options)
	assert.Nil(t, err, "Error should not be returned")
	assert.Equal(t, "all-region", bestregion, "All (broadest) region should be returned if no prefs file")
	assert.Equal(t, false, infoSvc.loadFileCalled)
}

func TestLeastVpcRegionForced(t *testing.T) {
	// set a forced region
	os.Setenv(ForceTestRegionEnvName, "forced-region")
	defer os.Unsetenv(ForceTestRegionEnvName)
	infoSvc := cloudInfoServiceMock{loadFileCalled: false, getLeastVpcTestRegionCalled: false}
	options := TesthelperTerraformOptions{CloudInfoService: &infoSvc}

	bestregion, err := GetBestVpcRegionO("FAKEKEY", "goodfile", "default-region", options)

	assert.False(t, infoSvc.getLeastVpcTestRegionCalled, "GetLeastVpcTestRegion() should NOT have been called")
	assert.Nil(t, err, "Must not return error")
	assert.Equal(t, "forced-region", bestregion, "Should return FORCED region")
}

func TestLeastPowerConnectionZoneFound(t *testing.T) {
	infoSvc := cloudInfoServiceMock{loadFileCalled: false, getLeastPowerConnectionZoneCalled: false}
	options := TesthelperTerraformOptions{CloudInfoService: &infoSvc}

	bestregion, err := GetBestPowerSystemsRegionO("FAKEKEY", "goodfile", "default-region", options)

	assert.True(t, infoSvc.getLeastPowerConnectionZoneCalled, "GetLeastPowerConnectionZone() should have been called")
	assert.Nil(t, err, "Must not return error")
	assert.Equal(t, "best-region", bestregion, "Should return best region")
}

func TestLeastPowerConnectionZoneDefault(t *testing.T) {
	infoSvc := cloudInfoServiceMock{loadFileCalled: false, getLeastPowerConnectionZoneCalled: false}
	options := TesthelperTerraformOptions{CloudInfoService: &infoSvc}

	// error returned, should default
	bestregion1, err1 := GetBestPowerSystemsRegionO("FAKEKEY", "errorfile", "default-region", options)
	assert.NotNil(t, err1, "Error condition should have returned error")
	assert.Equal(t, "default-region", bestregion1, "Error condition should return default region")

	// empty region returned, should default
	bestregion2, err2 := GetBestPowerSystemsRegionO("FAKEKEY", "empty-region", "default-region", options)
	assert.Nil(t, err2, "Empty condition should NOT have returned error")
	assert.Equal(t, "default-region", bestregion2, "Empty condition should return default region")
}

func TestLeastPowerConnectionZoneWithFile(t *testing.T) {
	infoSvc := cloudInfoServiceMock{loadFileCalled: false, getLeastPowerConnectionZoneCalled: false}
	options := TesthelperTerraformOptions{CloudInfoService: &infoSvc}

	_, err := GetBestPowerSystemsRegionO("FAKEKEY", "goodfile", "default-region", options)
	assert.Nil(t, err, "Error should not be returned")
	assert.Equal(t, true, infoSvc.loadFileCalled, "Load file function should be called")
}

func TestLeastPowerConnectionZoneNoFile(t *testing.T) {
	infoSvc := cloudInfoServiceMock{loadFileCalled: false, getLeastPowerConnectionZoneCalled: false}
	options := TesthelperTerraformOptions{CloudInfoService: &infoSvc}

	bestregion, err := GetBestPowerSystemsRegionO("FAKEKEY", "", "default-region", options)
	assert.Nil(t, err, "Error should not be returned")
	assert.Equal(t, "all-region", bestregion, "All (broadest) region should be returned if no prefs file")
	assert.Equal(t, false, infoSvc.loadFileCalled)
}

func TestLeastPowerConnectionZoneForced(t *testing.T) {
	// set a forced region
	os.Setenv(ForceTestRegionEnvName, "forced-region")
	defer os.Unsetenv(ForceTestRegionEnvName)

	infoSvc := cloudInfoServiceMock{loadFileCalled: false, getLeastPowerConnectionZoneCalled: false}
	options := TesthelperTerraformOptions{CloudInfoService: &infoSvc}

	bestregion, err := GetBestPowerSystemsRegionO("FAKEKEY", "goodfile", "default-region", options)

	assert.False(t, infoSvc.getLeastPowerConnectionZoneCalled, "GetLeastPowerConnectionZone() should NOT have been called")
	assert.Nil(t, err, "Must not return error")
	assert.Equal(t, "forced-region", bestregion, "Should return FORCED region")
}

func TestGetRequiredEnvVarsSuccess(t *testing.T) {
	t.Setenv("A_REQUIRED_VARIABLE", "The Value")
	t.Setenv("ANOTHER_VARIABLE", "Another Value")

	expected := make(map[string]string)
	expected["A_REQUIRED_VARIABLE"] = "The Value"
	expected["ANOTHER_VARIABLE"] = "Another Value"

	assert.Equal(t, expected, GetRequiredEnvVars(t, []string{"A_REQUIRED_VARIABLE", "ANOTHER_VARIABLE"}))
}

func TestGetRequiredEnvVarsEmptyInput(t *testing.T) {

	expected := make(map[string]string)
	assert.Equal(t, expected, GetRequiredEnvVars(t, []string{}))
}
