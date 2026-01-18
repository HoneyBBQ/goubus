// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package network

import (
	"context"

	"github.com/honeybbq/goubus/v2"
	"github.com/honeybbq/goubus/v2/internal/base/network"
)

type RAX3000MDialect struct{}

// Manager handles Network operations for CMCC RAX3000M.
type Manager struct {
	base *network.Manager
}

func New(t goubus.Transport) *Manager {
	return &Manager{
		base: network.New(t, RAX3000MDialect{}),
	}
}

func (m *Manager) Dump(ctx context.Context) ([]InterfaceInfo, error) {
	return m.base.DumpInterfaces(ctx)
}

func (m *Manager) Interface(name string) *InterfaceContext {
	return m.base.Interface(name)
}

func (m *Manager) Restart(ctx context.Context) error {
	return m.base.Restart(ctx)
}

func (m *Manager) Reload(ctx context.Context) error {
	return m.base.Reload(ctx)
}

func (m *Manager) AddHostRoute(ctx context.Context, req HostRouteRequest) error {
	return m.base.AddHostRoute(ctx, req)
}

func (m *Manager) GetProtoHandlers(ctx context.Context) (map[string]any, error) {
	return m.base.GetProtoHandlers(ctx)
}

func (m *Manager) AddDynamic(ctx context.Context, name string) error {
	return m.base.AddDynamic(ctx, name)
}

func (m *Manager) NetnsUpDown(ctx context.Context, req NetnsUpDownRequest) error {
	return m.base.NetnsUpDown(ctx, req)
}

func (m *Manager) Devices() *DeviceContext {
	return m.base.Devices()
}

func (m *Manager) Wireless() *WirelessContext {
	return m.base.Wireless()
}

// Type aliases for public use.
type (
	InterfaceInfo          = network.InterfaceInfo
	InterfaceDetails       = network.InterfaceDetails
	RadioStatus            = network.RadioStatus
	InterfaceContext       = network.InterfaceContext
	DeviceContext          = network.DeviceContext
	WirelessContext        = network.WirelessContext
	HostRouteRequest       = network.HostRouteRequest
	NetnsUpDownRequest     = network.NetnsUpDownRequest
	DeviceSetAliasRequest  = network.DeviceSetAliasRequest
	DeviceSetStateRequest  = network.DeviceSetStateRequest
	InterfaceDeviceRequest = network.InterfaceDeviceRequest
	WirelessNotifyRequest  = network.WirelessNotifyRequest
)
