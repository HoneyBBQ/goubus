// Package luci provides LuCI and luci-rpc ubus operations.
package luci

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"github.com/honeybbq/goubus/v2"
)

// Option configures a LuCI client.
type Option interface {
	apply(client *Client)
}

type clientOption func(*Client)

func (option clientOption) apply(client *Client) { option(client) }

// WithTimeMethod fixes the method used to retrieve the system time.
func WithTimeMethod(method string) Option {
	return clientOption(func(client *Client) {
		methodCopy := method
		client.fixedTimeMethod = &methodCopy
	})
}

// Client provides LuCI and luci-rpc ubus operations.
type Client struct {
	timeMethod      atomic.Pointer[string]
	caller          goubus.Caller
	fixedTimeMethod *string
}

// New creates a LuCI client.
func New(caller goubus.Caller, opts ...Option) *Client {
	if caller == nil {
		panic("goubus/luci: nil caller")
	}
	client := &Client{caller: caller}
	for _, option := range opts {
		if option != nil {
			option.apply(client)
		}
	}

	return client
}

// Version retrieves LuCI version metadata.
func (client *Client) Version(ctx context.Context) (*Version, error) {
	return goubus.Call[Version](ctx, client.caller, "luci", "getVersion", nil)
}

// Time retrieves the current system time, adapting to the available ubus method.
func (client *Client) Time(ctx context.Context) (time.Time, error) {
	method := ""
	if client.fixedTimeMethod != nil {
		method = *client.fixedTimeMethod
	}
	if method == "" {
		if cached := client.timeMethod.Load(); cached != nil {
			method = *cached
		} else {
			method = "getUnixtime"
		}
	}
	unixTime, err := readTime(ctx, client, method)
	if err == nil {
		if client.fixedTimeMethod == nil {
			cacheTimeMethod(client, method)
		}

		return unixTime, nil
	}
	if client.fixedTimeMethod != nil || !timeMethodUnavailable(err) || method != "getUnixtime" {
		return time.Time{}, err
	}
	unixTime, err = readTime(ctx, client, "getLocaltime")
	if err == nil && client.fixedTimeMethod == nil {
		cacheTimeMethod(client, "getLocaltime")
	}

	return unixTime, err
}

func readTime(ctx context.Context, client *Client, method string) (time.Time, error) {
	result, err := goubus.Call[timeResponse](ctx, client.caller, "luci", method, nil)
	if err != nil {
		return time.Time{}, err
	}

	return time.Unix(result.Time, 0), nil
}

func cacheTimeMethod(client *Client, method string) {
	stored := new(string)
	*stored = method
	client.timeMethod.CompareAndSwap(nil, stored)
}

func timeMethodUnavailable(err error) bool {
	return errors.Is(err, goubus.ErrMethodNotFound) || errors.Is(err, goubus.ErrInvalidCommand) ||
		errors.Is(err, goubus.ErrPermissionDenied)
}

type timeResponse struct {
	Time int64 `json:"result"`
}

// InitList retrieves initialization scripts, optionally filtered by name.
func (client *Client) InitList(ctx context.Context, name string) (map[string]any, error) {
	params := map[string]any{}
	if name != "" {
		params["name"] = name
	}

	return callValue[map[string]any](ctx, client.caller, "luci", "getInitList", params)
}

// Timezones retrieves the available system timezones.
func (client *Client) Timezones(ctx context.Context) (map[string]any, error) {
	return callValue[map[string]any](ctx, client.caller, "luci", "getTimezones", nil)
}

// LEDs retrieves configured system LEDs.
func (client *Client) LEDs(ctx context.Context) (map[string]LED, error) {
	return callValue[map[string]LED](ctx, client.caller, "luci", "getLEDs", nil)
}

// USBDevices retrieves connected USB devices and available ports.
func (client *Client) USBDevices(ctx context.Context) (*USBInfo, error) {
	return goubus.Call[USBInfo](ctx, client.caller, "luci", "getUSBDevices", nil)
}

// ConntrackHelpers retrieves connection tracking helper names.
func (client *Client) ConntrackHelpers(ctx context.Context) ([]string, error) {
	return callValue[[]string](ctx, client.caller, "luci", "getConntrackHelpers", nil)
}

// Features retrieves system feature flags and their subfeatures.
func (client *Client) Features(ctx context.Context) (map[string]Feature, error) {
	return callValue[map[string]Feature](ctx, client.caller, "luci", "getFeatures", nil)
}

// SwconfigFeatures retrieves feature details for a switch.
func (client *Client) SwconfigFeatures(ctx context.Context, switchName string) (map[string]any, error) {
	params := map[string]any{"switch": switchName}

	return callValue[map[string]any](ctx, client.caller, "luci", "getSwconfigFeatures", params)
}

// SwconfigPortState retrieves port states for a switch.
func (client *Client) SwconfigPortState(ctx context.Context, switchName string) (map[string]any, error) {
	params := map[string]any{"switch": switchName}

	return callValue[map[string]any](ctx, client.caller, "luci", "getSwconfigPortState", params)
}

