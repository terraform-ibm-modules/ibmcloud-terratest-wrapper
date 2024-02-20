package cloudinfo

import (
	"testing"

	"github.com/IBM/cloud-databases-go-sdk/clouddatabasesv5"
	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/stretchr/testify/assert"
)

func TestGetAvailableIcdVersions(t *testing.T) {

	mockType := "icd"
	mockVersion1 := "1.0.0"
	mackStable := "stable"
	mockVersion2 := "2.0.0"
	mackBeta := "stable"

	// Create a mock ListDeployables method to return a pre-defined response
	infoSvc := CloudInfoService{
		authenticator: &core.IamAuthenticator{},
	}
	infoSvc.ListDeployablesResponse = &clouddatabasesv5.ListDeployablesResponse{
		Deployables: []clouddatabasesv5.Deployables{
			{
				Type: &mockType,
				Versions: []clouddatabasesv5.DeployablesVersionsItem{
					{
						Version: &mockVersion1,
						Status:  &mackStable,
					},
					{
						Version: &mockVersion2,
						Status:  &mackBeta,
					},
				},
			},
		},
	}

	// Call the GetAvailableIcdVersions method with a valid input and expect it to return the correct response
	expectedResponse := []string{"1.0.0", "2.0.0"}
	actualResponse, err := infoSvc.GetVersionsList("icd")
	assert.NoError(t, err)
	assert.EqualValues(t, expectedResponse, actualResponse)
}
