// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package uci

import "github.com/honeybbq/goubus/v2"

// RequestGeneric represents the basic UCI request structure.
type RequestGeneric struct {
	Config  string `json:"config"`
	Section string `json:"section,omitempty"`
	Option  string `json:"option,omitempty"`
	Type    string `json:"type,omitempty"`
	Match   string `json:"match,omitempty"`
	Name    string `json:"name,omitempty"`
}

// Request represents a UCI request with values.
type Request struct {
	Values map[string]any `json:"values,omitempty"`
	RequestGeneric
}

// GetRequest represents a UCI get request.
type GetRequest struct {
	RequestGeneric
}

// GetResponse holds the response from a uci.get call.
type GetResponse struct {
	Values map[string]any `json:"values"`
	Value  string         `json:"value"`
}

// ConfigsResponse holds the response from a uci.configs call.
type ConfigsResponse struct {
	Configs []string `json:"configs"`
}

// StateRequest represents a UCI state request.
type StateRequest struct {
	RequestGeneric
}

// StateResponse represents a UCI state response.
type StateResponse struct {
	Values map[string]string `json:"values"`
	Value  string            `json:"value"`
}

// RenameRequest represents a UCI rename request.
type RenameRequest struct {
	Config  string `json:"config"`
	Section string `json:"section,omitempty"`
	Option  string `json:"option,omitempty"`
	Name    string `json:"name"`
}

// OrderRequest represents a UCI order request.
type OrderRequest struct {
	Config   string   `json:"config"`
	Sections []string `json:"sections"`
}

// ChangesRequest represents a UCI changes request.
type ChangesRequest struct {
	Config string `json:"config"`
}

// ChangesResponse holds the response from a uci.changes call.
type ChangesResponse struct {
	Changes map[string]any `json:"changes"`
}

// RevertRequest represents a UCI revert request.
type RevertRequest struct {
	Config string `json:"config"`
}

// ApplyRequest represents a UCI apply request.
type ApplyRequest struct {
	Rollback goubus.Bool `json:"rollback,omitempty"`
	Timeout  int         `json:"timeout,omitempty"`
}

// Metadata holds the read-only metadata associated with a UCI section.
type Metadata struct {
	Index     *int        `json:".index"`
	Type      string      `json:".type"`
	Name      string      `json:".name"`
	Anonymous goubus.Bool `json:".anonymous"`
}
