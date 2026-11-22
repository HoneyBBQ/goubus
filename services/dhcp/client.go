// Package dhcp provides DHCP ubus operations and lease models.
package dhcp

import (
	"context"
	"encoding/json"
	"errors"
	"sync/atomic"

	"github.com/honeybbq/goubus/v2"
)

// DUIDFormat selects the wire representation used for DHCP DUID arguments.
type DUIDFormat uint8

const (
	// DUIDAuto detects the accepted DUID representation at runtime.
	DUIDAuto DUIDFormat = iota
	// DUIDString sends DUID values as strings.
	DUIDString
	// DUIDArray sends DUID values as one-element string arrays.
	DUIDArray
)

// Option configures a DHCP client.
type Option interface {
	apply(client *Client)
}

type clientOption func(*Client)

func (option clientOption) apply(client *Client) { option(client) }

// WithDUIDFormat fixes the DUID representation used for add_lease calls.
func WithDUIDFormat(format DUIDFormat) Option {
	return clientOption(func(client *Client) { client.format = format })
}

// Client provides DHCP ubus operations.
type Client struct {
	caller  goubus.Caller
	format  DUIDFormat
	duidFmt atomic.Uint32
}

// New creates a DHCP client.
func New(caller goubus.Caller, opts ...Option) *Client {
	if caller == nil {
		panic("goubus/dhcp: nil caller")
	}
	client := &Client{caller: caller}
	for _, option := range opts {
		if option != nil {
			option.apply(client)
		}
	}

	return client
}

// AddLease creates a static DHCP lease.
func (client *Client) AddLease(ctx context.Context, request AddLeaseRequest) error {
	params := leaseParams(request)
	if request.DUID == "" { //nolint:whitespace // Keep return separated per nlreturn.

		return callAddLease(ctx, client, params)
	}
	if client.format == DUIDString {
		params["duid"] = request.DUID

		return callAddLease(ctx, client, params)
	}
	if client.format == DUIDArray {
		params["duid"] = []string{request.DUID}

		return callAddLease(ctx, client, params)
	}

	return addLeaseAuto(ctx, client, params, request.DUID)
}

func leaseParams(request AddLeaseRequest) map[string]any {
	params := make(map[string]any, 6)
	if request.IP != "" {
		params["ip"] = request.IP
	}
	if request.MAC != "" {
		params["mac"] = []string{request.MAC}
	}
	if request.HostID != "" {
		params["hostid"] = request.HostID
	}
	if request.LeaseTime != "" {
		params["leasetime"] = request.LeaseTime
	}
	if request.Name != "" {
		params["name"] = request.Name
	}

	return params
}

func addLeaseAuto(ctx context.Context, client *Client, params map[string]any, duid string) error {
	if cached := client.duidFmt.Load(); cached != 0 {
		params["duid"] = duidArgument(cached, duid)

		return callAddLease(ctx, client, params)
	}
	if format := signatureFormat(ctx, client); format != 0 {
		client.duidFmt.Store(format)
		params["duid"] = duidArgument(format, duid)

		return callAddLease(ctx, client, params)
	}
	params["duid"] = []string{duid}
	err := callAddLease(ctx, client, params)
	if err == nil {
		client.duidFmt.Store(uint32(DUIDArray))

		return nil
	}
	if !errors.Is(err, goubus.ErrInvalidParameter) {
		return err
	}
	params["duid"] = duid
	err = callAddLease(ctx, client, params)
	if err == nil {
		client.duidFmt.Store(uint32(DUIDString))
	}

	return err
}

func signatureFormat(ctx context.Context, client *Client) uint32 {
	signatures, err := goubus.List(ctx, client.caller, "dhcp")
	if err != nil {
		return 0
	}
	argument := signatures["dhcp"]["add_lease"]["duid"]
	switch argument {
	case goubus.ArgTypeArray:
		return uint32(DUIDArray)
	case goubus.ArgTypeString:
		return uint32(DUIDString)
	case goubus.ArgTypeNumber, goubus.ArgTypeBoolean, goubus.ArgTypeObject, goubus.ArgTypeUnknown:
		return 0
	default:
		return 0
	}
}

