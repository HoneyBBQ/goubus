package testutil

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/honeybbq/goubus/v2"
	"github.com/honeybbq/goubus/v2/errdefs"
)

// MockTransport is a mock implementation of goubus.Transport for testing.
type MockTransport struct {
	Logger    *slog.Logger
	Responses map[string]any // key: "service.method" or "service.method.jsonArgs"
	Calls     []MockCall
	mu        sync.Mutex
}

// MockCall records a call to the transport.
type MockCall struct {
	Data    any
	Service string
	Method  string
}

// MockResult is a mock implementation of goubus.Result.
type MockResult struct {
	Data any
}

func (r *MockResult) Unmarshal(target any) error {
	if r.Data == nil {
		return errdefs.ErrNoData
	}

	// Convert map/struct to JSON and then to target to simulate real unmarshaling
	b, err := json.Marshal(r.Data)
	if err != nil {
		return err
	}

	return json.Unmarshal(b, target)
}

func NewMockTransport() *MockTransport {
	return &MockTransport{
		Responses: make(map[string]any),
	}
}

func (m *MockTransport) Call(ctx context.Context, service, method string, data any) (goubus.Result, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Calls = append(m.Calls, MockCall{
		Service: service,
		Method:  method,
		Data:    data,
	})

	key := fmt.Sprintf("%s.%s", service, method)

	resp, ok := m.Responses[key]

	if !ok {
		return nil, errdefs.Wrapf(errdefs.ErrNotFound, "no mock response for %s", key)
	}

	return &MockResult{Data: resp}, nil
}

func (m *MockTransport) SetLogger(logger *slog.Logger) {
	m.Logger = logger
}

func (m *MockTransport) Close() error {
	return nil
}

// AddResponse adds a mock response for a service and method.
func (m *MockTransport) AddResponse(service, method string, response any) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Responses[fmt.Sprintf("%s.%s", service, method)] = response
}

// AddResponseFromFile loads a mock response from a JSON file in the testdata directory.
// The path should be relative to the project root, e.g., "internal/testdata/rax3000m/system_board.json".
func (m *MockTransport) AddResponseFromFile(service, method string, filePath string) error {
	data, err := os.ReadFile(filepath.Clean(filePath))
	if err != nil {
		return err
	}

	var response any

	err = json.Unmarshal(data, &response)
	if err != nil {
		return err
	}

	m.AddResponse(service, method, response)

	return nil
}

// GetLastCall returns the last call made to the transport.
func (m *MockTransport) GetLastCall() MockCall {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.Calls) == 0 {
		return MockCall{}
	}

	return m.Calls[len(m.Calls)-1]
}
