package cloudinfo

import (
	project "github.com/IBM/project-go-sdk/projectv1"
)

type CatalogJson struct {
	Products []struct {
		Label           string   `json:"label"`
		Name            string   `json:"name"`
		ProductKind     string   `json:"product_kind"`
		Tags            []string `json:"tags"`
		OfferingIconUrl string   `json:"offering_icon_url"`
		Flavors         []struct {
			Compliance struct {
			} `json:"compliance"`
			Architecture struct {
			} `json:"architecture"`
		} `json:"flavors"`
	} `json:"products"`
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
	StackLocatorID    string
	StackDefinition   *project.StackDefinitionBlockPrototype
	EnvironmentID     *string
	Members           []project.StackConfigMember
	ComplianceProfile *project.ProjectComplianceProfile
}

// ProjectsConfig Config for creating a project
type ProjectsConfig struct {
	ProjectID          string                            `json:"project_id,omitempty"`
	Location           string                            `json:"location,omitempty"`
	ProjectName        string                            `json:"project_name,omitempty"`
	ProjectDescription string                            `json:"project_description,omitempty"`
	ResourceGroup      string                            `json:"resource_group,omitempty"`
	DestroyOnDelete    bool                              `json:"destroy_on_delete"`
	MonitoringEnabled  bool                              `json:"monitoring_enabled"`
	AutoDeploy         bool                              `json:"auto_deploy"`
	Configs            []project.ProjectConfigPrototype  `json:"configs,omitempty"`
	Environments       []project.EnvironmentPrototype    `json:"environments,omitempty"`
	Headers            map[string]string                 `json:"headers,omitempty"`
	Store              *project.ProjectDefinitionStore   `json:"store,omitempty"`
	ComplianceProfile  *project.ProjectComplianceProfile `json:"compliance_profile,omitempty"`
}
