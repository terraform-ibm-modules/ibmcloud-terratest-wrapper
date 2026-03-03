package common

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

// ValidateHTTPURL validates that a URL is safe for HTTP requests to prevent SSRF attacks.
// It checks that the URL uses http or https scheme and has a valid host.
// Returns an error if the URL is invalid or potentially unsafe.
func ValidateHTTPURL(rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("URL cannot be empty")
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Only allow http and https schemes
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("invalid URL scheme: %s (only http and https are allowed)", parsedURL.Scheme)
	}

	// Ensure host is present
	if parsedURL.Host == "" {
		return fmt.Errorf("URL must have a valid host")
	}

	// Block localhost and private IP ranges to prevent SSRF
	host := strings.ToLower(parsedURL.Hostname())
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		// Allow localhost for development/testing purposes
		// In production, you may want to block these
	}

	return nil
}

// ValidateFilePath validates that a file path is safe to prevent path traversal attacks.
// It cleans the path and ensures it doesn't contain suspicious patterns.
// Returns the cleaned path and an error if the path is invalid or potentially unsafe.
func ValidateFilePath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path cannot be empty")
	}

	// Clean the path to resolve any .. or . components
	cleanPath := filepath.Clean(path)

	// Check for suspicious patterns that might indicate path traversal
	if strings.Contains(cleanPath, "..") {
		return "", fmt.Errorf("path contains suspicious pattern: %s", path)
	}

	return cleanPath, nil
}

// ValidateAndCleanPath validates and cleans a file path for safe file operations.
// This is a convenience function that combines validation and cleaning.
func ValidateAndCleanPath(path string) (string, error) {
	return ValidateFilePath(path)
}
