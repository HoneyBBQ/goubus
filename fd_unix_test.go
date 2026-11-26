//go:build unix

package goubus_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/honeybbq/goubus/v2"
	"github.com/honeybbq/goubus/v2/internal/blobmsg"
)

func TestClient_CallFDUnixSCMRights(t *testing.T) {
	sockPath := filepath.Join(t.TempDir(), "fd.sock")
	var lc net.ListenConfig
	ln, err := lc.Listen(t.Context(), "unix", sockPath)
	if err != nil {
		t.Fatalf("Listen unix: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	go serveUnixFDMock(ln)

	client, err := goubus.NewClient(t.Context(), sockPath)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })

	assertBidirectionalFDTransfer(t, client)
	assertCallFDValidationAndUnsolicitedDrop(t, client)
}

func assertBidirectionalFDTransfer(t *testing.T, client *goubus.Client) {
	t.Helper()

	sendR, sendW, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	defer func() { _ = sendR.Close() }()
	_, _ = sendW.WriteString("from-client")
	_ = sendW.Close()

	res, replyFile, err := client.CallFD(t.Context(), "log", "stream", map[string]any{"lines": 1}, sendR)
	if err != nil {
		t.Fatalf("CallFD: %v", err)
	}
	if replyFile == nil {
		t.Fatal("CallFD returned nil replyFile, want SCM_RIGHTS pipe")
	}
	defer func() { _ = replyFile.Close() }()

	var meta struct {
		Received string `json:"received"`
	}
	unmarshalErr := res.Unmarshal(&meta)
	if unmarshalErr != nil || meta.Received != "from-client" {
		t.Fatalf("CallFD result = (%+v, %v), want received=from-client", meta, unmarshalErr)
	}

	gotBytes, err := io.ReadAll(replyFile)
	if err != nil || string(gotBytes) != "from-server" {
		t.Fatalf("replyFile content = (%q, %v), want from-server", string(gotBytes), err)
	}
}

func assertCallFDValidationAndUnsolicitedDrop(t *testing.T, client *goubus.Client) {
	t.Helper()

	// Regular Call receiving an unsolicited SCM_RIGHTS fd must close it cleanly without error.
	_, err := client.Call(t.Context(), "log", "unsolicited", nil)
	if err != nil {
		t.Fatalf("Call(unsolicited fd): %v", err)
	}

	closedR, closedW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	_ = closedR.Close()
	_ = closedW.Close()

	_, _, err = client.CallFD(t.Context(), "log", "stream", nil, closedR)
	if !errors.Is(err, goubus.ErrInvalidParameter) {
		t.Errorf("CallFD(closed file) = %v, want ErrInvalidParameter", err)
	}
	// Invalid sendFile must not close the healthy ubus client connection.
	pingErr := client.Ping(t.Context())
	if pingErr != nil {
		t.Fatalf("Ping after CallFD(closed file): %v", pingErr)
	}

	// Non-Unix net.Pipe client returns ErrNotSupported.
	_, pipeClient := newMuxBrokerClient(t)
	_, _, err = pipeClient.CallFD(t.Context(), testSystemService, testInfoMethod, nil, nil)
	if !errors.Is(err, goubus.ErrNotSupported) {
		t.Errorf("CallFD(net.Pipe) = %v, want ErrNotSupported", err)
	}

	// Non-OK status on CallFD closes any received fd and returns CallError.
	_, _, err = client.CallFD(t.Context(), "log", "deny", nil, nil)
	if !errors.Is(err, goubus.ErrPermissionDenied) {
		t.Errorf("CallFD(deny) = %v, want ErrPermissionDenied", err)
	}
}

func serveUnixFDMock(ln net.Listener) {
	conn, err := ln.Accept()
	if err != nil {
		return
	}
	unixConn, ok := conn.(*net.UnixConn)
	if !ok {
		_ = conn.Close()

		return
	}
	defer func() { _ = unixConn.Close() }()

	writeUnixFrame(unixConn, blobmsg.UbusMsgHello, 0, []byte{0, 0, 0, 4}, -1)
	for {
		hdr, payload, recvFD, readErr := readUnixMockMsg(unixConn)
		if readErr != nil {
			if recvFD >= 0 {
				_ = syscall.Close(recvFD)
			}

			return
		}
		handleUnixFDMockFrame(unixConn, hdr, payload, recvFD)
	}
}

