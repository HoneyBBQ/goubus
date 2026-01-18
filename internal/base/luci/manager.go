// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package luci

import (
	"context"
	"time"

	"github.com/honeybbq/goubus/v2"
)

// Dialect defines the differences in Luci ubus calls.
type Dialect interface {
	GetTimeMethod() string
}

// Manager provides the base implementation for LuCI operations.
type Manager struct {
	caller  goubus.Transport
	dialect Dialect
}

// New creates a new base LuCI Manager.
func New(t goubus.Transport, d Dialect) *Manager {
	return &Manager{caller: t, dialect: d}
}

// GetVersion retrieves the LuCI version information from the device.
func (m *Manager) GetVersion(ctx context.Context) (*Version, error) {
	return goubus.Call[Version](ctx, m.caller, "luci", "getVersion", nil)
}

type timeResponse struct {
	Time int64 `json:"result"`
}

// GetTime retrieves the current system time from the device using the dialect's method.
func (m *Manager) GetTime(ctx context.Context) (time.Time, error) {
	result, err := goubus.Call[timeResponse](ctx, m.caller, "luci", m.dialect.GetTimeMethod(), nil)
	if err != nil {
		return time.Time{}, err
	}

	return time.Unix(result.Time, 0), nil
}

// SetLocaltime sets the system local time on the device.
func (m *Manager) SetLocaltime(ctx context.Context, t time.Time) error {
	params := map[string]any{"localtime": t.Unix()}
	_, err := m.caller.Call(ctx, "luci", "setLocaltime", params)

	return err
}

// GetInitList retrieves the list of initialization scripts.
func (m *Manager) GetInitList(ctx context.Context, name string) (map[string]any, error) {
	params := map[string]any{}
	if name != "" {
		params["name"] = name
	}

	res, err := goubus.Call[map[string]any](ctx, m.caller, "luci", "getInitList", params)
	if err != nil {
		return nil, err
	}

	return *res, nil
}

// SetInitAction performs an action on an init script.
func (m *Manager) SetInitAction(ctx context.Context, name, action string) error {
	params := map[string]any{
		"name":   name,
		"action": action,
	}
	_, err := m.caller.Call(ctx, "luci", "setInitAction", params)

	return err
}

// GetTimezones retrieves the list of available system timezones.
func (m *Manager) GetTimezones(ctx context.Context) (map[string]any, error) {
	res, err := goubus.Call[map[string]any](ctx, m.caller, "luci", "getTimezones", nil)
	if err != nil {
		return nil, err
	}

	return *res, nil
}

// GetLEDs retrieves the list of system LEDs.
func (m *Manager) GetLEDs(ctx context.Context) (map[string]LED, error) {
	res, err := goubus.Call[map[string]LED](ctx, m.caller, "luci", "getLEDs", nil)
	if err != nil {
		return nil, err
	}

	return *res, nil
}

// GetUSBDevices retrieves the list of connected USB devices.
func (m *Manager) GetUSBDevices(ctx context.Context) ([]USBDevice, error) {
	res, err := goubus.Call[[]USBDevice](ctx, m.caller, "luci", "getUSBDevices", nil)
	if err != nil {
		return nil, err
	}

	return *res, nil
}

// GetConntrackHelpers retrieves the list of connection tracking helpers.
func (m *Manager) GetConntrackHelpers(ctx context.Context) ([]string, error) {
	res, err := goubus.Call[[]string](ctx, m.caller, "luci", "getConntrackHelpers", nil)
	if err != nil {
		return nil, err
	}

	return *res, nil
}

// GetFeatures retrieves the list of supported system features.
func (m *Manager) GetFeatures(ctx context.Context) (map[string]goubus.Bool, error) {
	res, err := goubus.Call[map[string]goubus.Bool](ctx, m.caller, "luci", "getFeatures", nil)
	if err != nil {
		return nil, err
	}

	return *res, nil
}

// GetSwconfigFeatures retrieves features for a swconfig switch.
func (m *Manager) GetSwconfigFeatures(ctx context.Context, switchName string) (map[string]any, error) {
	params := map[string]any{"switch": switchName}

	res, err := goubus.Call[map[string]any](ctx, m.caller, "luci", "getSwconfigFeatures", params)
	if err != nil {
		return nil, err
	}

	return *res, nil
}

// GetSwconfigPortState retrieves the port state for a swconfig switch.
func (m *Manager) GetSwconfigPortState(ctx context.Context, switchName string) (map[string]any, error) {
	params := map[string]any{"switch": switchName}

	res, err := goubus.Call[map[string]any](ctx, m.caller, "luci", "getSwconfigPortState", params)
	if err != nil {
		return nil, err
	}

	return *res, nil
}

