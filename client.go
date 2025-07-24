package goubus

import (
	"github.com/honeybbq/goubus/types"
)

// Client is the main struct for ubus connection and API access.
// All Manager types are created through factory methods on this client.
type Client struct {
	// Shared RPC client for all operations
	caller types.Transport
}

// NewClient creates a new client with the provided caller interface.
// This allows for dependency injection and easy testing.
func NewClient(caller types.Transport) *Client {
	return &Client{
		caller: caller,
	}
}

// Caller returns the underlying caller interface.
// This can be useful for advanced usage or testing.
func (c *Client) Caller() types.Transport {
	return c.caller
}

func (c *Client) Close() error {
	return c.caller.Close()
}
