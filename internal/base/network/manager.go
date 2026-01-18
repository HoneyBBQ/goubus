// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package network

import (
	"context"

	"github.com/honeybbq/goubus/v2"
	"github.com/honeybbq/goubus/v2/errdefs"
)

// Dialect defines the differences in Network ubus calls.
type Dialect any

// Manager provides methods to interact with network configuration and status.
type Manager struct {
	caller  goubus.Transport
	dialect Dialect
}

// New creates a new base network Manager.
func New(t goubus.Transport, d Dialect) *Manager {
	return &Manager{caller: t, dialect: d}
}

// Restart restarts the network service.
func (m *Manager) Restart(ctx context.Context) error {
	_, err := m.caller.Call(ctx, "network", "restart", nil)

	return err
}

// Reload reloads the network configuration.
func (m *Manager) Reload(ctx context.Context) error {
	_, err := m.caller.Call(ctx, "network", "reload", nil)

	return err
}

// AddHostRoute adds a host route.
func (m *Manager) AddHostRoute(ctx context.Context, req HostRouteRequest) error {
	_, err := m.caller.Call(ctx, "network", "add_host_route", req)

	return err
}

// GetProtoHandlers retrieves available protocol handlers.
func (m *Manager) GetProtoHandlers(ctx context.Context) (map[string]any, error) {
	res, err := goubus.Call[map[string]any](ctx, m.caller, "network", "get_proto_handlers", nil)
	if err != nil {
		return nil, err
	}

	return *res, nil
}

// AddDynamic adds a dynamic interface.
func (m *Manager) AddDynamic(ctx context.Context, name string) error {
	params := map[string]any{"name": name}
	_, err := m.caller.Call(ctx, "network", "add_dynamic", params)

	return err
}

// NetnsUpDown manages network namespace up/down.
func (m *Manager) NetnsUpDown(ctx context.Context, req NetnsUpDownRequest) error {
	_, err := m.caller.Call(ctx, "network", "netns_updown", req)

	return err
}

// Interface selects a specific interface for operations.
func (m *Manager) Interface(name string) *InterfaceContext {
	return &InterfaceContext{
		manager: m,
		name:    name,
	}
}

// Devices returns a context for device operations.
func (m *Manager) Devices() *DeviceContext {
	return &DeviceContext{manager: m}
}

// Wireless returns a context for wireless status operations.
func (m *Manager) Wireless() *WirelessContext {
	return &WirelessContext{manager: m}
}

// InterfaceContext provides methods to manage a network interface.
type InterfaceContext struct {
	manager *Manager
	name    string
}

type interfaceDumpResult struct {
	Interface []InterfaceInfo `json:"interface"`
}

// DumpInterfaces retrieves detailed information about all network interfaces.
func (m *Manager) DumpInterfaces(ctx context.Context) ([]InterfaceInfo, error) {
	ubusData, err := goubus.Call[interfaceDumpResult](ctx, m.caller, "network.interface", "dump", nil)
	if err != nil {
		return nil, err
	}

	return ubusData.Interface, nil
}

// Up brings the network interface up.
func (ic *InterfaceContext) Up(ctx context.Context) error {
	_, err := ic.manager.caller.Call(ctx, "network.interface."+ic.name, "up", nil)

	return err
}

// Down takes the network interface down.
func (ic *InterfaceContext) Down(ctx context.Context) error {
	_, err := ic.manager.caller.Call(ctx, "network.interface."+ic.name, "down", nil)

	return err
}

// Renew renews the network interface (e.g., DHCP lease).
func (ic *InterfaceContext) Renew(ctx context.Context) error {
	_, err := ic.manager.caller.Call(ctx, "network.interface."+ic.name, "renew", nil)

	return err
}

// Prepare prepares the network interface.
func (ic *InterfaceContext) Prepare(ctx context.Context) error {
	_, err := ic.manager.caller.Call(ctx, "network.interface."+ic.name, "prepare", nil)

	return err
}

// AddDevice adds a device to the interface.
func (ic *InterfaceContext) AddDevice(ctx context.Context, req InterfaceDeviceRequest) error {
	_, err := ic.manager.caller.Call(ctx, "network.interface."+ic.name, "add_device", req)

	return err
}

// RemoveDevice removes a device from the interface.
func (ic *InterfaceContext) RemoveDevice(ctx context.Context, req InterfaceDeviceRequest) error {
	_, err := ic.manager.caller.Call(ctx, "network.interface."+ic.name, "remove_device", req)

	return err
}

// NotifyProto sends a notification to the interface protocol handler.
func (ic *InterfaceContext) NotifyProto(ctx context.Context) error {
	_, err := ic.manager.caller.Call(ctx, "network.interface."+ic.name, "notify_proto", nil)

	return err
}

// Remove removes the network interface.
func (ic *InterfaceContext) Remove(ctx context.Context) error {
	_, err := ic.manager.caller.Call(ctx, "network.interface."+ic.name, "remove", nil)

	return err
}

