package transport

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"os"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/honeybbq/goubus/errdefs"
	"github.com/honeybbq/goubus/types"
)

const (
	defaultSocketPath   = "/tmp/run/ubus/ubus.sock"
	defaultDialTimeout  = 3 * time.Second
	defaultReadTimeout  = 60 * time.Second // iwinfo scan and similar operations may take longer
	defaultWriteTimeout = 3 * time.Second
)

// ubus message types.
const (
	ubusMsgHello        = 0
	ubusMsgStatus       = 1
	ubusMsgData         = 2
	ubusMsgPing         = 3
	ubusMsgLookup       = 4
	ubusMsgInvoke       = 5
	ubusMsgAddObject    = 6
	ubusMsgRemoveObject = 7
	ubusMsgSubscribe    = 8
	ubusMsgUnsubscribe  = 9
	ubusMsgNotify       = 10
	ubusMsgMonitor      = 11
)

// ubus attribute ids.
const (
	ubusAttrUnspec      = 0
	ubusAttrStatus      = 1
	ubusAttrObjPath     = 2
	ubusAttrObjID       = 3
	ubusAttrMethod      = 4
	ubusAttrObjType     = 5
	ubusAttrSignature   = 6
	ubusAttrData        = 7
	ubusAttrTarget      = 8
	ubusAttrActive      = 9
	ubusAttrNoReply     = 10
	ubusAttrSubscribers = 11
	ubusAttrUser        = 12
	ubusAttrGroup       = 13
)

// blobmsg type constants (aligned with libubox/blobmsg.h).
const (
	blobmsgTypeUnspec = 0
	blobmsgTypeArray  = 1
	blobmsgTypeTable  = 2
	blobmsgTypeString = 3
	blobmsgTypeInt64  = 4
	blobmsgTypeInt32  = 5
	blobmsgTypeInt16  = 6
	blobmsgTypeInt8   = 7
	blobmsgTypeDouble = 8
	blobmsgTypeBool   = blobmsgTypeInt8
)

const (
	blobAttrIDMask   = 0x7f000000
	blobAttrIDShift  = 24
	blobAttrLenMask  = 0x00ffffff
	blobAttrExtended = 0x80000000
	stringTerminator = byte(0)
	minBlobAttrLen   = 4
	headerBytes      = 8
	blobHeaderBytes  = 4
)

type ubusMessageHeader struct {
	Version uint8
	Type    uint8
	Seq     uint16
	Peer    uint32
}

// SocketClient implements direct ubus unix socket transport.
type SocketClient struct {
	conn   net.Conn
	seq    uint16
	peerID uint32

	sockPath     string
	dialTimeout  time.Duration
	readTimeout  time.Duration
	writeTimeout time.Duration

	mu     sync.Mutex
	closed bool

	objectMu    sync.RWMutex
	objectCache map[string]uint32

	debug bool
}

var _ types.Transport = (*SocketClient)(nil)

// NewSocketClient creates a new ubus socket client and performs the HELLO handshake.
func NewSocketClient(sockPath string) (*SocketClient, error) {
	if sockPath == "" {
		sockPath = defaultSocketPath
	}
	if err := validateSocketPath(sockPath); err != nil {
		return nil, errdefs.Wrapf(errdefs.ErrConnectionFailed, "%v", err)
	}
	conn, err := net.DialTimeout("unix", sockPath, defaultDialTimeout)
	if err != nil {
		return nil, errdefs.Wrapf(errdefs.ErrConnectionFailed, "dial unix socket: %v", err)
	}
	client := &SocketClient{
		conn:         conn,
		seq:          1,
		sockPath:     sockPath,
		dialTimeout:  defaultDialTimeout,
		readTimeout:  defaultReadTimeout,
		writeTimeout: defaultWriteTimeout,
		objectCache:  make(map[string]uint32),
	}
	if err := client.exchangeHello(); err != nil {
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
		return fmt.Sprintf("%x...", data[:maxLen])
	}
	return fmt.Sprintf("%x", data)
}

func previewJSON(v any, max int) string {
	if v == nil {
		return "<nil>"
	}
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("<error: %v>", err)
	}
	if len(b) > max {
		return string(b[:max]) + "..."
	}
	return string(b)
}

// SetDebug toggles verbose debug logging to stdout.
func (c *SocketClient) SetDebug(debug bool) *SocketClient {
	c.debug = debug
	return c
}

