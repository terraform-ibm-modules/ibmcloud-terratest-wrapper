package cloudinfo

import (
	"sync"

	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/platform-services-go-sdk/catalogmanagementv1"
	"github.com/IBM/platform-services-go-sdk/contextbasedrestrictionsv1"
	projects "github.com/IBM/project-go-sdk/projectv1"
	"github.com/IBM/schematics-go-sdk/schematicsv1"
	"github.com/stretchr/testify/mock"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/common"
)

// MockCloudInfoServiceForPermutation is a mock implementation of CloudInfoServiceI for permutation testing
type MockCloudInfoServiceForPermutation struct {
	mock.Mock
}

func (m *MockCloudInfoServiceForPermutation) CreateCatalog(catalogName string) (*catalogmanagementv1.Catalog, error) {
	args := m.Called(catalogName)

	var catalog *catalogmanagementv1.Catalog
	if args.Get(0) != nil {
		catalog = args.Get(0).(*catalogmanagementv1.Catalog)
	}

	return catalog, args.Error(1)
}

func (m *MockCloudInfoServiceForPermutation) DeleteCatalog(catalogID string) error {
	args := m.Called(catalogID)
	return args.Error(0)
}

func (m *MockCloudInfoServiceForPermutation) ImportOfferingWithValidation(catalogID, offeringName, offeringFlavor, version string, installKind InstallKind) (*catalogmanagementv1.Offering, error) {
	args := m.Called(catalogID, offeringName, offeringFlavor, version, installKind)

	var offering *catalogmanagementv1.Offering
	if args.Get(0) != nil {
		offering = args.Get(0).(*catalogmanagementv1.Offering)
	}

	return offering, args.Error(1)
}

func (m *MockCloudInfoServiceForPermutation) GetComponentReferences(versionLocator string) (*OfferingReferenceResponse, error) {
	args := m.Called(versionLocator)

	var response *OfferingReferenceResponse
	if args.Get(0) != nil {
		response = args.Get(0).(*OfferingReferenceResponse)
	}

	return response, args.Error(1)
}

// Implement other required methods with no-op implementations for testing
func (m *MockCloudInfoServiceForPermutation) GetLeastVpcTestRegion() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockCloudInfoServiceForPermutation) GetLeastVpcTestRegionWithoutActivityTracker() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockCloudInfoServiceForPermutation) GetLeastPowerConnectionZone() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockCloudInfoServiceForPermutation) LoadRegionPrefsFromFile(filepath string) error {
	args := m.Called(filepath)
	return args.Error(0)
}

func (m *MockCloudInfoServiceForPermutation) HasRegionData() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockCloudInfoServiceForPermutation) RemoveRegionForTest(region string) {
	m.Called(region)
}

func (m *MockCloudInfoServiceForPermutation) ReplaceCBRRule(updatedExistingRule *contextbasedrestrictionsv1.Rule, eTag *string) (*contextbasedrestrictionsv1.Rule, *core.DetailedResponse, error) {
	args := m.Called(updatedExistingRule, eTag)

	var rule *contextbasedrestrictionsv1.Rule
	if args.Get(0) != nil {
		rule = args.Get(0).(*contextbasedrestrictionsv1.Rule)
	}

	var response *core.DetailedResponse
	if args.Get(1) != nil {
		response = args.Get(1).(*core.DetailedResponse)
	}

	return rule, response, args.Error(2)
}

func (m *MockCloudInfoServiceForPermutation) GetThreadLock() *sync.Mutex {
	args := m.Called()
	return args.Get(0).(*sync.Mutex)
}

func (m *MockCloudInfoServiceForPermutation) GetClusterIngressStatus(clusterId string) (string, error) {
	args := m.Called(clusterId)
	return args.String(0), args.Error(1)
}

func (m *MockCloudInfoServiceForPermutation) GetCatalogVersionByLocator(locator string) (*catalogmanagementv1.Version, error) {
	args := m.Called(locator)

	var version *catalogmanagementv1.Version
	if args.Get(0) != nil {
		version = args.Get(0).(*catalogmanagementv1.Version)
	}

	return version, args.Error(1)
}

