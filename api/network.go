package api

import (
	"github.com/honeybbq/goubus/errdefs"
	"github.com/honeybbq/goubus/types"
)

// Network service constants
const (
	ServiceNetwork          = "network"
	ServiceNetworkInterface = "network.interface"
	ServiceNetworkDevice    = "network.device"
	ServiceNetworkWireless  = "network.wireless"

	NetworkMethodRestart = "restart"
	NetworkMethodReload  = "reload"

	NetworkInterfaceMethodDump         = "dump"
	NetworkInterfaceMethodStatus       = "status"
	NetworkInterfaceMethodUp           = "up"
	NetworkInterfaceMethodDown         = "down"
	NetworkInterfaceMethodRenew        = "renew"
	NetworkInterfaceMethodAddDevice    = "add_device"
	NetworkInterfaceMethodRemoveDevice = "remove_device"

	NetworkDeviceMethodStatus   = "status"
	NetworkDeviceMethodSetAlias = "set_alias"
	NetworkDeviceMethodSetState = "set_state"
	NetworkDeviceMethodStpInit  = "stp_init"

	NetworkWirelessMethodUp     = "up"
	NetworkWirelessMethodDown   = "down"
	NetworkWirelessMethodStatus = "status"
)

const (
	NetworkInterfaceParamInterface = "interface"
	NetworkInterfaceParamName      = "name"
	NetworkDeviceParamName         = "name"
	NetworkDeviceParamDevice       = "device"
	NetworkDeviceParamAlias        = "alias"
	NetworkDeviceParamDefer        = "defer"
	NetworkDeviceParamAuthStatus   = "auth_status"
	NetworkDeviceParamAuthVlans    = "auth_vlans"
)

// RestartNetwork restarts the network service.
func RestartNetwork(caller types.Transport) error {
	_, err := caller.Call(ServiceNetwork, NetworkMethodRestart, nil)
	return err
}

// ReloadNetwork reloads the network configuration.
func ReloadNetwork(caller types.Transport) error {
	_, err := caller.Call(ServiceNetwork, NetworkMethodReload, nil)
	return err
}

// =============================================================================
// NETWORK INTERFACE OPERATIONS
// =============================================================================

// NetworkInterfaceDumpResult represents the full dump of network interfaces.
type NetworkInterfaceDumpResult struct {
	Interface []types.NetworkInterfaceInfo `json:"interface"`
}

// DumpNetworkInterfaces retrieves information about all network interfaces.
func DumpNetworkInterfaces(caller types.Transport) ([]types.NetworkInterfaceInfo, error) {
	resp, err := caller.Call(ServiceNetworkInterface, NetworkInterfaceMethodDump, nil)
	if err != nil {
		return nil, err
	}
	ubusData := &NetworkInterfaceDumpResult{}
	if err := resp.Unmarshal(ubusData); err != nil {
		return nil, err
	}
	return ubusData.Interface, nil
}

// GetNetworkInterfaceStatus retrieves the status of a specific network interface.
func GetNetworkInterfaceStatus(caller types.Transport, name string) (*types.NetworkInterface, error) {
	// First try: Direct interface status query
	params := map[string]any{"interface": name}
	resp, err := caller.Call(ServiceNetworkInterface+"."+name, NetworkInterfaceMethodStatus, params)
	if err == nil {
		ubusData := &types.NetworkInterface{}
		if err := resp.Unmarshal(ubusData); err != nil {
			return nil, err
		}
		return ubusData, nil
	}

	// First attempt failed, try fallback using dump
	dump, dumpErr := DumpNetworkInterfaces(caller)
	if dumpErr != nil {
		return nil, err
	}

	// Search for the interface in the dump results
	for _, iface := range dump {
		if iface.Interface == name {
			return &iface.NetworkInterface, nil
		}
	}

	return nil, errdefs.Wrapf(errdefs.ErrNotFound, "interface '%s' not found", name)
}

// UpNetworkInterface brings the interface up.
func UpNetworkInterface(caller types.Transport, name string) error {
	_, err := caller.Call(ServiceNetworkInterface+"."+name, NetworkInterfaceMethodUp, nil)
	return err
}

