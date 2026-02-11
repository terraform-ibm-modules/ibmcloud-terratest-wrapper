package cloudinfo

import (
	"fmt"
	"testing"

	"github.com/IBM-Cloud/bluemix-go/api/container/containerv1"
	"github.com/IBM-Cloud/bluemix-go/api/container/containerv2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetClusterConfigPath(t *testing.T) {
	mockClusterId := "test-cluster"
	mockBasePath := "."
	mockEndpoint := "public"
	mockFilePath := "/path/to/config"
	mockError := fmt.Errorf("error getting cluster config")

	testCases := []struct {
		name               string
		admin              bool
		createCalicoConfig bool
		expectedError      error
		mockError          error
		expectedFilePath   string
	}{
		{
			name:               "Success case",
			admin:              false,
			createCalicoConfig: false,
			expectedError:      nil,
			mockError:          nil,
			expectedFilePath:   mockFilePath,
		},
		{
			name:               "Failure case",
			admin:              false,
			createCalicoConfig: false,
			expectedError:      fmt.Errorf("failed to get cluster config details: %s", mockError),
			mockError:          mockError,
			expectedFilePath:   "",
		},
		// Add more cases as needed
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockContainerClient := &containerClientMock{}
			mockClusters := &ClustersMock{}

			mockContainerClient.On("Clusters").Return(mockClusters)
			mockClusters.On("StoreConfigDetail", mockClusterId, mockBasePath, tc.admin, tc.createCalicoConfig, mock.AnythingOfType("containerv2.ClusterTargetHeader"), mockEndpoint).Return(mockFilePath, containerv1.ClusterKeyInfo{FilePath: tc.expectedFilePath}, tc.mockError)

			infoSvc := CloudInfoService{containerClient: mockContainerClient}

			filePath, err := infoSvc.GetClusterConfigPath(mockClusterId, mockBasePath, tc.admin, tc.createCalicoConfig, mockEndpoint)

			if tc.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tc.expectedError, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedFilePath, filePath)
			}

			mockClusters.AssertExpectations(t)
			mockContainerClient.AssertExpectations(t)
		})
	}
}

func TestGetClusterIngressStatus(t *testing.T) {
	// Mock data
	mockClusterId := "test-cluster"
	mockError := fmt.Errorf("error getting cluster ingress status")

	// Define test cases
	testCases := []struct {
		name           string
		expectedError  error
		mockError      error
		expectedStatus string
	}{
		{
			name:           "Success case 1",
			expectedError:  nil,
			mockError:      nil,
			expectedStatus: "healthy",
		},
		{
			name:           "Success case 2",
			expectedError:  nil,
			mockError:      nil,
			expectedStatus: "critical",
		},
		{
			name:           "Failure case",
			expectedError:  fmt.Errorf("failed to get cluster ingress status: %s", mockError),
			mockError:      mockError,
			expectedStatus: "",
		},
		// Add more cases as needed
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create mock objects
			mockContainerClient := &containerClientMock{}
			mockAlbs := &AlbsMock{}

			// Setup mock method responses
			mockContainerClient.On("Albs").Return(mockAlbs)
			mockAlbs.On("GetIngressStatus", mockClusterId, mock.AnythingOfType("containerv2.ClusterTargetHeader")).Return(containerv2.IngressStatus{Status: tc.expectedStatus}, tc.mockError)

			// Initialize service with mock container client
			infoSvc := CloudInfoService{containerClient: mockContainerClient}

			// Call method under test
			status, err := infoSvc.GetClusterIngressStatus(mockClusterId)

			// Assertions
			if tc.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tc.expectedError, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedStatus, status)
			}

			// Verify expectations
			mockAlbs.AssertExpectations(t)
			mockContainerClient.AssertExpectations(t)

		})
	}

}

// TestGetKubeVersions validates that GetKubeVersions returns the correct
// major.minor version strings for supported platforms and returns an
// error for unsupported platforms.
func TestGetKubeVersions(t *testing.T) {
	mockData := containerv1.V1Version{
		"kubernetes": []containerv1.KubeVersion{
			{Major: 1, Minor: 31, Patch: 14, Default: false},
			{Major: 1, Minor: 33, Patch: 6, Default: true},
		},
		"openshift": []containerv1.KubeVersion{
			{Major: 4, Minor: 16, Patch: 52, Default: false},
			{Major: 4, Minor: 19, Patch: 19, Default: true},
		},
	}

	tests := []struct {
		name        string   // Descriptive name of the test case
		platform    string   // Platform passed to GetKubeVersions
		expected    []string // Expected major.minor versions
		expectError bool     // Indicates whether an error is expected
	}{
		{
			name:     "openshift platform",
			platform: "openshift",
			expected: []string{"4.16", "4.19"},
		},
		{
			name:     "kubernetes platform",
			platform: "kubernetes",
			expected: []string{"1.31", "1.33"},
		},
		{
			name:        "invalid platform",
			platform:    "invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockContainerV1Client := &containerV1ClientMock{}
			mockKubeVersions := &KubeVersionsMock{}

			mockContainerV1Client.On("KubeVersions").Return(mockKubeVersions)
			mockKubeVersions.
				On("ListV1", containerv1.ClusterTargetHeader{}).
				Return(mockData, nil)

			infoSvc := CloudInfoService{
				containerV1Client: mockContainerV1Client,
			}

			versions, err := infoSvc.GetKubeVersions(tt.platform)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, versions)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, versions)
			}
		})
	}
}
