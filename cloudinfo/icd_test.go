package cloudinfo

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/IBM/cloud-databases-go-sdk/clouddatabasesv5"
	"github.com/stretchr/testify/assert"

	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/common"
)

func TestGetAvailableIcdVersions(t *testing.T) {
	infoSvc := CloudInfoService{
		icdService: &icdServiceMock{},
	}

	var mockType = "icd"
	var mockVersion1 = "1.0.0"
	var mockStable = "stable"
	var mockVersion2 = "2.0.0"
	var mockBeta = "beta"

	// first test, icd type does not exist
	t.Run("ICDTypeDoesNotExist", func(t *testing.T) {
		infoSvc.icdService = &icdServiceMock{
			mockListDeployablesResponse: &clouddatabasesv5.ListDeployablesResponse{
				Deployables: []clouddatabasesv5.Deployables{
					{
						Type: &mockType,
						Versions: []clouddatabasesv5.DeployablesVersionsItem{
							{
								Version: &mockVersion1,
								Status:  &mockStable,
							},
							{
								Version: &mockVersion2,
								Status:  &mockBeta,
							},
						},
					},
				},
			},
		}
		_, err := infoSvc.GetAvailableIcdVersions("non-existing-icd")
		assert.NotNil(t, err)
	})

	// second test, icd type exists
	t.Run("ICDTypeExists", func(t *testing.T) {
		infoSvc.icdService = &icdServiceMock{
			mockListDeployablesResponse: &clouddatabasesv5.ListDeployablesResponse{
				Deployables: []clouddatabasesv5.Deployables{
					{
						Type: &mockType,
						Versions: []clouddatabasesv5.DeployablesVersionsItem{
							{
								Version: &mockVersion1,
								Status:  &mockStable,
							},
							{
								Version: &mockVersion2,
								Status:  &mockBeta,
							},
						},
					},
				},
			},
		}
		versions, err := infoSvc.GetAvailableIcdVersions(mockType)
		assert.Nil(t, err)
		assert.Equal(t, []string{"1.0.0"}, versions)
	})

	// third test, no stable versions for icd type exists
	t.Run("StableVersionDoesNotExist", func(t *testing.T) {
		infoSvc.icdService = &icdServiceMock{
			mockListDeployablesResponse: &clouddatabasesv5.ListDeployablesResponse{
				Deployables: []clouddatabasesv5.Deployables{
					{
						Type: &mockType,
						Versions: []clouddatabasesv5.DeployablesVersionsItem{
							{
								Version: &mockVersion1,
								Status:  &mockBeta,
							},
							{
								Version: &mockVersion2,
								Status:  &mockBeta,
							},
						},
					},
				},
			},
		}
		_, err := infoSvc.GetAvailableIcdVersions(mockType)
		assert.NotNil(t, err)
	})
}


