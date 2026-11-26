package goubus_test

import (
	"bytes"
	"context"
	"errors"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/honeybbq/goubus/v2"
	"github.com/honeybbq/goubus/v2/internal/blobmsg"
)

var errStopStream = errors.New("stop stream")

type protocolMockServer struct {
	conn         net.Conn
	removedCh    chan uint32
	onAddObject  func(seq uint16, objID uint32)
	onInvoke     func(seq uint16, objID uint32, method string)
	onSubscribe  func(seq uint16)
	onNotify     func(seq uint16)
	mu           sync.Mutex
	nextObjID    uint32
	monitorAdded bool
}

func TestNotify_ZeroAndMultipleSubscribersStayConnected(t *testing.T) {
	srv, client := newProtocolMockClient(t)
	var notifyCount int
	srv.onNotify = func(seq uint16) {
		notifyCount++
		subsPayload := []byte{}
		if notifyCount > 1 {
			subsPayload = blobmsg.EncodeUintAttributes([2]uint32{0, 11}, [2]uint32{0, 22})[blobmsg.BlobHeaderBytes:]
		}
		body, _ := blobmsg.CreateBlobMessage(map[uint32]any{
			blobmsg.UbusAttrObjID:       uint32(0x90000001),
			blobmsg.UbusAttrSubscribers: subsPayload,
			blobmsg.UbusAttrStatus:      uint32(0),
		}, nil)
		srv.send(blobmsg.UbusMsgStatus, seq, 1, body)
	}

	pub, err := client.AddObject(t.Context(), goubus.ObjectDef{
		Name: "notify.demo",
		Methods: map[string]goubus.Method{
			"noop": {Handler: func(context.Context, goubus.CallRequest) (any, error) { return map[string]any{}, nil }},
		},
	})
	if err != nil {
		t.Fatalf("AddObject: %v", err)
	}

	err = pub.Notify(t.Context(), "idle", map[string]any{"zero": true})
	if err != nil {
		t.Fatalf("Notify(0 subscribers): %v", err)
	}
	err = pub.Notify(t.Context(), "active", map[string]any{"count": 2})
	if err != nil {
		t.Fatalf("Notify(2 subscribers): %v", err)
	}
	err = client.Ping(t.Context())
	if err != nil {
		t.Fatalf("Ping after 0-subscriber Notify: %v", err)
	}
}

func TestOrphanObjectCleanupOnCancelledRegistration(t *testing.T) {
	srv, client := newProtocolMockClient(t)

	listenCtx, cancelListen := context.WithCancel(t.Context())
	srv.onInvoke = func(seq uint16, objID uint32, method string) {
		if objID == 1 && method == "register" {
			cancelListen()

			return
		}
		srv.sendOK(seq)
	}
	_, err := client.Listen(listenCtx, "*", func(goubus.Event) {})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Listen cancelled error = %v, want context.Canceled", err)
	}
	waitForRemovedObjID(t, srv.removedCh, "Listen rollback")

	subCtx, cancelSub := context.WithCancel(t.Context())
	srv.onSubscribe = func(uint16) {
		cancelSub()
	}
	_, err = client.Subscribe(subCtx, testSystemService, func(goubus.Notification) {})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Subscribe cancelled error = %v, want context.Canceled", err)
	}
	waitForRemovedObjID(t, srv.removedCh, "Subscribe rollback")

	assertAddObjectCancelRemovesOrphan(t, srv, client)
}

