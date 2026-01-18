// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package errdefs

import (
	"errors"
	"fmt"
)

// Common error types.
var (
	// ErrInvalidCommand represents an invalid command error.
	ErrInvalidCommand = errors.New("invalid command")
	// ErrInvalidParameter represents an invalid parameter error.
	ErrInvalidParameter = errors.New("invalid parameter")
	// ErrMethodNotFound represents a method not found error.
	ErrMethodNotFound = errors.New("method not found")
	// ErrNotFound represents a not found error.
	ErrNotFound = errors.New("not found")
	// ErrNoData represents a no data error.
	ErrNoData = errors.New("no data")
	// ErrPermissionDenied represents a permission denied error.
	ErrPermissionDenied = errors.New("permission denied")
	// ErrTimeout represents a timeout error.
	ErrTimeout = errors.New("timeout")
	// ErrNotSupported represents a not supported error.
	ErrNotSupported = errors.New("not supported")
	// ErrUnknown represents an unknown error.
	ErrUnknown = errors.New("unknown error")
	// ErrConnectionFailed represents a connection failed error.
	ErrConnectionFailed = errors.New("connection failed")
	// ErrClosed represents a client closed error.
	ErrClosed = errors.New("client closed")

	// ErrInvalidResponse represents an invalid response error.
	ErrInvalidResponse = errors.New("invalid response")
	// ErrTestSkipped represents a test skipped error.
	ErrTestSkipped = errors.New("test skipped")

	// ErrNotUnixSocket represents an error when the path is not a unix socket.
	ErrNotUnixSocket = errors.New("not a unix socket")
	// ErrUnsupportedAttributeType represents an unsupported attribute value type error.
	ErrUnsupportedAttributeType = errors.New("unsupported attribute value type")
	// ErrInvalidBlobLength represents an invalid blob length error.
	ErrInvalidBlobLength = errors.New("invalid blob length")
	// ErrArrayEntryNotExtended represents an error when an array entry is not extended.
	ErrArrayEntryNotExtended = errors.New("array entry not extended")
	// ErrTableEntryNotExtended represents an error when a table entry is not extended.
	ErrTableEntryNotExtended = errors.New("table entry not extended")
	// ErrBlobmsgPayloadTooShort represents an error when a blobmsg payload is too short.
	ErrBlobmsgPayloadTooShort = errors.New("blobmsg payload too short")
	// ErrInvalidBlobmsgHeaderLength represents an error when a blobmsg header length is invalid.
	ErrInvalidBlobmsgHeaderLength = errors.New("invalid blobmsg header length")
)

// IsInvalidCommand checks if err is ErrInvalidCommand.
func IsInvalidCommand(err error) bool {
	return errors.Is(err, ErrInvalidCommand)
}

// IsInvalidParameter checks if err is ErrInvalidParameter.
func IsInvalidParameter(err error) bool {
	return errors.Is(err, ErrInvalidParameter)
}

// IsMethodNotFound checks if err is ErrMethodNotFound.
func IsMethodNotFound(err error) bool {
	return errors.Is(err, ErrMethodNotFound)
}

// IsNotFound checks if err is ErrNotFound.
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IsNoData checks if err is ErrNoData.
func IsNoData(err error) bool {
	return errors.Is(err, ErrNoData)
}

// IsPermissionDenied checks if err is ErrPermissionDenied.
func IsPermissionDenied(err error) bool {
	return errors.Is(err, ErrPermissionDenied)
}

// IsTimeout checks if err is ErrTimeout.
func IsTimeout(err error) bool {
	return errors.Is(err, ErrTimeout)
}

// IsNotSupported checks if err is ErrNotSupported.
func IsNotSupported(err error) bool {
	return errors.Is(err, ErrNotSupported)
}

// IsUnknown checks if err is ErrUnknown.
func IsUnknown(err error) bool {
	return errors.Is(err, ErrUnknown)
}

// IsConnectionFailed checks if err is ErrConnectionFailed.
func IsConnectionFailed(err error) bool {
	return errors.Is(err, ErrConnectionFailed)
}

// IsInvalidResponse checks if err is ErrInvalidResponse.
func IsInvalidResponse(err error) bool {
	return errors.Is(err, ErrInvalidResponse)
}

// IsTestSkipped checks if err is ErrTestSkipped.
func IsTestSkipped(err error) bool {
	return errors.Is(err, ErrTestSkipped)
}

// Wrapf wraps an error with a formatting message.
func Wrapf(err error, format string, a ...any) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("%s: %w", fmt.Sprintf(format, a...), err)
}
