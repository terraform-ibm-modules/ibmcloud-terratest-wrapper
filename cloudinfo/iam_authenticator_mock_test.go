package cloudinfo

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// MockAuthenticator implements the Authenticator interface for testing
type MockAuthenticator struct {
	Token string
	CoreAuthenticator
}

// AuthenticationType returns the authentication type
func (a *MockAuthenticator) AuthenticationType() string {
	return "Bearer"
}

// GetToken returns a mock token without making API calls
func (a *MockAuthenticator) GetToken() (string, error) {
	return a.Token, nil
}

// Authenticate adds the mock token to the request's Authorization header
func (a *MockAuthenticator) Authenticate(request *http.Request) error {
	request.Header.Set("Authorization", "Bearer "+a.Token)
	return nil
}

// Validate checks if the authenticator is properly configured
func (a *MockAuthenticator) Validate() error {
	// A mock authenticator is always valid for testing purposes
	return nil
}

func TestMockAuthenticator(t *testing.T) {
	// Create a new MockAuthenticator with a test token
	mockToken := "test-mock-token"
	auth := &MockAuthenticator{
		Token: mockToken,
	}

	// Test AuthenticationType method
	assert.Equal(t, "Bearer", auth.AuthenticationType())

	// Test GetToken method
	token, err := auth.GetToken()
	assert.NoError(t, err)
	assert.Equal(t, mockToken, token)

	// Test Authenticate method
	req, _ := http.NewRequest("GET", "https://example.com", nil)
	err = auth.Authenticate(req)
	assert.NoError(t, err)
	assert.Equal(t, "Bearer "+mockToken, req.Header.Get("Authorization"))

	// Test Validate method
	err = auth.Validate()
	assert.NoError(t, err)
}
