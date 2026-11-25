package rpc

import (
	"cmp"
	"log/slog"
	"net/http"
)

// Option configures a Client.
type Option interface {
	apply(client *Client)
}

type optionFunc func(*Client)

// WithLogger configures the structured logger used by Client.
func WithLogger(logger *slog.Logger) Option {
	return optionFunc(func(client *Client) {
		client.logger = cmp.Or(logger, discardLogger)
	})
}

// WithHTTPClient replaces the underlying HTTP client used for requests.
func WithHTTPClient(httpClient *http.Client) Option {
	return optionFunc(func(client *Client) {
		client.httpClient = httpClient
	})
}

// WithEndpoint overrides the URL path used for JSON-RPC requests (default "/ubus").
func WithEndpoint(path string) Option {
	return optionFunc(func(client *Client) {
		client.endpoint = path
	})
}

// WithAutoReauth enables or disables automatic single re-authentication and retry
// when a cached rpcd session is rejected with JSON-RPC -32002 (for example after
// a router reboot). Disabled by default so session-scoped state loss is not hidden.
func WithAutoReauth(enabled bool) Option {
	return optionFunc(func(client *Client) {
		client.autoReauth = enabled
	})
}

func (f optionFunc) apply(client *Client) {
	f(client)
}
