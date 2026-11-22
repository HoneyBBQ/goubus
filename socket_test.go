package goubus_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/honeybbq/goubus/v2"
	"github.com/honeybbq/goubus/v2/errdefs"
	"github.com/honeybbq/goubus/v2/internal/blobmsg"
	"github.com/honeybbq/goubus/v2/internal/logging"
)

type signalingConn struct {
	net.Conn
	readStarted chan struct{}
	once        sync.Once
}

type countingConn struct {
	net.Conn
	closeCalls atomic.Int32
}

type socketMockHandler func(net.Conn, *blobmsg.UbusMessageHeader, []byte)

var errTestRequestBudgetCause = errors.New("request budget exhausted")

func (c *countingConn) Close() error {
	c.closeCalls.Add(1)

	return c.Conn.Close()
}

func newPipeSocketClient(
	t *testing.T,
	handler socketMockHandler,
	opts ...goubus.SocketOption,
) *goubus.SocketClient {
	t.Helper()

	serverConn, clientConn := net.Pipe()
	client := newSocketClient(t, clientConn, serverConn, handler, opts...)

	return client
}

func newSocketClient(
	t *testing.T,
	clientConn net.Conn,
	serverConn net.Conn,
	handler socketMockHandler,
	opts ...goubus.SocketOption,
) *goubus.SocketClient {
	t.Helper()

	go serveSocketMock(serverConn, handler)

	client, err := goubus.NewSocketClientFromConn(t.Context(), clientConn, opts...)
	if err != nil {
		_ = serverConn.Close()

		t.Fatalf("create socket client: %v", err)
	}

	t.Cleanup(func() {
		_ = client.Close()
		_ = serverConn.Close()
	})

	return client
}

func serveSocketMock(conn net.Conn, handler socketMockHandler) {
	defer func() {
		_ = conn.Close()
	}()

	var buf bytes.Buffer

	_ = blobmsg.EncodeHeader(&buf, &blobmsg.UbusMessageHeader{Type: blobmsg.UbusMsgHello, Peer: 1})
	_, _ = buf.Write([]byte{0, 0, 0, 4})
	_, _ = conn.Write(buf.Bytes())

	for {
		hdr, payload, err := blobmsg.ReadMessage(conn)
		if err != nil {
			return
		}

		handler(conn, hdr, payload)
	}
}

func systemMockHandler(conn net.Conn, hdr *blobmsg.UbusMessageHeader, payload []byte) {
	switch hdr.Type {
	case blobmsg.UbusMsgLookup:
		handleLookup(conn, hdr.Seq, payload)
	case blobmsg.UbusMsgInvoke:
		handleInvoke(conn, hdr.Seq, payload)
	}
}

func delayedSystemMockHandler(conn net.Conn, hdr *blobmsg.UbusMessageHeader, payload []byte) {
	time.Sleep(30 * time.Millisecond)
	systemMockHandler(conn, hdr, payload)
}

func blockInvokeMock(received chan struct{}) socketMockHandler {
	return func(conn net.Conn, hdr *blobmsg.UbusMessageHeader, payload []byte) {
		if hdr.Type == blobmsg.UbusMsgLookup {
			handleLookup(conn, hdr.Seq, payload)
		}

		if hdr.Type == blobmsg.UbusMsgInvoke {
			close(received)
		}
	}
}

func waitForSignal(t *testing.T, signal <-chan struct{}, failure string) {
	t.Helper()

	select {
	case <-signal:
	case <-time.After(time.Second):
		t.Fatal(failure)
	}
}

func waitForError(t *testing.T, done <-chan error, failure string) error {
	t.Helper()

	select {
	case err := <-done:
		return err
	case <-time.After(time.Second):
		t.Fatal(failure)

		return nil
	}
}

func (c *signalingConn) Read(data []byte) (int, error) {
	c.once.Do(func() {
		close(c.readStarted)
	})

	return c.Conn.Read(data)
}

