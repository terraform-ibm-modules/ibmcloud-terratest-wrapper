// Package cloudinfo contains functions and methods for searching and detailing various resources located in the IBM Cloud
package cloudinfo

import (
	"errors"
	schematics "github.com/IBM/schematics-go-sdk/schematicsv1"
	"log"
	"os"
	"sync"

	"github.com/IBM/platform-services-go-sdk/catalogmanagementv1"
	projects "github.com/IBM/project-go-sdk/projectv1"

	"github.com/IBM-Cloud/bluemix-go"
	"github.com/IBM-Cloud/bluemix-go/api/container/containerv2"
	"github.com/IBM-Cloud/bluemix-go/session"
	ibmpimodels "github.com/IBM-Cloud/power-go-client/power/models"
	"github.com/IBM/cloud-databases-go-sdk/clouddatabasesv5"
	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/platform-services-go-sdk/contextbasedrestrictionsv1"
	"github.com/IBM/platform-services-go-sdk/iamidentityv1"
	"github.com/IBM/platform-services-go-sdk/iampolicymanagementv1"
	"github.com/IBM/platform-services-go-sdk/resourcecontrollerv2"
	"github.com/IBM/platform-services-go-sdk/resourcemanagerv2"

	"github.com/IBM/vpc-go-sdk/vpcv1"
)

// CloudInfoService is a structure that is used as the receiver to many methods in this package.
// It contains references to other important services and data structures needed to perform these methods.
type CloudInfoService struct {
	authenticator             *core.IamAuthenticator // shared authenticator
	apiKeyDetail              *iamidentityv1.APIKey  // IBMCloud account for user
	vpcService                vpcService
	iamIdentityService        iamIdentityService
	iamPolicyService          iamPolicyService
	resourceControllerService resourceControllerService
	resourceManagerService    resourceManagerService
	cbrService                cbrService
	containerClient           containerClient
	catalogService            catalogService
	// stackDefinitionCreator is used to create stack definitions and only added to support testing/mocking
	stackDefinitionCreator StackDefinitionCreator
	regionsData            []RegionData
	lock                   sync.Mutex
	icdService             icdService
	projectsService        projectsService
	schematicsService      schematicsService
	ApiKey                 string
}