func (c *SocketClient) debugf(format string, args ...any) {
	if c.debug {
		fmt.Printf("[socket] "+format+"\n", args...)
	}
}

// Call invokes a ubus method through the socket transport.
func (c *SocketClient) Call(service, method string, data any) (types.Result, error) {
	if service == "" || method == "" {
		return nil, errdefs.Wrapf(errdefs.ErrInvalidParameter, "service and method required")
	}
	args, err := normalizeArgs(data)
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
	if err := c.sendMessage(ubusMsgInvoke, body); err != nil {
		return nil, err
	}
	var (
		resultData map[string]any
		statusCode uint32
		statusSeen bool
	)
	for !statusSeen {
		hdr, payload, err := c.receiveMessage()
		if err != nil {
			return nil, err
		}
		attrs, err := parseTopLevelAttributes(payload)
		if err != nil {
			return nil, errdefs.Wrapf(errdefs.ErrInvalidResponse, "parse invoke response: %v", err)
		}
		switch hdr.Type {
		case ubusMsgData:
			if c.debug {
				c.debugf("Parsed data attributes: %s", previewJSON(attrs, 256))
			}
			extracted := extractDataSection(attrs)
			if len(extracted) != 0 {
				if resultData == nil {
					resultData = make(map[string]any, len(extracted))
				}
				mergeMaps(resultData, extracted)
			}
		case ubusMsgStatus:
			statusSeen = true
			if val, ok := readUint(attrs["status"]); ok {
				statusCode = val
			}
		default:
			c.debugf("ignored message type=%d during invoke", hdr.Type)
		}
	}
	return &socketResult{
		status: statusCode,
		data:   resultData,
	}, nil
}

// Close terminates the underlying socket connection.
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

// getObjectID resolves the ubus object ID and caches it for future calls.
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
		if id, ok := readUint(obj["objid"]); ok {
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
		attrs[ubusAttrObjPath] = path
	}
	body, err := createBlobMessage(attrs, []uint32{ubusAttrObjPath})
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil, errdefs.ErrClosed
	}
	if err := c.sendMessage(ubusMsgLookup, body); err != nil {
		return nil, err
	}
	var (
		objects    []map[string]any
		statusCode uint32
		statusSeen bool
	)
	for !statusSeen {
		hdr, payload, err := c.receiveMessage()
		if err != nil {
			return nil, err
		}
		attrs, err := parseTopLevelAttributes(payload)
		if err != nil {
			return nil, errdefs.Wrapf(errdefs.ErrInvalidResponse, "parse lookup response: %v", err)
		}
		switch hdr.Type {
		case ubusMsgData:
			if len(attrs) != 0 {
				objects = append(objects, attrs)
			}
		case ubusMsgStatus:
			statusSeen = true
			if val, ok := readUint(attrs["status"]); ok {
				statusCode = val
			}
		default:
			c.debugf("ignored message type=%d during lookup", hdr.Type)
		}
	}
	if err := mapUbusCodeToError(int(statusCode)); err != nil {
		return nil, err
	}
	return objects, nil
}

func (c *SocketClient) exchangeHello() error {
	if err := c.conn.SetReadDeadline(time.Now().Add(c.readTimeout)); err != nil {
		return errdefs.Wrapf(errdefs.ErrConnectionFailed, "set read deadline: %v", err)
	}
	hdr, payload, err := readMessage(c.conn)
	if err != nil {
		return errdefs.Wrapf(errdefs.ErrConnectionFailed, "read hello: %v", err)
	}
	if hdr.Type != ubusMsgHello {
		return errdefs.Wrapf(errdefs.ErrInvalidResponse, "expected HELLO, got %d", hdr.Type)
	}
	c.peerID = hdr.Peer
	if len(payload) != 0 {
		c.debugf("HELLO payload: %x", payload)
	}
	return nil
}

