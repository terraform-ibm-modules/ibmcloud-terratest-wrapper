package common

import (
	"bytes"
	"log"
	"regexp"
	"testing"
)

func TestLoggerFunction(t *testing.T) {
	testCases := []struct {
		name           string
		prefix         string
		logFunc        func(*TestLogger)
		expectedOutput string
	}{
		{
			name:   "Info",
			prefix: "v1.0",
			logFunc: func(logger *TestLogger) {
				logger.Info("This is an info message")
			},
			expectedOutput: `\033\[32mINFO\033\[0m: \033\[32m\[TestName - v1.0\]\033\[0m .* This is an info message`,
		},
		{
			name:   "Error",
			prefix: "v1.0",
			logFunc: func(logger *TestLogger) {
				logger.Error("This is an error message")
			},
			expectedOutput: `\033\[31mERROR\033\[0m: \033\[31m\[TestName - v1.0\]\033\[0m .* This is an error message`,
		},
		{
			name:   "Debug",
			prefix: "v1.0",
			logFunc: func(logger *TestLogger) {
				logger.Debug("This is a debug message")
			},
			expectedOutput: `\033\[34mDEBUG\033\[0m: \033\[34m\[TestName - v1.0\]\033\[0m .* This is a debug message`,
		},
		{
			name:   "Warn",
			prefix: "v1.0",
			logFunc: func(logger *TestLogger) {
				logger.Warn("This is a warning message")
			},
			expectedOutput: `\033\[33mWARN\033\[0m: \033\[33m\[TestName - v1.0\]\033\[0m .* This is a warning message`,
		},
		{
			name:   "ShortInfo",
			prefix: "Projects - 1234",
			logFunc: func(logger *TestLogger) {
				logger.ShortInfo("This is a short info message")
			},
			expectedOutput: `\033\[32m\[TestName - Projects - 1234\]\033\[0m This is a short info message`,
		},
		{
			name: "ShortInfoNoPrefix",
			logFunc: func(logger *TestLogger) {
				logger.ShortInfo("This is a short info message")
			},
			expectedOutput: `\033\[32m\[TestName\]\033\[0m This is a short info message`,
		},
		{
			name:   "Custom",
			prefix: "v1.0",
			logFunc: func(logger *TestLogger) {
				logger.Custom("CUSTOM", "This is a custom message", Colors.Blue)
			},
			expectedOutput: `\033\[34mCUSTOM\033\[0m: \033\[34m\[TestName - v1.0\]\033\[0m .* This is a custom message`,
		},
		{
			name: "ShortCustom",
			logFunc: func(logger *TestLogger) {
				logger.ShortCustom("This is a short custom message", Colors.Blue)
			},
			expectedOutput: `\033\[34m\[TestName\]\033\[0m This is a short custom message`,
		},
		{
			name: "ShortError",
			logFunc: func(logger *TestLogger) {
				logger.ShortError("This is a short error message")
			},
			expectedOutput: `\033\[31m\[TestName\]\033\[0m This is a short error message`,
		},
		{
			name: "ShortDebug",
			logFunc: func(logger *TestLogger) {
				logger.ShortDebug("This is a short debug message")
			},
			expectedOutput: `\033\[34m\[TestName\]\033\[0m This is a short debug message`,
		},
		{
			name: "ShortWarn",
			logFunc: func(logger *TestLogger) {
				logger.ShortWarn("This is a short warning message")
			},
			expectedOutput: `\033\[33m\[TestName\]\033\[0m This is a short warning message`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := &TestLogger{
				logger:   log.New(&buf, "", 0),
				testName: "TestName",
				prefix:   tc.prefix,
			}

			// Call the logging function
			tc.logFunc(logger)

			// Compile the expected output regex
			re := regexp.MustCompile(tc.expectedOutput)

			// Check if the actual output matches the expected output regex
			if !re.MatchString(buf.String()) {
				t.Errorf("expected log to match '%s', got %s", tc.expectedOutput, buf.String())
			}
		})
	}
}
