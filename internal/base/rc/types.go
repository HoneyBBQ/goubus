// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package rc

import "github.com/honeybbq/goubus/v2"

// ListInfo represents init script status.
type ListInfo struct {
	Running goubus.Bool `json:"running"`
	Enabled goubus.Bool `json:"enabled"`
}

// InitRequest represents parameters for init script action.
type InitRequest struct {
	Name   string `json:"name"`
	Action string `json:"action"`
}