func (c *SocketClient) sendMessage(msgType uint8, body []byte) error {
	var buf bytes.Buffer
	header := &ubusMessageHeader{
		Version: 0,
		Type:    msgType,
		Seq:     c.seq,
		Peer:    c.peerID,
	}
	c.seq++
	if err := binary.Write(&buf, binary.BigEndian, header); err != nil {
		return errdefs.Wrapf(errdefs.ErrInvalidParameter, "encode header: %v", err)
	}
	if len(body) > 0 {
		if _, err := buf.Write(body); err != nil {
			return errdefs.Wrapf(errdefs.ErrInvalidParameter, "write body: %v", err)
		}
	}
	if c.debug {
		c.debugf("Sending message: type=%d seq=%d body_len=%d body=%s",
			header.Type, header.Seq, len(body), hexPreview(body, 64))
	}
	if err := c.conn.SetWriteDeadline(time.Now().Add(c.writeTimeout)); err != nil {
		return errdefs.Wrapf(errdefs.ErrConnectionFailed, "set write deadline: %v", err)
	}
	_, err := c.conn.Write(buf.Bytes())
	if err != nil {
		return errdefs.Wrapf(errdefs.ErrConnectionFailed, "write message: %v", err)
	}
	return nil
}

func (c *SocketClient) receiveMessage() (*ubusMessageHeader, []byte, error) {
	if err := c.conn.SetReadDeadline(time.Now().Add(c.readTimeout)); err != nil {
		return nil, nil, errdefs.Wrapf(errdefs.ErrConnectionFailed, "set read deadline: %v", err)
	}
	hdr, payload, err := readMessage(c.conn)
	if err != nil {
		return nil, nil, err
	}
	if c.debug {
		c.debugf("Received message: type=%d seq=%d payload_len=%d payload=%s",
			hdr.Type, hdr.Seq, len(payload), hexPreview(payload, 64))
	}
	return hdr, payload, nil
}

func readMessage(r io.Reader) (*ubusMessageHeader, []byte, error) {
	headerBytesBuf := make([]byte, headerBytes)
	if _, err := io.ReadFull(r, headerBytesBuf); err != nil {
		return nil, nil, errdefs.Wrapf(errdefs.ErrConnectionFailed, "read header: %v", err)
	}
	hdr := &ubusMessageHeader{}
	if err := binary.Read(bytes.NewReader(headerBytesBuf), binary.BigEndian, hdr); err != nil {
		return nil, nil, errdefs.Wrapf(errdefs.ErrInvalidResponse, "decode header: %v", err)
	}
	blobHeader := make([]byte, blobHeaderBytes)
	if _, err := io.ReadFull(r, blobHeader); err != nil {
		return nil, nil, errdefs.Wrapf(errdefs.ErrConnectionFailed, "read blob header: %v", err)
	}
	blobLen := binary.BigEndian.Uint32(blobHeader)
	payload := make([]byte, blobHeaderBytes)
	copy(payload, blobHeader)
	if blobLen > blobHeaderBytes {
		remaining := int(blobLen) - blobHeaderBytes
		body := make([]byte, remaining)
		if _, err := io.ReadFull(r, body); err != nil {
			return nil, nil, errdefs.Wrapf(errdefs.ErrConnectionFailed, "read blob body: %v", err)
		}
		payload = append(payload, body...)
	}
	return hdr, payload, nil
}

type socketResult struct {
	status uint32
	data   map[string]any
}

func (r *socketResult) Unmarshal(target any) error {
	if err := mapUbusCodeToError(int(r.status)); err != nil {
		return err
	}
	if len(r.data) == 0 {
		return errdefs.ErrNoData
	}
	raw, err := json.Marshal(r.data)
	if err != nil {
		return errdefs.Wrapf(errdefs.ErrInvalidResponse, "marshal result: %v", err)
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return errdefs.Wrapf(errdefs.ErrInvalidResponse, "unmarshal result: %v", err)
	}
	return nil
}

func validateSocketPath(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSocket == 0 {
		return fmt.Errorf("path '%s' is not a unix socket", path)
	}
	return nil
}

func normalizeArgs(data any) (map[string]any, error) {
	if data == nil {
		return map[string]any{}, nil
	}
	switch v := data.(type) {
	case map[string]any:
		return cloneMap(v), nil
	case string:
		return decodeJSONMap([]byte(v))
	case []byte:
		return decodeJSONMap(v)
	default:
		raw, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}
		return decodeJSONMap(raw)
	}
}

