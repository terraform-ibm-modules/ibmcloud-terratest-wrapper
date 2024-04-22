package cloudinfo

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
		Name  string `json:"name"`
		Value string `json:"value"`
	} `json:"inputs"`
	Members []struct {
		Inputs []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"inputs"`
		Name           string `json:"name"`
		VersionLocator string `json:"version_locator"`
	} `json:"members"`
}
