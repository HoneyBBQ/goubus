package goubus

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"net"
	"sync"
	"time"

	"github.com/honeybbq/goubus/v2/internal/blobmsg"
	"github.com/honeybbq/goubus/v2/internal/errs"
	"github.com/honeybbq/goubus/v2/internal/logging"
)

const (
	logJSONLimit    = 256
	logHexLimit     = 64
	logLongHexLimit = 128
)

const (
	defaultSocketPath   = "/tmp/run/ubus/ubus.sock"
	defaultDialTimeout  = 3 * time.Second
	defaultReadTimeout  = 60 * time.Second // iwinfo scan and similar operations may take longer
	defaultWriteTimeout = 3 * time.Second
)

func deadlineFromTimeout(timeout time.Duration) time.Time {
	if timeout <= 0 {
		return time.Time{}
	}

	return time.Now().Add(timeout)
}

func normalizeContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}

	return ctx
}

func wrapCtxErr(ctx context.Context, prefix string) error {
	ctxErr := ctx.Err()
	if ctxErr == nil {
		return nil
	}

	cause := context.Cause(ctx)
	if cause != nil && !errors.Is(cause, ctxErr) {
		return fmt.Errorf("%s: %w: %w", prefix, ctxErr, cause)
	}

	return errs.Wrapf(cause, "%s", prefix)
}

// SocketClient implements direct ubus unix socket transport.
// It communicates directly with the ubusd daemon on the local system.
type SocketClient struct {
	conn         net.Conn
	objectCache  map[string]uint32
	logger       *slog.Logger
	objectMu     sync.RWMutex
	mu           sync.Mutex
	closeMu      sync.Mutex
	dialTimeout  time.Duration
	readTimeout  time.Duration
	writeTimeout time.Duration
	peerID       uint32
	seq          uint16
	closed       bool
}

var _ Transport = (*SocketClient)(nil)
var _ Lister = (*SocketClient)(nil)

type socketOptionFunc func(*SocketClient)

func (f socketOptionFunc) apply(client *SocketClient) {
	f(client)
}

// SocketOption configures a SocketClient.
type SocketOption interface {
	apply(client *SocketClient)
}

// WithSocketLogger sets the logger for the socket client.
func WithSocketLogger(logger *slog.Logger) SocketOption {
	return socketOptionFunc(func(client *SocketClient) {
		if logger == nil {
			client.logger = logging.Discard()

			return
		}
		client.logger = logger
	})
}

// WithDialTimeout sets the timeout for connecting to the socket.
// A zero or negative timeout disables this timeout; context cancellation and deadlines still apply.
func WithDialTimeout(timeout time.Duration) SocketOption {
	return socketOptionFunc(func(client *SocketClient) {
		client.dialTimeout = timeout
	})
}

// WithReadTimeout sets the timeout for reading from the socket.
// A zero or negative timeout disables this timeout; context cancellation and deadlines still apply.
func WithReadTimeout(timeout time.Duration) SocketOption {
	return socketOptionFunc(func(client *SocketClient) {
		client.readTimeout = timeout
	})
}

// WithWriteTimeout sets the timeout for writing to the socket.
// A zero or negative timeout disables this timeout; context cancellation and deadlines still apply.
func WithWriteTimeout(timeout time.Duration) SocketOption {
	return socketOptionFunc(func(client *SocketClient) {
		client.writeTimeout = timeout
	})
}

// NewSocketClient creates a new ubus socket client and performs the HELLO handshake.
// If sockPath is empty, it uses the default path (/tmp/run/ubus/ubus.sock).
func NewSocketClient(ctx context.Context, sockPath string, opts ...SocketOption) (*SocketClient, error) {
	ctx = normalizeContext(ctx)

	if sockPath == "" {
		sockPath = defaultSocketPath
	}

	err := validateSocketPath(sockPath)
	if err != nil {
		return nil, errs.Wrapf(errs.ErrConnectionFailed, "%v", err)
	}

	client := &SocketClient{
		seq:          1,
		dialTimeout:  defaultDialTimeout,
		readTimeout:  defaultReadTimeout,
		writeTimeout: defaultWriteTimeout,
		objectCache:  make(map[string]uint32),
		logger:       logging.Discard(),
	}

	for _, opt := range opts {
		if opt != nil {
			opt.apply(client)
		}
	}

	dialer := net.Dialer{Timeout: max(client.dialTimeout, 0)}

	conn, err := dialer.DialContext(ctx, "unix", sockPath)
	if err != nil {
		return nil, errs.Wrapf(errs.ErrConnectionFailed, "dial unix socket: %v", err)
	}

	client.conn = conn

	err = client.exchangeHello(ctx)
	if err != nil {
		_ = client.closeConn()

		return nil, err
	}

	return client, nil
}

