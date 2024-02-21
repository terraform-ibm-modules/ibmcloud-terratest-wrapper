package cloudinfo

import (
	"testing"

	"github.com/IBM/cloud-databases-go-sdk/clouddatabasesv5"
	"github.com/stretchr/testify/assert"
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
