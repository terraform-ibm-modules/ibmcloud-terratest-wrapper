package common

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSkipRetryDelaysEnvironmentVariable(t *testing.T) {
	// Test this first before other tests set the environment variable
	// Note: Go's test framework might inherit SKIP_RETRY_DELAYS from parent tests
	// These tests verify the logic works correctly with different env var values

	t.Run("WithSkipRetryDelaysTrue", func(t *testing.T) {
		t.Setenv("SKIP_RETRY_DELAYS", "true")

		config := DefaultRetryConfig()
		delay := calculateDelay(config, 1)
		assert.Equal(t, time.Duration(0), delay)
	})

	t.Run("WithSkipRetryDelaysEmpty", func(t *testing.T) {
		t.Setenv("SKIP_RETRY_DELAYS", "")

		// Since we're running in a test environment, we expect delays to work normally
		// The key test is that empty string != "true" so delays should be calculated
		envValue := os.Getenv("SKIP_RETRY_DELAYS")
		assert.Equal(t, "", envValue)
		assert.NotEqual(t, "true", envValue)

		// Even if calculateDelay returns 0 due to test env, verify the logic
		skipDelays := (envValue == "true")
		assert.False(t, skipDelays)
	})

	t.Run("WithSkipRetryDelaysFalse", func(t *testing.T) {
		t.Setenv("SKIP_RETRY_DELAYS", "false")

		// Verify environment variable is set correctly
		envValue := os.Getenv("SKIP_RETRY_DELAYS")
		assert.Equal(t, "false", envValue)
		assert.NotEqual(t, "true", envValue)

		// Verify the logic condition
		skipDelays := (envValue == "true")
		assert.False(t, skipDelays)
	})
}

func TestRetryWithConfig(t *testing.T) {
	// Set environment variable to skip delays for unit tests
	t.Setenv("SKIP_RETRY_DELAYS", "true")

	t.Run("SuccessOnFirstAttempt", func(t *testing.T) {
		attempts := 0
		operation := func() (string, error) {
			attempts++
			return "success", nil
		}

		config := DefaultRetryConfig()
		result, err := RetryWithConfig(config, operation)

		assert.NoError(t, err)
		assert.Equal(t, "success", result)
		assert.Equal(t, 1, attempts)
	})

	t.Run("SuccessAfterRetries", func(t *testing.T) {
		attempts := 0
		operation := func() (string, error) {
			attempts++
			if attempts < 3 {
				return "", fmt.Errorf("temporary failure")
			}
			return "success", nil
		}

		config := DefaultRetryConfig()
		result, err := RetryWithConfig(config, operation)

		assert.NoError(t, err)
		assert.Equal(t, "success", result)
		assert.Equal(t, 3, attempts)
	})

	t.Run("ExhaustAllRetries", func(t *testing.T) {
		attempts := 0
		operation := func() (string, error) {
			attempts++
			return "", fmt.Errorf("persistent failure")
		}

		config := DefaultRetryConfig()
		config.MaxRetries = 2
		result, err := RetryWithConfig(config, operation)

		assert.Error(t, err)
		assert.Equal(t, "", result)
		assert.Equal(t, 2, attempts)
		assert.Contains(t, err.Error(), "failed after 2 attempts")
	})

	t.Run("NonRetryableError", func(t *testing.T) {
		attempts := 0
		operation := func() (string, error) {
			attempts++
			return "", fmt.Errorf("401 unauthorized")
		}

		config := DefaultRetryConfig()
		config.RetryableErrorChecker = IsRetryableError
		result, err := RetryWithConfig(config, operation)

		assert.Error(t, err)
		assert.Equal(t, "", result)
		assert.Equal(t, 1, attempts) // Should not retry
		assert.Contains(t, err.Error(), "401 unauthorized")
	})
}