// NewSocketClientFromConn creates a new ubus socket client from an existing
// socket connection and performs the HELLO handshake.  The given connection
// will be managed by the new client, including being closed if the handshake
// fails.
func NewSocketClientFromConn(ctx context.Context, conn net.Conn, opts ...SocketOption) (*SocketClient, error) {
	ctx = normalizeContext(ctx)

	if conn == nil {
		return nil, errs.Wrapf(errs.ErrInvalidParameter, "connection is required")
	}

	client := &SocketClient{
		conn:         conn,
		seq:          1,
		dialTimeout:  defaultDialTimeout,
		readTimeout:  defaultReadTimeout,
		writeTimeout: defaultWriteTimeout,
		objectCache:  make(map[string]uint32),
		logger:       logging.Discard(),
	}

	for _, opt := range opts {
		if opt != nil {
			opt.apply(client)
		}
	}

	err := client.exchangeHello(ctx)
	if err != nil {
		_ = client.closeConn()

		return nil, err
	}

	return client, nil
}

func hexPreview(data []byte, maxLen int) string {
	if len(data) == 0 {
		return ""
	}

	if len(data) > maxLen {
		return hex.EncodeToString(data[:maxLen]) + "..."
	}

	return hex.EncodeToString(data)
}

func previewJSON(v any, maxLen int) string {
	if v == nil {
		return "<nil>"
	}

	bytes, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("<error: %v>", err)
	}

	if len(bytes) > maxLen {
		return string(bytes[:maxLen]) + "..."
	}

	return string(bytes)
}

// Call invokes a method through the socket transport.
func (c *SocketClient) Call(ctx context.Context, service, method string, data any) (Result, error) {
	ctx = normalizeContext(ctx)

	if service == "" || method == "" {
		return nil, errs.Wrapf(errs.ErrInvalidParameter, "service and method required")
	}

	args, err := blobmsg.NormalizeArgs(data)
	if err != nil {
		return nil, errs.Wrapf(errs.ErrInvalidParameter, "normalize arguments: %v", err)
	}

	if c.isClosed() {
		return nil, errs.ErrClosed
	}

	ctxErr := wrapCtxErr(ctx, "ubus call")
	if ctxErr != nil {
		return nil, ctxErr
	}

	objectID, err := c.getObjectID(ctx, service)
	if err != nil {
		for code, statusErr := range ubusErrorMap {
			if code != StatusOK && statusErr != nil && errors.Is(err, statusErr) {
				return nil, &CallError{Err: statusErr, Service: service, Method: method, Code: code}
			}
		}

		return nil, err
	}

	body, err := c.createInvokeBody(objectID, method, args)
	if err != nil {
		return nil, err
	}

	c.logger.Debug("Invoke",
		slog.String("service", service),
		slog.String("method", method),
		slog.String("args", previewJSON(args, logBodyLimit)),
		slog.String("body", hexPreview(body, logLongHexLimit)))

	var result *socketResult

	err = c.executeRoundTrip(ctx, "ubus call", blobmsg.UbusMsgInvoke, body, func() error {
		result, err = c.readCallResponse()

		return err
	})
	if err != nil {
		return nil, err
	}

	if result.status != StatusOK {
		return nil, &CallError{
			Err: StatusError(int(result.status)), Service: service, Method: method, Code: int(result.status),
		}
	}

	return result, nil
}

// DialTimeout returns the configured socket dial timeout.
func (c *SocketClient) DialTimeout() time.Duration {
	return c.dialTimeout
}

// ReadTimeout returns the configured socket read timeout.
func (c *SocketClient) ReadTimeout() time.Duration {
	return c.readTimeout
}

