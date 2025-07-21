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
	"time"

	"github.com/IBM/go-sdk-core/v5/core"
)

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

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

// isDevTestStagingRegion checks if a region is a development, test, or staging environment
func isDevTestStagingRegion(region string) bool {
	devTestStagingRegions := []string{"dev", "test", "staging"}
	for _, envRegion := range devTestStagingRegions {
		if region == envRegion {
			return true
		}
	}
	return false
}

// getPreferredFallbackRegions returns a list of preferred fallback regions for the given region
// Regions are ordered by geographic proximity and reliability
func getPreferredFallbackRegions(region string) []string {
	// Regional fallback preferences based on geographic proximity
	fallbackMap := map[string][]string{
		// US regions
		"us-south":        {"us-east", "ca-tor", "eu-de", "eu-gb", "mon01"},
		"ibm:yp:us-south": {"us-east", "ca-tor", "eu-de", "eu-gb", "mon01"},
		"us-east":         {"us-south", "ca-tor", "eu-de", "eu-gb", "mon01"},
		"ibm:yp:us-east":  {"us-south", "ca-tor", "eu-de", "eu-gb", "mon01"},

		// Canadian regions
		"ca-tor":        {"us-east", "us-south", "eu-de", "eu-gb", "mon01"},
		"ibm:yp:ca-tor": {"us-east", "us-south", "eu-de", "eu-gb", "mon01"},

		// European regions
		"eu-de":        {"eu-gb", "us-south", "us-east", "ca-tor", "mon01"},
		"ibm:yp:eu-de": {"eu-gb", "us-south", "us-east", "ca-tor", "mon01"},
		"eu-gb":        {"eu-de", "us-south", "us-east", "ca-tor", "mon01"},
		"ibm:yp:eu-gb": {"eu-de", "us-south", "us-east", "ca-tor", "mon01"},

		// Montreal (special case)
		"mon01":        {"ca-tor", "us-east", "us-south", "eu-de", "eu-gb"},
		"ibm:yp:mon01": {"ca-tor", "us-east", "us-south", "eu-de", "eu-gb"},
	}

	if fallbacks, exists := fallbackMap[region]; exists {
		return fallbacks
	}

	// Default fallback order if region not specifically mapped
	return []string{"us-south", "us-east", "eu-de", "eu-gb", "ca-tor", "mon01"}
}

// Define variables that can be overridden in tests
var (
	CloudInfo_GetRefResolverServiceURLForRegion = getRefResolverServiceURLForRegion
	CloudInfo_HttpClient                        = &http.Client{}
)

const (
	defaultRetryCount       = 3
	defaultInitialRetryWait = 2 // seconds
)

