package goubus_test

import (
	"errors"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/honeybbq/goubus/v2"
	"github.com/honeybbq/goubus/v2/internal/blobmsg"
)

func TestSocketClientList(t *testing.T) {
	t.Run("all objects and signatures", func(t *testing.T) {
		client := newPipeSocketClient(t, listTestHandler)
		objects, err := client.List(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		want := map[string]goubus.ObjectSignatures{
			"alpha.one": {
				"info": {"name": goubus.ArgTypeString, "enabled": goubus.ArgTypeBoolean, "count": goubus.ArgTypeNumber},
			},
			"alpha.empty": {},
		}
		if !reflect.DeepEqual(objects, want) {
			t.Errorf("List() = %#v", objects)
		}
	})

	t.Run("multiple patterns ignore not found", func(t *testing.T) {
		client := newPipeSocketClient(t, listTestHandler)
		objects, err := client.List(t.Context(), "alpha*", "missing*")
		if err != nil {
			t.Fatal(err)
		}
		if len(objects) != 2 {
			t.Errorf("List() returned %d objects, want 2: %#v", len(objects), objects)
		}
	})

	t.Run("zero matches return non-nil map", func(t *testing.T) {
		client := newPipeSocketClient(t, listTestHandler)
		objects, err := client.List(t.Context(), "missing*")
		if err != nil {
			t.Fatal(err)
		}
		if objects == nil || len(objects) != 0 {
			t.Errorf("List() = %#v, want empty non-nil map", objects)
		}
	})

	t.Run("empty pattern rejected before lookup", func(t *testing.T) {
		client := newPipeSocketClient(t, listTestHandler)
		_, err := client.List(t.Context(), "alpha", "")
		if !errors.Is(err, goubus.ErrInvalidParameter) {
			t.Errorf("List() error = %v, want ErrInvalidParameter", err)
		}
	})

	t.Run("closed client returns ErrClosed", func(t *testing.T) {
		client := newPipeSocketClient(t, listTestHandler)
		closeErr := client.Close()
		if closeErr != nil {
			t.Fatal(closeErr)
		}
		_, err := client.List(t.Context(), "system")
		if !errors.Is(err, goubus.ErrClosed) {
			t.Errorf("List() error = %v, want ErrClosed", err)
		}
	})
}

func TestSocketClientCallReportsStatusImmediately(t *testing.T) {
	client := newPipeSocketClient(t, permissionDeniedInvokeHandler)
	result, err := client.Call(t.Context(), testSystemService, testInfoMethod, nil)
	if result != nil {
		t.Fatalf("Call() result = %#v, want nil", result)
	}
	if !errors.Is(err, goubus.ErrPermissionDenied) || !errors.Is(err, goubus.ErrCall) {
		t.Fatalf("Call() error = %v, want permission denied and ErrCall", err)
	}
	var callErr *goubus.CallError
	if !errors.As(err, &callErr) || callErr.Code != goubus.StatusPermissionDenied {
		t.Fatalf("Call() error = %#v, want status CallError", err)
	}
}

func TestSocketClientCallLookupStatusErrorIsCallError(t *testing.T) {
	client := newPipeSocketClient(t, permissionDeniedLookupHandler)
	result, err := client.Call(t.Context(), testSystemService, testInfoMethod, nil)
	if result != nil {
		t.Fatalf("Call() result = %#v, want nil", result)
	}
	if !errors.Is(err, goubus.ErrPermissionDenied) || !errors.Is(err, goubus.ErrCall) {
		t.Fatalf("Call() error = %v, want permission denied and ErrCall", err)
	}
	var callErr *goubus.CallError
	if !errors.As(err, &callErr) || callErr.Code != goubus.StatusPermissionDenied {
		t.Fatalf("Call() error = %#v, want status CallError", err)
	}
}

func TestSocketClientCallTypedNilPointersDoNotError(t *testing.T) {
	client := newPipeSocketClient(t, echoInvokeHandler)
	result, err := client.Call(t.Context(), testSystemService, testInfoMethod, map[string]any{
		"nil_bool": (*goubus.Bool)(nil),
		"nil_time": (*goubus.UnixTime)(nil),
		"nil_int":  (*int)(nil),
	})
	if err != nil {
		t.Fatalf("Call() with typed nil pointers: %v", err)
	}
	if result == nil {
		t.Fatal("Call() result is nil")
	}
}

func TestSocketClientCallReflectiveValuesRoundTrip(t *testing.T) {
	client := newPipeSocketClient(t, echoInvokeHandler)
	wantTime := time.Unix(1_700_000_000, 0).UTC()
	input := map[string]any{
		"enabled": goubus.Bool(true),
		"count":   goubus.StringInt(4014),
		"devices": goubus.StringOrSlice{"eth0", "eth1"},
		"updated": goubus.UnixTime{Time: wantTime},
	}
	result, err := client.Call(t.Context(), testSystemService, testInfoMethod, input)
	if err != nil {
		t.Fatal(err)
	}
	var output struct {
		Updated goubus.UnixTime      `json:"updated"`
		Devices goubus.StringOrSlice `json:"devices"`
		Count   goubus.StringInt     `json:"count"`
		Enabled goubus.Bool          `json:"enabled"`
	}
	err = result.Unmarshal(&output)
	if err != nil {
		t.Fatal(err)
	}
	if output.Enabled != goubus.Bool(true) || output.Count != 4014 ||
		!reflect.DeepEqual(output.Devices, input["devices"]) || !output.Updated.Equal(wantTime) {
		t.Errorf("round-trip output = %#v", output)
	}
}

func listTestHandler(conn net.Conn, header *blobmsg.UbusMessageHeader, payload []byte) {
	if header.Type != blobmsg.UbusMsgLookup {
		return
	}
	attrs, _ := blobmsg.ParseTopLevelAttributes(payload)
	path, _ := attrs["objpath"].(string)
	switch path {
	case "":
		sendListedObject(conn, header.Seq, "alpha.one", 11, map[string]any{
			"info": map[string]any{
				"name":    uint8(blobmsg.TypeString),
				"enabled": uint8(blobmsg.TypeInt8),
				"count":   uint8(blobmsg.TypeInt32),
			},
		})
		sendListedObject(conn, header.Seq, "alpha.empty", 12, nil)
		sendLookupStatus(conn, header.Seq, 0)
	case "alpha*":
		sendListedObject(conn, header.Seq, "alpha.one", 11, nil)
		sendListedObject(conn, header.Seq, "alpha.empty", 12, nil)
		sendLookupStatus(conn, header.Seq, 0)
	case "missing*":
		sendLookupStatus(conn, header.Seq, goubus.StatusNotFound)
	default:
		sendLookupStatus(conn, header.Seq, 0)
	}
}

func sendListedObject(conn net.Conn, seq uint16, path string, id uint32, signature map[string]any) {
	attrs := map[uint32]any{
		blobmsg.UbusAttrObjPath: path,
		blobmsg.UbusAttrObjID:   id,
	}
	if signature != nil {
		encoded, _ := blobmsg.CreateBlobmsgTable(signature)
		attrs[blobmsg.UbusAttrSignature] = encoded[blobmsg.BlobHeaderBytes:]
	}
	body, _ := blobmsg.CreateBlobMessage(attrs, nil)
	sendMsg(conn, blobmsg.UbusMsgData, seq, body)
}

func sendLookupStatus(conn net.Conn, seq uint16, code uint32) {
	body, _ := blobmsg.CreateBlobMessage(map[uint32]any{blobmsg.UbusAttrStatus: code}, nil)
	sendMsg(conn, blobmsg.UbusMsgStatus, seq, body)
}

func permissionDeniedInvokeHandler(conn net.Conn, header *blobmsg.UbusMessageHeader, payload []byte) {
	if header.Type == blobmsg.UbusMsgLookup {
		handleLookup(conn, header.Seq, payload)

		return
	}
	if header.Type == blobmsg.UbusMsgInvoke {
		sendLookupStatus(conn, header.Seq, goubus.StatusPermissionDenied)
	}
}

func permissionDeniedLookupHandler(conn net.Conn, header *blobmsg.UbusMessageHeader, _ []byte) {
	if header.Type == blobmsg.UbusMsgLookup {
		sendLookupStatus(conn, header.Seq, goubus.StatusPermissionDenied)
	}
}

func echoInvokeHandler(conn net.Conn, header *blobmsg.UbusMessageHeader, payload []byte) {
	if header.Type == blobmsg.UbusMsgLookup {
		handleLookup(conn, header.Seq, payload)

		return
	}
	if header.Type != blobmsg.UbusMsgInvoke {
		return
	}
	attrs, _ := blobmsg.ParseTopLevelAttributes(payload)
	args, _ := attrs["data"].(map[string]any)
	encoded, _ := blobmsg.CreateBlobmsgTable(args)
	body, _ := blobmsg.CreateBlobMessage(map[uint32]any{
		blobmsg.UbusAttrData: encoded[blobmsg.BlobHeaderBytes:],
	}, nil)
	sendMsg(conn, blobmsg.UbusMsgData, header.Seq, body)
	sendLookupStatus(conn, header.Seq, 0)
}
