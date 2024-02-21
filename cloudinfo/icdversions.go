package cloudinfo

import (
	"fmt"
)

// GetAvailableIcdVersions will retrieve the available versions of a specified ICD type.
// icdType is the type of the ICD
// returns a list of stable versions of a specified ICD type.
func (infoSvc *CloudInfoService) GetAvailableIcdVersions(icdType string) ([]string, error) {
	listDeployablesOptions := infoSvc.icdService.NewListDeployablesOptions()
	icdVersions, _, err := infoSvc.icdService.ListDeployables(listDeployablesOptions)
	if err != nil {
		return nil, fmt.Errorf("error listing icd versions: %w", err)
	}

	versions := []string{}
	for _, deployable := range icdVersions.Deployables {
		if *deployable.Type == icdType {
			for _, version := range deployable.Versions {
				if *version.Status == "stable" {
					versions = append(versions, *version.Version)
				}
			}
		}
	}

	if len(versions) != 0 {
		return versions, nil
	}
	return nil, fmt.Errorf("version for ICD type %s not found", icdType)
}
