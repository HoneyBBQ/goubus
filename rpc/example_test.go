package rpc_test

import (
	"context"
	"fmt"
	"time"

	"github.com/honeybbq/goubus/v2"
	"github.com/honeybbq/goubus/v2/rpc"
)

func ExampleNewClient() {
	routerURL, stop := startExampleRouter()
	defer stop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := rpc.NewClient(ctx, routerURL, "root", "secret")
	if err != nil {
		return
	}
	defer func() { _ = conn.Close() }()

	board, err := goubus.Call[struct {
		Hostname string `json:"hostname"`
	}](ctx, conn, "system", "board", nil)
	if err != nil {
		return
	}
	fmt.Println(board.Hostname)
	// Output: OpenWrt
}

func ExampleClient_List() {
	routerURL, stop := startExampleRouter()
	defer stop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := rpc.NewClient(ctx, routerURL, "root", "secret")
	if err != nil {
		return
	}
	defer func() { _ = conn.Close() }()

	objects, err := goubus.List(ctx, conn, "system")
	if err != nil {
		return
	}
	fmt.Println(objects["system"]["board"]["hostname"])
	// Output: string
}
