// Package errdefs preserves the legacy names for goubus sentinel errors.
package errdefs

import (
	"errors"

	"github.com/honeybbq/goubus/v2/internal/errs"
)

// ErrCall and the other sentinels alias the shared internal error identities.
var (
	ErrCall                       = errs.ErrCall
	ErrInvalidCommand             = errs.ErrInvalidCommand
	ErrInvalidParameter           = errs.ErrInvalidParameter
	ErrMethodNotFound             = errs.ErrMethodNotFound
	ErrNotFound                   = errs.ErrNotFound
	ErrNoData                     = errs.ErrNoData
	ErrPermissionDenied           = errs.ErrPermissionDenied
	ErrTimeout                    = errs.ErrTimeout
	ErrNotSupported               = errs.ErrNotSupported
	ErrUnknown                    = errs.ErrUnknown
	ErrConnectionFailed           = errs.ErrConnectionFailed
	ErrClosed                     = errs.ErrClosed
	ErrInvalidResponse            = errs.ErrInvalidResponse
	ErrNotUnixSocket              = errs.ErrNotUnixSocket
	ErrUnsupportedAttributeType   = errs.ErrUnsupportedAttributeType
	ErrInvalidBlobLength          = errs.ErrInvalidBlobLength
	ErrArrayEntryNotExtended      = errs.ErrArrayEntryNotExtended
	ErrTableEntryNotExtended      = errs.ErrTableEntryNotExtended
	ErrBlobmsgPayloadTooShort     = errs.ErrBlobmsgPayloadTooShort
	ErrInvalidBlobmsgHeaderLength = errs.ErrInvalidBlobmsgHeaderLength
	ErrTestSkipped                = errors.New("test skipped")
)

// IsInvalidCommand reports whether err is ErrInvalidCommand.
func IsInvalidCommand(err error) bool { return errors.Is(err, ErrInvalidCommand) }

// IsInvalidParameter reports whether err is ErrInvalidParameter.
func IsInvalidParameter(err error) bool { return errors.Is(err, ErrInvalidParameter) }

// IsMethodNotFound reports whether err is ErrMethodNotFound.
func IsMethodNotFound(err error) bool { return errors.Is(err, ErrMethodNotFound) }

// IsNotFound reports whether err is ErrNotFound.
func IsNotFound(err error) bool { return errors.Is(err, ErrNotFound) }

// IsNoData reports whether err is ErrNoData.
func IsNoData(err error) bool { return errors.Is(err, ErrNoData) }

// IsPermissionDenied reports whether err is ErrPermissionDenied.
func IsPermissionDenied(err error) bool { return errors.Is(err, ErrPermissionDenied) }

// IsTimeout reports whether err is ErrTimeout.
func IsTimeout(err error) bool { return errors.Is(err, ErrTimeout) }

// IsNotSupported reports whether err is ErrNotSupported.
func IsNotSupported(err error) bool { return errors.Is(err, ErrNotSupported) }

// IsUnknown reports whether err is ErrUnknown.
func IsUnknown(err error) bool { return errors.Is(err, ErrUnknown) }

// IsConnectionFailed reports whether err is ErrConnectionFailed.
func IsConnectionFailed(err error) bool { return errors.Is(err, ErrConnectionFailed) }

// IsInvalidResponse reports whether err is ErrInvalidResponse.
func IsInvalidResponse(err error) bool { return errors.Is(err, ErrInvalidResponse) }

// IsTestSkipped reports whether err is ErrTestSkipped.
func IsTestSkipped(err error) bool { return errors.Is(err, ErrTestSkipped) }

// Wrapf wraps err with a formatted message.
func Wrapf(err error, format string, a ...any) error {
	return errs.Wrapf(err, format, a...)
}
