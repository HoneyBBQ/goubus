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
	return goubus.Call[*Info](ctx, c.caller, "iwinfo", "info", map[string]any{"device": device})
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

	return res["results"], nil
}

// TxPowerList retrieves available transmit power settings.
func (c *Client) TxPowerList(ctx context.Context, device string) ([]any, error) {
	res, err := goubus.Call[map[string][]any](ctx, c.caller, "iwinfo", "txpowerlist", map[string]any{"device": device})
	if err != nil {
		return nil, err
	}

	return res["results"], nil
}

// CountryList retrieves available country codes.
func (c *Client) CountryList(ctx context.Context, device string) ([]any, error) {
	res, err := goubus.Call[map[string][]any](ctx, c.caller, "iwinfo", "countrylist", map[string]any{"device": device})
	if err != nil {
		return nil, err
	}

	return res["results"], nil
}

// Survey retrieves channel survey information.
func (c *Client) Survey(ctx context.Context, device string) ([]any, error) {
	res, err := goubus.Call[map[string][]any](ctx, c.caller, "iwinfo", "survey", map[string]any{"device": device})
	if err != nil {
		return nil, err
	}

	return res["results"], nil
}

// PhyName retrieves the physical name for a UCI section.
func (c *Client) PhyName(ctx context.Context, section string) (string, error) {
	res, err := goubus.Call[map[string]string](ctx, c.caller, "iwinfo", "phyname", map[string]any{"section": section})
	if err != nil {
		return "", err
	}

	return res["phyname"], nil
}

// Info contains wireless interface information.
type Info struct {
	// Encryption contains the wireless encryption details.
	Encryption *Encryption `json:"encryption"`
	// SSID is the network name.
	SSID string `json:"ssid"`
	// BSSID is the access point address.
	BSSID string `json:"bssid"`
	// Phy is the physical wireless device name.
	Phy string `json:"phy"`
	// Mode is the wireless operation mode.
	Mode string `json:"mode"`
	// Country is the regulatory country code.
	Country string `json:"country"`
	// HTModes lists supported high-throughput modes.
	HTModes []string `json:"htmodes"`
	// HWModes lists supported hardware modes.
	HWModes []string `json:"hwmodes"`
	// Channel is the wireless channel.
	Channel int `json:"channel"`
	// Frequency is the operating frequency in megahertz.
	Frequency int `json:"frequency"`
	// TxPower is the transmit power in dBm.
	TxPower int `json:"txpower"`
	// Quality is the signal quality.
	Quality int `json:"quality"`
	// QualityMax is the maximum signal quality.
	QualityMax int `json:"quality_max"`
	// Signal is the signal strength in dBm.
	Signal int `json:"signal"`
	// Noise is the noise level in dBm.
	Noise int `json:"noise"`
	// Bitrate is the data rate in kilobits per second.
	Bitrate int `json:"bitrate"`
}

// Encryption contains wireless encryption information.
type Encryption struct {
	// WPA lists the enabled WPA versions.
	WPA []int `json:"wpa"`
	// Authentication lists the configured authentication methods.
	Authentication []string `json:"authentication"`
	// Ciphers lists the configured encryption ciphers.
	Ciphers []string `json:"ciphers"`
	// Enabled indicates whether encryption is enabled.
	Enabled goubus.Bool `json:"enabled"`
}

// ScanResult contains a discovered wireless network.
type ScanResult struct {
	// Encryption contains the wireless encryption details.
	Encryption *Encryption `json:"encryption"`
	// SSID is the network name.
	SSID string `json:"ssid"`
	// BSSID is the access point address.
	BSSID string `json:"bssid"`
	// Mode is the wireless operation mode.
	Mode string `json:"mode"`
	// Channel is the wireless channel.
	Channel int `json:"channel"`
	// Band is the wireless frequency band.
	Band int `json:"band"`
	// MHz is the operating frequency in megahertz.
	MHz int `json:"mhz"`
	// Quality is the signal quality.
	Quality int `json:"quality"`
	// QualityMax is the maximum signal quality.
	QualityMax int `json:"quality_max"`
	// Signal is the signal strength in dBm.
	Signal int `json:"signal"`
}

// Assoc contains information about an associated wireless station.
type Assoc struct {
	// MAC is the station MAC address.
	MAC string `json:"mac"`
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
	// Throughput is the current throughput in kilobits per second.
	Throughput int `json:"thr"`
	// Authorized indicates whether the station is authorized.
	Authorized goubus.Bool `json:"authorized"`
	// Authenticated indicates whether the station is authenticated.
	Authenticated goubus.Bool `json:"authenticated"`
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
	// Packets is the number of packets transferred.
	Packets int `json:"packets"`
	// Bytes is the number of bytes transferred.
	Bytes int64 `json:"bytes"`
	// DropMisc is the number of miscellaneous dropped packets.
	DropMisc int `json:"drop_misc"`
	// Failed is the number of failed transmission attempts.
	Failed int `json:"failed"`
	// Retries is the number of transmission retries.
	Retries int `json:"retries"`
	// IsHt indicates support for 802.11n high throughput.
	IsHt goubus.Bool `json:"ht"`
	// IsVht indicates support for 802.11ac very high throughput.
	IsVht goubus.Bool `json:"vht"`
	// IsHe indicates support for 802.11ax high efficiency.
	IsHe goubus.Bool `json:"he"`
	// IsEht indicates support for 802.11be extremely high throughput.
	IsEht goubus.Bool `json:"eht"`
}
