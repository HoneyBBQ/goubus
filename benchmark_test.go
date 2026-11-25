package goubus_test

import (
	"context"
	"net"
	"testing"

	"github.com/honeybbq/goubus/v2"
	"github.com/honeybbq/goubus/v2/internal/blobmsg"
)

func BenchmarkBlobmsgRoundTrip(b *testing.B) {
	payload := map[string]any{
		"hostname": "OpenWrt",
		"uptime":   36000,
		"up":       true,
		"dns":      []string{"1.1.1.1", "8.8.8.8"},
	}

	b.ReportAllocs()
	for b.Loop() {
		encoded, err := blobmsg.CreateBlobmsgTable(payload)
		if err != nil {
			b.Fatal(err)
		}
		_, err = blobmsg.ParseBlobmsgContainer(encoded[blobmsg.BlobHeaderBytes:], blobmsg.TypeTable)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSocketClientCall(b *testing.B) {
	serverConn, clientConn := net.Pipe()
	go serveSocketMock(serverConn, systemMockHandler)

	ctx := context.Background()
	client, err := goubus.NewClientFromConn(ctx, clientConn)
	if err != nil {
		_ = serverConn.Close()
		b.Fatal(err)
	}
	b.Cleanup(func() {
		_ = client.Close()
		_ = serverConn.Close()
	})

	_, err = client.Call(ctx, testSystemService, testInfoMethod, nil)
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	for b.Loop() {
		res, callErr := goubus.Call[struct {
			Hostname string `json:"hostname"`
		}](ctx, client, testSystemService, testInfoMethod, nil)
		if callErr != nil || res.Hostname != testHostname {
			b.Fatalf("Call = %+v, err %v", res, callErr)
		}
	}
}