func decodeJSONMap(raw []byte) (map[string]any, error) {
	if len(raw) == 0 {
		return map[string]any{}, nil
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var out map[string]any
	if err := dec.Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}

func cloneMap(src map[string]any) map[string]any {
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func mergeMaps(dst, src map[string]any) {
	for k, v := range src {
		dst[k] = v
	}
}

func readUint(value any) (uint32, bool) {
	switch v := value.(type) {
	case uint8:
		return uint32(v), true
	case uint16:
		return uint32(v), true
	case uint32:
		return v, true
	case uint64:
		return uint32(v), true
	case int:
		if v < 0 {
			return 0, false
		}
		return uint32(v), true
	case int32:
		if v < 0 {
			return 0, false
		}
		return uint32(v), true
	case int64:
		if v < 0 {
			return 0, false
		}
		return uint32(v), true
	case json.Number:
		if i, err := v.Int64(); err == nil && i >= 0 {
			return uint32(i), true
		}
	}
	return 0, false
}

func extractDataSection(attrs map[string]any) map[string]any {
	val, ok := attrs["data"]
	if !ok {
		return attrs
	}
	if m, ok := val.(map[string]any); ok {
		return m
	}
	return map[string]any{"value": val}
}

func createBlobMessage(attrs map[uint32]any, ordered []uint32) ([]byte, error) {
	keys := make([]uint32, 0, len(attrs))
	if len(ordered) != 0 {
		for _, k := range ordered {
			if _, ok := attrs[k]; ok {
				keys = append(keys, k)
			}
		}
	}
	for k := range attrs {
		if len(ordered) == 0 || !containsUint32(keys, k) {
			keys = append(keys, k)
		}
	}
	var items [][]byte
	totalLen := uint32(blobHeaderBytes)
	for _, key := range keys {
		value := attrs[key]
		var item []byte
		var err error
		if key == ubusAttrData {
			if data, ok := value.([]byte); ok && len(data) > 0 {
				attrLen := uint32(minBlobAttrLen + len(data))
				idLen := (key << blobAttrIDShift) | (attrLen & blobAttrLenMask)
				var buf bytes.Buffer
				if err := binary.Write(&buf, binary.BigEndian, idLen); err != nil {
					return nil, err
				}
				if _, err := buf.Write(data); err != nil {
					return nil, err
				}
				item = alignBuffer(buf.Bytes())
			} else {
				item, err = encodeUbusAttribute(key, value)
			}
		} else {
			item, err = encodeUbusAttribute(key, value)
		}
		if err != nil {
			return nil, err
		}
		items = append(items, item)
		totalLen += uint32(len(item))
	}
	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.BigEndian, totalLen); err != nil {
		return nil, err
	}
	for _, item := range items {
		if _, err := buf.Write(item); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

func encodeUbusAttribute(attrID uint32, value any) ([]byte, error) {
	var attrValue []byte
	switch v := value.(type) {
	case string:
		attrValue = encodeStringValue(v)
	case []byte:
		if attrID == ubusAttrData && len(v) > 0 {
			attrLen := uint32(minBlobAttrLen + len(v))
			idLen := (attrID << blobAttrIDShift) | (attrLen & blobAttrLenMask)
			var buf bytes.Buffer
			if err := binary.Write(&buf, binary.BigEndian, idLen); err != nil {
				return nil, err
			}
			if _, err := buf.Write(v); err != nil {
				return nil, err
			}
			return alignBuffer(buf.Bytes()), nil
		}
		attrValue = padToAlign(v)
	case uint32:
		attrValue = encodeUint32(v)
	case uint16:
		attrValue = encodeUint32(uint32(v))
	case uint8:
		attrValue = encodeUint32(uint32(v))
	case int:
		attrValue = encodeUint32(uint32(v))
	default:
		return nil, fmt.Errorf("unsupported attribute value type %T", value)
	}
	attrLen := uint32(minBlobAttrLen + len(attrValue))
	idLen := (attrID << blobAttrIDShift) | (attrLen & blobAttrLenMask)
	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.BigEndian, idLen); err != nil {
		return nil, err
	}
	if _, err := buf.Write(attrValue); err != nil {
		return nil, err
	}
	padded := alignBuffer(buf.Bytes())
	return padded, nil
}

func encodeStringValue(value string) []byte {
	data := append([]byte(value), stringTerminator)
	return padToAlign(data)
}

func encodeUint32(value uint32) []byte {
	data := make([]byte, 4)
	binary.BigEndian.PutUint32(data, value)
	return data
}

func padToAlign(data []byte) []byte {
	paddedLen := align4(len(data))
	if paddedLen == len(data) {
		return data
	}
	padding := make([]byte, paddedLen-len(data))
	return append(data, padding...)
}

func alignBuffer(data []byte) []byte {
	paddedLen := align4(len(data))
	if paddedLen == len(data) {
		return data
	}
	padding := make([]byte, paddedLen-len(data))
	return append(data, padding...)
}

func align4(n int) int {
	return (n + 3) &^ 3
}

func containsUint32(list []uint32, target uint32) bool {
	for _, v := range list {
		if v == target {
			return true
		}
	}
	return false
}

func (c *SocketClient) createInvokeBody(objID uint32, method string, args map[string]any) ([]byte, error) {
	argData, err := c.createBlobmsgData(args)
	if err != nil {
		return nil, err
	}
	if c.debug {
		c.debugf("Invoke args: %s", previewJSON(args, 512))
		c.debugf("Encoded blobmsg data: %s", hexPreview(argData, 128))
	}
	attrs := map[uint32]any{
		ubusAttrObjID:  objID,
		ubusAttrMethod: method,
	}
	if argData != nil {
		attrs[ubusAttrData] = argData
	}
	return createBlobMessage(attrs, []uint32{ubusAttrObjID, ubusAttrMethod, ubusAttrData})
}

func (c *SocketClient) createBlobmsgData(args map[string]any) ([]byte, error) {
	if len(args) == 0 {
		return []byte{}, nil
	}
	body, err := c.createBlobmsgTable(args)
	if err != nil {
		return nil, err
	}
	if len(body) <= blobHeaderBytes {
		return []byte{}, nil
	}
	return body[blobHeaderBytes:], nil
}

func (c *SocketClient) createBlobmsgTable(values map[string]any) ([]byte, error) {
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var entries [][]byte
	totalLen := uint32(blobHeaderBytes)
	for _, key := range keys {
		item, err := c.createBlobmsgEntry(key, values[key])
		if err != nil {
			return nil, err
		}
		entries = append(entries, item)
		totalLen += uint32(len(item))
	}
	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.BigEndian, totalLen); err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if _, err := buf.Write(entry); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

func (c *SocketClient) createBlobmsgArray(values []any) ([]byte, error) {
	var entries [][]byte
	totalLen := uint32(blobHeaderBytes)
	for _, value := range values {
		item, err := c.createBlobmsgEntry("", value)
		if err != nil {
			return nil, err
		}
		entries = append(entries, item)
		totalLen += uint32(len(item))
	}
	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.BigEndian, totalLen); err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if _, err := buf.Write(entry); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

func (c *SocketClient) createBlobmsgEntry(name string, value any) ([]byte, error) {
	blobType, valueData, err := c.encodeBlobmsgValue(value)
	if err != nil {
		return nil, err
	}
	nameLen := len(name)
	nameHeaderLen := align4(2 + nameLen + 1)
	attrLen := uint32(minBlobAttrLen + nameHeaderLen + len(valueData))
	idLen := (uint32(blobType) << blobAttrIDShift) | (attrLen & blobAttrLenMask) | blobAttrExtended
	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.BigEndian, idLen); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.BigEndian, uint16(nameLen)); err != nil {
		return nil, err
	}
	if _, err := buf.WriteString(name); err != nil {
		return nil, err
	}
	if err := buf.WriteByte(stringTerminator); err != nil {
		return nil, err
	}
	for buf.Len()%4 != 0 {
		if err := buf.WriteByte(0); err != nil {
			return nil, err
		}
	}
	if _, err := buf.Write(valueData); err != nil {
		return nil, err
	}
	return alignBuffer(buf.Bytes()), nil
}