// interface for the cloudinfo service (can be mocked in tests)
type CloudInfoServiceI interface {
	GetLeastVpcTestRegion() (string, error)
	GetLeastVpcTestRegionWithoutActivityTracker() (string, error)
	GetLeastPowerConnectionZone() (string, error)
	LoadRegionPrefsFromFile(string) error
	HasRegionData() bool
	RemoveRegionForTest(string)
	ReplaceCBRRule(updatedExistingRule *contextbasedrestrictionsv1.Rule, eTag *string) (*contextbasedrestrictionsv1.Rule, *core.DetailedResponse, error)
	GetThreadLock() *sync.Mutex
	GetClusterIngressStatus(clusterId string) (string, error)
	GetCatalogVersionByLocator(string) (*catalogmanagementv1.Version, error)
	CreateProjectFromConfig(config *ProjectsConfig) (*projects.Project, *core.DetailedResponse, error)
	GetProject(projectID string) (*projects.Project, *core.DetailedResponse, error)
	GetProjectConfigs(projectID string) ([]projects.ProjectConfigSummary, error)
	GetConfig(configDetails *ConfigDetails) (result *projects.ProjectConfig, response *core.DetailedResponse, err error)
	DeleteProject(projectID string) (*projects.ProjectDeleteResponse, *core.DetailedResponse, error)
	CreateConfig(configDetails *ConfigDetails) (result *projects.ProjectConfig, response *core.DetailedResponse, err error)
	DeployConfig(configDetails *ConfigDetails) (result *projects.ProjectConfigVersion, response *core.DetailedResponse, err error)
	CreateDaConfig(configDetails *ConfigDetails) (result *projects.ProjectConfig, response *core.DetailedResponse, err error)
	CreateConfigFromCatalogJson(configDetails *ConfigDetails, catalogJson string) (result *projects.ProjectConfig, response *core.DetailedResponse, err error)
	UpdateConfig(configDetails *ConfigDetails, configuration projects.ProjectConfigDefinitionPatchIntf) (result *projects.ProjectConfig, response *core.DetailedResponse, err error)
	ValidateProjectConfig(configDetails *ConfigDetails) (result *projects.ProjectConfigVersion, response *core.DetailedResponse, err error)
	IsConfigDeployed(configDetails *ConfigDetails) (projectConfig *projects.ProjectConfigVersion, isDeployed bool)
	UndeployConfig(details *ConfigDetails) (result *projects.ProjectConfigVersion, response *core.DetailedResponse, err error)
	IsUndeploying(details *ConfigDetails) (projectConfig *projects.ProjectConfigVersion, isUndeploying bool)
	CreateStackFromConfigFile(stackConfig *ConfigDetails, stackConfigPath string, catalogJsonPath string) (result *projects.StackDefinition, response *core.DetailedResponse, err error)
	GetProjectConfigVersion(configDetails *ConfigDetails, version int64) (result *projects.ProjectConfigVersion, response *core.DetailedResponse, err error)
	GetStackMembers(stackConfig *ConfigDetails) ([]*projects.ProjectConfig, error)
	SyncConfig(projectID string, configID string) (response *core.DetailedResponse, err error)
	LookupMemberNameByID(stackDetails *projects.ProjectConfig, memberID string) (string, error)
	GetSchematicsJobLogs(jobID string) (result *schematics.JobLog, response *core.DetailedResponse, err error)
	GetSchematicsJobLogsText(jobID string) (logs string, err error)
	ArePipelineActionsRunning(stackConfig *ConfigDetails) (bool, error)
	GetSchematicsJobLogsForMember(member *projects.ProjectConfig, memberName string) (string, string)
}

// CloudInfoServiceOptions structure used as input params for service constructor.
type CloudInfoServiceOptions struct {
	ApiKey                    string
	Authenticator             *core.IamAuthenticator
	VpcService                vpcService
	ResourceControllerService resourceControllerService
	ResourceManagerService    resourceManagerService
	IamIdentityService        iamIdentityService
	IamPolicyService          iamPolicyService
	CbrService                cbrService
	ContainerClient           containerClient
	RegionPrefs               []RegionData
	IcdService                icdService
	ProjectsService           projectsService
	CatalogService            catalogService
	SchematicsService         schematicsService
	// StackDefinitionCreator is used to create stack definitions and only added to support testing/mocking
	StackDefinitionCreator StackDefinitionCreator
}

// RegionData is a data structure used for holding configurable information about a region.
// Most of this data is configured by the caller in order to affect certain processing routines.
type RegionData struct {
	Name          string
	UseForTest    bool `yaml:"useForTest"`
	TestPriority  int  `yaml:"testPriority"`
	Endpoint      string
	Status        string
	ResourceCount int
}

// vpcService interface for an external VPC Service API. Used for mocking external service in tests.
type vpcService interface {
	ListRegions(*vpcv1.ListRegionsOptions) (*vpcv1.RegionCollection, *core.DetailedResponse, error)
	GetRegion(*vpcv1.GetRegionOptions) (*vpcv1.Region, *core.DetailedResponse, error)
	NewGetRegionOptions(string) *vpcv1.GetRegionOptions
	ListVpcs(*vpcv1.ListVpcsOptions) (*vpcv1.VPCCollection, *core.DetailedResponse, error)
	SetServiceURL(string) error
}

// iamIdentityService interface for an external IBM IAM Identity V1 Service API. Used for mocking.
type iamIdentityService interface {
	GetAPIKeysDetails(*iamidentityv1.GetAPIKeysDetailsOptions) (*iamidentityv1.APIKey, *core.DetailedResponse, error)
}

type iamPolicyService interface {
	DeletePolicy(deletePolicyOptions *iampolicymanagementv1.DeletePolicyOptions) (response *core.DetailedResponse, err error)
}