// SetData sets data on the network interface.
func (ic *InterfaceContext) SetData(ctx context.Context, data map[string]any) error {
	_, err := ic.manager.caller.Call(ctx, "network.interface."+ic.name, "set_data", data)

	return err
}

// Status retrieves live status information for the interface.
func (ic *InterfaceContext) Status(ctx context.Context) (*InterfaceDetails, error) {
	params := map[string]any{"interface": ic.name}

	ubusData, err := goubus.Call[InterfaceDetails](ctx, ic.manager.caller, "network.interface."+ic.name, "status", params)
	if err == nil {
		return ubusData, nil
	}

	dump, dumpErr := ic.manager.DumpInterfaces(ctx)
	if dumpErr != nil {
		return nil, err
	}

	for _, iface := range dump {
		if iface.Interface == ic.name {
			return &iface.InterfaceDetails, nil
		}
	}

	return nil, errdefs.Wrapf(errdefs.ErrNotFound, "interface '%s' not found", ic.name)
}

// DeviceContext provides methods to configure network devices.
type DeviceContext struct {
	manager *Manager
}

// Status retrieves the live status information for network devices.
func (dc *DeviceContext) Status(ctx context.Context, name string) (map[string]Device, error) {
	params := map[string]any{}
	if name != "" {
		params["name"] = name

		device, err := goubus.Call[Device](ctx, dc.manager.caller, "network.device", "status", params)
		if err != nil {
			return nil, err
		}

		return map[string]Device{name: *device}, nil
	}

	ubusData, err := goubus.Call[map[string]Device](ctx, dc.manager.caller, "network.device", "status", params)
	if err != nil {
		return nil, err
	}

	return *ubusData, nil
}

// SetAlias sets an alias for a network device.
func (dc *DeviceContext) SetAlias(ctx context.Context, req DeviceSetAliasRequest) error {
	_, err := dc.manager.caller.Call(ctx, "network.device", "set_alias", req)

	return err
}

// SetState sets the state of a network device.
func (dc *DeviceContext) SetState(ctx context.Context, req DeviceSetStateRequest) error {
	_, err := dc.manager.caller.Call(ctx, "network.device", "set_state", req)

	return err
}

// STPInit initializes STP on network devices.
func (dc *DeviceContext) STPInit(ctx context.Context) error {
	_, err := dc.manager.caller.Call(ctx, "network.device", "stp_init", nil)

	return err
}

// WirelessContext provides methods for wireless status operations.
type WirelessContext struct {
	manager *Manager
}

type wirelessStatusResponse struct {
	Radio map[string]RadioStatus `json:"radio"`
}

// Status retrieves the live status of wireless radios.
func (wc *WirelessContext) Status(ctx context.Context, device string) (map[string]RadioStatus, error) {
	params := map[string]any{}
	if device != "" {
		params["device"] = device
	}

	ubusData, err := goubus.Call[wirelessStatusResponse](ctx, wc.manager.caller, "network.wireless", "status", params)
	if err != nil {
		return nil, err
	}

	return ubusData.Radio, nil
}

// Up brings up the wireless radio.
func (wc *WirelessContext) Up(ctx context.Context, device string) error {
	params := map[string]any{}
	if device != "" {
		params["device"] = device
	}

	_, err := wc.manager.caller.Call(ctx, "network.wireless", "up", params)

	return err
}

// Down takes down the wireless radio.
func (wc *WirelessContext) Down(ctx context.Context, device string) error {
	params := map[string]any{}
	if device != "" {
		params["device"] = device
	}

	_, err := wc.manager.caller.Call(ctx, "network.wireless", "down", params)

	return err
}

// Reconf reconfigures the wireless radio.
func (wc *WirelessContext) Reconf(ctx context.Context, device string) error {
	params := map[string]any{}
	if device != "" {
		params["device"] = device
	}

	_, err := wc.manager.caller.Call(ctx, "network.wireless", "reconf", params)

	return err
}

// Notify sends a notification to the wireless radio.
func (wc *WirelessContext) Notify(ctx context.Context, req WirelessNotifyRequest) error {
	_, err := wc.manager.caller.Call(ctx, "network.wireless", "notify", req)

	return err
}

// GetValidate retrieves validation information for the wireless radio.
func (wc *WirelessContext) GetValidate(ctx context.Context, device string) (map[string]any, error) {
	params := map[string]any{}
	if device != "" {
		params["device"] = device
	}

	res, err := goubus.Call[map[string]any](ctx, wc.manager.caller, "network.wireless", "get_validate", params)
	if err != nil {
		return nil, err
	}

	return *res, nil
}

// Retry retries the wireless radio configuration (RAX300M specific).
func (wc *WirelessContext) Retry(ctx context.Context, device string) error {
	params := map[string]any{}
	if device != "" {
		params["device"] = device
	}

	_, err := wc.manager.caller.Call(ctx, "network.wireless", "retry", params)

	return err
}
