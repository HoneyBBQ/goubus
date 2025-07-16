package goubus

import (
	"errors"
	"fmt"
)

// ErrorCode represents the type of error that occurred
type ErrorCode string

// Error codes for different types of errors
const (
	// Authentication errors
	ErrorCodeAuthenticationFailed ErrorCode = "AuthenticationFailed"
	ErrorCodeSessionExpired       ErrorCode = "SessionExpired"
	ErrorCodeNoActiveSession      ErrorCode = "NoActiveSession"
	ErrorCodePermissionDenied     ErrorCode = "PermissionDenied"

	// Connection errors
	ErrorCodeConnectionFailed  ErrorCode = "ConnectionFailed"
	ErrorCodeConnectionTimeout ErrorCode = "ConnectionTimeout"
	ErrorCodeNetworkError      ErrorCode = "NetworkError"

	// Data errors
	ErrorCodeInvalidData      ErrorCode = "InvalidData"
	ErrorCodeDataParsingError ErrorCode = "DataParsingError"
	ErrorCodeInvalidResponse  ErrorCode = "InvalidResponse"
	ErrorCodeUnexpectedFormat ErrorCode = "UnexpectedFormat"

	// Service errors
	ErrorCodeServiceNotFound ErrorCode = "ServiceNotFound"
	ErrorCodeMethodNotFound  ErrorCode = "MethodNotFound"
	ErrorCodeInvalidArgument ErrorCode = "InvalidArgument"
	ErrorCodeOperationFailed ErrorCode = "OperationFailed"

	// Module errors
	ErrorCodeModuleNotInstalled ErrorCode = "ModuleNotInstalled"
	ErrorCodeModuleNotFound     ErrorCode = "ModuleNotFound"

	// UCI errors
	ErrorCodeUCIOperationFailed ErrorCode = "UCIOperationFailed"
	ErrorCodeConfigNotFound     ErrorCode = "ConfigNotFound"
	ErrorCodeSectionNotFound    ErrorCode = "SectionNotFound"

	// File system errors
	ErrorCodeFileNotFound        ErrorCode = "FileNotFound"
	ErrorCodeFileOperationFailed ErrorCode = "FileOperationFailed"
	ErrorCodePermissionError     ErrorCode = "PermissionError"

	// Generic errors
	ErrorCodeInternal     ErrorCode = "InternalError"
	ErrorCodeNotSupported ErrorCode = "NotSupported"
	ErrorCodeTimeout      ErrorCode = "Timeout"
	ErrorCodeUnknown      ErrorCode = "Unknown"
)

// UbusError represents a structured error from the ubus system
type UbusError struct {
	Code     ErrorCode `json:"code"`
	Message  string    `json:"message"`
	Details  string    `json:"details,omitempty"`
	UbusCode int       `json:"ubus_code,omitempty"` // Original ubus error code
	Cause    error     `json:"-"`                   // Underlying error
	Service  string    `json:"service,omitempty"`   // Which ubus service caused the error
	Method   string    `json:"method,omitempty"`    // Which method caused the error
}

