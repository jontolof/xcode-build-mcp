package types

import (
	"fmt"
)

type ErrorCode string

const (
	ErrCodeInvalidParams    ErrorCode = "INVALID_PARAMS"
	ErrCodeProjectNotFound  ErrorCode = "PROJECT_NOT_FOUND"
	ErrCodeSchemeNotFound   ErrorCode = "SCHEME_NOT_FOUND"
	ErrCodeBuildFailed      ErrorCode = "BUILD_FAILED"
	ErrCodeTestFailed       ErrorCode = "TEST_FAILED"
	ErrCodeSimulatorNotFound ErrorCode = "SIMULATOR_NOT_FOUND"
	ErrCodeAppNotFound      ErrorCode = "APP_NOT_FOUND"
	ErrCodeInstallFailed    ErrorCode = "INSTALL_FAILED"
	ErrCodeLaunchFailed     ErrorCode = "LAUNCH_FAILED"
	ErrCodeTimeout          ErrorCode = "TIMEOUT"
	ErrCodeInternal         ErrorCode = "INTERNAL_ERROR"
)

type XcodeError struct {
	Code     ErrorCode              `json:"code"`
	Message  string                 `json:"message"`
	Details  map[string]interface{} `json:"details,omitempty"`
	Cause    error                  `json:"cause,omitempty"`
}

func (e *XcodeError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func NewXcodeError(code ErrorCode, message string, details map[string]interface{}) *XcodeError {
	return &XcodeError{
		Code:    code,
		Message: message,
		Details: details,
	}
}

func NewXcodeErrorWithCause(code ErrorCode, message string, cause error, details map[string]interface{}) *XcodeError {
	return &XcodeError{
		Code:    code,
		Message: message,
		Cause:   cause,
		Details: details,
	}
}

func WrapError(err error, code ErrorCode, message string) *XcodeError {
	return &XcodeError{
		Code:    code,
		Message: message,
		Cause:   err,
	}
}

type BuildError struct {
	File        string `json:"file"`
	Line        int    `json:"line,omitempty"`
	Column      int    `json:"column,omitempty"`
	Message     string `json:"message"`
	Severity    string `json:"severity"`
	Category    string `json:"category,omitempty"`
	Code        string `json:"code,omitempty"`
}

func (e *BuildError) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("%s:%d:%d: %s: %s", e.File, e.Line, e.Column, e.Severity, e.Message)
	}
	return fmt.Sprintf("%s: %s: %s", e.File, e.Severity, e.Message)
}

type BuildWarning struct {
	File        string `json:"file"`
	Line        int    `json:"line,omitempty"`
	Column      int    `json:"column,omitempty"`
	Message     string `json:"message"`
	Category    string `json:"category,omitempty"`
	Code        string `json:"code,omitempty"`
}

func (w *BuildWarning) Error() string {
	if w.Line > 0 {
		return fmt.Sprintf("%s:%d:%d: warning: %s", w.File, w.Line, w.Column, w.Message)
	}
	return fmt.Sprintf("%s: warning: %s", w.File, w.Message)
}

func IsXcodeError(err error, code ErrorCode) bool {
	if xerr, ok := err.(*XcodeError); ok {
		return xerr.Code == code
	}
	return false
}

func ExtractXcodeError(err error) *XcodeError {
	if xerr, ok := err.(*XcodeError); ok {
		return xerr
	}
	return nil
}