// Package log provides a client for the ubus system log service.
package log

import (
	"context"

	"github.com/honeybbq/goubus/v2"
)

// Client provides access to the ubus system log service.
type Client struct {
	caller goubus.Caller
}

// New creates a log client for caller.
func New(caller goubus.Caller) *Client {
	if caller == nil {
		panic("goubus/log: nil caller")
	}

	return &Client{caller: caller}
}

// Read retrieves log entries.
func (c *Client) Read(ctx context.Context, lines int, stream bool, oneshot bool) (*Log, error) {
	params := map[string]any{"lines": lines, "stream": stream, "oneshot": oneshot}

	return goubus.Call[*Log](ctx, c.caller, "log", "read", params)
}

// Write sends a log entry.
func (c *Client) Write(ctx context.Context, event string) error {
	_, err := c.caller.Call(ctx, "log", "write", map[string]any{"event": event})

	return err
}

// Log represents system log entries.
type Log struct {
	// Log contains the returned log entries.
	Log []Data `json:"log"`
}

// Data represents a single log entry.
type Data struct {
	// Text is the log entry text.
	Text string `json:"text"`
	// Time is the entry time as a Unix timestamp.
	Time int64 `json:"time"`
}