// resourceControllerService for external Resource Controller V2 Service API. Used for mocking.
type resourceControllerService interface {
	NewListResourceInstancesOptions() *resourcecontrollerv2.ListResourceInstancesOptions
	ListResourceInstances(*resourcecontrollerv2.ListResourceInstancesOptions) (*resourcecontrollerv2.ResourceInstancesList, *core.DetailedResponse, error)
}

// resourceManagerService for external Resource Manager V2 Service API. Used for mocking.
type resourceManagerService interface {
	NewListResourceGroupsOptions() *resourcemanagerv2.ListResourceGroupsOptions
	ListResourceGroups(*resourcemanagerv2.ListResourceGroupsOptions) (*resourcemanagerv2.ResourceGroupList, *core.DetailedResponse, error)
}

// ibmPowerService for external IBM Powercloud Service API. Used for mocking.
type ibmPICloudConnectionClient interface {
	GetAll() (*ibmpimodels.CloudConnections, error)
}

// containerClient interface for external Kubernetes Cluster Service API. Used for mocking.
type containerClient interface {
	Clusters() containerv2.Clusters
	Albs() containerv2.Alb
}

// cbrService interface for external Context Based Restrictions Service API. Used for mocking.
type cbrService interface {
	GetRule(*contextbasedrestrictionsv1.GetRuleOptions) (*contextbasedrestrictionsv1.Rule, *core.DetailedResponse, error)
	ReplaceRule(*contextbasedrestrictionsv1.ReplaceRuleOptions) (*contextbasedrestrictionsv1.Rule, *core.DetailedResponse, error)
	GetZone(*contextbasedrestrictionsv1.GetZoneOptions) (*contextbasedrestrictionsv1.Zone, *core.DetailedResponse, error)
}

// icdService for external Cloud Database V5 Service API. Used for mocking.
type icdService interface {
	NewListDeployablesOptions() *clouddatabasesv5.ListDeployablesOptions
	ListDeployables(*clouddatabasesv5.ListDeployablesOptions) (*clouddatabasesv5.ListDeployablesResponse, *core.DetailedResponse, error)
}

// projectsService for external Projects V1 Service API. Used for mocking.
type projectsService interface {
	CreateProject(createProjectOptions *projects.CreateProjectOptions) (result *projects.Project, response *core.DetailedResponse, err error)
	GetProject(getProjectOptions *projects.GetProjectOptions) (result *projects.Project, response *core.DetailedResponse, err error)
	UpdateProject(updateProjectOptions *projects.UpdateProjectOptions) (result *projects.Project, response *core.DetailedResponse, err error)
	DeleteProject(deleteProjectOptions *projects.DeleteProjectOptions) (result *projects.ProjectDeleteResponse, response *core.DetailedResponse, err error)

	NewCreateConfigOptions(projectID string, definition projects.ProjectConfigDefinitionPrototypeIntf) *projects.CreateConfigOptions
	NewConfigsPager(listConfigsOptions *projects.ListConfigsOptions) (*projects.ConfigsPager, error)
	NewGetConfigVersionOptions(projectID string, id string, version int64) *projects.GetConfigVersionOptions
	GetConfigVersion(getConfigVersionOptions *projects.GetConfigVersionOptions) (result *projects.ProjectConfigVersion, response *core.DetailedResponse, err error)

	CreateConfig(createConfigOptions *projects.CreateConfigOptions) (result *projects.ProjectConfig, response *core.DetailedResponse, err error)
	UpdateConfig(updateConfigOptions *projects.UpdateConfigOptions) (result *projects.ProjectConfig, response *core.DetailedResponse, err error)
	GetConfig(getConfigOptions *projects.GetConfigOptions) (result *projects.ProjectConfig, response *core.DetailedResponse, err error)
	DeleteConfig(deleteConfigOptions *projects.DeleteConfigOptions) (result *projects.ProjectConfigDelete, response *core.DetailedResponse, err error)

	CreateStackDefinition(createStackDefinitionOptions *projects.CreateStackDefinitionOptions) (result *projects.StackDefinition, response *core.DetailedResponse, err error)
	NewCreateStackDefinitionOptions(projectID string, id string, stackDefinition *projects.StackDefinitionBlockPrototype) *projects.CreateStackDefinitionOptions
	UpdateStackDefinition(updateStackDefinitionOptions *projects.UpdateStackDefinitionOptions) (result *projects.StackDefinition, response *core.DetailedResponse, err error)
	GetStackDefinition(getStackDefinitionOptions *projects.GetStackDefinitionOptions) (result *projects.StackDefinition, response *core.DetailedResponse, err error)
	ValidateConfig(validateConfigOptions *projects.ValidateConfigOptions) (result *projects.ProjectConfigVersion, response *core.DetailedResponse, err error)
	Approve(approveOptions *projects.ApproveOptions) (result *projects.ProjectConfigVersion, response *core.DetailedResponse, err error)
	DeployConfig(deployConfigOptions *projects.DeployConfigOptions) (result *projects.ProjectConfigVersion, response *core.DetailedResponse, err error)
	UndeployConfig(unDeployConfigOptions *projects.UndeployConfigOptions) (result *projects.ProjectConfigVersion, response *core.DetailedResponse, err error)

	SyncConfig(syncConfigOptions *projects.SyncConfigOptions) (response *core.DetailedResponse, err error)
}

