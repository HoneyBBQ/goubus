// Package container provides a client for the ubus container service.
package container

import (
	"context"

	"github.com/honeybbq/goubus/v2"
)

// Client provides access to the ubus container service.
type Client struct {
	caller goubus.Caller
}

// New creates a container client for caller.
func New(caller goubus.Caller) *Client {
	if caller == nil {
		panic("goubus/container: nil caller")
	}

	return &Client{caller: caller}
}

// Set configures a container.
func (c *Client) Set(ctx context.Context, req SetRequest) error {
	_, err := c.caller.Call(ctx, "container", "set", req)

	return err
}

// Add adds a container.
func (c *Client) Add(ctx context.Context, req SetRequest) error {
	_, err := c.caller.Call(ctx, "container", "add", req)

	return err
}

// List retrieves the status of containers.
func (c *Client) List(ctx context.Context, name string, verbose bool) (map[string]any, error) {
	params := make(map[string]any)
	if name != "" {
		params["name"] = name
	}
	if verbose {
		params["verbose"] = true
	}

	return goubus.Call[map[string]any](ctx, c.caller, "container", "list", params)
}

// Delete removes a container instance.
func (c *Client) Delete(ctx context.Context, name, instance string) error {
	_, err := c.caller.Call(ctx, "container", "delete", map[string]any{"name": name, "instance": instance})

	return err
}

// State retrieves the state of a container.
func (c *Client) State(ctx context.Context, name string, spawn bool) (map[string]any, error) {
	params := map[string]any{"name": name, "spawn": spawn}

	return goubus.Call[map[string]any](ctx, c.caller, "container", "state", params)
}

// Features retrieves supported container features.
func (c *Client) Features(ctx context.Context) (map[string]any, error) {
	return goubus.Call[map[string]any](ctx, c.caller, "container", "get_features", nil)
}

// ConsoleSet sets the console for a container instance.
func (c *Client) ConsoleSet(ctx context.Context, name, instance string) error {
	_, err := c.caller.Call(ctx, "container", "console_set", map[string]any{"name": name, "instance": instance})

	return err
}

// ConsoleAttach attaches to the console of a container instance.
func (c *Client) ConsoleAttach(ctx context.Context, name, instance string) error {
	_, err := c.caller.Call(ctx, "container", "console_attach", map[string]any{"name": name, "instance": instance})

	return err
}

// SetRequest contains parameters for adding or configuring a container.
type SetRequest struct {
	// Instances contains instance-specific configuration.
	Instances map[string]any `json:"instances,omitempty"`
	// Data contains container configuration data.
	Data map[string]any `json:"data,omitempty"`
	// Name is the container name.
	Name string `json:"name"`
	// Script identifies the container script.
	Script string `json:"script,omitempty"`
	// Triggers contains container event triggers.
	Triggers []any `json:"triggers,omitempty"`
	// Validate contains validation constraints.
	Validate []any `json:"validate,omitempty"`
	// Autostart reports whether the container starts automatically.
	Autostart goubus.Bool `json:"autostart,omitempty"`
}
