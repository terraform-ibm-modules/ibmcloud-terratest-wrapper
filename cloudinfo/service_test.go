package cloudinfo

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServiceWithKey(t *testing.T) {
	serviceOptions := CloudInfoServiceOptions{
		ApiKey:                    "dummy_key",
		VpcService:                new(vpcServiceMock),
		IamIdentityService:        new(iamIdentityServiceMock),
		IamPolicyService:          new(iamPolicyServiceMock),
		ResourceControllerService: new(resourceControllerServiceMock),
		ContainerClient:           new(containerClientMock),
		ContainerV1Client:         new(containerV1ClientMock),
	}

	_, err := NewCloudInfoServiceWithKey(serviceOptions)

	require.Nil(t, err, "Error returned getting new service")
}

func TestNewServiceWithEnv(t *testing.T) {
	serviceOptions := CloudInfoServiceOptions{
		VpcService:                new(vpcServiceMock),
		IamIdentityService:        new(iamIdentityServiceMock),
		IamPolicyService:          new(iamPolicyServiceMock),
		ResourceControllerService: new(resourceControllerServiceMock),
		ContainerClient:           new(containerClientMock),
		ContainerV1Client:         new(containerV1ClientMock),
	}

	if err := os.Setenv("TEST_KEY_VAL", "dummy_key"); err != nil {
		t.Fatalf("Failed to set environment variable: %v", err)
	}
	_, err := NewCloudInfoServiceFromEnv("TEST_KEY_VAL", serviceOptions)

	require.Nil(t, err, "Error returned getting new service")

}

func TestNewServiceWithEmptyKey(t *testing.T) {
	serviceOptions := CloudInfoServiceOptions{
		VpcService:                new(vpcServiceMock),
		IamIdentityService:        new(iamIdentityServiceMock),
		IamPolicyService:          new(iamPolicyServiceMock),
		ResourceControllerService: new(resourceControllerServiceMock),
		ContainerClient:           new(containerClientMock),
		ContainerV1Client:         new(containerV1ClientMock),
	}

	_, err := NewCloudInfoServiceWithKey(serviceOptions)

	require.NotNil(t, err, "Empty key should have resulted in error")
}

func TestNewServiceWithEmptyEnv(t *testing.T) {
	serviceOptions := CloudInfoServiceOptions{
		VpcService:                new(vpcServiceMock),
		IamIdentityService:        new(iamIdentityServiceMock),
		IamPolicyService:          new(iamPolicyServiceMock),
		ResourceControllerService: new(resourceControllerServiceMock),
		ContainerClient:           new(containerClientMock),
		ContainerV1Client:         new(containerV1ClientMock),
	}

	_, err := NewCloudInfoServiceFromEnv("", serviceOptions)

	require.NotNil(t, err, "Empty Environment key should have resulted in error")

}

func TestDefaultLogsRouterService_ListTenants(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		tenantID := "tenant-123"
		tenantName := "test-tenant"
		tenantCRN := "crn:v1:bluemix:public:logs-router:us-east:a/acc:::"
		createdAt := "2024-01-01T00:00:00Z"
		updatedAt := "2024-01-02T00:00:00Z"

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "/v1/tenants", r.URL.Path)
			assert.Equal(t, "application/json", r.Header.Get("Accept"))
			assert.Equal(t, "2024-03-01", r.Header.Get("IBM-API-Version"))
			assert.Equal(t, "Bearer mock-token", r.Header.Get("Authorization"))

			resp := LogsRouterTenantCollection{
				Tenants: []LogsRouterTenant{
					{
						ID:        &tenantID,
						Name:      &tenantName,
						CRN:       &tenantCRN,
						CreatedAt: &createdAt,
						UpdatedAt: &updatedAt,
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		auth := &mockAuthenticator{token: "mock-token"} // pragma: allowlist secret
		svc := &defaultLogsRouterService{
			authenticator: auth,
			urlTemplate:   server.URL,
		}

		tenants, resp, err := svc.ListTenants("us-east")
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		require.Len(t, tenants, 1)
		assert.Equal(t, tenantID, *tenants[0].ID)
		assert.Equal(t, tenantName, *tenants[0].Name)
		assert.Equal(t, tenantCRN, *tenants[0].CRN)
	})

	t.Run("EmptyTenants", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := LogsRouterTenantCollection{
				Tenants: []LogsRouterTenant{},
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		auth := &mockAuthenticator{token: "mock-token"} // pragma: allowlist secret
		svc := &defaultLogsRouterService{
			authenticator: auth,
			urlTemplate:   server.URL,
		}

		tenants, resp, err := svc.ListTenants("eu-de")
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Empty(t, tenants)
	})

	t.Run("ServerError", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"errors":[{"code":"internal_error","message":"something went wrong"}]}`))
		}))
		defer server.Close()

		auth := &mockAuthenticator{token: "mock-token"} // pragma: allowlist secret
		svc := &defaultLogsRouterService{
			authenticator: auth,
			urlTemplate:   server.URL,
		}

		tenants, resp, err := svc.ListTenants("us-south")
		require.Error(t, err)
		assert.Nil(t, tenants)
		if resp != nil {
			assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
		}
	})

	t.Run("DefaultURLFormatting", func(t *testing.T) {
		svc := &defaultLogsRouterService{
			authenticator: &mockAuthenticator{token: "mock-token"}, // pragma: allowlist secret
		}
		expectedURL := "https://management.us-east.logs-router.cloud.ibm.com"
		assert.Equal(t, expectedURL, svc.getServiceURL("us-east"))
	})
}
