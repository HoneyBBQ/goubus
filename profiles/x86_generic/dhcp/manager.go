// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package dhcp

import (
	"context"

	"github.com/honeybbq/goubus/v2"
	"github.com/honeybbq/goubus/v2/internal/base/dhcp"
)

// StandardDialect implements the standard DHCP ubus behavior.
type StandardDialect struct{}

func (d StandardDialect) PrepareAddLease(req dhcp.AddLeaseRequest) any {
	return map[string]any{
		"ip":        req.IP,
		"mac":       []string{req.Mac}, // Standard expect Array
		"duid":      req.DUID,          // Standard expect string
		"hostid":    req.HostID,
		"leasetime": req.LeaseTime,
		"name":      req.Name,
	}
}

// Manager handles DHCP operations for standard x86/generic OpenWrt.
type Manager struct {
	base *dhcp.Manager
}

// New creates a new DHCP Manager for generic.
func New(t goubus.Transport) *Manager {
	return &Manager{
		base: dhcp.New(t, StandardDialect{}),
	}
}

func (m *Manager) AddLease(ctx context.Context, req AddLeaseRequest) error {
	return m.base.AddLease(ctx, req)
}

func (m *Manager) IPv4Leases(ctx context.Context) ([]IPv4Lease, error) {
	return m.base.IPv4Leases(ctx)
}

func (m *Manager) IPv6Leases(ctx context.Context) ([]IPv6Lease, error) {
	return m.base.IPv6Leases(ctx)
}

func (m *Manager) IPv6RA(ctx context.Context) ([]IPv6RA, error) {
	return m.base.IPv6RA(ctx)
}

// Type aliases for public use.
type (
	IPv4Lease       = dhcp.IPv4Lease
	IPv6Lease       = dhcp.IPv6Lease
	IPv6RA          = dhcp.IPv6RA
	AddLeaseRequest = dhcp.AddLeaseRequest
)
