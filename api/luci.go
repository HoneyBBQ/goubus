package api

import (
	"time"

	"github.com/honeybbq/goubus/types"
)

// Luci service and method constants
const (
	ServiceLuci             = "luci"
	LuciMethodGetVersion    = "getVersion"
	LuciMethodGetLocaltime  = "getLocaltime"
	LuciMethodSetLocaltime  = "setLocaltime"
	LuciMethodGetInitList   = "getInitList"
	LuciMethodSetInitAction = "setInitAction"
	LuciMethodGetTimezones  = "getTimezones"

	ServiceLuciRPC = "luci-rpc"
)

// Luci-RPC method constants (ServiceLuciRPC defined in dhcp.go)
const (
	LuciRPCMethodGetNetworkDevices  = "getNetworkDevices"
	LuciRPCMethodGetWirelessDevices = "getWirelessDevices"
	LuciRPCMethodGetHostHints       = "getHostHints"
	LuciRPCMethodGetDUIDHints       = "getDUIDHints"
	LuciRPCMethodGetBoardJSON       = "getBoardJSON"
	// MethodGetDHCPLeases already defined in dhcp.go

	LuciRPCMethodGetDHCPLeases = "getDHCPLeases"
)

// Luci parameter constants
const (
	LuciParamLocaltime = "localtime"
	LuciParamAction    = "action"
	LuciParamName      = "name"
)

// GetLuciVersion retrieves the LuCI version information.
func GetLuciVersion(caller types.Transport) (*types.LuciVersion, error) {
	resp, err := caller.Call(ServiceLuci, LuciMethodGetVersion, nil)
	if err != nil {
		return nil, err
	}
	var ubusData types.LuciVersion
	if err := resp.Unmarshal(&ubusData); err != nil {
		return nil, err
	}
	return &ubusData, nil
}

type LuciLocaltimeResponse struct {
	Time int64 `json:"result"`
}

// GetLuciLocaltime retrieves the current local time from the device.
func GetLuciLocaltime(caller types.Transport) (time.Time, error) {
	resp, err := caller.Call(ServiceLuci, LuciMethodGetLocaltime, nil)
	if err != nil {
		return time.Time{}, err
	}
	var result LuciLocaltimeResponse
	if err := resp.Unmarshal(&result); err != nil {
		return time.Time{}, err
	}
	return time.Unix(result.Time, 0), nil
}

// SetLuciLocaltime sets the local time on the device.
func SetLuciLocaltime(caller types.Transport, t time.Time) error {
	_, err := caller.Call(ServiceLuci, LuciMethodSetLocaltime, map[string]any{
		LuciParamLocaltime: t.Unix(),
	})
	return err
}

// GetLuciInitList retrieves the list of init scripts.
func GetLuciInitList(caller types.Transport, name string) (map[string]any, error) {
	params := map[string]any{}
	if name != "" {
		params[LuciParamName] = name
	}
	resp, err := caller.Call(ServiceLuci, LuciMethodGetInitList, params)
	if err != nil {
		return nil, err
	}
	var ubusData map[string]any
	if err := resp.Unmarshal(&ubusData); err != nil {
		return nil, err
	}
	return ubusData, nil
}

// SetLuciInitAction performs an action on an init script.
func SetLuciInitAction(caller types.Transport, name, action string) error {
	params := map[string]any{
		LuciParamName:   name,
		LuciParamAction: action,
	}
	_, err := caller.Call(ServiceLuci, LuciMethodSetInitAction, params)
	return err
}

// GetLuciTimezones retrieves the list of available timezones.
func GetLuciTimezones(caller types.Transport) (map[string]any, error) {
	resp, err := caller.Call(ServiceLuci, LuciMethodGetTimezones, nil)
	if err != nil {
		return nil, err
	}
	var ubusData map[string]any
	if err := resp.Unmarshal(&ubusData); err != nil {
		return nil, err
	}
	return ubusData, nil
}

// =============================================================================
// LUCI-RPC FUNCTIONS
// =============================================================================

// GetLuciRPCNetworkDevices retrieves detailed network device information.
func GetLuciRPCNetworkDevices(caller types.Transport) (map[string]types.LuciNetworkDevice, error) {
	resp, err := caller.Call(ServiceLuciRPC, LuciRPCMethodGetNetworkDevices, nil)
	if err != nil {
		return nil, err
	}
	var ubusData map[string]types.LuciNetworkDevice
	if err := resp.Unmarshal(&ubusData); err != nil {
		return nil, err
	}
	return ubusData, nil
}

// GetLuciRPCWirelessDevices retrieves detailed wireless device information.
func GetLuciRPCWirelessDevices(caller types.Transport) (map[string]types.LuciWirelessDevice, error) {
	resp, err := caller.Call(ServiceLuciRPC, LuciRPCMethodGetWirelessDevices, nil)
	if err != nil {
		return nil, err
	}
	var ubusData map[string]types.LuciWirelessDevice
	if err := resp.Unmarshal(&ubusData); err != nil {
		return nil, err
	}
	return ubusData, nil
}

// GetLuciRPCHostHints retrieves host hint information.
func GetLuciRPCHostHints(caller types.Transport) (map[string]types.LuciHostHint, error) {
	resp, err := caller.Call(ServiceLuciRPC, LuciRPCMethodGetHostHints, nil)
	if err != nil {
		return nil, err
	}
	var ubusData map[string]types.LuciHostHint
	if err := resp.Unmarshal(&ubusData); err != nil {
		return nil, err
	}
	return ubusData, nil
}

// GetLuciRPCDUIDHints retrieves DUID hint information.
func GetLuciRPCDUIDHints(caller types.Transport) (map[string]any, error) {
	resp, err := caller.Call(ServiceLuciRPC, LuciRPCMethodGetDUIDHints, nil)
	if err != nil {
		return nil, err
	}
	var ubusData map[string]any
	if err := resp.Unmarshal(&ubusData); err != nil {
		return nil, err
	}
	return ubusData, nil
}

// GetLuciRPCBoardJSON retrieves board hardware information.
func GetLuciRPCBoardJSON(caller types.Transport) (*types.LuciBoardJSON, error) {
	resp, err := caller.Call(ServiceLuciRPC, LuciRPCMethodGetBoardJSON, nil)
	if err != nil {
		return nil, err
	}
	var ubusData types.LuciBoardJSON
	if err := resp.Unmarshal(&ubusData); err != nil {
		return nil, err
	}
	return &ubusData, nil
}

// GetDHCPLeases retrieves both IPv4 and IPv6 DHCP leases using luci-rpc.
// This is the recommended way as 'ubus call dhcp ipv4leases' often returns empty data on many devices.
func GetLuciRPCDHCPLeases(caller types.Transport) (*types.DHCPLeases, error) {
	resp, err := caller.Call(ServiceLuciRPC, LuciRPCMethodGetDHCPLeases, nil)
	if err != nil {
		return nil, err
	}
	var dhcpLeases types.DHCPLeases
	if err := resp.Unmarshal(&dhcpLeases); err != nil {
		return nil, err
	}
	return &dhcpLeases, nil
}
