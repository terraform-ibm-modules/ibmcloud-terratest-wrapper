package cloudinfo

import (
	"github.com/IBM/go-sdk-core/v5/core"
	project "github.com/IBM/project-go-sdk/projectv1"
)

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
	OfferingInputs      []CatalogInput
	ConfigID            string // The ID of the config after it is deployed to the project
	ConfigName          string
	ContainerConfigID   string // Temporary support for containers until they are removed
	ContainerConfigName string
	ExistingConfigID    string
	Enabled             *bool // Use pointer to distinguish between unset (nil), false, and true
	OnByDefault         *bool // Use pointer to distinguish between unset (nil), false, and true
	OfferingID          string
	OfferingName        string
	OfferingFlavor      string
	OfferingLabel       string
	OfferingInstallKind InstallKind // Only needed for the root DA to onboard the offering
	VersionLocator      string
	VersionID           string
	CatalogID           string
	ResolvedVersion     string
	Dependencies        []AddonConfig
	// IsRequired indicates if this dependency is marked as required in the catalog
	IsRequired *bool
	// RequiredBy lists the names of dependencies that require this one (for tracing force-enabled dependencies)
	RequiredBy []string
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

// newAddonConfig creates a new AddonConfig with the provided parameters and sensible defaults
// It defaults OfferingInstallKind to InstallKindTerraform if not provided
// prefix is used to create a unique name for the config
// name is the name of the offering
// flavor is the flavor of the offering
// installKind is the kind of installation (Terraform or Stack)
// inputs is a map of input variables for the offering
// It returns an AddonConfig struct
func newAddonConfig(prefix, name, flavor string, installKind *InstallKind, inputs map[string]interface{}) AddonConfig {
	config := AddonConfig{
		Prefix:         prefix,
		OfferingName:   name,
		OfferingFlavor: flavor,
		Inputs:         inputs,
		Enabled:        core.BoolPtr(true), // Pointer to true
		OnByDefault:    core.BoolPtr(true), // Pointer to true
	}

	// Default to Terraform install kind if not provided
	if installKind == nil {
		config.OfferingInstallKind = *NewInstallKindTerraform()
	} else {
		config.OfferingInstallKind = *installKind
	}

	return config
}

// NewAddonConfigTerraform creates a new AddonConfig with Terraform install kind
func NewAddonConfigTerraform(prefix, name, flavor string, inputs map[string]interface{}) AddonConfig {
	return newAddonConfig(prefix, name, flavor, NewInstallKindTerraform(), inputs)
}

// NewAddonConfigStack creates a new AddonConfig with Stack install kind
func NewAddonConfigStack(prefix, name, flavor string, inputs map[string]interface{}) AddonConfig {
	return newAddonConfig(prefix, name, flavor, NewInstallKindStack(), inputs)
}

// ConfigStates a flat list of config states
type ConfigStates struct {
	States []struct {
		Name      string `json:"name"`
		State     string `json:"state"`
		StateCode string `json:"state_code"`
	} `json:"states"`
	// ConfigID is the ID of the config
	ConfigID string `json:"config_id"`
	// ConfigName is the name of the config
	ConfigName string `json:"config_name"`
}

// InputDetail holds details about a configuration input
type InputDetail struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Required    bool        `json:"required"`
	Value       interface{} `json:"value"`
	Description string      `json:"description,omitempty"`
	Hidden      bool        `json:"hidden,omitempty"`
}
