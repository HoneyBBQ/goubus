// Package rpcsys provides a client for the ubus rpc-sys service.
package rpcsys

import (
	"context"

	"github.com/honeybbq/goubus/v2"
)

// Client provides access to the rpc-sys service.
type Client struct {
	caller goubus.Caller
}

// New creates an rpc-sys client for caller.
func New(caller goubus.Caller) *Client {
	if caller == nil {
		panic("goubus/rpcsys: nil caller")
	}

	return &Client{caller: caller}
}

// PackageList retrieves the installed package list.
func (c *Client) PackageList(ctx context.Context, all bool) (map[string]any, error) {
	res, err := goubus.Call[map[string]any](ctx, c.caller, "rpc-sys", "packagelist", map[string]any{"all": all})
	if err != nil {
		return nil, err
	}

	return *res, nil
}

// PasswordSet sets the password for a user.
func (c *Client) PasswordSet(ctx context.Context, user, password string) error {
	_, err := c.caller.Call(ctx, "rpc-sys", "password_set", map[string]any{"user": user, "password": password})

	return err
}

// Factory performs a factory reset.
func (c *Client) Factory(ctx context.Context) error {
	_, err := c.caller.Call(ctx, "rpc-sys", "factory", nil)

	return err
}

// UpgradeStart starts a system upgrade.
func (c *Client) UpgradeStart(ctx context.Context, keep bool) error {
	_, err := c.caller.Call(ctx, "rpc-sys", "upgrade_start", map[string]any{"keep": keep})

	return err
}

// UpgradeTest tests a system upgrade image.
func (c *Client) UpgradeTest(ctx context.Context) error {
	_, err := c.caller.Call(ctx, "rpc-sys", "upgrade_test", nil)

	return err
}

// UpgradeClean cleans up after a system upgrade.
func (c *Client) UpgradeClean(ctx context.Context) error {
	_, err := c.caller.Call(ctx, "rpc-sys", "upgrade_clean", nil)

	return err
}

// Reboot reboots the system.
func (c *Client) Reboot(ctx context.Context) error {
	_, err := c.caller.Call(ctx, "rpc-sys", "reboot", nil)

	return err
}
