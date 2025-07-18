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
		if deployable.Type == nil {
			// Safe to skip: we're filtering a list of deployables to find matching types.
			// A deployable without a type cannot match our criteria, so we continue
			// processing other deployables that might be valid.
			infoSvc.Logger.ShortWarn("Skipping deployable with nil Type")
			continue
		}
		if *deployable.Type == icdType {
			for _, version := range deployable.Versions {
				if version.Status == nil {
					// Safe to skip: we're looking for stable versions only.
					// A version without a status cannot be determined to be stable,
					// so we continue processing other versions that might be valid.
					infoSvc.Logger.ShortWarn("Skipping version with nil Status")
					continue
				}
				if version.Version == nil {
					// Safe to skip: we need the version string to return to the caller.
					// A version without a version string is unusable, so we continue
					// processing other versions that might have valid version strings.
					infoSvc.Logger.ShortWarn("Skipping version with nil Version")
					continue
				}
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
