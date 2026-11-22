// Package service provides a client for the ubus service manager.
package service

import (
	"context"

	"github.com/honeybbq/goubus/v2"
)

// Client provides access to the ubus service manager.
type Client struct {
	caller goubus.Caller
}

// New creates a service client for caller.
func New(caller goubus.Caller) *Client {
	if caller == nil {
		panic("goubus/service: nil caller")
	}

	return &Client{caller: caller}
}

// List retrieves the status of services.
func (c *Client) List(ctx context.Context, name string, verbose bool) (map[string]Info, error) {
	params := make(map[string]any)
	if name != "" {
		params["name"] = name
	}
	if verbose {
		params["verbose"] = true
	}
	res, err := goubus.Call[map[string]Info](ctx, c.caller, "service", "list", params)
	if err != nil {
		return nil, err
	}

	return *res, nil
}

// Delete removes a service instance.
func (c *Client) Delete(ctx context.Context, name, instance string) error {
	_, err := c.caller.Call(ctx, "service", "delete", map[string]any{"name": name, "instance": instance})

	return err
}

// Signal sends a Unix signal to a service instance.
func (c *Client) Signal(ctx context.Context, name, instance string, signal int) error {
	_, err := c.caller.Call(ctx, "service", "signal", map[string]any{"name": name, "instance": instance, "signal": signal})

	return err
}

// Set configures a service.
func (c *Client) Set(ctx context.Context, req SetRequest) error {
	_, err := c.caller.Call(ctx, "service", "set", req)

	return err
}

// Add adds a service.
func (c *Client) Add(ctx context.Context, req SetRequest) error {
	_, err := c.caller.Call(ctx, "service", "add", req)

	return err
}

// UpdateStart marks the start of a service update.
func (c *Client) UpdateStart(ctx context.Context, name string) error {
	_, err := c.caller.Call(ctx, "service", "update_start", map[string]any{"name": name})

	return err
}

// UpdateComplete marks the completion of a service update.
func (c *Client) UpdateComplete(ctx context.Context, name string) error {
	_, err := c.caller.Call(ctx, "service", "update_complete", map[string]any{"name": name})

	return err
}

// Event sends an event to the service.
func (c *Client) Event(ctx context.Context, req EventRequest) error {
	_, err := c.caller.Call(ctx, "service", "event", req)

	return err
}

// Validate validates the service configuration.
func (c *Client) Validate(ctx context.Context, req ValidateRequest) (map[string]any, error) {
	res, err := goubus.Call[map[string]any](ctx, c.caller, "service", "validate", req)
	if err != nil {
		return nil, err
	}

	return *res, nil
}

// Data retrieves typed data from a service instance.
func (c *Client) Data(ctx context.Context, name, instance, dataType string) (map[string]any, error) {
	params := map[string]any{"name": name, "instance": instance, "type": dataType}
	res, err := goubus.Call[map[string]any](ctx, c.caller, "service", "get_data", params)
	if err != nil {
		return nil, err
	}

	return *res, nil
}

// SetData sets data for a service instance.
func (c *Client) SetData(ctx context.Context, name, instance string, data map[string]any) error {
	_, err := c.caller.Call(ctx, "service", "set_data", map[string]any{"name": name, "instance": instance, "data": data})

	return err
}

// State retrieves the spawn state of a service.
func (c *Client) State(ctx context.Context, name string, spawn bool) (map[string]any, error) {
	params := map[string]any{"name": name, "spawn": spawn}
	res, err := goubus.Call[map[string]any](ctx, c.caller, "service", "state", params)
	if err != nil {
		return nil, err
	}

	return *res, nil
}

// Watchdog configures the watchdog for a service instance.
func (c *Client) Watchdog(ctx context.Context, name, instance string, mode, timeout int) error {
	params := map[string]any{
		"name": name, "instance": instance, "mode": mode, "timeout": timeout,
	}
	_, err := c.caller.Call(ctx, "service", "watchdog", params)

	return err
}

// Info contains information about a service.
type Info struct {
	// Instances maps instance names to their information.
	Instances map[string]Instance `json:"instances"`
}

// Instance contains information about a service instance.
type Instance struct {
	// Command is the process command line.
	Command []string `json:"command"`
	// Pid is the process identifier.
	Pid int `json:"pid"`
	// Running indicates whether the instance is running.
	Running goubus.Bool `json:"running"`
}

// Respawn contains a service respawn configuration.
type Respawn struct {
	// Threshold is the respawn failure threshold.
	Threshold int `json:"threshold"`
	// Timeout is the respawn timeout.
	Timeout int `json:"timeout"`
	// Retry is the number of respawn retries.
	Retry int `json:"retry"`
}

// Jail contains service sandboxing configuration.
type Jail struct {
	// Name is the jail name.
	Name string `json:"name"`
}

// Limits contains service resource limits.
type Limits struct {
	// NoFile is the open file descriptor limit.
	NoFile string `json:"nofile,omitempty"`
}

// SetRequest contains parameters for configuring a service.
type SetRequest struct {
	Instances map[string]any `json:"instances,omitempty"`
	Data      map[string]any `json:"data,omitempty"`
	Name      string         `json:"name"`
	Script    string         `json:"script,omitempty"`
	Triggers  []any          `json:"triggers,omitempty"`
	Validate  []any          `json:"validate,omitempty"`
	Autostart goubus.Bool    `json:"autostart,omitempty"`
}

// EventRequest contains parameters for a service event.
type EventRequest struct {
	// Data contains event-specific data.
	Data map[string]any `json:"data"`
	// Type is the event type.
	Type string `json:"type"`
}

// ValidateRequest contains parameters for service validation.
type ValidateRequest struct {
	// Package is the package name.
	Package string `json:"package"`
	// Type is the configuration type.
	Type string `json:"type"`
	// Service is the service name.
	Service string `json:"service"`
}