func (c *SocketClient) encodeBlobmsgValue(value any) (uint8, []byte, error) {
	switch v := value.(type) {
	case nil:
		return blobmsgTypeUnspec, []byte{}, nil
	case bool:
		data := []byte{0}
		if v {
			data[0] = 1
		}
		return blobmsgTypeBool, data, nil
	case string:
		data := append([]byte(v), stringTerminator)
		return blobmsgTypeString, padToAlign(data), nil
	case json.Number:
		if i64, err := v.Int64(); err == nil {
			return c.encodeIntegerValue(i64)
		}
		if f64, err := v.Float64(); err == nil {
			return blobmsgTypeDouble, encodeFloat64(f64), nil
		}
		return 0, nil, fmt.Errorf("invalid number: %s", v.String())
	case int, int8, int16, int32, int64:
		return c.encodeIntegerValue(reflectInt64(v))
	case uint, uint8, uint16, uint32, uint64:
		return c.encodeUnsignedValue(reflectUint64(v))
	case float32:
		return blobmsgTypeDouble, encodeFloat64(float64(v)), nil
	case float64:
		return blobmsgTypeDouble, encodeFloat64(v), nil
	case []byte:
		data := append([]byte{}, v...)
		data = append(data, stringTerminator)
		return blobmsgTypeString, padToAlign(data), nil
	case map[string]any:
		table, err := c.createBlobmsgTable(v)
		if err != nil {
			return 0, nil, err
		}
		return blobmsgTypeTable, table[blobHeaderBytes:], nil
	case []any:
		array, err := c.createBlobmsgArray(v)
		if err != nil {
			return 0, nil, err
		}
		return blobmsgTypeArray, array[blobHeaderBytes:], nil
	default:
		return c.encodeReflectValue(value)
	}
}