// shouldRetryReferenceResolution checks if we should retry based on the error response
func shouldRetryReferenceResolution(statusCode int, body string) bool {
	// Retry on 404 errors where project provider instances cannot be found
	// This typically occurs when checking project details too quickly after project creation,
	// before the resolver API has been updated with the new project information (timing issue)
	if statusCode == 404 && strings.Contains(body, "could not be found") {
		return strings.Contains(body, "Specified provider") && strings.Contains(body, "project")
	}

	// Retry on 429 rate limiting errors
	// This is especially common in parallel test execution where multiple tests
	// hit the same API endpoints simultaneously
	if statusCode == 429 {
		return true
	}

	// Retry on all 5xx server errors (including API key validation issues)
	if statusCode >= 500 && statusCode < 600 {
		return true
	}

	return false
}

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

	// Check if it contains stack-style relative paths (../), these need project context
	if strings.Contains(reference, "../") {
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

// ResolveReferences resolves a list of references using the ref-resolver API with region failover
func (infoSvc *CloudInfoService) ResolveReferences(region string, references []Reference) (*ResolveResponse, error) {
	// Check if we have an active region from previous failover
	infoSvc.refResolverLock.Lock()
	activeRegion := infoSvc.activeRefResolverRegion
	infoSvc.refResolverLock.Unlock()

	// If we have an active region from previous failover, use it instead of the requested region
	if activeRegion != "" && activeRegion != region {
		if infoSvc.Logger != nil {
			infoSvc.Logger.ShortInfo(fmt.Sprintf("Using active region %s instead of requested region %s (from previous successful failover)", activeRegion, region))
		}
		region = activeRegion
	}

	// For dev/test/staging environments, use traditional retry logic
	if isDevTestStagingRegion(region) {
		return infoSvc.resolveReferencesWithRetry(region, references, defaultRetryCount, false)
	}

	// For production regions, try once then failover to alternative regions
	return infoSvc.resolveReferencesWithRegionFailover(region, references)
}

// resolveReferencesWithRetry implements the actual reference resolution with retry logic
func (infoSvc *CloudInfoService) resolveReferencesWithRetry(region string, references []Reference, maxRetries int, hasMoreRegions bool) (*ResolveResponse, error) {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		isRetryAttempt := attempt < maxRetries                      // True for all but the last attempt
		hasMoreAttempts := (attempt < maxRetries) || hasMoreRegions // True if more retries OR more regions available
		result, err := infoSvc.doResolveReferencesWithContext(region, references, isRetryAttempt, hasMoreAttempts)

		if err == nil {
			return result, nil
		}

		lastErr = err

		// Check if this is the last attempt
		if attempt == maxRetries {
			break
		}

		// Parse the error to check if it's retryable
		if httpErr, ok := err.(*HttpError); ok {
			if shouldRetryReferenceResolution(httpErr.StatusCode, httpErr.Body) {
				waitTime := time.Duration(defaultInitialRetryWait*(1<<attempt)) * time.Second // exponential backoff
				if infoSvc.Logger != nil {
					infoSvc.Logger.ShortWarn(fmt.Sprintf("Reference resolution failed with retryable error (attempt %d/%d), retrying in %v...",
						attempt+1, maxRetries+1, waitTime))

					// Enhanced debugging for different types of failures
					if strings.Contains(httpErr.Body, "Failed to validate api key token") {
						infoSvc.Logger.ShortWarn("  → Detected API key validation failure (known intermittent issue)")
					} else if httpErr.StatusCode == 404 && strings.Contains(httpErr.Body, "could not be found") {
						infoSvc.Logger.ShortWarn("  → Detected project reference not found (timing issue)")
						infoSvc.Logger.ShortWarn("  → This occurs when checking project details too quickly after creation")
						infoSvc.Logger.ShortWarn("  → The resolver API needs time to be updated with new project information")
					} else if httpErr.StatusCode >= 500 {
						infoSvc.Logger.ShortWarn("  → Detected server error (potentially transient)")
					}
				}

				// Force token refresh for API key validation errors by creating a new authenticator instance
				// This addresses a known issue where cached tokens expire during retry attempts
				if infoSvc.authenticator != nil && strings.Contains(httpErr.Body, "Failed to validate api key token") {
					if infoSvc.Logger != nil {
						infoSvc.Logger.ShortInfo("Forcing token refresh due to API key validation failure")
					}
					// Create a new authenticator to force fresh token retrieval
					if iamAuth, ok := infoSvc.authenticator.(*core.IamAuthenticator); ok {
						newAuth := &core.IamAuthenticator{
							ApiKey: iamAuth.ApiKey,
						}
						infoSvc.authenticator = newAuth
						if infoSvc.Logger != nil {
							infoSvc.Logger.ShortInfo("  Created new IAM authenticator for fresh token")
						}
					}
				}

				time.Sleep(waitTime)
				continue
			} else if infoSvc.Logger != nil {
				// Only log non-retryable errors at error level during the final attempt
				// For earlier attempts, we still want to break out, but this helps with debugging
				infoSvc.Logger.ShortError(fmt.Sprintf("Reference resolution failed with non-retryable error: %v", err))
				infoSvc.Logger.ShortError(fmt.Sprintf("  HTTP Status: %d", httpErr.StatusCode))
				infoSvc.Logger.ShortError(fmt.Sprintf("  Response body: %s", httpErr.Body))
			}
		} else if infoSvc.Logger != nil {
			infoSvc.Logger.ShortError(fmt.Sprintf("Reference resolution failed with non-HTTP error: %v", err))
		}

		// If it's not a retryable error, return immediately
		break
	}

	// Check if the final error is a retryable error and enhance the error message
	if httpErr, ok := lastErr.(*HttpError); ok {
		if shouldRetryReferenceResolution(httpErr.StatusCode, httpErr.Body) {
			var enhancedMessage string
			if strings.Contains(httpErr.Body, "Failed to validate api key token") {
				enhancedMessage = "This is a known intermittent issue with IBM Cloud's reference resolution service. Please re-run the test as the issue is typically transient"
			} else if httpErr.StatusCode == 404 && strings.Contains(httpErr.Body, "could not be found") {
				enhancedMessage = "This is a known intermittent issue where project references temporarily cannot be found during provisioning. Please re-run the test as the issue is typically transient"
			} else {
				enhancedMessage = "This appears to be a transient service issue. Please re-run the test as the issue is typically transient"
			}

			return nil, &EnhancedHttpError{
				HttpError: httpErr,
				Message:   enhancedMessage,
			}
		}
	}

	// Log final failure at error level
	if infoSvc.Logger != nil {
		infoSvc.Logger.ShortError(fmt.Sprintf("Reference resolution failed after %d attempts: %v", maxRetries+1, lastErr))
		if httpErr, ok := lastErr.(*HttpError); ok {
			infoSvc.Logger.ShortError(fmt.Sprintf("  Final HTTP Status: %d", httpErr.StatusCode))
			infoSvc.Logger.ShortError(fmt.Sprintf("  Final Response body: %s", httpErr.Body))
		}
	}

	return nil, lastErr
}