// WriteTimeout returns the configured socket write timeout.
func (c *SocketClient) WriteTimeout() time.Duration {
	return c.writeTimeout
}

// PeerID returns the peer identifier received in the HELLO handshake.
func (c *SocketClient) PeerID() uint32 {
	return c.peerID
}

// Close closes the underlying Unix socket connection.
func (c *SocketClient) Close() error {
	return c.closeConn()
}

func (c *SocketClient) isClosed() bool {
	c.closeMu.Lock()
	defer c.closeMu.Unlock()

	return c.closed
}

func (c *SocketClient) closeConn() error {
	c.closeMu.Lock()
	defer c.closeMu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true
	if c.conn != nil {
		return c.conn.Close()
	}

	return nil
}

func (c *SocketClient) setReadTimeoutDeadline() error {
	err := c.conn.SetReadDeadline(deadlineFromTimeout(c.readTimeout))
	if err != nil {
		return errs.Wrapf(errs.ErrConnectionFailed, "set read deadline: %v", err)
	}

	return nil
}

func (c *SocketClient) clearReadTimeoutDeadline() error {
	if c.readTimeout <= 0 {
		return nil
	}

	err := c.conn.SetReadDeadline(time.Time{})
	if err != nil {
		return errs.Wrapf(errs.ErrConnectionFailed, "clear read deadline: %v", err)
	}

	return nil
}

func (c *SocketClient) setWriteTimeoutDeadline() error {
	err := c.conn.SetWriteDeadline(deadlineFromTimeout(c.writeTimeout))
	if err != nil {
		return errs.Wrapf(errs.ErrConnectionFailed, "set write deadline: %v", err)
	}

	return nil
}

func (c *SocketClient) clearWriteTimeoutDeadline() error {
	if c.writeTimeout <= 0 {
		return nil
	}

	err := c.conn.SetWriteDeadline(time.Time{})
	if err != nil {
		return errs.Wrapf(errs.ErrConnectionFailed, "clear write deadline: %v", err)
	}

	return nil
}

func (c *SocketClient) readCallResponse() (*socketResult, error) {
	err := c.setReadTimeoutDeadline()
	if err != nil {
		return nil, err
	}

	var (
		resultData map[string]any
		statusCode uint32
		statusSeen bool
	)

	for !statusSeen {
		hdr, payload, readErr := blobmsg.ReadMessage(c.conn)
		if readErr != nil {
			return nil, readErr
		}

		attrs, parseErr := blobmsg.ParseTopLevelAttributes(payload)
		if parseErr != nil {
			return nil, errs.Wrapf(errs.ErrInvalidResponse, "parse invoke response: %v", parseErr)
		}

		resultData, statusCode, statusSeen = c.processCallResponse(hdr, attrs, resultData, statusCode)
	}

	err = c.clearReadTimeoutDeadline()
	if err != nil {
		return nil, err
	}

	return &socketResult{data: resultData, status: statusCode}, nil
}

func (c *SocketClient) processCallResponse(
	hdr *blobmsg.UbusMessageHeader,
	attrs map[string]any,
	resultData map[string]any,
	statusCode uint32,
) (map[string]any, uint32, bool) {
	switch hdr.Type {
	case blobmsg.UbusMsgData:
		c.logger.Debug("Parsed data attributes", slog.String("data", previewJSON(attrs, logJSONLimit)))

		extracted := blobmsg.ExtractDataSection(attrs)
		if len(extracted) != 0 {
			if resultData == nil {
				resultData = make(map[string]any, len(extracted))
			}

			maps.Copy(resultData, extracted)
		}
	case blobmsg.UbusMsgStatus:
		if val, ok := blobmsg.ReadUint(attrs["status"]); ok {
			statusCode = val
		}

		return resultData, statusCode, true
	default:
		c.logger.Debug("ignored message during invoke", slog.Int("type", int(hdr.Type)))
	}

	return resultData, statusCode, false
}

