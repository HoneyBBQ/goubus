package goubus_test

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/honeybbq/goubus/v2"
)

type exampleCaller struct{}

func (exampleCaller) Call(context.Context, string, string, any) (goubus.Result, error) {
	return goubus.NewJSONResult(json.RawMessage(`{"ready":true}`)), nil
}

func (exampleCaller) List(context.Context, ...string) (map[string]goubus.ObjectSignatures, error) {
	return map[string]goubus.ObjectSignatures{"system": {}}, nil
}

func ExampleCall() {
	ctx := context.Background()
	middleware := func(next goubus.Caller) goubus.Caller {
		return goubus.CallerFunc(func(ctx context.Context, service, method string, data any) (goubus.Result, error) {
			return next.Call(ctx, service, method, data)
		})
	}
	caller := goubus.ChainMiddleware(exampleCaller{}, middleware)
	result, err := goubus.Call[struct {
		Ready bool `json:"ready"`
	}](ctx, caller, "system", "info", nil)
	if err != nil {
		fmt.Println(err)

		return
	}
	objects, err := goubus.List(ctx, exampleCaller{}, "system")
	if err != nil {
		fmt.Println(err)

		return
	}
	fmt.Println(result.Ready, len(objects))
	// Output: true 1
}