// resolveReferencesWithRegionFailover implements region failover for production environments
func (infoSvc *CloudInfoService) resolveReferencesWithRegionFailover(primaryRegion string, references []Reference) (*ResolveResponse, error) {
	var lastErr error
	var attempts []string

	// Try primary region first with one retry for API key validation failures
	if infoSvc.Logger != nil {
		infoSvc.Logger.ShortInfo(fmt.Sprintf("Attempting reference resolution in primary region: %s", primaryRegion))
	}

	// Get preferred fallback regions to determine if more regions are available
	fallbackRegions := getPreferredFallbackRegions(primaryRegion)
	hasMoreRegions := len(fallbackRegions) > 0

	result, err := infoSvc.resolveReferencesWithRetry(primaryRegion, references, 1, hasMoreRegions)
	if err == nil {
		return result, nil
	}

	lastErr = err
	attempts = append(attempts, primaryRegion)

	if infoSvc.Logger != nil {
		infoSvc.Logger.ShortWarn(fmt.Sprintf("Primary region %s failed, trying fallback regions...", primaryRegion))
	}

	// Try each fallback region once
	for i, fallbackRegion := range fallbackRegions {
		if infoSvc.Logger != nil {
			infoSvc.Logger.ShortInfo(fmt.Sprintf("Attempting reference resolution in fallback region: %s", fallbackRegion))
		}

		// Check if this is the last fallback region
		hasMoreAttempts := i < len(fallbackRegions)-1
		result, err := infoSvc.doResolveReferencesWithContext(fallbackRegion, references, false, hasMoreAttempts)
		if err == nil {
			if infoSvc.Logger != nil {
				infoSvc.Logger.ShortInfo(fmt.Sprintf("Reference resolution successful in fallback region: %s", fallbackRegion))
			}

			// Set the successful fallback region as the active region for future requests
			infoSvc.refResolverLock.Lock()
			infoSvc.activeRefResolverRegion = fallbackRegion
			infoSvc.refResolverLock.Unlock()

			if infoSvc.Logger != nil {
				infoSvc.Logger.ShortInfo(fmt.Sprintf("Set active ref-resolver region to %s for subsequent requests", fallbackRegion))
			}

			return result, nil
		}

		lastErr = err
		attempts = append(attempts, fallbackRegion)

		if infoSvc.Logger != nil {
			infoSvc.Logger.ShortWarn(fmt.Sprintf("Fallback region %s failed: %v", fallbackRegion, err))
		}
	}

	// All regions failed, return enhanced error with attempt details
	if infoSvc.Logger != nil {
		infoSvc.Logger.ShortError(fmt.Sprintf("Reference resolution failed in all attempted regions: %v", attempts))
	}

	// Check if the final error is an API key validation failure and enhance the error message
	if httpErr, ok := lastErr.(*HttpError); ok {
		if shouldRetryReferenceResolution(httpErr.StatusCode, httpErr.Body) {
			return nil, &EnhancedHttpError{
				HttpError: httpErr,
				Message:   fmt.Sprintf("This is a known intermittent issue with IBM Cloud's reference resolution service. Attempted regions: %v. Please re-run the test as the issue is typically transient", attempts),
			}
		}
	}

	return nil, fmt.Errorf("reference resolution failed in all regions %v: %w", attempts, lastErr)
}

