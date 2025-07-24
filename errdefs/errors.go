package errdefs

import (
	"errors"
	"fmt"
)

// Definitions for common error types used throughout the project.
// These errors correspond to ubus error codes and client-side errors.
var (
	// ubus-specific errors
	ErrInvalidCommand   = errors.New("invalid command")
	ErrInvalidParameter = errors.New("invalid parameter")
	ErrMethodNotFound   = errors.New("method not found")
	ErrNotFound         = errors.New("not found")
	ErrNoData           = errors.New("no data")
	ErrPermissionDenied = errors.New("permission denied")
	ErrTimeout          = errors.New("timeout")
	ErrNotSupported     = errors.New("not supported")
	ErrUnknown          = errors.New("unknown error")
	ErrConnectionFailed = errors.New("connection failed")
	ErrClosed           = errors.New("client closed")
	// client-side errors
	ErrInvalidResponse = errors.New("invalid response")
	ErrTestSkipped     = errors.New("test skipped")
)

// Typed error checking functions

func IsInvalidCommand(err error) bool {
	return errors.Is(err, ErrInvalidCommand)
}

func IsInvalidParameter(err error) bool {
	return errors.Is(err, ErrInvalidParameter)
}

func IsMethodNotFound(err error) bool {
	return errors.Is(err, ErrMethodNotFound)
}

func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

func IsNoData(err error) bool {
	return errors.Is(err, ErrNoData)
}

func IsPermissionDenied(err error) bool {
	return errors.Is(err, ErrPermissionDenied)
}

func IsTimeout(err error) bool {
	return errors.Is(err, ErrTimeout)
}

func IsNotSupported(err error) bool {
	return errors.Is(err, ErrNotSupported)
}

func IsUnknown(err error) bool {
	return errors.Is(err, ErrUnknown)
}

func IsConnectionFailed(err error) bool {
	return errors.Is(err, ErrConnectionFailed)
}

func IsInvalidResponse(err error) bool {
	return errors.Is(err, ErrInvalidResponse)
}

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
