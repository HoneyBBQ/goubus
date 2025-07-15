package goubus

import (
	"encoding/json"
	"errors"
)

// NetworkInterfaceAddress represents an IP address with its prefix/mask
type NetworkInterfaceAddress struct {
	Address string `json:"address"`
	Mask    int    `json:"mask"`
}

// NetworkInterfaceRoute represents a routing table entry
type NetworkInterfaceRoute struct {
	Target  string `json:"target"`
	Mask    int    `json:"mask"`
	Nexthop string `json:"nexthop"`
	Source  string `json:"source"`
}

// NetworkInterfaceIPv6Prefix represents an IPv6 prefix assignment
type NetworkInterfaceIPv6Prefix struct {
	Address   string `json:"address"`
	Mask      int    `json:"mask"`
	Class     string `json:"class,omitempty"`
	Assigned  int    `json:"assigned,omitempty"`
	Hint      string `json:"hint,omitempty"`
	Preferred int    `json:"preferred,omitempty"`
	Valid     int    `json:"valid,omitempty"`
}

// NetworkInterfaceNeighbor represents a network neighbor entry
type NetworkInterfaceNeighbor struct {
	Address string `json:"address"`
	MAC     string `json:"mac"`
	Router  bool   `json:"router,omitempty"`
	State   int    `json:"state,omitempty"`
}

// NetworkInterfaceInactive represents inactive network configuration
type NetworkInterfaceInactive struct {
	Ipv4Address []NetworkInterfaceAddress  `json:"ipv4-address"`
	Ipv6Address []NetworkInterfaceAddress  `json:"ipv6-address"`
	Route       []NetworkInterfaceRoute    `json:"route"`
	DNSServer   []string                   `json:"dns-server"`
	DNSSearch   []string                   `json:"dns-search"`
	Neighbors   []NetworkInterfaceNeighbor `json:"neighbors"`
}

// NetworkInterfaceData represents additional interface data
type NetworkInterfaceData struct {
	Zone string `json:"zone,omitempty"`
}

// NetworkInterfaceDumpResult represents the result of network interface dump
type NetworkInterfaceDumpResult struct {
	Interface []NetworkInterfaceInfo `json:"interface"`
}

// NetworkInterfaceInfo represents individual interface information in dump result
type NetworkInterfaceInfo struct {
	Interface string `json:"interface"`
	NetworkInterface
}

type NetworkInterface struct {
	Up                   bool                         `json:"up"`
	Pending              bool                         `json:"pending"`
	Available            bool                         `json:"available"`
	Autostart            bool                         `json:"autostart"`
	Dynamic              bool                         `json:"dynamic"`
	Uptime               int                          `json:"uptime"`
	L3Device             string                       `json:"l3_device"`
	Proto                string                       `json:"proto"`
	Device               string                       `json:"device"`
	Updated              []string                     `json:"updated"`
	Metric               int                          `json:"metric"`
	DNSMetric            int                          `json:"dns_metric"`
	Delegation           bool                         `json:"delegation"`
	Ipv4Address          []NetworkInterfaceAddress    `json:"ipv4-address"`
	Ipv6Address          []NetworkInterfaceAddress    `json:"ipv6-address"`
	Ipv6Prefix           []NetworkInterfaceIPv6Prefix `json:"ipv6-prefix"`
	Ipv6PrefixAssignment []NetworkInterfaceIPv6Prefix `json:"ipv6-prefix-assignment"`
	Route                []NetworkInterfaceRoute      `json:"route"`
	DNSServer            []string                     `json:"dns-server"`
	DNSSearch            []string                     `json:"dns-search"`
	Neighbors            []NetworkInterfaceNeighbor   `json:"neighbors"`
	Inactive             NetworkInterfaceInactive     `json:"inactive"`
	Data                 NetworkInterfaceData         `json:"data"`
}

// NetworkInterfaceStatus retrieves the status of a specific network interface.
// It first tries to get the status directly from the interface, and if that fails
// (e.g., due to permission issues), it falls back to using the dump method.
func (u *Client) networkInterfaceStatus(name string) (NetworkInterface, error) {
	errLogin := u.LoginCheck()
	if errLogin != nil {
		return NetworkInterface{}, errLogin
	}

	// First try: Direct interface status query
	jsonStr := u.buildUbusCall("network.interface."+name, "status", nil)
	call, err := u.Call(jsonStr)
	if err == nil {
		// Success - parse and return the result
		ubusData := NetworkInterface{}
		ubusDataByte, err := json.Marshal(call.Result.([]interface{})[1])
		if err != nil {
			return NetworkInterface{}, errors.New("data error")
		}
		json.Unmarshal(ubusDataByte, &ubusData)
		return ubusData, nil
	}

	// First attempt failed, try fallback using dump
	// This handles cases where direct interface status queries are blocked by ACL
	dump, dumpErr := u.networkInterfaceDump()
	if dumpErr != nil {
		// Both methods failed, return the original error from the direct call
		return NetworkInterface{}, err
	}

	// Search for the interface in the dump results
	for _, iface := range dump.Interface {
		if iface.Interface == name {
			// Found the interface in dump results
			return iface.NetworkInterface, nil
		}
	}

	// Interface not found in dump, return the original error
	return NetworkInterface{}, errors.New("interface '" + name + "' not found")
}

// NetworkInterfaceDump retrieves information about all network interfaces.
func (u *Client) networkInterfaceDump() (NetworkInterfaceDumpResult, error) {
	errLogin := u.LoginCheck()
	if errLogin != nil {
		return NetworkInterfaceDumpResult{}, errLogin
	}
	jsonStr := u.buildUbusCall("network.interface", "dump", nil)
	call, err := u.Call(jsonStr)
	if err != nil {
		return NetworkInterfaceDumpResult{}, err
	}
	ubusData := NetworkInterfaceDumpResult{}

	ubusDataByte, err := json.Marshal(call.Result.([]interface{})[1])
	if err != nil {
		return NetworkInterfaceDumpResult{}, errors.New("data error")
	}
	json.Unmarshal(ubusDataByte, &ubusData)
	return ubusData, nil
}
