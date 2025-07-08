package goubus

import (
	"fmt"
	"strings"
)

type UbusNetwork struct {
	NetworkInterface NetworkInterface  `json:"network"`
	NetworkDevice    UbusNetworkDevice `json:"network.device"`
}

// NetworkInterfaceConfig represents the configuration parameters for a network interface
type NetworkInterfaceConfig struct {
	Proto    string   `json:"proto,omitempty"`    // Protocol: "static", "dhcp", "pppoe", etc.
	IPAddr   []string `json:"ipaddr,omitempty"`   // IP addresses for static configuration (can be CIDR)
	Gateway  string   `json:"gateway,omitempty"`  // Default gateway
	DNS      []string `json:"dns,omitempty"`      // DNS servers
	Device   string   `json:"device,omitempty"`   // Physical device name
	Type     string   `json:"type,omitempty"`     // Interface type: "bridge", etc.
	IfName   []string `json:"ifname,omitempty"`   // Interface name list
	Disabled string   `json:"disabled,omitempty"` // "0" or "1"
	Auto     string   `json:"auto,omitempty"`     // "0" or "1" - auto start on boot
	Metric   string   `json:"metric,omitempty"`   // Interface metric
	MTU      string   `json:"mtu,omitempty"`      // Maximum transmission unit
	Username string   `json:"username,omitempty"` // For PPPoE
	Password string   `json:"password,omitempty"` // For PPPoE
	Service  string   `json:"service,omitempty"`  // For PPPoE
}

// NetworkInterfaceCreateRequest represents the parameters for creating a new interface
type NetworkInterfaceCreateRequest struct {
	Type   string                 `json:"type"`   // Usually "interface"
	Config NetworkInterfaceConfig `json:"config"` // Initial configuration
}

// Network returns a manager for the 'network' UCI configuration.
func (c *Client) Network() *NetworkManager {
	return &NetworkManager{
		client: c,
	}
}

// NetworkManager provides methods to interact with the network configuration.
type NetworkManager struct {
	client *Client
}

// Interface selects a specific interface section (e.g., 'lan', 'wan') for configuration.
func (nm *NetworkManager) Interface(sectionName string) *InterfaceManager {
	return &InterfaceManager{
		client:  nm.client,
		section: sectionName,
	}
}

// Commit saves all staged network configuration changes.
func (nm *NetworkManager) Commit() error {
	req := UbusUciRequestGeneric{
		Config: "network",
	}
	return nm.client.uciCommit(nm.client.id, req)
}

// Dump retrieves information about all network interfaces.
func (nm *NetworkManager) Dump() (NetworkInterfaceDumpResult, error) {
	return nm.client.networkInterfaceDump()
}

// InterfaceManager provides methods to configure a specific network interface.
type InterfaceManager struct {
	client  *Client
	section string
}

// GetConfig retrieves the static configuration for the interface from UCI.
func (im *InterfaceManager) GetConfig() (*NetworkInterfaceConfig, error) {
	req := UbusUciGetRequest{
		UbusUciRequestGeneric: UbusUciRequestGeneric{
			Config:  "network",
			Section: im.section,
		},
	}
	resp, err := im.client.uciGet(im.client.id, req)
	if err != nil {
		return nil, err
	}

	config := &NetworkInterfaceConfig{}
	// Populate config struct from the map
	// This is simplified mapping, a real implementation might use reflection
	if val, ok := resp.Values["proto"]; ok {
		config.Proto = val
	}
	if val, ok := resp.Values["ipaddr"]; ok {
		config.IPAddr = strings.Split(val, " ")
	}
	if val, ok := resp.Values["gateway"]; ok {
		config.Gateway = val
	}
	if val, ok := resp.Values["dns"]; ok {
		config.DNS = strings.Split(val, " ")
	}
	if val, ok := resp.Values["device"]; ok {
		config.Device = val
	}
	if val, ok := resp.Values["type"]; ok {
		config.Type = val
	}
	if val, ok := resp.Values["ifname"]; ok {
		config.IfName = strings.Split(val, " ")
	}
	if val, ok := resp.Values["disabled"]; ok {
		config.Disabled = val
	}
	if val, ok := resp.Values["auto"]; ok {
		config.Auto = val
	}
	if val, ok := resp.Values["metric"]; ok {
		config.Metric = val
	}
	if val, ok := resp.Values["mtu"]; ok {
		config.MTU = val
	}
	if val, ok := resp.Values["username"]; ok {
		config.Username = val
	}
	if val, ok := resp.Values["password"]; ok {
		config.Password = val
	}
	if val, ok := resp.Values["service"]; ok {
		config.Service = val
	}

	return config, nil
}

