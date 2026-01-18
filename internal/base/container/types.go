// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package container

import (
	"github.com/honeybbq/goubus/v2"
)

// SetRequest represents parameters for setting up or adding a container.
type SetRequest struct {
	Instances map[string]any `json:"instances,omitempty"`
	Data      map[string]any `json:"data,omitempty"`
	Name      string         `json:"name"`
	Script    string         `json:"script,omitempty"`
	Triggers  []any          `json:"triggers,omitempty"`
	Validate  []any          `json:"validate,omitempty"`
	Autostart goubus.Bool    `json:"autostart,omitempty"`
}