// DownNetworkInterface takes the interface down.
func DownNetworkInterface(caller types.Transport, name string) error {
	_, err := caller.Call(ServiceNetworkInterface+"."+name, NetworkInterfaceMethodDown, nil)
	return err
}

// RenewNetworkInterface renews the DHCP lease for the interface.
func RenewNetworkInterface(caller types.Transport, name string) error {
	_, err := caller.Call(ServiceNetworkInterface+"."+name, NetworkInterfaceMethodRenew, nil)
	return err
}

// AddNetworkInterfaceDevice adds a device to the interface.
func AddNetworkInterfaceDevice(caller types.Transport, name, device string) error {
	params := map[string]any{NetworkInterfaceParamName: device}
	_, err := caller.Call(ServiceNetworkInterface+"."+name, NetworkInterfaceMethodAddDevice, params)
	return err
}

// RemoveNetworkInterfaceDevice removes a device from the interface.
func RemoveNetworkInterfaceDevice(caller types.Transport, name, device string) error {
	params := map[string]any{NetworkInterfaceParamName: device}
	_, err := caller.Call(ServiceNetworkInterface+"."+name, NetworkInterfaceMethodRemoveDevice, params)
	return err
}

// =============================================================================
// NETWORK DEVICE OPERATIONS
// =============================================================================

// GetNetworkDeviceStatus retrieves the status of a specific network device.
func GetNetworkDeviceStatus(caller types.Transport, name string) (map[string]types.NetworkDevice, error) {
	params := map[string]any{}
	if name != "" {
		params[NetworkDeviceParamName] = name
	}
	resp, err := caller.Call(ServiceNetworkDevice, NetworkDeviceMethodStatus, params)
	if err != nil {
		return nil, err
	}

	var ubusData map[string]types.NetworkDevice
	if err := resp.Unmarshal(&ubusData); err != nil {
		return nil, errdefs.Wrapf(err, "failed to unmarshal network device status")
	}
	return ubusData, nil

}

// SetNetworkDeviceAlias sets aliases for a network device.
func SetNetworkDeviceAlias(caller types.Transport, deviceName string, aliases []string) error {
	params := map[string]any{
		NetworkDeviceParamDevice: deviceName,
		NetworkDeviceParamAlias:  aliases,
	}
	_, err := caller.Call(ServiceNetworkDevice, NetworkDeviceMethodSetAlias, params)
	return err
}

// SetNetworkDeviceState sets the state of a network device.
func SetNetworkDeviceState(caller types.Transport, name string, _defer bool, authStatus bool, authVlans []string) error {
	params := map[string]any{
		NetworkDeviceParamName:       name,
		NetworkDeviceParamDefer:      _defer,
		NetworkDeviceParamAuthStatus: authStatus,
		NetworkDeviceParamAuthVlans:  authVlans,
	}
	_, err := caller.Call(ServiceNetworkDevice, NetworkDeviceMethodSetState, params)
	return err
}

// InitNetworkDeviceStp initializes STP on a bridge device.
func InitNetworkDeviceStp(caller types.Transport) error {
	_, err := caller.Call(ServiceNetworkDevice, NetworkDeviceMethodStpInit, nil)
	return err
}

// =============================================================================
// NETWORK WIRELESS OPERATIONS
// =============================================================================

func UpNetworkWireless(caller types.Transport) error {
	_, err := caller.Call(ServiceNetworkWireless, NetworkWirelessMethodUp, nil)
	return err
}

func DownNetworkWireless(caller types.Transport) error {
	_, err := caller.Call(ServiceNetworkWireless, NetworkWirelessMethodDown, nil)
	return err
}

// WirelessStatus represents the status of wireless interfaces.
type WirelessStatusResponse struct {
	Radio map[string]types.RadioStatus `json:"radio"`
}

// GetNetworkWirelessStatus retrieves the status of all wireless radios and their interfaces.
func GetNetworkWirelessStatus(caller types.Transport) (map[string]types.RadioStatus, error) {
	resp, err := caller.Call(ServiceNetworkWireless, NetworkWirelessMethodStatus, nil)
	if err != nil {
		return nil, err
	}
	ubusData := WirelessStatusResponse{}
	if err := resp.Unmarshal(&ubusData); err != nil {
		return nil, err
	}
	return ubusData.Radio, nil
}