// Set applies configuration parameters to the interface section.
func (im *InterfaceManager) Set(config NetworkInterfaceConfig) error {
	values := structToMap(config)
	req := UbusUciRequest{
		UbusUciRequestGeneric: UbusUciRequestGeneric{
			Config:  "network",
			Section: im.section,
		},
		Values: values,
	}
	return im.client.uciSet(im.client.id, req)
}

// SetProto is a helper method to set the protocol for the interface.
func (im *InterfaceManager) SetProto(proto string) error {
	return im.Set(NetworkInterfaceConfig{Proto: proto})
}

// SetIPAddr is a helper method to set the IP address(es) for the interface.
func (im *InterfaceManager) SetIPAddr(ipaddrs []string) error {
	return im.Set(NetworkInterfaceConfig{IPAddr: ipaddrs})
}

// SetGateway is a helper method to set the gateway for the interface.
func (im *InterfaceManager) SetGateway(gateway string) error {
	return im.Set(NetworkInterfaceConfig{Gateway: gateway})
}

// SetStatic is a helper method to configure a static IP interface.
func (im *InterfaceManager) SetStatic(ipaddrs []string, gateway string) error {
	return im.Set(NetworkInterfaceConfig{
		Proto:   "static",
		IPAddr:  ipaddrs,
		Gateway: gateway,
	})
}

// SetDHCP is a helper method to configure the interface for DHCP.
func (im *InterfaceManager) SetDHCP() error {
	return im.Set(NetworkInterfaceConfig{Proto: "dhcp"})
}

// Add creates a new network interface section with the specified configuration.
func (im *InterfaceManager) Add(request NetworkInterfaceCreateRequest) error {
	values := structToMap(request.Config)
	req := UbusUciRequest{
		UbusUciRequestGeneric: UbusUciRequestGeneric{
			Config:  "network",
			Section: im.section,
			Type:    request.Type,
		},
		Values: values,
	}
	return im.client.uciAdd(im.client.id, req)
}

// Delete removes the network interface section from the configuration.
func (im *InterfaceManager) Delete() error {
	req := UbusUciRequestGeneric{
		Config:  "network",
		Section: im.section,
	}
	return im.client.uciDelete(im.client.id, req)
}

// Status retrieves the live status information for the specific network interface.
func (im *InterfaceManager) Status() (*UbusNetwork, error) {
	// The original NetworkStatus function in network.go has been deleted,
	// so we will reconstruct the logic here. It involves getting interface
	// status and then device status.
	ifaceStatus, err := im.client.networkInterfaceStatus(im.section)
	if err != nil {
		return nil, err
	}

	if ifaceStatus.L3Device == "" {
		// For interfaces like 'lo', there might not be a layer 3 device.
		// We can return a partially populated struct.
		return &UbusNetwork{NetworkInterface: ifaceStatus}, nil
	}

	// Try to get device status, but don't fail the entire query if device status fails
	devStatus, err := im.client.networkDeviceStatus(ifaceStatus.L3Device)
	if err != nil {
		// Log error but continue returning interface info, graceful degradation
		return &UbusNetwork{NetworkInterface: ifaceStatus}, nil
	}

	return &UbusNetwork{
		NetworkInterface: ifaceStatus,
		NetworkDevice:    devStatus,
	}, nil
}

