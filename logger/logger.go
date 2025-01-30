package logger

import (
	"bytes"
	"log"
	"sync"
	"testing"
)

var (
	logBuffer bytes.Buffer
	mu        sync.Mutex
)

// GetLogOutput returns the captured log output.
func GetLogOutput(t *testing.T) string {
	mu.Lock()
	defer mu.Unlock()
	return logBuffer.String()
}

// Log captures log output to a buffer.
func Log(t *testing.T, message string) {
	mu.Lock()
	defer mu.Unlock()
	log.SetOutput(&logBuffer)
	log.Println(message)
}

// FlushLog flushes the log buffer to ensure all logs are written out.
func FlushLog() {
	mu.Lock()
	defer mu.Unlock()
	log.SetOutput(nil)
}
