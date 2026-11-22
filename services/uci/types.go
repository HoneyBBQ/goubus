package uci

import "github.com/honeybbq/goubus/v2"

// RequestGeneric contains common UCI request parameters.
type RequestGeneric struct {
	// Config is the configuration package name.
	Config string `json:"config"`
	// Section selects a configuration section.
	Section string `json:"section,omitempty"`
	// Option selects a section option.
	Option string `json:"option,omitempty"`
	// Type selects a section type.
	Type string `json:"type,omitempty"`
	// Match contains a matching expression.
	Match string `json:"match,omitempty"`
	// Name is a section or option name.
	Name string `json:"name,omitempty"`
}

// Request contains common UCI parameters and values.
type Request struct {
	// Values contains option values to write.
	Values map[string]any `json:"values,omitempty"`
	RequestGeneric
}

// GetRequest contains parameters for UCI get operations.
type GetRequest struct {
	RequestGeneric
}

// GetResponse contains values returned by UCI get or state operations.
type GetResponse struct {
	// Values contains section and option values.
	Values map[string]any `json:"values"`
	// Value contains a scalar or list option value.
	Value goubus.StringOrSlice `json:"value"`
}

// ConfigsResponse contains available configuration names.
type ConfigsResponse struct {
	// Configs lists configuration package names.
	Configs []string `json:"configs"`
}

// StateRequest contains parameters for UCI state operations.
type StateRequest struct {
	RequestGeneric
}

// RenameRequest contains parameters for renaming a section or option.
type RenameRequest struct {
	// Config is the configuration package name.
	Config string `json:"config"`
	// Section selects a section to rename.
	Section string `json:"section,omitempty"`
	// Option selects an option to rename.
	Option string `json:"option,omitempty"`
	// Name is the new section or option name.
	Name string `json:"name"`
}

// OrderRequest contains a package's desired section order.
type OrderRequest struct {
	// Config is the configuration package name.
	Config string `json:"config"`
	// Sections lists sections in their desired order.
	Sections []string `json:"sections"`
}

// ChangesRequest selects a package for staged changes.
type ChangesRequest struct {
	// Config is the configuration package name.
	Config string `json:"config"`
}

// ChangesResponse contains staged UCI changes.
type ChangesResponse struct {
	// Changes maps configuration changes by section.
	Changes map[string]any `json:"changes"`
}

// RevertRequest selects a package to revert.
type RevertRequest struct {
	// Config is the configuration package name.
	Config string `json:"config"`
}

// ApplyRequest contains parameters for activating staged changes.
type ApplyRequest struct {
	// Rollback enables automatic rollback after timeout.
	Rollback goubus.Bool `json:"rollback,omitempty"`
	// Timeout is the automatic rollback timeout in seconds.
	Timeout int `json:"timeout,omitempty"`
}

// Metadata contains read-only UCI section metadata.
type Metadata struct {
	// Index is the section ordering index.
	Index *int `json:".index"`
	// Type is the UCI section type.
	Type string `json:".type"`
	// Name is the UCI section name.
	Name string `json:".name"`
	// Anonymous reports whether the section has an anonymous name.
	Anonymous goubus.Bool `json:".anonymous"`
}
