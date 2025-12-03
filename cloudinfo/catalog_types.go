package cloudinfo

// CatalogJson struct represents the structure of catalog JSON data
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
				TypeMetadata string      `json:"type_metadata"`
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
					DisplayName string      `json:"displayname,omitempty"`
					Value       interface{} `json:"value"`
				} `json:"options,omitempty"`
			} `json:"configuration"`
			Outputs []struct {
				Key         string `json:"key"`
				Description string `json:"description"`
			} `json:"outputs"`
			Dependencies []struct {
				CatalogID   string   `json:"catalog_id,omitempty"`
				ID          string   `json:"id,omitempty"`
				Name        string   `json:"name"`
				Version     string   `json:"version,omitempty"`
				Flavors     []string `json:"flavors,omitempty"`
				InstallType string   `json:"install_type,omitempty"`
			} `json:"dependencies,omitempty"`
			InstallType string `json:"install_type"`
		} `json:"flavors"`
	} `json:"products"`
}

// CatalogInput represents an input from the catalog configuration
type CatalogInput struct {
	Key          string      `json:"key"`
	Type         string      `json:"type"`
	TypeMetadata string      `json:"type_metadata"`
	DefaultValue interface{} `json:"default_value"`
	Required     bool        `json:"required"`
	Description  string      `json:"description"`
	Options      []struct {
		DisplayName string      `json:"displayname,omitempty"`
		Value       interface{} `json:"value"`
	} `json:"options,omitempty"`
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

// DeployedAddonsDetails represents the details of deployed addons
type DeployedAddonsDetails struct {
	ProjectID string `json:"project_id"`
	Configs   []struct {
		Name     string `json:"name"`
		ConfigID string `json:"config_id"`
	} `json:"configs"`
}

type DependencyError struct {
	Addon                 OfferingReferenceDetail
	DependencyRequired    OfferingReferenceDetail
	DependenciesAvailable []OfferingReferenceDetail
}
