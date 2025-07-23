package goubus

import (
	"github.com/honeybbq/goubus/api"
	"github.com/honeybbq/goubus/types"
)

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

// Interfaces retrieves information about all network interfaces.
// This is equivalent to 'ubus call network.interface dump'.
func (nm *NetworkManager) Interfaces() (*types.NetworkInterfaceDumpResult, error) {
	return api.DumpNetworkInterfaces(nm.client.caller)
}

// Restart restarts the network service.
func (nm *NetworkManager) Restart() error {
	return api.RestartNetwork(nm.client.caller)
}

// Reload reloads the network configuration.
func (nm *NetworkManager) Reload() error {
	return api.ReloadNetwork(nm.client.caller)
}

// Interface selects a specific interface section (e.g., 'lan', 'wan') for configuration.
func (nm *NetworkManager) Interface(sectionName string) *InterfaceManager {
	return &InterfaceManager{
		client:  nm.client,
		section: sectionName,
	}
}

// Device selects a specific device section for configuration.
func (nm *NetworkManager) Device(sectionName string) *DeviceManager {
	return &DeviceManager{
		client:  nm.client,
		section: sectionName,
	}
}

// Wireless returns a manager for 'network.wireless' status operations.
func (nm *NetworkManager) Wireless() *NetworkWirelessManager {
	return &NetworkWirelessManager{
		client: nm.client,
	}
}

// InterfaceManager provides methods to configure a specific network interface.
type InterfaceManager struct {
	client  *Client
	section string
}

// DeviceManager provides methods to configure a specific network device section.
type DeviceManager struct {
	client  *Client
	section string
}

// Up brings the interface up.
func (im *InterfaceManager) Up() error {
	return api.UpNetworkInterface(im.client.caller, im.section)
}

// Down takes the interface down.
func (im *InterfaceManager) Down() error {
	return api.DownNetworkInterface(im.client.caller, im.section)
}

// Renew renews the DHCP lease for the interface.
func (im *InterfaceManager) Renew() error {
	return api.RenewNetworkInterface(im.client.caller, im.section)
}

// AddDevice adds a device to the interface.
func (im *InterfaceManager) AddDevice(device string) error {
	return api.AddNetworkInterfaceDevice(im.client.caller, im.section, device)
}

// RemoveDevice removes a device from the interface.
func (im *InterfaceManager) RemoveDevice(device string) error {
	return api.RemoveNetworkInterfaceDevice(im.client.caller, im.section, device)
}

// Status retrieves the live status information for the specific network interface.
func (im *InterfaceManager) Status() (*types.NetworkInterface, error) {
	return api.GetNetworkInterfaceStatus(im.client.caller, im.section)
}

// Status retrieves the live status information for the specific network device.
func (dm *DeviceManager) Status() (*types.NetworkDevice, error) {
	return api.GetNetworkDeviceStatus(dm.client.caller, dm.section)
}

// SetAlias sets aliases for the network device.
func (dm *DeviceManager) SetAlias(aliases []string) error {
	return api.SetNetworkDeviceAlias(dm.client.caller, dm.section, aliases)
}

// SetState brings the network device up or down.
func (dm *DeviceManager) SetState(up bool) error {
	return api.SetNetworkDeviceState(dm.client.caller, dm.section, up)
}

// StpInit initializes STP on the bridge device.
func (dm *DeviceManager) StpInit() error {
	return api.InitNetworkDeviceStp(dm.client.caller, dm.section)
}

// NetworkWirelessManager provides methods for 'network.wireless' operations.
type NetworkWirelessManager struct {
	client *Client
}

// Status retrieves the live status of all wireless radios and their interfaces.
// Corresponds to `ubus call network.wireless status`.
func (nwm *NetworkWirelessManager) Status() (map[string]types.RadioStatus, error) {
	return api.GetNetworkWirelessStatus(nwm.client.caller)
}
