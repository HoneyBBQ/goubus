// Package session provides a client for the ubus session service.
package session

import (
	"context"
	"time"

	"github.com/honeybbq/goubus/v2"
)

const ubusRPCSessionKey = "ubus_rpc_session"

// Client provides access to the ubus session service.
type Client struct {
	caller goubus.Caller
}

// New creates a session client for caller.
func New(caller goubus.Caller) *Client {
	if caller == nil {
		panic("goubus/session: nil caller")
	}

	return &Client{caller: caller}
}

// Create creates a new session.
func (c *Client) Create(ctx context.Context, timeout int) (*Data, error) {
	res, err := goubus.Call[Data](ctx, c.caller, "session", "create", map[string]any{"timeout": timeout})
	if err != nil {
		return nil, err
	}
	res.ExpireTime = time.Now().Add(time.Duration(res.Timeout) * time.Second)

	return res, nil
}

// List retrieves active sessions.
func (c *Client) List(ctx context.Context) (map[string]any, error) {
	res, err := goubus.Call[map[string]any](ctx, c.caller, "session", "list", nil)
	if err != nil {
		return nil, err
	}

	return *res, nil
}

// Grant grants access to a session.
func (c *Client) Grant(ctx context.Context, req GrantRequest) error {
	_, err := c.caller.Call(ctx, "session", "grant", req)

	return err
}

// Revoke revokes access from a session.
func (c *Client) Revoke(ctx context.Context, req GrantRequest) error {
	_, err := c.caller.Call(ctx, "session", "revoke", req)

	return err
}

// Access checks access for a session.
func (c *Client) Access(ctx context.Context, req AccessRequest) (bool, error) {
	res, err := goubus.Call[map[string]bool](ctx, c.caller, "session", "access", req)
	if err != nil {
		return false, err
	}

	return (*res)["access"], nil
}

// Set sets session values.
func (c *Client) Set(ctx context.Context, session string, values map[string]any) error {
	_, err := c.caller.Call(ctx, "session", "set", map[string]any{ubusRPCSessionKey: session, "values": values})

	return err
}

// Get retrieves session values.
func (c *Client) Get(ctx context.Context, session string, keys []string) (map[string]any, error) {
	params := map[string]any{ubusRPCSessionKey: session, "keys": keys}
	res, err := goubus.Call[map[string]any](ctx, c.caller, "session", "get", params)
	if err != nil {
		return nil, err
	}

	return *res, nil
}

// Unset removes values from a session.
func (c *Client) Unset(ctx context.Context, session string, keys []string) error {
	_, err := c.caller.Call(ctx, "session", "unset", map[string]any{ubusRPCSessionKey: session, "keys": keys})

	return err
}

// Destroy destroys a session.
func (c *Client) Destroy(ctx context.Context, session string) error {
	_, err := c.caller.Call(ctx, "session", "destroy", map[string]any{ubusRPCSessionKey: session})

	return err
}

// Login performs a session login.
func (c *Client) Login(ctx context.Context, req LoginRequest) (*Data, error) {
	res, err := goubus.Call[Data](ctx, c.caller, "session", "login", req)
	if err != nil {
		return nil, err
	}
	res.ExpireTime = time.Now().Add(time.Duration(res.Timeout) * time.Second)

	return res, nil
}

// Data contains session information.
type Data struct {
	ExpireTime     time.Time `json:"-"`
	UbusRPCSession string    `json:"ubus_rpc_session"`
	Timeout        int       `json:"timeout"`
}

// ACLs contains session access control lists.
type ACLs struct {
	// Ubus contains ubus access control entries.
	Ubus map[string][]string `json:"ubus"`
	// Uci contains UCI access control entries.
	Uci map[string][]string `json:"uci"`
}

// GrantRequest contains parameters for granting or revoking access.
type GrantRequest struct {
	Session string   `json:"ubus_rpc_session"`
	Scope   string   `json:"scope"`
	Objects []string `json:"objects"`
}

// AccessRequest contains parameters for checking session access.
type AccessRequest struct {
	// Session is the session identifier.
	Session string `json:"ubus_rpc_session"`
	// Scope is the access control scope.
	Scope string `json:"scope"`
	// Object is the ubus object name.
	Object string `json:"object"`
	// Function is the ubus method name.
	Function string `json:"function"`
}

// LoginRequest contains parameters for session login.
type LoginRequest struct {
	// Username is the account name.
	Username string `json:"username"`
	// Password is the account password.
	Password string `json:"password"`
	// Timeout is the requested session timeout in seconds.
	Timeout int `json:"timeout,omitempty"`
}
