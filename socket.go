// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package goubus

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"net"
	"sync"
	"time"

	"github.com/honeybbq/goubus/v2/errdefs"
	"github.com/honeybbq/goubus/v2/internal/blobmsg"
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

// SocketClient implements direct ubus unix socket transport.
// It communicates directly with the ubusd daemon on the local system.
type SocketClient struct {
	conn         net.Conn
	logger       *slog.Logger
	objectCache  map[string]uint32
	sockPath     string
	dialTimeout  time.Duration
	readTimeout  time.Duration
	writeTimeout time.Duration
	objectMu     sync.RWMutex
	mu           sync.Mutex
	peerID       uint32
	seq          uint16
	closed       bool
}

var _ Transport = (*SocketClient)(nil)

// SocketOption defines a functional option for a SocketClient.
type SocketOption func(*SocketClient)

// WithSocketLogger sets the logger for the socket client.
func WithSocketLogger(logger *slog.Logger) SocketOption {
	return func(c *SocketClient) {
		c.SetLogger(logger)
	}
}

// WithDialTimeout sets the timeout for connecting to the socket.
func WithDialTimeout(timeout time.Duration) SocketOption {
	return func(c *SocketClient) {
		c.dialTimeout = timeout
	}
}

// WithReadTimeout sets the timeout for reading from the socket.
func WithReadTimeout(timeout time.Duration) SocketOption {
	return func(c *SocketClient) {
		c.readTimeout = timeout
	}
}

// WithWriteTimeout sets the timeout for writing to the socket.
func WithWriteTimeout(timeout time.Duration) SocketOption {
	return func(c *SocketClient) {
		c.writeTimeout = timeout
	}
}

// NewSocketClient creates a new ubus socket client and performs the HELLO handshake.
// If sockPath is empty, it uses the default path (/tmp/run/ubus/ubus.sock).
func NewSocketClient(ctx context.Context, sockPath string, opts ...SocketOption) (*SocketClient, error) {
	if sockPath == "" {
		sockPath = defaultSocketPath
	}

	err := validateSocketPath(sockPath)
	if err != nil {
		return nil, errdefs.Wrapf(errdefs.ErrConnectionFailed, "%v", err)
	}

	client := &SocketClient{
		sockPath:     sockPath,
		seq:          1,
		dialTimeout:  defaultDialTimeout,
		readTimeout:  defaultReadTimeout,
		writeTimeout: defaultWriteTimeout,
		objectCache:  make(map[string]uint32),
		logger:       logging.Discard(),
	}

	for _, opt := range opts {
		opt(client)
	}

	dialer := net.Dialer{Timeout: client.dialTimeout}

	conn, err := dialer.DialContext(ctx, "unix", client.sockPath)
	if err != nil {
		return nil, errdefs.Wrapf(errdefs.ErrConnectionFailed, "dial unix socket: %v", err)
	}

	client.conn = conn

	err = client.exchangeHello()
	if err != nil {
		_ = conn.Close()

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

func (c *SocketClient) SetLogger(logger *slog.Logger) {
	if logger == nil {
		c.logger = logging.Discard()
	} else {
		c.logger = logger
	}
}

// Call invokes a ubus method through the socket transport.
func (c *SocketClient) Call(ctx context.Context, service, method string, data any) (Result, error) {
	if service == "" || method == "" {
		return nil, errdefs.Wrapf(errdefs.ErrInvalidParameter, "service and method required")
	}

	args, err := blobmsg.NormalizeArgs(data)
	if err != nil {
		return nil, errdefs.Wrapf(errdefs.ErrInvalidParameter, "normalize arguments: %v", err)
	}

	objectID, err := c.getObjectID(service)
	if err != nil {
		return nil, err
	}

	body, err := c.createInvokeBody(objectID, method, args)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil, errdefs.ErrClosed
	}

	err = c.sendMessage(blobmsg.UbusMsgInvoke, body)
	if err != nil {
		return nil, err
	}

	const logBodyLimit = logJSONLimit * 2

	c.logger.Debug("Invoke",
		slog.String("service", service),
		slog.String("method", method),
		slog.String("args", previewJSON(args, logBodyLimit)),
		slog.String("body", hexPreview(body, logLongHexLimit)))

	return c.handleCallResponse()
}

func (c *SocketClient) DialTimeout() time.Duration {
	return c.dialTimeout
}

func (c *SocketClient) ReadTimeout() time.Duration {
	return c.readTimeout
}

func (c *SocketClient) WriteTimeout() time.Duration {
	return c.writeTimeout
}

func (c *SocketClient) PeerID() uint32 {
	return c.peerID
}

func (c *SocketClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true
	if c.conn != nil {
		return c.conn.Close()
	}

	return nil
}

func (c *SocketClient) handleCallResponse() (Result, error) {
	var (
		resultData map[string]any
		statusCode uint32
		statusSeen bool
	)

	for !statusSeen {
		hdr, payload, err := blobmsg.ReadMessage(c.conn)
		if err != nil {
			return nil, err
		}

		attrs, err := blobmsg.ParseTopLevelAttributes(payload)
		if err != nil {
			return nil, errdefs.Wrapf(errdefs.ErrInvalidResponse, "parse invoke response: %v", err)
		}

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
			statusSeen = true

			if val, ok := blobmsg.ReadUint(attrs["status"]); ok {
				statusCode = val
			}
		default:
			c.logger.Debug("ignored message during invoke", slog.Int("type", int(hdr.Type)))
		}
	}

	return &socketResult{
		data:   resultData,
		status: statusCode,
	}, nil
}

