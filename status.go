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
