package cloudinfo

import "net/http"

// IiamAuthenticator defines the interface for authentication that can be implemented
// by both the real authenticator and test mocks.
// This interface matches the methods used from the *core.IamAuthenticator
type IiamAuthenticator interface {
	// GetToken returns an access token to be used in an Authorization header
	GetToken() (string, error)

	// AuthenticationType returns the authentication type for this authenticator
	AuthenticationType() string

	// Authenticate adds authentication information to the request
	Authenticate(request *http.Request) error

	// Validate checks if the authenticator is properly configured
	Validate() error
}

// Making the CoreAuthenticator an alias of the implementation from IBM SDK
// This allows our code to directly access the ApiKey field
type CoreAuthenticator struct {
	// The apikey used to fetch the bearer token from the IAM token server.
	ApiKey string
}
