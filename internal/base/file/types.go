// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package file

// List represents directory listing.
type List struct {
	Entries []ListData `json:"entries"`
}

// ListData represents a single entry.
type ListData struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Size int    `json:"size"`
}

// Stat represents file statistics.
type Stat struct {
	Path string `json:"path"`
	Type string `json:"type"`
	Size int    `json:"size"`
}

// Read represents file content.
type Read struct {
	Data string `json:"data"`
}

// Exec represents command output.
type Exec struct {
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`
	Code   int    `json:"code"`
}
