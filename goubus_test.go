package goubus_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/honeybbq/goubus/v2"
)

// mockTransport is a mock implementation of Transport for testing.
type mockTransport struct {
	callFunc func(ctx context.Context, service, method string, data any) (goubus.Result, error)
}

func (m *mockTransport) Call(ctx context.Context, service, method string, data any) (goubus.Result, error) {
	return m.callFunc(ctx, service, method, data)
}

func (m *mockTransport) SetLogger(_ *slog.Logger) {}

func (m *mockTransport) Close() error {
	return nil
}

// mockResult is a mock implementation of Result for testing.
type mockResult struct {
	unmarshalFunc func(target any) error
}

func (m *mockResult) Unmarshal(target any) error {
	return m.unmarshalFunc(target)
}

var (
	errMockTransport            = errors.New("transport error")
	errMockUnmarshal            = errors.New("unmarshal error")
	errMockUnexpectedTargetType = errors.New("unexpected target type")
)

type callTestResponse struct {
	Foo string `json:"foo"`
}

type callTestCase struct {
	mock    func(ctx context.Context, service, method string, data any) (goubus.Result, error)
	name    string
	wantErr bool
}

var callTestCases = []callTestCase{
	{
		name: "Success",
		mock: func(ctx context.Context, service, method string, data any) (goubus.Result, error) {
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
		wantErr: false,
	},
	{
		name: "Transport Error",
		mock: func(ctx context.Context, service, method string, data any) (goubus.Result, error) {
			return nil, errMockTransport
		},
		wantErr: true,
	},
	{
		name: "Unmarshal Error",
		mock: func(ctx context.Context, service, method string, data any) (goubus.Result, error) {
			return &mockResult{
				unmarshalFunc: func(target any) error {
					return errMockUnmarshal
				},
			}, nil
		},
		wantErr: true,
	},
}

func TestCall(t *testing.T) {
	for _, tt := range callTestCases {
		t.Run(tt.name, func(t *testing.T) {
			runCallTestCase(t, tt)
		})
	}
}

func runCallTestCase(t *testing.T, tt callTestCase) {
	t.Helper()

	transport := &mockTransport{callFunc: tt.mock}
	ctx := context.Background()
	res, err := goubus.Call[callTestResponse](ctx, transport, "service", "method", nil)

	if (err != nil) != tt.wantErr {
		t.Errorf("Call() error = %v, wantErr %v", err, tt.wantErr)

		return
	}

	if !tt.wantErr {
		if res == nil {
			t.Fatal("Call() returned nil result but no error")
		}

		if res.Foo != "bar" {
			t.Errorf("Call() expected result.Foo = 'bar', got %v", res.Foo)
		}
	}
}
