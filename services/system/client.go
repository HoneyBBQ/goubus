// Package system provides clients for system-wide ubus methods.
package system

import (
	"context"

	"github.com/honeybbq/goubus/v2"
)

// Client provides access to system-wide ubus methods.
type Client struct {
	caller goubus.Caller
}

// New creates a system client for caller.
func New(caller goubus.Caller) *Client {
	if caller == nil {
		panic("goubus/system: nil caller")
	}

	return &Client{caller: caller}
}

// Info retrieves runtime system information.
func (c *Client) Info(ctx context.Context) (*Info, error) {
	return goubus.Call[Info](ctx, c.caller, "system", "info", nil)
}

// Board retrieves board hardware information.
func (c *Client) Board(ctx context.Context) (*BoardInfo, error) {
	return goubus.Call[BoardInfo](ctx, c.caller, "system", "board", nil)
}

// Reboot reboots the system.
func (c *Client) Reboot(ctx context.Context) error {
	_, err := c.caller.Call(ctx, "system", "reboot", nil)

	return err
}

// Watchdog configures the system watchdog.
func (c *Client) Watchdog(ctx context.Context, req WatchdogRequest) error {
	_, err := c.caller.Call(ctx, "system", "watchdog", req)

	return err
}

// Signal sends a signal to a process.
func (c *Client) Signal(ctx context.Context, pid, signum int) error {
	_, err := c.caller.Call(ctx, "system", "signal", SignalRequest{Pid: pid, Signum: signum})

	return err
}

// ValidateFirmwareImage validates a firmware image file.
func (c *Client) ValidateFirmwareImage(ctx context.Context, path string) (map[string]any, error) {
	request := ValidateFirmwareImageRequest{Path: path}
	res, err := goubus.Call[map[string]any](ctx, c.caller, "system", "validate_firmware_image", request)
	if err != nil {
		return nil, err
	}

	return *res, nil
}

// Sysupgrade performs a system upgrade.
func (c *Client) Sysupgrade(ctx context.Context, req SysupgradeRequest) error {
	_, err := c.caller.Call(ctx, "system", "sysupgrade", req)

	return err
}

// Info contains runtime system information.
type Info struct {
	// Load contains the one-, five-, and fifteen-minute load averages.
	Load []int `json:"load"`
	// Memory contains memory usage statistics.
	Memory Memory `json:"memory"`
	// Root contains root filesystem usage statistics.
	Root Storage `json:"root"`
	// Tmp contains temporary filesystem usage statistics.
	Tmp Storage `json:"tmp"`
	// Swap contains swap usage statistics.
	Swap Swap `json:"swap"`
	// LocalTime is the local time as a Unix timestamp.
	LocalTime int64 `json:"localtime"`
	// Uptime is the system uptime in seconds.
	Uptime int64 `json:"uptime"`
}

// BoardInfo contains hardware-specific information.
type BoardInfo struct {
	// Kernel is the kernel version.
	Kernel string `json:"kernel"`
	// Hostname is the system hostname.
	Hostname string `json:"hostname"`
	// System is the processor or system description.
	System string `json:"system"`
	// Model is the hardware model.
	Model string `json:"model"`
	// BoardName is the canonical board name.
	BoardName string `json:"board_name"`
	// RootfsType is the root filesystem type.
	RootfsType string `json:"rootfs_type"`
	// Release contains firmware release information.
	Release Release `json:"release"`
}

// Release contains firmware release information.
type Release struct {
	// Distribution is the distribution name.
	Distribution string `json:"distribution"`
	// Version is the distribution version.
	Version string `json:"version"`
	// Revision is the source revision.
	Revision string `json:"revision"`
	// Target is the target platform.
	Target string `json:"target"`
	// Description is the release description.
	Description string `json:"description"`
	// BuildDate is the firmware build date.
	BuildDate string `json:"builddate"`
	// FirmwareURL is the firmware download URL.
	FirmwareURL string `json:"firmware_url"`
}

// Memory contains memory usage statistics in bytes.
type Memory struct {
	// Total is the total memory in bytes.
	Total uint64 `json:"total"`
	// Free is the free memory in bytes.
	Free uint64 `json:"free"`
	// Shared is the shared memory in bytes.
	Shared uint64 `json:"shared"`
	// Buffered is the buffered memory in bytes.
	Buffered uint64 `json:"buffered"`
	// Available is the available memory in bytes.
	Available uint64 `json:"available"`
	// Cached is the cached memory in bytes.
	Cached uint64 `json:"cached"`
}

// Storage contains filesystem usage statistics in bytes.
type Storage struct {
	// Total is the total filesystem size in bytes.
	Total uint64 `json:"total"`
	// Free is the free filesystem space in bytes.
	Free uint64 `json:"free"`
	// Used is the used filesystem space in bytes.
	Used uint64 `json:"used"`
	// Avail is the available filesystem space in bytes.
	Avail uint64 `json:"avail"`
}

// Swap contains swap usage statistics in bytes.
type Swap struct {
	// Total is the total swap size in bytes.
	Total uint64 `json:"total"`
	// Free is the free swap space in bytes.
	Free uint64 `json:"free"`
}

// WatchdogRequest contains parameters for configuring the system watchdog.
type WatchdogRequest struct {
	// Frequency is the watchdog frequency in hertz.
	Frequency int `json:"frequency,omitempty"`
	// Timeout is the watchdog timeout in seconds.
	Timeout int `json:"timeout,omitempty"`
	// MagicClose enables magic-close support.
	MagicClose goubus.Bool `json:"magicclose,omitempty"`
	// Stop stops the watchdog.
	Stop goubus.Bool `json:"stop,omitempty"`
}

// SignalRequest contains parameters for signaling a process.
type SignalRequest struct {
	// Pid is the process identifier.
	Pid int `json:"pid"`
	// Signum is the signal number.
	Signum int `json:"signum"`
}

// ValidateFirmwareImageRequest contains firmware validation parameters.
type ValidateFirmwareImageRequest struct {
	// Path is the firmware image path.
	Path string `json:"path"`
}

// SysupgradeRequest contains system upgrade parameters.
type SysupgradeRequest struct {
	// Options contains system upgrade options.
	Options map[string]any `json:"options,omitempty"`
	// Path is the firmware image path.
	Path string `json:"path"`
	// Backup is the backup archive path.
	Backup string `json:"backup,omitempty"`
	// Prefix is the upgrade prefix.
	Prefix string `json:"prefix,omitempty"`
	// Command is the upgrade command.
	Command string `json:"command,omitempty"`
	// Force allows an otherwise unsupported upgrade.
	Force goubus.Bool `json:"force,omitempty"`
}
