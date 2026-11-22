package network

import "github.com/honeybbq/goubus/v2"

// InterfaceInfo combines an interface name with its status details.
type InterfaceInfo struct {
	// Interface is the interface name.
	Interface string `json:"interface"`
	InterfaceDetails
}

// InterfaceDetails contains detailed network interface status.
type InterfaceDetails struct {
	Data                 Data         `json:"data"`
	L3Device             string       `json:"l3_device"`
	Zone                 string       `json:"zone"`
	Device               string       `json:"device"`
	Proto                string       `json:"proto"`
	Inactive             Inactive     `json:"inactive"`
	DNSServer            []string     `json:"dns-server"`
	DNSSearch            []string     `json:"dns-search"`
	IPv4Address          []Address    `json:"ipv4-address"`
	Updated              []string     `json:"updated"`
	Delegates            []string     `json:"delegates"`
	IPv6AddressGenerated []Address    `json:"ipv6-address-generated"`
	Route                []Route      `json:"route"`
	IPv6PrefixAssignment []IPv6Prefix `json:"ipv6-prefix-assignment"`
	IPv6Prefix           []IPv6Prefix `json:"ipv6-prefix"`
	Neighbors            []Neighbor   `json:"neighbors"`
	IPv6Address          []Address    `json:"ipv6-address"`
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

// Address represents an IP address and its prefix mask.
type Address struct {
	// Address is the IP address.
	Address string `json:"address"`
	// Mask is the prefix length.
	Mask int `json:"mask"`
}

// Route represents a network route.
type Route struct {
	// Target is the route destination.
	Target string `json:"target"`
	// Nexthop is the next-hop address.
	Nexthop string `json:"nexthop"`
	// Source is the preferred source address.
	Source string `json:"source"`
	// Mask is the route prefix length.
	Mask int `json:"mask"`
	// Metric is the route metric.
	Metric int `json:"metric"`
	// Valid is the remaining route lifetime.
	Valid int `json:"valid"`
}

// IPv6Prefix represents an IPv6 prefix assignment.
type IPv6Prefix struct {
	// LocalAddress is the local address associated with the prefix.
	LocalAddress *Address `json:"local-address,omitempty"`
	// Address is the IPv6 prefix address.
	Address string `json:"address"`
	// Class is the prefix assignment class.
	Class string `json:"class"`
	// Mask is the prefix length.
	Mask int `json:"mask"`
	// Preferred is the preferred lifetime in seconds.
	Preferred int `json:"preferred"`
	// Valid is the valid lifetime in seconds.
	Valid int `json:"valid"`
	// AssignedLength is the delegated prefix length.
	AssignedLength int `json:"assigned-length"`
}

// Neighbor represents a neighbor cache entry.
type Neighbor struct {
	// Address is the neighbor IP address.
	Address string `json:"address"`
	// MAC is the neighbor hardware address.
	MAC string `json:"mac"`
	// Router reports whether the neighbor is a router.
	Router goubus.Bool `json:"router"`
	// State is the neighbor cache state.
	State int `json:"state"`
}

// Inactive contains inactive interface address and route data.
type Inactive struct {
	// IPv4Address lists inactive IPv4 addresses.
	IPv4Address []Address `json:"ipv4-address"`
	// IPv6Address lists inactive IPv6 addresses.
	IPv6Address []Address `json:"ipv6-address"`
	// Route lists inactive routes.
	Route []Route `json:"route"`
}

// Data contains protocol-specific network data.
type Data struct{}

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

// DeviceStatistic contains device traffic counters.
type DeviceStatistic struct {
	// RxBytes is the received byte count.
	RxBytes int64 `json:"rx_bytes"`
	// TxBytes is the transmitted byte count.
	TxBytes int64 `json:"tx_bytes"`
	// RxPackets is the received packet count.
	RxPackets int64 `json:"rx_packets"`
	// TxPackets is the transmitted packet count.
	TxPackets int64 `json:"tx_packets"`
	// RxErrors is the receive error count.
	RxErrors int64 `json:"rx_errors"`
	// TxErrors is the transmit error count.
	TxErrors int64 `json:"tx_errors"`
}

// DeviceFlowControl contains Ethernet flow-control settings.
type DeviceFlowControl struct {
	// Autoneg reports whether flow-control autonegotiation is enabled.
	Autoneg goubus.Bool `json:"autoneg"`
}

// RadioStatus contains a wireless radio's runtime status.
type RadioStatus struct {
	// Interfaces lists interfaces attached to the radio.
	Interfaces []RadioInterface `json:"interfaces"`
	// Up reports whether the radio is up.
	Up goubus.Bool `json:"up"`
	// Pending reports whether a radio operation is pending.
	Pending goubus.Bool `json:"pending"`
	// Autostart reports whether the radio starts automatically.
	Autostart goubus.Bool `json:"autostart"`
	// Disabled reports whether the radio is disabled.
	Disabled goubus.Bool `json:"disabled"`
	// RetrySetupFailed reports whether setup retry failed.
	RetrySetupFailed goubus.Bool `json:"retry_setup_failed"`
}

// RadioInterface identifies a wireless interface on a radio.
type RadioInterface struct {
	// Section is the UCI section name.
	Section string `json:"section"`
	// Ifname is the operating system interface name.
	Ifname string `json:"ifname"`
}

// HostRouteRequest contains parameters for adding a host route.
type HostRouteRequest struct {
	// Target is the route destination.
	Target string `json:"target"`
	// Interface selects the outgoing interface.
	Interface string `json:"interface,omitempty"`
	// V6 reports whether the route is IPv6.
	V6 goubus.Bool `json:"v6,omitempty"`
	// Exclude reports whether the target should be excluded.
	Exclude goubus.Bool `json:"exclude,omitempty"`
}

// NetnsUpDownRequest contains network namespace operation parameters.
type NetnsUpDownRequest struct {
	// Jail is the network namespace name.
	Jail string `json:"jail"`
	// Start reports whether the namespace should be started.
	Start goubus.Bool `json:"start"`
}

// DeviceSetAliasRequest contains parameters for setting a device alias.
type DeviceSetAliasRequest struct {
	// Device selects the device name.
	Device string `json:"device,omitempty"`
	// Alias contains alias values.
	Alias []string `json:"alias"`
}

// DeviceSetStateRequest contains parameters for changing device state.
type DeviceSetStateRequest struct {
	// Name selects the device name.
	Name string `json:"name"`
	// AuthVlans lists authorized VLAN identifiers.
	AuthVlans []int `json:"auth_vlans,omitempty"`
	// Defer reports whether the operation should be deferred.
	Defer goubus.Bool `json:"defer,omitempty"`
	// AuthStatus reports the authorization state.
	AuthStatus goubus.Bool `json:"auth_status,omitempty"`
}

// InterfaceDeviceRequest contains parameters for changing interface devices.
type InterfaceDeviceRequest struct {
	// Name selects the device name.
	Name string `json:"name"`
	// Vlan lists VLAN identifiers.
	Vlan []int `json:"vlan,omitempty"`
	// LinkExt reports whether link extension is enabled.
	LinkExt goubus.Bool `json:"link-ext,omitempty"`
}

// WirelessNotifyRequest contains parameters for wireless notifications.
type WirelessNotifyRequest struct {
	// Data contains notification data.
	Data map[string]any `json:"data,omitempty"`
	// Device selects the radio device.
	Device string `json:"device,omitempty"`
	// Interface selects the wireless interface.
	Interface string `json:"interface,omitempty"`
	// Vlan selects the wireless VLAN.
	Vlan string `json:"vlan,omitempty"`
	// Command is the notification command.
	Command int `json:"command"`
}
