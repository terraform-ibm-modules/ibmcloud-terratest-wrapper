package cloudinfo

import (
	project "github.com/IBM/project-go-sdk/projectv1"
)

type CatalogJson struct {
	Products []struct {
		Label            string   `json:"label"`
		Name             string   `json:"name"`
		ProductKind      string   `json:"product_kind"`
		Tags             []string `json:"tags"`
		Keywords         []string `json:"keywords"`
		ShortDescription string   `json:"short_description"`
		LongDescription  string   `json:"long_description"`
		OfferingDocsURL  string   `json:"offering_docs_url"`
		OfferingIconURL  string   `json:"offering_icon_url"`
		ProviderName     string   `json:"provider_name"`
		Features         []struct {
			Title       string `json:"title"`
			Description string `json:"description"`
		} `json:"features"`
		SupportDetails string `json:"support_details"`
		Flavors        []struct {
			Label            string `json:"label"`
			Name             string `json:"name"`
			WorkingDirectory string `json:"working_directory"`
			Compliance       struct {
				Authority string `json:"authority"`
				Profiles  []struct {
					ProfileName    string `json:"profile_name"`
					ProfileVersion string `json:"profile_version"`
				} `json:"profiles"`
			} `json:"compliance"`
			IamPermissions []struct {
				ServiceName string   `json:"service_name"`
				RoleCrns    []string `json:"role_crns"`
			} `json:"iam_permissions"`
			Architecture struct {
				Features []struct {
					Title       string `json:"title"`
					Description string `json:"description"`
				} `json:"features"`
				Diagrams []struct {
					Diagram struct {
						URL          string `json:"url"`
						Caption      string `json:"caption"`
						Type         string `json:"type"`
						ThumbnailURL string `json:"thumbnail_url"`
					} `json:"diagram"`
					Description string `json:"description"`
				} `json:"diagrams"`
			} `json:"architecture"`
			Configuration []struct {
				Key          string      `json:"key"`
				Type         string      `json:"type"`
				Description  string      `json:"description"`
				DefaultValue interface{} `json:"default_value"`
				Required     bool        `json:"required"`
				DisplayName  string      `json:"display_name,omitempty"`
				CustomConfig struct {
					Type             string `json:"type"`
					Grouping         string `json:"grouping"`
					OriginalGrouping string `json:"original_grouping"`
				} `json:"custom_config,omitempty"`
				Options []struct {
					DisplayName string `json:"displayname"`
					Value       string `json:"value"`
				} `json:"options,omitempty"`
			} `json:"configuration"`
			Outputs []struct {
				Key         string `json:"key"`
				Description string `json:"description"`
			} `json:"outputs"`
			InstallType string `json:"install_type"`
		} `json:"flavors"`
	} `json:"products"`
}

// CatalogInput represents an input from the catalog configuration
type CatalogInput struct {
	Key          string      `json:"key"`
	Type         string      `json:"type"`
	DefaultValue interface{} `json:"default_value"`
	Required     bool        `json:"required"`
	Description  string      `json:"description"`
}

type Stack struct {
	Inputs []struct {
		Name        string      `json:"name"`
		Description string      `json:"description"`
		Required    bool        `json:"required"`
		Type        string      `json:"type"`
		Hidden      bool        `json:"hidden"`
		Default     interface{} `json:"default"`
	} `json:"inputs"`
	Outputs []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	} `json:"outputs"`
	Members []struct {
		Inputs []struct {
			Name  string      `json:"name"`
			Value interface{} `json:"value"`
		} `json:"inputs"`
		Name           string `json:"name"`
		VersionLocator string `json:"version_locator"`
	} `json:"members"`
}

// ConfigDetails Config details for a config or stack
type ConfigDetails struct {
	ProjectID      string
	Name           string
	Description    string
	ConfigID       string
	Authorizations *project.ProjectConfigAuth
	// Inputs used to override the default inputs
	Inputs map[string]interface{}
	// Settings used to override the default settings
	Settings map[string]interface{}
	// Stack specific
	StackLocatorID  string
	StackDefinition *project.StackDefinitionBlockPrototype
	EnvironmentID   *string
	Members         []project.ProjectConfig
	// Member Config details used to override the default member inputs
	// Only need to set the name and inputs
	MemberConfigDetails []ConfigDetails
	MemberConfigs       []project.StackConfigMember

	// CatalogProductName The name of the product in the catalog. Defaults to the first product in the catalog.
	CatalogProductName string
	// CatalogFlavorName The name of the flavor in the catalog. Defaults to the first flavor in the catalog.
	CatalogFlavorName string
}

// ProjectsConfig Config for creating a project
type ProjectsConfig struct {
	ProjectID          string                           `json:"project_id,omitempty"`
	Location           string                           `json:"location,omitempty"`
	ProjectName        string                           `json:"project_name,omitempty"`
	ProjectDescription string                           `json:"project_description,omitempty"`
	ResourceGroup      string                           `json:"resource_group,omitempty"`
	DestroyOnDelete    bool                             `json:"destroy_on_delete"`
	MonitoringEnabled  bool                             `json:"monitoring_enabled"`
	AutoDeploy         bool                             `json:"auto_deploy"`
	Configs            []project.ProjectConfigPrototype `json:"configs,omitempty"`
	Environments       []project.EnvironmentPrototype   `json:"environments,omitempty"`
	Headers            map[string]string                `json:"headers,omitempty"`
	Store              *project.ProjectDefinitionStore  `json:"store,omitempty"`
}