func assertAddObjectCancelRemovesOrphan(t *testing.T, srv *protocolMockServer, client *goubus.Client) {
	t.Helper()

	addCtx, cancelAdd := context.WithCancel(t.Context())
	srv.onAddObject = func(seq uint16, objID uint32) {
		dataBody := blobmsg.EncodeUintAttributes([2]uint32{blobmsg.UbusAttrObjID, objID})
		srv.send(blobmsg.UbusMsgData, seq, 1, dataBody)
		// On synchronous net.Pipe, writing a follow-up frame guarantees readLoop finished onData.
		srv.send(blobmsg.UbusMsgMonitor, 0, 0, []byte{0, 0, 0, 4})
		cancelAdd()
	}
	_, err := client.AddObject(addCtx, goubus.ObjectDef{
		Name: "cancel.mid.add",
		Methods: map[string]goubus.Method{
			"m": {Handler: func(context.Context, goubus.CallRequest) (any, error) { return map[string]any{}, nil }},
		},
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("AddObject cancelled mid-handshake = %v, want context.Canceled", err)
	}
	waitForRemovedObjID(t, srv.removedCh, "AddObject rollback")
}

func TestSendInvokeReply_EncodingFailureReturnsErrorStatus(t *testing.T) {
	broker, client := newMuxBrokerClient(t)

	pub, err := client.AddObject(t.Context(), goubus.ObjectDef{
		Name: "oversized.reply",
		Methods: map[string]goubus.Method{
			"huge": {
				Handler: func(context.Context, goubus.CallRequest) (any, error) {
					return map[string]any{"pad": strings.Repeat("x", blobmsg.MaxMessageBytes)}, nil
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("AddObject: %v", err)
	}

	_, code := broker.invokeClientObject(t, pub.ID(), "huge", nil)
	if code != goubus.StatusInvalidParameter {
		t.Errorf("oversized handler reply status = %d, want StatusInvalidParameter (%d)", code, goubus.StatusInvalidParameter)
	}

	hugeArg := map[string]any{"pad": strings.Repeat("y", blobmsg.MaxMessageBytes)}
	_, callErr := client.Call(t.Context(), testSystemService, testInfoMethod, hugeArg)
	if !errors.Is(callErr, goubus.ErrInvalidParameter) {
		t.Errorf("oversized outbound Call error = %v, want ErrInvalidParameter", callErr)
	}
	pingErr := client.Ping(t.Context())
	if pingErr != nil {
		t.Fatalf("Ping after rejected oversized Call: %v", pingErr)
	}
}

func TestClient_CallStreamMultiFrameNestedCallAndEarlyStop(t *testing.T) {
	srv, client := newProtocolMockClient(t)
	srv.onInvoke = func(seq uint16, _ uint32, method string) {
		if method == "multi" {
			for idx := range 3 {
				table, _ := blobmsg.CreateBlobmsgTable(map[string]any{"chunk": idx + 1})
				body, _ := blobmsg.CreateBlobMessage(map[uint32]any{
					blobmsg.UbusAttrData: table[blobmsg.BlobHeaderBytes:],
				}, nil)
				srv.send(blobmsg.UbusMsgData, seq, 1, body)
			}
		}
		srv.sendOK(seq)
	}

	var chunks []int
	err := client.CallStream(t.Context(), testSystemService, "multi", nil, func(res goubus.Result) error {
		var frame struct {
			Chunk int `json:"chunk"`
		}
		decodeErr := res.Unmarshal(&frame)
		if decodeErr != nil {
			return decodeErr
		}
		chunks = append(chunks, frame.Chunk)
		// Nested Ping inside onFrame verifies onFrame runs off readLoop with zero deadlock.
		return client.Ping(t.Context())
	})
	if err != nil {
		t.Fatalf("CallStream(multi): %v", err)
	}
	if len(chunks) != 3 || chunks[0] != 1 || chunks[1] != 2 || chunks[2] != 3 {
		t.Fatalf("CallStream chunks = %v, want [1 2 3]", chunks)
	}

	stopErr := client.CallStream(t.Context(), testSystemService, "multi", nil, func(goubus.Result) error {
		return errStopStream
	})
	if !errors.Is(stopErr, errStopStream) {
		t.Errorf("CallStream early stop error = %v, want errStopStream", stopErr)
	}
	nilHandlerErr := client.CallStream(t.Context(), testSystemService, "multi", nil, nil)
	if !errors.Is(nilHandlerErr, goubus.ErrInvalidParameter) {
		t.Errorf("CallStream(nil handler) = %v, want ErrInvalidParameter", nilHandlerErr)
	}

	assertCallStreamErrorAndStaleRefresh(t, srv, client)
}

func assertCallStreamErrorAndStaleRefresh(t *testing.T, srv *protocolMockServer, client *goubus.Client) {
	t.Helper()

	var attempts int
	srv.onInvoke = func(seq uint16, _ uint32, method string) {
		attempts++
		if method == "stale" && attempts == 1 {
			srv.send(blobmsg.UbusMsgStatus, seq, 1,
				blobmsg.EncodeUintAttributes([2]uint32{blobmsg.UbusAttrStatus, goubus.StatusNotFound}))

			return
		}
		if method == "denied" {
			srv.send(blobmsg.UbusMsgStatus, seq, 1,
				blobmsg.EncodeUintAttributes([2]uint32{blobmsg.UbusAttrStatus, goubus.StatusPermissionDenied}))

			return
		}
		srv.sendOK(seq)
	}

	// system is already cached from CallStream(multi); first stale call gets StatusNotFound and retries.
	err := client.CallStream(t.Context(), testSystemService, "stale", nil, func(goubus.Result) error { return nil })
	if err != nil || attempts != 2 {
		t.Fatalf("CallStream(stale) = (attempts=%d, err=%v), want 2, nil", attempts, err)
	}

	err = client.CallStream(t.Context(), testSystemService, "denied", nil, func(goubus.Result) error { return nil })
	if !errors.Is(err, goubus.ErrPermissionDenied) {
		t.Errorf("CallStream(denied) = %v, want ErrPermissionDenied", err)
	}

	cancelCtx, cancel := context.WithCancel(t.Context())
	cancel()
	err = client.CallStream(cancelCtx, testSystemService, "stale", nil, func(goubus.Result) error { return nil })
	if !errors.Is(err, context.Canceled) {
		t.Errorf("CallStream(cancelled ctx) = %v, want context.Canceled", err)
	}
}

func TestClient_MonitorCaptureReplaceAndClose(t *testing.T) {
	srv, client := newProtocolMockClient(t)
	received := make(chan goubus.MonitorMessage, 4)

	mon1, err := client.Monitor(t.Context(), func(msg goubus.MonitorMessage) {
		received <- msg
	})
	if err != nil {
		t.Fatalf("Monitor(1): %v", err)
	}

	innerAttrs, _ := blobmsg.CreateBlobMessage(map[uint32]any{
		blobmsg.UbusAttrMethod: "board",
	}, nil)
	monBody, _ := blobmsg.CreateBlobMessage(map[uint32]any{
		blobmsg.UbusMonitorClient: uint32(9),
		blobmsg.UbusMonitorPeer:   uint32(100),
		blobmsg.UbusMonitorSend:   []byte{0},
		blobmsg.UbusMonitorSeq:    uint32(12),
		blobmsg.UbusMonitorType:   uint32(blobmsg.UbusMsgInvoke),
		blobmsg.UbusMonitorData:   innerAttrs[blobmsg.BlobHeaderBytes:],
	}, nil)
	srv.send(blobmsg.UbusMsgMonitor, 1, 0, monBody)

	assertMonitorMessageReceived(t, received)

	// Calling Monitor a second time replaces the first monitor cleanly.
	mon2, err := client.Monitor(t.Context(), func(goubus.MonitorMessage) {})
	if err != nil {
		t.Fatalf("Monitor(2): %v", err)
	}
	<-mon1.Done()
	closeErr := mon2.Close()
	if closeErr != nil {
		t.Fatalf("mon2.Close(): %v", closeErr)
	}

	_, nilErr := client.Monitor(t.Context(), nil)
	if !errors.Is(nilErr, goubus.ErrInvalidParameter) {
		t.Errorf("Monitor(nil) = %v, want ErrInvalidParameter", nilErr)
	}

	// Malformed monitor frame is ignored without disconnecting the client.
	srv.send(blobmsg.UbusMsgMonitor, 2, 0, []byte{0, 0, 0, 8, 0, 0, 0, 1})
	pingErr := client.Ping(t.Context())
	if pingErr != nil {
		t.Fatalf("Ping after malformed monitor frame: %v", pingErr)
	}

	_ = client.Close()
	_, closedErr := client.Monitor(t.Context(), func(goubus.MonitorMessage) {})
	if !errors.Is(closedErr, goubus.ErrClosed) {
		t.Errorf("Monitor after Close = %v, want ErrClosed", closedErr)
	}
}

func assertMonitorMessageReceived(t *testing.T, received <-chan goubus.MonitorMessage) {
	t.Helper()

	select {
	case msg := <-received:
		if msg.Client != 9 || msg.Peer != 100 || msg.Seq != 12 || msg.Type != blobmsg.UbusMsgInvoke || msg.Send {
			t.Fatalf("MonitorMessage = %+v", msg)
		}
		var attrs struct {
			Method string `json:"method"`
		}
		decodeErr := msg.Unmarshal(&attrs)
		if decodeErr != nil || attrs.Method != "board" {
			t.Fatalf("MonitorMessage.Unmarshal = (%+v, %v), want method=board", attrs, decodeErr)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for MonitorMessage")
	}
}

func waitForRemovedObjID(t *testing.T, ch <-chan uint32, label string) {
	t.Helper()

	select {
	case id := <-ch:
		if id == 0 {
			t.Fatalf("%s sent zero objid in REMOVE_OBJECT", label)
		}
	case <-time.After(time.Second):
		t.Fatalf("%s did not send UBUS_MSG_REMOVE_OBJECT", label)
	}
}

func newProtocolMockClient(t *testing.T) (*protocolMockServer, *goubus.Client) {
	t.Helper()

	serverConn, clientConn := net.Pipe()
	srv := &protocolMockServer{
		conn:      serverConn,
		removedCh: make(chan uint32, 8),
		nextObjID: 0x90000001,
	}
	go srv.serve()

	client, err := goubus.NewClientFromConn(t.Context(), clientConn)
	if err != nil {
		_ = serverConn.Close()
		t.Fatalf("NewClientFromConn: %v", err)
	}
	t.Cleanup(func() {
		_ = client.Close()
		_ = serverConn.Close()
	})

	return srv, client
}

func (s *protocolMockServer) serve() {
	defer func() { _ = s.conn.Close() }()

	s.send(blobmsg.UbusMsgHello, 0, 1, []byte{0, 0, 0, 4})
	for {
		hdr, payload, err := blobmsg.ReadMessage(s.conn)
		if err != nil {
			return
		}
		attrs, _ := blobmsg.ParseTopLevelAttributes(payload)
		s.handle(hdr, attrs, payload)
	}
}

func (s *protocolMockServer) handle(hdr *blobmsg.UbusMessageHeader, attrs map[string]any, payload []byte) {
	switch hdr.Type {
	case blobmsg.UbusMsgLookup:
		s.mu.Lock()
		handleLookup(s.conn, hdr.Seq, payload)
		s.mu.Unlock()
	case blobmsg.UbusMsgAddObject:
		s.handleAddObject(hdr.Seq)
	case blobmsg.UbusMsgRemoveObject:
		if id, ok := blobmsg.ReadUint(attrs["objid"]); ok {
			s.removedCh <- id
		}
		s.sendOK(hdr.Seq)
	case blobmsg.UbusMsgSubscribe:
		if s.onSubscribe != nil {
			s.onSubscribe(hdr.Seq)

			return
		}
		s.sendOK(hdr.Seq)
	case blobmsg.UbusMsgNotify:
		if s.onNotify != nil {
			s.onNotify(hdr.Seq)

			return
		}
		s.sendOK(hdr.Seq)
	case blobmsg.UbusMsgInvoke:
		s.handleInvoke(hdr.Seq, attrs)
	case blobmsg.UbusMsgPing, blobmsg.UbusMsgUnsubscribe:
		s.sendOK(hdr.Seq)
	}
}

func (s *protocolMockServer) handleAddObject(seq uint16) {
	id := s.nextObjID
	s.nextObjID++
	if s.onAddObject != nil {
		s.onAddObject(seq, id)

		return
	}
	s.send(blobmsg.UbusMsgData, seq, 1, blobmsg.EncodeUintAttributes([2]uint32{blobmsg.UbusAttrObjID, id}))
	s.sendOK(seq)
}

func (s *protocolMockServer) handleInvoke(seq uint16, attrs map[string]any) {
	objID, _ := blobmsg.ReadUint(attrs["objid"])
	method, _ := attrs["method"].(string)
	if objID == 3 {
		s.monitorAdded = method == "add"
		s.sendOK(seq)

		return
	}
	if s.onInvoke != nil {
		s.onInvoke(seq, objID, method)

		return
	}
	s.sendOK(seq)
}

func (s *protocolMockServer) sendOK(seq uint16) {
	s.send(blobmsg.UbusMsgStatus, seq, 1, blobmsg.EncodeUintAttributes([2]uint32{blobmsg.UbusAttrStatus, 0}))
}

func (s *protocolMockServer) send(msgType uint8, seq uint16, peer uint32, body []byte) {
	var buf bytes.Buffer
	_ = blobmsg.EncodeHeader(&buf, &blobmsg.UbusMessageHeader{
		Type: msgType,
		Seq:  seq,
		Peer: peer,
	})
	_, _ = buf.Write(body)

	s.mu.Lock()
	_, _ = s.conn.Write(buf.Bytes())
	s.mu.Unlock()
}
