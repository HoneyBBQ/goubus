// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package system

import (
	"github.com/honeybbq/goubus/v2"
)

// Info holds runtime system information.
type Info struct {
	Load      []int   `json:"load"`
	Memory    Memory  `json:"memory"`
	Root      Storage `json:"root"`
	Tmp       Storage `json:"tmp"`
	Swap      Swap    `json:"swap"`
	LocalTime int64   `json:"localtime"`
	Uptime    int     `json:"uptime"`
}

// BoardInfo holds hardware-specific information.
type BoardInfo struct {
	Kernel    string  `json:"kernel"`
	Hostname  string  `json:"hostname"`
	System    string  `json:"system"`
	Model     string  `json:"model"`
	BoardName string  `json:"board_name"`
	Release   Release `json:"release"`
}

// Release holds release information.
type Release struct {
	Distribution string `json:"distribution"`
	Version      string `json:"version"`
	Revision     string `json:"revision"`
	Target       string `json:"target"`
	Description  string `json:"description"`
}

// Memory holds memory usage statistics.
type Memory struct {
	Total     int `json:"total"`
	Free      int `json:"free"`
	Available int `json:"available"`
	Cached    int `json:"cached"`
}

// Storage holds storage usage statistics.
type Storage struct {
	Total int `json:"total"`
	Free  int `json:"free"`
	Used  int `json:"used"`
}

// Swap holds swap usage statistics.
type Swap struct {
	Total int `json:"total"`
	Free  int `json:"free"`
}

// WatchdogRequest represents parameters for system watchdog.
type WatchdogRequest struct {
	Frequency  int         `json:"frequency,omitempty"`
	Timeout    int         `json:"timeout,omitempty"`
	MagicClose goubus.Bool `json:"magicclose,omitempty"`
	Stop       goubus.Bool `json:"stop,omitempty"`
}

// SignalRequest represents parameters for sending a signal.
type SignalRequest struct {
	Pid    int `json:"pid"`
	Signum int `json:"signum"`
}

// ValidateFirmwareImageRequest represents parameters for firmware validation.
type ValidateFirmwareImageRequest struct {
	Path string `json:"path"`
}

// SysupgradeRequest represents parameters for system upgrade.
type SysupgradeRequest struct {
	Options map[string]any `json:"options,omitempty"`
	Path    string         `json:"path"`
	Backup  string         `json:"backup,omitempty"`
	Prefix  string         `json:"prefix,omitempty"`
	Command string         `json:"command,omitempty"`
	Force   goubus.Bool    `json:"force,omitempty"`
}
