package goubus

import "github.com/honeybbq/goubus/v2/internal/errs"

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

	// UbusStatusOK aliases StatusOK.
	UbusStatusOK = StatusOK
	// UbusStatusInvalidCommand aliases StatusInvalidCommand.
	UbusStatusInvalidCommand = StatusInvalidCommand
	// UbusStatusInvalidParameter aliases StatusInvalidParameter.
	UbusStatusInvalidParameter = StatusInvalidParameter
	// UbusStatusMethodNotFound aliases StatusMethodNotFound.
	UbusStatusMethodNotFound = StatusMethodNotFound
	// UbusStatusNotFound aliases StatusNotFound.
	UbusStatusNotFound = StatusNotFound
	// UbusStatusNoData aliases StatusNoData.
	UbusStatusNoData = StatusNoData
	// UbusStatusPermissionDenied aliases StatusPermissionDenied.
	UbusStatusPermissionDenied = StatusPermissionDenied
	// UbusStatusTimeout aliases StatusTimeout.
	UbusStatusTimeout = StatusTimeout
	// UbusStatusNotSupported aliases StatusNotSupported.
	UbusStatusNotSupported = StatusNotSupported
	// UbusStatusUnknown aliases StatusUnknown.
	UbusStatusUnknown = StatusUnknown
	// UbusStatusConnectionFailed aliases StatusConnectionFailed.
	UbusStatusConnectionFailed = StatusConnectionFailed
)

var ubusErrorMap = map[int]error{
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
}

// StatusError maps a ubus status code to its sentinel error.
func StatusError(code int) error {
	if err, ok := ubusErrorMap[code]; ok {
		return err
	}

	return errs.Wrapf(ErrUnknown, "unknown ubus error code: %d", code)
}

// MapUbusCodeToError maps a ubus status code to its sentinel error.
func MapUbusCodeToError(code int) error {
	return StatusError(code)
}
