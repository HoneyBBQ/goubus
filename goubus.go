package goubus

import (
	"bytes"
	"context"
	"encoding/json"
	"slices"

	"github.com/honeybbq/goubus/v2/internal/errs"
)

// Caller executes a single ubus method call.
type Caller interface {
	// Call invokes one service method and returns its un-decoded result.
	Call(ctx context.Context, service, method string, data any) (Result, error)
}

// CallerFunc adapts a function to the Caller interface.
type CallerFunc func(ctx context.Context, service, method string, data any) (Result, error)

// Call invokes the adapted function.
func (f CallerFunc) Call(ctx context.Context, service, method string, data any) (Result, error) {
	return f(ctx, service, method, data)
}

// CallMiddleware wraps a Caller with call-specific behavior.
type CallMiddleware func(next Caller) Caller

// ChainMiddleware applies middleware from first to last around caller.
func ChainMiddleware(caller Caller, middlewares ...CallMiddleware) Caller {
	for _, middleware := range slices.Backward(middlewares) {
		if middleware != nil {
			caller = middleware(caller)
		}
	}

	return caller
}

// Transport combines a Caller with resource cleanup.
type Transport interface {
	Caller
	// Close releases resources owned by the transport.
	Close() error
}

// Result decodes the response data into a target value.
type Result interface {
	// Unmarshal decodes the response data into target.
	Unmarshal(target any) error
}

type jsonResult struct {
	raw json.RawMessage
}

// NewJSONResult creates a Result that decodes raw JSON response data.
func NewJSONResult(raw json.RawMessage) Result {
	return jsonResult{raw: append(json.RawMessage(nil), raw...)}
}

// Unmarshal decodes a JSON response into target.
func (r jsonResult) Unmarshal(target any) error {
	trimmed := bytes.TrimSpace(r.raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return ErrNoData
	}

	decodeErr := json.Unmarshal(trimmed, target)
	if decodeErr != nil {
		return errs.Wrapf(ErrInvalidResponse, "decode JSON result: %v", decodeErr)
	}

	return nil
}

// Call invokes caller and decodes its result into T.
func Call[T any](ctx context.Context, caller Caller, service, method string, data any) (T, error) {
	var target T

	resp, err := caller.Call(ctx, service, method, data)
	if err != nil {
		return target, err
	}

	err = resp.Unmarshal(&target)
	if err != nil {
		var zero T

		return zero, err
	}

	return target, nil
}
