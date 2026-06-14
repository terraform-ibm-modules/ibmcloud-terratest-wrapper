package cloudinfo

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// GlobalCatalogBaseURL is the base URL for the Global Catalog API
// This can be overridden in tests to point to a mock server
var GlobalCatalogBaseURL = "https://globalcatalog.cloud.ibm.com"

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

// GetAvailableIcdVersionsGen2 retrieves the available versions of a Gen2 ICD service.
// service is the service name (e.g., "databases-for-postgresql")
// plan is the plan name (e.g., "standard-gen2")
// region is the region (e.g., "ca-tor")
// returns a list of versions (excluding dead and hidden) of the specified Gen2 ICD service.
func (infoSvc *CloudInfoService) GetAvailableIcdVersionsGen2(service, plan, region string) ([]string, error) {
	// Get access token using existing authenticator
	token, err := infoSvc.authenticator.GetToken()
	if err != nil {
		return nil, fmt.Errorf("error getting auth token: %w", err)
	}

	// Build URL for Global Catalog API
	// User input (service, plan, region) is properly escaped and only affects the path, not the host
	reqURL := fmt.Sprintf("%s/api/v1/%s-%s:%s",
		GlobalCatalogBaseURL,
		url.PathEscape(service),
		url.PathEscape(plan),
		url.PathEscape(region))

	// Create HTTP request
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Add headers
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Add("Accept", "application/json")

	// Execute request
	// #nosec G107 - URL is constructed from hardcoded base + escaped user input (path only, not host)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %w", err)
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			if closeErr := resp.Body.Close(); closeErr != nil {
				infoSvc.Logger.ShortWarn(fmt.Sprintf("Error closing response body: %v", closeErr))
			}
		}
	}()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	// Handle 404 gracefully - service not available in this region/plan
	if resp.StatusCode == 404 {
		infoSvc.Logger.ShortWarn(fmt.Sprintf("Gen2 service %s-%s not available in region %s (404)", service, plan, region))
		return []string{}, nil // Return empty list, not an error
	}

	// Check other error status codes
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse JSON response
	var gen2Response struct {
		Metadata struct {
			Other struct {
				Versions []struct {
					Version string `json:"version"`
					Status  string `json:"status"`
				} `json:"versions"`
			} `json:"other"`
		} `json:"metadata"`
	}

	if err := json.Unmarshal(body, &gen2Response); err != nil {
		return nil, fmt.Errorf("error parsing JSON response: %w", err)
	}

	// Extract valid versions (filter out dead and hidden)
	versions := []string{}
	for _, version := range gen2Response.Metadata.Other.Versions {
		if version.Status != "dead" && version.Status != "hidden" {
			versions = append(versions, version.Version)
		}
	}

	if len(versions) == 0 {
		return nil, fmt.Errorf("no valid versions found for Gen2 service %s-%s in region %s", service, plan, region)
	}

	return versions, nil
}
