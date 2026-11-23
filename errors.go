package goubus

import (
	"fmt"

	"github.com/honeybbq/goubus/v2/internal/errs"
)

// ErrCall marks a failed ubus call.
var (
	// ErrCall identifies a failed ubus call.
	ErrCall = errs.ErrCall
	// ErrInvalidCommand identifies an invalid ubus command.
	ErrInvalidCommand = errs.ErrInvalidCommand
	// ErrInvalidParameter identifies an invalid ubus parameter.
	ErrInvalidParameter = errs.ErrInvalidParameter
	// ErrMethodNotFound identifies an unknown ubus method.
	ErrMethodNotFound = errs.ErrMethodNotFound
	// ErrNotFound identifies a missing ubus object or resource.
	ErrNotFound = errs.ErrNotFound
	// ErrNoData identifies a response with no data.
	ErrNoData = errs.ErrNoData
	// ErrPermissionDenied identifies an operation denied by ubus.
	ErrPermissionDenied = errs.ErrPermissionDenied
	// ErrTimeout identifies an ubus operation timeout.
	ErrTimeout = errs.ErrTimeout
	// ErrNotSupported identifies an unsupported ubus operation.
	ErrNotSupported = errs.ErrNotSupported
	// ErrUnknown identifies an unclassified ubus error.
	ErrUnknown = errs.ErrUnknown
	// ErrConnectionFailed identifies a failed transport connection.
	ErrConnectionFailed = errs.ErrConnectionFailed
	// ErrClosed identifies use of a closed client.
	ErrClosed = errs.ErrClosed
	// ErrInvalidResponse identifies a malformed or unexpected response.
	ErrInvalidResponse = errs.ErrInvalidResponse
	// ErrNotUnixSocket identifies a path that is not a Unix socket.
	ErrNotUnixSocket = errs.ErrNotUnixSocket
)

// CallError reports a failed ubus method invocation.
type CallError struct {
	// Err is the underlying cause of the failed call.
	Err error
	// Service is the ubus service that was called.
	Service string
	// Method is the ubus method that was called.
	Method string
	// Message is the server-provided error message.
	Message string
	// Code is the ubus or JSON-RPC error code.
	Code int
}

// Error formats the failed ubus method invocation.
func (e *CallError) Error() string {
	if e.Service == "" {
		if e.Message != "" {
			return fmt.Sprintf("ubus %s failed (code %d: %s): %v", e.Method, e.Code, e.Message, e.Err)
		}

		return fmt.Sprintf("ubus %s failed (code %d): %v", e.Method, e.Code, e.Err)
	}
	if e.Message != "" {
		return fmt.Sprintf("ubus call %s.%s failed (code %d: %s): %v", e.Service, e.Method, e.Code, e.Message, e.Err)
	}

	return fmt.Sprintf("ubus call %s.%s failed (code %d): %v", e.Service, e.Method, e.Code, e.Err)
}

// Unwrap returns the call sentinel and, when present, the underlying cause.
func (e *CallError) Unwrap() []error {
	if e.Err != nil {
		return []error{ErrCall, e.Err}
	}

	return []error{ErrCall}
}
