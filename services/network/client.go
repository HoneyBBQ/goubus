// Package network provides OpenWrt network ubus operations.
package network

import (
	"context"
	"fmt"

	"github.com/honeybbq/goubus/v2"
)

// Client provides network configuration and status operations.
type Client struct {
	caller goubus.Caller
}

// New creates a network client.
func New(caller goubus.Caller) *Client {
	if caller == nil {
		panic("goubus/network: nil caller")
	}

	return &Client{caller: caller}
}

// Restart restarts the network service.
func (client *Client) Restart(ctx context.Context) error {
	return client.call(ctx, "network", "restart", nil)
}

// Reload reloads the network configuration.
func (client *Client) Reload(ctx context.Context) error {
	return client.call(ctx, "network", "reload", nil)
}

// AddHostRoute adds a host route.
func (client *Client) AddHostRoute(ctx context.Context, request HostRouteRequest) error {
	return client.call(ctx, "network", "add_host_route", request)
}

// ProtoHandlers retrieves available network protocol handlers.
func (client *Client) ProtoHandlers(ctx context.Context) (map[string]any, error) {
	return goubus.Call[map[string]any](ctx, client.caller, "network", "get_proto_handlers", nil)
}

// AddDynamic adds a dynamic network interface.
func (client *Client) AddDynamic(ctx context.Context, name string) error {
	return client.call(ctx, "network", "add_dynamic", map[string]any{"name": name})
}

// NetnsUpDown starts or stops a network namespace.
func (client *Client) NetnsUpDown(ctx context.Context, request NetnsUpDownRequest) error {
	return client.call(ctx, "network", "netns_updown", request)
}

// Interface selects an interface for operations.
func (client *Client) Interface(name string) *InterfaceContext {
	return &InterfaceContext{manager: client, name: name}
}

// Devices returns a context for device operations.
func (client *Client) Devices() *DeviceContext {
	return &DeviceContext{manager: client}
}

// Wireless returns a context for wireless operations.
func (client *Client) Wireless() *WirelessContext {
	return &WirelessContext{manager: client}
}

// InterfaceContext provides methods for one network interface.
type InterfaceContext struct {
	manager *Client
	name    string
}

type interfaceDumpResult struct {
	Interface []InterfaceInfo `json:"interface"`
}

// DumpInterfaces retrieves detailed status for all network interfaces.
func (client *Client) DumpInterfaces(ctx context.Context) ([]InterfaceInfo, error) {
	result, err := goubus.Call[interfaceDumpResult](ctx, client.caller, "network.interface", "dump", nil)
	if err != nil {
		return nil, err
	}

	return result.Interface, nil
}

// Up brings the selected interface up.
func (ic *InterfaceContext) Up(ctx context.Context) error {
	return ic.manager.call(ctx, ic.service(), "up", nil)
}

// Down brings the selected interface down.
func (ic *InterfaceContext) Down(ctx context.Context) error {
	return ic.manager.call(ctx, ic.service(), "down", nil)
}

// Renew renews the selected interface lease.
func (ic *InterfaceContext) Renew(ctx context.Context) error {
	return ic.manager.call(ctx, ic.service(), "renew", nil)
}

// Prepare prepares the selected interface.
func (ic *InterfaceContext) Prepare(ctx context.Context) error {
	return ic.manager.call(ctx, ic.service(), "prepare", nil)
}

// AddDevice adds a device to the selected interface.
func (ic *InterfaceContext) AddDevice(ctx context.Context, request InterfaceDeviceRequest) error {
	return ic.manager.call(ctx, ic.service(), "add_device", request)
}

// RemoveDevice removes a device from the selected interface.
func (ic *InterfaceContext) RemoveDevice(ctx context.Context, request InterfaceDeviceRequest) error {
	return ic.manager.call(ctx, ic.service(), "remove_device", request)
}

// NotifyProto notifies the protocol handler for the selected interface.
func (ic *InterfaceContext) NotifyProto(ctx context.Context) error {
	return ic.manager.call(ctx, ic.service(), "notify_proto", nil)
}

// Remove removes the selected interface.
func (ic *InterfaceContext) Remove(ctx context.Context) error {
	return ic.manager.call(ctx, ic.service(), "remove", nil)
}

// SetData sets protocol data on the selected interface.
func (ic *InterfaceContext) SetData(ctx context.Context, data map[string]any) error {
	return ic.manager.call(ctx, ic.service(), "set_data", data)
}

