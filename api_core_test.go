package goubus_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/honeybbq/goubus/v2"
)

func TestCallErrorUnwrapTree(t *testing.T) {
	callErr := &goubus.CallError{
		Err: goubus.ErrNotFound, Service: "system", Method: "info", Message: "missing", Code: goubus.StatusNotFound,
	}
	if got, want := callErr.Error(), "ubus call system.info failed (code 4: missing): not found"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
	if !errors.Is(callErr, goubus.ErrCall) || !errors.Is(callErr, goubus.ErrNotFound) {
		t.Errorf("Unwrap() error tree does not contain both sentinels: %v", callErr)
	}
	got := callErr.Unwrap()
	if len(got) != 2 || !errors.Is(got[0], goubus.ErrCall) || !errors.Is(got[1], goubus.ErrNotFound) {
		t.Errorf("Unwrap() = %#v, want ErrCall and ErrNotFound", got)
	}
	got = (&goubus.CallError{Code: 99}).Unwrap()
	if len(got) != 1 || !errors.Is(got[0], goubus.ErrCall) {
		t.Errorf("Unwrap() without cause = %#v, want ErrCall", got)
	}
	if got := (&goubus.CallError{Method: "list", Code: 1}).Error(); got != "ubus list failed (code 1): <nil>" {
		t.Errorf("Error() without service = %q", got)
	}
}

func TestCallHelpers(t *testing.T) {
	var order []string
	base := goubus.CallerFunc(func(context.Context, string, string, any) (goubus.Result, error) {
		order = append(order, "call")

		return goubus.NewJSONResult(json.RawMessage(`{"ok":true}`)), nil
	})
	wrap := func(name string) goubus.CallMiddleware {
		return func(next goubus.Caller) goubus.Caller {
			return goubus.CallerFunc(func(ctx context.Context, service, method string, data any) (goubus.Result, error) {
				order = append(order, "before-"+name)
				result, err := next.Call(ctx, service, method, data)
				order = append(order, "after-"+name)

				return result, err
			})
		}
	}

	got, err := goubus.Call[struct {
		OK bool `json:"ok"`
	}](context.Background(), goubus.ChainMiddleware(base, wrap("one"), wrap("two")), "svc", "method", nil)
	if err != nil || got == nil || !got.OK {
		t.Fatalf("Call() = %#v, %v", got, err)
	}
	wantOrder := []string{"before-one", "before-two", "call", "after-two", "after-one"}
	if !reflect.DeepEqual(order, wantOrder) {
		t.Errorf("middleware order = %#v, want %#v", order, wantOrder)
	}
}

func TestNewJSONResultNoDataAndInvalid(t *testing.T) {
	for _, raw := range []json.RawMessage{nil, []byte(" "), []byte("null")} {
		err := goubus.NewJSONResult(raw).Unmarshal(new(any))
		if !errors.Is(err, goubus.ErrNoData) {
			t.Errorf("Unmarshal(%q) error = %v, want ErrNoData", raw, err)
		}
	}
	err := goubus.NewJSONResult([]byte("{")).Unmarshal(new(any))
	if !errors.Is(err, goubus.ErrInvalidResponse) {
		t.Errorf("invalid JSON result error = %v, want ErrInvalidResponse", err)
	}
}