// getObjectID resolves and caches the ubus object ID.
func (c *SocketClient) getObjectID(ctx context.Context, path string) (uint32, error) {
	c.objectMu.RLock()

	if id, ok := c.objectCache[path]; ok {
		c.objectMu.RUnlock()

		return id, nil
	}

	c.objectMu.RUnlock()

	objects, err := c.listObjects(ctx, path)
	if err != nil {
		return 0, err
	}

	for _, obj := range objects {
		objPath, ok := obj["objpath"].(string)
		if !ok {
			continue
		}

		if id, ok := blobmsg.ReadUint(obj["objid"]); ok {
			c.objectMu.Lock()
			c.objectCache[objPath] = id
			c.objectMu.Unlock()

			if objPath == path {
				return id, nil
			}
		}
	}

	return 0, errs.Wrapf(errs.ErrNotFound, "object '%s' not found", path)
}

func (c *SocketClient) listObjects(ctx context.Context, path string) ([]map[string]any, error) {
	if c.isClosed() {
		return nil, errs.ErrClosed
	}

	ctxErr := wrapCtxErr(ctx, "ubus lookup")
	if ctxErr != nil {
		return nil, ctxErr
	}

	attrs := map[uint32]any{}
	if path != "" {
		attrs[blobmsg.UbusAttrObjPath] = path
	}

	body, err := blobmsg.CreateBlobMessage(attrs, []uint32{blobmsg.UbusAttrObjPath})
	if err != nil {
		return nil, err
	}

	var (
		objects    []map[string]any
		statusCode uint32
	)

	err = c.executeRoundTrip(ctx, "ubus lookup", blobmsg.UbusMsgLookup, body, func() error {
		objects, statusCode, err = c.readLookupResponse()

		return err
	})
	if err != nil {
		return nil, err
	}

	statusErr := StatusError(int(statusCode))
	if statusErr != nil {
		return nil, statusErr
	}

	return objects, nil
}

func (c *SocketClient) readLookupResponse() ([]map[string]any, uint32, error) {
	err := c.setReadTimeoutDeadline()
	if err != nil {
		return nil, 0, err
	}

	var (
		objects    []map[string]any
		statusCode uint32
		statusSeen bool
	)

	for !statusSeen {
		hdr, payload, readErr := blobmsg.ReadMessage(c.conn)
		if readErr != nil {
			return nil, 0, readErr
		}

		attrs, parseErr := blobmsg.ParseTopLevelAttributes(payload)
		if parseErr != nil {
			return nil, 0, errs.Wrapf(errs.ErrInvalidResponse, "parse lookup response: %v", parseErr)
		}

		objects, statusCode, statusSeen = c.processLookupResponse(hdr, attrs, objects, statusCode)
	}

	err = c.clearReadTimeoutDeadline()
	if err != nil {
		return nil, 0, err
	}

	return objects, statusCode, nil
}

func (c *SocketClient) processLookupResponse(
	hdr *blobmsg.UbusMessageHeader,
	attrs map[string]any,
	objects []map[string]any,
	statusCode uint32,
) ([]map[string]any, uint32, bool) {
	switch hdr.Type {
	case blobmsg.UbusMsgData:
		if len(attrs) != 0 {
			objects = append(objects, attrs)
		}
	case blobmsg.UbusMsgStatus:
		if val, ok := blobmsg.ReadUint(attrs["status"]); ok {
			statusCode = val
		}

		return objects, statusCode, true
	default:
		c.logger.Debug("ignored message during lookup", slog.Int("type", int(hdr.Type)))
	}

	return objects, statusCode, false
}