func (m *MockCloudInfoServiceForPermutation) ImportOffering(catalogID string, zipUrl string, offeringName string, flavorName string, version string, installKind InstallKind) (*catalogmanagementv1.Offering, error) {
	args := m.Called(catalogID, zipUrl, offeringName, flavorName, version, installKind)

	var offering *catalogmanagementv1.Offering
	if args.Get(0) != nil {
		offering = args.Get(0).(*catalogmanagementv1.Offering)
	}

	return offering, args.Error(1)
}

func (m *MockCloudInfoServiceForPermutation) PrepareOfferingImport() (branchUrl, repo, branch string, err error) {
	args := m.Called()
	return args.String(0), args.String(1), args.String(2), args.Error(3)
}

func (m *MockCloudInfoServiceForPermutation) GetOffering(catalogID string, offeringID string) (*catalogmanagementv1.Offering, *core.DetailedResponse, error) {
	args := m.Called(catalogID, offeringID)

	var offering *catalogmanagementv1.Offering
	if args.Get(0) != nil {
		offering = args.Get(0).(*catalogmanagementv1.Offering)
	}

	var response *core.DetailedResponse
	if args.Get(1) != nil {
		response = args.Get(1).(*core.DetailedResponse)
	}

	return offering, response, args.Error(2)
}

func (m *MockCloudInfoServiceForPermutation) GetOfferingInputs(offering *catalogmanagementv1.Offering, versionID string, offeringID string) []CatalogInput {
	args := m.Called(offering, versionID, offeringID)
	return args.Get(0).([]CatalogInput)
}

func (m *MockCloudInfoServiceForPermutation) GetOfferingVersionLocatorByConstraint(catalogID, offeringID, versionConstraint, flavor string) (string, string, error) {
	args := m.Called(catalogID, offeringID, versionConstraint, flavor)
	return args.String(0), args.String(1), args.Error(2)
}

func (m *MockCloudInfoServiceForPermutation) DeployAddonToProject(addonConfig *AddonConfig, projectConfig *ProjectsConfig) (*DeployedAddonsDetails, error) {
	args := m.Called(addonConfig, projectConfig)

	var details *DeployedAddonsDetails
	if args.Get(0) != nil {
		details = args.Get(0).(*DeployedAddonsDetails)
	}

	return details, args.Error(1)
}

// Add stub implementations for remaining methods - these won't be used in permutation tests
func (m *MockCloudInfoServiceForPermutation) CreateProjectFromConfig(config *ProjectsConfig) (*projects.Project, *core.DetailedResponse, error) {
	args := m.Called(config)
	return args.Get(0).(*projects.Project), args.Get(1).(*core.DetailedResponse), args.Error(2)
}

func (m *MockCloudInfoServiceForPermutation) GetProject(projectID string) (*projects.Project, *core.DetailedResponse, error) {
	args := m.Called(projectID)
	return args.Get(0).(*projects.Project), args.Get(1).(*core.DetailedResponse), args.Error(2)
}

func (m *MockCloudInfoServiceForPermutation) GetProjectConfigs(projectID string) ([]projects.ProjectConfigSummary, error) {
	args := m.Called(projectID)
	return args.Get(0).([]projects.ProjectConfigSummary), args.Error(1)
}

func (m *MockCloudInfoServiceForPermutation) GetConfig(configDetails *ConfigDetails) (*projects.ProjectConfig, *core.DetailedResponse, error) {
	args := m.Called(configDetails)
	return args.Get(0).(*projects.ProjectConfig), args.Get(1).(*core.DetailedResponse), args.Error(2)
}

func (m *MockCloudInfoServiceForPermutation) GetConfigName(projectID, configID string) (string, error) {
	args := m.Called(projectID, configID)
	return args.String(0), args.Error(1)
}

func (m *MockCloudInfoServiceForPermutation) DeleteProject(projectID string) (*projects.ProjectDeleteResponse, *core.DetailedResponse, error) {
	args := m.Called(projectID)
	return args.Get(0).(*projects.ProjectDeleteResponse), args.Get(1).(*core.DetailedResponse), args.Error(2)
}

func (m *MockCloudInfoServiceForPermutation) CreateConfig(configDetails *ConfigDetails) (*projects.ProjectConfig, *core.DetailedResponse, error) {
	args := m.Called(configDetails)
	return args.Get(0).(*projects.ProjectConfig), args.Get(1).(*core.DetailedResponse), args.Error(2)
}