func duidArgument(format uint32, duid string) any {
	if format == uint32(DUIDArray) {
		return []string{duid}
	}

	return duid
}

func callAddLease(ctx context.Context, client *Client, params map[string]any) error {
	_, err := client.caller.Call(ctx, "dhcp", "add_lease", params)

	return err
}

// IPv4Leases retrieves active IPv4 leases with their interface names.
func (client *Client) IPv4Leases(ctx context.Context) ([]IPv4Lease, error) {
	raw, err := client.callRaw(ctx, "ipv4leases")
	if err != nil {
		return nil, err
	}

	return decodeLeases[IPv4Lease](raw)
}

// IPv6Leases retrieves active IPv6 leases with their interface names.
func (client *Client) IPv6Leases(ctx context.Context) ([]IPv6Lease, error) {
	raw, err := client.callRaw(ctx, "ipv6leases")
	if err != nil {
		return nil, err
	}

	return decodeLeases[IPv6Lease](raw)
}

// IPv6RA retrieves IPv6 router advertisement data with interface names.
func (client *Client) IPv6RA(ctx context.Context) ([]IPv6RA, error) {
	raw, err := client.callRaw(ctx, "ipv6ra")
	if err != nil {
		return nil, err
	}

	return decodeLeases[IPv6RA](raw)
}

func (client *Client) callRaw(ctx context.Context, method string) (json.RawMessage, error) {
	result, err := client.caller.Call(ctx, "dhcp", method, nil)
	if err != nil {
		return nil, err
	}
	var raw json.RawMessage
	err = result.Unmarshal(&raw)
	if err != nil {
		return nil, err
	}

	return raw, nil
}

func decodeLeases[T any](raw json.RawMessage) ([]T, error) {
	var outer map[string]json.RawMessage
	err := json.Unmarshal(raw, &outer)
	if err != nil {
		return nil, err
	}
	if nested, ok := outer["device"]; ok {
		err = json.Unmarshal(nested, &outer)
		if err != nil {
			return nil, err
		}
	}
	if nested, ok := outer["interfaces"]; ok {
		err = json.Unmarshal(nested, &outer)
		if err != nil {
			return nil, err
		}
	}
	result := make([]T, 0)
	for name, entry := range outer {
		if name == "device" || name == "interfaces" {
			continue
		}
		leases, err := decodeInterfaceEntries[T](entry)
		if err != nil {
			return nil, err
		}
		for i := range leases {
			setInterface(&leases[i], name)
		}
		result = append(result, leases...)
	}

	return result, nil
}

func decodeInterfaceEntries[T any](raw json.RawMessage) ([]T, error) {
	if len(raw) > 0 && raw[0] == '{' {
		var object map[string]json.RawMessage
		err := json.Unmarshal(raw, &object)
		if err != nil {
			return nil, err
		}
		if leases, ok := object["leases"]; ok {
			raw = leases
		} else {
			entries := make([]json.RawMessage, 0, len(object))
			for _, value := range object {
				entries = append(entries, value)
			}

			return decodeEntries[T](entries)
		}
	}
	var entries []json.RawMessage
	err := json.Unmarshal(raw, &entries)
	if err != nil {
		return nil, err
	}

	return decodeEntries[T](entries)
}

func decodeEntries[T any](raw []json.RawMessage) ([]T, error) {
	values := make([]T, 0, len(raw))
	for _, item := range raw {
		var value T
		err := json.Unmarshal(item, &value)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}

	return values, nil
}

func setInterface[T any](value *T, name string) {
	switch typed := any(value).(type) {
	case *IPv4Lease:
		typed.Interface = name
	case *IPv6Lease:
		typed.Interface = name
	case *IPv6RA:
		typed.Interface = name
	}
}
