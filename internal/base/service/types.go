// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package service

import (
	"github.com/honeybbq/goubus/v2"
)

// Info represents information about a service.
type Info struct {
	Instances map[string]Instance `json:"instances"`
}

// Instance represents a service instance.
type Instance struct {
	Command []string    `json:"command"`
	Pid     int         `json:"pid"`
	Running goubus.Bool `json:"running"`
}

// Respawn holds respawn configuration.
type Respawn struct {
	Threshold int `json:"threshold"`
	Timeout   int `json:"timeout"`
	Retry     int `json:"retry"`
}

// Jail holds sandboxing configuration.
type Jail struct {
	Name string `json:"name"`
}

// Limits represents resource limits.
type Limits struct {
	NoFile string `json:"nofile,omitempty"`
}

// SetRequest represents parameters for setting up a service.
type SetRequest struct {
	Instances map[string]any `json:"instances,omitempty"`
	Data      map[string]any `json:"data,omitempty"`
	Name      string         `json:"name"`
	Script    string         `json:"script,omitempty"`
	Triggers  []any          `json:"triggers,omitempty"`
	Validate  []any          `json:"validate,omitempty"`
	Autostart goubus.Bool    `json:"autostart,omitempty"`
}

// EventRequest represents parameters for a service event.
type EventRequest struct {
	Data map[string]any `json:"data"`
	Type string         `json:"type"`
}

// ValidateRequest represents parameters for service validation.
type ValidateRequest struct {
	Package string `json:"package"`
	Type    string `json:"type"`
	Service string `json:"service"`
}
