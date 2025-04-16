// Package cloudinfo contains functions and methods for searching and detailing various resources located in the IBM Cloud
package cloudinfo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

// getServiceURLForRegion returns the service URL for the specified region
func getRefResolverServiceURLForRegion(region string) (string, error) {
	// Define endpoints for supported regions in long format (ibm:yp:region-name)
	var endpoints = map[string]string{
		"dev":             "https://ref-resolver.us-south.devops.dev.cloud.ibm.com/devops/ref-resolver/api/v1/internal",
		"test":            "https://ref-resolver.us-south.devops.dev.cloud.ibm.com/devops/ref-resolver/api/v1/internal",
		"staging":         "https://ref-resolver.us-south.devops.dev.cloud.ibm.com/devops/ref-resolver/api/v1/internal",
		"ibm:yp:us-south": "https://ref-resolver.us-south.devops.dev.cloud.ibm.com/devops/ref-resolver/api/v1/internal",
		"ibm:yp:mon01":    "https://ref-resolver.mon01.devops.cloud.ibm.com/devops/ref-resolver/api/v1/internal",
		"ibm:yp:us-east":  "https://ref-resolver.us-east.devops.cloud.ibm.com/devops/ref-resolver/api/v1/internal",
		"ibm:yp:ca-tor":   "https://ref-resolver.ca-tor.devops.cloud.ibm.com/devops/ref-resolver/api/v1/internal",
		"ibm:yp:eu-de":    "https://ref-resolver.eu-de.devops.cloud.ibm.com/devops/ref-resolver/api/v1/internal",
		"ibm:yp:eu-gb":    "https://ref-resolver.eu-gb.devops.cloud.ibm.com/devops/ref-resolver/api/v1/internal",
		// Add short format regions mapping to their long format equivalents
		"us-south": "https://ref-resolver.us-south.devops.dev.cloud.ibm.com/devops/ref-resolver/api/v1/internal",
		"mon01":    "https://ref-resolver.mon01.devops.cloud.ibm.com/devops/ref-resolver/api/v1/internal",
		"us-east":  "https://ref-resolver.us-east.devops.cloud.ibm.com/devops/ref-resolver/api/v1/internal",
		"ca-tor":   "https://ref-resolver.ca-tor.devops.cloud.ibm.com/devops/ref-resolver/api/v1/internal",
		"eu-de":    "https://ref-resolver.eu-de.devops.cloud.ibm.com/devops/ref-resolver/api/v1/internal",
		"eu-gb":    "https://ref-resolver.eu-gb.devops.cloud.ibm.com/devops/ref-resolver/api/v1/internal",
	}

	if url, ok := endpoints[region]; ok {
		return url, nil
	}

	// Create a list of supported regions to include in the error message
	supportedRegions := make([]string, 0, len(endpoints))
	for k := range endpoints {
		supportedRegions = append(supportedRegions, k)
	}

	return "", fmt.Errorf("service URL for region '%s' not found. Supported regions: %s", region, strings.Join(supportedRegions, ", "))
}

// Define variables that can be overridden in tests
var (
	CloudInfo_GetRefResolverServiceURLForRegion = getRefResolverServiceURLForRegion
	CloudInfo_HttpClient                        = &http.Client{}
)

// normalizeReference ensures a reference has the correct format
func normalizeReference(reference string) string {
	if !strings.HasPrefix(reference, "ref:") {
		return "ref:" + reference
	}
	return reference
}

