package types

import (
	"errors"
	"testing"
)

func TestXcodeError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      XcodeError
		expected string
	}{
		{
			name: "error without cause",
			err: XcodeError{
				Code:    ErrCodeBuildFailed,
				Message: "Build failed",
			},
			expected: "BUILD_FAILED: Build failed",
		},
		{
			name: "error with cause",
			err: XcodeError{
				Code:    ErrCodeInternal,
				Message: "Internal error",
				Cause:   errors.New("underlying error"),
			},
			expected: "INTERNAL_ERROR: Internal error (caused by: underlying error)",
		},
		{
			name: "error with details",
			err: XcodeError{
				Code:    ErrCodeProjectNotFound,
				Message: "Project not found",
				Details: map[string]interface{}{"path": "/path/to/project"},
			},
			expected: "PROJECT_NOT_FOUND: Project not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestNewXcodeError(t *testing.T) {
	code := ErrCodeInvalidParams
	message := "Invalid parameters"
	details := map[string]interface{}{"param": "value"}

	err := NewXcodeError(code, message, details)

	if err.Code != code {
		t.Errorf("Code = %s, want %s", err.Code, code)
	}
	if err.Message != message {
		t.Errorf("Message = %q, want %q", err.Message, message)
	}
	if err.Details == nil {
		t.Error("Details should not be nil")
	}
	if err.Cause != nil {
		t.Error("Cause should be nil when not provided")
	}
}

func TestNewXcodeErrorWithCause(t *testing.T) {
	code := ErrCodeTestFailed
	message := "Test failed"
	cause := errors.New("test error")
	details := map[string]interface{}{"test": "TestExample"}

	err := NewXcodeErrorWithCause(code, message, cause, details)

	if err.Code != code {
		t.Errorf("Code = %s, want %s", err.Code, code)
	}
	if err.Message != message {
		t.Errorf("Message = %q, want %q", err.Message, message)
	}
	if err.Cause != cause {
		t.Errorf("Cause = %v, want %v", err.Cause, cause)
	}
	if err.Details == nil {
		t.Error("Details should not be nil")
	}
}

func TestWrapError(t *testing.T) {
	originalErr := errors.New("original error")
	code := ErrCodeLaunchFailed
	message := "Failed to launch app"

	err := WrapError(originalErr, code, message)

	if err.Code != code {
		t.Errorf("Code = %s, want %s", err.Code, code)
	}
	if err.Message != message {
		t.Errorf("Message = %q, want %q", err.Message, message)
	}
	if err.Cause != originalErr {
		t.Errorf("Cause = %v, want %v", err.Cause, originalErr)
	}
}

func TestBuildError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      BuildError
		expected string
	}{
		{
			name: "error with line info",
			err: BuildError{
				File:     "main.swift",
				Line:     42,
				Column:   10,
				Message:  "Cannot find 'foo' in scope",
				Severity: "error",
			},
			expected: "main.swift:42:10: error: Cannot find 'foo' in scope",
		},
		{
			name: "error without line info",
			err: BuildError{
				File:     "Package.swift",
				Message:  "Missing dependency",
				Severity: "error",
			},
			expected: "Package.swift: error: Missing dependency",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestBuildWarning_Error(t *testing.T) {
	tests := []struct {
		name     string
		warn     BuildWarning
		expected string
	}{
		{
			name: "warning with line info",
			warn: BuildWarning{
				File:    "ViewController.swift",
				Line:    100,
				Column:  5,
				Message: "Variable 'unused' was never used",
			},
			expected: "ViewController.swift:100:5: warning: Variable 'unused' was never used",
		},
		{
			name: "warning without line info",
			warn: BuildWarning{
				File:    "AppDelegate.swift",
				Message: "Deprecated API usage",
			},
			expected: "AppDelegate.swift: warning: Deprecated API usage",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.warn.Error()
			if got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestIsXcodeError(t *testing.T) {
	xcodeErr := &XcodeError{
		Code:    ErrCodeBuildFailed,
		Message: "Build failed",
	}

	if !IsXcodeError(xcodeErr, ErrCodeBuildFailed) {
		t.Error("Should identify XcodeError with matching code")
	}

	if IsXcodeError(xcodeErr, ErrCodeTestFailed) {
		t.Error("Should not match different error code")
	}

	regularErr := errors.New("regular error")
	if IsXcodeError(regularErr, ErrCodeBuildFailed) {
		t.Error("Should not match non-XcodeError")
	}
}

func TestExtractXcodeError(t *testing.T) {
	xcodeErr := &XcodeError{
		Code:    ErrCodeSchemeNotFound,
		Message: "Scheme not found",
	}

	extracted := ExtractXcodeError(xcodeErr)
	if extracted == nil {
		t.Fatal("Should extract XcodeError")
	}
	if extracted != xcodeErr {
		t.Error("Should return the same XcodeError instance")
	}

	regularErr := errors.New("regular error")
	extracted = ExtractXcodeError(regularErr)
	if extracted != nil {
		t.Error("Should return nil for non-XcodeError")
	}
}