func (c *SocketClient) encodeReflectValue(value any) (uint8, []byte, error) {
	rv := reflectValue(value)
	switch rv.Kind() {
	case 0:
		return blobmsgTypeUnspec, []byte{}, nil
	case reflect.Map:
		if rv.Type().Key().Kind() != reflect.String {
			return 0, nil, fmt.Errorf("map key must be string, got %s", rv.Type().Key())
		}
		iter := rv.MapRange()
		table := make(map[string]any, rv.Len())
		for iter.Next() {
			table[iter.Key().String()] = iter.Value().Interface()
		}
		return c.encodeBlobmsgValue(table)
	case reflect.Slice, reflect.Array:
		if rv.Type().Elem().Kind() == reflect.Uint8 {
			if rv.Len() == 0 {
				return blobmsgTypeUnspec, []byte{}, nil
			}
			data := make([]byte, rv.Len())
			copy(data, rv.Bytes())
			return blobmsgTypeString, padToAlign(append(data, stringTerminator)), nil
		}
		length := rv.Len()
		items := make([]any, 0, length)
		for i := range length {
			items = append(items, rv.Index(i).Interface())
		}
		return c.encodeBlobmsgValue(items)
	case reflect.Struct:
		fields := map[string]any{}
		rt := rv.Type()
		for i := 0; i < rv.NumField(); i++ {
			field := rt.Field(i)
			if !field.IsExported() {
				continue
			}
			name := field.Name
			if tag := field.Tag.Get("json"); tag != "" {
				name = parseJSONTag(tag)
				if name == "" {
					continue
				}
			}
			fields[name] = rv.Field(i).Interface()
		}
		return c.encodeBlobmsgValue(fields)
	default:
		return 0, nil, fmt.Errorf("unsupported value type %T", value)
	}
}

func (c *SocketClient) encodeIntegerValue(value int64) (uint8, []byte, error) {
	switch {
	case value >= math.MinInt32 && value <= math.MaxInt32:
		data := make([]byte, 4)
		binary.BigEndian.PutUint32(data, uint32(value))
		return blobmsgTypeInt32, data, nil
	default:
		data := make([]byte, 8)
		binary.BigEndian.PutUint64(data, uint64(value))
		return blobmsgTypeInt64, data, nil
	}
}

func (c *SocketClient) encodeUnsignedValue(value uint64) (uint8, []byte, error) {
	if value <= math.MaxUint32 {
		data := make([]byte, 4)
		binary.BigEndian.PutUint32(data, uint32(value))
		return blobmsgTypeInt32, data, nil
	}
	data := make([]byte, 8)
	binary.BigEndian.PutUint64(data, value)
	return blobmsgTypeInt64, data, nil
}

func encodeFloat64(value float64) []byte {
	data := make([]byte, 8)
	binary.BigEndian.PutUint64(data, math.Float64bits(value))
	return data
}

func parseTopLevelAttributes(data []byte) (map[string]any, error) {
	if len(data) < blobHeaderBytes {
		return map[string]any{}, nil
	}
	totalLen := binary.BigEndian.Uint32(data[:blobHeaderBytes])
	if totalLen == 0 || int(totalLen) > len(data) {
		return nil, errors.New("invalid blob length")
	}
	reader := blobReader{data: data[blobHeaderBytes:int(totalLen)]}
	result := map[string]any{}
	for reader.hasNext() {
		header, payload, err := reader.next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		value, err := parseAttribute(header, payload)
		if err != nil {
			return nil, err
		}
		name := getAttrName(header.id)
		result[name] = value
	}
	return result, nil
}

type attrHeader struct {
	id         uint32
	attrType   uint32
	length     int
	isExtended bool
}

