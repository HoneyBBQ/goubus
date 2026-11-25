package rpc

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/honeybbq/goubus/v2"
	"github.com/honeybbq/goubus/v2/internal/errs"
)

// DefaultEndpoint is the default HTTP path exposed by uhttpd-mod-ubus.
const DefaultEndpoint = "/ubus"

const defaultTimeout = 30 * time.Second

var (
	_ goubus.Transport = (*Client)(nil)
	_ goubus.Lister    = (*Client)(nil)

	discardLogger = slog.New(slog.DiscardHandler)
)

// Client calls ubus remotely through an uhttpd-mod-ubus JSON-RPC endpoint and
// manages its rpcd authentication session.
type Client struct {
	logger      *slog.Logger
	httpClient  *http.Client
	host        string
	username    string
	password    string
	endpoint    string
	sessionData sessionMetadata
	rwMutex     sync.RWMutex
	id          atomic.Uint64
	autoReauth  bool
	closed      bool
}

// NewClient creates an authenticated HTTP/HTTPS JSON-RPC gateway client.
func NewClient(ctx context.Context, host, username, password string, opts ...Option) (*Client, error) {
	ctx = normalizeContext(ctx)
	client := &Client{
		host:       host,
		username:   username,
		password:   password,
		logger:     discardLogger,
		httpClient: &http.Client{Timeout: defaultTimeout},
	}

	for _, opt := range opts {
		if opt != nil {
			opt.apply(client)
		}
	}
	client.httpClient = cmp.Or(client.httpClient, &http.Client{Timeout: defaultTimeout})

	endpoint, err := normalizeEndpoint(host, client.endpoint)
	if err != nil {
		return nil, errs.Wrapf(goubus.ErrInvalidParameter, "RPC endpoint: %v", err)
	}
	client.endpoint = endpoint

	err = client.Authenticate(ctx)
	if err != nil {
		return nil, errs.Wrapf(err, "failed to authenticate")
	}

	return client, nil
}

// Call invokes one ubus object method using the current rpcd session.
func (c *Client) Call(ctx context.Context, service, method string, data any) (goubus.Result, error) {
	ctx = normalizeContext(ctx)
	if service == "" || method == "" {
		return nil, errs.Wrapf(goubus.ErrInvalidParameter, "service and method required")
	}
	if c.isClosed() {
		return nil, goubus.ErrClosed
	}

	sessionID, err := c.getValidSessionID(ctx)
	if err != nil {
		return nil, err
	}

	result, explicitSession, err := c.rawCallWithMode(ctx, sessionID, service, method, data)
	if err != nil && c.autoReauth && !explicitSession && isRPCAccessDenied(err) {
		sessionID, err = c.reauthenticateStaleSession(ctx, sessionID)
		if err != nil {
			return nil, err
		}
		result, _, err = c.rawCallWithMode(ctx, sessionID, service, method, data)
	}
	if err != nil {
		return nil, err
	}

	c.slideSessionExpiry(sessionID)

	return result, nil
}

// Authenticate forces a session.login request and replaces the cached session.
func (c *Client) Authenticate(ctx context.Context) error {
	ctx = normalizeContext(ctx)
	c.rwMutex.Lock()
	defer c.rwMutex.Unlock()
	if c.closed {
		return goubus.ErrClosed
	}

	return c.authenticateLocked(ctx)
}

// Close destroys the current rpcd session and clears stored credentials.
func (c *Client) Close() error {
	c.rwMutex.Lock()
	defer c.rwMutex.Unlock()

	if c.closed {
		return nil
	}
	c.closed = true
	sessionID := c.sessionData.UbusRPCSession
	c.sessionData = sessionMetadata{}
	c.password = ""
	if sessionID == "" {
		return nil
	}

	_, _, err := c.rawCallWithMode(context.Background(), sessionID, "session", "destroy", nil)

	return err
}

// String formats the client without exposing its password or session token.
func (c *Client) String() string {
	c.rwMutex.RLock()
	defer c.rwMutex.RUnlock()

	return fmt.Sprintf("rpc.Client{host:%q, username:%q, password:%q, session:%q}",
		c.host, c.username, "<redacted>", "<redacted>")
}

// GoString formats the client for Go-syntax formatting without exposing secrets.
func (c *Client) GoString() string {
	return c.String()
}

// LogValue returns structured slog attributes with credentials redacted.
func (c *Client) LogValue() slog.Value {
	c.rwMutex.RLock()
	defer c.rwMutex.RUnlock()

	return slog.GroupValue(
		slog.String("host", c.host),
		slog.String("username", c.username),
		slog.String("password", "<redacted>"),
		slog.Group("sessionData",
			slog.String("ubus_rpc_session", "<redacted>"),
			slog.Int("timeout", c.sessionData.Timeout),
			slog.Time("expire_time", c.sessionData.ExpireTime),
		),
	)
}

func (c *Client) isClosed() bool {
	c.rwMutex.RLock()
	defer c.rwMutex.RUnlock()

	return c.closed
}

func normalizeContext(ctx context.Context) context.Context {
	return cmp.Or(ctx, context.Background())
}