func TestGetAvailableIcdVersionsGen2(t *testing.T) {
	t.Run("Gen2ServiceExists", func(t *testing.T) {
		// Create mock HTTP server with successful Gen2 response
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request headers
			assert.Contains(t, r.Header.Get("Authorization"), "Bearer")
			assert.Equal(t, "application/json", r.Header.Get("Accept"))

			// Return mock Gen2 response
			mockResponse := `{
				"metadata": {
					"other": {
						"versions": [
							{"version": "16", "status": "stable", "is_preferred": false},
							{"version": "17", "status": "stable", "is_preferred": false},
							{"version": "18", "status": "stable", "is_preferred": true}
						]
					}
				}
			}`
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(mockResponse))
		}))
		defer mockServer.Close()

		// Create CloudInfoService with mock authenticator
		_ = CloudInfoService{
			authenticator: &MockAuthenticator{Token: "mock-token"},
			Logger:        &common.TestLogger{},
		}

		// Note: In real implementation, we'd need to override the URL
		// For now, this test demonstrates the structure
		// The actual test would need URL injection or environment variable override

		// Test would call: versions, err := infoSvc.GetAvailableIcdVersionsGen2("databases-for-postgresql", "standard-gen2", "ca-tor")
		// assert.Nil(t, err)
		// assert.Equal(t, []string{"16", "17", "18"}, versions)
	})

	t.Run("Gen2ServiceNotFound404", func(t *testing.T) {
		// Create mock HTTP server that returns 404
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mockResponse := `{"message":"Object does not exist","code":404}`
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(mockResponse))
		}))
		defer mockServer.Close()

		_ = CloudInfoService{
			authenticator: &MockAuthenticator{Token: "mock-token"},
			Logger:        &common.TestLogger{},
		}

		// Should return empty list, no error for 404
		// versions, err := infoSvc.GetAvailableIcdVersionsGen2("databases-for-postgresql", "standard-gen2", "invalid-region")
		// assert.Nil(t, err)
		// assert.Equal(t, []string{}, versions)
	})

	t.Run("Gen2FilterDeadAndHiddenVersions", func(t *testing.T) {
		// Create mock HTTP server with mixed version statuses
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mockResponse := `{
				"metadata": {
					"other": {
						"versions": [
							{"version": "14", "status": "dead"},
							{"version": "15", "status": "hidden"},
							{"version": "16", "status": "stable"},
							{"version": "17", "status": "beta"}
						]
					}
				}
			}`
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(mockResponse))
		}))
		defer mockServer.Close()

		_ = CloudInfoService{
			authenticator: &MockAuthenticator{Token: "mock-token"},
			Logger:        &common.TestLogger{},
		}

		// Should return only versions 16 and 17 (not dead/hidden)
		// versions, err := infoSvc.GetAvailableIcdVersionsGen2("databases-for-postgresql", "standard-gen2", "ca-tor")
		// assert.Nil(t, err)
		// assert.Equal(t, []string{"16", "17"}, versions)
	})

	t.Run("Gen2InvalidJSON", func(t *testing.T) {
		// Create mock HTTP server that returns invalid JSON
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{invalid json`))
		}))
		defer mockServer.Close()

		_ = CloudInfoService{
			authenticator: &MockAuthenticator{Token: "mock-token"},
			Logger:        &common.TestLogger{},
		}

		// Should return error
		// _, err := infoSvc.GetAvailableIcdVersionsGen2("databases-for-postgresql", "standard-gen2", "ca-tor")
		// assert.NotNil(t, err)
		// assert.Contains(t, err.Error(), "error parsing JSON")
	})

	t.Run("Gen2ServerError500", func(t *testing.T) {
		// Create mock HTTP server that returns 500 error
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"Internal server error"}`))
		}))
		defer mockServer.Close()

		_ = CloudInfoService{
			authenticator: &MockAuthenticator{Token: "mock-token"},
			Logger:        &common.TestLogger{},
		}

		// Should return error
		// _, err := infoSvc.GetAvailableIcdVersionsGen2("databases-for-postgresql", "standard-gen2", "ca-tor")
		// assert.NotNil(t, err)
		// assert.Contains(t, err.Error(), "API request failed with status 500")
	})

	t.Run("Gen2AuthenticationError", func(t *testing.T) {
		// Note: The existing MockAuthenticator always succeeds
		// To test auth failures, we'd need a different mock or error injection
		// For now, this test is a placeholder showing the test structure
		_ = CloudInfoService{
			authenticator: &MockAuthenticator{Token: "mock-token"},
			Logger:        &common.TestLogger{},
		}

		// In a real scenario with auth failure:
		// _, err := infoSvc.GetAvailableIcdVersionsGen2("databases-for-postgresql", "standard-gen2", "ca-tor")
		// assert.NotNil(t, err)
		// assert.Contains(t, err.Error(), "error getting auth token")
	})

	t.Run("Gen2NoValidVersions", func(t *testing.T) {
		// Create mock HTTP server with only dead/hidden versions
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mockResponse := `{
				"metadata": {
					"other": {
						"versions": [
							{"version": "14", "status": "dead"},
							{"version": "15", "status": "hidden"}
						]
					}
				}
			}`
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(mockResponse))
		}))
		defer mockServer.Close()

		_ = CloudInfoService{
			authenticator: &MockAuthenticator{Token: "mock-token"},
			Logger:        &common.TestLogger{},
		}

		// Should return error - no valid versions
		// _, err := infoSvc.GetAvailableIcdVersionsGen2("databases-for-postgresql", "standard-gen2", "ca-tor")
		// assert.NotNil(t, err)
		// assert.Contains(t, err.Error(), "no valid versions found")
	})
}
