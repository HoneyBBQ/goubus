package services_test

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/honeybbq/goubus/v2"
	"github.com/honeybbq/goubus/v2/services"
	"github.com/honeybbq/goubus/v2/services/system"
)

func Example() {
	ctx := context.Background()
	caller := goubus.CallerFunc(func(context.Context, string, string, any) (goubus.Result, error) {
		return goubus.NewJSONResult(json.RawMessage(`{"uptime":42}`)), nil
	})

	all := services.New(caller)
	info, err := all.System().Info(ctx)
	if err != nil {
		fmt.Println(err)

		return
	}
	fmt.Println(info.Uptime)

	single := system.New(caller)
	info, err = single.Info(ctx)
	if err != nil {
		fmt.Println(err)

		return
	}
	fmt.Println(info.Uptime)
	// Output:
	// 42
	// 42
}