// getObjectID resolves and caches the ubus object ID.
func (c *SocketClient) getObjectID(path string) (uint32, error) {
	c.objectMu.RLock()

	if id, ok := c.objectCache[path]; ok {
		c.objectMu.RUnlock()

		return id, nil
	}

	c.objectMu.RUnlock()

	objects, err := c.listObjects(path)
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

	return 0, errdefs.Wrapf(errdefs.ErrNotFound, "object '%s' not found", path)
}

func (c *SocketClient) listObjects(path string) ([]map[string]any, error) {
	attrs := map[uint32]any{}
	if path != "" {
		attrs[blobmsg.UbusAttrObjPath] = path
	}

	body, err := blobmsg.CreateBlobMessage(attrs, []uint32{blobmsg.UbusAttrObjPath})
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil, errdefs.ErrClosed
	}

	err = c.sendMessage(blobmsg.UbusMsgLookup, body)
	if err != nil {
		return nil, err
	}

	return c.handleLookupResponse()
}

func (c *SocketClient) handleLookupResponse() ([]map[string]any, error) {
	var (
		objects    []map[string]any
		statusCode uint32
		statusSeen bool
	)

	for !statusSeen {
		hdr, payload, err := blobmsg.ReadMessage(c.conn)
		if err != nil {
			return nil, err
		}

		attrs, err := blobmsg.ParseTopLevelAttributes(payload)
		if err != nil {
			return nil, errdefs.Wrapf(errdefs.ErrInvalidResponse, "parse lookup response: %v", err)
		}

		switch hdr.Type {
		case blobmsg.UbusMsgData:
			if len(attrs) != 0 {
				objects = append(objects, attrs)
			}
		case blobmsg.UbusMsgStatus:
			statusSeen = true

			if val, ok := blobmsg.ReadUint(attrs["status"]); ok {
				statusCode = val
			}
		default:
			c.logger.Debug("ignored message during lookup", slog.Int("type", int(hdr.Type)))
		}
	}

	err := MapUbusCodeToError(int(statusCode))
	if err != nil {
		return nil, err
	}

	return objects, nil
}

func (c *SocketClient) exchangeHello() error {
	err := c.conn.SetReadDeadline(time.Now().Add(c.readTimeout))
	if err != nil {
		return errdefs.Wrapf(errdefs.ErrConnectionFailed, "set read deadline: %v", err)
	}

	hdr, payload, err := blobmsg.ReadMessage(c.conn)
	if err != nil {
		return errdefs.Wrapf(errdefs.ErrConnectionFailed, "read hello: %v", err)
	}

	if hdr.Type != blobmsg.UbusMsgHello {
		return errdefs.Wrapf(errdefs.ErrInvalidResponse, "expected HELLO, got %d", hdr.Type)
	}

	c.peerID = hdr.Peer
	if len(payload) != 0 {
		c.logger.Debug("HELLO payload", slog.String("payload", hex.EncodeToString(payload)))
	}

	return nil
}

func (c *SocketClient) sendMessage(msgType uint8, body []byte) error {
	var buf bytes.Buffer

	header := &blobmsg.UbusMessageHeader{
		Version: 0,
		Type:    msgType,
		Seq:     c.seq,
		Peer:    c.peerID,
	}
	c.seq++

	err := blobmsg.EncodeHeader(&buf, header)
	if err != nil {
		return errdefs.Wrapf(errdefs.ErrInvalidParameter, "encode header: %v", err)
	}

	if len(body) > 0 {
		_, err = buf.Write(body)
		if err != nil {
			return errdefs.Wrapf(errdefs.ErrInvalidParameter, "write body: %v", err)
		}
	}

	c.logger.Debug("Sending message",
		slog.Int("type", int(header.Type)),
		slog.Int("seq", int(header.Seq)),
		slog.Int("body_len", len(body)),
		slog.String("body", hexPreview(body, logHexLimit)))

	err = c.conn.SetWriteDeadline(time.Now().Add(c.writeTimeout))
	if err != nil {
		return errdefs.Wrapf(errdefs.ErrConnectionFailed, "set write deadline: %v", err)
	}

	_, err = c.conn.Write(buf.Bytes())
	if err != nil {
		return errdefs.Wrapf(errdefs.ErrConnectionFailed, "write message: %v", err)
	}

	return nil
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
	err := MapUbusCodeToError(int(r.status))
	if err != nil {
		return err
	}

	if len(r.data) == 0 {
		return errdefs.ErrNoData
	}

	raw, err := json.Marshal(r.data)
	if err != nil {
		return errdefs.Wrapf(errdefs.ErrInvalidResponse, "marshal result: %v", err)
	}

	err = json.Unmarshal(raw, target)
	if err != nil {
		return errdefs.Wrapf(errdefs.ErrInvalidResponse, "unmarshal result: %v", err)
	}

	return nil
}

func validateSocketPath(path string) error {
	return blobmsg.ValidateSocketPath(path)
}