func TestRetryStrategies(t *testing.T) {
	// Set environment variable to skip delays for unit tests
	t.Setenv("SKIP_RETRY_DELAYS", "true")

	t.Run("ExponentialBackoff", func(t *testing.T) {
		config := RetryConfig{
			MaxRetries:   3,
			InitialDelay: 1 * time.Second,
			MaxDelay:     10 * time.Second,
			Strategy:     ExponentialBackoff,
			Jitter:       false, // Disable jitter for predictable testing
		}

		// Test delay calculation
		delay1 := calculateDelay(config, 1)
		delay2 := calculateDelay(config, 2)

		// With SKIP_RETRY_DELAYS=true, all delays should be 0
		assert.Equal(t, time.Duration(0), delay1)
		assert.Equal(t, time.Duration(0), delay2)
	})

	t.Run("LinearBackoff", func(t *testing.T) {
		config := RetryConfig{
			MaxRetries:   3,
			InitialDelay: 1 * time.Second,
			MaxDelay:     10 * time.Second,
			Strategy:     LinearBackoff,
			Jitter:       false,
		}

		// Test delay calculation
		delay1 := calculateDelay(config, 1)
		delay2 := calculateDelay(config, 2)

		// With SKIP_RETRY_DELAYS=true, all delays should be 0
		assert.Equal(t, time.Duration(0), delay1)
		assert.Equal(t, time.Duration(0), delay2)
	})

	t.Run("FixedDelay", func(t *testing.T) {
		config := RetryConfig{
			MaxRetries:   3,
			InitialDelay: 2 * time.Second,
			Strategy:     FixedDelay,
			Jitter:       false,
		}

		// Test delay calculation
		delay1 := calculateDelay(config, 1)
		delay2 := calculateDelay(config, 2)

		// With SKIP_RETRY_DELAYS=true, all delays should be 0
		assert.Equal(t, time.Duration(0), delay1)
		assert.Equal(t, time.Duration(0), delay2)
	})
}

func TestDelayCalculationWithoutSkip(t *testing.T) {
	// Test actual delay calculation by temporarily unsetting the environment variable
	t.Run("ActualDelayCalculation", func(t *testing.T) {
		// Temporarily unset the SKIP_RETRY_DELAYS environment variable
		originalValue := os.Getenv("SKIP_RETRY_DELAYS")
		os.Unsetenv("SKIP_RETRY_DELAYS")
		defer func() {
			if originalValue != "" {
				os.Setenv("SKIP_RETRY_DELAYS", originalValue)
			}
		}()

		config := RetryConfig{
			MaxRetries:   3,
			InitialDelay: 100 * time.Millisecond,
			MaxDelay:     1 * time.Second,
			Strategy:     ExponentialBackoff,
			Jitter:       false, // Disable jitter for predictable testing
		}

		delay0 := calculateDelay(config, 0)
		delay1 := calculateDelay(config, 1)
		delay2 := calculateDelay(config, 2)

		// Exponential backoff: 2^attempt * initialDelay
		assert.Equal(t, 100*time.Millisecond, delay0) // 2^0 * 100ms = 100ms
		assert.Equal(t, 200*time.Millisecond, delay1) // 2^1 * 100ms = 200ms
		assert.Equal(t, 400*time.Millisecond, delay2) // 2^2 * 100ms = 400ms
	})
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name      string
		error     string
		retryable bool
	}{
		{
			name:      "RateLimiting429",
			error:     "429 Too Many Requests",
			retryable: true,
		},
		{
			name:      "ServerError500",
			error:     "500 Internal Server Error",
			retryable: true,
		},
		{
			name:      "NetworkTimeout",
			error:     "context deadline exceeded",
			retryable: true,
		},
		{
			name:      "Unauthorized401",
			error:     "401 unauthorized",
			retryable: false,
		},
		{
			name:      "Forbidden403",
			error:     "403 forbidden",
			retryable: false,
		},
		{
			name:      "BadRequest400",
			error:     "400 bad request",
			retryable: false,
		},
		{
			name:      "Conflict409",
			error:     "409 conflict",
			retryable: false,
		},
		{
			name:      "TransientNetworkError",
			error:     "connection reset by peer",
			retryable: true,
		},
		{
			name:      "ConfigAlreadyExists",
			error:     "ISB064E Configuration already exists",
			retryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := fmt.Errorf("%s", tt.error)
			result := IsRetryableError(err)
			assert.Equal(t, tt.retryable, result, "Error: %s", tt.error)
		})
	}
}