// HttpError represents an HTTP error with status code and body
type HttpError struct {
	StatusCode int
	Body       string
}

func (e *HttpError) Error() string {
	return fmt.Sprintf("invalid status code: %d, body: %s", e.StatusCode, e.Body)
}

// EnhancedHttpError wraps HttpError with additional context for intermittent issues
type EnhancedHttpError struct {
	*HttpError
	Message string
}

func (e *EnhancedHttpError) Error() string {
	return fmt.Sprintf("%s. %s", e.HttpError.Error(), e.Message)
}

func (e *EnhancedHttpError) Unwrap() error {
	return e.HttpError
}

// doResolveReferences performs the actual reference resolution without retry logic
func (infoSvc *CloudInfoService) doResolveReferences(region string, references []Reference) (*ResolveResponse, error) {
	return infoSvc.doResolveReferencesWithContext(region, references, false, false)
}

// doResolveReferencesWithContext performs the actual reference resolution with context for logging level
func (infoSvc *CloudInfoService) doResolveReferencesWithContext(region string, references []Reference, isRetryAttempt bool, hasMoreAttempts bool) (*ResolveResponse, error) {
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

	// Basic request logging
	if infoSvc.Logger != nil {
		infoSvc.Logger.ShortInfo(fmt.Sprintf("Resolving %d references for region %s", len(references), region))
	}

	// Create a new request
	req, err := http.NewRequest("POST", serviceURL+"/resolve", bytes.NewReader(jsonPayload))
	if err != nil {
		if infoSvc.Logger != nil {
			infoSvc.Logger.ShortError(fmt.Sprintf("Failed to create HTTP request: %v", err))
			infoSvc.Logger.ShortError(fmt.Sprintf("  Service URL: %s", serviceURL))
			infoSvc.Logger.ShortError(fmt.Sprintf("  Payload size: %d bytes", len(jsonPayload)))
		}
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set the headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Set the authentication token
	token, err := infoSvc.authenticator.GetToken()
	if err != nil {
		if infoSvc.Logger != nil {
			infoSvc.Logger.ShortError(fmt.Sprintf("Failed to get authentication token: %v", err))
		}
		return nil, fmt.Errorf("failed to get token: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	// Send the request using the global HTTP client that can be overridden in tests
	resp, err := CloudInfo_HttpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			if closeErr := resp.Body.Close(); closeErr != nil {
				if infoSvc.Logger != nil {
					infoSvc.Logger.ShortInfo(fmt.Sprintf("Error closing response body: %v", closeErr))
				}
			}
		}
	}()

	// Check the status code
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyStr := string(bodyBytes)

		// Enhanced error logging with context-aware log levels
		if infoSvc.Logger != nil {
			// Determine if this error should be retried
			isRetryableError := shouldRetryReferenceResolution(resp.StatusCode, bodyStr)

			// Use warning level for retryable errors during retry attempts, error level for final attempts or non-retryable errors
			logLevel := "error"
			if (isRetryAttempt && isRetryableError) || hasMoreAttempts {
				logLevel = "warn"
			}

			if logLevel == "warn" {
				// For warnings, show brief one-line message based on error type
				if resp.StatusCode == 500 && strings.Contains(bodyStr, "Failed to validate api key token") {
					infoSvc.Logger.ShortWarn(fmt.Sprintf("Reference resolution failed (HTTP %d), will retry - API key validation issue", resp.StatusCode))
				} else if resp.StatusCode == 404 && strings.Contains(bodyStr, "could not be found") {
					infoSvc.Logger.ShortWarn(fmt.Sprintf("Reference resolution failed (HTTP %d), will retry - project reference not found", resp.StatusCode))
				} else {
					infoSvc.Logger.ShortWarn(fmt.Sprintf("Reference resolution failed (HTTP %d), will retry - transient error", resp.StatusCode))
				}
			} else {
				infoSvc.Logger.ShortError("===== Reference Resolution Failed =====")
				infoSvc.Logger.ShortError(fmt.Sprintf("HTTP Status: %d", resp.StatusCode))
				infoSvc.Logger.ShortError(fmt.Sprintf("Service URL: %s", serviceURL))
				infoSvc.Logger.ShortError(fmt.Sprintf("Region: %s", region))
				infoSvc.Logger.ShortError(fmt.Sprintf("Number of references: %d", len(references)))
			}

			// For final errors, show detailed diagnostics
			if logLevel == "error" {
				// Log all references being resolved
				infoSvc.Logger.ShortError("References being resolved:")
				for i, ref := range references {
					infoSvc.Logger.ShortError(fmt.Sprintf("  [%d] %s", i+1, ref.Reference))
				}

				// Log authentication details (masked for security)
				if len(token) > 10 {
					maskedToken := token[:6] + "..." + token[len(token)-4:]
					infoSvc.Logger.ShortError(fmt.Sprintf("Token: %s", maskedToken))
				}

				// Log API key metadata for debugging
				if iamAuth, ok := infoSvc.authenticator.(*core.IamAuthenticator); ok {
					if len(iamAuth.ApiKey) > 0 {
						infoSvc.Logger.ShortError(fmt.Sprintf("API key length: %d characters", len(iamAuth.ApiKey)))
						infoSvc.Logger.ShortError(fmt.Sprintf("API key prefix: %s...", iamAuth.ApiKey[:min(6, len(iamAuth.ApiKey))]))
					}
				}

				// Log response details
				infoSvc.Logger.ShortError(fmt.Sprintf("Response body: %s", bodyStr))
				infoSvc.Logger.ShortError(fmt.Sprintf("Response headers: %v", resp.Header))
			}

			// Special handling for specific known intermittent issues - only show detailed explanations for final errors
			if logLevel == "error" {
				if resp.StatusCode == 500 && strings.Contains(bodyStr, "Failed to validate api key token") {
					infoSvc.Logger.ShortError("========================================")
					infoSvc.Logger.ShortError("API KEY VALIDATION FAILURE DETECTED")
					infoSvc.Logger.ShortError("This is a known intermittent issue with IBM Cloud's reference resolution service.")
					infoSvc.Logger.ShortError("The API key is valid (or tests wouldn't have reached this point).")
					infoSvc.Logger.ShortError("Re-running the test typically resolves this transient issue.")
					infoSvc.Logger.ShortError("========================================")
				} else if resp.StatusCode == 404 && strings.Contains(bodyStr, "could not be found") {
					infoSvc.Logger.ShortError("========================================")
					infoSvc.Logger.ShortError("PROJECT REFERENCE NOT FOUND DETECTED")
					infoSvc.Logger.ShortError("This is a known intermittent issue where project references cannot be found during provisioning.")
					infoSvc.Logger.ShortError("This commonly occurs when project resources are still being set up.")
					infoSvc.Logger.ShortError("Re-running the test typically resolves this transient issue.")
					infoSvc.Logger.ShortError("========================================")
				}
			} else if resp.StatusCode == 429 && logLevel == "error" {
				infoSvc.Logger.ShortError("========================================")
				infoSvc.Logger.ShortError("RATE LIMITING DETECTED (429)")
				infoSvc.Logger.ShortError("This occurs when multiple parallel tests hit the same API endpoints simultaneously.")
				infoSvc.Logger.ShortError("Consider using StaggerDelay in your AddonTestMatrix to space out test starts.")
				infoSvc.Logger.ShortError("Example: matrix.StaggerDelay = testaddons.StaggerDelay(10 * time.Second) // 10 second stagger")
				infoSvc.Logger.ShortError("Re-running the test typically resolves this issue.")
				infoSvc.Logger.ShortError("========================================")
			}
		}

		return nil, &HttpError{
			StatusCode: resp.StatusCode,
			Body:       bodyStr,
		}
	}

	// Read the entire response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		if infoSvc.Logger != nil {
			infoSvc.Logger.ShortError(fmt.Sprintf("Failed to read response body: %v", err))
		}
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Success logging - minimal but informative
	if infoSvc.Logger != nil {
		var response ResolveResponse
		if err := json.Unmarshal(bodyBytes, &response); err == nil {
			infoSvc.Logger.ShortInfo(fmt.Sprintf("References resolved successfully (%d references, %d bytes)",
				len(response.References), len(bodyBytes)))
		} else {
			infoSvc.Logger.ShortInfo(fmt.Sprintf("References resolved successfully (%d bytes)", len(bodyBytes)))
		}
	}

	// Parse the response
	var response ResolveResponse
	err = json.Unmarshal(bodyBytes, &response)
	if err != nil {
		if infoSvc.Logger != nil {
			infoSvc.Logger.ShortError(fmt.Sprintf("Failed to parse JSON response: %v", err))
			infoSvc.Logger.ShortError(fmt.Sprintf("Raw response: %s", string(bodyBytes)))
		}
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}