func TestSocketClient_NewSocketClient(t *testing.T) {
	sockPath := filepath.Join(t.TempDir(), "ubus.sock")

	// Create a mock ubusd
	var lc net.ListenConfig

	listener, err := lc.Listen(context.Background(), "unix", sockPath)
	if err != nil {
		t.Skipf("unix sockets not supported: %v", err)
	}

	defer func() {
		_ = listener.Close()
	}()

	go func() {
		conn, errAccept := listener.Accept()
		if errAccept != nil {
			return
		}

		defer func() {
			_ = conn.Close()
		}()

		// Send HELLO
		header := &blobmsg.UbusMessageHeader{
			Type: blobmsg.UbusMsgHello,
			Peer: 0x12345678,
		}

		var buf bytes.Buffer

		_ = blobmsg.EncodeHeader(&buf, header)
		_, _ = conn.Write(buf.Bytes())
		_, _ = conn.Write([]byte{0, 0, 0, 4}) // Empty payload length 4
	}()

	ctx := context.Background()

	client, err := goubus.NewSocketClient(ctx, sockPath)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	defer func() {
		_ = client.Close()
	}()

	if client.PeerID() != 0x12345678 {
		t.Errorf("expected peer ID 0x12345678, got 0x%x", client.PeerID())
	}
}

func TestSocketClient_NewSocketClientFromConn(t *testing.T) {
	const peerID = 0x456789ab

	serverConn, clientConn := net.Pipe()
	defer func() {
		_ = serverConn.Close()
	}()

	go func() {
		// Send HELLO
		header := &blobmsg.UbusMessageHeader{
			Type: blobmsg.UbusMsgHello,
			Peer: peerID,
		}

		var buf bytes.Buffer

		_ = blobmsg.EncodeHeader(&buf, header)
		_, _ = serverConn.Write(buf.Bytes())
		_, _ = serverConn.Write([]byte{0, 0, 0, 4}) // Empty payload length 4
	}()

	client, err := goubus.NewSocketClientFromConn(t.Context(), clientConn)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	defer func() {
		_ = client.Close()
	}()

	if client.PeerID() != peerID {
		t.Errorf("expected peer ID 0x%x, got 0x%x", peerID, client.PeerID())
	}
}

func TestSocketClient_NewSocketClientFromConnRejectsNil(t *testing.T) {
	client, err := goubus.NewSocketClientFromConn(t.Context(), nil)
	if client != nil {
		t.Fatal("expected nil client")
	}

	if !errdefs.IsInvalidParameter(err) {
		t.Fatalf("expected invalid parameter error, got %v", err)
	}
}

func TestSocketClient_NewSocketClientFromConnCancellation(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer func() {
		_ = serverConn.Close()
	}()

	err := serverConn.SetReadDeadline(time.Now().Add(time.Second))
	if err != nil {
		t.Fatalf("set peer read deadline: %v", err)
	}

	readStarted := make(chan struct{})
	conn := &signalingConn{
		Conn:        clientConn,
		readStarted: readStarted,
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	errCh := make(chan error, 1)

	go func() {
		_, handshakeErr := goubus.NewSocketClientFromConn(ctx, conn)
		errCh <- handshakeErr
	}()

	select {
	case <-readStarted:
	case <-time.After(time.Second):
		t.Fatal("HELLO handshake did not start reading")
	}

	cancel()

	select {
	case handshakeErr := <-errCh:
		if !errors.Is(handshakeErr, context.Canceled) {
			t.Fatalf("expected context canceled error, got %v", handshakeErr)
		}
	case <-time.After(time.Second):
		t.Fatal("HELLO handshake did not stop after cancellation")
	}

	_, err = serverConn.Read(make([]byte, 1))
	if !errors.Is(err, io.EOF) {
		t.Fatalf("expected owned connection to be closed, got %v", err)
	}
}

func TestSocketClient_Call(t *testing.T) {
	sockPath := filepath.Join(t.TempDir(), "ubus.sock")

	var lc net.ListenConfig

	listener, err := lc.Listen(context.Background(), "unix", sockPath)
	if err != nil {
		t.Skipf("unix sockets not supported: %v", err)
	}

	defer func() {
		_ = listener.Close()
	}()

	go mockUbusd(t, listener)

	ctx := context.Background()

	client, err := goubus.NewSocketClient(ctx, sockPath)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		_ = client.Close()
	}()

	res, err := client.Call(ctx, testSystemService, testInfoMethod, nil)
	if err != nil {
		t.Fatal(err)
	}

	var info struct {
		Hostname string `json:"hostname"`
	}

	errUnmarshal := res.Unmarshal(&info)
	if errUnmarshal != nil {
		t.Fatal(errUnmarshal)
	}

	if info.Hostname != testHostname {
		t.Errorf("expected OpenWrt, got %s", info.Hostname)
	}

	// Test cache: call again, should not trigger another lookup
	// (We can check this by making mockUbusd fail on second lookup if we wanted)
	_, err = client.Call(ctx, testSystemService, testInfoMethod, nil)
	if err != nil {
		t.Fatal(err)
	}
}

