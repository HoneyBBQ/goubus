// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package dhcp

import (
	"context"

	"github.com/honeybbq/goubus/v2"
)

// Dialect defines the differences in DHCP ubus calls.
type Dialect interface {
	PrepareAddLease(req AddLeaseRequest) any
}

// Manager provides an interface for interacting with the DHCP server.
type Manager struct {
	caller  goubus.Transport
	dialect Dialect
}

// New creates a new base DHCP Manager.
func New(t goubus.Transport, d Dialect) *Manager {
	return &Manager{caller: t, dialect: d}
}

// AddLease creates a new static DHCP lease.
func (m *Manager) AddLease(ctx context.Context, req AddLeaseRequest) error {
	params := m.dialect.PrepareAddLease(req)
	_, err := m.caller.Call(ctx, "dhcp", "add_lease", params)

	return err
}

// IPv4Leases retrieves current IPv4 DHCP leases.
func (m *Manager) IPv4Leases(ctx context.Context) ([]IPv4Lease, error) {
	res, err := goubus.Call[map[string][]IPv4Lease](ctx, m.caller, "dhcp", "ipv4leases", nil)
	if err != nil {
		return nil, err
	}

	// The response is usually a map where keys are interface names
	var allLeases []IPv4Lease
	for _, leases := range *res {
		allLeases = append(allLeases, leases...)
	}

	return allLeases, nil
}

// IPv6Leases retrieves current IPv6 DHCP leases.
func (m *Manager) IPv6Leases(ctx context.Context) ([]IPv6Lease, error) {
	res, err := goubus.Call[map[string][]IPv6Lease](ctx, m.caller, "dhcp", "ipv6leases", nil)
	if err != nil {
		return nil, err
	}

	var allLeases []IPv6Lease
	for _, leases := range *res {
		allLeases = append(allLeases, leases...)
	}

	return allLeases, nil
}

// IPv6RA retrieves current IPv6 Router Advertisement information.
func (m *Manager) IPv6RA(ctx context.Context) ([]IPv6RA, error) {
	res, err := goubus.Call[map[string][]IPv6RA](ctx, m.caller, "dhcp", "ipv6ra", nil)
	if err != nil {
		return nil, err
	}

	var allRAs []IPv6RA
	for _, ras := range *res {
		allRAs = append(allRAs, ras...)
	}

	return allRAs, nil
}