// SetPassword sets the password for a system user.
func (m *Manager) SetPassword(ctx context.Context, username, password string) error {
	params := map[string]any{
		"username": username,
		"password": password,
	}
	_, err := m.caller.Call(ctx, "luci", "setPassword", params)

	return err
}

// GetBlockDevices retrieves the list of block devices.
func (m *Manager) GetBlockDevices(ctx context.Context) ([]BlockDevice, error) {
	res, err := goubus.Call[[]BlockDevice](ctx, m.caller, "luci", "getBlockDevices", nil)
	if err != nil {
		return nil, err
	}

	return *res, nil
}

// SetBlockDetect triggers block device detection.
func (m *Manager) SetBlockDetect(ctx context.Context) error {
	_, err := m.caller.Call(ctx, "luci", "setBlockDetect", nil)

	return err
}

// GetMountPoints retrieves the list of mount points.
func (m *Manager) GetMountPoints(ctx context.Context) ([]MountPoint, error) {
	res, err := goubus.Call[[]MountPoint](ctx, m.caller, "luci", "getMountPoints", nil)
	if err != nil {
		return nil, err
	}

	return *res, nil
}

// GetRealtimeStats retrieves realtime statistics.
func (m *Manager) GetRealtimeStats(ctx context.Context, mode, device string) (*RealtimeStats, error) {
	params := map[string]any{
		"mode":   mode,
		"device": device,
	}

	return goubus.Call[RealtimeStats](ctx, m.caller, "luci", "getRealtimeStats", params)
}

// GetConntrackList retrieves the list of current connections.
func (m *Manager) GetConntrackList(ctx context.Context) ([]any, error) {
	res, err := goubus.Call[[]any](ctx, m.caller, "luci", "getConntrackList", nil)
	if err != nil {
		return nil, err
	}

	return *res, nil
}

// GetProcessList retrieves the list of system processes.
func (m *Manager) GetProcessList(ctx context.Context) ([]Process, error) {
	res, err := goubus.Call[[]Process](ctx, m.caller, "luci", "getProcessList", nil)
	if err != nil {
		return nil, err
	}

	return *res, nil
}

// GetBuiltinEthernetPorts retrieves the list of builtin ethernet ports.
func (m *Manager) GetBuiltinEthernetPorts(ctx context.Context) ([]any, error) {
	res, err := goubus.Call[[]any](ctx, m.caller, "luci", "getBuiltinEthernetPorts", nil)
	if err != nil {
		return nil, err
	}

	return *res, nil
}

// GetOdhcp6cStats retrieves statistics from odhcp6c (RAX300M specific).
func (m *Manager) GetOdhcp6cStats(ctx context.Context) (map[string]any, error) {
	res, err := goubus.Call[map[string]any](ctx, m.caller, "luci", "getOdhcp6cStats", nil)
	if err != nil {
		return nil, err
	}

	return *res, nil
}

// GetNetworkDevices retrieves detailed network device information.
func (m *Manager) GetNetworkDevices(ctx context.Context) (map[string]NetworkDevice, error) {
	res, err := goubus.Call[map[string]NetworkDevice](ctx, m.caller, "luci-rpc", "getNetworkDevices", nil)
	if err != nil {
		return nil, err
	}

	return *res, nil
}

// GetWirelessDevices retrieves detailed wireless device information.
func (m *Manager) GetWirelessDevices(ctx context.Context) (map[string]WirelessDevice, error) {
	res, err := goubus.Call[map[string]WirelessDevice](ctx, m.caller, "luci-rpc", "getWirelessDevices", nil)
	if err != nil {
		return nil, err
	}

	return *res, nil
}

// GetHostHints retrieves host hint information.
func (m *Manager) GetHostHints(ctx context.Context) (map[string]HostHint, error) {
	res, err := goubus.Call[map[string]HostHint](ctx, m.caller, "luci-rpc", "getHostHints", nil)
	if err != nil {
		return nil, err
	}

	return *res, nil
}

// GetDUIDHints retrieves DUID hint information.
func (m *Manager) GetDUIDHints(ctx context.Context) (map[string]any, error) {
	res, err := goubus.Call[map[string]any](ctx, m.caller, "luci-rpc", "getDUIDHints", nil)
	if err != nil {
		return nil, err
	}

	return *res, nil
}

// GetBoardJSON retrieves board hardware information.
func (m *Manager) GetBoardJSON(ctx context.Context) (*BoardJSON, error) {
	return goubus.Call[BoardJSON](ctx, m.caller, "luci-rpc", "getBoardJSON", nil)
}

// GetDHCPLeases retrieves DHCP leases with optional family filter.
func (m *Manager) GetDHCPLeases(ctx context.Context, family int) (*DHCPLeases, error) {
	params := map[string]any{}
	if family != 0 {
		params["family"] = family
	}

	return goubus.Call[DHCPLeases](ctx, m.caller, "luci-rpc", "getDHCPLeases", params)
}