func handleUnixFDMockFrame(conn *net.UnixConn, hdr *blobmsg.UbusMessageHeader, payload []byte, recvFD int) {
	if hdr.Type == blobmsg.UbusMsgLookup {
		if recvFD >= 0 {
			_ = syscall.Close(recvFD)
		}
		dataBody, _ := blobmsg.CreateBlobMessage(map[uint32]any{
			blobmsg.UbusAttrObjPath: "log",
			blobmsg.UbusAttrObjID:   uint32(500),
		}, nil)
		writeUnixFrame(conn, blobmsg.UbusMsgData, hdr.Seq, dataBody, -1)
		writeUnixFrame(conn, blobmsg.UbusMsgStatus, hdr.Seq,
			blobmsg.EncodeUintAttributes([2]uint32{blobmsg.UbusAttrStatus, 0}), -1)

		return
	}
	if hdr.Type == blobmsg.UbusMsgPing {
		if recvFD >= 0 {
			_ = syscall.Close(recvFD)
		}
		writeUnixFrame(conn, blobmsg.UbusMsgStatus, hdr.Seq,
			blobmsg.EncodeUintAttributes([2]uint32{blobmsg.UbusAttrStatus, 0}), -1)

		return
	}

	attrs, _ := blobmsg.ParseTopLevelAttributes(payload)
	method, _ := attrs["method"].(string)
	replyUnixFDInvoke(conn, hdr.Seq, method, recvFD)
}

func replyUnixFDInvoke(conn *net.UnixConn, seq uint16, method string, recvFD int) {
	var received string
	if recvFD >= 0 {
		f := os.NewFile(uintptr(recvFD), "client-pipe")
		data, _ := io.ReadAll(f)
		_ = f.Close()
		received = string(data)
	}

	if received != "" {
		table, _ := blobmsg.CreateBlobmsgTable(map[string]any{"received": received})
		dataBody, _ := blobmsg.CreateBlobMessage(map[uint32]any{
			blobmsg.UbusAttrData: table[blobmsg.BlobHeaderBytes:],
		}, nil)
		writeUnixFrame(conn, blobmsg.UbusMsgData, seq, dataBody, -1)
	}

	replyR, replyW, err := os.Pipe()
	if err != nil {
		return
	}
	_, _ = replyW.WriteString("from-server")
	_ = replyW.Close()

	statusCode := uint32(goubus.StatusOK)
	if method == "deny" {
		statusCode = goubus.StatusPermissionDenied
	}
	statusBody := blobmsg.EncodeUintAttributes([2]uint32{blobmsg.UbusAttrStatus, statusCode})
	rawConn, scErr := replyR.SyscallConn()
	if scErr == nil {
		_ = rawConn.Control(func(fd uintptr) {
			writeUnixFrame(conn, blobmsg.UbusMsgStatus, seq, statusBody, int(fd))
		})
	}
	_ = replyR.Close()
}

func readUnixMockMsg(conn *net.UnixConn) (*blobmsg.UbusMessageHeader, []byte, int, error) {
	var lead [12]byte
	oob := make([]byte, syscall.CmsgSpace(4))
	recvFD := -1
	offset := 0
	for offset < len(lead) {
		n, oobn, _, _, err := conn.ReadMsgUnix(lead[offset:], oob)
		if oobn > 0 {
			msgs, parseErr := syscall.ParseSocketControlMessage(oob[:oobn])
			if parseErr == nil {
				for _, msg := range msgs {
					fds, fdErr := syscall.ParseUnixRights(&msg)
					if fdErr == nil && len(fds) > 0 {
						recvFD = fds[0]
					}
				}
			}
		}
		offset += n
		if err != nil {
			return nil, nil, recvFD, err
		}
	}

	hdr := &blobmsg.UbusMessageHeader{
		Version: lead[0],
		Type:    lead[1],
		Seq:     binary.BigEndian.Uint16(lead[2:4]),
		Peer:    binary.BigEndian.Uint32(lead[4:8]),
	}
	blobLen := binary.BigEndian.Uint32(lead[8:12])
	payload := make([]byte, blobLen)
	copy(payload[:4], lead[8:12])
	if blobLen > 4 {
		_, err := io.ReadFull(conn, payload[4:])
		if err != nil {
			return nil, nil, recvFD, err
		}
	}

	return hdr, payload, recvFD, nil
}

func writeUnixFrame(conn *net.UnixConn, msgType uint8, seq uint16, body []byte, sendFD int) {
	var buf bytes.Buffer
	_ = blobmsg.EncodeHeader(&buf, &blobmsg.UbusMessageHeader{Type: msgType, Seq: seq, Peer: 1})
	_, _ = buf.Write(body)

	var oob []byte
	if sendFD >= 0 {
		oob = syscall.UnixRights(sendFD)
	}
	n, _, err := conn.WriteMsgUnix(buf.Bytes(), oob, nil)
	_, _ = n, err
}
