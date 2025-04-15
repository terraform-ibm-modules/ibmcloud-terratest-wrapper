package cloudinfo

import (
	"net/http"

	projects "github.com/IBM/project-go-sdk/projectv1"
)

// MockTransport is a custom http.RoundTripper for testing
type MockTransport struct {
	token     string
	transport http.RoundTripper
}

// RoundTrip implements the http.RoundTripper interface
func (t *MockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Add authorization header if not already present
	if req.Header.Get("Authorization") == "" {
		req.Header.Set("Authorization", "Bearer "+t.token)
	}
	// Use the underlying transport for the actual HTTP request
	return t.transport.RoundTrip(req)
}

// MockCloudInfoService is a mock implementation of CloudInfoServiceI for testing
type MockCloudInfoService struct {
	CloudInfoService
	MockResolveReferences            func(region string, references []Reference) (*ResolveResponse, error)
	MockGetProjectByName             func(projectName string) (*projects.Project, error)
	MockResolveReferencesFromStrings func(region string, refStrings []string, projectID string) (*ResolveResponse, error)
}

// ResolveReferences overrides the implementation in CloudInfoService to use our mock function
func (m *MockCloudInfoService) ResolveReferences(region string, references []Reference) (*ResolveResponse, error) {
	if m.MockResolveReferences != nil {
		return m.MockResolveReferences(region, references)
	}
	return m.CloudInfoService.ResolveReferences(region, references)
}

// GetProjectByName overrides the implementation in CloudInfoService to use our mock function
func (m *MockCloudInfoService) GetProjectByName(projectName string) (*projects.Project, error) {
	if m.MockGetProjectByName != nil {
		return m.MockGetProjectByName(projectName)
	}
	return m.CloudInfoService.GetProjectByName(projectName)
}

// ResolveReferencesFromStrings overrides the implementation in CloudInfoService to use our mock function
func (m *MockCloudInfoService) ResolveReferencesFromStrings(region string, refStrings []string, projectID string) (*ResolveResponse, error) {
	if m.MockResolveReferencesFromStrings != nil {
		return m.MockResolveReferencesFromStrings(region, refStrings, projectID)
	}
	return m.CloudInfoService.ResolveReferencesFromStrings(region, refStrings, projectID)
}

// MockProjectInfoProvider is a mock implementation that provides a custom getProjectInfoFromID method
type MockProjectInfoProvider struct {
	MockCloudInfoService
	MockGetProjectInfoFromID func(projectID string, projectCache map[string]*ProjectInfo) (*ProjectInfo, error)
}

// getProjectInfoFromID overrides the implementation to use our mock function
func (m *MockProjectInfoProvider) getProjectInfoFromID(projectID string, projectCache map[string]*ProjectInfo) (*ProjectInfo, error) {
	if m.MockGetProjectInfoFromID != nil {
		return m.MockGetProjectInfoFromID(projectID, projectCache)
	}
	return m.CloudInfoService.getProjectInfoFromID(projectID, projectCache)
}
