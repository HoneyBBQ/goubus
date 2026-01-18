package goubus_test

import (
	"bytes"
	"context"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/honeybbq/goubus/v2"
	"github.com/honeybbq/goubus/v2/internal/blobmsg"
	"github.com/honeybbq/goubus/v2/internal/logging"
)

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

	res, err := client.Call(ctx, "system", "info", nil)
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

	if info.Hostname != "OpenWrt" {
		t.Errorf("expected OpenWrt, got %s", info.Hostname)
	}

	// Test cache: call again, should not trigger another lookup
	// (We can check this by making mockUbusd fail on second lookup if we wanted)
	_, err = client.Call(ctx, "system", "info", nil)
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

	if path == "system" {
		// Send Data
		dataAttrs := map[uint32]any{
			blobmsg.UbusAttrObjPath: "system",
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

	if objID == 100 && method == "info" {
		// Send Data
		respData := map[string]any{"hostname": "OpenWrt"}
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