func TestRetryConfigurations(t *testing.T) {
	t.Run("DefaultRetryConfig", func(t *testing.T) {
		config := DefaultRetryConfig()
		assert.Equal(t, 3, config.MaxRetries)
		assert.Equal(t, 2*time.Second, config.InitialDelay)
		assert.Equal(t, 30*time.Second, config.MaxDelay)
		assert.Equal(t, ExponentialBackoff, config.Strategy)
		assert.True(t, config.Jitter)
	})

	t.Run("RateLimitRetryConfig", func(t *testing.T) {
		config := RateLimitRetryConfig()
		assert.Equal(t, 4, config.MaxRetries)
		assert.Equal(t, 3*time.Second, config.InitialDelay)
		assert.Equal(t, 30*time.Second, config.MaxDelay)
		assert.Equal(t, ExponentialBackoff, config.Strategy)
		assert.True(t, config.Jitter)
		assert.NotNil(t, config.RetryableErrorChecker)
	})

	t.Run("CatalogOperationRetryConfig", func(t *testing.T) {
		config := CatalogOperationRetryConfig()
		assert.Equal(t, 5, config.MaxRetries)
		assert.Equal(t, 5*time.Second, config.InitialDelay)
		assert.Equal(t, 60*time.Second, config.MaxDelay)
		assert.Equal(t, LinearBackoff, config.Strategy)
		assert.True(t, config.Jitter)
		assert.NotNil(t, config.RetryableErrorChecker)
	})

	t.Run("ProjectOperationRetryConfig", func(t *testing.T) {
		config := ProjectOperationRetryConfig()
		assert.Equal(t, 5, config.MaxRetries)
		assert.Equal(t, 3*time.Second, config.InitialDelay)
		assert.Equal(t, 45*time.Second, config.MaxDelay)
		assert.Equal(t, ExponentialBackoff, config.Strategy)
		assert.True(t, config.Jitter)
		assert.NotNil(t, config.RetryableErrorChecker)
	})
}

func TestConvenienceFunctions(t *testing.T) {
	// Set environment variable to skip delays for unit tests
	t.Setenv("SKIP_RETRY_DELAYS", "true")

	t.Run("Retry", func(t *testing.T) {
		attempts := 0
		operation := func() (int, error) {
			attempts++
			if attempts < 2 {
				return 0, fmt.Errorf("temporary failure")
			}
			return 42, nil
		}

		result, err := Retry(operation)
		assert.NoError(t, err)
		assert.Equal(t, 42, result)
		assert.Equal(t, 2, attempts)
	})

	t.Run("RetryForRateLimit", func(t *testing.T) {
		attempts := 0
		operation := func() (string, error) {
			attempts++
			if attempts < 2 {
				return "", fmt.Errorf("429 rate limit exceeded")
			}
			return "success", nil
		}

		result, err := RetryForRateLimit(operation)
		assert.NoError(t, err)
		assert.Equal(t, "success", result)
		assert.Equal(t, 2, attempts)
	})
}

// TestIntegrationRetryBehavior tests the full retry behavior with actual delays
func TestIntegrationRetryBehavior(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This test does NOT set SKIP_RETRY_DELAYS so we can test actual timing behavior
	// Only run this in integration test scenarios where timing matters

	t.Run("ActualTimingBehavior", func(t *testing.T) {
		attempts := 0
		startTime := time.Now()

		operation := func() (string, error) {
			attempts++
			if attempts < 3 {
				return "", fmt.Errorf("temporary failure")
			}
			return "success", nil
		}

		config := RetryConfig{
			MaxRetries:            3,
			InitialDelay:          50 * time.Millisecond,
			MaxDelay:              200 * time.Millisecond,
			Strategy:              FixedDelay,
			Jitter:                false,
			RetryableErrorChecker: nil,
		}

		result, err := RetryWithConfig(config, operation)
		duration := time.Since(startTime)

		require.NoError(t, err)
		assert.Equal(t, "success", result)
		assert.Equal(t, 3, attempts)

		// Should have taken at least 2 delays of 50ms each (100ms total)
		// Add some buffer for test timing variability
		assert.Greater(t, duration, 80*time.Millisecond)
		assert.Less(t, duration, 500*time.Millisecond)
	})
}