// catalogService for external Data Catalog V1 Service API. Used for mocking.
type catalogService interface {
	GetVersion(getVersionOptions *catalogmanagementv1.GetVersionOptions) (result *catalogmanagementv1.Offering, response *core.DetailedResponse, err error)
}

// schematicsService for external Schematics V1 Service API. Used for mocking.
type schematicsService interface {
	ListJobLogs(listJobLogsOptions *schematics.ListJobLogsOptions) (result *schematics.JobLog, response *core.DetailedResponse, err error)
}

// ReplaceCBRRule replaces a CBR rule using the provided options.
// updatedExistingRule is the rule to be replaced with the changes already made.
// eTag is the eTag of the existing rule that is being replaced.
func (infoSvc *CloudInfoService) ReplaceCBRRule(updatedExistingRule *contextbasedrestrictionsv1.Rule, eTag *string) (*contextbasedrestrictionsv1.Rule, *core.DetailedResponse, error) {
	// Ensure that the CBR service is initialized in the CloudInfoService
	if infoSvc.cbrService == nil {
		return nil, nil, errors.New("CBR service is not initialized")
	}

	updatedRuleOptions := &contextbasedrestrictionsv1.ReplaceRuleOptions{
		RuleID:          updatedExistingRule.ID,
		Description:     updatedExistingRule.Description,
		Contexts:        updatedExistingRule.Contexts,
		Resources:       updatedExistingRule.Resources,
		Operations:      updatedExistingRule.Operations,
		EnforcementMode: updatedExistingRule.EnforcementMode,
		IfMatch:         eTag,
	}
	// Call the ReplaceRuleWithContext method of the CBR service
	rule, response, err := infoSvc.cbrService.ReplaceRule(updatedRuleOptions)

	if err != nil {
		return nil, response, err
	}

	return rule, response, nil
}

// SortedRegionsDataByPriority is an array of RegionData struct that is used as a receiver to implement the
// sort interface (Len/Less/Swap) with supplied methods to sort the array on the field RegionData.TestPriority.
type SortedRegionsDataByPriority []RegionData

func (regions SortedRegionsDataByPriority) Len() int { return len(regions) }
func (regions SortedRegionsDataByPriority) Less(i, j int) bool {
	return regions[i].TestPriority < regions[j].TestPriority
}
func (regions SortedRegionsDataByPriority) Swap(i, j int) {
	regions[i], regions[j] = regions[j], regions[i]
}

