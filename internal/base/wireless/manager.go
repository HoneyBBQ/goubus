// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package wireless

import (
	"context"

	"github.com/honeybbq/goubus/v2"
)

// Manager provides methods to interact with 'iwinfo'.
type Manager struct {
	caller goubus.Transport
}

// New creates a new base wireless Manager.
func New(t goubus.Transport) *Manager {
	return &Manager{caller: t}
}

type devicesResponse struct {
	Devices []string `json:"devices"`
}

// Devices returns available wireless device names.
func (m *Manager) Devices(ctx context.Context) ([]string, error) {
	res, err := goubus.Call[devicesResponse](ctx, m.caller, "iwinfo", "devices", nil)
	if err != nil {
		return nil, err
	}

	return res.Devices, nil
}

// Info retrieves runtime information for a wireless interface.
func (m *Manager) Info(ctx context.Context, device string) (*Info, error) {
	params := map[string]any{"device": device}

	return goubus.Call[Info](ctx, m.caller, "iwinfo", "info", params)
}

type scanResponse struct {
	Results []ScanResult `json:"results"`
}

// Scan scans for nearby wireless networks.
func (m *Manager) Scan(ctx context.Context, device string) ([]ScanResult, error) {
	params := map[string]any{"device": device}

	res, err := goubus.Call[scanResponse](ctx, m.caller, "iwinfo", "scan", params)
	if err != nil {
		return nil, err
	}

	return res.Results, nil
}

type assocListResponse struct {
	Results []Assoc `json:"results"`
}

// AssocList retrieves the list of stations currently associated with the interface.
func (m *Manager) AssocList(ctx context.Context, device string) ([]Assoc, error) {
	params := map[string]any{"device": device}

	res, err := goubus.Call[assocListResponse](ctx, m.caller, "iwinfo", "assoclist", params)
	if err != nil {
		return nil, err
	}

	return res.Results, nil
}

// FreqList retrieves the list of available frequencies for the interface.
func (m *Manager) FreqList(ctx context.Context, device string) ([]any, error) {
	params := map[string]any{"device": device}

	res, err := goubus.Call[map[string][]any](ctx, m.caller, "iwinfo", "freqlist", params)
	if err != nil {
		return nil, err
	}

	return (*res)["results"], nil
}

// TxPowerList retrieves the list of available transmit power settings.
func (m *Manager) TxPowerList(ctx context.Context, device string) ([]any, error) {
	params := map[string]any{"device": device}

	res, err := goubus.Call[map[string][]any](ctx, m.caller, "iwinfo", "txpowerlist", params)
	if err != nil {
		return nil, err
	}

	return (*res)["results"], nil
}

// CountryList retrieves the list of available country codes.
func (m *Manager) CountryList(ctx context.Context, device string) ([]any, error) {
	params := map[string]any{"device": device}

	res, err := goubus.Call[map[string][]any](ctx, m.caller, "iwinfo", "countrylist", params)
	if err != nil {
		return nil, err
	}

	return (*res)["results"], nil
}

// Survey retrieves channel survey information.
func (m *Manager) Survey(ctx context.Context, device string) ([]any, error) {
	params := map[string]any{"device": device}

	res, err := goubus.Call[map[string][]any](ctx, m.caller, "iwinfo", "survey", params)
	if err != nil {
		return nil, err
	}

	return (*res)["results"], nil
}

// PhyName retrieves the physical name for a given UCI section.
func (m *Manager) PhyName(ctx context.Context, section string) (string, error) {
	params := map[string]any{"section": section}

	res, err := goubus.Call[map[string]string](ctx, m.caller, "iwinfo", "phyname", params)
	if err != nil {
		return "", err
	}

	return (*res)["phyname"], nil
}
