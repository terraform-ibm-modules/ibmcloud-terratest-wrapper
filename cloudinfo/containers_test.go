package cloudinfo

import (
	"fmt"
	"github.com/IBM-Cloud/bluemix-go/api/container/containerv1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
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
