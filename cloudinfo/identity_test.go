package cloudinfo

import (
	"testing"

	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/platform-services-go-sdk/iamidentityv1"
	"github.com/stretchr/testify/assert"
)

func TestApiKeyDetail(t *testing.T) {

	infoSvc := CloudInfoService{
		iamIdentityService: &iamIdentityServiceMock{},
		authenticator:      &core.IamAuthenticator{},
	}

	// first test, if keydetail exists already, return it without going further
	t.Run("UseExistingKeyDetail", func(t *testing.T) {
		existingKeyName := "EXISTING_API_KEY"
		infoSvc.apiKeyDetail = &iamidentityv1.APIKey{Name: &existingKeyName}
		existKey, existErr := infoSvc.getApiKeyDetail()
		assert.Nil(t, existErr)
		assert.Equal(t, *existKey.Name, existingKeyName)
		infoSvc.apiKeyDetail = nil // reset
	})

	// second test, if IAM service returns error, this does as well
	t.Run("IAMServiceError", func(t *testing.T) {
		infoSvc.authenticator.ApiKey = "ERROR"
		errorKey, errorErr := infoSvc.getApiKeyDetail()
		assert.NotNil(t, errorErr)
		assert.Nil(t, errorKey)
	})

	// third test, success and valid return key
	t.Run("NewKeyDetail", func(t *testing.T) {
		infoSvc.authenticator.ApiKey = "VALID_KEY"
		validKey, validErr := infoSvc.getApiKeyDetail()
		assert.Nil(t, validErr)
		assert.NotNil(t, validKey)
		assert.Equal(t, *validKey.ID, "MOCK_ID")
	})
}