// Error implements the error interface
func (e *UbusError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("ubus error [%s]: %s (%s)", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("ubus error [%s]: %s", e.Code, e.Message)
}

// Unwrap returns the underlying error for error chain support
func (e *UbusError) Unwrap() error {
	return e.Cause
}

// Is implements error matching for errors.Is()
func (e *UbusError) Is(target error) bool {
	if t, ok := target.(*UbusError); ok {
		return e.Code == t.Code
	}
	return false
}

// Error creation functions

// NewError creates a new UbusError with the specified code and message
func NewError(code ErrorCode, message string) *UbusError {
	return &UbusError{
		Code:    code,
		Message: message,
	}
}

// NewErrorWithDetails creates a new UbusError with details
func NewErrorWithDetails(code ErrorCode, message, details string) *UbusError {
	return &UbusError{
		Code:    code,
		Message: message,
		Details: details,
	}
}

// NewErrorWithCause creates a new UbusError wrapping an underlying error
func NewErrorWithCause(code ErrorCode, message string, cause error) *UbusError {
	return &UbusError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// NewUbusCodeError creates an error from a ubus error code
func NewUbusCodeError(ubusCode int, service, method string) *UbusError {
	code, message := mapUbusCodeToError(ubusCode)
	return &UbusError{
		Code:     code,
		Message:  message,
		UbusCode: ubusCode,
		Service:  service,
		Method:   method,
	}
}

// WithService adds service context to an error
func (e *UbusError) WithService(service string) *UbusError {
	e.Service = service
	return e
}

// WithMethod adds method context to an error
func (e *UbusError) WithMethod(method string) *UbusError {
	e.Method = method
	return e
}

// WithDetails adds details to an error
func (e *UbusError) WithDetails(details string) *UbusError {
	e.Details = details
	return e
}

// WithCause adds a cause to an error
func (e *UbusError) WithCause(cause error) *UbusError {
	e.Cause = cause
	return e
}

// Predefined errors for common scenarios

var (
	// Authentication errors
	ErrAuthenticationFailed = NewError(ErrorCodeAuthenticationFailed, "authentication failed")
	ErrSessionExpired       = NewError(ErrorCodeSessionExpired, "ubus session has expired")
	ErrNoActiveSession      = NewError(ErrorCodeNoActiveSession, "no active session")
	ErrPermissionDenied     = NewError(ErrorCodePermissionDenied, "permission denied")

	// Connection errors
	ErrConnectionFailed  = NewError(ErrorCodeConnectionFailed, "failed to connect to ubus")
	ErrConnectionTimeout = NewError(ErrorCodeConnectionTimeout, "connection timeout")

	// Data errors
	ErrInvalidData      = NewError(ErrorCodeInvalidData, "invalid data format")
	ErrDataParsingError = NewError(ErrorCodeDataParsingError, "failed to parse response data")
	ErrInvalidResponse  = NewError(ErrorCodeInvalidResponse, "invalid response format")

	// Service errors
	ErrServiceNotFound = NewError(ErrorCodeServiceNotFound, "ubus service not found")
	ErrMethodNotFound  = NewError(ErrorCodeMethodNotFound, "ubus method not found")

	// Module errors
	ErrUbusModuleNotInstalled = NewError(ErrorCodeModuleNotInstalled, "ubus module not installed, try 'opkg update && opkg install uhttpd-mod-ubus && service uhttpd restart'")
	ErrFileModuleNotFound     = NewError(ErrorCodeModuleNotFound, "file module not found, try 'opkg update && opkg install rpcd-mod-file && service rpcd restart'")

	// UCI errors
	ErrUCIOperationFailed = NewError(ErrorCodeUCIOperationFailed, "UCI operation failed")

	// Generic errors
	ErrInternal     = NewError(ErrorCodeInternal, "internal error")
	ErrNotSupported = NewError(ErrorCodeNotSupported, "operation not supported")
	ErrTimeout      = NewError(ErrorCodeTimeout, "operation timeout")
)

// Error checking functions

// IsAuthenticationError checks if an error is authentication-related
func IsAuthenticationError(err error) bool {
	var ubusErr *UbusError
	if errors.As(err, &ubusErr) {
		return ubusErr.Code == ErrorCodeAuthenticationFailed ||
			ubusErr.Code == ErrorCodeSessionExpired ||
			ubusErr.Code == ErrorCodeNoActiveSession ||
			ubusErr.Code == ErrorCodePermissionDenied
	}
	return false
}

// IsConnectionError checks if an error is connection-related
func IsConnectionError(err error) bool {
	var ubusErr *UbusError
	if errors.As(err, &ubusErr) {
		return ubusErr.Code == ErrorCodeConnectionFailed ||
			ubusErr.Code == ErrorCodeConnectionTimeout ||
			ubusErr.Code == ErrorCodeNetworkError
	}
	return false
}

// IsDataError checks if an error is data-related
func IsDataError(err error) bool {
	var ubusErr *UbusError
	if errors.As(err, &ubusErr) {
		return ubusErr.Code == ErrorCodeInvalidData ||
			ubusErr.Code == ErrorCodeDataParsingError ||
			ubusErr.Code == ErrorCodeInvalidResponse ||
			ubusErr.Code == ErrorCodeUnexpectedFormat
	}
	return false
}

// IsServiceError checks if an error is service-related
func IsServiceError(err error) bool {
	var ubusErr *UbusError
	if errors.As(err, &ubusErr) {
		return ubusErr.Code == ErrorCodeServiceNotFound ||
			ubusErr.Code == ErrorCodeMethodNotFound ||
			ubusErr.Code == ErrorCodeInvalidArgument ||
			ubusErr.Code == ErrorCodeOperationFailed
	}
	return false
}

// IsRetryable checks if an error condition might be retryable
func IsRetryable(err error) bool {
	var ubusErr *UbusError
	if errors.As(err, &ubusErr) {
		return ubusErr.Code == ErrorCodeConnectionTimeout ||
			ubusErr.Code == ErrorCodeTimeout ||
			ubusErr.Code == ErrorCodeSessionExpired
	}
	return false
}

// Map ubus error codes to our error types
func mapUbusCodeToError(ubusCode int) (ErrorCode, string) {
	switch ubusCode {
	case 1:
		return ErrorCodeInvalidArgument, "Invalid command"
	case 2:
		return ErrorCodeInvalidArgument, "Invalid argument"
	case 3:
		return ErrorCodeMethodNotFound, "Method not found"
	case 4:
		return ErrorCodeServiceNotFound, "Not found"
	case 5:
		return ErrorCodeInvalidData, "No data"
	case 6:
		return ErrorCodePermissionDenied, "Permission denied"
	case 7:
		return ErrorCodeTimeout, "Timeout"
	case 8:
		return ErrorCodeNotSupported, "Not supported"
	case 9:
		return ErrorCodeUnknown, "Unknown error"
	case 10:
		return ErrorCodeConnectionFailed, "Connection failed"
	case -32000:
		return ErrorCodeInternal, "Server error"
	case -32001:
		return ErrorCodeServiceNotFound, "Object not found"
	case -32002:
		return ErrorCodeMethodNotFound, "Method not found"
	case -32003:
		return ErrorCodeInvalidArgument, "Invalid command"
	case -32004:
		return ErrorCodeInvalidArgument, "Invalid argument"
	case -32005:
		return ErrorCodeTimeout, "Request timeout"
	case -32006:
		return ErrorCodePermissionDenied, "Access denied"
	case -32007:
		return ErrorCodeConnectionFailed, "Connection failed"
	case -32008:
		return ErrorCodeInvalidData, "No data"
	case -32009:
		return ErrorCodePermissionDenied, "Operation not permitted"
	case -32010:
		return ErrorCodeServiceNotFound, "Not found"
	case -32011:
		return ErrorCodeInternal, "Out of memory"
	case -32012:
		return ErrorCodeNotSupported, "Not supported"
	case -32013:
		return ErrorCodeUnknown, "Unknown error"
	case -32014:
		return ErrorCodeConnectionTimeout, "Connection timed out"
	case -32015:
		return ErrorCodeConnectionFailed, "Connection closed"
	case -32016:
		return ErrorCodeInternal, "System error"
	default:
		return ErrorCodeUnknown, fmt.Sprintf("Unknown error code: %d", ubusCode)
	}
}

// Helper functions for backward compatibility and ease of use

// WrapError wraps a generic error in a UbusError
func WrapError(err error, code ErrorCode, message string) *UbusError {
	return NewErrorWithCause(code, message, err)
}

// GetErrorCode extracts the error code from an error
func GetErrorCode(err error) ErrorCode {
	var ubusErr *UbusError
	if errors.As(err, &ubusErr) {
		return ubusErr.Code
	}
	return ErrorCodeUnknown
}

// GetUbusCode extracts the original ubus error code from an error
func GetUbusCode(err error) int {
	var ubusErr *UbusError
	if errors.As(err, &ubusErr) {
		return ubusErr.UbusCode
	}
	return 0
}