// getProjectInfoFromNameOrID resolves a project ID or name to a ProjectInfo
func (infoSvc *CloudInfoService) getProjectInfoFromID(projectID string, projectCache map[string]*ProjectInfo) (*ProjectInfo, error) {
	// Since only project IDs are supported, directly get the project info
	return infoSvc.GetProjectInfo(projectID, projectCache)
}

// transformStackStyleReference handles stack-style relative references like ref:../../configs/{configID}/inputs/prefix
func (infoSvc *CloudInfoService) transformStackStyleReference(normalizedRef, encodedProjectName string, projectInfo *ProjectInfo) (string, error) {
	// Extract the path after "ref:"
	refPath := strings.TrimPrefix(normalizedRef, "ref:")

	// Check if this is a stack-style reference with configs
	if strings.Contains(refPath, "/configs/") {
		// Pattern: ../../configs/{configID}/inputs/prefix or ../../members/{memberName}/...

		// Find the configs part
		configsIndex := strings.Index(refPath, "/configs/")
		if configsIndex == -1 {
			return "", fmt.Errorf("no /configs/ found in reference path")
		}

		// Extract everything after /configs/
		configPath := refPath[configsIndex+9:] // +9 for "/configs/"

		// Split to get configID and the rest of the path
		parts := strings.SplitN(configPath, "/", 2)
		if len(parts) < 1 {
			return "", fmt.Errorf("invalid config path format")
		}

		configID := parts[0]
		var resourcePath string
		if len(parts) > 1 {
			resourcePath = "/" + parts[1]
		}

		// Try to resolve config ID to name
		configName := configID // default fallback
		if name, exists := projectInfo.Configs[configID]; exists {
			configName = name
		} else if isUUID(configID) {
			// Try to fetch the config details if it looks like a UUID
			name, err := infoSvc.GetConfigName(projectInfo.ID, configID)
			if err == nil && name != "" {
				configName = name
				projectInfo.Configs[configID] = name
			}
		}

		// Create qualified reference
		qualifiedRef := fmt.Sprintf("ref://project.%s/configs/%s%s", encodedProjectName, configName, resourcePath)
		return qualifiedRef, nil
	}

	// Handle other stack-style patterns like ../../members/{memberName}/outputs/{output}
	if strings.Contains(refPath, "/members/") {
		// Pattern: ../../members/{memberName}/outputs/{output}
		membersIndex := strings.Index(refPath, "/members/")
		if membersIndex == -1 {
			return "", fmt.Errorf("no /members/ found in reference path")
		}

		// Extract everything after /members/
		memberPath := refPath[membersIndex+9:] // +9 for "/members/"

		// Create qualified reference preserving the member path structure
		qualifiedRef := fmt.Sprintf("ref://project.%s/members/%s", encodedProjectName, memberPath)
		return qualifiedRef, nil
	}

	// For other stack-style references, strip the ../ parts and create a project-qualified reference
	// Remove leading ../ patterns
	cleanPath := refPath
	for strings.HasPrefix(cleanPath, "../") {
		cleanPath = strings.TrimPrefix(cleanPath, "../")
	}

	// Remove leading slash if present
	if strings.HasPrefix(cleanPath, "/") {
		cleanPath = strings.TrimPrefix(cleanPath, "/")
	}

	qualifiedRef := fmt.Sprintf("ref://project.%s/%s", encodedProjectName, cleanPath)
	return qualifiedRef, nil
}

