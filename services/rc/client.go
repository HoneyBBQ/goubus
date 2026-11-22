// Package rc provides a client for the ubus init script service.
package rc

import (
	"context"

	"github.com/honeybbq/goubus/v2"
)

// Client provides access to the ubus init script service.
type Client struct {
	caller goubus.Caller
}

// New creates an init script client for caller.
func New(caller goubus.Caller) *Client {
	if caller == nil {
		panic("goubus/rc: nil caller")
	}

	return &Client{caller: caller}
}

// List retrieves available init scripts.
func (c *Client) List(ctx context.Context, name string, skipRunningCheck bool) (map[string]ListInfo, error) {
	params := make(map[string]any)
	if name != "" {
		params["name"] = name
	}
	if skipRunningCheck {
		params["skip_running_check"] = true
	}
	res, err := goubus.Call[map[string]ListInfo](ctx, c.caller, "rc", "list", params)
	if err != nil {
		return nil, err
	}

	return *res, nil
}

// Init performs an init script action.
func (c *Client) Init(ctx context.Context, name, action string) error {
	_, err := c.caller.Call(ctx, "rc", "init", initRequest{Name: name, Action: action})

	return err
}

// ListInfo contains init script status and ordering information.
type ListInfo struct {
	// Start is the startup priority.
	Start int `json:"start"`
	// Stop is the shutdown priority.
	Stop int `json:"stop"`
	// Running indicates whether the script is running.
	Running goubus.Bool `json:"running"`
	// Enabled indicates whether the script is enabled.
	Enabled goubus.Bool `json:"enabled"`
}

// initRequest contains parameters for an init script action.
type initRequest struct {
	// Name is the init script name.
	Name string `json:"name"`
	// Action is the requested init script action.
	Action string `json:"action"`
}