func (m *MockCloudInfoServiceForPermutation) DeployConfig(configDetails *ConfigDetails) (*projects.ProjectConfigVersion, *core.DetailedResponse, error) {
	args := m.Called(configDetails)
	return args.Get(0).(*projects.ProjectConfigVersion), args.Get(1).(*core.DetailedResponse), args.Error(2)
}

func (m *MockCloudInfoServiceForPermutation) CreateDaConfig(configDetails *ConfigDetails) (*projects.ProjectConfig, *core.DetailedResponse, error) {
	args := m.Called(configDetails)
	return args.Get(0).(*projects.ProjectConfig), args.Get(1).(*core.DetailedResponse), args.Error(2)
}

func (m *MockCloudInfoServiceForPermutation) CreateConfigFromCatalogJson(configDetails *ConfigDetails, catalogJson string) (*projects.ProjectConfig, *core.DetailedResponse, error) {
	args := m.Called(configDetails, catalogJson)
	return args.Get(0).(*projects.ProjectConfig), args.Get(1).(*core.DetailedResponse), args.Error(2)
}

func (m *MockCloudInfoServiceForPermutation) UpdateConfig(configDetails *ConfigDetails, configuration projects.ProjectConfigDefinitionPatchIntf) (*projects.ProjectConfig, *core.DetailedResponse, error) {
	args := m.Called(configDetails, configuration)
	return args.Get(0).(*projects.ProjectConfig), args.Get(1).(*core.DetailedResponse), args.Error(2)
}

func (m *MockCloudInfoServiceForPermutation) ValidateProjectConfig(configDetails *ConfigDetails) (*projects.ProjectConfigVersion, *core.DetailedResponse, error) {
	args := m.Called(configDetails)
	return args.Get(0).(*projects.ProjectConfigVersion), args.Get(1).(*core.DetailedResponse), args.Error(2)
}

func (m *MockCloudInfoServiceForPermutation) IsConfigDeployed(configDetails *ConfigDetails) (*projects.ProjectConfigVersion, bool) {
	args := m.Called(configDetails)
	return args.Get(0).(*projects.ProjectConfigVersion), args.Bool(1)
}

func (m *MockCloudInfoServiceForPermutation) UndeployConfig(details *ConfigDetails) (*projects.ProjectConfigVersion, *core.DetailedResponse, error) {
	args := m.Called(details)
	return args.Get(0).(*projects.ProjectConfigVersion), args.Get(1).(*core.DetailedResponse), args.Error(2)
}

func (m *MockCloudInfoServiceForPermutation) IsUndeploying(details *ConfigDetails) (*projects.ProjectConfigVersion, bool) {
	args := m.Called(details)
	return args.Get(0).(*projects.ProjectConfigVersion), args.Bool(1)
}

func (m *MockCloudInfoServiceForPermutation) CreateStackFromConfigFile(stackConfig *ConfigDetails, stackConfigPath string, catalogJsonPath string) (*projects.StackDefinition, *core.DetailedResponse, error) {
	args := m.Called(stackConfig, stackConfigPath, catalogJsonPath)
	return args.Get(0).(*projects.StackDefinition), args.Get(1).(*core.DetailedResponse), args.Error(2)
}

func (m *MockCloudInfoServiceForPermutation) GetProjectConfigVersion(configDetails *ConfigDetails, version int64) (*projects.ProjectConfigVersion, *core.DetailedResponse, error) {
	args := m.Called(configDetails, version)
	return args.Get(0).(*projects.ProjectConfigVersion), args.Get(1).(*core.DetailedResponse), args.Error(2)
}

func (m *MockCloudInfoServiceForPermutation) GetStackMembers(stackConfig *ConfigDetails) ([]*projects.ProjectConfig, error) {
	args := m.Called(stackConfig)
	return args.Get(0).([]*projects.ProjectConfig), args.Error(1)
}

func (m *MockCloudInfoServiceForPermutation) SyncConfig(projectID string, configID string) (*core.DetailedResponse, error) {
	args := m.Called(projectID, configID)
	return args.Get(0).(*core.DetailedResponse), args.Error(1)
}

func (m *MockCloudInfoServiceForPermutation) LookupMemberNameByID(stackDetails *projects.ProjectConfig, memberID string) (string, error) {
	args := m.Called(stackDetails, memberID)
	return args.String(0), args.Error(1)
}

