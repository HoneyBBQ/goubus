// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package luci

import (
	"github.com/honeybbq/goubus/v2"
	"github.com/honeybbq/goubus/v2/internal/base/dhcp"
	"github.com/honeybbq/goubus/v2/internal/base/network"
	"github.com/honeybbq/goubus/v2/internal/base/wireless"
)

// Version represents the version information of LuCI.
type Version struct {
	Revision string `json:"revision"`
	Branch   string `json:"branch"`
}

// WirelessDevice represents wireless device information with enhanced details.
type WirelessDevice struct {
	IWInfo wireless.Info `json:"iwinfo"`
	network.RadioStatus
}

// NetworkDevice represents detailed network device information from luci-rpc.
type NetworkDevice struct {
	MAC      string             `json:"mac,omitempty"`
	DevType  string             `json:"devtype"`
	Name     string             `json:"name"`
	ID       string             `json:"id,omitempty"`
	Master   string             `json:"master,omitempty"`
	IPAddrs  []IPAddress        `json:"ipaddrs"`
	Ports    []string           `json:"ports,omitempty"`
	IP6Addrs []IPAddress        `json:"ip6addrs"`
	Link     NetworkDeviceLink  `json:"link"`
	Stats    NetworkDeviceStats `json:"stats"`
	Type     int                `json:"type,omitempty"`
	IfIndex  int                `json:"ifindex,omitempty"`
	QLen     int                `json:"qlen"`
	MTU      int                `json:"mtu"`
	Flags    NetworkDeviceFlags `json:"flags"`
	Bridge   goubus.Bool        `json:"bridge,omitempty"`
	STP      goubus.Bool        `json:"stp,omitempty"`
	Up       goubus.Bool        `json:"up"`
	Wireless goubus.Bool        `json:"wireless"`
}

// IPAddress represents an IP address with netmask and optional broadcast.
type IPAddress struct {
	Address   string `json:"address"`
	Netmask   string `json:"netmask"`
	Broadcast string `json:"broadcast,omitempty"`
}

// NetworkDeviceStats represents network device statistics.
type NetworkDeviceStats struct {
	RxBytes    int `json:"rx_bytes"`
	TxBytes    int `json:"tx_bytes"`
	TxErrors   int `json:"tx_errors"`
	RxErrors   int `json:"rx_errors"`
	TxPackets  int `json:"tx_packets"`
	RxPackets  int `json:"rx_packets"`
	Multicast  int `json:"multicast"`
	Collisions int `json:"collisions"`
	RxDropped  int `json:"rx_dropped"`
	TxDropped  int `json:"tx_dropped"`
}

// NetworkDeviceFlags represents network device flags.
type NetworkDeviceFlags struct {
	Up           goubus.Bool `json:"up"`
	Broadcast    goubus.Bool `json:"broadcast"`
	Promisc      goubus.Bool `json:"promisc"`
	Loopback     goubus.Bool `json:"loopback"`
	NoARP        goubus.Bool `json:"noarp"`
	Multicast    goubus.Bool `json:"multicast"`
	PointToPoint goubus.Bool `json:"pointtopoint"`
}

// NetworkDeviceLink represents network device link information.
type NetworkDeviceLink struct {
	Duplex    string      `json:"duplex,omitempty"`
	Speed     int         `json:"speed,omitempty"`
	Changes   int         `json:"changes"`
	UpCount   int         `json:"up_count"`
	DownCount int         `json:"down_count"`
	Carrier   goubus.Bool `json:"carrier"`
}

// HostHint represents host hint information.
type HostHint struct {
	Name     string   `json:"name,omitempty"`
	IPAddrs  []string `json:"ipaddrs"`
	IP6Addrs []string `json:"ip6addrs"`
}

// BoardJSON represents board hardware information.
type BoardJSON struct {
	WLAN    map[string]BoardWLAN `json:"wlan"`
	Network BoardNetwork         `json:"network"`
	Model   BoardModel           `json:"model"`
}

// BoardModel represents board model information.
type BoardModel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// BoardNetwork represents board network configuration.
type BoardNetwork struct {
	WAN BoardInterface `json:"wan"`
	LAN BoardInterface `json:"lan"`
}

// BoardInterface represents a board network interface.
type BoardInterface struct {
	Device   string `json:"device"`
	Protocol string `json:"protocol"`
	IPAddr   string `json:"ipaddr,omitempty"`
	MacAddr  string `json:"macaddr"`
}

// BoardWLAN represents board wireless information.
type BoardWLAN struct {
	Path string        `json:"path"`
	Info BoardWLANInfo `json:"info"`
}

// BoardWLANInfo represents detailed wireless capability information.
type BoardWLANInfo struct {
	Bands     map[string]BoardWLANBand `json:"bands"`
	Radios    []any                    `json:"radios"`
	AntennaRx int                      `json:"antenna_rx"`
	AntennaTx int                      `json:"antenna_tx"`
}

// BoardWLANBand represents wireless band capabilities.
type BoardWLANBand struct {
	Modes          []string    `json:"modes"`
	MaxWidth       int         `json:"max_width"`
	DefaultChannel int         `json:"default_channel,omitempty"`
	HT             goubus.Bool `json:"ht,omitempty"`
	VHT            goubus.Bool `json:"vht,omitempty"`
	HE             goubus.Bool `json:"he,omitempty"`
}

// DHCPLeases is a re-export or alias for dhcp.Leases.
type DHCPLeases = dhcp.Leases

// LED represents LED information.
type LED struct {
	Name          string `json:"name"`
	Trigger       string `json:"trigger"`
	Brightness    int    `json:"brightness"`
	MaxBrightness int    `json:"max_brightness"`
}

// USBDevice represents a USB device.
type USBDevice struct {
	Bus     string `json:"bus"`
	Device  string `json:"device"`
	Vendor  string `json:"vendor"`
	Product string `json:"product"`
	Class   string `json:"class"`
}

// BlockDevice represents a block device.
type BlockDevice struct {
	Name   string `json:"name"`
	Dev    string `json:"dev"`
	Model  string `json:"model,omitempty"`
	Vendor string `json:"vendor,omitempty"`
	Size   int64  `json:"size"`
	Major  int    `json:"major"`
	Minor  int    `json:"minor"`
}

// MountPoint represents a mount point.
type MountPoint struct {
	Device string `json:"device"`
	Mount  string `json:"mount"`
	Type   string `json:"type"`
	Size   int64  `json:"size"`
	Used   int64  `json:"used"`
	Free   int64  `json:"free"`
}

// RealtimeStats represents realtime statistics.
type RealtimeStats struct {
	RxBytes   int64 `json:"rx_bytes"`
	TxBytes   int64 `json:"tx_bytes"`
	RxPackets int64 `json:"rx_packets"`
	TxPackets int64 `json:"tx_packets"`
}

// Process represents a system process.
type Process struct {
	User    string  `json:"user"`
	Stat    string  `json:"stat"`
	Command string  `json:"command"`
	CPU     float64 `json:"cpu"`
	PID     int     `json:"pid"`
	PPID    int     `json:"ppid"`
	VSZ     int     `json:"vsz"`
}
