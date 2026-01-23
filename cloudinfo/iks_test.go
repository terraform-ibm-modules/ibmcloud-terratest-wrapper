package cloudinfo

import (
	"testing"

	"github.com/IBM-Cloud/bluemix-go/api/container/containerv1"
	"github.com/stretchr/testify/assert"
)

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
