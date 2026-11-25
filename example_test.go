package goubus_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/honeybbq/goubus/v2"
)

type exampleCaller struct{}

func (exampleCaller) Call(context.Context, string, string, any) (goubus.Result, error) {
	return goubus.NewJSONResult(json.RawMessage(`{"up":true,"dns":["1.1.1.1","8.8.8.8"]}`)), nil
}

func (exampleCaller) List(context.Context, ...string) (map[string]goubus.ObjectSignatures, error) {
	return map[string]goubus.ObjectSignatures{
		"uci": {"apply": {"rollback": goubus.ArgTypeBoolean, "timeout": goubus.ArgTypeNumber}},
	}, nil
}

func ExampleCall() {
	ctx := context.Background()
	audit := func(next goubus.Caller) goubus.Caller {
		return goubus.CallerFunc(func(ctx context.Context, service, method string, data any) (goubus.Result, error) {
			return next.Call(ctx, service, method, data)
		})
	}

	caller := goubus.ChainMiddleware(exampleCaller{}, audit)
	status, err := goubus.Call[struct {
		DNS []string `json:"dns"`
		Up  bool     `json:"up"`
	}](ctx, caller, "network.interface.wan", "status", nil)
	if err != nil {
		fmt.Println(err)

		return
	}

	objects, err := goubus.List(ctx, exampleCaller{}, "uci")
	if err != nil {
		fmt.Println(err)

		return
	}

	fmt.Println(status.Up, status.DNS[0], objects["uci"]["apply"]["rollback"])
	// Output: true 1.1.1.1 boolean
}

func ExampleCallError() {
	caller := goubus.CallerFunc(func(context.Context, string, string, any) (goubus.Result, error) {
		return nil, &goubus.CallError{
			Err:     goubus.ErrNotFound,
			Service: "network.interface.wan6",
			Method:  "status",
			Code:    goubus.StatusNotFound,
		}
	})

	_, err := caller.Call(context.Background(), "network.interface.wan6", "status", nil)
	if errors.Is(err, goubus.ErrNotFound) {
		if callErr, ok := errors.AsType[*goubus.CallError](err); ok {
			fmt.Printf("%s.%s code=%d isCall=%v\n",
				callErr.Service, callErr.Method, callErr.Code, errors.Is(err, goubus.ErrCall))
		}
	}
	// Output: network.interface.wan6.status code=4 isCall=true
}

func ExampleNewClient() {
	socketPath, stop := startExampleUbusd()
	defer stop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := goubus.NewClient(ctx, socketPath)
	if err != nil {
		return
	}
	defer func() { _ = conn.Close() }()

	info, err := goubus.Call[struct {
		Hostname string `json:"hostname"`
	}](ctx, conn, "system", "info", nil)
	if err != nil {
		return
	}
	fmt.Println(info.Hostname)
	// Output: OpenWrt
}

func ExampleClient_Listen() {
	socketPath, stop := startExampleUbusd()
	defer stop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := goubus.NewClient(ctx, socketPath)
	if err != nil {
		return
	}
	defer func() { _ = conn.Close() }()

	events := make(chan string, 1)
	sub, err := conn.Listen(ctx, "custom.*", func(ev goubus.Event) {
		var payload struct {
			Status string `json:"status"`
		}
		decodeErr := ev.Unmarshal(&payload)
		if decodeErr == nil {
			events <- ev.Type + ":" + payload.Status
		}
	})
	if err != nil {
		return
	}
	defer func() { _ = sub.Close() }()

	_ = conn.SendEvent(ctx, "custom.ready", map[string]any{"status": "ok"})
	fmt.Println(<-events)
	// Output: custom.ready:ok
}

func ExampleClient_AddObject() {
	socketPath, stop := startExampleUbusd()
	defer stop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := goubus.NewClient(ctx, socketPath)
	if err != nil {
		return
	}
	defer func() { _ = conn.Close() }()

	obj, err := conn.AddObject(ctx, goubus.ObjectDef{
		Name: "myapp.greeter",
		Methods: map[string]goubus.Method{
			"hello": {
				Policy: goubus.MethodSignature{"name": goubus.ArgTypeString},
				Handler: func(_ context.Context, req goubus.CallRequest) (any, error) {
					var in struct {
						Name string `json:"name"`
					}
					decodeErr := req.Unmarshal(&in)
					if decodeErr != nil {
						return nil, goubus.ErrInvalidParameter
					}

					return map[string]any{"greeting": "Hello, " + in.Name}, nil
				},
			},
		},
	})
	if err != nil {
		return
	}
	defer func() { _ = obj.Close() }()

	reply, err := goubus.Call[struct {
		Greeting string `json:"greeting"`
	}](ctx, conn, "myapp.greeter", "hello", map[string]any{"name": "OpenWrt"})
	if err != nil {
		return
	}
	_ = obj.Notify(ctx, "greeted", map[string]any{"id": obj.ID()})
	fmt.Println(reply.Greeting)
	// Output: Hello, OpenWrt
}

func ExampleClient_Subscribe() {
	socketPath, stop := startExampleUbusd()
	defer stop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := goubus.NewClient(ctx, socketPath)
	if err != nil {
		return
	}
	defer func() { _ = conn.Close() }()

	notes := make(chan string, 1)
	sub, err := conn.Subscribe(ctx, "hostapd.phy0-ap0", func(note goubus.Notification) {
		var sta struct {
			Address string `json:"address"`
		}
		decodeErr := note.Unmarshal(&sta)
		if decodeErr == nil {
			notes <- note.Object + " " + note.Method + " " + sta.Address
		}
	})
	if err != nil {
		return
	}
	defer func() { _ = sub.Close() }()

	fmt.Println(<-notes)
	// Output: hostapd.phy0-ap0 sta.authorized aa:bb:cc:dd:ee:ff
}
