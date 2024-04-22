// Package cloudinfo contains functions and methods for searching and detailing various resources located in the IBM Cloud
package cloudinfo

import (
	"errors"
	projects "github.com/IBM/project-go-sdk/projectv1"
	"log"
	"os"
	"sync"

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
	regionsData               []RegionData
	lock                      sync.Mutex
	icdService                icdService
	projectsService           projectsService
	ApiKey                    string
}

// interface for the cloudinfo service (can be mocked in tests)
type CloudInfoServiceI interface {
	GetLeastVpcTestRegion() (string, error)
	GetLeastVpcTestRegionWithoutActivityTracker() (string, error)
	GetLeastPowerConnectionZone() (string, error)
	LoadRegionPrefsFromFile(string) error
	HasRegionData() bool
	RemoveRegionForTest(string)
	GetThreadLock() *sync.Mutex
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
	UpdateStackDefinition(updateStackDefinitionOptions *projects.UpdateStackDefinitionOptions) (result *projects.StackDefinition, response *core.DetailedResponse, err error)
	GetStackDefinition(getStackDefinitionOptions *projects.GetStackDefinitionOptions) (result *projects.StackDefinition, response *core.DetailedResponse, err error)
	ValidateConfig(validateConfigOptions *projects.ValidateConfigOptions) (result *projects.ProjectConfigVersion, response *core.DetailedResponse, err error)
	Approve(approveOptions *projects.ApproveOptions) (result *projects.ProjectConfigVersion, response *core.DetailedResponse, err error)
	DeployConfig(deployConfigOptions *projects.DeployConfigOptions) (result *projects.ProjectConfigVersion, response *core.DetailedResponse, err error)
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
