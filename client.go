package goubus

import (
	"context"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net"
	"os"
	"sync"
	"time"

	"github.com/honeybbq/goubus/v2/internal/blobmsg"
	"github.com/honeybbq/goubus/v2/internal/errs"
)

// DefaultSocketPath is the default Unix domain socket path for local ubusd connections.
const DefaultSocketPath = "/tmp/run/ubus/ubus.sock"

const (
	fallbackVarSocketPath = "/var/run/ubus/ubus.sock"

	defaultDialTimeout  = 3 * time.Second
	defaultReadTimeout  = 60 * time.Second // Long-running ubus calls block on the stream before replying.
	defaultWriteTimeout = 3 * time.Second

	ubusSystemObjectEvent uint32 = 1
)

var (
	_ Transport = (*Client)(nil)
	_ Lister    = (*Client)(nil)
)

// Client implements a multiplexed, full-duplex native ubus Unix domain socket client.
// It supports concurrent method calls, bus introspection, event listeners,
// object subscriptions, and local ubus object registration over one connection.
type Client struct {
	conn         net.Conn
	connCtx      context.Context
	terminalErr  error
	connCancel   context.CancelFunc
	readDone     chan struct{}
	logger       *slog.Logger
	objectCache  map[string]uint32
	pending      map[uint16]*pendingRequest
	localObjects map[uint32]*localObject
	dialTimeout  time.Duration
	readTimeout  time.Duration
	writeTimeout time.Duration
	objectMu     sync.RWMutex
	localMu      sync.RWMutex
	pendingMu    sync.Mutex
	writeMu      sync.Mutex
	closeMu      sync.Mutex
	peerID       uint32
	nextSeq      uint16
	closed       bool
}

// NewClient connects to ubusd over a local Unix domain socket and performs the HELLO handshake.
// If sockPath is empty, it uses DefaultSocketPath (or /var/run/ubus/ubus.sock if present).
func NewClient(ctx context.Context, sockPath string, opts ...Option) (*Client, error) {
	ctx = normalizeContext(ctx)
	sockPath = resolveSocketPath(sockPath)

	err := blobmsg.ValidateSocketPath(sockPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrConnectionFailed, err)
	}

	connCtx, connCancel := context.WithCancel(context.WithoutCancel(ctx))
	client := newBaseClient(connCtx, connCancel, opts)
	dialer := net.Dialer{Timeout: max(client.dialTimeout, 0)}

	conn, err := dialer.DialContext(ctx, "unix", sockPath)
	if err != nil {
		connCancel()

		return nil, errs.Wrapf(ErrConnectionFailed, "dial unix socket: %v", err)
	}
	client.conn = conn

	err = client.exchangeHello(ctx)
	if err != nil {
		_ = client.closeWithError(ErrClosed)

		return nil, err
	}

	go client.readLoop(connCtx)

	return client, nil
}

// NewClientFromConn creates a native ubus client over an existing stream connection
// (such as an SSH-forwarded Unix socket) and performs the HELLO handshake.
// The given connection is owned by the new client, including closing it if handshake fails.
func NewClientFromConn(ctx context.Context, conn net.Conn, opts ...Option) (*Client, error) {
	ctx = normalizeContext(ctx)
	if conn == nil {
		return nil, errs.Wrapf(ErrInvalidParameter, "connection is required")
	}

	connCtx, connCancel := context.WithCancel(context.WithoutCancel(ctx))
	client := newBaseClient(connCtx, connCancel, opts)
	client.conn = conn

	err := client.exchangeHello(ctx)
	if err != nil {
		_ = client.closeWithError(ErrClosed)

		return nil, err
	}

	go client.readLoop(connCtx)

	return client, nil
}

