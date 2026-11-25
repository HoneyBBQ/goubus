package goubus_test

import (
	"bytes"
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/honeybbq/goubus/v2"
	"github.com/honeybbq/goubus/v2/internal/blobmsg"
)

var (
	replyCaptureMu     sync.Mutex
	replyCaptureData   map[string]any
	replyCaptureStatus chan uint32
)

type muxMockBroker struct {
	conn       net.Conn
	registered map[uint32]string
	writeMu    sync.Mutex
	nextObjID  uint32
}

func TestSocketClient_OutOfOrderMultiplexing(t *testing.T) {
	slowRelease := make(chan struct{})
	slowArrived := make(chan struct{})
	client := newPipeSocketClient(t, outOfOrderMuxHandler(slowArrived, slowRelease))

	slowDone := make(chan string, 1)
	go func() {
		res, err := goubus.Call[struct {
			Method string `json:"method"`
		}](t.Context(), client, testSystemService, "slow", nil)
		if err != nil {
			t.Errorf("slow call failed: %v", err)
		}
		slowDone <- res.Method
	}()

	waitForSignal(t, slowArrived, "slow invoke did not arrive")

	// Fast call must overtake the blocked slow call on the same SocketClient.
	fastRes, err := goubus.Call[struct {
		Method string `json:"method"`
	}](t.Context(), client, testSystemService, "fast", nil)
	if err != nil || fastRes.Method != "fast" {
		t.Fatalf("fast call = %+v, err %v; want fast", fastRes, err)
	}

	close(slowRelease)
	select {
	case got := <-slowDone:
		if got != "slow" {
			t.Fatalf("slow call method = %q, want slow", got)
		}
	case <-time.After(time.Second):
		t.Fatal("slow call did not complete after release")
	}
}

func outOfOrderMuxHandler(slowArrived, slowRelease chan struct{}) socketMockHandler {
	var writeMu sync.Mutex

	return func(conn net.Conn, hdr *blobmsg.UbusMessageHeader, payload []byte) {
		if hdr.Type == blobmsg.UbusMsgLookup {
			writeMu.Lock()
			handleLookup(conn, hdr.Seq, payload)
			writeMu.Unlock()

			return
		}
		if hdr.Type != blobmsg.UbusMsgInvoke {
			return
		}

		attrs, _ := blobmsg.ParseTopLevelAttributes(payload)
		method, _ := attrs["method"].(string)
		go func(seq uint16, method string) {
			if method == "slow" {
				close(slowArrived)
				<-slowRelease
			}
			encoded, _ := blobmsg.CreateBlobmsgTable(map[string]any{"method": method})
			dataBody, _ := blobmsg.CreateBlobMessage(map[uint32]any{
				blobmsg.UbusAttrData: encoded[blobmsg.BlobHeaderBytes:],
			}, nil)
			statusBody, _ := blobmsg.CreateBlobMessage(map[uint32]any{
				blobmsg.UbusAttrStatus: uint32(0),
			}, nil)

			writeMu.Lock()
			sendMsg(conn, blobmsg.UbusMsgData, seq, dataBody)
			sendMsg(conn, blobmsg.UbusMsgStatus, seq, statusBody)
			writeMu.Unlock()
		}(hdr.Seq, method)
	}
}

func TestSocketClient_ListenAndSendEvent(t *testing.T) {
	broker, client := newMuxBrokerClient(t)

	_, err := client.Listen(t.Context(), "*", nil)
	if !errors.Is(err, goubus.ErrInvalidParameter) {
		t.Fatalf("Listen(nil handler) error = %v, want ErrInvalidParameter", err)
	}
	err = client.SendEvent(t.Context(), "", nil)
	if !errors.Is(err, goubus.ErrInvalidParameter) {
		t.Fatalf("SendEvent(empty id) error = %v, want ErrInvalidParameter", err)
	}

	received := make(chan goubus.Event, 1)
	sub, err := client.Listen(t.Context(), "", func(ev goubus.Event) {
		received <- ev
	})
	if err != nil {
		t.Fatalf("Listen(): %v", err)
	}

	err = client.SendEvent(t.Context(), "network.interface", map[string]any{"interface": "wan", "up": true})
	if err != nil {
		t.Fatalf("SendEvent(): %v", err)
	}

	broker.pushEvent(subObjID(t, broker), "network.interface", map[string]any{"interface": "wan", "up": true})

	select {
	case ev := <-received:
		var payload struct {
			Interface string `json:"interface"`
			Up        bool   `json:"up"`
		}
		unmarshalErr := ev.Unmarshal(&payload)
		if unmarshalErr != nil {
			t.Fatalf("Event.Unmarshal(): %v", unmarshalErr)
		}
		if ev.Type != "network.interface" || payload.Interface != "wan" || !payload.Up {
			t.Errorf("Event = (%q, %+v)", ev.Type, payload)
		}
	case <-time.After(time.Second):
		t.Fatal("did not receive broadcast event")
	}

	err = sub.Close()
	if err != nil {
		t.Fatalf("sub.Close(): %v", err)
	}
	select {
	case <-sub.Done():
	default:
		t.Fatal("sub.Done() not closed after Close()")
	}
}

