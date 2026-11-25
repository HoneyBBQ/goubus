package goubus

import (
	"cmp"
	"log/slog"
	"time"
)

// Option configures a Client.
type Option interface {
	apply(client *Client)
}

type optionFunc func(*Client)

// WithLogger sets the structured logger for the client.
func WithLogger(logger *slog.Logger) Option {
	return optionFunc(func(client *Client) {
		client.logger = cmp.Or(logger, discardLogger)
	})
}

// WithDialTimeout sets the timeout for connecting to the socket.
// A zero or negative timeout disables this timeout; context cancellation and deadlines still apply.
func WithDialTimeout(timeout time.Duration) Option {
	return optionFunc(func(client *Client) {
		client.dialTimeout = timeout
	})
}

// WithReadTimeout sets the per-request timeout waiting for a ubus reply.
// A zero or negative timeout disables this timeout; context cancellation and deadlines still apply.
func WithReadTimeout(timeout time.Duration) Option {
	return optionFunc(func(client *Client) {
		client.readTimeout = timeout
	})
}

// WithWriteTimeout sets the timeout for writing frames to the socket.
// A zero or negative timeout disables this timeout; context cancellation and deadlines still apply.
func WithWriteTimeout(timeout time.Duration) Option {
	return optionFunc(func(client *Client) {
		client.writeTimeout = timeout
	})
}

func (f optionFunc) apply(client *Client) {
	f(client)
}
