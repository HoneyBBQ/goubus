package goubus

import (
	"errors"

	"github.com/honeybbq/goubus/v2/internal/errs"
)

const (
	// StatusOK indicates that the ubus operation succeeded.
	StatusOK = 0
	// StatusInvalidCommand indicates an invalid ubus command.
	StatusInvalidCommand = 1
	// StatusInvalidParameter indicates an invalid ubus parameter.
	StatusInvalidParameter = 2
	// StatusMethodNotFound indicates an unknown ubus method.
	StatusMethodNotFound = 3
	// StatusNotFound indicates that the requested object or resource was not found.
	StatusNotFound = 4
	// StatusNoData indicates that the operation returned no data.
	StatusNoData = 5
	// StatusPermissionDenied indicates that the operation was denied.
	StatusPermissionDenied = 6
	// StatusTimeout indicates that the operation timed out.
	StatusTimeout = 7
	// StatusNotSupported indicates that the operation is unsupported.
	StatusNotSupported = 8
	// StatusUnknown indicates an unclassified ubus error.
	StatusUnknown = 9
	// StatusConnectionFailed indicates that the connection failed.
	StatusConnectionFailed = 10
	// StatusNoMemory indicates an out-of-memory condition in ubusd or a target daemon.
	StatusNoMemory = 11
	// StatusParseError indicates a message data parse failure in ubus.
	StatusParseError = 12
	// StatusSystemError indicates an OS or daemon-level system error in ubus.
	StatusSystemError = 13
)

var statusErrors = [...]error{
	StatusOK:               nil,
	StatusInvalidCommand:   ErrInvalidCommand,
	StatusInvalidParameter: ErrInvalidParameter,
	StatusMethodNotFound:   ErrMethodNotFound,
	StatusNotFound:         ErrNotFound,
	StatusNoData:           ErrNoData,
	StatusPermissionDenied: ErrPermissionDenied,
	StatusTimeout:          ErrTimeout,
	StatusNotSupported:     ErrNotSupported,
	StatusUnknown:          ErrUnknown,
	StatusConnectionFailed: ErrConnectionFailed,
	StatusNoMemory:         ErrNoMemory,
	StatusParseError:       ErrParseError,
	StatusSystemError:      ErrSystemError,
}

// StatusError maps a ubus status code to its sentinel error.
func StatusError(code int) error {
	if code >= 0 && code < len(statusErrors) {
		return statusErrors[code]
	}

	return errs.Wrapf(ErrUnknown, "unknown ubus error code: %d", code)
}

func lookupStatusError(err error) (int, bool, error) {
	for code := StatusInvalidCommand; code < len(statusErrors); code++ {
		sentinel := statusErrors[code]
		if sentinel != nil && errors.Is(err, sentinel) {
			return code, true, sentinel
		}
	}

	return 0, false, nil
}

func wireStatusFromError(err error) uint32 {
	if err == nil {
		return StatusOK
	}
	if callErr, ok := errors.AsType[*CallError](err); ok {
		if callErr.Code > StatusOK && callErr.Code <= StatusSystemError {
			return uint32(callErr.Code)
		}
	}
	if code, ok, _ := lookupStatusError(err); ok {
		if code > StatusOK && code <= StatusSystemError {
			return uint32(code)
		}
	}

	return StatusUnknown
}