type blobReader struct {
	data   []byte
	offset int
}

func (r *blobReader) hasNext() bool {
	return r.offset < len(r.data)
}

func (r *blobReader) next() (*attrHeader, []byte, error) {
	if r.offset+minBlobAttrLen > len(r.data) {
		return nil, nil, io.EOF
	}
	raw := binary.BigEndian.Uint32(r.data[r.offset : r.offset+4])
	attrLen := int(raw & blobAttrLenMask)
	if attrLen == 0 {
		r.offset = len(r.data)
		return nil, nil, io.EOF
	}
	if attrLen < minBlobAttrLen || r.offset+attrLen > len(r.data) {
		return nil, nil, fmt.Errorf("invalid attribute length %d", attrLen)
	}
	header := &attrHeader{
		id:         (raw & blobAttrIDMask) >> blobAttrIDShift,
		attrType:   (raw & blobAttrIDMask) >> blobAttrIDShift,
		length:     attrLen,
		isExtended: raw&blobAttrExtended != 0,
	}
	start := r.offset + minBlobAttrLen
	end := r.offset + attrLen
	payload := r.data[start:end]
	r.offset += align4(attrLen)
	return header, payload, nil
}

func parseAttribute(header *attrHeader, payload []byte) (any, error) {
	switch header.id {
	case ubusAttrStatus, ubusAttrObjID, ubusAttrObjType, ubusAttrSubscribers:
		return decodeUint(payload)
	case ubusAttrObjPath, ubusAttrMethod, ubusAttrTarget, ubusAttrUser, ubusAttrGroup:
		return decodeString(payload), nil
	case ubusAttrData, ubusAttrSignature:
		return parseBlobmsgContainer(payload, blobmsgTypeTable)
	default:
		if header.isExtended {
			_, value, err := parseBlobmsgEntry(header.attrType, payload)
			return value, err
		}
		return payload, nil
	}
}

func decodeUint(payload []byte) (uint32, error) {
	if len(payload) < 4 {
		return 0, fmt.Errorf("payload too short for uint32: %d", len(payload))
	}
	return binary.BigEndian.Uint32(payload[:4]), nil
}

func decodeString(payload []byte) string {
	if len(payload) == 0 {
		return ""
	}
	// Skip possible leading zero bytes (64-bit alignment padding).
	// But don't skip too many since empty strings also start with 0.
	// Only skip complete 4-byte zero blocks.
	for len(payload) >= 8 && bytes.Equal(payload[:4], []byte{0, 0, 0, 0}) {
		payload = payload[4:]
	}
	n := bytes.IndexByte(payload, stringTerminator)
	if n == -1 {
		return string(payload)
	}
	return string(payload[:n])
}

func parseBlobmsgContainer(payload []byte, expectedType uint8) (any, error) {
	if len(payload) == 0 {
		if expectedType == blobmsgTypeArray {
			return []any{}, nil
		}
		return map[string]any{}, nil
	}
	// Skip possible leading zero bytes (64-bit alignment padding).
	for len(payload) >= 4 && binary.BigEndian.Uint32(payload[:4]) == 0 {
		payload = payload[4:]
	}
	if len(payload) == 0 {
		if expectedType == blobmsgTypeArray {
			return []any{}, nil
		}
		return map[string]any{}, nil
	}
	reader := blobReader{data: payload}
	switch expectedType {
	case blobmsgTypeArray:
		var items []any
		for reader.hasNext() {
			header, data, err := reader.next()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				return nil, err
			}
			if !header.isExtended {
				return nil, errors.New("array entry not extended")
			}
			_, value, err := parseBlobmsgEntry(header.attrType, data)
			if err != nil {
				return nil, err
			}
			items = append(items, value)
		}
		return items, nil
	default:
		result := map[string]any{}
		for reader.hasNext() {
			header, data, err := reader.next()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				return nil, err
			}
			if !header.isExtended {
				return nil, errors.New("table entry not extended")
			}
			name, value, err := parseBlobmsgEntry(header.attrType, data)
			if err != nil {
				return nil, err
			}
			result[name] = value
		}
		return result, nil
	}
}

