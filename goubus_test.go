package goubus_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/honeybbq/goubus/v2"
)

var (
	errMockTransport            = errors.New("transport error")
	errMockUnmarshal            = errors.New("unmarshal error")
	errMockUnexpectedTargetType = errors.New("unexpected target type")
)

// mockTransport is a mock implementation of Transport for testing.
type mockTransport struct {
	callFunc func(ctx context.Context, service, method string, data any) (goubus.Result, error)
}

// mockResult is a mock implementation of Result for testing.
type mockResult struct {
	unmarshalFunc func(target any) error
}

type callTestResponse struct {
	Foo string `json:"foo"`
}

func (m *mockTransport) Call(ctx context.Context, service, method string, data any) (goubus.Result, error) {
	return m.callFunc(ctx, service, method, data)
}

func (m *mockTransport) Close() error {
	return nil
}

func (m *mockResult) Unmarshal(target any) error {
	return m.unmarshalFunc(target)
}

func TestCallPointer(t *testing.T) {
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		transport := &mockTransport{
			callFunc: func(context.Context, string, string, any) (goubus.Result, error) {
				return goubus.NewJSONResult(json.RawMessage(`{"foo":"bar"}`)), nil
			},
		}
		res, err := goubus.Call[*callTestResponse](ctx, transport, "service", "method", nil)
		if err != nil {
			t.Fatalf("Call() unexpected error = %v", err)
		}
		if res == nil || res.Foo != "bar" {
			t.Fatalf("Call() = %#v, want &callTestResponse{Foo: \"bar\"}", res)
		}
	})

	t.Run("Transport Error", func(t *testing.T) {
		transport := &mockTransport{
			callFunc: func(context.Context, string, string, any) (goubus.Result, error) {
				return nil, errMockTransport
			},
		}
		res, err := goubus.Call[*callTestResponse](ctx, transport, "service", "method", nil)
		if !errors.Is(err, errMockTransport) || res != nil {
			t.Fatalf("Call() = (%#v, %v), want (nil, %v)", res, err, errMockTransport)
		}
	})

	t.Run("Unmarshal Error Returns Zero", func(t *testing.T) {
		transport := &mockTransport{
			callFunc: func(context.Context, string, string, any) (goubus.Result, error) {
				return &mockResult{
					unmarshalFunc: func(target any) error {
						ptr, ok := target.(**callTestResponse)
						if !ok {
							return errMockUnexpectedTargetType
						}
						*ptr = &callTestResponse{Foo: "partial"}

						return errMockUnmarshal
					},
				}, nil
			},
		}
		res, err := goubus.Call[*callTestResponse](ctx, transport, "service", "method", nil)
		if !errors.Is(err, errMockUnmarshal) {
			t.Fatalf("Call() error = %v, want %v", err, errMockUnmarshal)
		}
		if res != nil {
			t.Fatalf("Call() = %#v, want nil zero value on unmarshal error", res)
		}
	})
}

func TestCallValue(t *testing.T) {
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		transport := &mockTransport{
			callFunc: func(context.Context, string, string, any) (goubus.Result, error) {
				return &mockResult{
					unmarshalFunc: func(target any) error {
						resp, ok := target.(*callTestResponse)
						if !ok {
							return errMockUnexpectedTargetType
						}
						resp.Foo = "bar"

						return nil
					},
				}, nil
			},
		}
		res, err := goubus.Call[callTestResponse](ctx, transport, "service", "method", nil)
		if err != nil || res.Foo != "bar" {
			t.Fatalf("Call() = (%#v, %v), want ({Foo: \"bar\"}, nil)", res, err)
		}
	})

	t.Run("Unmarshal Error Returns Zero", func(t *testing.T) {
		transport := &mockTransport{
			callFunc: func(context.Context, string, string, any) (goubus.Result, error) {
				return &mockResult{
					unmarshalFunc: func(target any) error {
						resp, ok := target.(*callTestResponse)
						if !ok {
							return errMockUnexpectedTargetType
						}
						resp.Foo = "partial"

						return errMockUnmarshal
					},
				}, nil
			},
		}
		res, err := goubus.Call[callTestResponse](ctx, transport, "service", "method", nil)
		if !errors.Is(err, errMockUnmarshal) {
			t.Fatalf("Call() error = %v, want %v", err, errMockUnmarshal)
		}
		if res != (callTestResponse{}) {
			t.Fatalf("Call() = %#v, want zero value on unmarshal error", res)
		}
	})
}

func TestCallMap(t *testing.T) {
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		transport := &mockTransport{
			callFunc: func(context.Context, string, string, any) (goubus.Result, error) {
				return goubus.NewJSONResult(json.RawMessage(`{"foo":"bar"}`)), nil
			},
		}
		res, err := goubus.Call[map[string]string](ctx, transport, "service", "method", nil)
		want := map[string]string{"foo": "bar"}
		if err != nil || !reflect.DeepEqual(res, want) {
			t.Fatalf("Call() = (%#v, %v), want (%#v, nil)", res, err, want)
		}
	})

	t.Run("Unmarshal Error Returns Nil Map", func(t *testing.T) {
		transport := &mockTransport{
			callFunc: func(context.Context, string, string, any) (goubus.Result, error) {
				return &mockResult{
					unmarshalFunc: func(target any) error {
						m, ok := target.(*map[string]string)
						if !ok {
							return errMockUnexpectedTargetType
						}
						*m = map[string]string{"foo": "partial"}

						return errMockUnmarshal
					},
				}, nil
			},
		}
		res, err := goubus.Call[map[string]string](ctx, transport, "service", "method", nil)
		if !errors.Is(err, errMockUnmarshal) {
			t.Fatalf("Call() error = %v, want %v", err, errMockUnmarshal)
		}
		if res != nil {
			t.Fatalf("Call() = %#v, want nil map on unmarshal error", res)
		}
	})
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
	if err != nil || !got.OK {
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

func TestListAndArgTypeEdgeCases(t *testing.T) {
	nonLister := goubus.CallerFunc(func(context.Context, string, string, any) (goubus.Result, error) {
		return nil, goubus.ErrNoData
	})
	_, err := goubus.List(t.Context(), nonLister)
	if !errors.Is(err, goubus.ErrNotSupported) {
		t.Fatalf("List(non-Lister) error = %v, want ErrNotSupported", err)
	}
	_, err = goubus.List(t.Context(), nonLister, "")
	if !errors.Is(err, goubus.ErrInvalidParameter) {
		t.Fatalf("List(empty pattern) error = %v, want ErrInvalidParameter", err)
	}

	cases := map[string]goubus.ArgType{
		`"boolean"`:            goubus.ArgTypeBoolean,
		`"array"`:              goubus.ArgTypeArray,
		`"object"`:             goubus.ArgTypeObject,
		`"mystery"`:            goubus.ArgTypeUnknown,
		`["unknown","number"]`: goubus.ArgTypeNumber,
	}
	for raw, want := range cases {
		var got goubus.ArgType
		unmarshalErr := json.Unmarshal([]byte(raw), &got)
		if unmarshalErr != nil || got != want {
			t.Errorf("Unmarshal(%s) = (%q, %v), want %q", raw, got, unmarshalErr, want)
		}
	}

	var bad goubus.ArgType
	if (&bad).UnmarshalJSON([]byte("{")) == nil || bad != goubus.ArgTypeUnknown {
		t.Errorf("invalid ArgType JSON = %q, want ArgTypeUnknown", bad)
	}
}