func (m *MockCloudInfoServiceForPermutation) GetSchematicsJobLogs(jobID string, location string) (*schematicsv1.JobLog, *core.DetailedResponse, error) {
	args := m.Called(jobID, location)
	return args.Get(0).(*schematicsv1.JobLog), args.Get(1).(*core.DetailedResponse), args.Error(2)
}

func (m *MockCloudInfoServiceForPermutation) GetSchematicsJobLogsText(jobID string, location string) (string, error) {
	args := m.Called(jobID, location)
	return args.String(0), args.Error(1)
}

func (m *MockCloudInfoServiceForPermutation) ArePipelineActionsRunning(stackConfig *ConfigDetails) (bool, error) {
	args := m.Called(stackConfig)
	return args.Bool(0), args.Error(1)
}

func (m *MockCloudInfoServiceForPermutation) DeleteInstanceFromReclamationByCRN(CRN string) error {
	args := m.Called(CRN)
	return args.Error(0)
}

func (m *MockCloudInfoServiceForPermutation) DeleteInstanceFromReclamationId(reclamationID string) error {
	args := m.Called(reclamationID)
	return args.Error(0)
}

func (m *MockCloudInfoServiceForPermutation) GetApiKey() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockCloudInfoServiceForPermutation) GetLogger() common.Logger {
	args := m.Called()
	return args.Get(0).(common.Logger)
}

func (m *MockCloudInfoServiceForPermutation) SetLogger(logger common.Logger) {
	m.Called(logger)
}

func (m *MockCloudInfoServiceForPermutation) GetReclamationIdFromCRN(CRN string) (string, error) {
	args := m.Called(CRN)
	return args.String(0), args.Error(1)
}

func (m *MockCloudInfoServiceForPermutation) GetSchematicsJobLogsForMember(member *projects.ProjectConfig, memberName string, projectRegion string) (string, string) {
	args := m.Called(member, memberName, projectRegion)
	return args.String(0), args.String(1)
}

func (m *MockCloudInfoServiceForPermutation) GetSchematicsJobFileData(jobID string, fileType string, location string) (*schematicsv1.JobFileData, error) {
	args := m.Called(jobID, fileType, location)

	var jobFileData *schematicsv1.JobFileData
	if args.Get(0) != nil {
		jobFileData = args.Get(0).(*schematicsv1.JobFileData)
	}

	return jobFileData, args.Error(1)
}

func (m *MockCloudInfoServiceForPermutation) GetSchematicsJobPlanJson(jobID string, location string) (string, error) {
	args := m.Called(jobID, location)
	return args.String(0), args.Error(1)
}

func (m *MockCloudInfoServiceForPermutation) GetSchematicsServiceByLocation(location string) (schematicsService, error) {
	args := m.Called(location)
	return args.Get(0).(schematicsService), args.Error(1)
}

func (m *MockCloudInfoServiceForPermutation) ResolveReferences(region string, references []Reference) (*ResolveResponse, error) {
	args := m.Called(region, references)

	var response *ResolveResponse
	if args.Get(0) != nil {
		response = args.Get(0).(*ResolveResponse)
	}

	return response, args.Error(1)
}

func (m *MockCloudInfoServiceForPermutation) ResolveReferencesFromStrings(region string, refStrings []string, projectNameOrID string) (*ResolveResponse, error) {
	args := m.Called(region, refStrings, projectNameOrID)

	var response *ResolveResponse
	if args.Get(0) != nil {
		response = args.Get(0).(*ResolveResponse)
	}

	return response, args.Error(1)
}

func (m *MockCloudInfoServiceForPermutation) ResolveReferencesFromStringsWithContext(region string, refStrings []string, projectNameOrID string, batchMode bool) (*ResolveResponse, error) {
	args := m.Called(region, refStrings, projectNameOrID, batchMode)

	var response *ResolveResponse
	if args.Get(0) != nil {
		response = args.Get(0).(*ResolveResponse)
	}

	return response, args.Error(1)
}

func (m *MockCloudInfoServiceForPermutation) ResolveReferencesWithContext(region string, references []Reference, batchMode bool) (*ResolveResponse, error) {
	args := m.Called(region, references, batchMode)

	var response *ResolveResponse
	if args.Get(0) != nil {
		response = args.Get(0).(*ResolveResponse)
	}

	return response, args.Error(1)
}
