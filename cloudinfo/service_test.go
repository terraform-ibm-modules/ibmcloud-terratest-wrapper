package cloudinfo

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewServiceWithKey(t *testing.T) {
	serviceOptions := CloudInfoServiceOptions{
		ApiKey:                    "dummy_key",
		VpcService:                new(vpcServiceMock),
		IamIdentityService:        new(iamIdentityServiceMock),
		ResourceControllerService: new(resourceControllerServiceMock),
		ContainerClient:           new(containerClientMock),
	}

	_, err := NewCloudInfoServiceWithKey(serviceOptions)

	require.Nil(t, err, "Error returned getting new service")
}

func TestNewServiceWithEnv(t *testing.T) {
	serviceOptions := CloudInfoServiceOptions{
		VpcService:                new(vpcServiceMock),
		IamIdentityService:        new(iamIdentityServiceMock),
		ResourceControllerService: new(resourceControllerServiceMock),
		ContainerClient:           new(containerClientMock),
	}

	os.Setenv("TEST_KEY_VAL", "dummy_key")
	_, err := NewCloudInfoServiceFromEnv("TEST_KEY_VAL", serviceOptions)

	require.Nil(t, err, "Error returned getting new service")

}

func TestNewServiceWithEmptyKey(t *testing.T) {
	serviceOptions := CloudInfoServiceOptions{
		VpcService:                new(vpcServiceMock),
		IamIdentityService:        new(iamIdentityServiceMock),
		ResourceControllerService: new(resourceControllerServiceMock),
		ContainerClient:           new(containerClientMock),
	}

	_, err := NewCloudInfoServiceWithKey(serviceOptions)

	require.NotNil(t, err, "Empty key should have resulted in error")
}

func TestNewServiceWithEmptyEnv(t *testing.T) {
	serviceOptions := CloudInfoServiceOptions{
		VpcService:                new(vpcServiceMock),
		IamIdentityService:        new(iamIdentityServiceMock),
		ResourceControllerService: new(resourceControllerServiceMock),
		ContainerClient:           new(containerClientMock),
	}

	_, err := NewCloudInfoServiceFromEnv("", serviceOptions)

	require.NotNil(t, err, "Empty Environment key should have resulted in error")

}
