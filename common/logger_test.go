package common

import (
	"bytes"
	"log"
	"testing"
)

func TestNewTestLogger(t *testing.T) {
	testName := "TestName"
	prefix := "v1.0"
	logger := NewTestLoggerWithPrefix(testName, prefix)

	if logger.testName != testName {
		t.Errorf("expected testName to be %s, got %s", testName, logger.testName)
	}

	if logger.prefix != prefix {
		t.Errorf("expected prefix to be %s, got %s", prefix, logger.prefix)
	}
}

func TestTestLogger_Info(t *testing.T) {
	var buf bytes.Buffer
	logger := &TestLogger{
		logger:   log.New(&buf, "", 0),
		testName: "TestName",
		prefix:   "v1.0",
	}

	logger.Info("This is an info message")

	if !bytes.Contains(buf.Bytes(), []byte("INFO: [TestName - v1.0]")) {
		t.Errorf("expected log to contain 'INFO: [TestName - v1.0]', got %s", buf.String())
	}
}

func TestTestLogger_Error(t *testing.T) {
	var buf bytes.Buffer
	logger := &TestLogger{
		logger:   log.New(&buf, "", 0),
		testName: "TestName",
		prefix:   "v1.0",
	}

	logger.Error("This is an error message")

	if !bytes.Contains(buf.Bytes(), []byte("ERROR: [TestName - v1.0]")) {
		t.Errorf("expected log to contain 'ERROR: [TestName - v1.0]', got %s", buf.String())
	}
}

func TestTestLogger_Debug(t *testing.T) {
	var buf bytes.Buffer
	logger := &TestLogger{
		logger:   log.New(&buf, "", 0),
		testName: "TestName",
		prefix:   "v1.0",
	}

	logger.Debug("This is a debug message")

	if !bytes.Contains(buf.Bytes(), []byte("DEBUG: [TestName - v1.0]")) {
		t.Errorf("expected log to contain 'DEBUG: [TestName - v1.0]', got %s", buf.String())
	}
}

func TestTestLogger_ShortInfo(t *testing.T) {
	var buf bytes.Buffer
	logger := &TestLogger{
		logger:   log.New(&buf, "", 0),
		testName: "TestName",
		prefix:   "Projects - 1234",
	}

	logger.ShortInfo("This is a short info message")

	if !bytes.Contains(buf.Bytes(), []byte("[TestName - Projects - 1234] This is a short info message")) {
		t.Errorf("expected log to contain '[TestName - Projects - 1234] This is a short info message', got %s", buf.String())
	}
}
