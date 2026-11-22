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
	// Inactive contains inactive address and route data.
	Inactive Inactive `json:"inactive"`
	// Data contains protocol-specific interface data.
	Data Data `json:"data"`
	// L3Device is the layer 3 device backing the interface.
	L3Device string `json:"l3_device"`
	// Zone is the firewall zone assigned to the interface.
	Zone string `json:"zone"`
	// Device is the network device backing the interface.
	Device string `json:"device"`
	// Proto is the interface protocol.
	Proto string `json:"proto"`
	// DNSServer lists the interface's DNS servers.
	DNSServer []string `json:"dns-server"`
	// DNSSearch lists the interface's DNS search domains.
	DNSSearch []string `json:"dns-search"`
	// IPv4Address lists the interface's IPv4 addresses.
	IPv4Address []Address `json:"ipv4-address"`
	// Updated lists the interface properties updated by the last operation.
	Updated []string `json:"updated"`
	// Delegates lists prefixes delegated to the interface.
	Delegates []string `json:"delegates"`
	// IPv6AddressGenerated lists generated IPv6 addresses.
	IPv6AddressGenerated []Address `json:"ipv6-address-generated"`
	// Route lists routes associated with the interface.
	Route []Route `json:"route"`
	// IPv6PrefixAssignment lists IPv6 prefix assignments.
	IPv6PrefixAssignment []IPv6Prefix `json:"ipv6-prefix-assignment"`
	// IPv6Prefix lists IPv6 prefixes associated with the interface.
	IPv6Prefix []IPv6Prefix `json:"ipv6-prefix"`
	// Neighbors lists IPv6 neighbor entries.
	Neighbors []Neighbor `json:"neighbors"`
	// IPv6Address lists the interface's IPv6 addresses.
	IPv6Address []Address `json:"ipv6-address"`
	// Errors contains interface status errors.
	Errors []InterfaceError `json:"errors"`
	// Metric is the interface route metric.
	Metric int `json:"metric"`
	// DNSMetric is the interface DNS route metric.
	DNSMetric int `json:"dns_metric"`
	// Uptime is the interface uptime in seconds.
	Uptime int `json:"uptime"`
	// Up reports whether the interface is up.
	Up goubus.Bool `json:"up"`
	// Pending reports whether an interface operation is pending.
	Pending goubus.Bool `json:"pending"`
	// Available reports whether the interface is available.
	Available goubus.Bool `json:"available"`
	// Autostart reports whether the interface starts automatically.
	Autostart goubus.Bool `json:"autostart"`
	// Dynamic reports whether the interface was created dynamically.
	Dynamic goubus.Bool `json:"dynamic"`
	// Delegation reports whether IPv6 prefix delegation is enabled.
	Delegation goubus.Bool `json:"delegation"`
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
type Data map[string]any

// InterfaceError contains an interface status error.
type InterfaceError struct {
	// Subsystem identifies the subsystem reporting the error.
	Subsystem string `json:"subsystem"`
	// Code is the error code.
	Code string `json:"code"`
}

// Device represents a network device status.
type Device struct {
	// FlowControl contains Ethernet flow-control settings.
	FlowControl *DeviceFlowControl `json:"flow-control,omitempty"`
	// Speed is the link speed.
	Speed string `json:"speed"`
	// Type is the device type.
	Type string `json:"type"`
	// MacAddr is the device hardware address.
	MacAddr string `json:"macaddr"`
	// Devtype is the device class.
	Devtype string `json:"devtype"`
	// Conduit is the network device carrying this device's traffic.
	Conduit string `json:"conduit"`
	// BridgeMembers lists devices attached to the bridge.
	BridgeMembers []string `json:"bridge-members,omitempty"`
	// LinkAdvertising lists advertised link modes.
	LinkAdvertising []string `json:"link-advertising"`
	// LinkPartnerAdvertising lists the link partner's advertised modes.
	LinkPartnerAdvertising []string `json:"link-partner-advertising"`
	// LinkSupported lists supported link modes.
	LinkSupported []string `json:"link-supported"`
	// Statistics contains device traffic counters.
	Statistics DeviceStatistic `json:"statistics"`
	// MTU is the maximum transmission unit.
	MTU int `json:"mtu"`
	// Up reports whether the device is up.
	Up goubus.Bool `json:"up"`
	// Carrier reports whether the device has a carrier signal.
	Carrier goubus.Bool `json:"carrier"`
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
	// RxDropped is the received packet drop count.
	RxDropped int64 `json:"rx_dropped"`
	// TxDropped is the transmitted packet drop count.
	TxDropped int64 `json:"tx_dropped"`
	// Multicast is the received multicast packet count.
	Multicast int64 `json:"multicast"`
	// Collisions is the transmit collision count.
	Collisions int64 `json:"collisions"`
}

// DeviceFlowControl contains Ethernet flow-control settings.
type DeviceFlowControl struct {
	// Supported lists locally supported flow-control modes.
	Supported []string `json:"supported"`
	// LinkAdvertising lists locally advertised flow-control modes.
	LinkAdvertising []string `json:"link-advertising"`
	// LinkPartnerAdvertising lists peer-advertised flow-control modes.
	LinkPartnerAdvertising []string `json:"link-partner-advertising"`
	// Negotiated lists negotiated flow-control modes.
	Negotiated []string `json:"negotiated"`
	// Autoneg reports whether flow-control autonegotiation is enabled.
	Autoneg goubus.Bool `json:"autoneg"`
}

// RadioStatus contains a wireless radio's runtime status.
type RadioStatus struct {
	// Config contains the radio configuration.
	Config map[string]any `json:"config"`
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
	// Config contains the interface configuration.
	Config map[string]any `json:"config"`
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
