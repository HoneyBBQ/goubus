// Package errs owns shared sentinel errors for the protocol and its internal layers.
package errs

import (
	"errors"
	"fmt"
)

// ErrCall and the other sentinels classify errors shared by goubus packages.
var (
	ErrCall                       = errors.New("ubus call failed")
	ErrInvalidCommand             = errors.New("invalid command")
	ErrInvalidParameter           = errors.New("invalid parameter")
	ErrMethodNotFound             = errors.New("method not found")
	ErrNotFound                   = errors.New("not found")
	ErrNoData                     = errors.New("no data")
	ErrPermissionDenied           = errors.New("permission denied")
	ErrTimeout                    = errors.New("timeout")
	ErrNotSupported               = errors.New("not supported")
	ErrUnknown                    = errors.New("unknown error")
	ErrConnectionFailed           = errors.New("connection failed")
	ErrClosed                     = errors.New("client closed")
	ErrInvalidResponse            = errors.New("invalid response")
	ErrNotUnixSocket              = errors.New("not a unix socket")
	ErrUnsupportedAttributeType   = errors.New("unsupported attribute value type")
	ErrInvalidBlobLength          = errors.New("invalid blob length")
	ErrArrayEntryNotExtended      = errors.New("array entry not extended")
	ErrTableEntryNotExtended      = errors.New("table entry not extended")
	ErrBlobmsgPayloadTooShort     = errors.New("blobmsg payload too short")
	ErrInvalidBlobmsgHeaderLength = errors.New("invalid blobmsg header length")
)

// Wrapf adds formatted context while preserving error identity.
func Wrapf(err error, format string, a ...any) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("%s: %w", fmt.Sprintf(format, a...), err)
}