func parseBlobmsgEntry(blobType uint32, payload []byte) (string, any, error) {
	if len(payload) < 2 {
		return "", nil, errors.New("blobmsg payload too short")
	}
	nameLen := int(binary.BigEndian.Uint16(payload[:2]))
	headerLen := align4(2 + nameLen)
	if len(payload) < headerLen {
		return "", nil, errors.New("invalid blobmsg header length")
	}
	nameBytes := payload[2 : 2+nameLen]
	name := strings.TrimRight(string(nameBytes), "\x00")
	valueData := payload[headerLen:]
	value, err := parseBlobmsgValue(blobType, valueData)
	if err != nil {
		return "", nil, err
	}
	return name, value, nil
}

func parseBlobmsgValue(blobType uint32, data []byte) (any, error) {
	switch blobType {
	case blobmsgTypeUnspec:
		return nil, nil
	case blobmsgTypeArray:
		return parseBlobmsgContainer(data, blobmsgTypeArray)
	case blobmsgTypeTable:
		return parseBlobmsgContainer(data, blobmsgTypeTable)
	case blobmsgTypeString:
		return decodeString(data), nil
	case blobmsgTypeInt64:
		if len(data) < 8 {
			return int64(0), nil
		}
		// Some platforms may send aligned data (e.g. 12 bytes), value is in the last 8 bytes.
		if len(data) > 8 {
			offset := len(data) - 8
			return int64(binary.BigEndian.Uint64(data[offset:])), nil
		}
		return int64(binary.BigEndian.Uint64(data[:8])), nil
	case blobmsgTypeInt32:
		if len(data) < 4 {
			return int64(0), nil
		}
		// Return int64 to avoid overflow for large values (traffic stats may exceed int32 range).
		// Some platforms (e.g. ARM64) may send 8-byte aligned data, value is in the last 4 bytes.
		if len(data) >= 8 {
			return int64(binary.BigEndian.Uint32(data[4:8])), nil
		}
		return int64(binary.BigEndian.Uint32(data[:4])), nil
	case blobmsgTypeInt16:
		if len(data) < 2 {
			return int64(0), nil
		}
		// Some platforms may send aligned data, value is in the last 2 bytes.
		if len(data) >= 8 {
			return int64(binary.BigEndian.Uint16(data[6:8])), nil
		}
		return int64(binary.BigEndian.Uint16(data[len(data)-2:])), nil
	case blobmsgTypeInt8:
		if len(data) == 0 {
			return int64(0), nil
		}
		// Some platforms may send aligned data, value is in the last byte.
		return int64(data[len(data)-1]), nil
	case blobmsgTypeDouble:
		if len(data) < 8 {
			return float64(0), nil
		}
		return math.Float64frombits(binary.BigEndian.Uint64(data[:8])), nil
	default:
		return data, nil
	}
}

func getAttrName(attrID uint32) string {
	switch attrID {
	case ubusAttrStatus:
		return "status"
	case ubusAttrObjPath:
		return "objpath"
	case ubusAttrObjID:
		return "objid"
	case ubusAttrMethod:
		return "method"
	case ubusAttrObjType:
		return "objtype"
	case ubusAttrSignature:
		return "signature"
	case ubusAttrData:
		return "data"
	case ubusAttrTarget:
		return "target"
	case ubusAttrActive:
		return "active"
	case ubusAttrNoReply:
		return "no_reply"
	case ubusAttrSubscribers:
		return "subscribers"
	case ubusAttrUser:
		return "user"
	case ubusAttrGroup:
		return "group"
	default:
		return fmt.Sprintf("attr_%d", attrID)
	}
}

func reflectValue(value any) reflect.Value {
	rv := reflect.ValueOf(value)
	for rv.Kind() == reflect.Pointer || rv.Kind() == reflect.Interface {
		if rv.IsNil() {
			return reflect.Value{}
		}
		rv = rv.Elem()
	}
	return rv
}

func reflectInt64(value any) int64 {
	switch v := value.(type) {
	case int:
		return int64(v)
	case int8:
		return int64(v)
	case int16:
		return int64(v)
	case int32:
		return int64(v)
	case int64:
		return v
	default:
		return 0
	}
}

func reflectUint64(value any) uint64 {
	switch v := value.(type) {
	case uint:
		return uint64(v)
	case uint8:
		return uint64(v)
	case uint16:
		return uint64(v)
	case uint32:
		return uint64(v)
	case uint64:
		return v
	default:
		return 0
	}
}

func parseJSONTag(tag string) string {
	if tag == "-" {
		return ""
	}
	if idx := strings.Index(tag, ","); idx != -1 {
		tag = tag[:idx]
	}
	return tag
}
