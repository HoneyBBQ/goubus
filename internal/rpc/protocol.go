// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package rpc

import (
	"encoding/json"
	"time"

	"github.com/honeybbq/goubus/v2/errdefs"
)

// SessionData holds authentication session information.
type SessionData struct {
	ExpireTime     time.Time `json:"-"`
	UbusRPCSession string    `json:"ubus_rpc_session"`
	Timeout        int       `json:"timeout"`
}

// UbusJsonRpcError represents the error structure in a JSON-RPC response.
type UbusJsonRpcError struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// UbusResponse represents a response from the ubus RPC interface.
type UbusResponse struct {
	Result  any               `json:"result,omitempty"`
	Error   *UbusJsonRpcError `json:"error,omitempty"`
	Jsonrpc string            `json:"jsonrpc"`
	ID      int               `json:"id"`
}

type UbusResult []any

func (r UbusResult) Unmarshal(target any, mapErr func(int) error) error {
	const (
		ubusAuthResultCodeIndex  = 0
		ubusAuthResultDataIndex  = 1
		ubusAuthMinResultLength  = 1
		ubusAuthDataResultLength = 2
	)

	if len(r) < ubusAuthMinResultLength {
		return errdefs.ErrInvalidResponse
	}

	// Check the error code (first element)
	code, ok := r[ubusAuthResultCodeIndex].(float64)
	if !ok {
		return errdefs.Wrapf(errdefs.ErrInvalidResponse, "expected numeric error code, got %T", r[0])
	}

	// If there's an error code, map it to a typed error
	if code != 0 {
		return mapErr(int(code))
	}

	// If there's only one element and it's 0, it means success but no data
	if len(r) == ubusAuthMinResultLength {
		return errdefs.ErrNoData
	}

	// If there are 2+ elements, the second element contains the data
	if len(r) >= ubusAuthDataResultLength {
		// The actual data is the second element of the result array
		ubusDataByte, err := json.Marshal(r[ubusAuthResultDataIndex])
		if err != nil {
			return errdefs.Wrapf(errdefs.ErrInvalidResponse, "failed to marshal response data: %v", err)
		}

		err = json.Unmarshal(ubusDataByte, target)
		if err != nil {
			return errdefs.Wrapf(errdefs.ErrInvalidResponse, "failed to unmarshal response data: %v", err)
		}

		return nil
	}

	return errdefs.ErrInvalidResponse
}