// BlockDevices retrieves block devices keyed by device name.
func (client *Client) BlockDevices(ctx context.Context) (map[string]BlockDevice, error) {
	return callValue[map[string]BlockDevice](ctx, client.caller, "luci", "getBlockDevices", nil)
}

// MountPoints retrieves mounted filesystem statistics.
func (client *Client) MountPoints(ctx context.Context) ([]MountPoint, error) {
	return callResult[[]MountPoint](ctx, client.caller, "luci", "getMountPoints", nil)
}

// RealtimeStats retrieves traffic counters for a device.
func (client *Client) RealtimeStats(ctx context.Context, mode, device string) (*RealtimeStats, error) {
	params := map[string]any{"mode": mode, "device": device}

	return goubus.Call[RealtimeStats](ctx, client.caller, "luci", "getRealtimeStats", params)
}

// ConntrackList retrieves the current connection tracking entries.
func (client *Client) ConntrackList(ctx context.Context) ([]any, error) {
	return callValue[[]any](ctx, client.caller, "luci", "getConntrackList", nil)
}

// Processes retrieves system processes.
func (client *Client) Processes(ctx context.Context) ([]Process, error) {
	return callResult[[]Process](ctx, client.caller, "luci", "getProcessList", nil)
}

// BuiltinEthernetPorts retrieves built-in Ethernet port data.
func (client *Client) BuiltinEthernetPorts(ctx context.Context) ([]any, error) {
	return callValue[[]any](ctx, client.caller, "luci", "getBuiltinEthernetPorts", nil)
}

// Odhcp6cStats retrieves odhcp6c statistics.
func (client *Client) Odhcp6cStats(ctx context.Context) (map[string]any, error) {
	return callResult[map[string]any](ctx, client.caller, "luci", "getOdhcp6cStats", nil)
}

// NetworkDevices retrieves detailed network device information.
func (client *Client) NetworkDevices(ctx context.Context) (map[string]NetworkDevice, error) {
	return callValue[map[string]NetworkDevice](ctx, client.caller, "luci-rpc", "getNetworkDevices", nil)
}

// WirelessDevices retrieves wireless radio and interface information.
func (client *Client) WirelessDevices(ctx context.Context) (map[string]WirelessDevice, error) {
	return callValue[map[string]WirelessDevice](ctx, client.caller, "luci-rpc", "getWirelessDevices", nil)
}

// HostHints retrieves host names and addresses.
func (client *Client) HostHints(ctx context.Context) (map[string]HostHint, error) {
	return callValue[map[string]HostHint](ctx, client.caller, "luci-rpc", "getHostHints", nil)
}

// DUIDHints retrieves DHCP unique identifier hints.
func (client *Client) DUIDHints(ctx context.Context) (map[string]any, error) {
	return callValue[map[string]any](ctx, client.caller, "luci-rpc", "getDUIDHints", nil)
}

// BoardJSON retrieves board hardware information.
func (client *Client) BoardJSON(ctx context.Context) (*BoardJSON, error) {
	return goubus.Call[BoardJSON](ctx, client.caller, "luci-rpc", "getBoardJSON", nil)
}

// DHCPLeases retrieves DHCP leases, optionally filtered by address family.
func (client *Client) DHCPLeases(ctx context.Context, family int) (*DHCPLeases, error) {
	params := map[string]any{}
	if family != 0 {
		params["family"] = family
	}

	return goubus.Call[DHCPLeases](ctx, client.caller, "luci-rpc", "getDHCPLeases", params)
}

// SetLocaltime sets the device's local Unix time.
func (client *Client) SetLocaltime(ctx context.Context, value time.Time) error {
	return client.call(ctx, "setLocaltime", map[string]any{"localtime": value.Unix()})
}

// SetInitAction performs an action on an initialization script.
func (client *Client) SetInitAction(ctx context.Context, name, action string) error {
	return client.call(ctx, "setInitAction", map[string]any{"name": name, "action": action})
}

// SetPassword sets a user's password.
func (client *Client) SetPassword(ctx context.Context, username, password string) error {
	return client.call(ctx, "setPassword", map[string]any{"username": username, "password": password})
}

// SetBlockDetect starts block device detection.
func (client *Client) SetBlockDetect(ctx context.Context) error {
	return client.call(ctx, "setBlockDetect", nil)
}

func (client *Client) call(ctx context.Context, method string, params any) error {
	_, err := client.caller.Call(ctx, "luci", method, params)

	return err
}

func callValue[T any](ctx context.Context, caller goubus.Caller, service, method string, params any) (T, error) {
	result, err := goubus.Call[T](ctx, caller, service, method, params)
	if err != nil {
		var zero T

		return zero, err
	}

	return *result, nil
}

func callResult[T any](ctx context.Context, caller goubus.Caller, service, method string, params any) (T, error) {
	result, err := goubus.Call[resultEnvelope[T]](ctx, caller, service, method, params)
	if err != nil {
		var zero T

		return zero, err
	}

	return result.Result, nil
}

type resultEnvelope[T any] struct {
	Result T `json:"result"`
}