func mockUbusd(t *testing.T, l net.Listener) {
	t.Helper()

	conn, errAccept := l.Accept()
	if errAccept != nil {
		return
	}

	defer func() {
		_ = conn.Close()
	}()

	// 1. Send HELLO
	helloHdr := &blobmsg.UbusMessageHeader{Type: blobmsg.UbusMsgHello, Peer: 1}

	var buf bytes.Buffer

	_ = blobmsg.EncodeHeader(&buf, helloHdr)
	_, _ = buf.Write([]byte{0, 0, 0, 4})
	_, _ = conn.Write(buf.Bytes())

	for {
		hdr, payload, errRead := blobmsg.ReadMessage(conn)
		if errRead != nil {
			return
		}

		switch hdr.Type {
		case blobmsg.UbusMsgLookup:
			handleLookup(conn, hdr.Seq, payload)
		case blobmsg.UbusMsgInvoke:
			handleInvoke(conn, hdr.Seq, payload)
		}
	}
}

func handleLookup(conn net.Conn, seq uint16, payload []byte) {
	// Handle lookup
	attrs, _ := blobmsg.ParseTopLevelAttributes(payload)

	path, ok := attrs["objpath"].(string)
	if !ok {
		return
	}

	if path == testSystemService {
		// Send Data
		dataAttrs := map[uint32]any{
			blobmsg.UbusAttrObjPath: testSystemService,
			blobmsg.UbusAttrObjID:   uint32(100),
		}
		dataBody, _ := blobmsg.CreateBlobMessage(dataAttrs, nil)
		sendMsg(conn, blobmsg.UbusMsgData, seq, dataBody)

		// Send Status
		statusAttrs := map[uint32]any{blobmsg.UbusAttrStatus: uint32(0)}
		statusBody, _ := blobmsg.CreateBlobMessage(statusAttrs, nil)
		sendMsg(conn, blobmsg.UbusMsgStatus, seq, statusBody)
	}
}

func handleInvoke(conn net.Conn, seq uint16, payload []byte) {
	// Handle invoke
	attrs, _ := blobmsg.ParseTopLevelAttributes(payload)
	objID, _ := blobmsg.ReadUint(attrs["objid"])

	method, ok := attrs["method"].(string)
	if !ok {
		return
	}

	if objID == 100 && method == testInfoMethod {
		// Send Data
		respData := map[string]any{"hostname": testHostname}
		dataPayload, _ := blobmsg.CreateBlobmsgTable(respData)
		// ParseBlobmsgContainer expects the payload WITHOUT the 4-byte length header
		dataBody, _ := blobmsg.CreateBlobMessage(map[uint32]any{
			blobmsg.UbusAttrData: dataPayload[4:],
		}, nil)
		sendMsg(conn, blobmsg.UbusMsgData, seq, dataBody)

		// Send Status
		statusBody, _ := blobmsg.CreateBlobMessage(map[uint32]any{
			blobmsg.UbusAttrStatus: uint32(0),
		}, nil)
		sendMsg(conn, blobmsg.UbusMsgStatus, seq, statusBody)
	}
}

func sendMsg(conn net.Conn, msgType uint8, seq uint16, body []byte) {
	const peer = 1

	hdr := &blobmsg.UbusMessageHeader{Type: msgType, Seq: seq, Peer: peer}

	var buf bytes.Buffer

	_ = blobmsg.EncodeHeader(&buf, hdr)
	_, _ = buf.Write(body)
	_, _ = conn.Write(buf.Bytes())
}

