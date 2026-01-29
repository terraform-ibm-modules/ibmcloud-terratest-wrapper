package testhelper

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/IBM/go-sdk-core/v5/core"
	projects "github.com/IBM/project-go-sdk/projectv1"
	schematics "github.com/IBM/schematics-go-sdk/schematicsv1"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/cloudinfo"

	"github.com/IBM/platform-services-go-sdk/catalogmanagementv1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

/**** START MOCK CloudInfoService ****/
type cloudInfoServiceMock struct {
	mock.Mock
	cloudinfo.CloudInfoServiceI
	prefsFileName                     string
	loadFileCalled                    bool
	getLeastVpcTestRegionCalled       bool
	getLeastVpcNoATTestRegionCalled   bool
	getLeastPowerConnectionZoneCalled bool
	lock                              sync.Mutex
}

func (mock *cloudInfoServiceMock) CreateStackDefinitionWrapper(stackDefOptions *projects.CreateStackDefinitionOptions, members []projects.StackMember) (result *projects.StackDefinition, response *core.DetailedResponse, err error) {
	return nil, nil, nil
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
func (mock *cloudInfoServiceMock) CreateProjectFromConfig(*cloudinfo.ProjectsConfig) (*projects.Project, *core.DetailedResponse, error) {
	return nil, nil, nil
}

func (mock *cloudInfoServiceMock) GetProject(string) (*projects.Project, *core.DetailedResponse, error) {
	return nil, nil, nil
}

func (mock *cloudInfoServiceMock) GetProjectConfigs(string) ([]projects.ProjectConfigSummary, error) {
	return nil, nil
}

func (mock *cloudInfoServiceMock) GetConfig(*cloudinfo.ConfigDetails) (result *projects.ProjectConfig, response *core.DetailedResponse, err error) {
	return nil, nil, nil
}

func (mock *cloudInfoServiceMock) DeleteProject(string) (*projects.ProjectDeleteResponse, *core.DetailedResponse, error) {
	return nil, nil, nil
}

func (mock *cloudInfoServiceMock) CreateConfig(*cloudinfo.ConfigDetails) (result *projects.ProjectConfig, response *core.DetailedResponse, err error) {
	return nil, nil, nil
}

func (mock *cloudInfoServiceMock) DeployConfig(*cloudinfo.ConfigDetails) (result *projects.ProjectConfigVersion, response *core.DetailedResponse, err error) {
	return nil, nil, nil
}

func (mock *cloudInfoServiceMock) CreateDaConfig(*cloudinfo.ConfigDetails) (result *projects.ProjectConfig, response *core.DetailedResponse, err error) {
	return nil, nil, nil
}

func (mock *cloudInfoServiceMock) CreateConfigFromCatalogJson(*cloudinfo.ConfigDetails, string) (result *projects.ProjectConfig, response *core.DetailedResponse, err error) {
	return nil, nil, nil
}

func (mock *cloudInfoServiceMock) UpdateConfig(*cloudinfo.ConfigDetails, projects.ProjectConfigDefinitionPatchIntf) (result *projects.ProjectConfig, response *core.DetailedResponse, err error) {
	return nil, nil, nil
}

func (mock *cloudInfoServiceMock) ValidateProjectConfig(*cloudinfo.ConfigDetails) (result *projects.ProjectConfigVersion, response *core.DetailedResponse, err error) {
	return nil, nil, nil
}

func (mock *cloudInfoServiceMock) IsConfigDeployed(*cloudinfo.ConfigDetails) (projectConfig *projects.ProjectConfigVersion, isDeployed bool) {
	return nil, false
}

func (mock *cloudInfoServiceMock) UndeployConfig(*cloudinfo.ConfigDetails) (result *projects.ProjectConfigVersion, response *core.DetailedResponse, err error) {
	return nil, nil, nil
}

func (mock *cloudInfoServiceMock) IsUndeploying(*cloudinfo.ConfigDetails) (projectConfig *projects.ProjectConfigVersion, isUndeploying bool) {
	return nil, false
}

func (mock *cloudInfoServiceMock) CreateStackFromConfigFile(*cloudinfo.ConfigDetails, string, string) (result *projects.StackDefinition, response *core.DetailedResponse, err error) {
	return nil, nil, nil
}

func (mock *cloudInfoServiceMock) GetProjectConfigVersion(*cloudinfo.ConfigDetails, int64) (result *projects.ProjectConfigVersion, response *core.DetailedResponse, err error) {
	return nil, nil, nil
}

func (mock *cloudInfoServiceMock) GetStackMembers(stackConfig *cloudinfo.ConfigDetails) ([]*projects.ProjectConfig, error) {
	return nil, nil
}

func (mock *cloudInfoServiceMock) SyncConfig(projectID string, configID string) (response *core.DetailedResponse, err error) {
	return nil, nil
}

func (mock *cloudInfoServiceMock) LookupMemberNameByID(stackDetails *projects.ProjectConfig, memberID string) (string, error) {
	return "", nil
}

func (mock *cloudInfoServiceMock) GetClusterIngressStatus(string) (string, error) {
	return "", nil
}

func (mock *cloudInfoServiceMock) GetSchematicsJobLogs(string, string) (*schematics.JobLog, *core.DetailedResponse, error) {
	return nil, nil, nil

}
func (mock *cloudInfoServiceMock) GetSchematicsJobLogsText(string, string) (string, error) {
	return "", nil

}

func (mock *cloudInfoServiceMock) ArePipelineActionsRunning(stackConfig *cloudinfo.ConfigDetails) (bool, error) {
	return false, nil
}

func (mock *cloudInfoServiceMock) GetSchematicsJobLogsForMember(member *projects.ProjectConfig, memberName string, projectRegion string, projectID string, configID string) (string, string) {
	return "", ""
}

// special mock for CreateStackDefinition
// we do not have enough information when mocking projectv1.CreateStackDefinition to return a valid response
// to get around this we create a wrapper that can take in the missing list of members that can be used in the mock
// to return a valid response

func (mock *cloudInfoServiceMock) CreateStackDefinition(stackDefOptions *projects.CreateStackDefinitionOptions, members []projects.StackMember) (result *projects.StackDefinition, response *core.DetailedResponse, err error) {
	args := mock.Called(stackDefOptions, members)
	return args.Get(0).(*projects.StackDefinition), args.Get(1).(*core.DetailedResponse), args.Error(2)
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

// Common directories to exclude and file types to include in tests
var (
	dirsToExclude = []string{
		".terraform", ".docs", ".github", ".git", ".idea",
		"common-dev-assets", "examples", "tests", "reference-architectures",
	}
	fileTypesToInclude = []string{".tf", ".yaml", ".py", ".tpl"}
)

// mkdirAll creates multiple directories under a base directory
func mkdirAll(t *testing.T, base string, relPaths ...string) {
	t.Helper()
	for _, p := range relPaths {
		if err := os.MkdirAll(filepath.Join(base, p), 0o755); err != nil {
			t.Fatalf("failed to create dir %q: %v", p, err)
		}
	}
}

// mustChdir changes the current working directory to dir
// It returns a function to restore the previous working directory
// If any operation fails, it calls t.Fatalf
func mustChdir(t *testing.T, dir string) func() {
	t.Helper()
	prev, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir %q: %v", dir, err)
	}
	return func() { _ = os.Chdir(prev) }
}

// assertAllPresent checks that all wanted strings are present in got slice
func assertAllPresent(t *testing.T, got []string, wants ...string) {
	t.Helper()
	set := make(map[string]struct{}, len(got))
	for _, g := range got {
		set[g] = struct{}{}
	}
	for _, w := range wants {
		if _, ok := set[w]; !ok {
			t.Fatalf("missing %q in %v", w, got)
		}
	}
}

// assertNoneContains checks that no entry in got slice contains the given substring
func assertNoneContains(t *testing.T, got []string, substr string) {
	t.Helper()
	for _, g := range got {
		if strings.Contains(g, substr) {
			t.Fatalf("expected no entry containing %q, but found %q in %v", substr, g, got)
		}
	}
}

// assertNoPrefix checks that no entry in got slice has the given prefix
func assertNoPrefix(t *testing.T, got []string, prefix string) {
	t.Helper()
	for _, g := range got {
		if strings.HasPrefix(g, prefix) {
			t.Fatalf("expected no entry with prefix %q, but found %q in %v", prefix, g, got)
		}
	}
}

func TestGetTarIncludePatterns(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(t *testing.T) (walkRoot string, cleanup func())
		excludes   []string
		filetypes  []string
		assertions func(t *testing.T, got []string, err error, walkRoot string)
	}{
		{
			name: "recursively collects include patterns for multiple file types while excluding specified directories",
			setup: func(t *testing.T) (string, func()) {
				dir := t.TempDir()
				mkdirAll(t, dir, "a/b")                // included
				mkdirAll(t, dir, ".terraform/modules") // excluded
				return dir, func() {}
			},
			excludes:  dirsToExclude,
			filetypes: fileTypesToInclude,
			assertions: func(t *testing.T, got []string, err error, walkRoot string) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				// included dirs visited: walkRoot, walkRoot/a, walkRoot/a/b => 3 dirs
				// patterns per dir = 4 filetypes => 12 patterns total
				if len(got) != 12 {
					t.Fatalf("expected 12 patterns, got %d: %v", len(got), got)
				}

				// ensures that all wanted strings are present in got slice
				a := filepath.Join(walkRoot, "a")
				assertAllPresent(t, got,
					a+"/*.tf",
					a+"/*.yaml",
					a+"/*.py",
					a+"/*.tpl",
				)

				// ensures that no entry in got slice contains the given substring
				assertNoneContains(t, got, ".terraform")
			},
		},
		{
			name: "returns no patterns when no file types are provided",
			setup: func(t *testing.T) (string, func()) {
				dir := t.TempDir()
				mkdirAll(t, dir, "x")
				return dir, func() {}
			},
			excludes:  dirsToExclude,
			filetypes: nil,
			assertions: func(t *testing.T, got []string, err error, walkRoot string) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				// expect no patterns when no file types are provided
				if len(got) != 0 {
					t.Fatalf("expected 0 patterns, got %d: %v", len(got), got)
				}
			},
		},
		{
			name: "returns an error when the root directory does not exist",
			setup: func(t *testing.T) (string, func()) {
				nonExistent := filepath.Join(t.TempDir(), "does-not-exist")
				return nonExistent, func() {}
			},
			excludes:  dirsToExclude,
			filetypes: []string{".tf"},
			assertions: func(t *testing.T, got []string, err error, walkRoot string) {
				if err == nil {
					t.Fatalf("expected error, got nil (patterns=%v)", got)
				}
				// on error, expect no patterns
				if len(got) != 0 {
					t.Fatalf("expected 0 patterns on error, got %d: %v", len(got), got)
				}
			},
		},
		{
			// Special case: walking the parent directory
			name: "adds a literal include pattern when walking the parent directory",
			setup: func(t *testing.T) (string, func()) {
				root := t.TempDir()
				child := filepath.Join(root, "child")
				mkdirAll(t, root, "child")
				return "..", mustChdir(t, child)
			},
			excludes:  dirsToExclude,
			filetypes: []string{".tf"},
			assertions: func(t *testing.T, got []string, err error, walkRoot string) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				// expect to find the literal pattern "*.tf"
				found := false
				for _, p := range got {
					if p == "*.tf" {
						found = true
						break
					}
				}
				// fail if the literal pattern "*.tf" is not found
				if !found {
					t.Fatalf("expected to find '*.tf' when walking '..', got: %v", got)
				}
			},
		},
		{
			name: "strips parent directory prefixes from generated include patterns",
			setup: func(t *testing.T) (string, func()) {
				root := t.TempDir()
				child := filepath.Join(root, "child")
				sibling := filepath.Join(root, "sibling")
				mkdirAll(t, root, "child", "sibling/x")
				_ = sibling
				return "../sibling", mustChdir(t, child)
			},
			excludes:  dirsToExclude,
			filetypes: []string{".tf"},
			assertions: func(t *testing.T, got []string, err error, walkRoot string) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				// ensure no entry in got slice has the given prefix "../"
				assertNoPrefix(t, got, "../")

				// ensure all wanted strings are present in got slice
				assertAllPresent(t, got,
					filepath.Join("sibling")+"/*.tf",
					filepath.Join("sibling", "x")+"/*.tf",
				)
			},
		},
	}

	for i := range tests {
		tc := tests[i]
		t.Run(tc.name, func(t *testing.T) {
			walkRoot, cleanup := tc.setup(t)
			defer cleanup()

			got, err := GetTarIncludePatterns(walkRoot, tc.excludes, tc.filetypes)
			tc.assertions(t, got, err, walkRoot)
		})
	}
}