// AddDNS adds a new DNS server to the interface's DNS list.
func (im *InterfaceManager) AddDNS(dnsServer string) error {
	// Get current configuration
	currentConfig, err := im.GetConfig()
	if err != nil {
		return fmt.Errorf("failed to get current config: %w", err)
	}

	// Check if DNS server already exists
	for _, dns := range currentConfig.DNS {
		if dns == dnsServer {
			return nil // DNS server already exists, no need to add
		}
	}

	// Add new DNS server
	currentConfig.DNS = append(currentConfig.DNS, dnsServer)

	// Update configuration using Set method
	return im.Set(*currentConfig)
}

// DeleteDNS removes a specific DNS server from the interface's DNS list.
func (im *InterfaceManager) DeleteDNS(dnsServer string) error {
	// Get current configuration
	currentConfig, err := im.GetConfig()
	if err != nil {
		return fmt.Errorf("failed to get current config: %w", err)
	}

	// Remove specified server from DNS list
	newDNSList := make([]string, 0, len(currentConfig.DNS))
	found := false
	for _, dns := range currentConfig.DNS {
		if dns != dnsServer {
			newDNSList = append(newDNSList, dns)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("DNS server '%s' not found in the current DNS list", dnsServer)
	}

	// Update DNS list
	currentConfig.DNS = newDNSList

	// Update configuration using Set method
	return im.Set(*currentConfig)
}

// AddIfName adds a new interface name to the interface's ifname list.
func (im *InterfaceManager) AddIfName(ifname string) error {
	// Get current configuration
	currentConfig, err := im.GetConfig()
	if err != nil {
		return fmt.Errorf("failed to get current config: %w", err)
	}

	// Check if interface name already exists
	for _, iface := range currentConfig.IfName {
		if iface == ifname {
			return nil // Interface name already exists, no need to add
		}
	}

	// Add new interface name
	currentConfig.IfName = append(currentConfig.IfName, ifname)

	// Update configuration using Set method
	return im.Set(*currentConfig)
}

// DeleteIfName removes a specific interface name from the interface's ifname list.
func (im *InterfaceManager) DeleteIfName(ifname string) error {
	// Get current configuration
	currentConfig, err := im.GetConfig()
	if err != nil {
		return fmt.Errorf("failed to get current config: %w", err)
	}

	// Remove specified interface name from interface name list
	newIfNameList := make([]string, 0, len(currentConfig.IfName))
	found := false
	for _, iface := range currentConfig.IfName {
		if iface != ifname {
			newIfNameList = append(newIfNameList, iface)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("interface name '%s' not found in the current ifname list", ifname)
	}

	// Update interface name list
	currentConfig.IfName = newIfNameList

	// Update configuration using Set method
	return im.Set(*currentConfig)
}

// structToMap converts a struct to map[string]string for UCI operations
// This is a helper function to convert our typed structs to the format expected by UCI
func structToMap(v interface{}) map[string]string {
	// This is a simplified implementation
	// In a real implementation, you'd use reflection to convert struct fields to map
	// For now, we'll implement specific converters for each type
	result := make(map[string]string)

	switch config := v.(type) {
	case NetworkInterfaceConfig:
		if config.Proto != "" {
			result["proto"] = config.Proto
		}
		if len(config.IPAddr) > 0 {
			result["ipaddr"] = strings.Join(config.IPAddr, " ")
		}
		if config.Gateway != "" {
			result["gateway"] = config.Gateway
		}
		if len(config.DNS) > 0 {
			result["dns"] = strings.Join(config.DNS, " ")
		}
		if config.Device != "" {
			result["device"] = config.Device
		}
		if config.Type != "" {
			result["type"] = config.Type
		}
		if len(config.IfName) > 0 {
			result["ifname"] = strings.Join(config.IfName, " ")
		}
		if config.Disabled != "" {
			result["disabled"] = config.Disabled
		}
		if config.Auto != "" {
			result["auto"] = config.Auto
		}
		if config.Metric != "" {
			result["metric"] = config.Metric
		}
		if config.MTU != "" {
			result["mtu"] = config.MTU
		}
		if config.Username != "" {
			result["username"] = config.Username
		}
		if config.Password != "" {
			result["password"] = config.Password
		}
		if config.Service != "" {
			result["service"] = config.Service
		}
	}

	return result
}
