package testhelper

// Exemptions Struct to hold the list of exemptions
type Exemptions struct {
	List []string
}

// IsExemptedResource Checks if resource string is in the list of exemptions
func (exemptions Exemptions) IsExemptedResource(resource string) bool {

	for _, exemption := range exemptions.List {
		if exemption == resource {
			return true
		}
	}

	return false
}