func TestSocketClient_AddObjectInvokeAndNotify(t *testing.T) {
	broker, client := newMuxBrokerClient(t)
	assertInvalidObjectDefs(t, client)

	pub, err := client.AddObject(t.Context(), goubus.ObjectDef{
		Name: "demo.agent",
		Methods: map[string]goubus.Method{
			"echo": {
				Policy: goubus.MethodSignature{
					"msg":     goubus.ArgTypeString,
					"count":   goubus.ArgTypeNumber,
					"enabled": goubus.ArgTypeBoolean,
					"items":   goubus.ArgTypeArray,
					"meta":    goubus.ArgTypeObject,
				},
				Handler: func(ctx context.Context, req goubus.CallRequest) (any, error) {
					var input struct {
						Msg string `json:"msg"`
					}
					decodeErr := req.Unmarshal(&input)
					if decodeErr != nil {
						return nil, goubus.ErrInvalidParameter
					}
					// Verify re-entrant outbound Call from inside a MethodHandler does not deadlock.
					sysInfo, callErr := goubus.Call[struct {
						Hostname string `json:"hostname"`
					}](ctx, client, testSystemService, testInfoMethod, nil)
					if callErr != nil {
						return nil, callErr
					}

					return map[string]any{
						"echo":   input.Msg,
						"caller": req.User + ":" + req.Group,
						"host":   sysInfo.Hostname,
					}, nil
				},
			},
			"fail": {
				Handler: func(context.Context, goubus.CallRequest) (any, error) {
					return nil, goubus.ErrPermissionDenied
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("AddObject(): %v", err)
	}
	if pub.Name() != "demo.agent" || pub.ID() == 0 {
		t.Fatalf("published object = (%q, %d)", pub.Name(), pub.ID())
	}

	assertObjectNotificationsAndCalls(t, broker, pub)

	err = pub.Close()
	if err != nil {
		t.Fatalf("pub.Close(): %v", err)
	}
	err = pub.Notify(t.Context(), "tick", nil)
	if !errors.Is(err, goubus.ErrClosed) {
		t.Errorf("Notify after Close error = %v, want ErrClosed", err)
	}
}

func TestSocketClient_SubscribeAndTargetRemoval(t *testing.T) {
	broker, client := newMuxBrokerClient(t)

	_, err := client.Subscribe(t.Context(), "", func(goubus.Notification) {})
	if !errors.Is(err, goubus.ErrInvalidParameter) {
		t.Fatalf("Subscribe(empty object) error = %v, want ErrInvalidParameter", err)
	}
	_, err = client.Subscribe(t.Context(), testSystemService, nil)
	if !errors.Is(err, goubus.ErrInvalidParameter) {
		t.Fatalf("Subscribe(nil handler) error = %v, want ErrInvalidParameter", err)
	}

	gotCh := make(chan goubus.Notification, 1)
	sub, err := client.Subscribe(t.Context(), testSystemService, func(n goubus.Notification) {
		gotCh <- n
	})
	if err != nil {
		t.Fatalf("Subscribe(): %v", err)
	}

	listenerID := subObjID(t, broker)
	broker.pushEvent(listenerID, "state.changed", map[string]any{"code": 7})

	select {
	case note := <-gotCh:
		var data struct {
			Code int `json:"code"`
		}
		unmarshalErr := note.Unmarshal(&data)
		if unmarshalErr != nil {
			t.Fatalf("Notification.Unmarshal(): %v", unmarshalErr)
		}
		if note.Object != testSystemService || note.Method != "state.changed" || data.Code != 7 {
			t.Errorf("Notification = %+v (%+v)", note, data)
		}
	case <-time.After(time.Second):
		t.Fatal("did not receive object notification")
	}

	// Simulate target object removal (UBUS_MSG_UNSUBSCRIBE from ubusd).
	broker.pushUnsubscribe(listenerID)
	select {
	case <-sub.Done():
	case <-time.After(time.Second):
		t.Fatal("sub.Done() did not close after target object removal")
	}

	// Verify readLoop was not blocked by target removal cleanup.
	callCtx, cancel := context.WithTimeout(t.Context(), 200*time.Millisecond)
	defer cancel()
	_, err = client.Call(callCtx, testSystemService, testInfoMethod, nil)
	if err != nil {
		t.Fatalf("Call immediately after target removal failed: %v", err)
	}
}

func TestClient_StaleObjectCacheRefreshesAndLifecycleSignals(t *testing.T) {
	broker, client := newMuxBrokerClient(t)
	if client.Err() != nil {
		t.Fatalf("open Client.Err() = %v, want nil", client.Err())
	}
	pingErr := client.Ping(t.Context())
	if pingErr != nil {
		t.Fatalf("Client.Ping(): %v", pingErr)
	}

	subStates := make(chan bool, 2)
	pub1, err := client.AddObject(t.Context(), goubus.ObjectDef{
		Name: "svc.restarting",
		OnSubscribe: func(active bool) {
			subStates <- active
		},
		Methods: map[string]goubus.Method{
			"gen": {
				Handler: func(context.Context, goubus.CallRequest) (any, error) {
					return map[string]any{"gen": 1}, nil
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("AddObject(pub1): %v", err)
	}

	broker.pushSubscriberState(pub1.ID(), true)
	broker.pushSubscriberState(pub1.ID(), false)
	if !<-subStates || <-subStates {
		t.Fatal("OnSubscribe transitions did not arrive in order")
	}

	assertRestartCacheRefresh(t, broker, client, pub1)

	err = client.Close()
	if err != nil {
		t.Fatalf("Client.Close(): %v", err)
	}
	select {
	case <-client.Done():
	default:
		t.Fatal("Client.Done() not closed after Close()")
	}
	if !errors.Is(client.Err(), goubus.ErrClosed) {
		t.Fatalf("closed Client.Err() = %v, want ErrClosed", client.Err())
	}
}

func assertRestartCacheRefresh(
	t *testing.T,
	broker *muxMockBroker,
	client *goubus.Client,
	pub1 *goubus.RegisteredObject,
) {
	t.Helper()

	first, err := goubus.Call[struct {
		Gen int `json:"gen"`
	}](t.Context(), client, "svc.restarting", "gen", nil)
	if err != nil || first.Gen != 1 {
		t.Fatalf("first Call = %+v, err %v", first, err)
	}

	// Unregister pub1 (evicting it on the broker) and register pub2 under the same name with a new ID.
	_ = pub1.Close()
	select {
	case <-pub1.Done():
	default:
		t.Fatal("pub1.Done() not closed after Close()")
	}
	broker.writeMu.Lock()
	delete(broker.registered, pub1.ID())
	broker.writeMu.Unlock()

	pub2, err := client.AddObject(t.Context(), goubus.ObjectDef{
		Name: "svc.restarting",
		Methods: map[string]goubus.Method{
			"gen": {
				Handler: func(context.Context, goubus.CallRequest) (any, error) {
					return map[string]any{"gen": 2}, nil
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("AddObject(pub2): %v", err)
	}
	defer func() { _ = pub2.Close() }()

	second, err := goubus.Call[struct {
		Gen int `json:"gen"`
	}](t.Context(), client, "svc.restarting", "gen", nil)
	if err != nil || second.Gen != 2 {
		t.Fatalf("Call after object restart = %+v, err %v; want gen=2", second, err)
	}
}

func assertInvalidObjectDefs(t *testing.T, client *goubus.Client) {
	t.Helper()

	_, err := client.AddObject(t.Context(), goubus.ObjectDef{})
	if !errors.Is(err, goubus.ErrInvalidParameter) {
		t.Fatalf("AddObject(empty name) error = %v, want ErrInvalidParameter", err)
	}
	_, err = client.AddObject(t.Context(), goubus.ObjectDef{
		Name:    "bad.handler",
		Methods: map[string]goubus.Method{"ping": {}},
	})
	if !errors.Is(err, goubus.ErrInvalidParameter) {
		t.Fatalf("AddObject(nil handler) error = %v, want ErrInvalidParameter", err)
	}
	_, err = client.AddObject(t.Context(), goubus.ObjectDef{
		Name: "bad.argtype",
		Methods: map[string]goubus.Method{
			"ping": {
				Policy:  goubus.MethodSignature{"x": goubus.ArgType("bogus")},
				Handler: func(context.Context, goubus.CallRequest) (any, error) { return map[string]any{}, nil },
			},
		},
	})
	if !errors.Is(err, goubus.ErrInvalidParameter) {
		t.Fatalf("AddObject(invalid ArgType) error = %v, want ErrInvalidParameter", err)
	}
}

func assertObjectNotificationsAndCalls(t *testing.T, broker *muxMockBroker, pub *goubus.RegisteredObject) {
	t.Helper()

	broker.pushSubscriberState(pub.ID(), true)
	deadline := time.Now().Add(time.Second)
	for !pub.HasSubscribers() && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if !pub.HasSubscribers() {
		t.Fatal("HasSubscribers() = false, want true after active notify")
	}

	err := pub.Notify(t.Context(), "tick", map[string]any{"seq": 1})
	if err != nil {
		t.Fatalf("pub.Notify(): %v", err)
	}

	data, status := broker.invokeClientObject(t, pub.ID(), "echo", map[string]any{"msg": "hello"})
	if status != goubus.StatusOK || data["echo"] != "hello" ||
		data["caller"] != "root:wheel" || data["host"] != testHostname {
		t.Fatalf("echo invoke = (%#v, status %d)", data, status)
	}

	_, failStatus := broker.invokeClientObject(t, pub.ID(), "fail", nil)
	if failStatus != goubus.StatusPermissionDenied {
		t.Errorf("fail invoke status = %d, want %d", failStatus, goubus.StatusPermissionDenied)
	}

	_, missingStatus := broker.invokeClientObject(t, pub.ID(), "nonexistent", nil)
	if missingStatus != goubus.StatusMethodNotFound {
		t.Errorf("missing method invoke status = %d, want %d", missingStatus, goubus.StatusMethodNotFound)
	}
}

func newMuxBrokerClient(t *testing.T) (*muxMockBroker, *goubus.Client) {
	t.Helper()

	serverConn, clientConn := net.Pipe()
	broker := &muxMockBroker{
		conn:       serverConn,
		nextObjID:  0x90000001, // High bit set to verify int32 event registration encoding.
		registered: make(map[uint32]string),
	}

	go broker.serve()

	client, err := goubus.NewClientFromConn(t.Context(), clientConn)
	if err != nil {
		_ = serverConn.Close()
		t.Fatalf("NewSocketClientFromConn: %v", err)
	}
	t.Cleanup(func() {
		_ = client.Close()
		_ = serverConn.Close()
	})

	return broker, client
}

func subObjID(t *testing.T, broker *muxMockBroker) uint32 {
	t.Helper()

	broker.writeMu.Lock()
	defer broker.writeMu.Unlock()
	for id, name := range broker.registered {
		if name == "" {
			return id
		}
	}
	t.Fatal("no anonymous listener object registered")

	return 0
}

func startExampleUbusd() (string, func()) {
	dir, err := os.MkdirTemp("", "ub")
	if err != nil {
		panic(err)
	}
	sockPath := filepath.Join(dir, "u.sock")

	var lc net.ListenConfig
	listener, err := lc.Listen(context.Background(), "unix", sockPath)
	if err != nil {
		_ = os.RemoveAll(dir)
		panic(err)
	}

	go func() {
		for {
			conn, acceptErr := listener.Accept()
			if acceptErr != nil {
				return
			}
			broker := &muxMockBroker{
				conn:       conn,
				nextObjID:  0x90000001,
				registered: map[uint32]string{200: "hostapd.phy0-ap0"},
			}
			go broker.serve()
		}
	}()

	return sockPath, func() {
		_ = listener.Close()
		_ = os.RemoveAll(dir)
	}
}

func (b *muxMockBroker) serve() {
	defer func() { _ = b.conn.Close() }()

	b.sendFrame(blobmsg.UbusMsgHello, 0, 1, []byte{0, 0, 0, 4})

	for {
		hdr, payload, err := blobmsg.ReadMessage(b.conn)
		if err != nil {
			return
		}
		if hdr.Peer == 0xfeedbeef {
			b.deliverIncomingReply(hdr, payload)

			continue
		}
		b.handleClientRequest(hdr, payload)
	}
}

func (b *muxMockBroker) deliverIncomingReply(hdr *blobmsg.UbusMessageHeader, payload []byte) {
	attrs, _ := blobmsg.ParseTopLevelAttributes(payload)
	replyCaptureMu.Lock()
	defer replyCaptureMu.Unlock()

	if hdr.Type == blobmsg.UbusMsgData {
		replyCaptureData = blobmsg.ExtractDataSection(attrs)
	}
	if hdr.Type == blobmsg.UbusMsgStatus && replyCaptureStatus != nil {
		code, _ := blobmsg.ReadUint(attrs["status"])
		replyCaptureStatus <- code
	}
}

func (b *muxMockBroker) handleClientRequest(hdr *blobmsg.UbusMessageHeader, payload []byte) {
	attrs, _ := blobmsg.ParseTopLevelAttributes(payload)

	switch hdr.Type {
	case blobmsg.UbusMsgLookup:
		b.lookupObject(hdr.Seq, attrs, payload)
	case blobmsg.UbusMsgInvoke:
		b.dispatchInvoke(hdr.Seq, attrs, payload)
	case blobmsg.UbusMsgAddObject:
		b.writeMu.Lock()
		id := b.nextObjID
		b.nextObjID++
		name, _ := attrs["objpath"].(string)
		b.registered[id] = name
		b.writeMu.Unlock()

		dataBody, _ := blobmsg.CreateBlobMessage(map[uint32]any{
			blobmsg.UbusAttrObjID:   id,
			blobmsg.UbusAttrObjType: uint32(77),
		}, nil)
		b.sendFrame(blobmsg.UbusMsgData, hdr.Seq, 1, dataBody)
		b.sendStatus(hdr.Seq, 0)
	case blobmsg.UbusMsgSubscribe:
		b.sendStatus(hdr.Seq, 0)
		target, _ := blobmsg.ReadUint(attrs["target"])
		if target == 200 {
			subID, _ := blobmsg.ReadUint(attrs["objid"])
			b.pushEvent(subID, "sta.authorized", map[string]any{"address": "aa:bb:cc:dd:ee:ff"})
		}
	case blobmsg.UbusMsgRemoveObject:
		if id, ok := blobmsg.ReadUint(attrs["objid"]); ok {
			b.writeMu.Lock()
			delete(b.registered, id)
			b.writeMu.Unlock()
		}
		b.sendStatus(hdr.Seq, 0)
	case blobmsg.UbusMsgPing, blobmsg.UbusMsgUnsubscribe, blobmsg.UbusMsgNotify:
		b.sendStatus(hdr.Seq, 0)
	}
}

func (b *muxMockBroker) lookupObject(seq uint16, attrs map[string]any, payload []byte) {
	path, _ := attrs["objpath"].(string)
	if path != "" {
		b.writeMu.Lock()
		var matchedID uint32
		for id, name := range b.registered {
			if name == path {
				matchedID = id

				break
			}
		}
		b.writeMu.Unlock()

		if matchedID != 0 {
			dataBody, _ := blobmsg.CreateBlobMessage(map[uint32]any{
				blobmsg.UbusAttrObjPath: path,
				blobmsg.UbusAttrObjID:   matchedID,
			}, nil)
			b.sendFrame(blobmsg.UbusMsgData, seq, 1, dataBody)
			b.sendStatus(seq, 0)

			return
		}
	}

	b.writeMu.Lock()
	handleLookup(b.conn, seq, payload)
	b.writeMu.Unlock()
}

func (b *muxMockBroker) dispatchInvoke(seq uint16, attrs map[string]any, payload []byte) {
	objID, _ := blobmsg.ReadUint(attrs["objid"])
	method, _ := attrs["method"].(string)
	if objID == 1 {
		b.sendStatus(seq, 0)
		if method == "send" {
			b.forwardEventSend(attrs)
		}

		return
	}
	if objID >= 0x90000000 {
		b.writeMu.Lock()
		_, exists := b.registered[objID]
		b.writeMu.Unlock()
		if !exists {
			b.sendStatus(seq, goubus.StatusNotFound)

			return
		}
		go b.forwardRegisteredInvoke(seq, objID, method, blobmsg.ExtractDataSection(attrs))

		return
	}

	b.writeMu.Lock()
	handleInvoke(b.conn, seq, payload)
	b.writeMu.Unlock()
}

func (b *muxMockBroker) forwardEventSend(attrs map[string]any) {
	evData := blobmsg.ExtractDataSection(attrs)
	evID, _ := evData["id"].(string)
	inner, _ := evData["data"].(map[string]any)
	if evID == "" {
		return
	}

	b.writeMu.Lock()
	var listeners []uint32
	for id, name := range b.registered {
		if name == "" {
			listeners = append(listeners, id)
		}
	}
	b.writeMu.Unlock()

	for _, id := range listeners {
		b.pushEvent(id, evID, inner)
	}
}

func (b *muxMockBroker) forwardRegisteredInvoke(seq uint16, objID uint32, method string, reqData map[string]any) {
	data, code, ok := b.callClientObject(objID, method, reqData)
	if !ok {
		return
	}
	if len(data) > 0 {
		encoded, _ := blobmsg.CreateBlobmsgTable(data)
		dataBody, _ := blobmsg.CreateBlobMessage(map[uint32]any{
			blobmsg.UbusAttrData: encoded[blobmsg.BlobHeaderBytes:],
		}, nil)
		b.sendFrame(blobmsg.UbusMsgData, seq, 1, dataBody)
	}
	b.sendStatus(seq, code)
}

func (b *muxMockBroker) invokeClientObject(
	t *testing.T,
	objID uint32,
	method string,
	args map[string]any,
) (map[string]any, uint32) {
	t.Helper()

	data, code, ok := b.callClientObject(objID, method, args)
	if !ok {
		t.Fatalf("timed out waiting for client object %s reply", method)
	}

	return data, code
}

func (b *muxMockBroker) callClientObject(
	objID uint32,
	method string,
	args map[string]any,
) (map[string]any, uint32, bool) {
	statusCh := make(chan uint32, 1)
	replyCaptureMu.Lock()
	replyCaptureData = nil
	replyCaptureStatus = statusCh
	replyCaptureMu.Unlock()

	argBytes, _ := blobmsg.CreateBlobmsgData(args)
	msgAttrs := map[uint32]any{
		blobmsg.UbusAttrObjID:  objID,
		blobmsg.UbusAttrMethod: method,
		blobmsg.UbusAttrUser:   "root",
		blobmsg.UbusAttrGroup:  "wheel",
	}
	if len(argBytes) > 0 {
		msgAttrs[blobmsg.UbusAttrData] = argBytes
	}
	body, _ := blobmsg.CreateBlobMessage(msgAttrs, nil)
	b.sendFrame(blobmsg.UbusMsgInvoke, 42, 0xfeedbeef, body)

	select {
	case code := <-statusCh:
		replyCaptureMu.Lock()
		data := replyCaptureData
		replyCaptureMu.Unlock()

		return data, code, true
	case <-time.After(time.Second):
		return nil, 0, false
	}
}

func (b *muxMockBroker) pushEvent(objID uint32, method string, data map[string]any) {
	argBytes, _ := blobmsg.CreateBlobmsgData(data)
	body, _ := blobmsg.CreateBlobMessage(map[uint32]any{
		blobmsg.UbusAttrObjID:   objID,
		blobmsg.UbusAttrMethod:  method,
		blobmsg.UbusAttrData:    argBytes,
		blobmsg.UbusAttrNoReply: []byte{1},
	}, nil)
	b.sendFrame(blobmsg.UbusMsgInvoke, 1, 1, body)
}

func (b *muxMockBroker) pushSubscriberState(objID uint32, active bool) {
	flag := byte(0)
	if active {
		flag = 1
	}
	body, _ := blobmsg.CreateBlobMessage(map[uint32]any{
		blobmsg.UbusAttrObjID:  objID,
		blobmsg.UbusAttrActive: []byte{flag},
	}, nil)
	b.sendFrame(blobmsg.UbusMsgNotify, 0, 1, body)
}

func (b *muxMockBroker) pushUnsubscribe(objID uint32) {
	body, _ := blobmsg.CreateBlobMessage(map[uint32]any{
		blobmsg.UbusAttrObjID: objID,
	}, nil)
	b.sendFrame(blobmsg.UbusMsgUnsubscribe, 0, 1, body)
}

func (b *muxMockBroker) sendStatus(seq uint16, status uint32) {
	body, _ := blobmsg.CreateBlobMessage(map[uint32]any{
		blobmsg.UbusAttrStatus: status,
	}, nil)
	b.sendFrame(blobmsg.UbusMsgStatus, seq, 1, body)
}

func (b *muxMockBroker) sendFrame(msgType uint8, seq uint16, peer uint32, body []byte) {
	var buf bytes.Buffer
	_ = blobmsg.EncodeHeader(&buf, &blobmsg.UbusMessageHeader{
		Type: msgType,
		Seq:  seq,
		Peer: peer,
	})
	_, _ = buf.Write(body)

	b.writeMu.Lock()
	_, _ = b.conn.Write(buf.Bytes())
	b.writeMu.Unlock()
}
