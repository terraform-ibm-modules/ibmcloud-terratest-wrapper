package common

import (
	"bytes"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnhancedErrorMethods(t *testing.T) {
	t.Run("TestLoggerCriticalError", func(t *testing.T) {
		var buf bytes.Buffer
		testLogger := &TestLogger{
			logger:   log.New(&buf, "", 0),
			testName: "test",
		}

		testLogger.CriticalError("This is a critical error")

		output := buf.String()
		assert.Contains(t, output, "================================================================================")
		assert.Contains(t, output, "CRITICAL ERROR: This is a critical error")
	})

	t.Run("TestLoggerFatalError", func(t *testing.T) {
		var buf bytes.Buffer
		testLogger := &TestLogger{
			logger:   log.New(&buf, "", 0),
			testName: "test",
		}

		testLogger.FatalError("This is a fatal error")

		output := buf.String()
		assert.Contains(t, output, "FATAL ERROR: This is a fatal error")
		assert.NotContains(t, output, "================") // No separator for fatal errors
	})

	t.Run("TestLoggerErrorWithContext", func(t *testing.T) {
		var buf bytes.Buffer
		testLogger := &TestLogger{
			logger:   log.New(&buf, "", 0),
			testName: "test",
		}

		testLogger.ErrorWithContext("This is an error with context")

		output := buf.String()
		assert.Contains(t, output, "------------------------------------------------------------")
		assert.Contains(t, output, "ERROR: This is an error with context")
	})

	t.Run("BufferedTestLoggerCriticalError", func(t *testing.T) {
		var buf bytes.Buffer
		baseLogger := &TestLogger{
			logger:   log.New(&buf, "", 0),
			testName: "test",
		}
		bufferedLogger := WrapWithBufferingIfNeeded(baseLogger, true)

		// Add some content to buffer
		bufferedLogger.ShortInfo("Some buffered info")
		bufferedLogger.ShortWarn("Some buffered warning")

		// Call critical error - should flush buffer first
		bufferedLogger.CriticalError("This is a critical error")

		output := buf.String()
		// Should see buffer flush indicators
		assert.Contains(t, output, "=== BUFFERED LOG OUTPUT ===")
		assert.Contains(t, output, "=== END BUFFERED LOG OUTPUT ===")
		// Then the critical error
		assert.Contains(t, output, "CRITICAL ERROR: This is a critical error")
		assert.Contains(t, output, "================================================================================")
		// Buffer size should be > 0 initially, then cleared after flush
		assert.Equal(t, 0, bufferedLogger.GetBufferSize())
	})

	t.Run("SmartLoggerErrorMethods", func(t *testing.T) {
		var buf bytes.Buffer
		baseLogger := &TestLogger{
			logger:   log.New(&buf, "", 0),
			testName: "test",
		}
		config := SmartLoggerConfig{PhasePatterns: AddonPhasePatterns}
		smartLogger := NewSmartLogger(baseLogger, config)

		smartLogger.FatalError("Smart logger fatal error")

		output := buf.String()
		assert.Contains(t, output, "FATAL ERROR: Smart logger fatal error")
	})
}
