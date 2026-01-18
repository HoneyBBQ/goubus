// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package goubus

import "github.com/honeybbq/goubus/v2/errdefs"

// Ubus error codes.
const (
	UbusStatusOK               = 0
	UbusStatusInvalidCommand   = 1
	UbusStatusInvalidParameter = 2
	UbusStatusMethodNotFound   = 3
	UbusStatusNotFound         = 4
	UbusStatusNoData           = 5
	UbusStatusPermissionDenied = 6
	UbusStatusTimeout          = 7
	UbusStatusNotSupported     = 8
	UbusStatusUnknown          = 9
	UbusStatusConnectionFailed = 10
)

var ubusErrorMap = map[int]error{
	UbusStatusOK:               nil,
	UbusStatusInvalidCommand:   errdefs.ErrInvalidCommand,
	UbusStatusInvalidParameter: errdefs.ErrInvalidParameter,
	UbusStatusMethodNotFound:   errdefs.ErrMethodNotFound,
	UbusStatusNotFound:         errdefs.ErrNotFound,
	UbusStatusNoData:           errdefs.ErrNoData,
	UbusStatusPermissionDenied: errdefs.ErrPermissionDenied,
	UbusStatusTimeout:          errdefs.ErrTimeout,
	UbusStatusNotSupported:     errdefs.ErrNotSupported,
	UbusStatusUnknown:          errdefs.ErrUnknown,
	UbusStatusConnectionFailed: errdefs.ErrConnectionFailed,
}

// MapUbusCodeToError maps a ubus integer code to a typed error defined in errdefs.
// If the code is not recognized, it returns an unknown error wrapping the code.
func MapUbusCodeToError(code int) error {
	if err, ok := ubusErrorMap[code]; ok {
		return err
	}

	return errdefs.Wrapf(errdefs.ErrUnknown, "unknown ubus error code: %d", code)
}
