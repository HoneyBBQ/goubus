// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package session

import "time"

// Data represents session information.
type Data struct {
	ExpireTime     time.Time `json:"-"`
	UbusRPCSession string    `json:"ubus_rpc_session"`
	Timeout        int       `json:"timeout"`
}

// ACLs represents access control lists.
type ACLs struct {
	Ubus map[string][]string `json:"ubus"`
	Uci  map[string][]string `json:"uci"`
}

// GrantRequest represents parameters for granting session access.
type GrantRequest struct {
	Session string   `json:"ubus_rpc_session"`
	Scope   string   `json:"scope"`
	Objects []string `json:"objects"`
}

// AccessRequest represents parameters for checking session access.
type AccessRequest struct {
	Session  string `json:"ubus_rpc_session"`
	Scope    string `json:"scope"`
	Object   string `json:"object"`
	Function string `json:"function"`
}

// LoginRequest represents parameters for session login.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Timeout  int    `json:"timeout,omitempty"`
}
