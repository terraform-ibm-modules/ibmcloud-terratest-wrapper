package cloudinfo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetAvailableIcdVersions(t *testing.T) {

	infoSvc, _ := NewCloudInfoServiceFromEnv(ibmcloudApiKeyVar, CloudInfoServiceOptions{})

	t.Run("Valid icdType", func(t *testing.T) {
		got, err := infoSvc.GetAvailableIcdVersions("mongodb")
		assert.NoError(t, err)
		assert.Equal(t, []string{"5.0", "4.4"}, got)
	})

	t.Run("Invalid icdType", func(t *testing.T) {
		got, _ := infoSvc.GetAvailableIcdVersions("invalidType")
		assert.NotNil(t, got)
	})
}
