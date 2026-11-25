package goubus_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/honeybbq/goubus/v2"
	"github.com/honeybbq/goubus/v2/internal/blobmsg"
)

var errTestInternalFailure = errors.New("unexpected internal failure")

func TestClient_DialAndValidationErrors(t *testing.T) {
	_, err := goubus.NewClientFromConn(t.Context(), nil)
	if !errors.Is(err, goubus.ErrInvalidParameter) {
		t.Fatalf("NewClientFromConn(nil) = %v, want ErrInvalidParameter", err)
	}

	_, err = goubus.NewClient(t.Context(), "")
	if !errors.Is(err, goubus.ErrConnectionFailed) {
		t.Fatalf("NewClient(default missing) = %v, want ErrConnectionFailed", err)
	}

	regFile := filepath.Join(t.TempDir(), "not-a-socket")
	_ = os.WriteFile(regFile, []byte("x"), 0o600)
	_, err = goubus.NewClient(t.Context(), regFile)
	if !errors.Is(err, goubus.ErrNotUnixSocket) {
		t.Fatalf("NewClient(regular file) = %v, want ErrNotUnixSocket", err)
	}

	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	_, client := newMuxBrokerClientWithOpts(t, goubus.WithLogger(logger))

	_, err = client.Call(t.Context(), "", "info", nil)
	if !errors.Is(err, goubus.ErrInvalidParameter) {
		t.Fatalf("Call(empty service) = %v, want ErrInvalidParameter", err)
	}
	_, err = client.Call(t.Context(), testSystemService, testInfoMethod, make(chan int))
	if !errors.Is(err, goubus.ErrInvalidParameter) {
		t.Fatalf("Call(chan arg) = %v, want ErrInvalidParameter", err)
	}

	bigArg := map[string]any{"pad": strings.Repeat("a", 600)}
	res, err := client.Call(t.Context(), testSystemService, testInfoMethod, bigArg)
	if err != nil {
		t.Fatalf("Call(bigArg): %v", err)
	}
	var badTarget struct {
		Hostname int `json:"hostname"`
	}
	if !errors.Is(res.Unmarshal(&badTarget), goubus.ErrInvalidResponse) {
		t.Fatal("expected ErrInvalidResponse on field type mismatch")
	}
}

