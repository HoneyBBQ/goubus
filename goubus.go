// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package goubus

import (
	"context"
	"log/slog"
)

// Transport is the interface that wraps the basic ubus call method.
// Transport provides a unified way to interact with local ubus sockets and remote JSON-RPC endpoints.
type Transport interface {
	// Call executes a ubus call for the specified service and method.
	// The data parameter is the payload for the call, which will be marshaled to JSON/blobmsg.
	Call(ctx context.Context, service, method string, data any) (Result, error)
	// SetLogger configures the logger used by the transport.
	SetLogger(logger *slog.Logger)
	// Close releases any resources held by the transport (e.g., closing connections).
	Close() error
}

// Result is the interface that wraps the basic Unmarshal method.
// It allows for lazy unmarshaling of ubus call responses into specific Go types.
type Result interface {
	// Unmarshal decodes the response data into the provided target.
	// The target must be a pointer to a compatible type.
	Unmarshal(target any) error
}

// Call is a generic helper that wraps Transport.Call and unmarshals the response.
// T represents the expected type of the response data.
func Call[T any](ctx context.Context, t Transport, service, method string, data any) (*T, error) {
	resp, err := t.Call(ctx, service, method, data)
	if err != nil {
		return nil, err
	}

	var target T

	err = resp.Unmarshal(&target)
	if err != nil {
		return nil, err
	}

	return &target, nil
}