// NewCloudInfoServiceWithKey is a factory function used for creating a new initialized service structure.
// This function can be called if an IBM Cloud API Key is known and passed in directly.
// Returns a pointer to an initialized CloudInfoService and error.
func NewCloudInfoServiceWithKey(options CloudInfoServiceOptions) (*CloudInfoService, error) {
	infoSvc := new(CloudInfoService)

	// need a valid key
	if len(options.ApiKey) == 0 {
		log.Println("ERROR: empty API KEY")
		return nil, errors.New("empty API Key supplied")
	}

	// if authenticator is not supplied, create new IamAuthenticator with supplied api key
	if options.Authenticator != nil {
		infoSvc.authenticator = options.Authenticator
	} else {
		infoSvc.authenticator = &core.IamAuthenticator{
			ApiKey: options.ApiKey,
		}
	}
	infoSvc.ApiKey = options.ApiKey
	// if IamIdentity is not supplied, use default external service
	if options.IamIdentityService != nil {
		infoSvc.iamIdentityService = options.IamIdentityService
	} else {
		iamService, iamErr := iamidentityv1.NewIamIdentityV1(&iamidentityv1.IamIdentityV1Options{
			Authenticator: infoSvc.authenticator,
		})
		if iamErr != nil {
			log.Println("ERROR: Could not create NewIamIdentityV1 service:", iamErr)
			return nil, iamErr
		}
		infoSvc.iamIdentityService = iamService
	}

	// if IamPolicyService is not supplied, use default external service
	if options.IamPolicyService != nil {
		infoSvc.iamPolicyService = options.IamPolicyService
	} else {
		policyService, err := iampolicymanagementv1.NewIamPolicyManagementV1UsingExternalConfig(
			&iampolicymanagementv1.IamPolicyManagementV1Options{
				Authenticator: infoSvc.authenticator,
			})
		if err != nil {
			log.Println("ERROR: Could not create NewIamPolicyManagementV1 service:", err)
			return nil, err
		}
		infoSvc.iamPolicyService = policyService
	}

	// if vpcService is not supplied, use default of external service
	if options.VpcService != nil {
		infoSvc.vpcService = options.VpcService
	} else {
		// Instantiate the service with an API key based IAM authenticator
		vpcService, vpcErr := vpcv1.NewVpcV1(&vpcv1.VpcV1Options{
			Authenticator: infoSvc.authenticator,
		})
		if vpcErr != nil {
			log.Println("ERROR: Could not create NewVpcV1 service:", vpcErr)
			return nil, vpcErr
		}

		infoSvc.vpcService = vpcService
	}

	// if CbrService is not supplied, use default of external service
	if options.CbrService != nil {
		infoSvc.cbrService = options.CbrService
	} else {
		// Instantiate the service with an API key based IAM authenticator
		cbrService, cbrErr := contextbasedrestrictionsv1.NewContextBasedRestrictionsV1(&contextbasedrestrictionsv1.ContextBasedRestrictionsV1Options{
			Authenticator: infoSvc.authenticator,
		})

		if cbrErr != nil {
			log.Println("ERROR: Could not create contextbasedrestrictionsv1 service:", cbrErr)
			return nil, cbrErr
		}

		infoSvc.cbrService = cbrService
	}

	// if containerClient is not supplied, use default external service
	if options.ContainerClient != nil {
		infoSvc.containerClient = options.ContainerClient
	} else {
		// Create a new Bluemix session
		sess, sessErr := session.New(&bluemix.Config{
			BluemixAPIKey: infoSvc.authenticator.ApiKey, // pragma: allowlist secret
		})
		if sessErr != nil {
			log.Println("ERROR: Could not create Bluemix session:", sessErr)
			return nil, sessErr
		}

		// Initialize the container service client with the session
		containerClient, containerErr := containerv2.New(sess)
		if containerErr != nil {
			log.Println("ERROR: Could not create container service client:", containerErr)
			return nil, containerErr
		}
		infoSvc.containerClient = containerClient
	}
	// if resourceControllerService is not supplied use new external
	if options.ResourceControllerService != nil {
		infoSvc.resourceControllerService = options.ResourceControllerService
	} else {
		controllerClient, resCtrlErr := resourcecontrollerv2.NewResourceControllerV2(&resourcecontrollerv2.ResourceControllerV2Options{
			Authenticator: infoSvc.authenticator,
		})
		if resCtrlErr != nil {
			log.Println("Error creating resourcecontrollerv2 client:", resCtrlErr)
			return nil, resCtrlErr
		}

		infoSvc.resourceControllerService = controllerClient
	}

	// if resourceManagerService is not supplied use new external
	if options.ResourceControllerService != nil {
		infoSvc.resourceManagerService = options.ResourceManagerService
	} else {
		managerClient, resMgrErr := resourcemanagerv2.NewResourceManagerV2(&resourcemanagerv2.ResourceManagerV2Options{
			Authenticator: infoSvc.authenticator,
		})
		if resMgrErr != nil {
			log.Println("Error creating resourcemanagerv2 client:", resMgrErr)
			return nil, resMgrErr
		}

		infoSvc.resourceManagerService = managerClient
	}

	// if icdService is not supplied use new external
	if options.IcdService != nil {
		infoSvc.icdService = options.IcdService
	} else {
		icdClient, icdMgrErr := clouddatabasesv5.NewCloudDatabasesV5(&clouddatabasesv5.CloudDatabasesV5Options{
			Authenticator: infoSvc.authenticator,
		})
		if icdMgrErr != nil {
			log.Println("Error creating clouddatabasesv5 client:", icdMgrErr)
			return nil, icdMgrErr
		}

		infoSvc.icdService = icdClient
	}

	if options.ProjectsService != nil {
		infoSvc.projectsService = options.ProjectsService
	} else {
		projectsClient, projectsErr := projects.NewProjectV1(&projects.ProjectV1Options{
			Authenticator: infoSvc.authenticator,
		})
		if projectsErr != nil {
			log.Println("Error creating projects client:", projectsErr)
			return nil, projectsErr
		}

		infoSvc.projectsService = projectsClient

	}

	if options.CatalogService != nil {
		infoSvc.catalogService = options.CatalogService
	} else {
		catalogClient, catalogErr := catalogmanagementv1.NewCatalogManagementV1(&catalogmanagementv1.CatalogManagementV1Options{
			Authenticator: infoSvc.authenticator,
		})
		if catalogErr != nil {
			log.Println("Error creating catalog client:", catalogErr)
			return nil, catalogErr
		}

		infoSvc.catalogService = catalogClient

	}

	if options.SchematicsService != nil {
		infoSvc.schematicsService = options.SchematicsService
	} else {
		schematicsClient, schematicsErr := schematics.NewSchematicsV1(&schematics.SchematicsV1Options{
			Authenticator: infoSvc.authenticator,
		})
		if schematicsErr != nil {
			log.Println("Error creating schematics client:", schematicsErr)
			return nil, schematicsErr
		}

		infoSvc.schematicsService = schematicsClient
	}
	if options.StackDefinitionCreator != nil {
		infoSvc.stackDefinitionCreator = options.StackDefinitionCreator
	} else {
		infoSvc.stackDefinitionCreator = infoSvc
	}

	return infoSvc, nil
}

// NewCloudInfoServiceFromEnv is a factory function used for creating a new initialized service structure.
// This function can be called if the IBM Cloud API Key should be extracted from an existing OS level environment variable.
// Returns a pointer to an initialized CloudInfoService and error.
func NewCloudInfoServiceFromEnv(apiKeyEnv string, options CloudInfoServiceOptions) (*CloudInfoService, error) {
	apiKey := os.Getenv(apiKeyEnv)
	if apiKey == "" {
		return nil, errors.New("no API key Environment variable set")
	}

	options.ApiKey = apiKey

	return NewCloudInfoServiceWithKey(options)
}

func (infoSvc *CloudInfoService) GetThreadLock() *sync.Mutex {
	return &infoSvc.lock
}

type StackDefinitionCreator interface {
	CreateStackDefinitionWrapper(options *projects.CreateStackDefinitionOptions, members []projects.ProjectConfig) (*projects.StackDefinition, *core.DetailedResponse, error)
}
