package common

import (
	"bytes"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSmartLoggerPhaseDetection(t *testing.T) {
	// Test that the SmartLogger automatically detects phases and shows progress
	t.Run("AutomaticPhaseDetectionInQuietMode", func(t *testing.T) {
		// Capture output
		var buf bytes.Buffer

		// Create base logger that outputs to buffer
		baseTestLogger := &TestLogger{
			logger:   log.New(&buf, "", 0),
			testName: "SmartLoggerTest",
		}

		// Wrap with buffering (quiet mode enabled)
		bufferedLogger := WrapWithBufferingIfNeeded(baseTestLogger, true)
		assert.NotNil(t, bufferedLogger)
		assert.True(t, bufferedLogger.IsQuietMode())

		// Wrap with smart detection using addon patterns
		config := SmartLoggerConfig{PhasePatterns: AddonPhasePatterns}
		smartLogger := NewSmartLogger(bufferedLogger, config)
		assert.NotNil(t, smartLogger)

		// Test phase detection - this should trigger a progress message
		smartLogger.ShortInfo("Getting offering details: catalogID='test-catalog', offeringID='test-offering'")

		// The raw debug message should be buffered (not shown), but progress should be shown
		output := buf.String()
		assert.Contains(t, output, "🔄 Retrieving catalog information", "Expected progress stage to be shown immediately")
		assert.NotContains(t, output, "Getting offering details: catalogID", "Raw debug message should be buffered in quiet mode")

		// Test that subsequent similar messages don't repeat the progress
		buf.Reset()
		smartLogger.ShortInfo("Getting offering details: catalogID='test-catalog2', offeringID='test-offering2'")

		// Should not show progress again for same phase
		output = buf.String()
		assert.NotContains(t, output, "🔄 Retrieving catalog information", "Should not repeat same progress stage")
		assert.NotContains(t, output, "Getting offering details: catalogID='test-catalog2'", "Raw debug should still be buffered")

		// Test a different phase
		buf.Reset()
		smartLogger.ShortInfo("Starting reference resolution for project: test-project")
		output = buf.String()
		assert.Contains(t, output, "🔄 Resolving project references", "New phase should trigger new progress stage")
	})

	t.Run("PhaseDetectionInNormalMode", func(t *testing.T) {
		// In normal mode, both progress and raw messages should be shown
		var buf bytes.Buffer

		baseTestLogger := &TestLogger{
			logger:   log.New(&buf, "", 0),
			testName: "SmartLoggerTest",
		}

		// No buffering (normal mode)
		normalLogger := WrapWithBufferingIfNeeded(baseTestLogger, false)
		config := SmartLoggerConfig{PhasePatterns: AddonPhasePatterns}
		smartLogger := NewSmartLogger(normalLogger, config)

		smartLogger.ShortInfo("Getting offering details: catalogID='test-catalog', offeringID='test-offering'")

		output := buf.String()
		assert.Contains(t, output, "🔄 Retrieving catalog information", "Progress stage should be shown")
		assert.Contains(t, output, "Getting offering details: catalogID", "Raw message should also be shown in normal mode")
	})

	t.Run("UnrecognizedMessagesPassThrough", func(t *testing.T) {
		// Messages that don't match phase patterns should pass through normally
		var buf bytes.Buffer

		baseTestLogger := &TestLogger{
			logger:   log.New(&buf, "", 0),
			testName: "SmartLoggerTest",
		}

		config := SmartLoggerConfig{PhasePatterns: AddonPhasePatterns}
		smartLogger := NewSmartLogger(WrapWithBufferingIfNeeded(baseTestLogger, false), config)

		smartLogger.ShortInfo("This is just a regular log message")

		output := buf.String()
		assert.Contains(t, output, "This is just a regular log message", "Regular messages should pass through")
		assert.NotContains(t, output, "🔄", "No progress stage should be triggered for unrecognized messages")
	})
}

func TestSmartLoggerInterfaceCompliance(t *testing.T) {
	// Test that SmartLogger properly implements the Logger interface
	baseLogger := NewTestLogger("test")
	config := SmartLoggerConfig{PhasePatterns: AddonPhasePatterns}
	smartLogger := NewSmartLogger(baseLogger, config)

	// This should compile without errors if interface is properly implemented
	var logger Logger = smartLogger
	assert.NotNil(t, logger)

	// Test all interface methods work
	logger.ShortInfo("test")
	logger.Info("test")
	logger.ShortError("test")
	logger.Error("test")
	logger.ShortWarn("test")
	logger.Warn("test")
	logger.ShortDebug("test")
	logger.Debug("test")
	logger.Custom("CUSTOM", "test", Colors.Blue)
	logger.ShortCustom("test", Colors.Green)
	logger.ProgressStage("test stage")
	logger.ProgressSuccess("test success")
	logger.ProgressInfo("test info")
	logger.SetPrefix("test-prefix")
	logger.EnableDateTime(true)
	logger.SetQuietMode(false)
	assert.False(t, logger.IsQuietMode())
	logger.MarkFailed()
	logger.FlushBuffer()
	logger.FlushOnFailure()
	logger.ClearBuffer()
	assert.Equal(t, 0, logger.GetBufferSize())
	assert.NotNil(t, logger.GetUnderlyingLogger())
}

func TestAutoBufferingHelpers(t *testing.T) {
	t.Run("CreateSmartAutoBufferingLogger", func(t *testing.T) {
		// Test quiet mode
		logger := CreateSmartAutoBufferingLogger("test", true)
		assert.NotNil(t, logger)
		assert.True(t, logger.IsQuietMode())

		// Test normal mode
		logger = CreateSmartAutoBufferingLogger("test", false)
		assert.NotNil(t, logger)
		assert.False(t, logger.IsQuietMode())
	})

	t.Run("CreateSmartAutoBufferingLoggerWithPrefix", func(t *testing.T) {
		logger := CreateSmartAutoBufferingLoggerWithPrefix("test", "prefix", true)
		assert.NotNil(t, logger)
		assert.True(t, logger.IsQuietMode())

		// Verify prefix is set (check underlying logger)
		underlying := logger.GetUnderlyingLogger()
		assert.NotNil(t, underlying)
		assert.Equal(t, "prefix", underlying.prefix)
	})
}
