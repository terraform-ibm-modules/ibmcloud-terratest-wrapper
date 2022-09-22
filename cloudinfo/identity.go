package cloudinfo

import (
	"github.com/IBM/platform-services-go-sdk/iamidentityv1"
)

// getApiKeyDetail (private) will return ApiKey detail object from infoSvc if it exists already.
// If not this method will retrieve ApiKey detail object from the iamidentityv1 API and set in infoSvc for future calls.
// NOTE: the API key used for lookup will be extracted from the current service authenticator.
func (infoSvc *CloudInfoService) getApiKeyDetail() (*iamidentityv1.APIKey, error) {
	// if APIKey has been retrieved already, simply return
	if infoSvc.apiKeyDetail != nil {
		return infoSvc.apiKeyDetail, nil
	} else {
		// retrieve API key from current authentication and return/set
		apiKey, _, err := infoSvc.iamIdentityService.GetAPIKeysDetails(&iamidentityv1.GetAPIKeysDetailsOptions{
			IamAPIKey: &infoSvc.authenticator.ApiKey, // pragma: allowlist secret
		})
		if err != nil {
			return nil, err
		}

		// set for future calls AND return
		infoSvc.apiKeyDetail = apiKey
		return apiKey, nil
	}
}