func newBaseClient(connCtx context.Context, connCancel context.CancelFunc, opts []Option) *Client {
	client := &Client{
		connCtx:      connCtx,
		connCancel:   connCancel,
		readDone:     make(chan struct{}),
		nextSeq:      1,
		dialTimeout:  defaultDialTimeout,
		readTimeout:  defaultReadTimeout,
		writeTimeout: defaultWriteTimeout,
		objectCache:  make(map[string]uint32),
		pending:      make(map[uint16]*pendingRequest),
		localObjects: make(map[uint32]*localObject),
		logger:       discardLogger,
	}
	for _, opt := range opts {
		if opt != nil {
			opt.apply(client)
		}
	}

	return client
}

// Ping sends a native UBUS_MSG_PING frame to ubusd to verify socket liveness.
func (c *Client) Ping(ctx context.Context) error {
	ctx = normalizeContext(ctx)
	_, status, err := c.sendRequest(ctx, "ubus ping", blobmsg.UbusMsgPing, 0, []byte{0, 0, 0, 4})
	if err != nil {
		return err
	}
	if status != StatusOK {
		return &CallError{Err: StatusError(int(status)), Method: "ping", Code: int(status)}
	}

	return nil
}

// PeerID returns the peer identifier received in the HELLO handshake.
func (c *Client) PeerID() uint32 {
	return c.peerID
}

// Done returns a channel that closes when the client closes or loses its socket connection.
func (c *Client) Done() <-chan struct{} {
	return c.connCtx.Done()
}

// Err returns nil while the client is open, ErrClosed after Close, or the underlying disconnect error.
func (c *Client) Err() error {
	c.closeMu.Lock()
	defer c.closeMu.Unlock()

	return c.terminalErr
}

// Close closes the underlying Unix socket connection and stops all active subscriptions and objects.
func (c *Client) Close() error {
	return c.closeWithError(ErrClosed)
}

func (c *Client) isClosed() bool {
	c.closeMu.Lock()
	defer c.closeMu.Unlock()

	return c.closed
}

func (c *Client) closedReason() error {
	c.closeMu.Lock()
	defer c.closeMu.Unlock()

	return c.terminalErr
}

func (c *Client) closeWithError(cause error) error {
	c.closeMu.Lock()
	if c.closed {
		c.closeMu.Unlock()

		return nil
	}
	c.closed = true
	c.terminalErr = cause
	conn := c.conn
	c.closeMu.Unlock()

	c.connCancel()
	c.shutdownLocalObjects()

	return conn.Close()
}

func (c *Client) exchangeHello(ctx context.Context) (retErr error) {
	ctxErr := wrapCtxErr(ctx, "hello handshake")
	if ctxErr != nil {
		return ctxErr
	}

	cancelDone := make(chan struct{})
	stopCancel := context.AfterFunc(ctx, func() {
		_ = c.closeWithError(ErrClosed)
		close(cancelDone)
	})
	defer func() {
		if !stopCancel() {
			<-cancelDone
			retErr = wrapCtxErr(ctx, "hello handshake")

			return
		}
		if retErr != nil {
			_ = c.closeWithError(retErr)
			afterErr := wrapCtxErr(ctx, "hello handshake")
			if afterErr != nil {
				retErr = afterErr
			}
		}
	}()

	err := c.setReadTimeoutDeadline()
	if err != nil {
		return err
	}
	hdr, payload, err := blobmsg.ReadMessage(c.conn)
	if err != nil {
		return errs.Wrapf(ErrConnectionFailed, "read hello: %v", err)
	}
	err = c.clearReadTimeoutDeadline()
	if err != nil {
		return err
	}
	if hdr.Type != blobmsg.UbusMsgHello {
		return errs.Wrapf(ErrInvalidResponse, "expected HELLO, got %d", hdr.Type)
	}

	c.peerID = hdr.Peer
	if len(payload) != 0 {
		c.logger.Debug("HELLO payload", slog.String("payload", hex.EncodeToString(payload)))
	}

	return nil
}

func resolveSocketPath(sockPath string) string {
	if sockPath != "" {
		return sockPath
	}
	_, err := os.Stat(DefaultSocketPath)
	if err == nil {
		return DefaultSocketPath
	}
	_, err = os.Stat(fallbackVarSocketPath)
	if err == nil {
		return fallbackVarSocketPath
	}

	return DefaultSocketPath
}