type AddonConfig struct {
	Prefix              string
	Inputs              map[string]interface{}
	ExistingConfigID    string
	Enabled             bool
	OnByDefault         bool
	OfferingID          string
	OfferingName        string
	OfferingFlavor      string
	OfferingLabel       string
	OfferingInstallKind InstallKind
	VersionLocator      string
	ResolvedVersion     string
	VersionConstraints  string
	Dependencies        []AddonConfig
}

// InstallKind represents the type of install
type InstallKind string

const (
	// InstallKindTerraform represents a terraform installation
	InstallKindTerraform InstallKind = "terraform"
	// InstallKindStack represents a stack installation
	InstallKindStack InstallKind = "stack"
)

// String returns the string representation of the InstallKind
func (i InstallKind) String() string {
	return string(i)
}

// Valid checks if the InstallKind is valid
func (i InstallKind) Valid() bool {
	switch i {
	case InstallKindTerraform, InstallKindStack:
		return true
	default:
		return false
	}
}

// NewInstallKindTerraform returns a pointer to InstallKindTerraform
func NewInstallKindTerraform() *InstallKind {
	k := InstallKindTerraform
	return &k
}

// NewInstallKindStack returns a pointer to InstallKindStack
func NewInstallKindStack() *InstallKind {
	k := InstallKindStack
	return &k
}

// OfferingReferenceResponse represents the entire response object for offering references
type OfferingReferenceResponse struct {
	Included              IncludedReferences     `json:"included"`
	Required              RequiredReferences     `json:"required"`
	Optional              OptionalReferences     `json:"optional"`
	Extension             map[string]interface{} `json:"extension"`
	DependentStackMembers map[string]interface{} `json:"dependent_stack_members"`
}

// IncludedReferences represents the included section of offering references
type IncludedReferences struct {
	OfferingReferences []SimpleOfferingReference `json:"offering_references"`
}

// RequiredReferences represents the required section of offering references
type RequiredReferences struct {
	OfferingReferences []OfferingReferenceItem `json:"offering_references,omitempty"`
}

// OptionalReferences represents the optional section of offering references
type OptionalReferences struct {
	OfferingReferences []OfferingReferenceItem `json:"offering_references"`
}

// SimpleOfferingReference represents a simple offering reference with just name and source
type SimpleOfferingReference struct {
	Name   string `json:"name"`
	Source string `json:"source"`
}

// OfferingReferenceItem represents an offering reference with detailed information
type OfferingReferenceItem struct {
	Name              string                  `json:"name"`
	OfferingReference OfferingReferenceDetail `json:"offering_reference,omitempty"`
}

// OfferingReferenceDetail represents the detailed information about an offering
type OfferingReferenceDetail struct {
	CatalogID            string   `json:"catalog_id"`
	ID                   string   `json:"id"`
	Label                string   `json:"label"`
	Name                 string   `json:"name"`
	OfferingIconURL      string   `json:"offering_icon_url"`
	ShortDescription     string   `json:"short_description"`
	LongDescription      string   `json:"long_description"`
	ProductKind          string   `json:"product_kind"`
	Tags                 []string `json:"tags"`
	FormatKind           string   `json:"format_kind"`
	TargetKind           string   `json:"target_kind"`
	Version              string   `json:"version"`
	Flavor               Flavor   `json:"flavor"`
	VersionLocator       string   `json:"version_locator"`
	Hidden               bool     `json:"hidden"`
	Cost                 Cost     `json:"cost"`
	DeployTime           int64    `json:"deploy_time"`
	OnByDefault          bool     `json:"on_by_default"`
	DefaultFlavor        string   `json:"default_flavor"`
	ParentVersionLocator string   `json:"parent_version_locator"`
	IsPublicCatalog      bool     `json:"is_public_catalog"`
}

// Flavor represents the flavor information of an offering
type Flavor struct {
	Name  string `json:"name"`
	Label string `json:"label"`
	Index int    `json:"index"`
}

// Cost represents the cost information of an offering
type Cost struct {
	Version              string                 `json:"version"`
	Currency             string                 `json:"currency"`
	Projects             []CostProject          `json:"projects"`
	Summary              map[string]interface{} `json:"summary"`
	TotalHourlyCost      string                 `json:"totalHourlyCost"`
	TotalMonthlyCost     string                 `json:"totalMonthlyCost"`
	PastTotalHourlyCost  string                 `json:"pastTotalHourlyCost"`
	PastTotalMonthlyCost string                 `json:"pastTotalMonthlyCost"`
	DiffTotalHourlyCost  string                 `json:"diffTotalHourlyCost"`
	DiffTotalMonthlyCost string                 `json:"diffTotalMonthlyCost"`
	TimeGenerated        string                 `json:"timeGenerated"`
}

// CostProject represents a project's cost information
type CostProject struct {
	Metadata      CostMetadata           `json:"metadata"`
	PastBreakdown CostBreakdown          `json:"pastBreakdown"`
	Breakdown     CostBreakdown          `json:"breakdown"`
	Diff          CostBreakdown          `json:"diff"`
	Summary       map[string]interface{} `json:"summary"`
}

// CostMetadata represents metadata about a project's cost information
type CostMetadata struct {
	Path       string `json:"path"`
	Type       string `json:"type"`
	VcsSubPath string `json:"vcsSubPath"`
}

// CostBreakdown represents cost breakdown information
type CostBreakdown struct {
	TotalHourlyCost  string `json:"totalHourlyCost"`
	TotalMonthlyCost string `json:"totalMonthlyCost"`
}