// transformReferencesToQualifiedReferences transforms a slice of reference strings to fully qualified references
// This function contains the core reference transformation logic and can be tested independently
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

		// Handle stack-style relative references (e.g., ref:../../configs/{configID}/inputs/prefix)
		if strings.Contains(normalizedRef, "../") {
			qualifiedRef, err := infoSvc.transformStackStyleReference(normalizedRef, encodedProjectName, projectInfo)
			if err != nil {
				if infoSvc.Logger != nil {
					infoSvc.Logger.ShortWarn(fmt.Sprintf("Failed to transform stack-style reference %s: %v", normalizedRef, err))
				}
				// Fall back to basic project qualification
				qualifiedRef = fmt.Sprintf("ref://project.%s/%s", encodedProjectName, strings.TrimPrefix(normalizedRef, "ref:"))
			}
			references = append(references, Reference{
				Reference: qualifiedRef,
			})
			continue
		}

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

	if infoSvc.Logger != nil {
		infoSvc.Logger.ShortInfo(fmt.Sprintf("Starting reference resolution for project: %s (%d references)", projectID, len(refStrings)))
	}

	// Get project info first
	projectInfo, err := infoSvc.getProjectInfoFromID(projectID, projectCache)
	if err != nil {
		if infoSvc.Logger != nil {
			infoSvc.Logger.ShortError(fmt.Sprintf("Failed to get project info for project ID %s: %v", projectID, err))
		}
		return nil, fmt.Errorf("failed to get project info: %v", err)
	}

	if infoSvc.Logger != nil {
		infoSvc.Logger.ShortInfo(fmt.Sprintf("Using project: %s (ID: %s)", projectInfo.Name, projectInfo.ID))
	}

	references, err := infoSvc.transformReferencesToQualifiedReferences(refStrings, projectInfo)
	if err != nil {
		if infoSvc.Logger != nil {
			infoSvc.Logger.ShortError(fmt.Sprintf("Failed to transform references: %v", err))
		}
		return nil, fmt.Errorf("failed to transform references: %v", err)
	}

	return infoSvc.ResolveReferences(region, references)
}