// isUUID checks if a string is in UUID format
func isUUID(s string) bool {
	pattern := regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`)
	return pattern.MatchString(s)
}

// needsProjectContext determines if a reference needs a project context
func needsProjectContext(reference string) bool {
	// Normalize the reference first
	reference = normalizeReference(reference)

	// If it's already a fully qualified project reference, no additional context needed
	if strings.HasPrefix(reference, "ref://project.") {
		return false
	}

	// Check if it's a project reference that's not fully qualified
	if strings.HasPrefix(reference, "ref:/configs/") {
		return true
	}

	// Check if it starts with relative path but doesn't have project context
	if strings.HasPrefix(reference, "ref:./") && !strings.Contains(reference, "project.") {
		return true
	}

	return false
}

// extractConfigID extracts just the config ID from a reference string
func extractConfigID(reference string) (string, bool) {
	// Extract config ID from paths like ref:/configs/{configID}/inputs/prefix
	configIDPattern := regexp.MustCompile(`ref:/configs/([a-f0-9\-]+)`)
	configMatches := configIDPattern.FindStringSubmatch(reference)
	if len(configMatches) > 1 {
		return configMatches[1], true
	}
	return "", false
}

// replaceConfigIDWithName replaces a config ID with its name in a reference path
func replaceConfigIDWithName(reference, configID, configName string) string {
	// For references like ref:/configs/{configID}/inputs/prefix
	// Convert to ref:/configs/{configName}/inputs/prefix
	pattern := fmt.Sprintf("ref:/configs/%s", configID)
	replacement := fmt.Sprintf("ref:/configs/%s", configName)
	return strings.Replace(reference, pattern, replacement, 1)
}

// ResolveReferences resolves a list of references using the ref-resolver API
func (infoSvc *CloudInfoService) ResolveReferences(region string, references []Reference) (*ResolveResponse, error) {
	// Get service URL for the region using the variable that can be overridden in tests
	serviceURL, err := CloudInfo_GetRefResolverServiceURLForRegion(region)
	if err != nil {
		return nil, err
	}

	// Create the request body
	requestBody := ResolveRequest{
		References: references,
	}

	// Marshal the request body to JSON
	jsonPayload, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Log the payload for debugging
	if infoSvc.Logger != nil {
		infoSvc.Logger.ShortInfo("Sending reference resolution request")
	}

	// Create a new request
	req, err := http.NewRequest("POST", serviceURL+"/resolve", bytes.NewReader(jsonPayload))
	if err != nil {
		infoSvc.Logger.ShortError(fmt.Sprintf("Failed to create request: %v", err))
		infoSvc.Logger.ShortInfo("Request body: " + string(jsonPayload))
		infoSvc.Logger.ShortInfo("Service URL: " + serviceURL)
		infoSvc.Logger.ShortInfo("Region: " + region)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set the headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Set the authentication token
	token, err := infoSvc.authenticator.GetToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	// Send the request using the global HTTP client that can be overridden in tests
	resp, err := CloudInfo_HttpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check the status code
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("invalid status code: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	// Read the entire response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse the response
	var response ResolveResponse
	err = json.Unmarshal(bodyBytes, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}

// getProjectInfoFromNameOrID resolves a project ID or name to a ProjectInfo
func (infoSvc *CloudInfoService) getProjectInfoFromID(projectID string, projectCache map[string]*ProjectInfo) (*ProjectInfo, error) {
	// Since only project IDs are supported, directly get the project info
	return infoSvc.GetProjectInfo(projectID, projectCache)
}

// transformReferencesToQualifiedReferences transforms a slice of reference strings to fully qualified references
// This function contains the core reference transformation logic and can be tested independently
// NOTE: Currently does not support stack style relative references eg ref:../../configs/{configID}/inputs/prefix
func (infoSvc *CloudInfoService) transformReferencesToQualifiedReferences(
	refStrings []string,
	projectInfo *ProjectInfo) ([]Reference, error) {

	// URL encode the project name for use in references
	encodedProjectName := url.QueryEscape(projectInfo.Name)

	references := make([]Reference, 0, len(refStrings))

	for _, refString := range refStrings {
		normalizedRef := normalizeReference(refString)

		// Skip fully-qualified references that don't need project context
		if !needsProjectContext(normalizedRef) {
			references = append(references, Reference{
				Reference: normalizedRef,
			})
			continue
		}

		// At this point we know the reference needs project qualification

		// Check if this is a config reference
		if strings.HasPrefix(normalizedRef, "ref:/configs/") {
			// Extract config ID if present
			configID, found := extractConfigID(normalizedRef)
			if found {
				configName := ""
				// Try to get config name from cache
				if name, exists := projectInfo.Configs[configID]; exists {
					configName = name
				} else {
					// Try to fetch the config details if not in cache
					name, err := infoSvc.GetConfigName(projectInfo.ID, configID)
					if err == nil && name != "" {
						configName = name
						projectInfo.Configs[configID] = name
					}
				}

				// If we have a config name, create a project-qualified reference
				if configName != "" {
					// Extract the path after the config ID
					// Support paths with /inputs/, /outputs/, and /authorizations/
					pathRegex := regexp.MustCompile(fmt.Sprintf(`ref:/configs/%s(/(?:inputs|outputs|authorizations)/.*)?$`, configID))
					matches := pathRegex.FindStringSubmatch(normalizedRef)

					var path string
					if len(matches) > 1 && matches[1] != "" {
						path = matches[1]
					} else {
						path = "" // No path segment after config ID
					}

					// Create a project-qualified reference in the format: ref://project.{projectName}/configs/{configName}{path}
					qualifiedRef := fmt.Sprintf("ref://project.%s/configs/%s%s",
						encodedProjectName, configName, path)

					references = append(references, Reference{
						Reference: qualifiedRef,
					})
					continue
				}
			}

			// If we couldn't find the config name or extract the config ID,
			// convert to project-qualified format anyway, preserving the original path
			configPath := strings.TrimPrefix(normalizedRef, "ref:/configs/")

			// Split the path at the first occurrence of /inputs/, /outputs/, or /authorizations/
			var configIdentifier, resourcePath string

			// Try to split at known resource types
			for _, resourceType := range []string{"/inputs/", "/outputs/", "/authorizations/"} {
				if parts := strings.SplitN(configPath, resourceType, 2); len(parts) > 1 {
					configIdentifier = parts[0]
					resourcePath = resourceType + parts[1]
					break
				}
			}

			// If no resource type was found, use the full path as configID
			if resourcePath == "" {
				configIdentifier = configPath
			}

			qualifiedRef := fmt.Sprintf("ref://project.%s/configs/%s%s",
				encodedProjectName, configIdentifier, resourcePath)

			references = append(references, Reference{
				Reference: qualifiedRef,
			})
			continue
		}

		// For relative references, add project context
		if strings.HasPrefix(normalizedRef, "ref:./") {
			// Remove the './' part and replace with project-qualified path
			relPath := strings.TrimPrefix(normalizedRef, "ref:./")
			qualifiedRef := fmt.Sprintf("ref://project.%s/%s", encodedProjectName, relPath)

			references = append(references, Reference{
				Reference: qualifiedRef,
			})
			continue
		}

		// For any other reference format that needs qualification but doesn't match the above patterns,
		// convert to a fully qualified reference format without using context
		qualifiedRef := fmt.Sprintf("ref://project.%s/custom/%s",
			encodedProjectName, strings.TrimPrefix(normalizedRef, "ref:"))

		references = append(references, Reference{
			Reference: qualifiedRef,
		})
	}

	return references, nil
}

// ResolveReferencesFromStrings resolves references from a slice of strings in a specific project context
func (infoSvc *CloudInfoService) ResolveReferencesFromStrings(
	region string,
	refStrings []string,
	projectID string) (*ResolveResponse, error) {

	// Initialize a cache to hold project info to avoid redundant API calls
	projectCache := make(map[string]*ProjectInfo)

	// Get project info first
	projectInfo, err := infoSvc.getProjectInfoFromID(projectID, projectCache)
	if err != nil {
		return nil, fmt.Errorf("failed to get project info: %v", err)
	}

	if infoSvc.Logger != nil {
		infoSvc.Logger.ShortInfo(fmt.Sprintf("Using project: %s (ID: %s)", projectInfo.Name, projectInfo.ID))
	}

	references, err := infoSvc.transformReferencesToQualifiedReferences(refStrings, projectInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to transform references: %v", err)
	}

	return infoSvc.ResolveReferences(region, references)
}
