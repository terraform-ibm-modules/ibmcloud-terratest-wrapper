package common

import (
	"fmt"
	"math/rand"
	"os"
	"time"
)

// RetryStrategy defines different backoff strategies for retries
type RetryStrategy string

const (
	// ExponentialBackoff uses exponential backoff: initialDelay * 2^attempt
	ExponentialBackoff RetryStrategy = "exponential"
	// LinearBackoff uses linear backoff with jitter: initialDelay * (attempt + 1) + jitter
	LinearBackoff RetryStrategy = "linear"
	// FixedDelay uses a fixed delay between retries
	FixedDelay RetryStrategy = "fixed"
)

// RetryConfig contains configuration for retry behavior
type RetryConfig struct {
	// MaxRetries is the maximum number of retry attempts (default: 3)
	MaxRetries int
	// InitialDelay is the initial delay between retries (default: 2 seconds)
	InitialDelay time.Duration
	// MaxDelay is the maximum delay between retries to cap exponential backoff (default: 30 seconds)
	MaxDelay time.Duration
	// Strategy defines the backoff strategy (default: ExponentialBackoff)
	Strategy RetryStrategy
	// Jitter adds randomness to delays to avoid thundering herd (default: true for non-fixed strategies)
	Jitter bool
	// RetryableErrorChecker is a function that determines if an error should be retried
	RetryableErrorChecker func(error) bool
	// Logger is used for logging retry attempts (optional) - use Logger interface
	Logger Logger
	// OperationName is used in log messages to identify the operation being retried
	OperationName string
}

// DefaultRetryConfig returns a sensible default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:            3,
		InitialDelay:          2 * time.Second,
		MaxDelay:              30 * time.Second,
		Strategy:              ExponentialBackoff,
		Jitter:                true,
		RetryableErrorChecker: nil, // Will retry all errors by default
		Logger:                nil, // No logging by default
		OperationName:         "operation",
	}
}

// RateLimitRetryConfig returns a retry configuration optimized for rate limiting scenarios
func RateLimitRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:            4,
		InitialDelay:          3 * time.Second,
		MaxDelay:              30 * time.Second,
		Strategy:              ExponentialBackoff,
		Jitter:                true,
		RetryableErrorChecker: IsRetryableError, // Use common retryable error checker
		Logger:                nil,
		OperationName:         "rate-limited operation",
	}
}

// CatalogOperationRetryConfig returns a retry configuration for IBM Cloud catalog operations
func CatalogOperationRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:            10,
		InitialDelay:          5 * time.Second,
		MaxDelay:              120 * time.Second,
		Strategy:              ExponentialBackoff,
		Jitter:                true,
		RetryableErrorChecker: IsRetryableError,
		Logger:                nil,
		OperationName:         "catalog operation",
	}
}

// ProjectOperationRetryConfig returns a retry configuration optimized for IBM Cloud Projects operations
func ProjectOperationRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:            5,
		InitialDelay:          3 * time.Second,
		MaxDelay:              90 * time.Second,
		Strategy:              ExponentialBackoff,
		Jitter:                true,
		RetryableErrorChecker: IsProjectRetryableError,
		Logger:                nil,
		OperationName:         "project operation",
	}
}

// RetryWithConfig executes a function with retry logic based on the provided configuration
func RetryWithConfig[T any](config RetryConfig, operation func() (T, error)) (T, error) {
	var lastErr error
	var zeroValue T

	for attempt := 0; attempt < config.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := calculateDelay(config, attempt)
			if config.Logger != nil {
				config.Logger.ShortInfo(fmt.Sprintf("Retrying %s after %v (attempt %d/%d)", config.OperationName, delay, attempt+1, config.MaxRetries))
			}
			time.Sleep(delay)
		}

		result, err := operation()
		if err == nil {
			return result, nil
		}

		// Check if this error should be retried
		if config.RetryableErrorChecker != nil && !config.RetryableErrorChecker(err) {
			return zeroValue, err
		}

		lastErr = err

		// Don't retry on the last attempt
		if attempt == config.MaxRetries-1 {
			break
		}
	}

	return zeroValue, fmt.Errorf("failed after %d attempts: %w", config.MaxRetries, lastErr)
}

// Retry executes a function with default retry configuration
func Retry[T any](operation func() (T, error)) (T, error) {
	return RetryWithConfig(DefaultRetryConfig(), operation)
}

