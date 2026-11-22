package rpc

import "time"

// SessionData holds authentication session information.
type SessionData struct {
	ExpireTime     time.Time `json:"-"`
	UbusRPCSession string    `json:"ubus_rpc_session"`
	Timeout        int       `json:"timeout"`
}

// UbusJSONRPCError represents an error in a JSON-RPC response.
type UbusJSONRPCError struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// UbusResponse represents a response from the ubus JSON-RPC interface.
type UbusResponse struct {
	Result  any                `json:"result,omitempty"`
	Error   *UbusJSONRPCError  `json:"error,omitempty"`
	Jsonrpc string             `json:"jsonrpc"`
	ID      int                `json:"id"`
}