// Status retrieves live status for the selected interface.
func (ic *InterfaceContext) Status(ctx context.Context) (*InterfaceDetails, error) {
	params := map[string]any{"interface": ic.name}
	status, err := goubus.Call[*InterfaceDetails](ctx, ic.manager.caller, ic.service(), "status", params)
	if err == nil {
		return status, nil
	}

	return ic.dumpStatus(ctx, err)
}

func (ic *InterfaceContext) dumpStatus(ctx context.Context, original error) (*InterfaceDetails, error) {
	interfaces, err := ic.manager.DumpInterfaces(ctx)
	if err != nil {
		return nil, original
	}
	for _, iface := range interfaces {
		if iface.Interface == ic.name {
			return &iface.InterfaceDetails, nil
		}
	}

	return nil, fmt.Errorf("%w: interface %q not found", goubus.ErrNotFound, ic.name)
}

func (ic *InterfaceContext) service() string {
	return "network.interface." + ic.name
}

// DeviceContext provides network device operations.
type DeviceContext struct {
	manager *Client
}

// Status retrieves status for one device or all devices.
func (dc *DeviceContext) Status(ctx context.Context, name string) (map[string]Device, error) {
	params := map[string]any{}
	if name != "" {
		params["name"] = name
		device, err := goubus.Call[Device](ctx, dc.manager.caller, "network.device", "status", params)
		if err != nil {
			return nil, err
		}

		return map[string]Device{name: device}, nil
	}

	return goubus.Call[map[string]Device](ctx, dc.manager.caller, "network.device", "status", params)
}

// SetAlias sets an alias for a network device.
func (dc *DeviceContext) SetAlias(ctx context.Context, request DeviceSetAliasRequest) error {
	return dc.manager.call(ctx, "network.device", "set_alias", request)
}

// SetState changes a network device's state.
func (dc *DeviceContext) SetState(ctx context.Context, request DeviceSetStateRequest) error {
	return dc.manager.call(ctx, "network.device", "set_state", request)
}

// STPInit initializes spanning tree on network devices.
func (dc *DeviceContext) STPInit(ctx context.Context) error {
	return dc.manager.call(ctx, "network.device", "stp_init", nil)
}

// WirelessContext provides wireless radio operations.
type WirelessContext struct {
	manager *Client
}

type wirelessStatusResponse struct {
	Radio map[string]RadioStatus `json:"radio"`
}

// Status retrieves the live status of wireless radios.
func (wc *WirelessContext) Status(ctx context.Context, device string) (map[string]RadioStatus, error) {
	params := optionalDevice(device)
	result, err := goubus.Call[wirelessStatusResponse](ctx, wc.manager.caller, "network.wireless", "status", params)
	if err != nil {
		return nil, err
	}

	return result.Radio, nil
}

// Up brings up a wireless radio.
func (wc *WirelessContext) Up(ctx context.Context, device string) error {
	return wc.manager.call(ctx, "network.wireless", "up", optionalDevice(device))
}

// Down brings down a wireless radio.
func (wc *WirelessContext) Down(ctx context.Context, device string) error {
	return wc.manager.call(ctx, "network.wireless", "down", optionalDevice(device))
}

// Reconf reconfigures a wireless radio.
func (wc *WirelessContext) Reconf(ctx context.Context, device string) error {
	return wc.manager.call(ctx, "network.wireless", "reconf", optionalDevice(device))
}

// Notify sends a notification to the wireless subsystem.
func (wc *WirelessContext) Notify(ctx context.Context, request WirelessNotifyRequest) error {
	return wc.manager.call(ctx, "network.wireless", "notify", request)
}

// Validate retrieves wireless radio validation information.
func (wc *WirelessContext) Validate(ctx context.Context, device string) (map[string]any, error) {
	return goubus.Call[map[string]any](ctx, wc.manager.caller, "network.wireless", "get_validate", optionalDevice(device))
}

// Retry retries wireless radio setup.
func (wc *WirelessContext) Retry(ctx context.Context, device string) error {
	return wc.manager.call(ctx, "network.wireless", "retry", optionalDevice(device))
}

func optionalDevice(device string) map[string]any {
	params := map[string]any{}
	if device != "" {
		params["device"] = device
	}

	return params
}

func (client *Client) call(ctx context.Context, service, method string, params any) error {
	_, err := client.caller.Call(ctx, service, method, params)

	return err
}
