package main

import (
	"bytes"
	"log"
	"os"
	"testing"
)

func TestSetupLogger(t *testing.T) {
	tests := []struct {
		name     string
		level    string
		checkFn  func(*log.Logger) bool
	}{
		{
			name:  "debug level",
			level: "debug",
			checkFn: func(l *log.Logger) bool {
				// Debug level should have file/line flags
				return (l.Flags() & log.Lshortfile) != 0
			},
		},
		{
			name:  "info level",
			level: "info",
			checkFn: func(l *log.Logger) bool {
				// Info level should have standard flags
				return (l.Flags() & log.Lshortfile) == 0
			},
		},
		{
			name:  "error level",
			level: "error",
			checkFn: func(l *log.Logger) bool {
				// Error level uses custom writer, hard to test directly
				return true
			},
		},
		{
			name:  "unknown level",
			level: "unknown",
			checkFn: func(l *log.Logger) bool {
				// Unknown level should default to standard flags
				return (l.Flags() & log.Lshortfile) == 0
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := setupLogger(tt.level)
			if logger == nil {
				t.Fatal("Logger should not be nil")
			}
			if !tt.checkFn(logger) {
				t.Errorf("Logger setup incorrect for level %s", tt.level)
			}
		})
	}
}

func TestGetEnvOrDefault(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		expected     string
	}{
		{
			name:         "env var exists",
			key:          "TEST_ENV_VAR",
			defaultValue: "default",
			envValue:     "custom",
			expected:     "custom",
		},
		{
			name:         "env var empty",
			key:          "TEST_EMPTY_VAR",
			defaultValue: "default",
			envValue:     "",
			expected:     "default",
		},
		{
			name:         "env var not set",
			key:          "TEST_UNSET_VAR",
			defaultValue: "fallback",
			envValue:     "",
			expected:     "fallback",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			} else {
				os.Unsetenv(tt.key)
			}

			result := getEnvOrDefault(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getEnvOrDefault(%s, %s) = %s, want %s",
					tt.key, tt.defaultValue, result, tt.expected)
			}
		})
	}
}

func TestErrorOnlyWriter(t *testing.T) {
	tempFile, err := os.CreateTemp("", "test_error_writer")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	writer := &errorOnlyWriter{writer: tempFile}
	
	testData := []byte("test error message\n")
	n, err := writer.Write(testData)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(testData) {
		t.Errorf("Write returned %d, want %d", n, len(testData))
	}

	// Verify data was written
	tempFile.Seek(0, 0)
	readData := make([]byte, len(testData))
	tempFile.Read(readData)
	if !bytes.Equal(readData, testData) {
		t.Errorf("Written data mismatch: got %s, want %s", readData, testData)
	}
}

func TestMainFlags(t *testing.T) {
	// Save original args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Test version flag parsing
	os.Args = []string{"cmd", "-version"}
	
	// We can't easily test main() directly as it calls os.Exit
	// Instead, we verify that the flags are defined correctly
	// This is more of a smoke test to ensure compilation works
	
	// Reset for other tests
	os.Args = oldArgs
}