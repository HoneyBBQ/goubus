// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package wireless

import "github.com/honeybbq/goubus/v2"

// Info represents wireless interface information.
type Info struct {
	SSID    string `json:"ssid"`
	BSSID   string `json:"bssid"`
	Channel int    `json:"channel"`
	Signal  int    `json:"signal"`
}

// Encryption represents encryption info.
type Encryption struct {
	Enabled goubus.Bool `json:"enabled"`
}

// ScanResult represents a scan result.
type ScanResult struct {
	SSID    string `json:"ssid"`
	BSSID   string `json:"bssid"`
	Channel int    `json:"channel"`
	Signal  int    `json:"signal"`
}

// Assoc represents an associated wireless station.
type Assoc struct {
	Mac           string    `json:"mac"`
	Signal        int       `json:"signal"`
	SignalAvg     int       `json:"signal_avg"`
	Noise         int       `json:"noise"`
	Inactive      int       `json:"inactive"`
	ConnectedTime int       `json:"connected_time"`
	Authorized    bool      `json:"authorized"`
	Authenticated bool      `json:"authenticated"`
	Rx            AssocRate `json:"rx"`
	Tx            AssocRate `json:"tx"`
}

// AssocRate represents wireless association rate information.
type AssocRate struct {
	Rate  int         `json:"rate"`
	Mcs   int         `json:"mcs"`
	Nss   int         `json:"nss"`
	IsHt  goubus.Bool `json:"ht"`
	IsVht goubus.Bool `json:"vht"`
	IsHe  goubus.Bool `json:"he"`
	IsEht goubus.Bool `json:"eht"`
	Mhz   int         `json:"mhz"`
}
