// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package luci

import (
	"context"
	"time"

	"github.com/honeybbq/goubus/v2"
	"github.com/honeybbq/goubus/v2/internal/base/luci"
)

// StandardDialect implements the standard LuCI ubus behavior.
type StandardDialect struct{}

func (d StandardDialect) GetTimeMethod() string { return "getLocaltime" }

// Manager handles LuCI operations for standard x86/generic OpenWrt.
type Manager struct {
	base *luci.Manager
}

// New creates a new Luci Manager for generic.
func New(t goubus.Transport) *Manager {
	return &Manager{
		base: luci.New(t, StandardDialect{}),
	}
}

func (m *Manager) GetVersion(ctx context.Context) (*Version, error) {
	return m.base.GetVersion(ctx)
}

func (m *Manager) GetLocaltime(ctx context.Context) (time.Time, error) {
	return m.base.GetTime(ctx)
}

func (m *Manager) GetDHCPLeases(ctx context.Context, family int) (*DHCPLeases, error) {
	return m.base.GetDHCPLeases(ctx, family)
}

func (m *Manager) GetInitList(ctx context.Context, name string) (map[string]any, error) {
	return m.base.GetInitList(ctx, name)
}

func (m *Manager) SetInitAction(ctx context.Context, name, action string) error {
	return m.base.SetInitAction(ctx, name, action)
}

func (m *Manager) SetLocaltime(ctx context.Context, t time.Time) error {
	return m.base.SetLocaltime(ctx, t)
}

func (m *Manager) GetTimezones(ctx context.Context) (map[string]any, error) {
	return m.base.GetTimezones(ctx)
}

func (m *Manager) GetLEDs(ctx context.Context) (map[string]LED, error) {
	return m.base.GetLEDs(ctx)
}

func (m *Manager) GetUSBDevices(ctx context.Context) ([]USBDevice, error) {
	return m.base.GetUSBDevices(ctx)
}

func (m *Manager) GetConntrackHelpers(ctx context.Context) ([]string, error) {
	return m.base.GetConntrackHelpers(ctx)
}

func (m *Manager) GetFeatures(ctx context.Context) (map[string]goubus.Bool, error) {
	return m.base.GetFeatures(ctx)
}

func (m *Manager) GetSwconfigFeatures(ctx context.Context, switchName string) (map[string]any, error) {
	return m.base.GetSwconfigFeatures(ctx, switchName)
}

func (m *Manager) GetSwconfigPortState(ctx context.Context, switchName string) (map[string]any, error) {
	return m.base.GetSwconfigPortState(ctx, switchName)
}

func (m *Manager) SetPassword(ctx context.Context, username, password string) error {
	return m.base.SetPassword(ctx, username, password)
}

func (m *Manager) GetBlockDevices(ctx context.Context) ([]BlockDevice, error) {
	return m.base.GetBlockDevices(ctx)
}

func (m *Manager) SetBlockDetect(ctx context.Context) error {
	return m.base.SetBlockDetect(ctx)
}

func (m *Manager) GetMountPoints(ctx context.Context) ([]MountPoint, error) {
	return m.base.GetMountPoints(ctx)
}

func (m *Manager) GetRealtimeStats(ctx context.Context, mode, device string) (*RealtimeStats, error) {
	return m.base.GetRealtimeStats(ctx, mode, device)
}

func (m *Manager) GetConntrackList(ctx context.Context) ([]any, error) {
	return m.base.GetConntrackList(ctx)
}

func (m *Manager) GetProcessList(ctx context.Context) ([]Process, error) {
	return m.base.GetProcessList(ctx)
}

func (m *Manager) GetBuiltinEthernetPorts(ctx context.Context) ([]any, error) {
	return m.base.GetBuiltinEthernetPorts(ctx)
}

func (m *Manager) GetNetworkDevices(ctx context.Context) (map[string]NetworkDevice, error) {
	return m.base.GetNetworkDevices(ctx)
}

func (m *Manager) GetWirelessDevices(ctx context.Context) (map[string]WirelessDevice, error) {
	return m.base.GetWirelessDevices(ctx)
}

func (m *Manager) GetHostHints(ctx context.Context) (map[string]HostHint, error) {
	return m.base.GetHostHints(ctx)
}

func (m *Manager) GetDUIDHints(ctx context.Context) (map[string]any, error) {
	return m.base.GetDUIDHints(ctx)
}

func (m *Manager) GetBoardJSON(ctx context.Context) (*BoardJSON, error) {
	return m.base.GetBoardJSON(ctx)
}

// Type aliases for public use.
type (
	Version        = luci.Version
	DHCPLeases     = luci.DHCPLeases
	LED            = luci.LED
	USBDevice      = luci.USBDevice
	BlockDevice    = luci.BlockDevice
	MountPoint     = luci.MountPoint
	RealtimeStats  = luci.RealtimeStats
	Process        = luci.Process
	NetworkDevice  = luci.NetworkDevice
	WirelessDevice = luci.WirelessDevice
	HostHint       = luci.HostHint
	BoardJSON      = luci.BoardJSON
)
