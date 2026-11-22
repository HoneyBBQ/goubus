package uci

import "github.com/honeybbq/goubus/v2"

// requestGeneric contains common UCI request parameters.
type requestGeneric struct {
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

// writeRequest contains common UCI parameters and values.
type writeRequest struct {
	// Values contains option values to write.
	Values map[string]any `json:"values,omitempty"`
	requestGeneric
}

// getRequest contains parameters for UCI get operations.
type getRequest struct {
	requestGeneric
}

// GetResponse contains values returned by UCI get or state operations.
type GetResponse struct {
	// Values contains section and option values.
	Values map[string]any `json:"values"`
	// Value contains a scalar or list option value.
	Value goubus.StringOrSlice `json:"value"`
}

// configsResponse contains available configuration names.
type configsResponse struct {
	// Configs lists configuration package names.
	Configs []string `json:"configs"`
}

// StateRequest contains parameters for UCI state operations.
type StateRequest struct {
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

// renameRequest contains parameters for renaming a section or option.
type renameRequest struct {
	// Config is the configuration package name.
	Config string `json:"config"`
	// Section selects a section to rename.
	Section string `json:"section,omitempty"`
	// Option selects an option to rename.
	Option string `json:"option,omitempty"`
	// Name is the new section or option name.
	Name string `json:"name"`
}

// orderRequest contains a package's desired section order.
type orderRequest struct {
	// Config is the configuration package name.
	Config string `json:"config"`
	// Sections lists sections in their desired order.
	Sections []string `json:"sections"`
}

// changesRequest selects a package for staged changes.
type changesRequest struct {
	// Config is the configuration package name.
	Config string `json:"config"`
}

// ChangesResponse contains staged UCI changes.
type ChangesResponse struct {
	// Changes maps configuration changes by section.
	Changes map[string]any `json:"changes"`
}

// revertRequest selects a package to revert.
type revertRequest struct {
	// Config is the configuration package name.
	Config string `json:"config"`
}

// applyRequest contains parameters for activating staged changes.
type applyRequest struct {
	// Timeout is the automatic rollback timeout in seconds.
	Timeout int `json:"timeout,omitempty"`
	// Rollback enables automatic rollback after timeout.
	Rollback goubus.Bool `json:"rollback,omitempty"`
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