func TestClient_DispatchAndControlErrorPaths(t *testing.T) {
	broker, client := newMuxBrokerClient(t)
	inSlow := make(chan struct{})

	pub, err := client.AddObject(t.Context(), goubus.ObjectDef{
		Name: "edge.obj",
		Methods: map[string]goubus.Method{
			"empty": {
				Policy:  goubus.MethodSignature{"any": goubus.ArgTypeUnknown},
				Handler: func(context.Context, goubus.CallRequest) (any, error) { return struct{}{}, nil },
			},
			"callerr": {
				Handler: func(context.Context, goubus.CallRequest) (any, error) {
					return nil, &goubus.CallError{Code: goubus.StatusTimeout, Err: goubus.ErrTimeout}
				},
			},
			"unknownerr": {
				Handler: func(context.Context, goubus.CallRequest) (any, error) { return nil, errTestInternalFailure },
			},
			"badreturn": {
				Handler: func(context.Context, goubus.CallRequest) (any, error) { return []int{1, 2, 3}, nil },
			},
			"slow": {
				Handler: func(ctx context.Context, _ goubus.CallRequest) (any, error) {
					close(inSlow)
					<-ctx.Done()

					return struct{}{}, nil
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("AddObject: %v", err)
	}

	assertEdgeMethodInvocations(t, broker, client, pub)
	assertMalformedInboundFramesIgnored(t, broker, client)

	sub, err := client.Listen(t.Context(), "*", func(goubus.Event) {})
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	broker.pushEvent(pub.ID(), "slow", nil)
	<-inSlow
	// Closing client while pub, active handler, and sub are live exercises full teardown.
	_ = client.Close()
	_ = client.Close()
	_ = pub.Close()
	_ = sub.Close()
	<-pub.Done()
	<-sub.Done()
}

func assertEdgeMethodInvocations(
	t *testing.T,
	broker *muxMockBroker,
	client *goubus.Client,
	pub *goubus.RegisteredObject,
) {
	t.Helper()

	res, err := client.Call(t.Context(), "edge.obj", "empty", nil)
	if err != nil {
		t.Fatalf("Call(empty): %v", err)
	}
	if !errors.Is(res.Unmarshal(&map[string]any{}), goubus.ErrNoData) {
		t.Fatal("expected ErrNoData for nil handler reply")
	}

	_, code := broker.invokeClientObject(t, pub.ID(), "callerr", nil)
	if code != goubus.StatusTimeout {
		t.Errorf("callerr status = %d, want StatusTimeout", code)
	}
	_, code = broker.invokeClientObject(t, pub.ID(), "unknownerr", nil)
	if code != goubus.StatusUnknown {
		t.Errorf("unknownerr status = %d, want StatusUnknown", code)
	}
	_, code = broker.invokeClientObject(t, pub.ID(), "badreturn", nil)
	if code != goubus.StatusInvalidParameter {
		t.Errorf("badreturn status = %d, want StatusInvalidParameter", code)
	}
	_, code = broker.invokeClientObject(t, 0xdead0001, "missing", nil)
	if code != goubus.StatusNotFound {
		t.Errorf("unknown objID invoke status = %d, want StatusNotFound", code)
	}

	if !errors.Is(pub.Notify(t.Context(), "", nil), goubus.ErrInvalidParameter) {
		t.Error("expected ErrInvalidParameter on empty notify method")
	}
	if !errors.Is(pub.Notify(t.Context(), "ev", make(chan int)), goubus.ErrInvalidParameter) {
		t.Error("expected ErrInvalidParameter on unmarshalable notify payload")
	}
}

func assertMalformedInboundFramesIgnored(t *testing.T, broker *muxMockBroker, client *goubus.Client) {
	t.Helper()

	emptyBody := []byte{0, 0, 0, 4}
	broker.sendFrame(blobmsg.UbusMsgMonitor, 0, 1, emptyBody)
	broker.sendFrame(blobmsg.UbusMsgInvoke, 0, 1, emptyBody)
	broker.sendFrame(blobmsg.UbusMsgNotify, 0, 1, emptyBody)
	broker.sendFrame(blobmsg.UbusMsgUnsubscribe, 0, 1, emptyBody)

	objOnlyBody, _ := blobmsg.CreateBlobMessage(map[uint32]any{
		blobmsg.UbusAttrObjID: uint32(999),
	}, nil)
	broker.sendFrame(blobmsg.UbusMsgNotify, 0, 1, objOnlyBody)

	pingErr := client.Ping(t.Context())
	if pingErr != nil {
		t.Fatalf("Ping after ignored frames: %v", pingErr)
	}
}

func TestClient_PubSubQueueOverflowAndSubscribeRefresh(t *testing.T) {
	broker, client := newMuxBrokerClient(t)

	blockHandler := make(chan struct{})
	firstEntered := make(chan struct{})
	sub, err := client.Listen(t.Context(), "*", func(goubus.Event) {
		select {
		case firstEntered <- struct{}{}:
			<-blockHandler
		default:
		}
	})
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}

	listenerID := subObjID(t, broker)
	// Deliver one synchronous event without no_reply, then flood past maxSubscriptionQueue (1024).
	syncEventBody, _ := blobmsg.CreateBlobMessage(map[uint32]any{
		blobmsg.UbusAttrObjID:  listenerID,
		blobmsg.UbusAttrMethod: "ev",
	}, nil)
	broker.sendFrame(blobmsg.UbusMsgInvoke, 77, 0xfeedbeef, syncEventBody)
	<-firstEntered

	for range 1030 {
		broker.pushEvent(listenerID, "flood", nil)
	}
	close(blockHandler)
	_ = sub.Close()
	_ = sub.Close()

	assertSubscribeCacheRefresh(t, broker, client)
}

func assertSubscribeCacheRefresh(t *testing.T, broker *muxMockBroker, client *goubus.Client) {
	t.Helper()

	listed, err := client.List(t.Context(), "*")
	if err != nil || len(listed) == 0 {
		t.Fatalf("List(*): %v", err)
	}

	// Warm objectCache["hostapd.phy0-ap0"] = 200 via Subscribe.
	broker.writeMu.Lock()
	broker.registered[200] = "hostapd.phy0-ap0"
	broker.writeMu.Unlock()

	warmSub, err := client.Subscribe(t.Context(), "hostapd.phy0-ap0", func(goubus.Notification) {})
	if err != nil {
		t.Fatalf("initial Subscribe: %v", err)
	}
	_ = warmSub.Close()

	// Simulate hostapd restarting at ID 201 while client still caches 200.
	broker.writeMu.Lock()
	delete(broker.registered, 200)
	broker.registered[201] = "hostapd.phy0-ap0"
	broker.writeMu.Unlock()

	sub, err := client.Subscribe(t.Context(), "hostapd.phy0-ap0", func(goubus.Notification) {})
	if err != nil {
		t.Fatalf("Subscribe after target restart: %v", err)
	}
	_ = sub.Close()

	// Simulate target disappearing permanently while still cached at 201.
	broker.writeMu.Lock()
	delete(broker.registered, 201)
	broker.writeMu.Unlock()

	_, err = client.Subscribe(t.Context(), "hostapd.phy0-ap0", func(goubus.Notification) {})
	if !errors.Is(err, goubus.ErrNotFound) {
		t.Errorf("Subscribe to vanished cached target = %v, want ErrNotFound", err)
	}
	_, err = client.Call(t.Context(), "hostapd.phy0-ap0", "status", nil)
	if !errors.Is(err, goubus.ErrNotFound) {
		t.Errorf("Call to vanished target = %v, want ErrNotFound", err)
	}
}

func TestNewClient_EOF(t *testing.T) {
	sockPath := filepath.Join(t.TempDir(), "u.sock")
	var lc net.ListenConfig
	ln, err := lc.Listen(t.Context(), "unix", sockPath)
	if err != nil {
		t.Fatalf("Listen unix: %v", err)
	}
	if ul, ok := ln.(*net.UnixListener); ok {
		ul.SetUnlinkOnClose(false)
	}
	go func() {
		conn, acceptErr := ln.Accept()
		if acceptErr == nil {
			_ = conn.Close()
		}
	}()

	_, err = goubus.NewClient(t.Context(), sockPath)
	if !errors.Is(err, goubus.ErrConnectionFailed) {
		t.Errorf("NewClient(EOF hello) = %v, want ErrConnectionFailed", err)
	}
	_ = ln.Close()
	_, err = goubus.NewClient(t.Context(), sockPath)
	if !errors.Is(err, goubus.ErrConnectionFailed) {
		t.Errorf("NewClient(closed listener) = %v, want ErrConnectionFailed", err)
	}
}

func TestClient_WireRejectionStatuses(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	go serveRejectingUbusd(serverConn)

	client, err := goubus.NewClientFromConn(t.Context(), clientConn)
	if err != nil {
		t.Fatalf("NewClientFromConn: %v", err)
	}
	t.Cleanup(func() {
		_ = client.Close()
		_ = serverConn.Close()
	})

	if !errors.Is(client.Ping(t.Context()), goubus.ErrPermissionDenied) {
		t.Error("expected ErrPermissionDenied from rejected Ping")
	}
	_, listErr := client.List(t.Context())
	if !errors.Is(listErr, goubus.ErrPermissionDenied) {
		t.Errorf("List rejected error = %v, want ErrPermissionDenied", listErr)
	}
	assertAddObjectRejections(t, client)

	_, err = client.Listen(t.Context(), "*", func(goubus.Event) {})
	if !errors.Is(err, goubus.ErrPermissionDenied) {
		t.Errorf("Listen rejected error = %v, want ErrPermissionDenied", err)
	}
	if !errors.Is(client.SendEvent(t.Context(), "ev", make(chan int)), goubus.ErrInvalidParameter) {
		t.Error("expected ErrInvalidParameter on invalid SendEvent payload")
	}
}

func assertAddObjectRejections(t *testing.T, client *goubus.Client) {
	t.Helper()

	noop := func(context.Context, goubus.CallRequest) (any, error) { return map[string]any{}, nil }
	_, err := client.AddObject(t.Context(), goubus.ObjectDef{
		Name:    "bad.policy",
		Methods: map[string]goubus.Method{"m": {Policy: goubus.MethodSignature{"": goubus.ArgTypeString}, Handler: noop}},
	})
	if !errors.Is(err, goubus.ErrInvalidParameter) {
		t.Errorf("AddObject empty policy arg error = %v, want ErrInvalidParameter", err)
	}
	_, err = client.AddObject(t.Context(), goubus.ObjectDef{
		Name:    "denied.obj",
		Methods: map[string]goubus.Method{"m": {Handler: noop}},
	})
	if !errors.Is(err, goubus.ErrPermissionDenied) {
		t.Errorf("AddObject rejected error = %v, want ErrPermissionDenied", err)
	}
	_, err = client.AddObject(t.Context(), goubus.ObjectDef{
		Name:    "missing.id",
		Methods: map[string]goubus.Method{"m": {Handler: noop}},
	})
	if !errors.Is(err, goubus.ErrInvalidResponse) {
		t.Errorf("AddObject missing objid error = %v, want ErrInvalidResponse", err)
	}
	pub, err := client.AddObject(t.Context(), goubus.ObjectDef{
		Name:    "allow.add",
		Methods: map[string]goubus.Method{"m": {Handler: noop}},
	})
	if err != nil {
		t.Fatalf("AddObject(allow.add): %v", err)
	}
	if !errors.Is(pub.Notify(t.Context(), "ev", nil), goubus.ErrPermissionDenied) {
		t.Error("expected ErrPermissionDenied on rejected Notify")
	}
	if !errors.Is(pub.Close(), goubus.ErrPermissionDenied) {
		t.Error("expected ErrPermissionDenied on rejected RemoveObject")
	}
	if pub.Close() != nil {
		t.Error("expected second pub.Close() to return nil")
	}
	if !errors.Is(pub.Notify(t.Context(), "ev", nil), goubus.ErrClosed) {
		t.Error("expected ErrClosed on Notify after pub.Close()")
	}
}

func TestClient_PubSubRegistrationRollbackOnFailure(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	go servePartialPubSubUbusd(serverConn)

	client, err := goubus.NewClientFromConn(t.Context(), clientConn)
	if err != nil {
		t.Fatalf("NewClientFromConn: %v", err)
	}
	t.Cleanup(func() {
		_ = client.Close()
		_ = serverConn.Close()
	})

	_, err = client.Subscribe(t.Context(), "missing.target", func(goubus.Notification) {})
	if !errors.Is(err, goubus.ErrNotFound) {
		t.Errorf("Subscribe(missing.target) = %v, want ErrNotFound", err)
	}
	_, err = client.Subscribe(t.Context(), "deny.sub", func(goubus.Notification) {})
	if !errors.Is(err, goubus.ErrPermissionDenied) {
		t.Errorf("Subscribe(deny.sub) = %v, want ErrPermissionDenied", err)
	}
	_, err = client.Listen(t.Context(), "deny.ev", func(goubus.Event) {})
	if !errors.Is(err, goubus.ErrPermissionDenied) {
		t.Errorf("Listen(deny.ev) = %v, want ErrPermissionDenied", err)
	}

	_ = client.Close()
	if !errors.Is(client.Ping(t.Context()), goubus.ErrClosed) {
		t.Error("expected ErrClosed from Ping after Close")
	}
}

func serveRejectingUbusd(conn net.Conn) {
	defer func() { _ = conn.Close() }()

	var hello bytes.Buffer
	_ = blobmsg.EncodeHeader(&hello, &blobmsg.UbusMessageHeader{Type: blobmsg.UbusMsgHello, Peer: 1})
	_, _ = hello.Write([]byte{0, 0, 0, 4})
	_, _ = conn.Write(hello.Bytes())

	for {
		hdr, payload, err := blobmsg.ReadMessage(conn)
		if err != nil {
			return
		}
		code := uint32(goubus.StatusPermissionDenied)
		if hdr.Type == blobmsg.UbusMsgAddObject {
			attrs, _ := blobmsg.ParseTopLevelAttributes(payload)
			switch attrs["objpath"] {
			case "missing.id":
				code = goubus.StatusOK
			case "allow.add":
				code = goubus.StatusOK
				data := blobmsg.EncodeUintAttributes([2]uint32{blobmsg.UbusAttrObjID, 555})
				sendMsg(conn, blobmsg.UbusMsgData, hdr.Seq, data)
			}
		}
		statusBody := blobmsg.EncodeUintAttributes([2]uint32{blobmsg.UbusAttrStatus, code})
		sendMsg(conn, blobmsg.UbusMsgStatus, hdr.Seq, statusBody)
	}
}

func servePartialPubSubUbusd(conn net.Conn) {
	defer func() { _ = conn.Close() }()

	var hello bytes.Buffer
	_ = blobmsg.EncodeHeader(&hello, &blobmsg.UbusMessageHeader{Type: blobmsg.UbusMsgHello, Peer: 1})
	_, _ = hello.Write([]byte{0, 0, 0, 4})
	_, _ = conn.Write(hello.Bytes())

	var nextID uint32 = 700
	for {
		hdr, payload, err := blobmsg.ReadMessage(conn)
		if err != nil {
			return
		}
		attrs, _ := blobmsg.ParseTopLevelAttributes(payload)
		code := replyPartialPubSub(conn, hdr, attrs, &nextID)
		sendMsg(conn, blobmsg.UbusMsgStatus, hdr.Seq, blobmsg.EncodeUintAttributes([2]uint32{blobmsg.UbusAttrStatus, code}))
	}
}

func replyPartialPubSub(
	conn net.Conn,
	hdr *blobmsg.UbusMessageHeader,
	attrs map[string]any,
	nextID *uint32,
) uint32 {
	switch hdr.Type {
	case blobmsg.UbusMsgLookup:
		if attrs["objpath"] == "deny.sub" {
			dataBody, _ := blobmsg.CreateBlobMessage(map[uint32]any{
				blobmsg.UbusAttrObjPath: "deny.sub",
				blobmsg.UbusAttrObjID:   uint32(300),
			}, nil)
			sendMsg(conn, blobmsg.UbusMsgData, hdr.Seq, dataBody)
		}

		return goubus.StatusOK
	case blobmsg.UbusMsgAddObject:
		*nextID++
		sendMsg(conn, blobmsg.UbusMsgData, hdr.Seq, blobmsg.EncodeUintAttributes([2]uint32{blobmsg.UbusAttrObjID, *nextID}))

		return goubus.StatusOK
	case blobmsg.UbusMsgRemoveObject:
		return goubus.StatusOK
	default:
		return goubus.StatusPermissionDenied
	}
}

func newMuxBrokerClientWithOpts(t *testing.T, opts ...goubus.Option) (*muxMockBroker, *goubus.Client) {
	t.Helper()

	serverConn, clientConn := net.Pipe()
	broker := &muxMockBroker{
		conn:       serverConn,
		nextObjID:  0x90000001,
		registered: map[uint32]string{200: "hostapd.phy0-ap0"},
	}
	go broker.serve()

	client, err := goubus.NewClientFromConn(t.Context(), clientConn, opts...)
	if err != nil {
		_ = serverConn.Close()
		t.Fatalf("NewClientFromConn: %v", err)
	}
	t.Cleanup(func() {
		_ = client.Close()
		_ = serverConn.Close()
	})

	return broker, client
}