func TestSocketClient_Timeout(t *testing.T) {
	sockPath := filepath.Join(t.TempDir(), "ubus_timeout.sock")

	var lc net.ListenConfig

	listener, err := lc.Listen(context.Background(), "unix", sockPath)
	if err != nil {
		t.Skipf("unix sockets not supported: %v", err)
	}

	defer func() {
		_ = listener.Close()
	}()

	go func() {
		conn, _ := listener.Accept()
		if conn == nil {
			return
		}

		defer func() {
			_ = conn.Close()
		}()
		// Send HELLO
		helloHdr := &blobmsg.UbusMessageHeader{Type: blobmsg.UbusMsgHello, Peer: 1}

		var buf bytes.Buffer

		_ = blobmsg.EncodeHeader(&buf, helloHdr)
		_, _ = buf.Write([]byte{0, 0, 0, 4})
		_, _ = conn.Write(buf.Bytes())

		// Don't respond to anything else
		time.Sleep(200 * time.Millisecond)
	}()

	ctx := context.Background()

	client, err := goubus.NewSocketClient(ctx, sockPath, goubus.WithReadTimeout(100*time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		_ = client.Close()
	}()

	// This should timeout because mock ubusd won't respond to lookup
	_, err = client.Call(ctx, "any", "thing", nil)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

func TestSocketClient_Options(t *testing.T) {
	client := &goubus.SocketClient{}
	goubus.WithSocketLogger(logging.Discard())(client)
	goubus.WithDialTimeout(time.Second)(client)
	goubus.WithReadTimeout(time.Second)(client)
	goubus.WithWriteTimeout(time.Second)(client)

	if client.DialTimeout() != time.Second {
		t.Errorf("dialTimeout mismatch")
	}

	if client.ReadTimeout() != time.Second {
		t.Errorf("readTimeout mismatch")
	}

	if client.WriteTimeout() != time.Second {
		t.Errorf("writeTimeout mismatch")
	}
}

func TestSocketClient_ReadTimeoutResetsAfterIdle(t *testing.T) {
	client := newPipeSocketClient(t, systemMockHandler, goubus.WithReadTimeout(80*time.Millisecond))

	time.Sleep(150 * time.Millisecond)

	_, err := client.Call(t.Context(), testSystemService, testInfoMethod, nil)
	if err != nil {
		t.Fatalf("first call after idle: %v", err)
	}

	time.Sleep(150 * time.Millisecond)

	_, err = client.Call(t.Context(), testSystemService, testInfoMethod, nil)
	if err != nil {
		t.Fatalf("cached call after idle: %v", err)
	}
}

func TestSocketClient_NonPositiveTimeouts(t *testing.T) {
	sockPath := filepath.Join(t.TempDir(), "ubus_no_timeout.sock")

	listenConfig := net.ListenConfig{}

	listener, err := listenConfig.Listen(t.Context(), "unix", sockPath)
	if err != nil {
		t.Skipf("unix sockets not supported: %v", err)
	}

	defer func() {
		_ = listener.Close()
	}()

	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr == nil {
			serveSocketMock(conn, delayedSystemMockHandler)
		}
	}()

	client, err := goubus.NewSocketClient(t.Context(), sockPath,
		goubus.WithDialTimeout(-time.Second),
		goubus.WithReadTimeout(0),
		goubus.WithWriteTimeout(0))
	if err != nil {
		t.Fatalf("create client with disabled timeouts: %v", err)
	}
	defer func() {
		_ = client.Close()
	}()

	_, err = client.Call(t.Context(), testSystemService, testInfoMethod, nil)
	if err != nil {
		t.Fatalf("call with disabled timeouts: %v", err)
	}
}

func TestSocketClient_CallCanceledBeforeWriteKeepsConnection(t *testing.T) {
	client := newPipeSocketClient(t, systemMockHandler)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := client.Call(ctx, testSystemService, testInfoMethod, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected canceled error, got %v", err)
	}

	_, err = client.Call(context.Background(), testSystemService, testInfoMethod, nil)
	if err != nil {
		t.Fatalf("connection unusable after pre-canceled call: %v", err)
	}
}

func TestSocketClient_CallTimeoutCauseClosesConnection(t *testing.T) {
	received := make(chan struct{})
	client := newPipeSocketClient(t, blockInvokeMock(received), goubus.WithReadTimeout(0))

	ctx, cancel := context.WithTimeoutCause(t.Context(), 50*time.Millisecond, errTestRequestBudgetCause)

	defer cancel()

	callDone := make(chan error, 1)

	go func() {
		_, callErr := client.Call(ctx, testSystemService, testInfoMethod, nil)
		callDone <- callErr
	}()

	select {
	case <-received:
	case <-time.After(time.Second):
		t.Fatal("server did not receive invoke")
	}

	var callErr error
	select {
	case callErr = <-callDone:
	case <-time.After(time.Second):
		t.Fatal("call did not stop after context timeout")
	}

	if !errors.Is(callErr, context.DeadlineExceeded) || !errors.Is(callErr, errTestRequestBudgetCause) {
		t.Fatalf("expected deadline and custom cause, got %v", callErr)
	}

	_, err := client.Call(context.Background(), testSystemService, testInfoMethod, nil)
	if !errors.Is(err, errdefs.ErrClosed) {
		t.Fatalf("expected closed connection after timed-out round trip, got %v", err)
	}
}

func TestSocketClient_CallCanceledDuringResponseClosesConnection(t *testing.T) {
	received := make(chan struct{})
	client := newPipeSocketClient(t, blockInvokeMock(received), goubus.WithReadTimeout(0))

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	callDone := make(chan error, 1)

	go func() {
		_, callErr := client.Call(ctx, testSystemService, testInfoMethod, nil)
		callDone <- callErr
	}()

	waitForSignal(t, received, "server did not receive invoke")
	cancel()

	callErr := waitForError(t, callDone, "call did not stop after cancellation")
	if !errors.Is(callErr, context.Canceled) {
		t.Fatalf("expected canceled error, got %v", callErr)
	}
}

func TestSocketClient_CallNilContext(t *testing.T) {
	var nilCtx context.Context

	client := newPipeSocketClient(t, systemMockHandler)

	_, err := client.Call(nilCtx, testSystemService, testInfoMethod, nil)
	if err != nil {
		t.Fatalf("call with nil context: %v", err)
	}
}

func TestSocketClient_CloseUnblocksCallOnce(t *testing.T) {
	serverConn, pipeConn := net.Pipe()
	countedConn := &countingConn{Conn: pipeConn}
	received := make(chan struct{})
	client := newSocketClient(t, countedConn, serverConn, blockInvokeMock(received), goubus.WithReadTimeout(0))

	callDone := make(chan error, 1)

	go func() {
		_, callErr := client.Call(context.Background(), testSystemService, testInfoMethod, nil)
		callDone <- callErr
	}()

	waitForSignal(t, received, "server did not receive invoke")

	closeDone := make(chan error, 1)

	go func() {
		closeDone <- client.Close()
	}()

	closeErr := waitForError(t, closeDone, "Close did not return")
	if closeErr != nil {
		t.Fatalf("close client: %v", closeErr)
	}

	callErr := waitForError(t, callDone, "blocked call did not stop after Close")
	if callErr == nil {
		t.Fatal("expected blocked call to fail after Close")
	}

	closeErr = client.Close()
	if closeErr != nil {
		t.Fatalf("second close: %v", closeErr)
	}

	if countedConn.closeCalls.Load() != 1 {
		t.Fatalf("expected one underlying Close call, got %d", countedConn.closeCalls.Load())
	}
}
func TestSocketClient_LookupStatusErrorKeepsConnection(t *testing.T) {
	client := newPipeSocketClient(t, missingServiceMockHandler)

	_, err := client.Call(t.Context(), "missing.service", testInfoMethod, nil)
	if !errors.Is(err, errdefs.ErrNotFound) {
		t.Fatalf("expected not found error, got %v", err)
	}

	_, err = client.Call(t.Context(), testSystemService, testInfoMethod, nil)
	if err != nil {
		t.Fatalf("call after lookup status error: %v", err)
	}
}

func missingServiceMockHandler(conn net.Conn, hdr *blobmsg.UbusMessageHeader, payload []byte) {
	if hdr.Type == blobmsg.UbusMsgInvoke {
		handleInvoke(conn, hdr.Seq, payload)

		return
	}

	if hdr.Type != blobmsg.UbusMsgLookup {
		return
	}

	attrs, err := blobmsg.ParseTopLevelAttributes(payload)
	if err != nil {
		return
	}

	path, _ := attrs["objpath"].(string)
	if path != "missing.service" {
		handleLookup(conn, hdr.Seq, payload)

		return
	}

	statusBody, _ := blobmsg.CreateBlobMessage(map[uint32]any{blobmsg.UbusAttrStatus: uint32(4)}, nil)
	sendMsg(conn, blobmsg.UbusMsgStatus, hdr.Seq, statusBody)
}
