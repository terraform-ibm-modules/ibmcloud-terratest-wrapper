package cloudinfo

import (
	"testing"

	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/stretchr/testify/assert"
)

func TestGetAvailableIcdVersions(t *testing.T) {

	infoSvc := CloudInfoService{
		authenticator: &core.IamAuthenticator{
			ApiKey: "dummy_key", // pragma: allowlist secret
		},
	}

	t.Run("Valid icdType", func(t *testing.T) {
		got, err := infoSvc.GetAvailableIcdVersions("mongodb")
		assert.NoError(t, err)
		assert.Equal(t, []string{"5.0", "4.4"}, got)
	})

	t.Run("Invalid icdType", func(t *testing.T) {
		got, err := infoSvc.GetAvailableIcdVersions("invalidType")
		assert.Error(t, err)
		assert.NotNil(t, got)
	})
}
