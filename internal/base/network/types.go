// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package network

import "github.com/honeybbq/goubus/v2"

// InterfaceInfo wraps network interface information with its name.
type InterfaceInfo struct {
	Interface        string `json:"interface"`
	InterfaceDetails `json:",inline"`
}

// InterfaceDetails represents detailed network interface information.
type InterfaceDetails struct {
	Data                 Data         `json:"data"`
	Proto                string       `json:"proto"`
	Device               string       `json:"device"`
	Zone                 string       `json:"zone"`
	L3Device             string       `json:"l3_device"`
	Inactive             Inactive     `json:"inactive"`
	IPv6Address          []Address    `json:"ipv6-address"`
	DNSSearch            []string     `json:"dns-search"`
	IPv6Prefix           []IPv6Prefix `json:"ipv6-prefix"`
	IPv6PrefixAssignment []IPv6Prefix `json:"ipv6-prefix-assignment"`
	Route                []Route      `json:"route"`
	Neighbors            []Neighbor   `json:"neighbors"`
	IPv4Address          []Address    `json:"ipv4-address"`
	Updated              []string     `json:"updated"`
	Delegates            []string     `json:"delegates"`
	IPv6AddressGenerated []Address    `json:"ipv6-address-generated"`
	DNSServer            []string     `json:"dns-server"`
	Metric               int          `json:"metric"`
	DNSMetric            int          `json:"dns_metric"`
	Uptime               int          `json:"uptime"`
	Up                   goubus.Bool  `json:"up"`
	Pending              goubus.Bool  `json:"pending"`
	Available            goubus.Bool  `json:"available"`
	Autostart            goubus.Bool  `json:"autostart"`
	Dynamic              goubus.Bool  `json:"dynamic"`
	Delegation           goubus.Bool  `json:"delegation"`
}

// Address represents an IP address assignment.
type Address struct {
	Address string `json:"address"`
	Mask    int    `json:"mask"`
}

// Route represents a routing table entry.
type Route struct {
	Target  string `json:"target"`
	Nexthop string `json:"nexthop"`
	Source  string `json:"source"`
	Mask    int    `json:"mask"`
	Metric  int    `json:"metric"`
	Valid   int    `json:"valid"`
}

// IPv6Prefix represents an IPv6 prefix assignment.
type IPv6Prefix struct {
	LocalAddress   *Address `json:"local-address,omitempty"`
	Address        string   `json:"address"`
	Class          string   `json:"class"`
	Mask           int      `json:"mask"`
	Preferred      int      `json:"preferred"`
	Valid          int      `json:"valid"`
	AssignedLength int      `json:"assigned-length"`
}

// Neighbor represents a neighbor cache entry.
type Neighbor struct {
	Address string      `json:"address"`
	MAC     string      `json:"mac"`
	Router  goubus.Bool `json:"router"`
	State   int         `json:"state"`
}

// Inactive represents inactive interface configuration.
type Inactive struct {
	IPv4Address []Address `json:"ipv4-address"`
	IPv6Address []Address `json:"ipv6-address"`
	Route       []Route   `json:"route"`
}

// Data represents additional interface data.
type Data struct {
	// Protocol-specific data
}

// Device represents a network device status.
type Device struct {
	FlowControl            *DeviceFlowControl `json:"flow-control,omitempty"`
	Speed                  string             `json:"speed"`
	Type                   string             `json:"type"`
	MacAddr                string             `json:"macaddr"`
	Devtype                string             `json:"devtype"`
	LinkAdvertising        []string           `json:"link-advertising"`
	LinkPartnerAdvertising []string           `json:"link-partner-advertising"`
	LinkSupported          []string           `json:"link-supported"`
	Statistics             DeviceStatistic    `json:"statistics"`
	MTU                    int                `json:"mtu"`
	Up                     goubus.Bool        `json:"up"`
	Carrier                goubus.Bool        `json:"carrier"`
}

// DeviceStatistic represents network device statistics.
type DeviceStatistic struct {
	RxBytes   int64 `json:"rx_bytes"`
	TxBytes   int64 `json:"tx_bytes"`
	RxPackets int64 `json:"rx_packets"`
	TxPackets int64 `json:"tx_packets"`
	RxErrors  int64 `json:"rx_errors"`
	TxErrors  int64 `json:"tx_errors"`
}

// DeviceFlowControl represents flow control configuration.
type DeviceFlowControl struct {
	Autoneg goubus.Bool `json:"autoneg"`
}

// RadioStatus represents the status of a wireless radio.
type RadioStatus struct {
	Interfaces []RadioInterface `json:"interfaces"`
	Up         goubus.Bool      `json:"up"`
	Pending    goubus.Bool      `json:"pending"`
	Disabled   goubus.Bool      `json:"disabled"`
}

// RadioInterface represents a wireless interface attached to a radio.
type RadioInterface struct {
	Section string `json:"section"`
	Ifname  string `json:"ifname"`
}

// HostRouteRequest represents parameters for adding a host route.
type HostRouteRequest struct {
	Target    string      `json:"target"`
	Interface string      `json:"interface,omitempty"`
	V6        goubus.Bool `json:"v6,omitempty"`
	Exclude   goubus.Bool `json:"exclude,omitempty"`
}

// NetnsUpDownRequest represents parameters for network namespace up/down.
type NetnsUpDownRequest struct {
	Jail  string      `json:"jail"`
	Start goubus.Bool `json:"start"`
}

// DeviceSetAliasRequest represents parameters for setting a device alias.
type DeviceSetAliasRequest struct {
	Device string   `json:"device,omitempty"`
	Alias  []string `json:"alias"`
}

// DeviceSetStateRequest represents parameters for setting a device state.
type DeviceSetStateRequest struct {
	Name       string      `json:"name"`
	AuthVlans  []int       `json:"auth_vlans,omitempty"`
	Defer      goubus.Bool `json:"defer,omitempty"`
	AuthStatus goubus.Bool `json:"auth_status,omitempty"`
}

// InterfaceDeviceRequest represents parameters for adding/removing a device from an interface.
type InterfaceDeviceRequest struct {
	Name    string      `json:"name"`
	Vlan    []int       `json:"vlan,omitempty"`
	LinkExt goubus.Bool `json:"link-ext,omitempty"`
}

// WirelessNotifyRequest represents parameters for wireless notification.
type WirelessNotifyRequest struct {
	Data      map[string]any `json:"data,omitempty"`
	Device    string         `json:"device,omitempty"`
	Interface string         `json:"interface,omitempty"`
	Vlan      string         `json:"vlan,omitempty"`
	Command   int            `json:"command"`
}
