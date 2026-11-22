package system_test

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/honeybbq/goubus/v2"
	"github.com/honeybbq/goubus/v2/services/system"
)

func Example() {
	ctx := context.Background()
	caller := goubus.CallerFunc(func(context.Context, string, string, any) (goubus.Result, error) {
		return goubus.NewJSONResult(json.RawMessage(`{"uptime":42}`)), nil
	})

	info, err := system.New(caller).Info(ctx)
	if err != nil {
		fmt.Println(err)

		return
	}
	fmt.Println(info.Uptime)
	// Output: 42
}
