package testhelper

import (
	"errors"
	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/platform-services-go-sdk/catalogmanagementv1"
	projects "github.com/IBM/project-go-sdk/projectv1"
	"os"
	"sync"
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
	lock                              sync.Mutex
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

func (mock *cloudInfoServiceMock) HasRegionData() bool {
	return false
}

func (mock *cloudInfoServiceMock) RemoveRegionForTest(regionID string) {
	// nothing to really do here
}

func (mock *cloudInfoServiceMock) GetThreadLock() *sync.Mutex {
	return &mock.lock
}
func (mock *cloudInfoServiceMock) GetCatalogVersionByLocator(string) (*catalogmanagementv1.Version, error) {
	return nil, nil
}
func (mock *cloudInfoServiceMock) CreateDefaultProject(string, string, string) (*projects.Project, *core.DetailedResponse, error) {
	return nil, nil, nil
}
func (mock *cloudInfoServiceMock) GetProject(string) (*projects.Project, *core.DetailedResponse, error) {
	return nil, nil, nil
}
func (mock *cloudInfoServiceMock) GetProjectConfigs(string) ([]projects.ProjectConfigSummary, error) {
	return nil, nil
}
func (mock *cloudInfoServiceMock) GetConfig(string, string) (result *projects.ProjectConfig, response *core.DetailedResponse, err error) {
	return nil, nil, nil
}
func (mock *cloudInfoServiceMock) DeleteProject(string) (*projects.ProjectDeleteResponse, *core.DetailedResponse, error) {
	return nil, nil, nil
}
func (mock *cloudInfoServiceMock) CreateConfig(string, string, string, string) (result *projects.ProjectConfig, response *core.DetailedResponse, err error) {
	return nil, nil, nil
}
func (mock *cloudInfoServiceMock) CreateDaConfig(string, string, string, string, projects.ProjectConfigAuth, map[string]interface{}, map[string]interface{}) (result *projects.ProjectConfig, response *core.DetailedResponse, err error) {
	return nil, nil, nil
}
func (mock *cloudInfoServiceMock) CreateConfigFromCatalogJson(string, string, string) (result *projects.ProjectConfig, response *core.DetailedResponse, err error) {
	return nil, nil, nil
}
func (mock *cloudInfoServiceMock) UpdateConfig(string, string, projects.ProjectConfigDefinitionPatchIntf) (result *projects.ProjectConfig, response *core.DetailedResponse, err error) {
	return nil, nil, nil
}
func (mock *cloudInfoServiceMock) ApproveConfig(string, string) (result *projects.ProjectConfigVersion, response *core.DetailedResponse, err error) {
	return nil, nil, nil
}
func (mock *cloudInfoServiceMock) IsConfigApproved(string, string) (projectConfig *projects.ProjectConfigVersion, isApproved bool) {
	return nil, false
}
func (mock *cloudInfoServiceMock) ValidateProjectConfig(string, string) (result *projects.ProjectConfigVersion, response *core.DetailedResponse, err error) {
	return nil, nil, nil
}
func (mock *cloudInfoServiceMock) IsConfigValidated(string, string) (projectConfig *projects.ProjectConfigVersion, isValidated bool) {
	return nil, false
}
func (mock *cloudInfoServiceMock) DeployConfig(string, string) (result *projects.ProjectConfigVersion, response *core.DetailedResponse, err error) {
	return nil, nil, nil
}
func (mock *cloudInfoServiceMock) IsConfigDeployed(string, string) (projectConfig *projects.ProjectConfigVersion, isDeployed bool) {
	return nil, false
}
func (mock *cloudInfoServiceMock) UndeployConfig(string, string) (result *projects.ProjectConfigVersion, response *core.DetailedResponse, err error) {
	return nil, nil, nil
}
func (mock *cloudInfoServiceMock) IsUndeploying(string, string) (projectConfig *projects.ProjectConfigVersion, isUndeploying bool) {
	return nil, false

}
func (mock *cloudInfoServiceMock) CreateStackFromConfigFileWithInputs(string, string, string, map[string]interface{}) (result *projects.StackDefinition, response *core.DetailedResponse, err error) {
	return nil, nil, nil

}
func (mock *cloudInfoServiceMock) GetProjectConfigVersion(string, string, int64) (result *projects.ProjectConfigVersion, response *core.DetailedResponse, err error) {
	return nil, nil, nil
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
