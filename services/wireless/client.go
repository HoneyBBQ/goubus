// Package wireless provides a client for the ubus iwinfo service.
package wireless

import (
	"context"

	"github.com/honeybbq/goubus/v2"
)

// Client provides access to the ubus iwinfo service.
type Client struct {
	caller goubus.Caller
}

// New creates a wireless client for caller.
func New(caller goubus.Caller) *Client {
	if caller == nil {
		panic("goubus/wireless: nil caller")
	}

	return &Client{caller: caller}
}

type devicesResponse struct {
	Devices []string `json:"devices"`
}

// Devices returns available wireless device names.
func (c *Client) Devices(ctx context.Context) ([]string, error) {
	res, err := goubus.Call[devicesResponse](ctx, c.caller, "iwinfo", "devices", nil)
	if err != nil {
		return nil, err
	}

	return res.Devices, nil
}

// Info retrieves runtime information for a wireless interface.
func (c *Client) Info(ctx context.Context, device string) (*Info, error) {
	return goubus.Call[Info](ctx, c.caller, "iwinfo", "info", map[string]any{"device": device})
}

type scanResponse struct {
	Results []ScanResult `json:"results"`
}

// Scan scans for nearby wireless networks.
func (c *Client) Scan(ctx context.Context, device string) ([]ScanResult, error) {
	res, err := goubus.Call[scanResponse](ctx, c.caller, "iwinfo", "scan", map[string]any{"device": device})
	if err != nil {
		return nil, err
	}

	return res.Results, nil
}

type assocListResponse struct {
	Results []Assoc `json:"results"`
}

// AssocList retrieves stations associated with the interface.
func (c *Client) AssocList(ctx context.Context, device string) ([]Assoc, error) {
	res, err := goubus.Call[assocListResponse](ctx, c.caller, "iwinfo", "assoclist", map[string]any{"device": device})
	if err != nil {
		return nil, err
	}

	return res.Results, nil
}

// FreqList retrieves available frequencies for the interface.
func (c *Client) FreqList(ctx context.Context, device string) ([]any, error) {
	res, err := goubus.Call[map[string][]any](ctx, c.caller, "iwinfo", "freqlist", map[string]any{"device": device})
	if err != nil {
		return nil, err
	}

	return (*res)["results"], nil
}

// TxPowerList retrieves available transmit power settings.
func (c *Client) TxPowerList(ctx context.Context, device string) ([]any, error) {
	res, err := goubus.Call[map[string][]any](ctx, c.caller, "iwinfo", "txpowerlist", map[string]any{"device": device})
	if err != nil {
		return nil, err
	}

	return (*res)["results"], nil
}

// CountryList retrieves available country codes.
func (c *Client) CountryList(ctx context.Context, device string) ([]any, error) {
	res, err := goubus.Call[map[string][]any](ctx, c.caller, "iwinfo", "countrylist", map[string]any{"device": device})
	if err != nil {
		return nil, err
	}

	return (*res)["results"], nil
}

// Survey retrieves channel survey information.
func (c *Client) Survey(ctx context.Context, device string) ([]any, error) {
	res, err := goubus.Call[map[string][]any](ctx, c.caller, "iwinfo", "survey", map[string]any{"device": device})
	if err != nil {
		return nil, err
	}

	return (*res)["results"], nil
}

// PhyName retrieves the physical name for a UCI section.
func (c *Client) PhyName(ctx context.Context, section string) (string, error) {
	res, err := goubus.Call[map[string]string](ctx, c.caller, "iwinfo", "phyname", map[string]any{"section": section})
	if err != nil {
		return "", err
	}

	return (*res)["phyname"], nil
}

// Info contains wireless interface information.
type Info struct {
	// SSID is the network name.
	SSID string `json:"ssid"`
	// BSSID is the access point address.
	BSSID string `json:"bssid"`
	// Channel is the wireless channel.
	Channel int `json:"channel"`
	// Signal is the signal strength in dBm.
	Signal int `json:"signal"`
}

// Encryption contains wireless encryption information.
type Encryption struct {
	// Enabled indicates whether encryption is enabled.
	Enabled goubus.Bool `json:"enabled"`
}

// ScanResult contains a discovered wireless network.
type ScanResult struct {
	// SSID is the network name.
	SSID string `json:"ssid"`
	// BSSID is the access point address.
	BSSID string `json:"bssid"`
	// Channel is the wireless channel.
	Channel int `json:"channel"`
	// Signal is the signal strength in dBm.
	Signal int `json:"signal"`
}

// Assoc contains information about an associated wireless station.
type Assoc struct {
	// Mac is the station MAC address.
	Mac string `json:"mac"`
	// Rx contains receive rate information.
	Rx AssocRate `json:"rx"`
	// Tx contains transmit rate information.
	Tx AssocRate `json:"tx"`
	// Signal is the signal strength in dBm.
	Signal int `json:"signal"`
	// SignalAvg is the average signal strength in dBm.
	SignalAvg int `json:"signal_avg"`
	// Noise is the noise level in dBm.
	Noise int `json:"noise"`
	// Inactive is the time since the station was active in milliseconds.
	Inactive int `json:"inactive"`
	// ConnectedTime is the station connection duration in seconds.
	ConnectedTime int `json:"connected_time"`
	// Authorized indicates whether the station is authorized.
	Authorized bool `json:"authorized"`
	// Authenticated indicates whether the station is authenticated.
	Authenticated bool `json:"authenticated"`
}

// AssocRate contains wireless association rate information.
type AssocRate struct {
	// Rate is the data rate in kilobits per second.
	Rate int `json:"rate"`
	// Mcs is the modulation and coding scheme index.
	Mcs int `json:"mcs"`
	// Nss is the number of spatial streams.
	Nss int `json:"nss"`
	// Mhz is the channel width in megahertz.
	Mhz int `json:"mhz"`
	// IsHt indicates support for 802.11n high throughput.
	IsHt goubus.Bool `json:"ht"`
	// IsVht indicates support for 802.11ac very high throughput.
	IsVht goubus.Bool `json:"vht"`
	// IsHe indicates support for 802.11ax high efficiency.
	IsHe goubus.Bool `json:"he"`
	// IsEht indicates support for 802.11be extremely high throughput.
	IsEht goubus.Bool `json:"eht"`
}