// RetryForRateLimit executes a function with rate limit optimized retry configuration
func RetryForRateLimit[T any](operation func() (T, error)) (T, error) {
	return RetryWithConfig(RateLimitRetryConfig(), operation)
}

// calculateDelay calculates the delay for the next retry attempt
func calculateDelay(config RetryConfig, attempt int) time.Duration {
	// Skip delays when SKIP_RETRY_DELAYS environment variable is set to "true"
	// This allows unit tests to run quickly while preserving retry logic and counting
	// Integration tests should NOT set this variable to allow proper rate limiting protection
	if os.Getenv("SKIP_RETRY_DELAYS") == "true" {
		return 0
	}

	var delay time.Duration

	switch config.Strategy {
	case ExponentialBackoff:
		// 2^attempt * initialDelay
		multiplier := 1 << uint(attempt) // 2^attempt
		delay = time.Duration(multiplier) * config.InitialDelay

	case LinearBackoff:
		// (attempt + 1) * initialDelay
		delay = time.Duration(attempt+1) * config.InitialDelay

	case FixedDelay:
		delay = config.InitialDelay

	default:
		// Default to exponential backoff
		multiplier := 1 << uint(attempt)
		delay = time.Duration(multiplier) * config.InitialDelay
	}

	// Cap the delay at MaxDelay
	if delay > config.MaxDelay {
		delay = config.MaxDelay
	}

	// Add jitter to avoid thundering herd problem
	if config.Jitter && config.Strategy != FixedDelay {
		jitterRange := float64(delay) * 0.30 // ±30% jitter
		jitter := time.Duration(jitterRange * (rand.Float64()*2 - 1))
		delay += jitter

		// Ensure we don't go negative
		if delay < 0 {
			delay = config.InitialDelay / 2
		}
	}

	return delay
}

// IsRetryableError determines if an error should be retried using a deny list approach
// DEFAULT BEHAVIOR: Retry all errors EXCEPT those in the non-retryable list
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := fmt.Sprintf("%v", err)

	// Non-retryable errors - these should fail immediately without retry
	// Default behavior: RETRY ALL OTHER ERRORS (including transient 404s, network errors, etc.)
	nonRetryablePatterns := []string{
		// Authentication and authorization errors
		"401",
		"unauthorized",
		"403",
		"forbidden",
		"invalid token",
		"authentication failed",
		"access denied",

		// Validation and permanent client errors
		"400",
		"bad request",
		"invalid parameter",
		"validation error",
		"malformed request",

		// Permanent conflicts and duplicates
		"ISB064E",                   // Config already exists
		"already exists in project", // Config already exists
		"409",
		"conflict",
		"duplicate",

		// Permanent not found errors (specific cases only)
		"resource permanently deleted",
		"catalog not found",
		"offering not found",
		"permanently removed",

		// Quota and limit exceeded (non-temporary)
		"quota exceeded",
		"limit exceeded permanently",
		"subscription expired",
	}

	for _, pattern := range nonRetryablePatterns {
		if StringContainsIgnoreCase(errStr, pattern) {
			return false
		}
	}

	// Default: RETRY all other errors including:
	// - Transient 404s (like ISB143E configuration not found due to eventual consistency)
	// - Network errors (timeouts, connection issues)
	// - Server errors (500s, 502s, 503s, 504s)
	// - Rate limiting (429)
	// - Any other transient errors
	return true
}

// IsProjectRetryableError determines if a project-related error should be retried
// Uses the same deny list approach as IsRetryableError with additional project-specific non-retryable patterns
func IsProjectRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check general retryable errors first (uses deny list approach - defaults to retry)
	if !IsRetryableError(err) {
		return false
	}

	// Project-specific non-retryable patterns
	// These are project errors that should NOT be retried even though they might pass the general check
	errStr := fmt.Sprintf("%v", err)
	projectNonRetryablePatterns := []string{
		// Project permission/authorization errors specific to Projects service
		"project access denied",
		"project not authorized",
		"insufficient project permissions",

		// Project validation errors
		"invalid project configuration",
		"project validation failed",
		"invalid project parameters",

		// Permanent project state errors
		"project permanently deleted",
		"project archived",
		"project disabled permanently",
	}

	for _, pattern := range projectNonRetryablePatterns {
		if StringContainsIgnoreCase(errStr, pattern) {
			return false
		}
	}

	// Default: Retry (following deny list philosophy)
	return true
}