func (c *SocketClient) exchangeHello(ctx context.Context) (retErr error) {
	ctxErr := wrapCtxErr(ctx, "hello handshake")
	if ctxErr != nil {
		return ctxErr
	}

	cancelDone := make(chan struct{})

	stopCancel := context.AfterFunc(ctx, func() {
		_ = c.closeConn()

		close(cancelDone)
	})
	defer func() {
		if !stopCancel() {
			<-cancelDone

			retErr = wrapCtxErr(ctx, "hello handshake")

			return
		}

		if retErr != nil {
			_ = c.closeConn()

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
		return errs.Wrapf(errs.ErrConnectionFailed, "read hello: %v", err)
	}

	err = c.clearReadTimeoutDeadline()
	if err != nil {
		return err
	}

	if hdr.Type != blobmsg.UbusMsgHello {
		return errs.Wrapf(errs.ErrInvalidResponse, "expected HELLO, got %d", hdr.Type)
	}

	c.peerID = hdr.Peer
	if len(payload) != 0 {
		c.logger.Debug("HELLO payload", slog.String("payload", hex.EncodeToString(payload)))
	}

	return nil
}

func (c *SocketClient) executeRoundTrip(
	ctx context.Context,
	action string,
	msgType uint8,
	body []byte,
	readFn func() error,
) (retErr error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	ctxErr := wrapCtxErr(ctx, action)
	if ctxErr != nil {
		return ctxErr
	}

	if c.isClosed() {
		return errs.ErrClosed
	}

	frame, seq, err := c.encodeMessage(msgType, body)
	if err != nil {
		return err
	}

	cancelDone := make(chan struct{})

	stopCancel := context.AfterFunc(ctx, func() {
		_ = c.closeConn()

		close(cancelDone)
	})
	defer func() {
		if !stopCancel() {
			<-cancelDone

			retErr = wrapCtxErr(ctx, action)

			return
		}

		if retErr != nil {
			_ = c.closeConn()

			afterErr := wrapCtxErr(ctx, action)
			if afterErr != nil {
				retErr = afterErr
			}
		}
	}()

	err = c.writeFrame(frame, msgType, seq, body)
	if err != nil {
		return err
	}

	return readFn()
}

func (c *SocketClient) encodeMessage(msgType uint8, body []byte) ([]byte, uint16, error) {
	var buf bytes.Buffer

	seq := c.seq
	c.seq++

	header := &blobmsg.UbusMessageHeader{
		Version: 0,
		Type:    msgType,
		Seq:     seq,
		Peer:    c.peerID,
	}

	err := blobmsg.EncodeHeader(&buf, header)
	if err != nil {
		return nil, 0, errs.Wrapf(errs.ErrInvalidParameter, "encode header: %v", err)
	}

	if len(body) > 0 {
		_, _ = buf.Write(body)
	}

	return buf.Bytes(), seq, nil
}

func (c *SocketClient) writeFrame(frame []byte, msgType uint8, seq uint16, body []byte) error {
	c.logger.Debug("Sending message",
		slog.Int("type", int(msgType)),
		slog.Int("seq", int(seq)),
		slog.Int("body_len", len(body)),
		slog.String("body", hexPreview(body, logHexLimit)))

	err := c.setWriteTimeoutDeadline()
	if err != nil {
		return err
	}

	_, err = c.conn.Write(frame)
	if err != nil {
		return errs.Wrapf(errs.ErrConnectionFailed, "write message: %v", err)
	}

	return c.clearWriteTimeoutDeadline()
}

func (c *SocketClient) createInvokeBody(objID uint32, method string, args map[string]any) ([]byte, error) {
	argData, err := blobmsg.CreateBlobmsgData(args)
	if err != nil {
		return nil, err
	}

	c.logger.Debug("Create invoke body",
		slog.String("args", previewJSON(args, logBodyLimit)),
		slog.String("blobmsg_data", hexPreview(argData, logLongHexLimit)))

	attrs := map[uint32]any{
		blobmsg.UbusAttrObjID:  objID,
		blobmsg.UbusAttrMethod: method,
	}
	if argData != nil {
		attrs[blobmsg.UbusAttrData] = argData
	}

	return blobmsg.CreateBlobMessage(attrs, []uint32{blobmsg.UbusAttrObjID, blobmsg.UbusAttrMethod, blobmsg.UbusAttrData})
}

type socketResult struct {
	data   map[string]any
	status uint32
}

func (r *socketResult) Unmarshal(target any) error {
	if len(r.data) == 0 {
		return errs.ErrNoData
	}

	raw, err := json.Marshal(r.data)
	if err != nil {
		return errs.Wrapf(errs.ErrInvalidResponse, "marshal result: %v", err)
	}

	err = json.Unmarshal(raw, target)
	if err != nil {
		return errs.Wrapf(errs.ErrInvalidResponse, "unmarshal result: %v", err)
	}

	return nil
}

func validateSocketPath(path string) error {
	return blobmsg.ValidateSocketPath(path)
}
