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
