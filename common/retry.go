package common

import (
	"fmt"
	"math/rand"
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
		MaxRetries:            5,
		InitialDelay:          3 * time.Second,
		MaxDelay:              30 * time.Second,
		Strategy:              LinearBackoff,
		Jitter:                true,
		RetryableErrorChecker: IsRetryableError,
		Logger:                nil,
		OperationName:         "catalog operation",
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

// IsRetryableError determines if an error should be retried based on common patterns
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Network-related errors that are common in parallel test execution
	retryablePatterns := []string{
		"timeout",
		"connection refused",
		"connection reset",
		"network is unreachable",
		"temporary failure",
		"rate limit",
		"too many requests",
		"service unavailable",
		"internal server error",
		"bad gateway",
		"gateway timeout",
		"deadline exceeded",
		"context deadline exceeded",
		"operation timed out",
		"server error",
		"502",
		"503",
		"504",
		"429",
		"500",
	}

	errLower := fmt.Sprintf("%v", err)
	for _, pattern := range retryablePatterns {
		if StringContainsIgnoreCase(errLower, pattern) {
			return true
		}
	}

	return false
}
